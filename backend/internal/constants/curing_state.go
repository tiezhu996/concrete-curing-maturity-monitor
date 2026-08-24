package constants

type CuringState string

const (
	CuringPrepared         CuringState = "prepared"
	CuringPoured           CuringState = "poured"
	CuringActive           CuringState = "curing"
	CuringSuspended        CuringState = "suspended"
	CuringThresholdReached CuringState = "threshold_reached"
	CuringClosed           CuringState = "closed"
)

var curingTransitions = map[CuringState]map[CuringState]struct{}{
	CuringPrepared: {
		CuringPoured:    {},
		CuringSuspended: {},
	},
	CuringPoured: {
		CuringActive:    {},
		CuringSuspended: {},
	},
	CuringActive: {
		CuringThresholdReached: {},
		CuringSuspended:        {},
	},
	CuringThresholdReached: {
		CuringClosed:    {},
		CuringSuspended: {},
	},
	CuringSuspended: {
		CuringActive: {},
		CuringClosed: {},
	},
}

func (s CuringState) Valid() bool {
	_, ok := curingTransitions[s]
	return ok || s == CuringClosed
}

func CanTransitionCuring(from, to CuringState) bool {
	next, ok := curingTransitions[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}

func CuringStateValues() []string {
	return []string{
		string(CuringPrepared), string(CuringPoured), string(CuringActive),
		string(CuringSuspended), string(CuringThresholdReached), string(CuringClosed),
	}
}
