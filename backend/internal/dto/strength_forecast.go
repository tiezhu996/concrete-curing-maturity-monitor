package dto

import (
	"concrete-curing-maturity-monitor/backend/internal/model"
	"encoding/json"
	"time"
)

type RunStrengthForecastRequest struct {
	PourSectionID       uint `json:"pour_section_id" binding:"required"`
	TemperatureSeriesID uint `json:"temperature_series_id" binding:"required"`
}

type ForecastActionRequest struct {
	Note string `json:"note" binding:"required,min=3,max=1000"`
}

type StrengthForecastQuery struct {
	PourSectionID       uint
	TemperatureSeriesID uint
	ForecastState       string
	Page                int
	PageSize            int
}

type StrengthForecastResponse struct {
	ID                    uint            `json:"id"`
	PourSectionID         uint            `json:"pour_section_id"`
	SectionCode           string          `json:"section_code"`
	TemperatureSeriesID   uint            `json:"temperature_series_id"`
	SensorCode            string          `json:"sensor_code"`
	MixDesignVersion      int             `json:"mix_design_version"`
	FormulaVersion        string          `json:"formula_version"`
	InputHash             string          `json:"input_hash"`
	InputSnapshot         json.RawMessage `json:"input_snapshot"`
	MaturityDegreeHours   float64         `json:"maturity_degree_hours"`
	PredictedStrengthMPA  float64         `json:"predicted_strength_mpa"`
	ThresholdETA          *time.Time      `json:"threshold_eta,omitempty"`
	ConfidenceLevel       string          `json:"confidence_level"`
	ForecastState         string          `json:"forecast_state"`
	Explanation           json.RawMessage `json:"explanation"`
	CalculatedBy          uint            `json:"calculated_by"`
	CalculatedByName      string          `json:"calculated_by_name"`
	CalculatedAt          time.Time       `json:"calculated_at"`
	ReviewedBy            *uint           `json:"reviewed_by,omitempty"`
	ReviewedByName        string          `json:"reviewed_by_name,omitempty"`
	ReviewedAt            *time.Time      `json:"reviewed_at,omitempty"`
	ConfirmedBy           *uint           `json:"confirmed_by,omitempty"`
	ConfirmedAt           *time.Time      `json:"confirmed_at,omitempty"`
	IdempotencyKey        string          `json:"idempotency_key"`
	DurationMilliseconds  int64           `json:"duration_milliseconds"`
	FailureReason         string          `json:"failure_reason,omitempty"`
	DeterminismReplayPass *bool           `json:"determinism_replay_passed,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
}

type StrengthForecastListResponse struct {
	Items []StrengthForecastResponse `json:"items"`
	Total int64                      `json:"total"`
	Page  int                        `json:"page"`
	Size  int                        `json:"page_size"`
}

type ForecastComparisonResponse struct {
	BaseID            uint    `json:"base_id"`
	ComparedID        uint    `json:"compared_id"`
	MaturityDelta     float64 `json:"maturity_delta"`
	StrengthDelta     float64 `json:"strength_delta_mpa"`
	FormulaChanged    bool    `json:"formula_changed"`
	InputChanged      bool    `json:"input_changed"`
	ConfidenceChanged bool    `json:"confidence_changed"`
}

func NewStrengthForecastResponse(forecast model.StrengthForecast) StrengthForecastResponse {
	return StrengthForecastResponse{
		ID: forecast.ID, PourSectionID: forecast.PourSectionID,
		SectionCode:         forecast.PourSection.SectionCode,
		TemperatureSeriesID: forecast.TemperatureSeriesID,
		SensorCode:          forecast.TemperatureSeries.SensorCode,
		MixDesignVersion:    forecast.MixDesignVersion, FormulaVersion: forecast.FormulaVersion,
		InputHash: forecast.InputHash, InputSnapshot: validRaw(forecast.InputSnapshot),
		MaturityDegreeHours:  forecast.MaturityDegreeHours,
		PredictedStrengthMPA: forecast.PredictedStrengthMPA, ThresholdETA: forecast.ThresholdETA,
		ConfidenceLevel: forecast.ConfidenceLevel, ForecastState: forecast.ForecastState,
		Explanation: validRaw(forecast.Explanation), CalculatedBy: forecast.CalculatedBy,
		CalculatedByName: forecast.CalculatedByName, CalculatedAt: forecast.CalculatedAt,
		ReviewedBy: forecast.ReviewedBy, ReviewedByName: forecast.ReviewedByName,
		ReviewedAt: forecast.ReviewedAt, ConfirmedBy: forecast.ConfirmedBy,
		ConfirmedAt: forecast.ConfirmedAt, IdempotencyKey: forecast.IdempotencyKey,
		DurationMilliseconds: forecast.DurationMilliseconds, FailureReason: forecast.FailureReason,
		DeterminismReplayPass: forecast.DeterminismReplayPass, CreatedAt: forecast.CreatedAt,
	}
}

func validRaw(value string) json.RawMessage {
	if !json.Valid([]byte(value)) {
		return json.RawMessage("null")
	}
	return json.RawMessage(value)
}
