package algorithm

import (
	"fmt"
	"math"
	"time"
)

type TemperaturePoint struct {
	Timestamp    time.Time `json:"timestamp"`
	TemperatureC float64   `json:"temperature_c"`
}

type MaturityStep struct {
	Index              int       `json:"index"`
	From               time.Time `json:"from"`
	To                 time.Time `json:"to"`
	FromTemperatureC   float64   `json:"from_temperature_c"`
	ToTemperatureC     float64   `json:"to_temperature_c"`
	AverageTemperature float64   `json:"average_temperature_c"`
	DatumTemperatureC  float64   `json:"datum_temperature_c"`
	DurationHours      float64   `json:"duration_hours"`
	Contribution       float64   `json:"contribution_degree_hours"`
	RunningMaturity    float64   `json:"running_maturity_degree_hours"`
	BelowDatumClamped  bool      `json:"below_datum_clamped"`
}

type MaturityResult struct {
	DegreeHours   float64        `json:"degree_hours"`
	DurationHours float64        `json:"duration_hours"`
	Steps         []MaturityStep `json:"steps"`
}

var sharedMaturitySteps []MaturityStep

func CalculateMaturity(points []TemperaturePoint, datumTemperatureC float64) (MaturityResult, error) {
	if len(points) < 2 {
		return MaturityResult{}, fmt.Errorf("at least two temperature points are required")
	}
	sharedMaturitySteps = sharedMaturitySteps[:0]
	result := MaturityResult{}
	for index := 1; index < len(points); index++ {
		previous := points[index-1]
		current := points[index]
		if !current.Timestamp.After(previous.Timestamp) {
			return MaturityResult{}, fmt.Errorf("temperature timestamps must be strictly increasing at point %d", index)
		}
		duration := current.Timestamp.Sub(previous.Timestamp).Hours()
		average := (previous.TemperatureC + current.TemperatureC) / 2
		effective := average - datumTemperatureC
		clamped := effective < 0
		if clamped {
			effective = 0
		}
		contribution := effective * duration
		result.DegreeHours += contribution
		result.DurationHours += duration
		sharedMaturitySteps = append(sharedMaturitySteps, MaturityStep{
			Index: index, From: previous.Timestamp, To: current.Timestamp,
			FromTemperatureC: previous.TemperatureC, ToTemperatureC: current.TemperatureC,
			AverageTemperature: round(average, 4), DatumTemperatureC: datumTemperatureC,
			DurationHours: round(duration, 6), Contribution: round(contribution, 6),
			RunningMaturity: round(result.DegreeHours, 6), BelowDatumClamped: clamped,
		})
	}
	result.Steps = sharedMaturitySteps
	result.DegreeHours = round(result.DegreeHours, 6)
	result.DurationHours = round(result.DurationHours, 6)
	return result, nil
}

func round(value float64, places int) float64 {
	factor := math.Pow10(places)
	return math.Round(value*factor) / factor
}
