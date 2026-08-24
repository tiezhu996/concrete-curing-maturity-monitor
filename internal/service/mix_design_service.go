package service

import (
	"concrete-curing-maturity-monitor/backend/internal/algorithm"
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/middleware"
	"concrete-curing-maturity-monitor/backend/internal/model"
	"concrete-curing-maturity-monitor/backend/internal/repository"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type MixDesignService interface {
	List(context.Context, dto.MixDesignQuery) (dto.MixDesignListResponse, error)
	Get(context.Context, uint) (dto.MixDesignResponse, error)
	Create(context.Context, dto.CreateMixDesignRequest, util.Actor) (dto.MixDesignResponse, error)
	Update(context.Context, uint, dto.UpdateMixDesignRequest, util.Actor) (dto.MixDesignResponse, error)
	Transition(context.Context, uint, string, dto.MixDesignActionRequest, util.Actor) (dto.MixDesignResponse, error)
}

type mixDesignService struct {
	designs repository.MixDesignRepository
	audit   *middleware.AuditRecorder
	now     func() time.Time
}

func NewMixDesignService(designs repository.MixDesignRepository, audit *middleware.AuditRecorder) MixDesignService {
	return &mixDesignService{designs: designs, audit: audit, now: func() time.Time { return time.Now().UTC() }}
}

func (service *mixDesignService) List(ctx context.Context, query dto.MixDesignQuery) (dto.MixDesignListResponse, error) {
	designs, total, err := service.designs.List(ctx, query)
	if err != nil {
		return dto.MixDesignListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list mix designs", err)
	}
	items := make([]dto.MixDesignResponse, 0, len(designs))
	for _, design := range designs {
		references, err := service.designs.ReferenceCount(ctx, design.ID)
		if err != nil {
			return dto.MixDesignListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to count mix references", err)
		}
		items = append(items, dto.NewMixDesignResponse(design, references))
	}
	return dto.MixDesignListResponse{Items: items, Total: total, Page: query.Page, Size: query.PageSize}, nil
}

func (service *mixDesignService) Get(ctx context.Context, id uint) (dto.MixDesignResponse, error) {
	design, err := service.designs.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.MixDesignResponse{}, util.NotFound("mix design")
		}
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load mix design", err)
	}
	references, err := service.designs.ReferenceCount(ctx, id)
	if err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to count mix references", err)
	}
	return dto.NewMixDesignResponse(design, references), nil
}

func (service *mixDesignService) Create(ctx context.Context, request dto.CreateMixDesignRequest, actor util.Actor) (dto.MixDesignResponse, error) {
	request.Normalize()
	if err := validateCalibrationPoints(request.CalibrationPoints); err != nil {
		return dto.MixDesignResponse{}, util.ErrorWithDetails(http.StatusUnprocessableEntity, util.CodeValidation, "calibration points are not valid", err.Error())
	}
	if request.ValidFrom != nil && request.ValidTo != nil && !request.ValidTo.After(*request.ValidFrom) {
		return dto.MixDesignResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "valid_to must be later than valid_from")
	}
	encoded, err := json.Marshal(request.CalibrationPoints)
	if err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to encode calibration points", err)
	}
	design := model.MixDesign{
		MixCode: request.MixCode, Version: request.Version, CementType: request.CementType,
		WaterBinderRatio: request.WaterBinderRatio, DatumTemperatureC: request.DatumTemperatureC,
		CalibrationPointsJSON: string(encoded), ValidFrom: request.ValidFrom, ValidTo: request.ValidTo,
		DesignState: constants.MixDraft, CreatedBy: actor.UserID, CreatedByName: actor.DisplayName, LockVersion: 1,
	}
	if err := service.designs.Create(ctx, &design); err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusConflict, util.CodeConflict, "mix code and version must be unique", err)
	}
	if err := service.audit.Record(ctx, actor, "mix_design", design.ID, "create", nil, design, map[string]any{"calibration_point_count": len(request.CalibrationPoints)}); err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record mix design audit", err)
	}
	return dto.NewMixDesignResponse(design, 0), nil
}

