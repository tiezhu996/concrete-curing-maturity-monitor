package timeseries

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type Point struct {
	Timestamp    time.Time `json:"timestamp"`
	TemperatureC float64   `json:"temperature_c"`
}

type Analysis struct {
	StartedAt        time.Time `json:"started_at"`
	EndedAt          time.Time `json:"ended_at"`
	DurationHours    float64   `json:"duration_hours"`
	ExpectedPoints   int       `json:"expected_points"`
	ObservedPoints   int       `json:"observed_points"`
	MissingIntervals int       `json:"missing_intervals"`
	MissingRatio     float64   `json:"missing_ratio"`
	MinimumC         float64   `json:"minimum_c"`
	MaximumC         float64   `json:"maximum_c"`
	AverageC         float64   `json:"average_c"`
	QualityNote      string    `json:"quality_note"`
}

func Analyze(points []Point, sampleIntervalMinutes int) (Analysis, error) {
	if sampleIntervalMinutes < 1 || sampleIntervalMinutes > 1440 {
		return Analysis{}, fmt.Errorf("sample interval must be between 1 and 1440 minutes")
	}
	if len(points) < 2 {
		return Analysis{}, fmt.Errorf("at least two temperature points are required")
	}
	minimum := points[0].TemperatureC
	maximum := points[0].TemperatureC
	total := 0.0
	missing := 0
	interval := time.Duration(sampleIntervalMinutes) * time.Minute
	for index, point := range points {
		if point.TemperatureC < -50 || point.TemperatureC > 120 {
			return Analysis{}, fmt.Errorf("temperature point %d is outside the accepted -50C to 120C range", index)
		}
		if index > 0 {
			previous := points[index-1]
			if !point.Timestamp.After(previous.Timestamp) {
				return Analysis{}, fmt.Errorf("timestamps must be strictly increasing at point %d", index)
			}
			gap := point.Timestamp.Sub(previous.Timestamp)
			if gap > interval+interval/2 {
				expectedSegments := int(math.Round(float64(gap) / float64(interval)))
				if expectedSegments > 1 {
					missing += expectedSegments - 1
				}
			}
		}
		minimum = math.Min(minimum, point.TemperatureC)
		maximum = math.Max(maximum, point.TemperatureC)
		total += point.TemperatureC
	}
	expected := len(points) + missing
	ratio := float64(missing) / float64(expected)
	duration := points[len(points)-1].Timestamp.Sub(points[0].Timestamp).Hours()
	note := "complete chronological series"
	if missing > 0 {
		note = fmt.Sprintf("%d inferred missing sampling intervals", missing)
	}
	return Analysis{
		StartedAt: points[0].Timestamp, EndedAt: points[len(points)-1].Timestamp,
		DurationHours: round(duration, 4), ExpectedPoints: expected, ObservedPoints: len(points),
		MissingIntervals: missing, MissingRatio: round(ratio, 6),
		MinimumC: round(minimum, 3), MaximumC: round(maximum, 3), AverageC: round(total/float64(len(points)), 3),
		QualityNote: note,
	}, nil
}

func CanonicalJSONAndChecksum(points []Point) (string, string, error) {
	encoded, err := json.Marshal(points)
	if err != nil {
		return "", "", fmt.Errorf("encode temperature series: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return string(encoded), hex.EncodeToString(digest[:]), nil
}

func round(value float64, places int) float64 {
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}
