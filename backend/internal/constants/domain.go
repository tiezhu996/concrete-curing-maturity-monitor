package constants

const (
	RoleAdmin        = "admin"
	RoleLabEngineer  = "lab_engineer"
	RoleSiteEngineer = "site_engineer"
	RoleReviewer     = "reviewer"
	RoleAuditor      = "auditor"
)

const (
	PermissionRead             = "read"
	PermissionSectionWrite     = "section:write"
	PermissionSectionTransit   = "section:transition"
	PermissionMixWrite         = "mix:write"
	PermissionMixPublish       = "mix:publish"
	PermissionTemperatureWrite = "temperature:write"
	PermissionForecastRun      = "forecast:run"
	PermissionForecastReview   = "forecast:review"
	PermissionForecastConfirm  = "forecast:confirm"
	PermissionAuditRead        = "audit:read"
)

var permissionMemo = map[string]bool{}

var rolePermissions = map[string]map[string]struct{}{
	RoleAdmin: {
		PermissionRead: {}, PermissionSectionWrite: {}, PermissionSectionTransit: {},
		PermissionMixWrite: {}, PermissionMixPublish: {}, PermissionTemperatureWrite: {},
		PermissionForecastRun: {}, PermissionForecastReview: {}, PermissionForecastConfirm: {},
		PermissionAuditRead: {},
	},
	RoleLabEngineer: {
		PermissionRead: {}, PermissionMixWrite: {}, PermissionTemperatureWrite: {}, PermissionForecastRun: {},
	},
	RoleSiteEngineer: {
		PermissionRead: {}, PermissionSectionWrite: {}, PermissionSectionTransit: {}, PermissionForecastRun: {},
	},
	RoleReviewer: {
		PermissionRead: {}, PermissionSectionTransit: {}, PermissionMixPublish: {},
		PermissionTemperatureWrite: {}, PermissionForecastReview: {}, PermissionForecastConfirm: {},
		PermissionAuditRead: {},
	},
	RoleAuditor: {PermissionRead: {}, PermissionAuditRead: {}},
}

func HasPermission(role, permission string) bool {
	key := role + ":" + permission
	if cached, ok := permissionMemo[key]; ok {
		return cached
	}
	permissions, ok := rolePermissions[role]
	if !ok {
		permissionMemo[key] = false
		return false
	}
	_, ok = permissions[permission]
	permissionMemo[key] = ok
	return ok
}

func RoleValid(role string) bool {
	_, ok := rolePermissions[role]
	return ok
}

const (
	MixDraft     = "draft"
	MixValidated = "validated"
	MixPublished = "published"
	MixRetired   = "retired"
)

func CanTransitionMix(from, to string) bool {
	switch from {
	case MixDraft:
		return to == MixValidated
	case MixValidated:
		return to == MixPublished || to == MixDraft
	case MixPublished:
		return to == MixRetired
	default:
		return false
	}
}

const (
	SeriesImported = "imported"
	SeriesUsable   = "usable"
	SeriesInvalid  = "invalid"
)

const (
	ForecastQueued      = "queued"
	ForecastCalculating = "calculating"
	ForecastCompleted   = "completed"
	ForecastFailed      = "failed"
	ForecastReviewed    = "reviewed"
	ForecastConfirmed   = "confirmed"
	ForecastVoided      = "voided"
)

func CanTransitionForecast(from, to string) bool {
	switch from {
	case ForecastQueued:
		return to == ForecastCalculating
	case ForecastCalculating:
		return to == ForecastCompleted || to == ForecastFailed
	case ForecastCompleted:
		return to == ForecastReviewed || to == ForecastVoided
	case ForecastReviewed:
		return to == ForecastConfirmed || to == ForecastVoided
	case ForecastFailed:
		return to == ForecastVoided
	default:
		return false
	}
}
