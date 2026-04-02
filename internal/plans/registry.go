package plans

import (
	"fmt"
	"sync"
)

// Registry holds the set of service plans available to the operator.
// It is populated at operator startup from Helm values or a ConfigMap
// and is immutable during the operator's lifetime (until next restart).
type Registry struct {
	mu    sync.RWMutex
	plans map[string]ServicePlan
}

// NewRegistry creates an empty plan registry.
func NewRegistry() *Registry {
	return &Registry{
		plans: make(map[string]ServicePlan),
	}
}

// NewRegistryFromMap creates a registry pre-populated with the given plans.
// This is the primary constructor used at operator startup.
func NewRegistryFromMap(plans map[string]ServicePlan) *Registry {
	r := &Registry{
		plans: make(map[string]ServicePlan, len(plans)),
	}
	for k, v := range plans {
		r.plans[k] = v
	}
	return r
}

// Get returns the plan with the given name, or an error if not found.
func (r *Registry) Get(name string) (ServicePlan, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	plan, ok := r.plans[name]
	if !ok {
		return ServicePlan{}, fmt.Errorf("unknown service plan: %q", name)
	}
	return plan, nil
}

// Has returns true if the registry contains a plan with the given name.
func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.plans[name]
	return ok
}

// List returns all plan names in the registry.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.plans))
	for name := range r.plans {
		names = append(names, name)
	}
	return names
}

// Len returns the number of plans in the registry.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.plans)
}
