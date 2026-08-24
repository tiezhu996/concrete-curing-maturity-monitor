package algorithm

import (
	"testing"
	"time"
)

func TestCalculateForecastProducesGroundedETA(t *testing.T) {
	start := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	result, err := CalculateForecast(
		[]TemperaturePoint{{start, 20}, {start.Add(2 * time.Hour), 20}, {start.Add(4 * time.Hour), 20}},
		0,
		[]CalibrationPoint{{0, 0}, {100, 10}, {200, 20}},
		15,
		0,
	)
	if err != nil {
		t.Fatalf("CalculateForecast() error = %v", err)
	}
	if result.MaturityDegreeHours != 80 || result.PredictedStrengthMPA != 8 {
		t.Fatalf("unexpected forecast: %+v", result)
	}
	want := start.Add(7*time.Hour + 30*time.Minute)
	if result.ThresholdETA == nil || !result.ThresholdETA.Equal(want) {
		t.Fatalf("ThresholdETA = %v, want %v", result.ThresholdETA, want)
	}
	if result.Explanation.FormulaVersion != FormulaVersion || result.Explanation.OutsideCalibration {
		t.Fatalf("forecast evidence is incomplete: %+v", result.Explanation)
	}
}

func TestCalculateForecastWithholdsOutOfRangeTargetETA(t *testing.T) {
	start := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	result, err := CalculateForecast(
		[]TemperaturePoint{{start, 20}, {start.Add(time.Hour), 20}},
		0,
		[]CalibrationPoint{{0, 0}, {100, 10}},
		30,
		0,
	)
	if err != nil {
		t.Fatalf("CalculateForecast() error = %v", err)
	}
	if result.ThresholdETA != nil || !result.OutsideCalibration {
		t.Fatalf("out-of-range target must not receive ETA: %+v", result)
	}
}
