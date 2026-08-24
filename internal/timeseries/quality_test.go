package timeseries

import (
	"testing"
	"time"
)

func TestAnalyzeInfersMissingIntervals(t *testing.T) {
	start := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	analysis, err := Analyze([]Point{{start, 20}, {start.Add(2 * time.Hour), 22}}, 60)
	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}
	if analysis.MissingIntervals != 1 || analysis.ExpectedPoints != 3 || analysis.MissingRatio != 0.333333 {
		t.Fatalf("unexpected missing-data analysis: %+v", analysis)
	}
}

func TestAnalyzeRejectsUnorderedPoints(t *testing.T) {
	start := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	if _, err := Analyze([]Point{{start, 20}, {start, 22}}, 60); err == nil {
		t.Fatal("expected repeated timestamp to be rejected")
	}
}
