package plans

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolve_EmptyName(t *testing.T) {
	r := NewRegistryFromMap(map[string]ServicePlan{
		"dev-small": {},
	})

	result, err := Resolve(r, "")
	require.NoError(t, err)
	assert.False(t, result.Found)
	assert.Empty(t, result.PlanName)
}

func TestResolve_PlanExists(t *testing.T) {
	r := NewRegistryFromMap(map[string]ServicePlan{
		"dev-small": {
			DisplayName: "Dev Small",
			Resources: PlanResources{
				Requests: PlanResourceList{CPU: "500m", Memory: "1Gi"},
			},
		},
	})

	result, err := Resolve(r, "dev-small")
	require.NoError(t, err)
	assert.True(t, result.Found)
	assert.Equal(t, "dev-small", result.PlanName)
	assert.Equal(t, "Dev Small", result.Plan.DisplayName)
	assert.Equal(t, "500m", result.Plan.Resources.Requests.CPU)
}

func TestResolve_PlanNotFound(t *testing.T) {
	r := NewRegistry()

	_, err := Resolve(r, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "plan resolution failed")
	assert.Contains(t, err.Error(), "nonexistent")
}
