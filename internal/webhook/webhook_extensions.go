// webhook_extensions.go contains anynines-specific webhook validation logic,
// extracted from the upstream webhook file to minimise merge conflicts.
package webhook

import (
	"fmt"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
	"github.com/openclawrocks/openclaw-operator/internal/plans"
)

// validatePlanExtension validates the service plan reference on the instance.
// Returns an error if spec.plan names an unknown plan; returns nil if no plan
// is set or if the registry is nil (backwards-compatible).
func validatePlanExtension(instance *openclawv1alpha1.OpenClawInstance, registry *plans.Registry) error {
	if instance.Spec.Plan == "" || registry == nil {
		return nil
	}
	if !registry.Has(instance.Spec.Plan) {
		return fmt.Errorf("unknown service plan %q; available plans: %v",
			instance.Spec.Plan, registry.List())
	}
	return nil
}
