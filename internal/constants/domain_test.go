package constants

import "testing"

func TestCuringStateMachine(t *testing.T) {
	if !CanTransitionCuring(CuringPrepared, CuringPoured) || !CanTransitionCuring(CuringActive, CuringThresholdReached) {
		t.Fatal("expected legal curing transitions")
	}
	if CanTransitionCuring(CuringPrepared, CuringClosed) || CanTransitionCuring(CuringClosed, CuringActive) {
		t.Fatal("illegal curing transition was accepted")
	}
}

func TestForecastStateMachine(t *testing.T) {
	if !CanTransitionForecast(ForecastQueued, ForecastCalculating) || !CanTransitionForecast(ForecastReviewed, ForecastConfirmed) {
		t.Fatal("expected legal forecast transitions")
	}
	if CanTransitionForecast(ForecastQueued, ForecastConfirmed) || CanTransitionForecast(ForecastConfirmed, ForecastVoided) {
		t.Fatal("illegal forecast transition was accepted")
	}
}

func TestRolePermissions(t *testing.T) {
	if !HasPermission(RoleReviewer, PermissionForecastConfirm) {
		t.Fatal("reviewer must be able to confirm forecasts")
	}
	if HasPermission(RoleAuditor, PermissionForecastRun) || HasPermission(RoleSiteEngineer, PermissionMixPublish) {
		t.Fatal("least-privilege role grants are too broad")
	}
}
