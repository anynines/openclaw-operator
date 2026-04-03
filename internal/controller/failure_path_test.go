package controller

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
	"github.com/openclawrocks/openclaw-operator/internal/plans"
)

// ---------------------------------------------------------------------------
// 1. Plan resolution — plan not found
// ---------------------------------------------------------------------------

func TestPlanResolution_PlanNotFound(t *testing.T) {
	registry := plans.NewRegistryFromMap(map[string]plans.ServicePlan{
		"dev-small": {DisplayName: "Dev Small"},
	})
	_, err := plans.Resolve(registry, "nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown plan, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the plan name, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 2. Plan merge — invalid instance config JSON
// ---------------------------------------------------------------------------

func TestPlanResolution_MergeFailed_InvalidJSON(t *testing.T) {
	r := newReconcilerForTest()
	inst := newInstanceForTest("merge-fail")
	inst.Spec.Plan = "test"
	// Inject broken JSON into config.raw so mergePlanConfig fails.
	inst.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: k8sruntime.RawExtension{Raw: []byte(`{not valid json`)},
	}

	plan := &plans.ServicePlan{
		Config: map[string]interface{}{"agents": map[string]interface{}{"defaults": map[string]interface{}{"model": "test"}}},
	}
	err := r.applyPlanDefaults(inst, plan)
	if err == nil {
		t.Fatal("expected error from applyPlanDefaults with invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "merge plan config") || !strings.Contains(err.Error(), "parse") {
		t.Errorf("error should mention merge/parse failure, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 3. Condition messages contain the actual error
// ---------------------------------------------------------------------------

func TestConditionMessage_MergePlanConfig_ContainsError(t *testing.T) {
	// When mergePlanConfig fails, applyPlanDefaults wraps the error.
	// The controller sets PlanResolved=False with the wrapped message.
	// We test the wrapper text here; the controller condition-set is tested
	// via the envtest suite (TestReconcile_InvalidPlan).
	r := newReconcilerForTest()
	inst := newInstanceForTest("msg-test")
	inst.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: k8sruntime.RawExtension{Raw: []byte(`BROKEN`)},
	}
	plan := &plans.ServicePlan{
		Config: map[string]interface{}{"key": "value"},
	}
	err := r.applyPlanDefaults(inst, plan)
	if err == nil {
		t.Fatal("expected error")
	}
	// The message must contain the root cause (JSON parse error), not a generic string.
	if !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("error should contain JSON parse root cause, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 4. Ready aggregation — multiple failures listed in message
// ---------------------------------------------------------------------------

func TestReadyAggregation_FailureMessage_ListsAll(t *testing.T) {
	inst := newBareInstance()
	// Set all required True except two.
	for _, ct := range requiredConditionTypes {
		status := metav1.ConditionTrue
		if ct == openclawv1alpha1.ConditionTypeStorageReady || ct == openclawv1alpha1.ConditionTypeStatefulSetReady {
			status = metav1.ConditionFalse
		}
		setCondition(inst, ct, status)
	}

	aggr := computeReadyCondition(inst)
	if aggr.Condition.Status != metav1.ConditionFalse {
		t.Fatalf("expected Ready=False, got %s", aggr.Condition.Status)
	}
	if !strings.Contains(aggr.Condition.Message, openclawv1alpha1.ConditionTypeStorageReady) {
		t.Errorf("message should list StorageReady, got: %s", aggr.Condition.Message)
	}
	if !strings.Contains(aggr.Condition.Message, openclawv1alpha1.ConditionTypeStatefulSetReady) {
		t.Errorf("message should list StatefulSetReady, got: %s", aggr.Condition.Message)
	}
}

// ---------------------------------------------------------------------------
// 5. HealthVerified=False blocks Ready (it is a required condition)
// ---------------------------------------------------------------------------

func TestReadyAggregation_HealthVerifiedFalse_BlocksReady(t *testing.T) {
	inst := newBareInstance()
	// All required True, then set HealthVerified=False.
	allRequiredTrue(inst)
	setCondition(inst, openclawv1alpha1.ConditionTypeHealthVerified, metav1.ConditionFalse)

	aggr := computeReadyCondition(inst)
	if aggr.Condition.Status != metav1.ConditionFalse {
		t.Errorf("expected Ready=False when HealthVerified=False (required), got %s", aggr.Condition.Status)
	}
	if !strings.Contains(aggr.Condition.Message, openclawv1alpha1.ConditionTypeHealthVerified) {
		t.Errorf("message should mention HealthVerified, got: %s", aggr.Condition.Message)
	}
	if aggr.Phase != openclawv1alpha1.PhaseFailed {
		t.Errorf("expected Phase=Failed, got %q", aggr.Phase)
	}
}

func TestReadyAggregation_HealthVerifiedTrue_AllGreen(t *testing.T) {
	inst := newBareInstance()
	allRequiredTrue(inst) // includes HealthVerified=True
	aggr := computeReadyCondition(inst)
	if aggr.Condition.Status != metav1.ConditionTrue {
		t.Errorf("expected Ready=True when all required (incl HealthVerified) are True, got %s", aggr.Condition.Status)
	}
	if aggr.Phase != openclawv1alpha1.PhaseRunning {
		t.Errorf("expected Phase=Running, got %q", aggr.Phase)
	}
}