func (service *mixDesignService) Update(ctx context.Context, id uint, request dto.UpdateMixDesignRequest, actor util.Actor) (dto.MixDesignResponse, error) {
	request.Normalize()
	before, err := service.designs.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.MixDesignResponse{}, util.NotFound("mix design")
		}
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load mix design", err)
	}
	if before.DesignState != constants.MixDraft {
		return dto.MixDesignResponse{}, util.Conflict("only draft mix designs can be edited; create a new version instead")
	}
	updates := make(map[string]any)
	if request.CementType != nil {
		updates["cement_type"] = *request.CementType
	}
	if request.WaterBinderRatio != nil {
		updates["water_binder_ratio"] = *request.WaterBinderRatio
	}
	if request.DatumTemperatureC != nil {
		updates["datum_temperature_c"] = *request.DatumTemperatureC
	}
	if request.CalibrationPoints != nil {
		if err := validateCalibrationPoints(request.CalibrationPoints); err != nil {
			return dto.MixDesignResponse{}, util.ErrorWithDetails(http.StatusUnprocessableEntity, util.CodeValidation, "calibration points are not valid", err.Error())
		}
		encoded, _ := json.Marshal(request.CalibrationPoints)
		updates["calibration_points_json"] = string(encoded)
	}
	if request.ValidFrom != nil {
		updates["valid_from"] = request.ValidFrom
	}
	if request.ValidTo != nil {
		updates["valid_to"] = request.ValidTo
	}
	if len(updates) == 0 {
		return dto.MixDesignResponse{}, util.NewError(http.StatusBadRequest, util.CodeValidation, "at least one editable field is required")
	}
	changed, err := service.designs.Update(ctx, id, request.LockVersion, updates)
	if err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to update mix design", err)
	}
	if !changed {
		return dto.MixDesignResponse{}, util.Conflict("mix design state or version changed concurrently")
	}
	after, err := service.designs.GetByID(ctx, id)
	if err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload mix design", err)
	}
	if err := service.audit.Record(ctx, actor, "mix_design", id, "update", before, after, nil); err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record mix design audit", err)
	}
	references, _ := service.designs.ReferenceCount(ctx, id)
	return dto.NewMixDesignResponse(after, references), nil
}

func (service *mixDesignService) Transition(ctx context.Context, id uint, to string, request dto.MixDesignActionRequest, actor util.Actor) (dto.MixDesignResponse, error) {
	before, err := service.designs.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return dto.MixDesignResponse{}, util.NotFound("mix design")
		}
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load mix design", err)
	}
	if !constants.CanTransitionMix(before.DesignState, to) {
		return dto.MixDesignResponse{}, util.NewError(http.StatusConflict, util.CodeInvalidTransition, "the requested mix design transition is not allowed")
	}
	if to == constants.MixValidated || to == constants.MixPublished {
		var points []dto.CalibrationPoint
		if err := json.Unmarshal([]byte(before.CalibrationPointsJSON), &points); err != nil {
			return dto.MixDesignResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "stored calibration points cannot be decoded")
		}
		if err := validateCalibrationPoints(points); err != nil {
			return dto.MixDesignResponse{}, util.ErrorWithDetails(http.StatusUnprocessableEntity, util.CodeValidation, "calibration points are not valid", err.Error())
		}
	}
	updates := make(map[string]any)
	now := service.now()
	if to == constants.MixPublished && before.ValidFrom == nil {
		updates["valid_from"] = &now
	}
	if to == constants.MixRetired {
		updates["valid_to"] = &now
	}
	changed, err := service.designs.Transition(ctx, id, request.LockVersion, before.DesignState, to, updates)
	if err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to transition mix design", err)
	}
	if !changed {
		return dto.MixDesignResponse{}, util.Conflict("mix design state or version changed concurrently")
	}
	after, err := service.designs.GetByID(ctx, id)
	if err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload mix design", err)
	}
	if err := service.audit.Record(ctx, actor, "mix_design", id, "transition", before, after, map[string]any{"note": request.Note, "from": before.DesignState, "to": to}); err != nil {
		return dto.MixDesignResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record mix design transition", err)
	}
	references, _ := service.designs.ReferenceCount(ctx, id)
	return dto.NewMixDesignResponse(after, references), nil
}

func validateCalibrationPoints(points []dto.CalibrationPoint) error {
	converted := make([]algorithm.CalibrationPoint, 0, len(points))
	for _, point := range points {
		converted = append(converted, algorithm.CalibrationPoint{MaturityDegreeHours: point.MaturityDegreeHours, StrengthMPA: point.StrengthMPA})
	}
	return algorithm.ValidateCalibration(converted)
}
