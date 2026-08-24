package service

import (
	"concrete-curing-maturity-monitor/backend/internal/constants"
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/middleware"
	"concrete-curing-maturity-monitor/backend/internal/model"
	"concrete-curing-maturity-monitor/backend/internal/repository"
	"concrete-curing-maturity-monitor/backend/internal/util"
	"context"
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"
)

type PourSectionService interface {
	List(context.Context, dto.PourSectionQuery) (dto.PourSectionListResponse, error)
	Get(context.Context, uint) (dto.PourSectionResponse, error)
	Create(context.Context, dto.CreatePourSectionRequest, util.Actor) (dto.PourSectionResponse, error)
	Update(context.Context, uint, dto.UpdatePourSectionRequest, util.Actor) (dto.PourSectionResponse, error)
	Transition(context.Context, uint, dto.TransitionPourSectionRequest, util.Actor) (dto.PourSectionResponse, error)
}

type pourSectionService struct {
	sections repository.PourSectionRepository
	mixes    repository.MixDesignRepository
	audit    *middleware.AuditRecorder
	now      func() time.Time
}

func NewPourSectionService(
	sections repository.PourSectionRepository,
	mixes repository.MixDesignRepository,
	audit *middleware.AuditRecorder,
) PourSectionService {
	return &pourSectionService{sections: sections, mixes: mixes, audit: audit, now: func() time.Time { return time.Now().UTC() }}
}

func (service *pourSectionService) List(ctx context.Context, query dto.PourSectionQuery) (dto.PourSectionListResponse, error) {
	sections, total, err := service.sections.List(ctx, query)
	if err != nil {
		return dto.PourSectionListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to list pour sections", err)
	}
	items := make([]dto.PourSectionResponse, 0, len(sections))
	for _, section := range sections {
		summary, err := service.sections.Summary(ctx, section.ID)
		if err != nil {
			return dto.PourSectionListResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load pour section summary", err)
		}
		items = append(items, dto.NewPourSectionResponse(section, summary))
	}
	return dto.PourSectionListResponse{Items: items, Total: total, Page: query.Page, Size: query.PageSize}, nil
}

func (service *pourSectionService) Get(ctx context.Context, id uint) (dto.PourSectionResponse, error) {
	section, err := service.sections.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.PourSectionResponse{}, util.NotFound("pour section")
		}
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load pour section", err)
	}
	summary, err := service.sections.Summary(ctx, id)
	if err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load pour section summary", err)
	}
	return dto.NewPourSectionResponse(section, summary), nil
}

func (service *pourSectionService) Create(ctx context.Context, request dto.CreatePourSectionRequest, actor util.Actor) (dto.PourSectionResponse, error) {
	request.Normalize()
	mix, err := service.mixes.GetByID(ctx, request.MixDesignID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.PourSectionResponse{}, util.NotFound("mix design")
		}
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to validate mix design", err)
	}
	if mix.DesignState != constants.MixPublished {
		return dto.PourSectionResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "a pour section must reference a published mix design")
	}
	state := constants.CuringPrepared
	if request.PouredAt != nil {
		state = constants.CuringPoured
	}
	section := model.PourSection{
		SectionCode: request.SectionCode, Name: request.Name, StructurePart: request.StructurePart,
		VolumeM3: request.VolumeM3, MixDesignID: request.MixDesignID, PouredAt: request.PouredAt,
		TargetStrengthMPA: request.TargetStrengthMPA, CuringState: string(state),
		OwnerTeam: request.OwnerTeam, Version: 1,
	}
	if err := service.sections.Create(ctx, &section); err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusConflict, util.CodeConflict, "section code already exists or the section is invalid", err)
	}
	created, err := service.sections.GetByID(ctx, section.ID)
	if err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload created pour section", err)
	}
	if err := service.audit.Record(ctx, actor, "pour_section", section.ID, "create", nil, created, map[string]any{"mix_design_id": mix.ID, "mix_version": mix.Version}); err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record pour section audit", err)
	}
	return dto.NewPourSectionResponse(created, model.PourSectionSummary{}), nil
}

