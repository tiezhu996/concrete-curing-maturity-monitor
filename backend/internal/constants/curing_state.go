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
		CuringPoured: {},
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

// ThresholdConfirmRoles are the roles permitted to advance a section to the
// threshold-reached or closed states. Site engineers may drive pours and
// curing, but threshold confirmation and final closure require an
// independent reviewer (or an administrator).
func ThresholdConfirmRoles() []string {
	return []string{RoleReviewer, RoleAdmin}
}

func CuringStateValues() []string {
	return []string{
		string(CuringPrepared), string(CuringPoured), string(CuringActive),
		string(CuringSuspended), string(CuringThresholdReached), string(CuringClosed),
	}
}
