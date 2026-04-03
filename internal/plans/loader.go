package plans

import (
	"encoding/json"
	"fmt"
	"os"
)

const envServicePlans = "SERVICE_PLANS_JSON"

// LoadRegistryFromEnv creates a plan registry by parsing the SERVICE_PLANS_JSON
// environment variable. If the variable is empty or unset an empty registry is
// returned (no-plan mode). A parse error returns a descriptive error instead
// of silently falling back, so the operator logs a clear message at startup.
func LoadRegistryFromEnv() (*Registry, error) {
	data := os.Getenv(envServicePlans)
	if data == "" {
		return NewRegistry(), nil
	}

	var planMap map[string]ServicePlan
	if err := json.Unmarshal([]byte(data), &planMap); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", envServicePlans, err)
	}

	return NewRegistryFromMap(planMap), nil
}
