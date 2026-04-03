// reconcile_extensions.go contains anynines-specific reconcile logic extracted
// from the upstream controller so that openclawinstance_controller.go carries
// only one-line call-sites at clearly marked extension points.
package controller

import (
	"context"
	"encoding/json"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
	"github.com/openclawrocks/openclaw-operator/internal/plans"
)

// reconcilePlanResolution resolves the service plan from the registry,
// applies plan defaults via the Effective Spec pattern, and sets the
// PlanResolved condition. When no plan is specified, ActivePlan is cleared.
func (r *OpenClawInstanceReconciler) reconcilePlanResolution(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance) error {
	logger := log.FromContext(ctx)

	if instance.Spec.Plan == "" || r.PlanRegistry == nil {
		instance.Status.ActivePlan = ""
		return nil
	}

	result, err := plans.Resolve(r.PlanRegistry, instance.Spec.Plan)
	if err != nil {
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "PlanResolutionFailed",
			"Failed to resolve service plan %q: %v", instance.Spec.Plan, err)
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    openclawv1alpha1.ConditionTypePlanResolved,
			Status:  metav1.ConditionFalse,
			Reason:  "PlanResolutionFailed",
			Message: fmt.Sprintf("Failed to resolve service plan %q: %v", instance.Spec.Plan, err),
		})
		return fmt.Errorf("failed to resolve service plan: %w", err)
	}

	if !result.Found {
		return nil
	}

	instance.Status.ActivePlan = result.PlanName
	if err := r.applyPlanDefaults(instance, &result.Plan); err != nil {
		r.Recorder.Eventf(instance, corev1.EventTypeWarning, "PlanMergeFailed",
			"Failed to merge plan defaults: %v", err)
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    openclawv1alpha1.ConditionTypePlanResolved,
			Status:  metav1.ConditionFalse,
			Reason:  "PlanMergeFailed",
			Message: fmt.Sprintf("Failed to merge plan defaults: %v", err),
		})
		return fmt.Errorf("failed to merge plan defaults: %w", err)
	}

	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    openclawv1alpha1.ConditionTypePlanResolved,
		Status:  metav1.ConditionTrue,
		Reason:  "PlanApplied",
		Message: fmt.Sprintf("Service plan %q resolved and applied", result.PlanName),
	})
	logger.V(1).Info("Service plan resolved and applied", "plan", result.PlanName)
	return nil
}

// ---------------------------------------------------------------------------
// applyPlanDefaults + mergePlanConfig — Effective Spec pattern
// ---------------------------------------------------------------------------

// applyPlanDefaults merges plan defaults into the instance spec in-memory.
// Only empty/zero instance fields are filled from the plan. Existing instance
// values are never overwritten (they are treated as user overrides).
func (r *OpenClawInstanceReconciler) applyPlanDefaults(instance *openclawv1alpha1.OpenClawInstance, plan *plans.ServicePlan) error {
	if plan.Resources.Requests.CPU != "" && instance.Spec.Resources.Requests.CPU == "" {
		instance.Spec.Resources.Requests.CPU = plan.Resources.Requests.CPU
	}
	if plan.Resources.Requests.Memory != "" && instance.Spec.Resources.Requests.Memory == "" {
		instance.Spec.Resources.Requests.Memory = plan.Resources.Requests.Memory
	}
	if plan.Resources.Limits.CPU != "" && instance.Spec.Resources.Limits.CPU == "" {
		instance.Spec.Resources.Limits.CPU = plan.Resources.Limits.CPU
	}
	if plan.Resources.Limits.Memory != "" && instance.Spec.Resources.Limits.Memory == "" {
		instance.Spec.Resources.Limits.Memory = plan.Resources.Limits.Memory
	}
	if plan.Storage.Size != "" && instance.Spec.Storage.Persistence.Size == "" {
		instance.Spec.Storage.Persistence.Size = plan.Storage.Size
	}
	if len(plan.Config) > 0 {
		if err := r.mergePlanConfig(instance, plan.Config); err != nil {
			return fmt.Errorf("failed to merge plan config: %w", err)
		}
	}
	return nil
}

// mergePlanConfig deep-merges plan config values into instance.Spec.Config.Raw.
// Instance values always win over plan values.
func (r *OpenClawInstanceReconciler) mergePlanConfig(instance *openclawv1alpha1.OpenClawInstance, planConfig map[string]interface{}) error {
	var instanceConfig map[string]interface{}
	if instance.Spec.Config.Raw != nil && len(instance.Spec.Config.Raw.Raw) > 0 {
		if err := json.Unmarshal(instance.Spec.Config.Raw.Raw, &instanceConfig); err != nil {
			return fmt.Errorf("failed to parse instance config: %w", err)
		}
	}
	if instanceConfig == nil {
		instanceConfig = make(map[string]interface{})
	}
	merged := plans.Merge(plans.MergeInput{
		Plan:           &plans.ServicePlan{Config: planConfig},
		PlanName:       instance.Spec.Plan,
		InstanceConfig: instanceConfig,
	})
	mergedBytes, err := json.Marshal(merged.Config)
	if err != nil {
		return fmt.Errorf("failed to serialize merged config: %w", err)
	}
	instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
		RawExtension: runtime.RawExtension{Raw: mergedBytes},
	}
	return nil
}

// ---------------------------------------------------------------------------
// ConfigReady condition helper
// ---------------------------------------------------------------------------

// setConfigReadyCondition sets the ConfigReady condition after ConfigMap +
// Workspace ConfigMap reconciliation. Called from reconcileResources at the
// marked extension point.
func setConfigReadyCondition(instance *openclawv1alpha1.OpenClawInstance, err error, step string) {
	if err != nil {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    openclawv1alpha1.ConditionTypeConfigReady,
			Status:  metav1.ConditionFalse,
			Reason:  step,
			Message: fmt.Sprintf("Failed to reconcile config: %v", err),
		})
		return
	}
	meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
		Type:    openclawv1alpha1.ConditionTypeConfigReady,
		Status:  metav1.ConditionTrue,
		Reason:  "ConfigReady",
		Message: "Gateway ConfigMap and Workspace ConfigMap reconciled successfully",
	})
}
