package service

import (
	"concrete-curing-maturity-monitor/backend/internal/algorithm"
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/middleware"
	"concrete-curing-maturity-monitor/backend/internal/model"
	"concrete-curing-maturity-monitor/backend/internal/repository"
	"concrete-curing-maturity-monitor/backend/internal/timeseries"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

type StrengthForecastService interface {
	List(context.Context, dto.StrengthForecastQuery) (dto.StrengthForecastListResponse, error)
	Get(context.Context, uint) (dto.StrengthForecastResponse, error)
	Run(context.Context, dto.RunStrengthForecastRequest, string, util.Actor) (dto.StrengthForecastResponse, bool, error)
	Review(context.Context, uint, dto.ForecastActionRequest, util.Actor) (dto.StrengthForecastResponse, error)
	Confirm(context.Context, uint, dto.ForecastActionRequest, util.Actor) (dto.StrengthForecastResponse, error)
	Void(context.Context, uint, dto.ForecastActionRequest, util.Actor) (dto.StrengthForecastResponse, error)
	Replay(context.Context, uint, util.Actor) (dto.StrengthForecastResponse, error)
	Compare(context.Context, uint, uint) (dto.ForecastComparisonResponse, error)
}

type strengthForecastService struct {
	forecasts repository.StrengthForecastRepository
	sections  repository.PourSectionRepository
	series    repository.TemperatureSeriesRepository
	mixes     repository.MixDesignRepository
	audit     *middleware.AuditRecorder
	now       func() time.Time
}

type forecastSnapshot struct {
	PourSectionID       uint                         `json:"pour_section_id"`
	SectionCode         string                       `json:"section_code"`
	TargetStrengthMPA   float64                      `json:"target_strength_mpa"`
	TemperatureSeriesID uint                         `json:"temperature_series_id"`
	SensorCode          string                       `json:"sensor_code"`
	SourceChecksum      string                       `json:"source_checksum"`
	MissingRatio        float64                      `json:"missing_ratio"`
	MixDesignID         uint                         `json:"mix_design_id"`
	MixCode             string                       `json:"mix_code"`
	MixDesignVersion    int                          `json:"mix_design_version"`
	DatumTemperatureC   float64                      `json:"datum_temperature_c"`
	CalibrationPoints   []algorithm.CalibrationPoint `json:"calibration_points"`
	TemperaturePoints   []algorithm.TemperaturePoint `json:"temperature_points"`
	FormulaVersion      string                       `json:"formula_version"`
}

func NewStrengthForecastService(
	forecasts repository.StrengthForecastRepository,
	sections repository.PourSectionRepository,
	series repository.TemperatureSeriesRepository,
	mixes repository.MixDesignRepository,
	audit *middleware.AuditRecorder,
) StrengthForecastService {
	return &strengthForecastService{
		forecasts: forecasts, sections: sections, series: series, mixes: mixes,
		audit: audit, now: func() time.Time { return time.Now().UTC() },
	}
}

func (service *strengthForecastService) List(ctx context.Context, query dto.StrengthForecastQuery) (dto.StrengthForecastListResponse, error) {
	forecasts, total, err := service.forecasts.List(ctx, query)
	if err != nil {
		return dto.StrengthForecastListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list strength forecasts", err)
	}
	items := make([]dto.StrengthForecastResponse, 0, len(forecasts))
	for _, forecast := range forecasts {
		items = append(items, dto.NewStrengthForecastResponse(forecast))
	}
	return dto.StrengthForecastListResponse{Items: items, Total: total, Page: query.Page, Size: query.PageSize}, nil
}

func (service *strengthForecastService) Get(ctx context.Context, id uint) (dto.StrengthForecastResponse, error) {
	forecast, err := service.forecasts.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.StrengthForecastResponse{}, util.NotFound("strength forecast")
		}
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load strength forecast", err)
	}
	return dto.NewStrengthForecastResponse(forecast), nil
}

