package plans

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseResources_AllSet(t *testing.T) {
	r := PlanResources{
		Requests: PlanResourceList{CPU: "500m", Memory: "1Gi"},
		Limits:   PlanResourceList{CPU: "2", Memory: "4Gi"},
	}

	parsed, err := r.ParseResources()
	require.NoError(t, err)

	assert.Equal(t, "500m", parsed.RequestsCPU.String())
	assert.Equal(t, "1Gi", parsed.RequestsMemory.String())
	assert.Equal(t, "2", parsed.LimitsCPU.String())
	assert.Equal(t, "4Gi", parsed.LimitsMemory.String())
}

func TestParseResources_Empty(t *testing.T) {
	r := PlanResources{}

	parsed, err := r.ParseResources()
	require.NoError(t, err)

	assert.True(t, parsed.RequestsCPU.IsZero())
	assert.True(t, parsed.RequestsMemory.IsZero())
	assert.True(t, parsed.LimitsCPU.IsZero())
	assert.True(t, parsed.LimitsMemory.IsZero())
}

func TestParseResources_Partial(t *testing.T) {
	r := PlanResources{
		Requests: PlanResourceList{CPU: "250m"},
		Limits:   PlanResourceList{Memory: "2Gi"},
	}

	parsed, err := r.ParseResources()
	require.NoError(t, err)

	assert.Equal(t, "250m", parsed.RequestsCPU.String())
	assert.True(t, parsed.RequestsMemory.IsZero())
	assert.True(t, parsed.LimitsCPU.IsZero())
	assert.Equal(t, "2Gi", parsed.LimitsMemory.String())
}

func TestParseResources_InvalidQuantity(t *testing.T) {
	r := PlanResources{
		Requests: PlanResourceList{CPU: "not-a-quantity"},
	}

	_, err := r.ParseResources()
	assert.Error(t, err)
}

func TestParseResources_InvalidMemory(t *testing.T) {
	r := PlanResources{
		Limits: PlanResourceList{Memory: "lots"},
	}

	_, err := r.ParseResources()
	assert.Error(t, err)
}
