package service

import (
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/middleware"
	"concrete-curing-maturity-monitor/backend/internal/model"
	"concrete-curing-maturity-monitor/backend/internal/repository"
	"concrete-curing-maturity-monitor/backend/internal/timeseries"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"
)

type TemperatureSeriesService interface {
	List(context.Context, dto.TemperatureSeriesQuery) (dto.TemperatureSeriesListResponse, error)
	Get(context.Context, uint) (dto.TemperatureSeriesResponse, error)
	Import(context.Context, dto.ImportTemperatureSeriesRequest, util.Actor) (dto.TemperatureSeriesResponse, error)
	Confirm(context.Context, uint, dto.TemperatureSeriesActionRequest, util.Actor) (dto.TemperatureSeriesResponse, error)
	Invalidate(context.Context, uint, dto.TemperatureSeriesActionRequest, util.Actor) (dto.TemperatureSeriesResponse, error)
}

type temperatureSeriesService struct {
	series          repository.TemperatureSeriesRepository
	sections        repository.PourSectionRepository
	audit           *middleware.AuditRecorder
	maxMissingRatio float64
}

func NewTemperatureSeriesService(
	series repository.TemperatureSeriesRepository,
	sections repository.PourSectionRepository,
	audit *middleware.AuditRecorder,
	maxMissingRatio float64,
) TemperatureSeriesService {
	return &temperatureSeriesService{series: series, sections: sections, audit: audit, maxMissingRatio: maxMissingRatio}
}

func (service *temperatureSeriesService) List(ctx context.Context, query dto.TemperatureSeriesQuery) (dto.TemperatureSeriesListResponse, error) {
	series, total, err := service.series.List(ctx, query)
	if err != nil {
		return dto.TemperatureSeriesListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list temperature series", err)
	}
	items := make([]dto.TemperatureSeriesResponse, 0, len(series))
	for _, item := range series {
		items = append(items, dto.NewTemperatureSeriesResponse(item))
	}
	return dto.TemperatureSeriesListResponse{Items: items, Total: total, Page: query.Page, Size: query.PageSize}, nil
}

func (service *temperatureSeriesService) Get(ctx context.Context, id uint) (dto.TemperatureSeriesResponse, error) {
	series, err := service.series.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.TemperatureSeriesResponse{}, util.NotFound("temperature series")
		}
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load temperature series", err)
	}
	return dto.NewTemperatureSeriesResponse(series), nil
}