func (service *strengthForecastService) Run(ctx context.Context, request dto.RunStrengthForecastRequest, idempotencyKey string, actor util.Actor) (dto.StrengthForecastResponse, bool, error) {
	key := strings.TrimSpace(idempotencyKey)
	if key == "" || len(key) > 100 {
		return dto.StrengthForecastResponse{}, false, util.NewError(http.StatusBadRequest, util.CodeValidation, "Idempotency-Key is required and must not exceed 100 characters")
	}
	existing, err := service.forecasts.GetByIdempotency(ctx, actor.UserID, key)
	if err == nil {
		return dto.NewStrengthForecastResponse(existing), false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.StrengthForecastResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to check idempotent forecast", err)
	}
	snapshot, encodedSnapshot, inputHash, err := service.prepareSnapshot(ctx, request)
	if err != nil {
		return dto.StrengthForecastResponse{}, false, err
	}
	existing, err = service.forecasts.GetByInputHash(ctx, inputHash, algorithm.FormulaVersion)
	if err == nil {
		return dto.NewStrengthForecastResponse(existing), false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return dto.StrengthForecastResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to check frozen forecast input", err)
	}
	now := service.now()
	forecast := model.StrengthForecast{
		PourSectionID: snapshot.PourSectionID, TemperatureSeriesID: snapshot.TemperatureSeriesID,
		MixDesignVersion: snapshot.MixDesignVersion, FormulaVersion: algorithm.FormulaVersion,
		InputHash: inputHash, InputSnapshot: encodedSnapshot, ConfidenceLevel: string(constants.ConfidenceLow),
		ForecastState: constants.ForecastQueued, Explanation: "{}", CalculatedBy: actor.UserID,
		CalculatedByName: actor.DisplayName, CalculatedAt: now,
		IdempotencyKey: key, IdempotencyActorID: actor.UserID,
	}
	if err := service.forecasts.Create(ctx, &forecast); err != nil {
		existing, lookupErr := service.forecasts.GetByIdempotency(ctx, actor.UserID, key)
		if lookupErr == nil {
			return dto.NewStrengthForecastResponse(existing), false, nil
		}
		existing, lookupErr = service.forecasts.GetByInputHash(ctx, inputHash, algorithm.FormulaVersion)
		if lookupErr == nil {
			return dto.NewStrengthForecastResponse(existing), false, nil
		}
		return dto.StrengthForecastResponse{}, false, util.WrapError(http.StatusConflict, util.CodeConflict, "forecast idempotency key already exists", err)
	}
	changed, err := service.forecasts.Transition(ctx, forecast.ID, []string{constants.ForecastQueued}, constants.ForecastCalculating, map[string]any{})
	if err != nil {
		return dto.StrengthForecastResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to begin forecast calculation", err)
	}
	if !changed {
		return dto.StrengthForecastResponse{}, false, util.Conflict("forecast state changed before calculation started")
	}
	startedAt := time.Now()
	result, calculationErr := algorithm.CalculateForecast(
		snapshot.TemperaturePoints, snapshot.DatumTemperatureC, snapshot.CalibrationPoints,
		snapshot.TargetStrengthMPA, snapshot.MissingRatio,
	)
	duration := time.Since(startedAt).Milliseconds()
	if calculationErr != nil {
		_, transitionErr := service.forecasts.Transition(ctx, forecast.ID, []string{constants.ForecastCalculating}, constants.ForecastFailed, map[string]any{
			"failure_reason": calculationErr.Error(), "duration_milliseconds": duration, "calculated_at": service.now(),
		})
		if transitionErr != nil {
			return dto.StrengthForecastResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to persist failed forecast", transitionErr)
		}
		failed, _ := service.forecasts.GetByID(ctx, forecast.ID)
		_ = service.audit.Record(ctx, actor, "strength_forecast", forecast.ID, "calculate_failed", nil, failed, map[string]any{"input_hash": inputHash, "error": calculationErr.Error()})
		return dto.StrengthForecastResponse{}, false, util.ErrorWithDetails(http.StatusUnprocessableEntity, util.CodeValidation, "forecast calculation could not use the frozen inputs", calculationErr.Error())
	}
	confidence := constants.ConfidenceFor(snapshot.MissingRatio, result.OutsideCalibration, result.DurationHours)
	explanationJSON, err := json.Marshal(result.Explanation)
	if err != nil {
		return dto.StrengthForecastResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to encode forecast explanation", err)
	}
	completedAt := service.now()
	changed, err = service.forecasts.Transition(ctx, forecast.ID, []string{constants.ForecastCalculating}, constants.ForecastCompleted, map[string]any{
		"maturity_degree_hours":  result.MaturityDegreeHours,
		"predicted_strength_mpa": result.PredictedStrengthMPA,
		"threshold_eta":          result.ThresholdETA, "confidence_level": string(confidence),
		"explanation": string(explanationJSON), "calculated_at": completedAt,
		"duration_milliseconds": duration, "failure_reason": "",
	})
	if err != nil {
		return dto.StrengthForecastResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to persist forecast result", err)
	}
	if !changed {
		return dto.StrengthForecastResponse{}, false, util.Conflict("forecast state changed while calculation was running")
	}
	completed, err := service.forecasts.GetByID(ctx, forecast.ID)
	if err != nil {
		return dto.StrengthForecastResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload completed forecast", err)
	}
	metadata := map[string]any{
		"input_hash": inputHash, "formula_version": algorithm.FormulaVersion,
		"source_checksum": snapshot.SourceChecksum, "mix_design_version": snapshot.MixDesignVersion,
		"duration_milliseconds": duration, "maturity_degree_hours": result.MaturityDegreeHours,
		"predicted_strength_mpa": result.PredictedStrengthMPA,
	}
	if err := service.audit.Record(ctx, actor, "strength_forecast", forecast.ID, "calculate", nil, completed, metadata); err != nil {
		return dto.StrengthForecastResponse{}, false, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record forecast audit", err)
	}
	return dto.NewStrengthForecastResponse(completed), true, nil
}

