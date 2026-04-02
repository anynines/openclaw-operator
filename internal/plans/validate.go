package plans

import (
	"fmt"
	"strings"
)

// ValidationError represents a field override that was rejected.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("field %q: %s", e.Field, e.Message)
}

// ValidationResult holds the outcome of override validation.
type ValidationResult struct {
	// Errors contains all validation failures.
	Errors []ValidationError

	// Warnings contains non-fatal issues (e.g., override of a field that matches plan default).
	Warnings []string
}

// IsValid returns true if no validation errors were found.
func (r ValidationResult) IsValid() bool {
	return len(r.Errors) == 0
}

// Error returns a combined error message, or empty string if valid.
func (r ValidationResult) Error() string {
	if r.IsValid() {
		return ""
	}
	msgs := make([]string, len(r.Errors))
	for i, e := range r.Errors {
		msgs[i] = e.Error()
	}
	return fmt.Sprintf("override validation failed: %s", strings.Join(msgs, "; "))
}

// ValidateOverrides checks whether the given override fields are allowed by the plan.
// If the plan's Overridable list is empty, all fields are allowed (permissive mode).
// overrideFields is a list of dot-notation field paths that the instance wants to override.
func ValidateOverrides(plan ServicePlan, overrideFields []string) ValidationResult {
	result := ValidationResult{}

	// Permissive mode: if no overridable list is defined, everything is allowed.
	if len(plan.Overridable) == 0 {
		return result
	}

	allowed := make(map[string]bool, len(plan.Overridable))
	for _, f := range plan.Overridable {
		allowed[f] = true
	}

	for _, field := range overrideFields {
		if !isFieldAllowed(field, allowed) {
			result.Errors = append(result.Errors, ValidationError{
				Field:   field,
				Message: "field is not overridable in this plan",
			})
		}
	}

	return result
}

// isFieldAllowed checks if a field path is allowed by the overridable set.
// A field is allowed if it matches exactly or if any of its parent paths are allowed.
// For example, if "config" is in the allowed set, "config.agents.defaults.model" is also allowed.
func isFieldAllowed(field string, allowed map[string]bool) bool {
	// Direct match
	if allowed[field] {
		return true
	}

	// Check parent paths: "config.agents.defaults.model" is allowed if "config" is allowed
	parts := strings.Split(field, ".")
	for i := 1; i < len(parts); i++ {
		parent := strings.Join(parts[:i], ".")
		if allowed[parent] {
			return true
		}
	}

	return false
}
