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
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
)

// setCondition is a test helper that appends or replaces a condition on instance.
func setCondition(instance *openclawv1alpha1.OpenClawInstance, condType string, status metav1.ConditionStatus) {
	for i := range instance.Status.Conditions {
		if instance.Status.Conditions[i].Type == condType {
			instance.Status.Conditions[i].Status = status
			return
		}
	}
	instance.Status.Conditions = append(instance.Status.Conditions, metav1.Condition{
		Type:   condType,
		Status: status,
		Reason: "Test",
	})
}

// allRequiredTrue sets every required condition to True on the given instance.
func allRequiredTrue(instance *openclawv1alpha1.OpenClawInstance) {
	for _, ct := range requiredConditionTypes {
		setCondition(instance, ct, metav1.ConditionTrue)
	}
}

// newBareInstance returns an OpenClawInstance with no conditions set.
func newBareInstance() *openclawv1alpha1.OpenClawInstance {
	return &openclawv1alpha1.OpenClawInstance{}
}

// ---------------------------------------------------------------------------
// computeReadyCondition tests
// ---------------------------------------------------------------------------

func TestComputeReadyCondition_AllRequiredTrue_Ready(t *testing.T) {
	instance := newBareInstance()
	allRequiredTrue(instance)

	aggr := computeReadyCondition(instance)

	if aggr.Condition.Status != metav1.ConditionTrue {
		t.Errorf("expected Ready=True when all required conditions are True, got %s", aggr.Condition.Status)
	}
	if aggr.Condition.Reason != "ReconcileSucceeded" {
		t.Errorf("expected Reason=ReconcileSucceeded, got %q", aggr.Condition.Reason)
	}
	if aggr.Phase != openclawv1alpha1.PhaseRunning {
		t.Errorf("expected Phase=Running, got %q", aggr.Phase)
	}
}

func TestComputeReadyCondition_NoConditions_NotReady(t *testing.T) {
	instance := newBareInstance()
	// No conditions set at all — unknown state.

	aggr := computeReadyCondition(instance)

	if aggr.Condition.Status != metav1.ConditionFalse {
		t.Errorf("expected Ready=False with no conditions set, got %s", aggr.Condition.Status)
	}
	if aggr.Condition.Reason != "RequiredConditionsFailed" {
		t.Errorf("expected Reason=RequiredConditionsFailed, got %q", aggr.Condition.Reason)
	}
	if aggr.Phase != openclawv1alpha1.PhaseFailed {
		t.Errorf("expected Phase=Failed, got %q", aggr.Phase)
	}
	// Message must list all required conditions.
	for _, ct := range requiredConditionTypes {
		if !strings.Contains(aggr.Condition.Message, ct) {
			t.Errorf("expected message to mention %q, got: %s", ct, aggr.Condition.Message)
		}
	}
}

func TestComputeReadyCondition_OneRequiredFalse_NotReady(t *testing.T) {
	instance := newBareInstance()
	allRequiredTrue(instance)
	// Override StorageReady to False.
	setCondition(instance, openclawv1alpha1.ConditionTypeStorageReady, metav1.ConditionFalse)

	aggr := computeReadyCondition(instance)

	if aggr.Condition.Status != metav1.ConditionFalse {
		t.Errorf("expected Ready=False when StorageReady=False, got %s", aggr.Condition.Status)
	}
	if !strings.Contains(aggr.Condition.Message, openclawv1alpha1.ConditionTypeStorageReady) {
		t.Errorf("message should mention StorageReady, got: %s", aggr.Condition.Message)
	}
	if aggr.Phase != openclawv1alpha1.PhaseFailed {
		t.Errorf("expected Phase=Failed, got %q", aggr.Phase)
	}
}

func TestComputeReadyCondition_MultipleRequiredFalse_MessageListsAll(t *testing.T) {
	instance := newBareInstance()
	allRequiredTrue(instance)
	setCondition(instance, openclawv1alpha1.ConditionTypeRBACReady, metav1.ConditionFalse)
	setCondition(instance, openclawv1alpha1.ConditionTypeStatefulSetReady, metav1.ConditionFalse)

	aggr := computeReadyCondition(instance)

	if aggr.Condition.Status != metav1.ConditionFalse {
		t.Errorf("expected Ready=False, got %s", aggr.Condition.Status)
	}
	if !strings.Contains(aggr.Condition.Message, openclawv1alpha1.ConditionTypeRBACReady) {
		t.Errorf("message should mention RBACReady, got: %s", aggr.Condition.Message)
	}
	if !strings.Contains(aggr.Condition.Message, openclawv1alpha1.ConditionTypeStatefulSetReady) {
		t.Errorf("message should mention StatefulSetReady, got: %s", aggr.Condition.Message)
	}
}