func (service *strengthForecastService) Review(ctx context.Context, id uint, request dto.ForecastActionRequest, actor util.Actor) (dto.StrengthForecastResponse, error) {
	before, err := service.forecasts.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.StrengthForecastResponse{}, util.NotFound("strength forecast")
		}
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load strength forecast", err)
	}
	now := service.now()
	changed, err := service.forecasts.Transition(ctx, id, []string{constants.ForecastCompleted}, constants.ForecastReviewed, map[string]any{
		"reviewed_by": actor.UserID, "reviewed_by_name": actor.DisplayName, "reviewed_at": &now,
	})
	if err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to review strength forecast", err)
	}
	if !changed {
		return dto.StrengthForecastResponse{}, util.Conflict("only completed forecasts can be reviewed")
	}
	after, err := service.forecasts.GetByID(ctx, id)
	if err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload reviewed forecast", err)
	}
	if err := service.audit.Record(ctx, actor, "strength_forecast", id, "review", before, after, map[string]any{"note": request.Note, "input_hash": before.InputHash}); err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record forecast review audit", err)
	}
	return dto.NewStrengthForecastResponse(after), nil
}

func (service *strengthForecastService) Confirm(ctx context.Context, id uint, request dto.ForecastActionRequest, actor util.Actor) (dto.StrengthForecastResponse, error) {
	before, err := service.forecasts.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.StrengthForecastResponse{}, util.NotFound("strength forecast")
		}
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load strength forecast", err)
	}
	if before.CalculatedBy == actor.UserID {
		return dto.StrengthForecastResponse{}, util.NewError(http.StatusForbidden, util.CodeForbidden, "forecast initiators cannot confirm their own results")
	}
	now := service.now()
	changed, err := service.forecasts.Transition(ctx, id, []string{constants.ForecastReviewed}, constants.ForecastConfirmed, map[string]any{
		"confirmed_by": actor.UserID, "confirmed_at": &now,
	})
	if err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to confirm strength forecast", err)
	}
	if !changed {
		return dto.StrengthForecastResponse{}, util.Conflict("only reviewed forecasts can be confirmed")
	}
	after, err := service.forecasts.GetByID(ctx, id)
	if err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload confirmed forecast", err)
	}
	if err := service.audit.Record(ctx, actor, "strength_forecast", id, "confirm", before, after, map[string]any{"note": request.Note, "input_hash": before.InputHash}); err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record forecast confirmation audit", err)
	}
	return dto.NewStrengthForecastResponse(after), nil
}

