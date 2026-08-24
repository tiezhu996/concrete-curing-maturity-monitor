package repository

import (
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type PourSectionRepository interface {
	List(context.Context, dto.PourSectionQuery) ([]model.PourSection, int64, error)
	GetByID(context.Context, uint) (model.PourSection, error)
	Create(context.Context, *model.PourSection) error
	Update(context.Context, uint, int, map[string]any) (bool, error)
	Transition(context.Context, uint, int, string, string, map[string]any) (bool, error)
	Summary(context.Context, uint) (model.PourSectionSummary, error)
}

type pourSectionRepository struct{ db *gorm.DB }

func NewPourSectionRepository(db *gorm.DB) PourSectionRepository {
	return &pourSectionRepository{db: db}
}

func (repository *pourSectionRepository) List(ctx context.Context, query dto.PourSectionQuery) ([]model.PourSection, int64, error) {
	database := repository.db.WithContext(ctx).Model(&model.PourSection{})
	if search := query.Search; search != "" {
		like := "%" + search + "%"
		database = database.Where("LOWER(section_code) LIKE LOWER(?) OR LOWER(name) LIKE LOWER(?) OR LOWER(structure_part) LIKE LOWER(?)", like, like, like)
	}
	if query.CuringState != "" {
		database = database.Where("curing_state = ?", query.CuringState)
	}
	if query.MixDesignID > 0 {
		database = database.Where("mix_design_id = ?", query.MixDesignID)
	}
	if query.OwnerTeam != "" {
		database = database.Where("owner_team = ?", query.OwnerTeam)
	}
	var total int64
	if err := database.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count pour sections: %w", err)
	}
	var sections []model.PourSection
	offset := (query.Page - 1) * query.PageSize
	if err := database.Preload("MixDesign").Order("updated_at DESC").Offset(offset).Limit(query.PageSize).Find(&sections).Error; err != nil {
		return nil, 0, fmt.Errorf("list pour sections: %w", err)
	}
	return sections, total, nil
}

func (repository *pourSectionRepository) GetByID(ctx context.Context, id uint) (model.PourSection, error) {
	var section model.PourSection
	if err := repository.db.WithContext(ctx).Preload("MixDesign").First(&section, id).Error; err != nil {
		return model.PourSection{}, fmt.Errorf("get pour section %d: %w", id, err)
	}
	return section, nil
}

func (repository *pourSectionRepository) Create(ctx context.Context, section *model.PourSection) error {
	if err := repository.db.WithContext(ctx).Create(section).Error; err != nil {
		return fmt.Errorf("create pour section: %w", err)
	}
	return nil
}

func (repository *pourSectionRepository) Update(ctx context.Context, id uint, version int, updates map[string]any) (bool, error) {
	updates["version"] = gorm.Expr("version + 1")
	result := repository.db.WithContext(ctx).Model(&model.PourSection{}).Where("id = ? AND version = ?", id, version).Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("update pour section %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (repository *pourSectionRepository) Transition(ctx context.Context, id uint, version int, from, to string, updates map[string]any) (bool, error) {
	updates["curing_state"] = to
	updates["version"] = gorm.Expr("version + 1")
	result := repository.db.WithContext(ctx).Model(&model.PourSection{}).
		Where("id = ? AND version = ? AND curing_state = ?", id, version, from).Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("transition pour section %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (repository *pourSectionRepository) Summary(ctx context.Context, sectionID uint) (model.PourSectionSummary, error) {
	var summary model.PourSectionSummary
	if err := repository.db.WithContext(ctx).Model(&model.TemperatureSeries{}).Where("pour_section_id = ?", sectionID).Count(&summary.TemperatureSeriesCount).Error; err != nil {
		return summary, fmt.Errorf("count section temperature series: %w", err)
	}
	if err := repository.db.WithContext(ctx).Model(&model.StrengthForecast{}).Where("pour_section_id = ?", sectionID).Count(&summary.ForecastCount).Error; err != nil {
		return summary, fmt.Errorf("count section forecasts: %w", err)
	}
	var latest model.StrengthForecast
	err := repository.db.WithContext(ctx).Where("pour_section_id = ? AND forecast_state IN ?", sectionID, []string{"completed", "reviewed", "confirmed"}).Order("calculated_at DESC").First(&latest).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return summary, fmt.Errorf("load latest section forecast: %w", err)
	}
	if err == nil {
		summary.LatestStrengthMPA = latest.PredictedStrengthMPA
		summary.LatestConfidence = latest.ConfidenceLevel
	}
	return summary, nil
}
