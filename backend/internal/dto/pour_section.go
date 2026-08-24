package dto

import (
	"concrete-curing-maturity-monitor/backend/internal/model"
	"strings"
	"time"
)

type CreatePourSectionRequest struct {
	SectionCode       string     `json:"section_code" binding:"required,min=2,max=48"`
	Name              string     `json:"name" binding:"required,min=2,max=180"`
	StructurePart     string     `json:"structure_part" binding:"required,min=2,max=180"`
	VolumeM3          float64    `json:"volume_m3" binding:"required,gt=0,lte=100000"`
	MixDesignID       uint       `json:"mix_design_id" binding:"required"`
	PouredAt          *time.Time `json:"poured_at"`
	TargetStrengthMPA float64    `json:"target_strength_mpa" binding:"required,gt=0,lte=200"`
	OwnerTeam         string     `json:"owner_team" binding:"required,min=2,max=140"`
}

func (request *CreatePourSectionRequest) Normalize() {
	request.SectionCode = strings.ToUpper(strings.TrimSpace(request.SectionCode))
	request.Name = strings.TrimSpace(request.Name)
	request.StructurePart = strings.TrimSpace(request.StructurePart)
	request.OwnerTeam = strings.TrimSpace(request.OwnerTeam)
}

type UpdatePourSectionRequest struct {
	Name              *string  `json:"name" binding:"omitempty,min=2,max=180"`
	StructurePart     *string  `json:"structure_part" binding:"omitempty,min=2,max=180"`
	VolumeM3          *float64 `json:"volume_m3" binding:"omitempty,gt=0,lte=100000"`
	MixDesignID       *uint    `json:"mix_design_id" binding:"omitempty,gt=0"`
	TargetStrengthMPA *float64 `json:"target_strength_mpa" binding:"omitempty,gt=0,lte=200"`
	OwnerTeam         *string  `json:"owner_team" binding:"omitempty,min=2,max=140"`
	Version           int      `json:"version" binding:"required,min=1"`
}

func (request *UpdatePourSectionRequest) Normalize() {
	request.Name = trimString(request.Name)
	request.StructurePart = trimString(request.StructurePart)
	request.OwnerTeam = trimString(request.OwnerTeam)
}

type TransitionPourSectionRequest struct {
	ToState string `json:"to_state" binding:"required,oneof=poured curing suspended threshold_reached closed"`
	Version int    `json:"version" binding:"required,min=1"`
	Note    string `json:"note" binding:"required,min=3,max=1000"`
}

type PourSectionQuery struct {
	Search      string
	CuringState string
	MixDesignID uint
	OwnerTeam   string
	Page        int
	PageSize    int
}

type PourSectionResponse struct {
	ID                uint                     `json:"id"`
	SectionCode       string                   `json:"section_code"`
	Name              string                   `json:"name"`
	StructurePart     string                   `json:"structure_part"`
	VolumeM3          float64                  `json:"volume_m3"`
	MixDesignID       uint                     `json:"mix_design_id"`
	MixCode           string                   `json:"mix_code"`
	MixVersion        int                      `json:"mix_version"`
	PouredAt          *time.Time               `json:"poured_at,omitempty"`
	TargetStrengthMPA float64                  `json:"target_strength_mpa"`
	CuringState       string                   `json:"curing_state"`
	OwnerTeam         string                   `json:"owner_team"`
	Version           int                      `json:"version"`
	Summary           model.PourSectionSummary `json:"summary"`
	CreatedAt         time.Time                `json:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at"`
}

type PourSectionListResponse struct {
	Items []PourSectionResponse `json:"items"`
	Total int64                 `json:"total"`
	Page  int                   `json:"page"`
	Size  int                   `json:"page_size"`
}

func NewPourSectionResponse(section model.PourSection, summary model.PourSectionSummary) PourSectionResponse {
	return PourSectionResponse{
		ID: section.ID, SectionCode: section.SectionCode, Name: section.Name,
		StructurePart: section.StructurePart, VolumeM3: section.VolumeM3,
		MixDesignID: section.MixDesignID, MixCode: section.MixDesign.MixCode,
		MixVersion: section.MixDesign.Version, PouredAt: section.PouredAt,
		TargetStrengthMPA: section.TargetStrengthMPA, CuringState: section.CuringState,
		OwnerTeam: section.OwnerTeam, Version: section.Version, Summary: summary,
		CreatedAt: section.CreatedAt, UpdatedAt: section.UpdatedAt,
	}
}

func trimString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
