package constants

type ConfidenceLevel string

const (
	ConfidenceLow    ConfidenceLevel = "low"
	ConfidenceMedium ConfidenceLevel = "medium"
	ConfidenceHigh   ConfidenceLevel = "high"
)

func (c ConfidenceLevel) Valid() bool {
	return c == ConfidenceLow || c == ConfidenceMedium || c == ConfidenceHigh
}

func ConfidenceLevelValues() []string {
	return []string{string(ConfidenceLow), string(ConfidenceMedium), string(ConfidenceHigh)}
}

func ConfidenceFor(missingRatio float64, extrapolated bool, durationHours float64) ConfidenceLevel {
	if extrapolated || missingRatio > 0.08 || durationHours < 2 {
		return ConfidenceLow
	}
	if missingRatio > 0.03 || durationHours < 8 {
		return ConfidenceMedium
	}
	return ConfidenceHigh
}