func TestComputeReadyCondition_RequiredTrueSkillPacksFalse_Degraded(t *testing.T) {
	instance := newBareInstance()
	allRequiredTrue(instance)
	setCondition(instance, openclawv1alpha1.ConditionTypeSkillPacksReady, metav1.ConditionFalse)

	aggr := computeReadyCondition(instance)

	if aggr.Condition.Status != metav1.ConditionTrue {
		t.Errorf("expected Ready=True when only soft condition fails, got %s", aggr.Condition.Status)
	}
	if aggr.Condition.Reason != "ReconcileSucceededDegraded" {
		t.Errorf("expected Reason=ReconcileSucceededDegraded, got %q", aggr.Condition.Reason)
	}
	if aggr.Phase != openclawv1alpha1.PhaseDegraded {
		t.Errorf("expected Phase=Degraded, got %q", aggr.Phase)
	}
	if !strings.Contains(aggr.Condition.Message, openclawv1alpha1.ConditionTypeSkillPacksReady) {
		t.Errorf("degraded message should mention SkillPacksReady, got: %s", aggr.Condition.Message)
	}
}

func TestComputeReadyCondition_RequiredTruePlanResolvedFalse_Degraded(t *testing.T) {
	instance := newBareInstance()
	allRequiredTrue(instance)
	setCondition(instance, openclawv1alpha1.ConditionTypePlanResolved, metav1.ConditionFalse)

	aggr := computeReadyCondition(instance)

	if aggr.Condition.Status != metav1.ConditionTrue {
		t.Errorf("expected Ready=True when PlanResolved=False (soft), got %s", aggr.Condition.Status)
	}
	if aggr.Phase != openclawv1alpha1.PhaseDegraded {
		t.Errorf("expected Phase=Degraded, got %q", aggr.Phase)
	}
}

func TestComputeReadyCondition_SoftConditionAbsent_NotDegraded(t *testing.T) {
	instance := newBareInstance()
	allRequiredTrue(instance)
	// Soft conditions not set at all (nil) — should not degrade.

	aggr := computeReadyCondition(instance)

	if aggr.Phase != openclawv1alpha1.PhaseRunning {
		t.Errorf("absent soft conditions should not cause Degraded; expected Running, got %q", aggr.Phase)
	}
}

func TestComputeReadyCondition_RequiredMissing_NotReady(t *testing.T) {
	instance := newBareInstance()
	// Set all required True except ServiceReady (absent = not yet set).
	for _, ct := range requiredConditionTypes {
		if ct == openclawv1alpha1.ConditionTypeServiceReady {
			continue
		}
		setCondition(instance, ct, metav1.ConditionTrue)
	}

	aggr := computeReadyCondition(instance)

	if aggr.Condition.Status != metav1.ConditionFalse {
		t.Errorf("expected Ready=False when ServiceReady is absent, got %s", aggr.Condition.Status)
	}
	if !strings.Contains(aggr.Condition.Message, openclawv1alpha1.ConditionTypeServiceReady) {
		t.Errorf("message should mention missing ServiceReady, got: %s", aggr.Condition.Message)
	}
}

func TestComputeReadyCondition_SoftFalseDoesNotBlockRequired(t *testing.T) {
	instance := newBareInstance()
	// All required True, all soft False — should be Degraded but Ready=True.
	allRequiredTrue(instance)
	for _, ct := range softConditionTypes {
		setCondition(instance, ct, metav1.ConditionFalse)
	}

	aggr := computeReadyCondition(instance)

	if aggr.Condition.Status != metav1.ConditionTrue {
		t.Errorf("soft False conditions must not block Ready=True, got %s", aggr.Condition.Status)
	}
	if aggr.Phase != openclawv1alpha1.PhaseDegraded {
		t.Errorf("expected Phase=Degraded, got %q", aggr.Phase)
	}
}

func TestComputeReadyCondition_ConditionType(t *testing.T) {
	instance := newBareInstance()
	allRequiredTrue(instance)

	aggr := computeReadyCondition(instance)

	if aggr.Condition.Type != openclawv1alpha1.ConditionTypeReady {
		t.Errorf("condition type must be %q, got %q", openclawv1alpha1.ConditionTypeReady, aggr.Condition.Type)
	}
}
