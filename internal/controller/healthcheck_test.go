package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
)

func instanceWithSTS(name string, ready metav1.ConditionStatus) *openclawv1alpha1.OpenClawInstance {
	inst := &openclawv1alpha1.OpenClawInstance{}
	inst.Name = name
	inst.Namespace = "default"
	inst.Status.Conditions = []metav1.Condition{{
		Type:   openclawv1alpha1.ConditionTypeStatefulSetReady,
		Status: ready,
		Reason: "Test",
	}}
	return inst
}

func healthCond(inst *openclawv1alpha1.OpenClawInstance) *metav1.Condition {
	return findCondition(inst.Status.Conditions, openclawv1alpha1.ConditionTypeHealthVerified)
}

// redirectDoer sends every request to a test server, preserving the path.
type redirectDoer struct {
	base  string
	inner *http.Client
}

func (d *redirectDoer) Get(url string) (*http.Response, error) {
	path := healthCheckPath
	if i := strings.Index(url, healthCheckPath); i >= 0 {
		path = url[i:]
	}
	return d.inner.Get(d.base + path)
}

// failDoer panics if called — used to assert the check was skipped.
type failDoer struct{ t *testing.T }

func (d *failDoer) Get(_ string) (*http.Response, error) {
	d.t.Fatal("HTTP GET called but health check should have been skipped")
	return nil, nil
}

func TestHealthCheck_200_SetsTrue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	r := newReconcilerForTest()
	inst := instanceWithSTS("hc200", metav1.ConditionTrue)
	if err := r.doHealthCheck(context.Background(), inst, &redirectDoer{srv.URL, srv.Client()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := healthCond(inst)
	if c == nil {
		t.Fatal("HealthVerified condition not set")
	}
	if c.Status != metav1.ConditionTrue {
		t.Errorf("want True, got %s: %s", c.Status, c.Message)
	}
}

func TestHealthCheck_500_SetsFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()

	r := newReconcilerForTest()
	inst := instanceWithSTS("hc500", metav1.ConditionTrue)
	if err := r.doHealthCheck(context.Background(), inst, &redirectDoer{srv.URL, srv.Client()}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	c := healthCond(inst)
	if c == nil {
		t.Fatal("HealthVerified condition not set")
	}
	if c.Status != metav1.ConditionFalse {
		t.Errorf("want False, got %s", c.Status)
	}
	if !strings.Contains(c.Message, "500") {
		t.Errorf("message should mention 500, got: %s", c.Message)
	}
}

func TestHealthCheck_Unreachable_SetsFalse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	srv.Close() // kill it

	r := newReconcilerForTest()
	inst := instanceWithSTS("hcdown", metav1.ConditionTrue)
	if err := r.doHealthCheck(context.Background(), inst, &redirectDoer{srv.URL, &http.Client{}}); err != nil {
		t.Fatalf("health check must be non-fatal, got error: %v", err)
	}
	c := healthCond(inst)
	if c == nil {
		t.Fatal("HealthVerified condition not set")
	}
	if c.Status != metav1.ConditionFalse {
		t.Errorf("want False on unreachable, got %s", c.Status)
	}
}

func TestHealthCheck_STSNotReady_Skipped(t *testing.T) {
	r := newReconcilerForTest()
	inst := instanceWithSTS("hcskip", metav1.ConditionFalse)
	if err := r.doHealthCheck(context.Background(), inst, &failDoer{t}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if healthCond(inst) != nil {
		t.Error("HealthVerified should not be set when STS not ready")
	}
}

func TestHealthCheck_NoSTSCondition_Skipped(t *testing.T) {
	r := newReconcilerForTest()
	inst := &openclawv1alpha1.OpenClawInstance{}
	inst.Name = "hcnocon"
	inst.Namespace = "default"
	if err := r.doHealthCheck(context.Background(), inst, &failDoer{t}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if healthCond(inst) != nil {
		t.Error("HealthVerified should not be set when no STS condition exists")
	}
}
