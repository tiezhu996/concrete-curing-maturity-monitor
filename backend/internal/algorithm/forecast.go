package algorithm

import (
	"fmt"
	"time"
)

const FormulaVersion = "nurse-saul-linear-v1.0.0"

type ForecastExplanation struct {
	FormulaVersion      string                `json:"formula_version"`
	Formula             string                `json:"formula"`
	TemperatureUnit     string                `json:"temperature_unit"`
	TimeUnit            string                `json:"time_unit"`
	BelowDatumRule      string                `json:"below_datum_rule"`
	Maturity            MaturityResult        `json:"maturity"`
	Interpolation       InterpolationEvidence `json:"interpolation"`
	CalibrationPoints   []CalibrationPoint    `json:"calibration_points"`
	TargetStrengthMPA   float64               `json:"target_strength_mpa"`
	TargetMaturity      *float64              `json:"target_maturity_degree_hours,omitempty"`
	RecentMaturityRate  float64               `json:"recent_maturity_rate_per_hour"`
	ThresholdETA        *time.Time            `json:"threshold_eta,omitempty"`
	OutsideCalibration  bool                  `json:"outside_calibration_range"`
	DataCoveragePercent float64               `json:"data_coverage_percent"`
	QualityWarnings     []string              `json:"quality_warnings"`
	BoundaryNote        string                `json:"boundary_note"`
}

type ForecastResult struct {
	MaturityDegreeHours  float64
	PredictedStrengthMPA float64
	ThresholdETA         *time.Time
	OutsideCalibration   bool
	DurationHours        float64
	Explanation          ForecastExplanation
}

func CalculateForecast(
	points []TemperaturePoint,
	datumTemperatureC float64,
	calibration []CalibrationPoint,
	targetStrengthMPA float64,
	missingRatio float64,
) (ForecastResult, error) {
	maturity, err := CalculateMaturity(points, datumTemperatureC)
	if err != nil {
		return ForecastResult{}, fmt.Errorf("calculate Nurse-Saul maturity: %w", err)
	}
	strength, interpolation, err := InterpolateStrength(calibration, maturity.DegreeHours)
	if err != nil {
		return ForecastResult{}, fmt.Errorf("interpolate calibrated strength: %w", err)
	}
	targetMaturity, targetOutside, err := MaturityForStrength(calibration, targetStrengthMPA)
	if err != nil {
		return ForecastResult{}, fmt.Errorf("locate target calibration threshold: %w", err)
	}
	rate := recentMaturityRate(maturity.Steps)
	var eta *time.Time
	if !targetOutside && targetMaturity > maturity.DegreeHours && rate > 0 {
		hours := (targetMaturity - maturity.DegreeHours) / rate
		value := points[len(points)-1].Timestamp.Add(time.Duration(hours * float64(time.Hour)))
		eta = &value
	}
	targetPointer := &targetMaturity
	if targetOutside {
		targetPointer = nil
	}
	warnings := make([]string, 0, 3)
	if missingRatio > 0 {
		warnings = append(warnings, fmt.Sprintf("temperature coverage contains %.2f%% missing intervals", missingRatio*100))
	}
	if interpolation.Extrapolated {
		warnings = append(warnings, "maturity is outside the validated calibration range; strength is clamped")
	}
	if targetOutside {
		warnings = append(warnings, "target strength exceeds the published calibration range; ETA is withheld")
	}
	explanation := ForecastExplanation{
		FormulaVersion:  FormulaVersion,
		Formula:         "M = sum((Tavg - T0) * delta_hours)",
		TemperatureUnit: "degree Celsius", TimeUnit: "hour",
		BelowDatumRule: "negative segment contributions are clamped to zero",
		Maturity:       maturity, Interpolation: interpolation, CalibrationPoints: calibration,
		TargetStrengthMPA: targetStrengthMPA, TargetMaturity: targetPointer,
		RecentMaturityRate: round(rate, 6), ThresholdETA: eta,
		OutsideCalibration:  interpolation.Extrapolated || targetOutside,
		DataCoveragePercent: round((1-missingRatio)*100, 2), QualityWarnings: warnings,
		BoundaryNote: "Offline decision support only. Verify with physical specimens and a licensed engineer before construction release.",
	}
	return ForecastResult{
		MaturityDegreeHours: maturity.DegreeHours, PredictedStrengthMPA: strength,
		ThresholdETA: eta, OutsideCalibration: explanation.OutsideCalibration,
		DurationHours: maturity.DurationHours, Explanation: explanation,
	}, nil
}

func recentMaturityRate(steps []MaturityStep) float64 {
	if len(steps) == 0 {
		return 0
	}
	start := len(steps) - 3
	if start < 0 {
		start = 0
	}
	var contribution float64
	var hours float64
	for _, step := range steps[start:] {
		contribution += step.Contribution
		hours += step.DurationHours
	}
	if hours <= 0 {
		return 0
	}
	return contribution / hours
}
