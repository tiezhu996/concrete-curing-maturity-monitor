package dto

import (
	"concrete-curing-maturity-monitor/backend/internal/model"
	"encoding/json"
	"strings"
	"time"
)

type TemperaturePoint struct {
	Timestamp    time.Time `json:"timestamp" binding:"required"`
	TemperatureC float64   `json:"temperature_c" binding:"gte=-50,lte=120"`
}

type ImportTemperatureSeriesRequest struct {
	PourSectionID     uint               `json:"pour_section_id" binding:"required"`
	SensorCode        string             `json:"sensor_code" binding:"required,min=2,max=80"`
	SampleIntervalMin int                `json:"sample_interval_min" binding:"required,min=1,max=1440"`
	Points            []TemperaturePoint `json:"points" binding:"required,min=2,max=10000,dive"`
	QualityNote       string             `json:"quality_note" binding:"omitempty,max=2000"`
}

func (request *ImportTemperatureSeriesRequest) Normalize() {
	request.SensorCode = strings.ToUpper(strings.TrimSpace(request.SensorCode))
	request.QualityNote = strings.TrimSpace(request.QualityNote)
}

type TemperatureSeriesActionRequest struct {
	Reason string `json:"reason" binding:"required,min=3,max=1000"`
}

type TemperatureSeriesQuery struct {
	PourSectionID uint
	SeriesState   string
	SensorCode    string
	Page          int
	PageSize      int
}

type TemperatureSeriesResponse struct {
	ID                uint               `json:"id"`
	PourSectionID     uint               `json:"pour_section_id"`
	SectionCode       string             `json:"section_code"`
	SensorCode        string             `json:"sensor_code"`
	SampleIntervalMin int                `json:"sample_interval_min"`
	Points            []TemperaturePoint `json:"points"`
	StartedAt         time.Time          `json:"started_at"`
	EndedAt           time.Time          `json:"ended_at"`
	SourceChecksum    string             `json:"source_checksum"`
	MissingRatio      float64            `json:"missing_ratio"`
	SeriesState       string             `json:"series_state"`
	QualityNote       string             `json:"quality_note"`
	ImportedBy        uint               `json:"imported_by"`
	ImportedByName    string             `json:"imported_by_name"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

type TemperatureSeriesListResponse struct {
	Items []TemperatureSeriesResponse `json:"items"`
	Total int64                       `json:"total"`
	Page  int                         `json:"page"`
	Size  int                         `json:"page_size"`
}

var sharedSeriesPoints []TemperaturePoint

func NewTemperatureSeriesResponse(series model.TemperatureSeries) TemperatureSeriesResponse {
	sharedSeriesPoints = sharedSeriesPoints[:0]
	_ = json.Unmarshal([]byte(series.PointsJSON), &sharedSeriesPoints)
	points := sharedSeriesPoints
	return TemperatureSeriesResponse{
		ID: series.ID, PourSectionID: series.PourSectionID, SectionCode: series.PourSection.SectionCode,
		SensorCode: series.SensorCode, SampleIntervalMin: series.SampleIntervalMin,
		Points: points, StartedAt: series.StartedAt, EndedAt: series.EndedAt,
		SourceChecksum: series.SourceChecksum, MissingRatio: series.MissingRatio,
		SeriesState: series.SeriesState, QualityNote: series.QualityNote,
		ImportedBy: series.ImportedBy, ImportedByName: series.ImportedByName,
		CreatedAt: series.CreatedAt, UpdatedAt: series.UpdatedAt,
	}
}
