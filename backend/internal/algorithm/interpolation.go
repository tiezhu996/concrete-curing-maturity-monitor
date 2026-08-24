package algorithm

import (
	"fmt"
	"sort"
)

type CalibrationPoint struct {
	MaturityDegreeHours float64 `json:"maturity_degree_hours"`
	StrengthMPA         float64 `json:"strength_mpa"`
}

type InterpolationEvidence struct {
	LowerPoint   CalibrationPoint `json:"lower_point"`
	UpperPoint   CalibrationPoint `json:"upper_point"`
	Fraction     float64          `json:"fraction"`
	Clamped      bool             `json:"clamped"`
	Extrapolated bool             `json:"outside_calibration_range"`
	Rule         string           `json:"rule"`
}

func ValidateCalibration(points []CalibrationPoint) error {
	if len(points) < 2 {
		return fmt.Errorf("at least two calibration points are required")
	}
	if points[0].MaturityDegreeHours < 0 || points[0].StrengthMPA < 0 {
		return fmt.Errorf("calibration values cannot be negative")
	}
	for index := range points {
		if points[index].MaturityDegreeHours < 0 || points[index].StrengthMPA < 0 {
			return fmt.Errorf("calibration point %d contains a negative value", index)
		}
		if index == 0 {
			continue
		}
		if points[index].MaturityDegreeHours <= points[index-1].MaturityDegreeHours {
			return fmt.Errorf("calibration maturity must be strictly increasing at point %d", index)
		}
		if points[index].StrengthMPA <= points[index-1].StrengthMPA {
			return fmt.Errorf("calibration strength must be strictly increasing at point %d", index)
		}
	}
	return nil
}

func InterpolateStrength(points []CalibrationPoint, maturity float64) (float64, InterpolationEvidence, error) {
	if err := ValidateCalibration(points); err != nil {
		return 0, InterpolationEvidence{}, err
	}
	if maturity <= points[0].MaturityDegreeHours {
		return points[0].StrengthMPA, InterpolationEvidence{
			LowerPoint: points[0], UpperPoint: points[1], Clamped: maturity < points[0].MaturityDegreeHours,
			Extrapolated: maturity < points[0].MaturityDegreeHours, Rule: "clamped to the first validated calibration point",
		}, nil
	}
	last := points[len(points)-1]
	if maturity >= last.MaturityDegreeHours {
		return last.StrengthMPA, InterpolationEvidence{
			LowerPoint: points[len(points)-2], UpperPoint: last, Fraction: 1,
			Clamped: maturity > last.MaturityDegreeHours, Extrapolated: maturity > last.MaturityDegreeHours,
			Rule: "clamped to the last validated calibration point; no ungrounded extrapolation",
		}, nil
	}
	upperIndex := sort.Search(len(points), func(index int) bool {
		return points[index].MaturityDegreeHours >= maturity
	})
	lower := points[upperIndex-1]
	upper := points[upperIndex]
	fraction := (maturity - lower.MaturityDegreeHours) / (upper.MaturityDegreeHours - lower.MaturityDegreeHours)
	strength := lower.StrengthMPA + fraction*(upper.StrengthMPA-lower.StrengthMPA)
	return round(strength, 4), InterpolationEvidence{
		LowerPoint: lower, UpperPoint: upper, Fraction: round(fraction, 6),
		Rule: "monotonic piecewise-linear interpolation",
	}, nil
}

func MaturityForStrength(points []CalibrationPoint, strength float64) (float64, bool, error) {
	if err := ValidateCalibration(points); err != nil {
		return 0, false, err
	}
	if strength <= points[0].StrengthMPA {
		return points[0].MaturityDegreeHours, false, nil
	}
	last := points[len(points)-1]
	if strength > last.StrengthMPA {
		return 0, true, nil
	}
	upperIndex := sort.Search(len(points), func(index int) bool {
		return points[index].StrengthMPA >= strength
	})
	lower := points[upperIndex-1]
	upper := points[upperIndex]
	fraction := (strength - lower.StrengthMPA) / (upper.StrengthMPA - lower.StrengthMPA)
	maturity := lower.MaturityDegreeHours + fraction*(upper.MaturityDegreeHours-lower.MaturityDegreeHours)
	return round(maturity, 4), false, nil
}
