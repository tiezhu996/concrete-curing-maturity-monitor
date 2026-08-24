package repository

import (
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type StrengthForecastRepository interface {
	List(context.Context, dto.StrengthForecastQuery) ([]model.StrengthForecast, int64, error)
	GetByID(context.Context, uint) (model.StrengthForecast, error)
	GetByIdempotency(context.Context, uint, string) (model.StrengthForecast, error)
	GetByInputHash(context.Context, string, string) (model.StrengthForecast, error)
	Create(context.Context, *model.StrengthForecast) error
	Transition(context.Context, uint, []string, string, map[string]any) (bool, error)
	UpdateReplay(context.Context, uint, bool) error
}

type strengthForecastRepository struct{ db *gorm.DB }

func NewStrengthForecastRepository(db *gorm.DB) StrengthForecastRepository {
	return &strengthForecastRepository{db: db}
}

func (repository *strengthForecastRepository) List(ctx context.Context, query dto.StrengthForecastQuery) ([]model.StrengthForecast, int64, error) {
	database := repository.db.Model(&model.StrengthForecast{})
	if query.PourSectionID > 0 {
		database = database.Where("pour_section_id = ?", query.PourSectionID)
	}
	if query.TemperatureSeriesID > 0 {
		database = database.Where("temperature_series_id = ?", query.TemperatureSeriesID)
	}
	if query.ForecastState != "" {
		database = database.Where("forecast_state = ?", query.ForecastState)
	}
	var total int64
	if err := database.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count strength forecasts: %w", err)
	}
	var forecasts []model.StrengthForecast
	if err := database.Preload("PourSection").Preload("TemperatureSeries").Order("calculated_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&forecasts).Error; err != nil {
		return nil, 0, fmt.Errorf("list strength forecasts: %w", err)
	}
	return forecasts, total, nil
}

func (repository *strengthForecastRepository) GetByID(ctx context.Context, id uint) (model.StrengthForecast, error) {
	var forecast model.StrengthForecast
	if err := repository.db.Preload("PourSection.MixDesign").Preload("TemperatureSeries").First(&forecast, id).Error; err != nil {
		return model.StrengthForecast{}, fmt.Errorf("get strength forecast %d: %w", id, err)
	}
	return forecast, nil
}

func (repository *strengthForecastRepository) GetByIdempotency(ctx context.Context, actorID uint, key string) (model.StrengthForecast, error) {
	var forecast model.StrengthForecast
	if err := repository.db.Preload("PourSection").Preload("TemperatureSeries").Where("idempotency_actor_id = ? AND idempotency_key = ?", actorID, key).First(&forecast).Error; err != nil {
		return model.StrengthForecast{}, fmt.Errorf("get idempotent strength forecast: %w", err)
	}
	return forecast, nil
}

func (repository *strengthForecastRepository) GetByInputHash(ctx context.Context, inputHash, formulaVersion string) (model.StrengthForecast, error) {
	var forecast model.StrengthForecast
	if err := repository.db.WithContext(ctx).Preload("PourSection").Preload("TemperatureSeries").
		Where("input_hash = ? AND formula_version = ?", inputHash, formulaVersion).First(&forecast).Error; err != nil {
		return model.StrengthForecast{}, fmt.Errorf("get forecast by frozen input: %w", err)
	}
	return forecast, nil
}

func (repository *strengthForecastRepository) Create(ctx context.Context, forecast *model.StrengthForecast) error {
	if err := repository.db.WithContext(ctx).Create(forecast).Error; err != nil {
		return fmt.Errorf("create strength forecast: %w", err)
	}
	return nil
}

func (repository *strengthForecastRepository) Transition(ctx context.Context, id uint, from []string, to string, updates map[string]any) (bool, error) {
	updates["forecast_state"] = to
	result := repository.db.WithContext(ctx).Model(&model.StrengthForecast{}).Where("id = ? AND forecast_state IN ?", id, from).Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("transition strength forecast %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (repository *strengthForecastRepository) UpdateReplay(ctx context.Context, id uint, passed bool) error {
	if err := repository.db.WithContext(ctx).Model(&model.StrengthForecast{}).Where("id = ?", id).Update("determinism_replay_pass", passed).Error; err != nil {
		return fmt.Errorf("store deterministic replay result: %w", err)
	}
	return nil
}

var _ StrengthForecastRepository = (*strengthForecastRepository)(nil)
var _ = gorm.ErrRecordNotFound
