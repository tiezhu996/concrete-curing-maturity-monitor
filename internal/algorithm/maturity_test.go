package algorithm

import (
	"testing"
	"time"
)

func TestCalculateMaturityFixedFixture(t *testing.T) {
	start := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	result, err := CalculateMaturity([]TemperaturePoint{
		{Timestamp: start, TemperatureC: 10},
		{Timestamp: start.Add(time.Hour), TemperatureC: 20},
	}, 0)
	if err != nil {
		t.Fatalf("CalculateMaturity() error = %v", err)
	}
	if result.DegreeHours != 15 || result.DurationHours != 1 {
		t.Fatalf("unexpected fixture result: %+v", result)
	}
	if len(result.Steps) != 1 || result.Steps[0].Contribution != 15 {
		t.Fatalf("step evidence was not preserved: %+v", result.Steps)
	}
}

func TestCalculateMaturityClampsBelowDatum(t *testing.T) {
	start := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	result, err := CalculateMaturity([]TemperaturePoint{
		{Timestamp: start, TemperatureC: -5},
		{Timestamp: start.Add(2 * time.Hour), TemperatureC: -3},
	}, 0)
	if err != nil {
		t.Fatalf("CalculateMaturity() error = %v", err)
	}
	if result.DegreeHours != 0 || !result.Steps[0].BelowDatumClamped {
		t.Fatalf("below-datum contribution was not clamped: %+v", result)
	}
}
