package algorithm

import "testing"

func TestInterpolateStrengthAndClamp(t *testing.T) {
	points := []CalibrationPoint{{0, 0}, {100, 10}, {200, 20}}
	strength, evidence, err := InterpolateStrength(points, 50)
	if err != nil || strength != 5 || evidence.Fraction != 0.5 || evidence.Extrapolated {
		t.Fatalf("unexpected interpolation: strength=%v evidence=%+v err=%v", strength, evidence, err)
	}
	strength, evidence, err = InterpolateStrength(points, 250)
	if err != nil || strength != 20 || !evidence.Clamped || !evidence.Extrapolated {
		t.Fatalf("outside range must clamp without extrapolation: strength=%v evidence=%+v err=%v", strength, evidence, err)
	}
}

func TestValidateCalibrationRejectsNonMonotonicPoints(t *testing.T) {
	points := []CalibrationPoint{{0, 0}, {100, 10}, {90, 12}}
	if err := ValidateCalibration(points); err == nil {
		t.Fatal("expected non-monotonic maturity to be rejected")
	}
}