func (service *pourSectionService) Update(ctx context.Context, id uint, request dto.UpdatePourSectionRequest, actor util.Actor) (dto.PourSectionResponse, error) {
	request.Normalize()
	before, err := service.sections.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.PourSectionResponse{}, util.NotFound("pour section")
		}
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load pour section", err)
	}
	if before.CuringState == string(constants.CuringClosed) {
		return dto.PourSectionResponse{}, util.Conflict("closed pour sections are immutable")
	}
	updates := make(map[string]any)
	if request.Name != nil {
		updates["name"] = *request.Name
	}
	if request.StructurePart != nil {
		updates["structure_part"] = *request.StructurePart
	}
	if request.VolumeM3 != nil {
		updates["volume_m3"] = *request.VolumeM3
	}
	if request.TargetStrengthMPA != nil {
		updates["target_strength_mpa"] = *request.TargetStrengthMPA
	}
	if request.OwnerTeam != nil {
		updates["owner_team"] = *request.OwnerTeam
	}
	if request.MixDesignID != nil {
		if before.CuringState != string(constants.CuringPrepared) {
			return dto.PourSectionResponse{}, util.Conflict("mix design can only change before pouring")
		}
		mix, err := service.mixes.GetByID(ctx, *request.MixDesignID)
		if err != nil || mix.DesignState != constants.MixPublished {
			return dto.PourSectionResponse{}, util.NewError(http.StatusUnprocessableEntity, util.CodeValidation, "replacement mix design must be published")
		}
		updates["mix_design_id"] = *request.MixDesignID
	}
	if len(updates) == 0 {
		return dto.PourSectionResponse{}, util.NewError(http.StatusBadRequest, util.CodeValidation, "at least one editable field is required")
	}
	changed, err := service.sections.Update(ctx, id, request.Version, updates)
	if err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to update pour section", err)
	}
	if !changed {
		return dto.PourSectionResponse{}, util.Conflict("pour section version changed concurrently")
	}
	after, err := service.sections.GetByID(ctx, id)
	if err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload pour section", err)
	}
	if err := service.audit.Record(ctx, actor, "pour_section", id, "update", before, after, nil); err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record pour section audit", err)
	}
	summary, _ := service.sections.Summary(ctx, id)
	return dto.NewPourSectionResponse(after, summary), nil
}

func (service *pourSectionService) Transition(ctx context.Context, id uint, request dto.TransitionPourSectionRequest, actor util.Actor) (dto.PourSectionResponse, error) {
	before, err := service.sections.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return dto.PourSectionResponse{}, util.NotFound("pour section")
		}
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to load pour section", err)
	}
	from := constants.CuringState(before.CuringState)
	to := constants.CuringState(request.ToState)
	if !constants.CanTransitionCuring(from, to) {
		return dto.PourSectionResponse{}, util.NewError(http.StatusConflict, util.CodeInvalidTransition, "the requested curing state transition is not allowed")
	}
	if to == constants.CuringThresholdReached && actor.Role != constants.RoleReviewer && actor.Role != constants.RoleAdmin {
		return dto.PourSectionResponse{}, util.NewError(http.StatusForbidden, util.CodeForbidden, "only a reviewer may confirm the strength threshold")
	}
	updates := make(map[string]any)
	if to == constants.CuringPoured && before.PouredAt == nil {
		now := service.now()
		updates["poured_at"] = &now
	}
	changed, err := service.sections.Transition(ctx, id, request.Version, before.CuringState, request.ToState, updates)
	if err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to transition pour section", err)
	}
	if !changed {
		return dto.PourSectionResponse{}, util.Conflict("pour section state or version changed concurrently")
	}
	after, err := service.sections.GetByID(ctx, id)
	if err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to reload pour section", err)
	}
	if err := service.audit.Record(ctx, actor, "pour_section", id, "transition", before, after, map[string]any{"note": request.Note, "from": before.CuringState, "to": request.ToState}); err != nil {
		return dto.PourSectionResponse{}, util.WrapError(http.StatusInternalServerError, util.CodeInternal, "unable to record pour section transition", err)
	}
	summary, _ := service.sections.Summary(ctx, id)
	return dto.NewPourSectionResponse(after, summary), nil
}
