// Package plans provides service plan resolution and merging for OpenClaw instances.
// A service plan is a named configuration preset that the operator ships as part of
// its own configuration. Plans are resolved at reconcile time and merged with
// instance-level overrides to produce an effective spec.
package plans

import (
	"k8s.io/apimachinery/pkg/api/resource"
)

// ServicePlan defines a named configuration preset for OpenClaw instances.
type ServicePlan struct {
	// DisplayName is the human-readable name of the plan.
	DisplayName string `json:"displayName,omitempty"`

	// Description explains the purpose and target audience of this plan.
	Description string `json:"description,omitempty"`

	// Resources defines CPU and memory requests and limits.
	Resources PlanResources `json:"resources,omitempty"`

	// Storage defines the persistent volume configuration.
	Storage PlanStorage `json:"storage,omitempty"`

	// Config contains OpenClaw gateway configuration defaults.
	// This is a free-form map that gets merged into the instance's config.raw.
	Config map[string]interface{} `json:"config,omitempty"`

	// Overridable lists the field paths that an instance may override.
	// If empty, all fields are overridable (permissive mode).
	// Field paths use dot notation: "config", "storage.size", "resources.limits.memory"
	Overridable []string `json:"overridable,omitempty"`
}

// PlanResources defines resource requests and limits for a plan.
type PlanResources struct {
	Requests PlanResourceList `json:"requests,omitempty"`
	Limits   PlanResourceList `json:"limits,omitempty"`
}

// PlanResourceList holds CPU and memory quantities as strings.
// Strings are used to allow Helm values.yaml compatibility.
type PlanResourceList struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// PlanStorage defines storage configuration for a plan.
type PlanStorage struct {
	Size string `json:"size,omitempty"`
}

// ParsedResources holds parsed resource.Quantity values ready for use in K8s specs.
type ParsedResources struct {
	RequestsCPU    resource.Quantity
	RequestsMemory resource.Quantity
	LimitsCPU      resource.Quantity
	LimitsMemory   resource.Quantity
}

// ParseResources converts string-based resource values to resource.Quantity.
// Returns zero values for empty strings.
func (p PlanResources) ParseResources() (ParsedResources, error) {
	var parsed ParsedResources
	var err error

	if p.Requests.CPU != "" {
		parsed.RequestsCPU, err = resource.ParseQuantity(p.Requests.CPU)
		if err != nil {
			return parsed, err
		}
	}
	if p.Requests.Memory != "" {
		parsed.RequestsMemory, err = resource.ParseQuantity(p.Requests.Memory)
		if err != nil {
			return parsed, err
		}
	}
	if p.Limits.CPU != "" {
		parsed.LimitsCPU, err = resource.ParseQuantity(p.Limits.CPU)
		if err != nil {
			return parsed, err
		}
	}
	if p.Limits.Memory != "" {
		parsed.LimitsMemory, err = resource.ParseQuantity(p.Limits.Memory)
		if err != nil {
			return parsed, err
		}
	}

	return parsed, nil
}
