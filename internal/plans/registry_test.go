package plans

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRegistry_Empty(t *testing.T) {
	r := NewRegistry()
	assert.Equal(t, 0, r.Len())
	assert.Empty(t, r.List())
}

func TestNewRegistryFromMap(t *testing.T) {
	plans := map[string]ServicePlan{
		"dev-small": {
			DisplayName: "Dev Small",
			Resources: PlanResources{
				Requests: PlanResourceList{CPU: "500m", Memory: "1Gi"},
				Limits:   PlanResourceList{CPU: "1", Memory: "2Gi"},
			},
		},
		"prod-standard": {
			DisplayName: "Production Standard",
		},
	}

	r := NewRegistryFromMap(plans)
	assert.Equal(t, 2, r.Len())
}

func TestRegistry_Get_Exists(t *testing.T) {
	r := NewRegistryFromMap(map[string]ServicePlan{
		"dev-small": {DisplayName: "Dev Small"},
	})

	plan, err := r.Get("dev-small")
	require.NoError(t, err)
	assert.Equal(t, "Dev Small", plan.DisplayName)
}

func TestRegistry_Get_NotFound(t *testing.T) {
	r := NewRegistry()

	_, err := r.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown service plan")
}

func TestRegistry_Has(t *testing.T) {
	r := NewRegistryFromMap(map[string]ServicePlan{
		"dev-small": {},
	})

	assert.True(t, r.Has("dev-small"))
	assert.False(t, r.Has("nonexistent"))
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistryFromMap(map[string]ServicePlan{
		"dev-small":     {},
		"prod-standard": {},
	})

	names := r.List()
	sort.Strings(names)
	assert.Equal(t, []string{"dev-small", "prod-standard"}, names)
}

func TestRegistry_IsolatedFromSource(t *testing.T) {
	// Ensure the registry makes a copy and mutations to the source don't affect it
	source := map[string]ServicePlan{
		"dev-small": {DisplayName: "Original"},
	}
	r := NewRegistryFromMap(source)

	// Mutate source
	source["dev-small"] = ServicePlan{DisplayName: "Mutated"}
	source["new-plan"] = ServicePlan{}

	// Registry should be unchanged
	plan, err := r.Get("dev-small")
	require.NoError(t, err)
	assert.Equal(t, "Original", plan.DisplayName)
	assert.False(t, r.Has("new-plan"))
}
