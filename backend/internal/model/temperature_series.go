package model

import "time"

type TemperatureSeries struct {
	ID                uint        `gorm:"primaryKey" json:"id"`
	PourSectionID     uint        `gorm:"not null;index" json:"pour_section_id"`
	PourSection       PourSection `gorm:"foreignKey:PourSectionID" json:"pour_section,omitempty"`
	SensorCode        string      `gorm:"size:80;not null;index" json:"sensor_code"`
	SampleIntervalMin int         `gorm:"not null" json:"sample_interval_min"`
	PointsJSON        string      `gorm:"type:text;not null" json:"points_json"`
	StartedAt         time.Time   `gorm:"not null" json:"started_at"`
	EndedAt           time.Time   `gorm:"not null" json:"ended_at"`
	SourceChecksum    string      `gorm:"size:64;not null;uniqueIndex" json:"source_checksum"`
	MissingRatio      float64     `gorm:"not null" json:"missing_ratio"`
	SeriesState       string      `gorm:"size:24;not null;index" json:"series_state"`
	QualityNote       string      `gorm:"type:text;not null" json:"quality_note"`
	ImportedBy        uint        `gorm:"not null;index" json:"imported_by"`
	ImportedByName    string      `gorm:"size:120;not null" json:"imported_by_name"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}
