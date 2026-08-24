package timeseries

import (
	"encoding/json"
	"testing"
	"time"
)

func TestChecksumPreservesInputOrder(t *testing.T) {
	start := time.Date(2026, 8, 24, 0, 0, 0, 0, time.UTC)
	points := []Point{
		{Timestamp: start, TemperatureC: 30},
		{Timestamp: start.Add(time.Hour), TemperatureC: 10},
		{Timestamp: start.Add(2 * time.Hour), TemperatureC: 20},
	}
	encoded, _, err := CanonicalJSONAndChecksum(points)
	if err != nil {
		t.Fatalf("CanonicalJSONAndChecksum() error = %v", err)
	}
	var decoded []Point
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(decoded) != 3 || decoded[0].TemperatureC != 30 || decoded[1].TemperatureC != 10 || decoded[2].TemperatureC != 20 {
		t.Fatalf("stored series order was mutated: %+v", decoded)
	}
	if points[0].TemperatureC != 30 || points[1].TemperatureC != 10 {
		t.Fatalf("input slice was mutated in place: %+v", points)
	}
}