func (service *strengthForecastService) Void(ctx context.Context, id uint, request dto.ForecastActionRequest, actor util.Actor) (dto.StrengthForecastResponse, error) {
	before, err := service.forecasts.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.StrengthForecastResponse{}, util.NotFound("strength forecast")
		}
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load strength forecast", err)
	}
	changed, err := service.forecasts.Transition(ctx, id, []string{constants.ForecastCompleted, constants.ForecastReviewed, constants.ForecastFailed}, constants.ForecastVoided, map[string]any{})
	if err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to void strength forecast", err)
	}
	if !changed {
		return dto.StrengthForecastResponse{}, util.Conflict("the forecast cannot be voided from its current state")
	}
	after, err := service.forecasts.GetByID(ctx, id)
	if err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload voided forecast", err)
	}
	if err := service.audit.Record(ctx, actor, "strength_forecast", id, "void", before, after, map[string]any{"note": request.Note}); err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record forecast void audit", err)
	}
	return dto.NewStrengthForecastResponse(after), nil
}

func (service *strengthForecastService) Replay(ctx context.Context, id uint, actor util.Actor) (dto.StrengthForecastResponse, error) {
	forecast, err := service.forecasts.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.StrengthForecastResponse{}, util.NotFound("strength forecast")
		}
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load strength forecast", err)
	}
	var snapshot forecastSnapshot
	if err := json.Unmarshal([]byte(forecast.InputSnapshot), &snapshot); err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusUnprocessableEntity, util.CodeValidation, "frozen forecast inputs cannot be decoded", err)
	}
	encoded, inputHash, err := encodeAndHash(snapshot)
	if err != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to hash frozen inputs", err)
	}
	result, err := algorithm.CalculateForecast(snapshot.TemperaturePoints, snapshot.DatumTemperatureC, snapshot.CalibrationPoints, snapshot.TargetStrengthMPA, snapshot.MissingRatio)
	passed := err == nil && encoded == forecast.InputSnapshot && inputHash == forecast.InputHash &&
		math.Abs(result.MaturityDegreeHours-forecast.MaturityDegreeHours) < 0.000001 &&
		math.Abs(result.PredictedStrengthMPA-forecast.PredictedStrengthMPA) < 0.000001
	if updateErr := service.forecasts.UpdateReplay(ctx, id, passed); updateErr != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to store replay result", updateErr)
	}
	after, loadErr := service.forecasts.GetByID(ctx, id)
	if loadErr != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload replayed forecast", loadErr)
	}
	metadata := map[string]any{"input_hash": inputHash, "formula_version": snapshot.FormulaVersion, "passed": passed}
	if err != nil {
		metadata["error"] = err.Error()
	}
	if auditErr := service.audit.Record(ctx, actor, "strength_forecast", id, "deterministic_replay", forecast, after, metadata); auditErr != nil {
		return dto.StrengthForecastResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record deterministic replay audit", auditErr)
	}
	if !passed {
		return dto.StrengthForecastResponse{}, util.NewError(http.StatusConflict, util.CodeConflict, "deterministic replay did not match the immutable forecast result")
	}
	return dto.NewStrengthForecastResponse(after), nil
}

