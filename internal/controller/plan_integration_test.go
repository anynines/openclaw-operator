/*
Copyright 2026 OpenClaw.rocks

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
	"github.com/openclawrocks/openclaw-operator/internal/plans"
)

// newReconcilerForTest returns a minimal reconciler suitable for unit-testing
// methods that do not require a real cluster (applyPlanDefaults, mergePlanConfig).
func newReconcilerForTest() *OpenClawInstanceReconciler {
	return &OpenClawInstanceReconciler{}
}

// newInstanceForTest returns a bare OpenClawInstance with the given name.
func newInstanceForTest(name string) *openclawv1alpha1.OpenClawInstance {
	return &openclawv1alpha1.OpenClawInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
	}
}

// rawConfig encodes a map as a RawConfig for use in instance.Spec.Config.Raw.
func rawConfig(t *testing.T, m map[string]interface{}) *openclawv1alpha1.RawConfig {
	t.Helper()
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("rawConfig: failed to marshal: %v", err)
	}
	return &openclawv1alpha1.RawConfig{
		RawExtension: k8sruntime.RawExtension{Raw: b},
	}
}

// parseConfig decodes instance.Spec.Config.Raw into a map.
func parseConfig(t *testing.T, instance *openclawv1alpha1.OpenClawInstance) map[string]interface{} {
	t.Helper()
	if instance.Spec.Config.Raw == nil || len(instance.Spec.Config.Raw.Raw) == 0 {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(instance.Spec.Config.Raw.Raw, &m); err != nil {
		t.Fatalf("parseConfig: failed to unmarshal: %v", err)
	}
	return m
}

// ---------------------------------------------------------------------------
// applyPlanDefaults tests
// ---------------------------------------------------------------------------

func TestApplyPlanDefaults_FillsEmptyResources(t *testing.T) {
	r := newReconcilerForTest()
	instance := newInstanceForTest("fill-test")
	// Instance has no resources set.

	plan := &plans.ServicePlan{
		Resources: plans.PlanResources{
			Requests: plans.PlanResourceList{CPU: "500m", Memory: "1Gi"},
			Limits:   plans.PlanResourceList{CPU: "1", Memory: "2Gi"},
		},
		Storage: plans.PlanStorage{Size: "5Gi"},
	}

	if err := r.applyPlanDefaults(instance, plan); err != nil {
		t.Fatalf("applyPlanDefaults returned unexpected error: %v", err)
	}

	if instance.Spec.Resources.Requests.CPU != "500m" {
		t.Errorf("Requests.CPU: want 500m, got %q", instance.Spec.Resources.Requests.CPU)
	}
	if instance.Spec.Resources.Requests.Memory != "1Gi" {
		t.Errorf("Requests.Memory: want 1Gi, got %q", instance.Spec.Resources.Requests.Memory)
	}
	if instance.Spec.Resources.Limits.CPU != "1" {
		t.Errorf("Limits.CPU: want 1, got %q", instance.Spec.Resources.Limits.CPU)
	}
	if instance.Spec.Resources.Limits.Memory != "2Gi" {
		t.Errorf("Limits.Memory: want 2Gi, got %q", instance.Spec.Resources.Limits.Memory)
	}
	if instance.Spec.Storage.Persistence.Size != "5Gi" {
		t.Errorf("Storage.Size: want 5Gi, got %q", instance.Spec.Storage.Persistence.Size)
	}
}

func TestApplyPlanDefaults_InstanceOverridesTakePrecedence(t *testing.T) {
	r := newReconcilerForTest()
	instance := newInstanceForTest("override-test")
	// Instance has its own resource values pre-set.
	instance.Spec.Resources.Requests.CPU = "250m"
	instance.Spec.Resources.Requests.Memory = "512Mi"
	instance.Spec.Resources.Limits.CPU = "2"
	instance.Spec.Resources.Limits.Memory = "4Gi"
	instance.Spec.Storage.Persistence.Size = "20Gi"

	plan := &plans.ServicePlan{
		Resources: plans.PlanResources{
			Requests: plans.PlanResourceList{CPU: "500m", Memory: "1Gi"},
			Limits:   plans.PlanResourceList{CPU: "1", Memory: "2Gi"},
		},
		Storage: plans.PlanStorage{Size: "5Gi"},
	}

	if err := r.applyPlanDefaults(instance, plan); err != nil {
		t.Fatalf("applyPlanDefaults returned unexpected error: %v", err)
	}

	// All instance values must survive unchanged.
	if instance.Spec.Resources.Requests.CPU != "250m" {
		t.Errorf("Requests.CPU should not be overwritten; want 250m, got %q", instance.Spec.Resources.Requests.CPU)
	}
	if instance.Spec.Resources.Requests.Memory != "512Mi" {
		t.Errorf("Requests.Memory should not be overwritten; want 512Mi, got %q", instance.Spec.Resources.Requests.Memory)
	}
	if instance.Spec.Resources.Limits.CPU != "2" {
		t.Errorf("Limits.CPU should not be overwritten; want 2, got %q", instance.Spec.Resources.Limits.CPU)
	}
	if instance.Spec.Resources.Limits.Memory != "4Gi" {
		t.Errorf("Limits.Memory should not be overwritten; want 4Gi, got %q", instance.Spec.Resources.Limits.Memory)
	}
	if instance.Spec.Storage.Persistence.Size != "20Gi" {
		t.Errorf("Storage.Size should not be overwritten; want 20Gi, got %q", instance.Spec.Storage.Persistence.Size)
	}
}

func TestApplyPlanDefaults_PartialInstanceOverride(t *testing.T) {
	r := newReconcilerForTest()
	instance := newInstanceForTest("partial-test")
	// Instance sets only CPU limits; memory and storage should come from the plan.
	instance.Spec.Resources.Limits.CPU = "4"

	plan := &plans.ServicePlan{
		Resources: plans.PlanResources{
			Requests: plans.PlanResourceList{CPU: "500m", Memory: "1Gi"},
			Limits:   plans.PlanResourceList{CPU: "1", Memory: "2Gi"},
		},
		Storage: plans.PlanStorage{Size: "5Gi"},
	}

	if err := r.applyPlanDefaults(instance, plan); err != nil {
		t.Fatalf("applyPlanDefaults returned unexpected error: %v", err)
	}

	// CPU limit: instance value wins.
	if instance.Spec.Resources.Limits.CPU != "4" {
		t.Errorf("Limits.CPU: instance value should win; want 4, got %q", instance.Spec.Resources.Limits.CPU)
	}
	// Memory and storage: plan fills the gaps.
	if instance.Spec.Resources.Requests.CPU != "500m" {
		t.Errorf("Requests.CPU: plan should fill; want 500m, got %q", instance.Spec.Resources.Requests.CPU)
	}
	if instance.Spec.Resources.Limits.Memory != "2Gi" {
		t.Errorf("Limits.Memory: plan should fill; want 2Gi, got %q", instance.Spec.Resources.Limits.Memory)
	}
	if instance.Spec.Storage.Persistence.Size != "5Gi" {
		t.Errorf("Storage.Size: plan should fill; want 5Gi, got %q", instance.Spec.Storage.Persistence.Size)
	}
}

func TestApplyPlanDefaults_EmptyPlanNoChange(t *testing.T) {
	r := newReconcilerForTest()
	instance := newInstanceForTest("empty-plan-test")
	instance.Spec.Resources.Requests.CPU = "500m"
	instance.Spec.Resources.Limits.Memory = "2Gi"
	instance.Spec.Storage.Persistence.Size = "10Gi"

	// Plan with no values set.
	plan := &plans.ServicePlan{}

	if err := r.applyPlanDefaults(instance, plan); err != nil {
		t.Fatalf("applyPlanDefaults returned unexpected error: %v", err)
	}

	if instance.Spec.Resources.Requests.CPU != "500m" {
		t.Errorf("Requests.CPU should be unchanged; want 500m, got %q", instance.Spec.Resources.Requests.CPU)
	}
	if instance.Spec.Resources.Limits.Memory != "2Gi" {
		t.Errorf("Limits.Memory should be unchanged; want 2Gi, got %q", instance.Spec.Resources.Limits.Memory)
	}
	if instance.Spec.Storage.Persistence.Size != "10Gi" {
		t.Errorf("Storage.Size should be unchanged; want 10Gi, got %q", instance.Spec.Storage.Persistence.Size)
	}
}

// ---------------------------------------------------------------------------
// mergePlanConfig tests
// ---------------------------------------------------------------------------

func TestMergePlanConfig_EmptyInstanceConfig(t *testing.T) {
	r := newReconcilerForTest()
	instance := newInstanceForTest("empty-config-test")
	// No instance config: plan config should be adopted wholesale.
	instance.Spec.Config.Raw = nil

	planConfig := map[string]interface{}{
		"agents": map[string]interface{}{
			"defaults": map[string]interface{}{
				"model": "anthropic/claude-sonnet-4.6",
			},
		},
	}

	if err := r.mergePlanConfig(instance, planConfig); err != nil {
		t.Fatalf("mergePlanConfig returned unexpected error: %v", err)
	}

	cfg := parseConfig(t, instance)
	if cfg == nil {
		t.Fatal("expected config to be set after merge, got nil")
	}

	model := nestedString(cfg, "agents", "defaults", "model")
	if model != "anthropic/claude-sonnet-4.6" {
		t.Errorf("agents.defaults.model: want anthropic/claude-sonnet-4.6, got %q", model)
	}
}

func TestMergePlanConfig_DeepMerge_InstanceWins(t *testing.T) {
	r := newReconcilerForTest()
	instance := newInstanceForTest("deep-merge-test")
	// Instance overrides the model; plan sets a default.
	instance.Spec.Config.Raw = rawConfig(t, map[string]interface{}{
		"agents": map[string]interface{}{
			"defaults": map[string]interface{}{
				"model": "openrouter/anthropic/claude-opus-4.6",
			},
		},
	})

	planConfig := map[string]interface{}{
		"agents": map[string]interface{}{
			"defaults": map[string]interface{}{
				"model": "anthropic/claude-sonnet-4.6",
			},
		},
	}

	if err := r.mergePlanConfig(instance, planConfig); err != nil {
		t.Fatalf("mergePlanConfig returned unexpected error: %v", err)
	}

	cfg := parseConfig(t, instance)
	model := nestedString(cfg, "agents", "defaults", "model")
	if model != "openrouter/anthropic/claude-opus-4.6" {
		t.Errorf("instance model should win over plan default; want openrouter/anthropic/claude-opus-4.6, got %q", model)
	}
}

func TestMergePlanConfig_DeepMerge_PlanFillsMissingKeys(t *testing.T) {
	r := newReconcilerForTest()
	instance := newInstanceForTest("plan-fills-test")
	// Instance config has unrelated keys; plan contributes agents.defaults.model.
	instance.Spec.Config.Raw = rawConfig(t, map[string]interface{}{
		"gateway": map[string]interface{}{
			"mode": "local",
		},
	})

	planConfig := map[string]interface{}{
		"agents": map[string]interface{}{
			"defaults": map[string]interface{}{
				"model": "anthropic/claude-sonnet-4.6",
			},
		},
	}

	if err := r.mergePlanConfig(instance, planConfig); err != nil {
		t.Fatalf("mergePlanConfig returned unexpected error: %v", err)
	}

	cfg := parseConfig(t, instance)

	// Plan key should be present.
	model := nestedString(cfg, "agents", "defaults", "model")
	if model != "anthropic/claude-sonnet-4.6" {
		t.Errorf("plan key should be present; want anthropic/claude-sonnet-4.6, got %q", model)
	}

	// Instance key must survive.
	mode := nestedString(cfg, "gateway", "mode")
	if mode != "local" {
		t.Errorf("instance key should survive merge; want local, got %q", mode)
	}
}

func TestMergePlanConfig_EmptyPlanConfig(t *testing.T) {
	r := newReconcilerForTest()
	instance := newInstanceForTest("empty-plan-config-test")
	instance.Spec.Config.Raw = rawConfig(t, map[string]interface{}{
		"gateway": map[string]interface{}{"mode": "local"},
	})

	// Empty plan config: instance config must be unchanged.
	if err := r.mergePlanConfig(instance, map[string]interface{}{}); err != nil {
		t.Fatalf("mergePlanConfig returned unexpected error: %v", err)
	}

	cfg := parseConfig(t, instance)
	mode := nestedString(cfg, "gateway", "mode")
	if mode != "local" {
		t.Errorf("instance config should be unchanged; want local, got %q", mode)
	}
}

func TestMergePlanConfig_InvalidInstanceJSON(t *testing.T) {
	r := newReconcilerForTest()
	instance := newInstanceForTest("invalid-json-test")
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: k8sruntime.RawExtension{Raw: []byte(`{not valid json`)},
	}

	planConfig := map[string]interface{}{"key": "value"}

	err := r.mergePlanConfig(instance, planConfig)
	if err == nil {
		t.Fatal("expected error for invalid instance JSON, got nil")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// nestedString traverses a nested map by keys and returns the string value.
// Returns "" if any key is missing or the final value is not a string.
func nestedString(m map[string]interface{}, keys ...string) string {
	current := interface{}(m)
	for _, k := range keys {
		sub, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current = sub[k]
	}
	s, _ := current.(string)
	return s
}
