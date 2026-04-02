package plans

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMerge_NoPlan(t *testing.T) {
	input := MergeInput{
		Plan: nil,
		InstanceResources: &PlanResources{
			Requests: PlanResourceList{CPU: "1", Memory: "2Gi"},
		},
		InstanceStorageSize: "10Gi",
		InstanceConfig: map[string]interface{}{
			"agents": map[string]interface{}{
				"defaults": map[string]interface{}{
					"model": "anthropic/claude-opus-4.6",
				},
			},
		},
	}

	result := Merge(input)
	assert.Empty(t, result.PlanName)
	assert.Equal(t, "1", result.Resources.Requests.CPU)
	assert.Equal(t, "10Gi", result.StorageSize)
	assert.Equal(t, "anthropic/claude-opus-4.6", result.Config["agents"].(map[string]interface{})["defaults"].(map[string]interface{})["model"])
}

func TestMerge_PlanOnly(t *testing.T) {
	plan := &ServicePlan{
		Resources: PlanResources{
			Requests: PlanResourceList{CPU: "500m", Memory: "1Gi"},
			Limits:   PlanResourceList{CPU: "1", Memory: "2Gi"},
		},
		Storage: PlanStorage{Size: "5Gi"},
		Config: map[string]interface{}{
			"agents": map[string]interface{}{
				"defaults": map[string]interface{}{
					"model": "anthropic/claude-sonnet-4.6",
				},
			},
		},
	}

	input := MergeInput{
		Plan:     plan,
		PlanName: "dev-small",
	}

	result := Merge(input)
	assert.Equal(t, "dev-small", result.PlanName)
	assert.Equal(t, "500m", result.Resources.Requests.CPU)
	assert.Equal(t, "1Gi", result.Resources.Requests.Memory)
	assert.Equal(t, "5Gi", result.StorageSize)
	assert.Equal(t, "anthropic/claude-sonnet-4.6", result.Config["agents"].(map[string]interface{})["defaults"].(map[string]interface{})["model"])
}

func TestMerge_OverrideResources(t *testing.T) {
	plan := &ServicePlan{
		Resources: PlanResources{
			Requests: PlanResourceList{CPU: "500m", Memory: "1Gi"},
			Limits:   PlanResourceList{CPU: "1", Memory: "2Gi"},
		},
	}

	input := MergeInput{
		Plan:     plan,
		PlanName: "dev-small",
		InstanceResources: &PlanResources{
			Limits: PlanResourceList{Memory: "4Gi"},
		},
	}

	result := Merge(input)
	// Plan defaults preserved for non-overridden fields
	assert.Equal(t, "500m", result.Resources.Requests.CPU)
	assert.Equal(t, "1Gi", result.Resources.Requests.Memory)
	assert.Equal(t, "1", result.Resources.Limits.CPU)
	// Override wins
	assert.Equal(t, "4Gi", result.Resources.Limits.Memory)
}

func TestMerge_OverrideStorage(t *testing.T) {
	plan := &ServicePlan{
		Storage: PlanStorage{Size: "5Gi"},
	}

	input := MergeInput{
		Plan:                plan,
		PlanName:            "dev-small",
		InstanceStorageSize: "20Gi",
	}

	result := Merge(input)
	assert.Equal(t, "20Gi", result.StorageSize)
}

func TestMerge_OverrideConfig_DeepMerge(t *testing.T) {
	plan := &ServicePlan{
		Config: map[string]interface{}{
			"agents": map[string]interface{}{
				"defaults": map[string]interface{}{
					"model":          "anthropic/claude-sonnet-4.6",
					"timeoutSeconds": 300,
				},
			},
			"gateway": map[string]interface{}{
				"mode": "local",
			},
		},
	}

	input := MergeInput{
		Plan:     plan,
		PlanName: "dev-small",
		InstanceConfig: map[string]interface{}{
			"agents": map[string]interface{}{
				"defaults": map[string]interface{}{
					"model": "openrouter/anthropic/claude-opus-4.6",
				},
			},
		},
	}

	result := Merge(input)

	agents := result.Config["agents"].(map[string]interface{})
	defaults := agents["defaults"].(map[string]interface{})

	// Override wins
	assert.Equal(t, "openrouter/anthropic/claude-opus-4.6", defaults["model"])
	// Plan default preserved
	assert.Equal(t, 300, defaults["timeoutSeconds"])
	// Untouched plan config preserved
	gateway := result.Config["gateway"].(map[string]interface{})
	assert.Equal(t, "local", gateway["mode"])
}

func TestMerge_NoPlan_NilConfig(t *testing.T) {
	input := MergeInput{
		Plan:           nil,
		InstanceConfig: nil,
	}

	result := Merge(input)
	assert.Nil(t, result.Config)
}

func TestMerge_PlanConfig_NoInstanceOverride(t *testing.T) {
	plan := &ServicePlan{
		Config: map[string]interface{}{
			"key": "value",
		},
	}

	input := MergeInput{
		Plan:     plan,
		PlanName: "test",
	}

	result := Merge(input)
	assert.Equal(t, "value", result.Config["key"])

	// Verify deep copy — mutating result shouldn't affect plan
	result.Config["key"] = "mutated"
	assert.Equal(t, "value", plan.Config["key"])
}

func TestMerge_NilInstanceResources(t *testing.T) {
	plan := &ServicePlan{
		Resources: PlanResources{
			Requests: PlanResourceList{CPU: "500m"},
		},
	}

	input := MergeInput{
		Plan:              plan,
		PlanName:          "test",
		InstanceResources: nil,
	}

	result := Merge(input)
	assert.Equal(t, "500m", result.Resources.Requests.CPU)
}

func TestDeepMergeMaps_NilBase(t *testing.T) {
	override := map[string]interface{}{"key": "value"}
	result := deepMergeMaps(nil, override)
	assert.Equal(t, "value", result["key"])

	// Verify it's a copy
	override["key"] = "mutated"
	assert.Equal(t, "value", result["key"])
}

func TestDeepCopyMap_Nil(t *testing.T) {
	result := deepCopyMap(nil)
	assert.Nil(t, result)
}

func TestDeepCopyMap_Nested(t *testing.T) {
	original := map[string]interface{}{
		"a": map[string]interface{}{
			"b": "value",
		},
	}

	copied := deepCopyMap(original)
	// Mutate original
	original["a"].(map[string]interface{})["b"] = "mutated"

	// Copy should be unchanged
	assert.Equal(t, "value", copied["a"].(map[string]interface{})["b"])
}
