package algorithm

import (
	"testing"
	"time"
)

func TestMaturityStepsNotSharedAcrossCalls(t *testing.T) {
	start := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	first, err := CalculateMaturity([]TemperaturePoint{
		{Timestamp: start, TemperatureC: 10},
		{Timestamp: start.Add(time.Hour), TemperatureC: 20},
	}, 0)
	if err != nil {
		t.Fatalf("first CalculateMaturity() error = %v", err)
	}
	second, err := CalculateMaturity([]TemperaturePoint{
		{Timestamp: start.Add(2 * time.Hour), TemperatureC: 20},
		{Timestamp: start.Add(3 * time.Hour), TemperatureC: 40},
	}, 0)
	if err != nil {
		t.Fatalf("second CalculateMaturity() error = %v", err)
	}
	if second.DegreeHours != 30 {
		t.Fatalf("unexpected second maturity: %v", second.DegreeHours)
	}
	if len(first.Steps) != 1 || first.Steps[0].Contribution != 15 {
		t.Fatalf("first result steps were overwritten by second call: %+v", first.Steps)
	}
}
