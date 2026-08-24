package model

import "time"

type StrengthForecast struct {
	ID                    uint              `gorm:"primaryKey" json:"id"`
	PourSectionID         uint              `gorm:"not null;index" json:"pour_section_id"`
	PourSection           PourSection       `gorm:"foreignKey:PourSectionID" json:"pour_section,omitempty"`
	TemperatureSeriesID   uint              `gorm:"not null;index" json:"temperature_series_id"`
	TemperatureSeries     TemperatureSeries `gorm:"foreignKey:TemperatureSeriesID" json:"temperature_series,omitempty"`
	MixDesignVersion      int               `gorm:"not null" json:"mix_design_version"`
	FormulaVersion        string            `gorm:"size:48;not null;index;uniqueIndex:idx_forecast_input" json:"formula_version"`
	InputHash             string            `gorm:"size:64;not null;index;uniqueIndex:idx_forecast_input" json:"input_hash"`
	InputSnapshot         string            `gorm:"type:text;not null" json:"input_snapshot"`
	MaturityDegreeHours   float64           `gorm:"not null" json:"maturity_degree_hours"`
	PredictedStrengthMPA  float64           `gorm:"not null" json:"predicted_strength_mpa"`
	ThresholdETA          *time.Time        `json:"threshold_eta,omitempty"`
	ConfidenceLevel       string            `gorm:"size:16;not null;index" json:"confidence_level"`
	ForecastState         string            `gorm:"size:24;not null;index" json:"forecast_state"`
	Explanation           string            `gorm:"type:text;not null" json:"explanation"`
	CalculatedBy          uint              `gorm:"not null;index" json:"calculated_by"`
	CalculatedByName      string            `gorm:"size:120;not null" json:"calculated_by_name"`
	CalculatedAt          time.Time         `gorm:"not null" json:"calculated_at"`
	ReviewedBy            *uint             `json:"reviewed_by,omitempty"`
	ReviewedByName        string            `gorm:"size:120" json:"reviewed_by_name,omitempty"`
	ReviewedAt            *time.Time        `json:"reviewed_at,omitempty"`
	ConfirmedBy           *uint             `json:"confirmed_by,omitempty"`
	ConfirmedAt           *time.Time        `json:"confirmed_at,omitempty"`
	IdempotencyKey        string            `gorm:"size:100;not null;uniqueIndex:idx_actor_idempotency" json:"idempotency_key"`
	IdempotencyActorID    uint              `gorm:"not null;uniqueIndex:idx_actor_idempotency" json:"idempotency_actor_id"`
	DurationMilliseconds  int64             `gorm:"not null" json:"duration_milliseconds"`
	FailureReason         string            `gorm:"type:text" json:"failure_reason,omitempty"`
	DeterminismReplayPass *bool             `json:"determinism_replay_passed,omitempty"`
	CreatedAt             time.Time         `json:"created_at"`
	UpdatedAt             time.Time         `json:"updated_at"`
}