func (service *strengthForecastService) Compare(ctx context.Context, baseID, otherID uint) (dto.ForecastComparisonResponse, error) {
	base, err := service.forecasts.GetByID(ctx, baseID)
	if err != nil {
		return dto.ForecastComparisonResponse{}, util.NotFound("base strength forecast")
	}
	other, err := service.forecasts.GetByID(ctx, otherID)
	if err != nil {
		return dto.ForecastComparisonResponse{}, util.NotFound("compared strength forecast")
	}
	if base.PourSectionID != other.PourSectionID {
		return dto.ForecastComparisonResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "only forecasts for the same pour section can be compared")
	}
	return dto.ForecastComparisonResponse{
		BaseID: base.ID, ComparedID: other.ID,
		MaturityDelta:     roundDifference(other.MaturityDegreeHours - base.MaturityDegreeHours),
		StrengthDelta:     roundDifference(other.PredictedStrengthMPA - base.PredictedStrengthMPA),
		FormulaChanged:    base.FormulaVersion != other.FormulaVersion,
		InputChanged:      base.InputHash != other.InputHash,
		ConfidenceChanged: base.ConfidenceLevel != other.ConfidenceLevel,
	}, nil
}

func (service *strengthForecastService) prepareSnapshot(ctx context.Context, request dto.RunStrengthForecastRequest) (forecastSnapshot, string, string, error) {
	section, err := service.sections.GetByID(ctx, request.PourSectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return forecastSnapshot{}, "", "", util.NotFound("pour section")
		}
		return forecastSnapshot{}, "", "", util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load pour section", err)
	}
	series, err := service.series.GetByID(ctx, request.TemperatureSeriesID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return forecastSnapshot{}, "", "", util.NotFound("temperature series")
		}
		return forecastSnapshot{}, "", "", util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load temperature series", err)
	}
	if series.PourSectionID != section.ID {
		return forecastSnapshot{}, "", "", util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "temperature series does not belong to the selected pour section")
	}
	if series.SeriesState != constants.SeriesUsable {
		return forecastSnapshot{}, "", "", util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "only confirmed usable temperature series can be forecast")
	}
	mix, err := service.mixes.GetByID(ctx, section.MixDesignID)
	if err != nil {
		return forecastSnapshot{}, "", "", util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load mix design", err)
	}
	if mix.DesignState != constants.MixPublished {
		return forecastSnapshot{}, "", "", util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "the referenced mix design is not published")
	}
	var rawPoints []timeseries.Point
	if err := json.Unmarshal([]byte(series.PointsJSON), &rawPoints); err != nil {
		return forecastSnapshot{}, "", "", util.WrapError(http.StatusUnprocessableEntity, util.CodeValidation, "stored temperature sequence cannot be decoded", err)
	}
	points := make([]algorithm.TemperaturePoint, 0, len(rawPoints))
	for _, point := range rawPoints {
		points = append(points, algorithm.TemperaturePoint{Timestamp: point.Timestamp, TemperatureC: point.TemperatureC})
	}
	var calibration []algorithm.CalibrationPoint
	if err := json.Unmarshal([]byte(mix.CalibrationPointsJSON), &calibration); err != nil {
		return forecastSnapshot{}, "", "", util.WrapError(http.StatusUnprocessableEntity, util.CodeValidation, "published calibration points cannot be decoded", err)
	}
	snapshot := forecastSnapshot{
		PourSectionID: section.ID, SectionCode: section.SectionCode, TargetStrengthMPA: section.TargetStrengthMPA,
		TemperatureSeriesID: series.ID, SensorCode: series.SensorCode, SourceChecksum: series.SourceChecksum,
		MissingRatio: series.MissingRatio, MixDesignID: mix.ID, MixCode: mix.MixCode,
		MixDesignVersion: mix.Version, DatumTemperatureC: mix.DatumTemperatureC,
		CalibrationPoints: calibration, TemperaturePoints: points, FormulaVersion: algorithm.FormulaVersion,
	}
	encoded, hash, err := encodeAndHash(snapshot)
	if err != nil {
		return forecastSnapshot{}, "", "", util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to freeze forecast inputs", err)
	}
	return snapshot, encoded, hash, nil
}

func encodeAndHash(snapshot forecastSnapshot) (string, string, error) {
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return "", "", err
	}
	digest := sha256.Sum256(encoded)
	return string(encoded), hex.EncodeToString(digest[:]), nil
}

func roundDifference(value float64) float64 { return math.Round(value*10000) / 10000 }
