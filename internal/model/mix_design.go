package model

import "time"

type MixDesign struct {
	ID                    uint       `gorm:"primaryKey" json:"id"`
	MixCode               string     `gorm:"size:60;not null;uniqueIndex:idx_mix_version" json:"mix_code"`
	Version               int        `gorm:"not null;uniqueIndex:idx_mix_version" json:"version"`
	CementType            string     `gorm:"size:140;not null" json:"cement_type"`
	WaterBinderRatio      float64    `gorm:"not null" json:"water_binder_ratio"`
	DatumTemperatureC     float64    `gorm:"not null" json:"datum_temperature_c"`
	CalibrationPointsJSON string     `gorm:"type:text;not null" json:"calibration_points_json"`
	ValidFrom             *time.Time `json:"valid_from,omitempty"`
	ValidTo               *time.Time `json:"valid_to,omitempty"`
	DesignState           string     `gorm:"size:24;not null;index" json:"design_state"`
	CreatedBy             uint       `gorm:"not null;index" json:"created_by"`
	CreatedByName         string     `gorm:"size:120;not null" json:"created_by_name"`
	LockVersion           int        `gorm:"not null;default:1" json:"lock_version"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}
