package repository

import (
	"concrete-curing-maturity-monitor/backend/internal/dto"
	"concrete-curing-maturity-monitor/backend/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type MixDesignRepository interface {
	List(context.Context, dto.MixDesignQuery) ([]model.MixDesign, int64, error)
	GetByID(context.Context, uint) (model.MixDesign, error)
	Create(context.Context, *model.MixDesign) error
	Update(context.Context, uint, int, map[string]any) (bool, error)
	Transition(context.Context, uint, int, string, string, map[string]any) (bool, error)
	ReferenceCount(context.Context, uint) (int64, error)
}

type mixDesignRepository struct{ db *gorm.DB }

func NewMixDesignRepository(db *gorm.DB) MixDesignRepository {
	return &mixDesignRepository{db: db}
}

func (repository *mixDesignRepository) List(ctx context.Context, query dto.MixDesignQuery) ([]model.MixDesign, int64, error) {
	database := repository.db.WithContext(ctx).Model(&model.MixDesign{})
	if query.Search != "" {
		like := "%" + query.Search + "%"
		database = database.Where("LOWER(mix_code) LIKE LOWER(?) OR LOWER(cement_type) LIKE LOWER(?)", like, like)
	}
	if query.DesignState != "" {
		database = database.Where("design_state = ?", query.DesignState)
	}
	var total int64
	if err := database.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count mix designs: %w", err)
	}
	var designs []model.MixDesign
	if err := database.Order("mix_code ASC, version DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&designs).Error; err != nil {
		return nil, 0, fmt.Errorf("list mix designs: %w", err)
	}
	return designs, total, nil
}

func (repository *mixDesignRepository) GetByID(ctx context.Context, id uint) (model.MixDesign, error) {
	var design model.MixDesign
	if err := repository.db.WithContext(ctx).First(&design, id).Error; err != nil {
		return model.MixDesign{}, fmt.Errorf("get mix design %d: %v", id, err)
	}
	return design, nil
}

func (repository *mixDesignRepository) Create(ctx context.Context, design *model.MixDesign) error {
	if err := repository.db.WithContext(ctx).Create(design).Error; err != nil {
		return fmt.Errorf("create mix design: %w", err)
	}
	return nil
}

func (repository *mixDesignRepository) Update(ctx context.Context, id uint, version int, updates map[string]any) (bool, error) {
	updates["lock_version"] = gorm.Expr("lock_version + 1")
	result := repository.db.WithContext(ctx).Model(&model.MixDesign{}).
		Where("id = ? AND lock_version = ? AND design_state = ?", id, version, "draft").Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("update mix design %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (repository *mixDesignRepository) Transition(ctx context.Context, id uint, version int, from, to string, updates map[string]any) (bool, error) {
	updates["design_state"] = to
	updates["lock_version"] = gorm.Expr("lock_version + 1")
	result := repository.db.WithContext(ctx).Model(&model.MixDesign{}).
		Where("id = ? AND lock_version = ? AND design_state = ?", id, version, from).Updates(updates)
	if result.Error != nil {
		return false, fmt.Errorf("transition mix design %d: %w", id, result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (repository *mixDesignRepository) ReferenceCount(ctx context.Context, id uint) (int64, error) {
	var count int64
	if err := repository.db.WithContext(ctx).Model(&model.PourSection{}).Where("mix_design_id = ?", id).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count mix design references: %w", err)
	}
	return count, nil
}
