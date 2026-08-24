package model

import "time"

type PourSection struct {
	ID                uint       `gorm:"primaryKey" json:"id"`
	SectionCode       string     `gorm:"size:48;not null;uniqueIndex" json:"section_code"`
	Name              string     `gorm:"size:180;not null" json:"name"`
	StructurePart     string     `gorm:"size:180;not null;index" json:"structure_part"`
	VolumeM3          float64    `gorm:"not null" json:"volume_m3"`
	MixDesignID       uint       `gorm:"not null;index" json:"mix_design_id"`
	MixDesign         MixDesign  `gorm:"foreignKey:MixDesignID" json:"mix_design,omitempty"`
	PouredAt          *time.Time `json:"poured_at,omitempty"`
	TargetStrengthMPA float64    `gorm:"not null" json:"target_strength_mpa"`
	CuringState       string     `gorm:"size:32;not null;index" json:"curing_state"`
	OwnerTeam         string     `gorm:"size:140;not null;index" json:"owner_team"`
	Version           int        `gorm:"not null;default:1" json:"version"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type PourSectionSummary struct {
	TemperatureSeriesCount int64   `json:"temperature_series_count"`
	ForecastCount          int64   `json:"forecast_count"`
	LatestStrengthMPA      float64 `json:"latest_strength_mpa"`
	LatestConfidence       string  `json:"latest_confidence"`
}
