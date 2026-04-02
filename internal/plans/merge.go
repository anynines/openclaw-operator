package plans

// EffectiveSpec represents the merged result of plan defaults and instance overrides.
// This is what the resource builders (ConfigMap, StatefulSet, etc.) consume.
type EffectiveSpec struct {
	// Resources holds the final CPU/memory values.
	Resources PlanResources

	// StorageSize is the final PVC size.
	StorageSize string

	// Config is the merged configuration map.
	Config map[string]interface{}

	// PlanName is set if a plan was used. Empty for no-plan mode.
	PlanName string
}

// MergeInput contains the inputs for the merge operation.
type MergeInput struct {
	// Plan is the resolved plan. Nil if no plan was specified.
	Plan *ServicePlan

	// PlanName is the name of the resolved plan (empty if no plan).
	PlanName string

	// InstanceResources are the resource values from the instance spec.
	InstanceResources *PlanResources

	// InstanceStorageSize is the storage size from the instance spec.
	InstanceStorageSize string

	// InstanceConfig is the config.raw from the instance spec.
	InstanceConfig map[string]interface{}
}

// Merge combines plan defaults with instance overrides to produce an effective spec.
//
// Rules:
//   - No plan: instance values are used directly (full control mode).
//   - Plan with no overrides: plan defaults are used.
//   - Plan with overrides: instance values win for overridable fields.
//     Non-overridable fields always use plan values.
//
// Note: Override validation (which fields are allowed) is handled by ValidateOverrides
// in the webhook. Merge trusts that validation has already happened and applies
// instance overrides for any non-empty instance value.
func Merge(input MergeInput) EffectiveSpec {
	// No plan: use instance values directly
	if input.Plan == nil {
		return EffectiveSpec{
			Resources:   derefResources(input.InstanceResources),
			StorageSize: input.InstanceStorageSize,
			Config:      input.InstanceConfig,
		}
	}

	plan := input.Plan
	effective := EffectiveSpec{
		PlanName:    input.PlanName,
		Resources:   plan.Resources,
		StorageSize: plan.Storage.Size,
		Config:      deepCopyMap(plan.Config),
	}

	// Merge instance resource overrides
	if input.InstanceResources != nil {
		mergeResources(&effective.Resources, input.InstanceResources)
	}

	// Merge instance storage override
	if input.InstanceStorageSize != "" {
		effective.StorageSize = input.InstanceStorageSize
	}

	// Merge instance config override (deep merge)
	if input.InstanceConfig != nil {
		effective.Config = deepMergeMaps(effective.Config, input.InstanceConfig)
	}

	return effective
}

// mergeResources applies non-empty instance resource values over plan defaults.
func mergeResources(base *PlanResources, override *PlanResources) {
	if override.Requests.CPU != "" {
		base.Requests.CPU = override.Requests.CPU
	}
	if override.Requests.Memory != "" {
		base.Requests.Memory = override.Requests.Memory
	}
	if override.Limits.CPU != "" {
		base.Limits.CPU = override.Limits.CPU
	}
	if override.Limits.Memory != "" {
		base.Limits.Memory = override.Limits.Memory
	}
}

func derefResources(r *PlanResources) PlanResources {
	if r == nil {
		return PlanResources{}
	}
	return *r
}

// deepCopyMap creates a deep copy of a string-keyed map.
func deepCopyMap(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		if subMap, ok := v.(map[string]interface{}); ok {
			result[k] = deepCopyMap(subMap)
		} else {
			result[k] = v
		}
	}
	return result
}

// deepMergeMaps merges override into base. Override values win.
// Nested maps are merged recursively.
func deepMergeMaps(base, override map[string]interface{}) map[string]interface{} {
	if base == nil {
		return deepCopyMap(override)
	}
	result := deepCopyMap(base)
	for k, v := range override {
		if subOverride, ok := v.(map[string]interface{}); ok {
			if subBase, ok := result[k].(map[string]interface{}); ok {
				result[k] = deepMergeMaps(subBase, subOverride)
				continue
			}
		}
		result[k] = v
	}
	return result
}
