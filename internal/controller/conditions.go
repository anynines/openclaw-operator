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
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
)

// requiredConditionTypes lists the condition types that must all be True for
// the instance to be considered Ready. A single False here blocks Ready.
var requiredConditionTypes = []string{
	openclawv1alpha1.ConditionTypeRBACReady,
	openclawv1alpha1.ConditionTypeNetworkPolicyReady,
	openclawv1alpha1.ConditionTypeConfigReady,
	openclawv1alpha1.ConditionTypeStorageReady,
	openclawv1alpha1.ConditionTypeStatefulSetReady,
	openclawv1alpha1.ConditionTypeServiceReady,
}

// softConditionTypes lists condition types that are informational but do not
// block the Ready condition. Failures here result in Phase=Degraded.
var softConditionTypes = []string{
	openclawv1alpha1.ConditionTypeSkillPacksReady,
	openclawv1alpha1.ConditionTypePlanResolved,
	openclawv1alpha1.ConditionTypeConfigValid,
	openclawv1alpha1.ConditionTypeWorkspaceReady,
	openclawv1alpha1.ConditionTypeSecretsReady,
}

// ReadyAggregation is the result of computeReadyCondition.
type ReadyAggregation struct {
	// Condition is the Ready metav1.Condition to set on the instance.
	Condition metav1.Condition
	// Phase is the recommended phase string for instance.Status.Phase.
	Phase string
}

// computeReadyCondition derives the Ready condition and recommended Phase by
// inspecting the individual reconcile-step conditions on the instance.
//
// Rules:
//   - Ready=True when all required conditions are True.
//   - Ready=False when any required condition is False or missing.
//   - Phase=Running when Ready=True and all soft conditions are ok (or absent).
//   - Phase=Degraded when Ready=True but at least one soft condition is False.
//   - Phase=Error when Ready=False.
func computeReadyCondition(instance *openclawv1alpha1.OpenClawInstance) ReadyAggregation {
	// Collect failing required conditions.
	var failing []string
	for _, condType := range requiredConditionTypes {
		c := findCondition(instance.Status.Conditions, condType)
		if c == nil || c.Status != metav1.ConditionTrue {
			failing = append(failing, condType)
		}
	}

	if len(failing) > 0 {
		msg := fmt.Sprintf("Required conditions not met: %s", strings.Join(failing, ", "))
		return ReadyAggregation{
			Condition: metav1.Condition{
				Type:    openclawv1alpha1.ConditionTypeReady,
				Status:  metav1.ConditionFalse,
				Reason:  "RequiredConditionsFailed",
				Message: msg,
			},
			Phase: openclawv1alpha1.PhaseFailed,
		}
	}

	// All required conditions True — check soft conditions for degraded state.
	var degraded []string
	for _, condType := range softConditionTypes {
		c := findCondition(instance.Status.Conditions, condType)
		if c != nil && c.Status == metav1.ConditionFalse {
			degraded = append(degraded, condType)
		}
	}

	if len(degraded) > 0 {
		return ReadyAggregation{
			Condition: metav1.Condition{
				Type:    openclawv1alpha1.ConditionTypeReady,
				Status:  metav1.ConditionTrue,
				Reason:  "ReconcileSucceededDegraded",
				Message: fmt.Sprintf("Resources reconciled but some optional conditions are not met: %s", strings.Join(degraded, ", ")),
			},
			Phase: openclawv1alpha1.PhaseDegraded,
		}
	}

	return ReadyAggregation{
		Condition: metav1.Condition{
			Type:    openclawv1alpha1.ConditionTypeReady,
			Status:  metav1.ConditionTrue,
			Reason:  "ReconcileSucceeded",
			Message: "All resources reconciled successfully",
		},
		Phase: openclawv1alpha1.PhaseRunning,
	}
}

// findCondition returns the condition with the given type, or nil if absent.
func findCondition(conditions []metav1.Condition, condType string) *metav1.Condition {
	for i := range conditions {
		if conditions[i].Type == condType {
			return &conditions[i]
		}
	}
	return nil
}
