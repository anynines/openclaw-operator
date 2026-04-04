package controller

import (
	"testing"
	"time"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
)

func Test_parseCheckInterval(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want time.Duration
	}{
		{"empty", "", 24 * time.Hour},
		{"invalid", "notaduration", 24 * time.Hour},
		{"below min", "10m", time.Hour},
		{"above max", "200h", 168 * time.Hour},
		{"within bounds", "2h", 2 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCheckInterval(tt.in)
			if got != tt.want {
				t.Errorf("parseCheckInterval(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func Test_parseHealthCheckTimeout(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want time.Duration
	}{
		{"empty", "", 10 * time.Minute},
		{"invalid", "bad", 10 * time.Minute},
		{"below min", "1m", 2 * time.Minute},
		{"above max", "40m", 30 * time.Minute},
		{"within bounds", "5m", 5 * time.Minute},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseHealthCheckTimeout(tt.in)
			if got != tt.want {
				t.Errorf("parseHealthCheckTimeout(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func Test_isAutoUpdateEnabled(t *testing.T) {
	trueVal := true
	falseVal := false
	tests := []struct {
		name     string
		instance openclawv1alpha1.OpenClawInstance
		want     bool
	}{
		{"nil enabled", openclawv1alpha1.OpenClawInstance{}, false},
		{"enabled false", openclawv1alpha1.OpenClawInstance{Spec: openclawv1alpha1.OpenClawInstanceSpec{AutoUpdate: openclawv1alpha1.AutoUpdateSpec{Enabled: &falseVal}}}, false},
		{"enabled true, no digest", openclawv1alpha1.OpenClawInstance{Spec: openclawv1alpha1.OpenClawInstanceSpec{AutoUpdate: openclawv1alpha1.AutoUpdateSpec{Enabled: &trueVal}}}, true},
		{"enabled true, with digest", openclawv1alpha1.OpenClawInstance{Spec: openclawv1alpha1.OpenClawInstanceSpec{
			AutoUpdate: openclawv1alpha1.AutoUpdateSpec{Enabled: &trueVal},
			Image:      openclawv1alpha1.ImageSpec{Digest: "sha256:abc"},
		}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAutoUpdateEnabled(&tt.instance)
			if got != tt.want {
				t.Errorf("isAutoUpdateEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shouldCheckForUpdate(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name     string
		instance openclawv1alpha1.OpenClawInstance
		want     bool
	}{
		{"no last check", openclawv1alpha1.OpenClawInstance{}, true},
		{"interval elapsed", openclawv1alpha1.OpenClawInstance{
			Status: openclawv1alpha1.OpenClawInstanceStatus{
				AutoUpdate: openclawv1alpha1.AutoUpdateStatus{
					LastCheckTime: &openclawv1alpha1.Metav1Time{Time: now.Add(-25 * time.Hour)},
				},
			},
			Spec: openclawv1alpha1.OpenClawInstanceSpec{
				AutoUpdate: openclawv1alpha1.AutoUpdateSpec{CheckInterval: "24h"},
			},
		}, true},
		{"interval not elapsed", openclawv1alpha1.OpenClawInstance{
			Status: openclawv1alpha1.OpenClawInstanceStatus{
				AutoUpdate: openclawv1alpha1.AutoUpdateStatus{
					LastCheckTime: &openclawv1alpha1.Metav1Time{Time: now.Add(-1 * time.Hour)},
				},
			},
			Spec: openclawv1alpha1.OpenClawInstanceSpec{
				AutoUpdate: openclawv1alpha1.AutoUpdateSpec{CheckInterval: "24h"},
			},
		}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldCheckForUpdate(&tt.instance)
			if got != tt.want {
				t.Errorf("shouldCheckForUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}
