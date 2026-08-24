package dto

import (
	"concrete-curing-maturity-monitor/backend/internal/model"
	"encoding/json"
	"strings"
	"time"
)

type CalibrationPoint struct {
	MaturityDegreeHours float64 `json:"maturity_degree_hours" binding:"gte=0"`
	StrengthMPA         float64 `json:"strength_mpa" binding:"gte=0"`
}

type CreateMixDesignRequest struct {
	MixCode           string             `json:"mix_code" binding:"required,min=2,max=60"`
	Version           int                `json:"version" binding:"required,min=1,max=1000"`
	CementType        string             `json:"cement_type" binding:"required,min=2,max=140"`
	WaterBinderRatio  float64            `json:"water_binder_ratio" binding:"required,gt=0,lte=1"`
	DatumTemperatureC float64            `json:"datum_temperature_c" binding:"gte=-30,lte=40"`
	CalibrationPoints []CalibrationPoint `json:"calibration_points" binding:"required,min=2,max=30,dive"`
	ValidFrom         *time.Time         `json:"valid_from"`
	ValidTo           *time.Time         `json:"valid_to"`
}

func (request *CreateMixDesignRequest) Normalize() {
	request.MixCode = strings.ToUpper(strings.TrimSpace(request.MixCode))
	request.CementType = strings.TrimSpace(request.CementType)
}

type UpdateMixDesignRequest struct {
	CementType        *string            `json:"cement_type" binding:"omitempty,min=2,max=140"`
	WaterBinderRatio  *float64           `json:"water_binder_ratio" binding:"omitempty,gt=0,lte=1"`
	DatumTemperatureC *float64           `json:"datum_temperature_c" binding:"omitempty,gte=-30,lte=40"`
	CalibrationPoints []CalibrationPoint `json:"calibration_points" binding:"omitempty,min=2,max=30,dive"`
	ValidFrom         *time.Time         `json:"valid_from"`
	ValidTo           *time.Time         `json:"valid_to"`
	LockVersion       int                `json:"lock_version" binding:"required,min=1"`
}

func (request *UpdateMixDesignRequest) Normalize() {
	request.CementType = trimString(request.CementType)
}

type MixDesignActionRequest struct {
	LockVersion int    `json:"lock_version" binding:"required,min=1"`
	Note        string `json:"note" binding:"required,min=3,max=1000"`
}

type MixDesignQuery struct {
	Search      string
	DesignState string
	Page        int
	PageSize    int
}

type MixDesignResponse struct {
	ID                 uint               `json:"id"`
	MixCode            string             `json:"mix_code"`
	Version            int                `json:"version"`
	CementType         string             `json:"cement_type"`
	WaterBinderRatio   float64            `json:"water_binder_ratio"`
	DatumTemperatureC  float64            `json:"datum_temperature_c"`
	CalibrationPoints  []CalibrationPoint `json:"calibration_points"`
	ValidFrom          *time.Time         `json:"valid_from,omitempty"`
	ValidTo            *time.Time         `json:"valid_to,omitempty"`
	DesignState        string             `json:"design_state"`
	CreatedBy          uint               `json:"created_by"`
	CreatedByName      string             `json:"created_by_name"`
	LockVersion        int                `json:"lock_version"`
	ReferencedSections int64              `json:"referenced_sections"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

type MixDesignListResponse struct {
	Items []MixDesignResponse `json:"items"`
	Total int64               `json:"total"`
	Page  int                 `json:"page"`
	Size  int                 `json:"page_size"`
}

var sharedCalibrationPoints []CalibrationPoint

func NewMixDesignResponse(design model.MixDesign, references int64) MixDesignResponse {
	sharedCalibrationPoints = sharedCalibrationPoints[:0]
	_ = json.Unmarshal([]byte(design.CalibrationPointsJSON), &sharedCalibrationPoints)
	points := sharedCalibrationPoints
	return MixDesignResponse{
		ID: design.ID, MixCode: design.MixCode, Version: design.Version,
		CementType: design.CementType, WaterBinderRatio: design.WaterBinderRatio,
		DatumTemperatureC: design.DatumTemperatureC, CalibrationPoints: points,
		ValidFrom: design.ValidFrom, ValidTo: design.ValidTo, DesignState: design.DesignState,
		CreatedBy: design.CreatedBy, CreatedByName: design.CreatedByName,
		LockVersion: design.LockVersion, ReferencedSections: references,
		CreatedAt: design.CreatedAt, UpdatedAt: design.UpdatedAt,
	}
}
