package repository

import (
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type TemperatureSeriesRepository interface {
	List(context.Context, dto.TemperatureSeriesQuery) ([]model.TemperatureSeries, int64, error)
	GetByID(context.Context, uint) (model.TemperatureSeries, error)
	Create(context.Context, *model.TemperatureSeries) error
	SetState(context.Context, uint, []string, string, map[string]any) (bool, error)
}

type temperatureSeriesRepository struct{ db *gorm.DB }

func NewTemperatureSeriesRepository(db *gorm.DB) TemperatureSeriesRepository {
	return &temperatureSeriesRepository{db: db}
}

func (repository *temperatureSeriesRepository) List(ctx context.Context, query dto.TemperatureSeriesQuery) ([]model.TemperatureSeries, int64, error) {
	database := repository.db.WithContext(ctx).Model(&model.TemperatureSeries{})
	if query.PourSectionID > 0 {
		database = database.Where("pour_section_id = ?", query.PourSectionID)
	}
	if query.SeriesState != "" {
		database = database.Where("series_state = ?", query.SeriesState)
	}
	if query.SensorCode != "" {
		database = database.Where("LOWER(sensor_code) LIKE LOWER(?)", "%"+query.SensorCode+"%")
	}
	var total int64
	if err := database.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count temperature series: %w", err)
	}
	var series []model.TemperatureSeries
	if err := database.Preload("PourSection").Order("started_at DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&series).Error; err != nil {
		return nil, 0, fmt.Errorf("list temperature series: %w", err)
	}
	return series, total, nil
}

func (repository *temperatureSeriesRepository) GetByID(ctx context.Context, id uint) (model.TemperatureSeries, error) {
	var series model.TemperatureSeries
	if err := repository.db.WithContext(ctx).Preload("PourSection.MixDesign").First(&series, id).Error; err != nil {
		return model.TemperatureSeries{}, fmt.Errorf("get temperature series %d: %w", id, err)
	}
	return series, nil
}

func (repository *temperatureSeriesRepository) Create(ctx context.Context, series *model.TemperatureSeries) error {
	if err := repository.db.WithContext(ctx).Create(series).Error; err != nil {
		return fmt.Errorf("create temperature series: %w", err)
	}
	return nil
}

func (repository *temperatureSeriesRepository) SetState(ctx context.Context, id uint, from []string, to string, updates map[string]any) (bool, error) {
	updates["series_state"] = to
	result := repository.db.WithContext(ctx).Model(&model.TemperatureSeries{}).Where("id = ? AND series_state IN ?", id, from).Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("change temperature series %d state: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}

var _ TemperatureSeriesRepository = (*temperatureSeriesRepository)(nil)
var _ = gorm.ErrRecordNotFound