func (service *temperatureSeriesService) Import(ctx context.Context, request dto.ImportTemperatureSeriesRequest, actor util.Actor) (dto.TemperatureSeriesResponse, error) {
	request.Normalize()
	section, err := service.sections.GetByID(ctx, request.PourSectionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.TemperatureSeriesResponse{}, util.NotFound("pour section")
		}
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to validate pour section", err)
	}
	if section.CuringState == string(constants.CuringClosed) {
		return dto.TemperatureSeriesResponse{}, util.Conflict("temperature data cannot be imported for a closed pour section")
	}
	points := make([]timeseries.Point, 0, len(request.Points))
	for _, point := range request.Points {
		points = append(points, timeseries.Point{Timestamp: point.Timestamp, TemperatureC: point.TemperatureC})
	}
	analysis, err := timeseries.Analyze(points, request.SampleIntervalMin)
	if err != nil {
		return dto.TemperatureSeriesResponse{}, util.ErrorWithDetails(http.StatusUnprocessableEntity, util.CodeValidation, "temperature sequence failed quality validation", err.Error())
	}
	if analysis.MissingRatio > service.maxMissingRatio {
		return dto.TemperatureSeriesResponse{}, util.ErrorWithDetails(http.StatusUnprocessableEntity, util.CodeValidation, "temperature sequence exceeds the allowed missing-data ratio", map[string]any{"missing_ratio": analysis.MissingRatio, "limit": service.maxMissingRatio})
	}
	encoded, checksum, err := timeseries.CanonicalJSONAndChecksum(points)
	if err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to normalize temperature sequence", err)
	}
	qualityNote := analysis.QualityNote
	if request.QualityNote != "" {
		qualityNote = strings.TrimSpace(request.QualityNote) + "; " + analysis.QualityNote
	}
	series := model.TemperatureSeries{
		PourSectionID: request.PourSectionID, SensorCode: request.SensorCode,
		SampleIntervalMin: request.SampleIntervalMin, PointsJSON: encoded,
		StartedAt: analysis.StartedAt, EndedAt: analysis.EndedAt,
		SourceChecksum: checksum, MissingRatio: analysis.MissingRatio,
		SeriesState: constants.SeriesImported, QualityNote: qualityNote,
		ImportedBy: actor.UserID, ImportedByName: actor.DisplayName,
	}
	if err := service.series.Create(ctx, &series); err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusConflict, util.CodeConflict, "the same normalized temperature sequence has already been imported", err)
	}
	created, err := service.series.GetByID(ctx, series.ID)
	if err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload temperature series", err)
	}
	metadata := map[string]any{
		"source_checksum": checksum, "observed_points": analysis.ObservedPoints,
		"expected_points": analysis.ExpectedPoints, "missing_ratio": analysis.MissingRatio,
		"minimum_c": analysis.MinimumC, "maximum_c": analysis.MaximumC,
	}
	if err := service.audit.Record(ctx, actor, "temperature_series", series.ID, "import", nil, created, metadata); err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record temperature import audit", err)
	}
	return dto.NewTemperatureSeriesResponse(created), nil
}

func (service *temperatureSeriesService) Confirm(ctx context.Context, id uint, request dto.TemperatureSeriesActionRequest, actor util.Actor) (dto.TemperatureSeriesResponse, error) {
	before, err := service.series.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.TemperatureSeriesResponse{}, util.NotFound("temperature series")
		}
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load temperature series", err)
	}
	if before.MissingRatio > service.maxMissingRatio {
		return dto.TemperatureSeriesResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "temperature sequence cannot be confirmed because missing data exceeds the limit")
	}
	changed, err := service.series.SetState(ctx, id, []string{constants.SeriesImported}, constants.SeriesUsable, map[string]any{"quality_note": before.QualityNote + "; confirmed: " + request.Reason})
	if err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to confirm temperature series", err)
	}
	if !changed {
		return dto.TemperatureSeriesResponse{}, util.Conflict("only imported temperature series can be confirmed")
	}
	after, err := service.series.GetByID(ctx, id)
	if err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload temperature series", err)
	}
	if err := service.audit.Record(ctx, actor, "temperature_series", id, "confirm", before, after, map[string]any{"reason": request.Reason}); err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record temperature confirmation", err)
	}
	return dto.NewTemperatureSeriesResponse(after), nil
}

func (service *temperatureSeriesService) Invalidate(ctx context.Context, id uint, request dto.TemperatureSeriesActionRequest, actor util.Actor) (dto.TemperatureSeriesResponse, error) {
	before, err := service.series.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.TemperatureSeriesResponse{}, util.NotFound("temperature series")
		}
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load temperature series", err)
	}
	changed, err := service.series.SetState(ctx, id, []string{constants.SeriesImported, constants.SeriesUsable}, constants.SeriesInvalid, map[string]any{"quality_note": fmt.Sprintf("%s; invalidated: %s", before.QualityNote, request.Reason)})
	if err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to invalidate temperature series", err)
	}
	if !changed {
		return dto.TemperatureSeriesResponse{}, util.Conflict("temperature series is already invalid")
	}
	after, err := service.series.GetByID(ctx, id)
	if err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload temperature series", err)
	}
	if err := service.audit.Record(ctx, actor, "temperature_series", id, "invalidate", before, after, map[string]any{"reason": request.Reason}); err != nil {
		return dto.TemperatureSeriesResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record temperature invalidation", err)
	}
	return dto.NewTemperatureSeriesResponse(after), nil
}
