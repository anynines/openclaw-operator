package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
	"github.com/openclawrocks/openclaw-operator/internal/resources"
)

const (
	healthCheckTimeout     = 10 * time.Second
	healthCheckPath        = "/health"
	healthCheckGracePeriod = 60 * time.Second
	defaultGatewayPort     = 18789
)

// healthCheckDoer abstracts HTTP GET for testing.
type healthCheckDoer interface {
	Get(url string) (*http.Response, error)
}

// reconcileHealthCheck probes the gateway after all resources are reconciled.
// It is non-blocking: errors only set the HealthVerified condition, they never
// cause reconcileResources to fail.
func (r *OpenClawInstanceReconciler) reconcileHealthCheck(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance) error {
	return r.doHealthCheck(ctx, instance, nil)
}

// doHealthCheck is the testable core. Pass a non-nil doer to replace the real HTTP client.
func (r *OpenClawInstanceReconciler) doHealthCheck(ctx context.Context, instance *openclawv1alpha1.OpenClawInstance, doer healthCheckDoer) error {
	logger := log.FromContext(ctx)

	// Skip when StatefulSet is not ready — nothing to probe.
	stsCond := meta.FindStatusCondition(instance.Status.Conditions, openclawv1alpha1.ConditionTypeStatefulSetReady)
	if stsCond == nil || stsCond.Status != metav1.ConditionTrue {
		logger.V(2).Info("Skipping health check: StatefulSet not ready")
		return nil
	}

	// Grace period: skip health check if StatefulSet became ready less than
	// 60s ago — the gateway needs time to start up.
	if stsCond.LastTransitionTime.Time.After(time.Now().Add(-healthCheckGracePeriod)) {
		logger.V(2).Info("Skipping health check: within grace period",
			"readySince", stsCond.LastTransitionTime.Time,
			"graceUntil", stsCond.LastTransitionTime.Time.Add(healthCheckGracePeriod))
		return nil
	}

	// Determine gateway port: use resources.GatewayPort as default.
	gatewayPort := defaultGatewayPort
	if resources.GatewayPort > 0 {
		gatewayPort = int(resources.GatewayPort)
	}

	url := fmt.Sprintf("http://%s.%s.svc:%d%s",
		resources.ServiceName(instance), instance.Namespace,
		gatewayPort, healthCheckPath)

	if doer == nil {
		doer = &http.Client{Timeout: healthCheckTimeout}
	}

	resp, err := doer.Get(url)
	if err != nil {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    openclawv1alpha1.ConditionTypeHealthVerified,
			Status:  metav1.ConditionFalse,
			Reason:  "HealthCheckFailed",
			Message: fmt.Sprintf("Health check failed: %v", err),
		})
		logger.V(1).Info("Health check failed (non-fatal)", "url", url, "error", err)
		return nil
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusOK {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    openclawv1alpha1.ConditionTypeHealthVerified,
			Status:  metav1.ConditionTrue,
			Reason:  "HealthCheckPassed",
			Message: fmt.Sprintf("Gateway responded HTTP %d on %s", resp.StatusCode, healthCheckPath),
		})
	} else {
		meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
			Type:    openclawv1alpha1.ConditionTypeHealthVerified,
			Status:  metav1.ConditionFalse,
			Reason:  "HealthCheckFailed",
			Message: fmt.Sprintf("Gateway returned HTTP %d on %s", resp.StatusCode, healthCheckPath),
		})
	}
	return nil
}
