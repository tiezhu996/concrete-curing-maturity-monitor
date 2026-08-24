package dto

import (
	"testing"

	"concrete-curing-maturity-monitor/backend/internal/model"
)

func TestSeriesResponsesIndependentPoints(t *testing.T) {
	first := model.TemperatureSeries{
		ID: 1, PointsJSON: `[{"timestamp":"2026-08-01T00:00:00Z","temperature_c":10},{"timestamp":"2026-08-01T01:00:00Z","temperature_c":20}]`,
	}
	second := model.TemperatureSeries{
		ID: 2, PointsJSON: `[{"timestamp":"2026-08-02T00:00:00Z","temperature_c":30},{"timestamp":"2026-08-02T01:00:00Z","temperature_c":40}]`,
	}
	firstResp := NewTemperatureSeriesResponse(first)
	_ = NewTemperatureSeriesResponse(second)
	if len(firstResp.Points) != 2 || firstResp.Points[0].TemperatureC != 10 || firstResp.Points[1].TemperatureC != 20 {
		t.Fatalf("first series points were overwritten by later decode: %+v", firstResp.Points)
	}
}

func TestMixResponsesIndependentCalibration(t *testing.T) {
	first := model.MixDesign{
		ID: 1, CalibrationPointsJSON: `[{"maturity_degree_hours":0,"strength_mpa":0},{"maturity_degree_hours":100,"strength_mpa":10}]`,
	}
	second := model.MixDesign{
		ID: 2, CalibrationPointsJSON: `[{"maturity_degree_hours":0,"strength_mpa":0},{"maturity_degree_hours":200,"strength_mpa":25}]`,
	}
	firstResp := NewMixDesignResponse(first, 0)
	_ = NewMixDesignResponse(second, 0)
	if len(firstResp.CalibrationPoints) != 2 || firstResp.CalibrationPoints[1].StrengthMPA != 10 {
		t.Fatalf("first mix calibration was overwritten by later decode: %+v", firstResp.CalibrationPoints)
	}
}
