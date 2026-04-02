package plans

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateOverrides_PermissiveMode(t *testing.T) {
	// Empty overridable list = everything allowed
	plan := ServicePlan{
		Overridable: nil,
	}

	result := ValidateOverrides(plan, []string{"config", "resources.limits.memory", "anything"})
	assert.True(t, result.IsValid())
	assert.Empty(t, result.Errors)
}

func TestValidateOverrides_EmptyOverrides(t *testing.T) {
	plan := ServicePlan{
		Overridable: []string{"config"},
	}

	result := ValidateOverrides(plan, nil)
	assert.True(t, result.IsValid())
}

func TestValidateOverrides_AllAllowed(t *testing.T) {
	plan := ServicePlan{
		Overridable: []string{"config", "storage.size"},
	}

	result := ValidateOverrides(plan, []string{"config", "storage.size"})
	assert.True(t, result.IsValid())
}

func TestValidateOverrides_FieldRejected(t *testing.T) {
	plan := ServicePlan{
		Overridable: []string{"config"},
	}

	result := ValidateOverrides(plan, []string{"config", "resources.limits.memory"})
	assert.False(t, result.IsValid())
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, "resources.limits.memory", result.Errors[0].Field)
	assert.Contains(t, result.Errors[0].Message, "not overridable")
}

func TestValidateOverrides_MultipleRejected(t *testing.T) {
	plan := ServicePlan{
		Overridable: []string{"config"},
	}

	result := ValidateOverrides(plan, []string{"resources.limits.cpu", "resources.limits.memory"})
	assert.False(t, result.IsValid())
	assert.Len(t, result.Errors, 2)
}

func TestValidateOverrides_ParentPathAllows(t *testing.T) {
	// "config" allows "config.agents.defaults.model"
	plan := ServicePlan{
		Overridable: []string{"config"},
	}

	result := ValidateOverrides(plan, []string{"config.agents.defaults.model"})
	assert.True(t, result.IsValid())
}

func TestValidateOverrides_ChildDoesNotAllowParent(t *testing.T) {
	// "config.agents" does NOT allow "config"
	plan := ServicePlan{
		Overridable: []string{"config.agents"},
	}

	result := ValidateOverrides(plan, []string{"config"})
	assert.False(t, result.IsValid())
}

func TestValidateOverrides_ExactMatch(t *testing.T) {
	plan := ServicePlan{
		Overridable: []string{"storage.size"},
	}

	result := ValidateOverrides(plan, []string{"storage.size"})
	assert.True(t, result.IsValid())
}

func TestValidateOverrides_NestedParentPath(t *testing.T) {
	// "resources.limits" allows "resources.limits.memory"
	plan := ServicePlan{
		Overridable: []string{"resources.limits"},
	}

	result := ValidateOverrides(plan, []string{"resources.limits.memory", "resources.limits.cpu"})
	assert.True(t, result.IsValid())

	// But not "resources.requests.memory"
	result2 := ValidateOverrides(plan, []string{"resources.requests.memory"})
	assert.False(t, result2.IsValid())
}

func TestValidationResult_Error(t *testing.T) {
	result := ValidationResult{
		Errors: []ValidationError{
			{Field: "a", Message: "not allowed"},
			{Field: "b", Message: "not allowed"},
		},
	}

	errMsg := result.Error()
	assert.Contains(t, errMsg, "override validation failed")
	assert.Contains(t, errMsg, "a")
	assert.Contains(t, errMsg, "b")
}

func TestValidationResult_Error_Valid(t *testing.T) {
	result := ValidationResult{}
	assert.Empty(t, result.Error())
}
