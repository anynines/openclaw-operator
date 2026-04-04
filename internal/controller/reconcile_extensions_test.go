package controller

import (
	"encoding/json"
	"testing"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
	"github.com/openclawrocks/openclaw-operator/internal/plans"
)

type fakeReconciler struct {
	OpenClawInstanceReconciler
}

func Test_applyPlanDefaults_mergesOnlyEmptyFields(t *testing.T) {
	instance := &openclawv1alpha1.OpenClawInstance{
		Spec: openclawv1alpha1.OpenClawInstanceSpec{
			Resources: openclawv1alpha1.ResourcesSpec{
				Requests: openclawv1alpha1.ResourceList{CPU: "", Memory: ""},
				Limits:   openclawv1alpha1.ResourceList{CPU: "2", Memory: ""},
			},
			Storage: openclawv1alpha1.StorageSpec{
				Persistence: openclawv1alpha1.PersistenceSpec{Size: ""},
			},
		},
	}
	plan := &plans.ServicePlan{
		Resources: plans.PlanResources{
			Requests: plans.PlanResourceList{CPU: "1", Memory: "512Mi"},
			Limits:   plans.PlanResourceList{CPU: "4", Memory: "2Gi"},
		},
		Storage: plans.PlanStorage{Size: "10Gi"},
	}
	r := &fakeReconciler{}
	err := r.applyPlanDefaults(instance, plan)
	if err != nil {
		t.Fatalf("applyPlanDefaults returned error: %v", err)
	}
	if instance.Spec.Resources.Requests.CPU != "1" {
		t.Errorf("Requests.CPU not set from plan")
	}
	if instance.Spec.Resources.Limits.CPU != "2" {
		t.Errorf("Limits.CPU should not be overwritten by plan")
	}
	if instance.Spec.Storage.Persistence.Size != "10Gi" {
		t.Errorf("Persistence.Size not set from plan")
	}
}

func Test_mergePlanConfig_mergesDeeply(t *testing.T) {
	r := &fakeReconciler{}
	instance := &openclawv1alpha1.OpenClawInstance{
		Spec: openclawv1alpha1.OpenClawInstanceSpec{
			Config: openclawv1alpha1.ConfigSpec{
				Raw: &openclawv1alpha1.RawConfig{},
			},
			Plan: "foo",
		},
	}
	planConfig := map[string]interface{}{
		"a": 1,
		"b": map[string]interface{}{"x": 2, "y": 3},
	}
	instanceConfig := map[string]interface{}{
		"b": map[string]interface{}{"y": 99},
		"c": 42,
	}
	raw, _ := json.Marshal(instanceConfig)
	instance.Spec.Config.Raw.Raw = raw

	err := r.mergePlanConfig(instance, planConfig)
	if err != nil {
		t.Fatalf("mergePlanConfig returned error: %v", err)
	}
	var merged map[string]interface{}
	if err := json.Unmarshal(instance.Spec.Config.Raw.Raw, &merged); err != nil {
		t.Fatalf("unmarshal merged config: %v", err)
	}
	// Plan value present
	if merged["a"] != float64(1) {
		t.Errorf("plan value 'a' missing")
	}
	// Instance value overrides plan
	b := merged["b"].(map[string]interface{})
	if b["y"] != float64(99) {
		t.Errorf("instance value b.y should override plan")
	}
	// Plan value present in nested map
	if b["x"] != float64(2) {
		t.Errorf("plan value b.x missing")
	}
	// Instance-only value present
	if merged["c"] != float64(42) {
		t.Errorf("instance value 'c' missing")
	}
}

func Test_mergePlanConfig_handlesNilInstanceConfig(t *testing.T) {
	r := &fakeReconciler{}
	instance := &openclawv1alpha1.OpenClawInstance{
		Spec: openclawv1alpha1.OpenClawInstanceSpec{
			Config: openclawv1alpha1.ConfigSpec{
				Raw: &openclawv1alpha1.RawConfig{},
			},
			Plan: "foo",
		},
	}
	planConfig := map[string]interface{}{"foo": "bar"}
	instance.Spec.Config.Raw.Raw = nil // no config

	err := r.mergePlanConfig(instance, planConfig)
	if err != nil {
		t.Fatalf("mergePlanConfig returned error: %v", err)
	}
	var merged map[string]interface{}
	if err := json.Unmarshal(instance.Spec.Config.Raw.Raw, &merged); err != nil {
		t.Fatalf("unmarshal merged config: %v", err)
	}
	if merged["foo"] != "bar" {
		t.Errorf("plan value not merged when instance config is nil")
	}
}
