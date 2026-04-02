package plans

import (
	"fmt"
)

// ResolveResult contains the resolved plan and whether a plan was specified.
type ResolveResult struct {
	// Found is true if a plan name was specified and resolved.
	Found bool

	// Plan is the resolved service plan. Zero value if Found is false.
	Plan ServicePlan

	// PlanName is the name that was resolved.
	PlanName string
}

// Resolve looks up a plan by name in the registry.
// If planName is empty, returns a ResolveResult with Found=false (no plan mode).
// If planName is set but not found in the registry, returns an error.
func Resolve(registry *Registry, planName string) (ResolveResult, error) {
	if planName == "" {
		return ResolveResult{Found: false}, nil
	}

	plan, err := registry.Get(planName)
	if err != nil {
		return ResolveResult{}, fmt.Errorf("plan resolution failed: %w", err)
	}

	return ResolveResult{
		Found:    true,
		Plan:     plan,
		PlanName: planName,
	}, nil
}
