// openclawinstance_extensions.go contains anynines-specific condition type
// constants that extend the upstream OpenClawInstance CRD.
//
// These constants are defined in a separate file to minimise merge conflicts
// when rebasing on upstream releases. The upstream types.go file only carries
// two small marker comments and the Plan/ActivePlan struct fields (which
// cannot be separated from the struct definition in Go).
package v1alpha1

// --- anynines extension condition types ---
const (
	// ConditionTypeConfigReady indicates both the gateway ConfigMap and the
	// workspace seed ConfigMap have been successfully reconciled.
	ConditionTypeConfigReady = "ConfigReady"

	// ConditionTypePlanResolved indicates the service plan referenced by
	// spec.plan was found in the registry and its defaults have been applied.
	ConditionTypePlanResolved = "PlanResolved"

	// ConditionTypeHealthVerified indicates the OpenClaw gateway responded
	// successfully to a post-create/update health check.
	ConditionTypeHealthVerified = "HealthVerified"
)
