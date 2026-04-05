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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
)


func TestDetermineActions(t *testing.T) {
	tests := []struct {
		name     string
		sc       *openclawv1alpha1.OpenClawSelfConfig
		expected []openclawv1alpha1.SelfConfigAction
	}{
		{
			name: "no actions",
			sc:   &openclawv1alpha1.OpenClawSelfConfig{},
			expected: nil,
		},
		{
			name: "skills only - add",
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddSkills: []string{"skill1"},
				},
			},
			expected: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionSkills},
		},
		{
			name: "skills only - remove",
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveSkills: []string{"skill1"},
				},
			},
			expected: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionSkills},
		},
		{
			name: "config only",
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					ConfigPatch: &openclawv1alpha1.RawConfig{
						RawExtension: runtime.RawExtension{Raw: []byte(`{"key":"value"}`)},
					},
				},
			},
			expected: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionConfig},
		},
		{
			name: "workspace files only - add",
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddWorkspaceFiles: map[string]string{"file1": "content1"},
				},
			},
			expected: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionWorkspaceFiles},
		},
		{
			name: "workspace files only - remove",
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveWorkspaceFiles: []string{"file1"},
				},
			},
			expected: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionWorkspaceFiles},
		},
		{
			name: "env vars only - add",
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddEnvVars: []openclawv1alpha1.SelfConfigEnvVar{{Name: "TEST", Value: "value"}},
				},
			},
			expected: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionEnvVars},
		},
		{
			name: "env vars only - remove",
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveEnvVars: []string{"TEST"},
				},
			},
			expected: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionEnvVars},
		},
		{
			name: "multiple actions",
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddSkills:         []string{"skill1"},
					AddWorkspaceFiles: map[string]string{"file1": "content1"},
					AddEnvVars:        []openclawv1alpha1.SelfConfigEnvVar{{Name: "TEST", Value: "value"}},
				},
			},
			expected: []openclawv1alpha1.SelfConfigAction{
				openclawv1alpha1.SelfConfigActionSkills,
				openclawv1alpha1.SelfConfigActionWorkspaceFiles,
				openclawv1alpha1.SelfConfigActionEnvVars,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineActions(tt.sc)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d actions, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("expected action %v at index %d, got %v", expected, i, result[i])
				}
			}
		})
	}
}

func TestCheckAllowedActions(t *testing.T) {
	tests := []struct {
		name      string
		requested []openclawv1alpha1.SelfConfigAction
		allowed   []openclawv1alpha1.SelfConfigAction
		denied    []openclawv1alpha1.SelfConfigAction
	}{
		{
			name:      "all allowed",
			requested: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionSkills},
			allowed:   []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionSkills, openclawv1alpha1.SelfConfigActionConfig},
			denied:    nil,
		},
		{
			name:      "none allowed",
			requested: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionSkills},
			allowed:   []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionConfig},
			denied:    []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionSkills},
		},
		{
			name: "partially allowed",
			requested: []openclawv1alpha1.SelfConfigAction{
				openclawv1alpha1.SelfConfigActionSkills,
				openclawv1alpha1.SelfConfigActionConfig,
				openclawv1alpha1.SelfConfigActionEnvVars,
			},
			allowed: []openclawv1alpha1.SelfConfigAction{
				openclawv1alpha1.SelfConfigActionSkills,
				openclawv1alpha1.SelfConfigActionWorkspaceFiles,
			},
			denied: []openclawv1alpha1.SelfConfigAction{
				openclawv1alpha1.SelfConfigActionConfig,
				openclawv1alpha1.SelfConfigActionEnvVars,
			},
		},
		{
			name:      "empty requests",
			requested: []openclawv1alpha1.SelfConfigAction{},
			allowed:   []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionSkills},
			denied:    nil,
		},
		{
			name:      "empty allowed",
			requested: []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionSkills},
			allowed:   []openclawv1alpha1.SelfConfigAction{},
			denied:    []openclawv1alpha1.SelfConfigAction{openclawv1alpha1.SelfConfigActionSkills},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkAllowedActions(tt.requested, tt.allowed)
			if len(result) != len(tt.denied) {
				t.Errorf("expected %d denied actions, got %d", len(tt.denied), len(result))
				return
			}
			for i, expected := range tt.denied {
				if result[i] != expected {
					t.Errorf("expected denied action %v at index %d, got %v", expected, i, result[i])
				}
			}
		})
	}
}

func TestApplySkillChanges(t *testing.T) {
	tests := []struct {
		name     string
		initial  []string
		sc       *openclawv1alpha1.OpenClawSelfConfig
		expected []string
	}{
		{
			name:    "add to empty",
			initial: []string{},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddSkills: []string{"skill1", "skill2"},
				},
			},
			expected: []string{"skill1", "skill2"},
		},
		{
			name:    "add to existing",
			initial: []string{"skill1"},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddSkills: []string{"skill2", "skill3"},
				},
			},
			expected: []string{"skill1", "skill2", "skill3"},
		},
		{
			name:    "add duplicate (should deduplicate)",
			initial: []string{"skill1"},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddSkills: []string{"skill1", "skill2"},
				},
			},
			expected: []string{"skill1", "skill2"},
		},
		{
			name:    "remove from existing",
			initial: []string{"skill1", "skill2", "skill3"},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveSkills: []string{"skill2"},
				},
			},
			expected: []string{"skill1", "skill3"},
		},
		{
			name:    "remove non-existent (should be no-op)",
			initial: []string{"skill1"},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveSkills: []string{"skill2"},
				},
			},
			expected: []string{"skill1"},
		},
		{
			name:    "add and remove",
			initial: []string{"skill1", "skill2"},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveSkills: []string{"skill1"},
					AddSkills:    []string{"skill3"},
				},
			},
			expected: []string{"skill2", "skill3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &openclawv1alpha1.OpenClawInstance{
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Skills: make([]string, len(tt.initial)),
				},
			}
			copy(instance.Spec.Skills, tt.initial)

			applySkillChanges(instance, tt.sc)

			if len(instance.Spec.Skills) != len(tt.expected) {
				t.Errorf("expected %d skills, got %d", len(tt.expected), len(instance.Spec.Skills))
				return
			}
			for i, expected := range tt.expected {
				if instance.Spec.Skills[i] != expected {
					t.Errorf("expected skill %q at index %d, got %q", expected, i, instance.Spec.Skills[i])
				}
			}
		})
	}
}

func TestApplyConfigPatch(t *testing.T) {
	tests := []struct {
		name        string
		initial     map[string]interface{}
		patch       map[string]interface{}
		expectError bool
		errorMsg    string
		expected    map[string]interface{}
	}{
		{
			name:    "nil patch",
			initial: map[string]interface{}{"key": "value"},
			patch:   nil,
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:    "simple patch",
			initial: map[string]interface{}{"key1": "value1"},
			patch:   map[string]interface{}{"key2": "value2"},
			expected: map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		},
		{
			name:        "protected key",
			initial:     map[string]interface{}{"key": "value"},
			patch:       map[string]interface{}{"gateway": "blocked"},
			expectError: true,
			errorMsg:    "config key \"gateway\" is protected",
		},
		{
			name:    "nested merge",
			initial: map[string]interface{}{
				"agents": map[string]interface{}{
					"list": []interface{}{
						map[string]interface{}{"name": "agent1"},
					},
				},
			},
			patch: map[string]interface{}{
				"agents": map[string]interface{}{
					"timeout": "30s",
				},
			},
			expected: map[string]interface{}{
				"agents": map[string]interface{}{
					"list": []interface{}{
						map[string]interface{}{"name": "agent1"},
					},
					"timeout": "30s",
				},
			},
		},
		{
			name:    "empty base config",
			initial: nil,
			patch:   map[string]interface{}{"key": "value"},
			expected: map[string]interface{}{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &openclawv1alpha1.OpenClawInstance{
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Config: openclawv1alpha1.ConfigSpec{},
				},
			}

			// Set up initial config
			if tt.initial != nil {
				initialJSON, _ := json.Marshal(tt.initial)
				instance.Spec.Config.Raw = &openclawv1alpha1.RawConfig{
					RawExtension: runtime.RawExtension{Raw: initialJSON},
				}
			}

			// Create self-config with patch
			sc := &openclawv1alpha1.OpenClawSelfConfig{}
			if tt.patch != nil {
				patchJSON, _ := json.Marshal(tt.patch)
				sc.Spec.ConfigPatch = &openclawv1alpha1.RawConfig{
					RawExtension: runtime.RawExtension{Raw: patchJSON},
				}
			}

			err := applyConfigPatch(instance, sc)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check result
			if tt.expected != nil {
				var result map[string]interface{}
				if instance.Spec.Config.Raw != nil {
					json.Unmarshal(instance.Spec.Config.Raw.Raw, &result)
				}

				expectedJSON, _ := json.Marshal(tt.expected)
				resultJSON, _ := json.Marshal(result)

				if string(expectedJSON) != string(resultJSON) {
					t.Errorf("expected %s, got %s", string(expectedJSON), string(resultJSON))
				}
			}
		})
	}
}

func TestApplyWorkspaceFileChanges(t *testing.T) {
	tests := []struct {
		name     string
		initial  map[string]string
		sc       *openclawv1alpha1.OpenClawSelfConfig
		expected map[string]string
	}{
		{
			name:    "add to nil workspace",
			initial: nil,
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddWorkspaceFiles: map[string]string{"file1": "content1"},
				},
			},
			expected: map[string]string{"file1": "content1"},
		},
		{
			name:    "add to existing",
			initial: map[string]string{"file1": "content1"},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddWorkspaceFiles: map[string]string{"file2": "content2"},
				},
			},
			expected: map[string]string{
				"file1": "content1",
				"file2": "content2",
			},
		},
		{
			name:    "remove files",
			initial: map[string]string{"file1": "content1", "file2": "content2"},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveWorkspaceFiles: []string{"file1"},
				},
			},
			expected: map[string]string{"file2": "content2"},
		},
		{
			name:    "add and remove",
			initial: map[string]string{"file1": "content1"},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveWorkspaceFiles: []string{"file1"},
					AddWorkspaceFiles:    map[string]string{"file2": "content2"},
				},
			},
			expected: map[string]string{"file2": "content2"},
		},
		{
			name:    "overwrite existing file",
			initial: map[string]string{"file1": "old content"},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddWorkspaceFiles: map[string]string{"file1": "new content"},
				},
			},
			expected: map[string]string{"file1": "new content"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &openclawv1alpha1.OpenClawInstance{}
			if tt.initial != nil {
				instance.Spec.Workspace = &openclawv1alpha1.WorkspaceSpec{
					InitialFiles: make(map[string]string),
				}
				for k, v := range tt.initial {
					instance.Spec.Workspace.InitialFiles[k] = v
				}
			}

			applyWorkspaceFileChanges(instance, tt.sc)

			if len(instance.Spec.Workspace.InitialFiles) != len(tt.expected) {
				t.Errorf("expected %d files, got %d", len(tt.expected), len(instance.Spec.Workspace.InitialFiles))
				return
			}

			for k, expectedV := range tt.expected {
				if actualV, ok := instance.Spec.Workspace.InitialFiles[k]; !ok {
					t.Errorf("expected file %q not found", k)
				} else if actualV != expectedV {
					t.Errorf("expected file %q to have content %q, got %q", k, expectedV, actualV)
				}
			}
		})
	}
}

func TestApplyEnvVarChanges(t *testing.T) {
	tests := []struct {
		name        string
		initial     []corev1.EnvVar
		sc          *openclawv1alpha1.OpenClawSelfConfig
		expected    []corev1.EnvVar
		expectError bool
		errorMsg    string
	}{
		{
			name:    "add to empty",
			initial: []corev1.EnvVar{},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddEnvVars: []openclawv1alpha1.SelfConfigEnvVar{{Name: "TEST", Value: "value"}},
				},
			},
			expected: []corev1.EnvVar{{Name: "TEST", Value: "value"}},
		},
		{
			name:    "add to existing",
			initial: []corev1.EnvVar{{Name: "EXISTING", Value: "value1"}},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddEnvVars: []openclawv1alpha1.SelfConfigEnvVar{{Name: "TEST", Value: "value2"}},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "EXISTING", Value: "value1"},
				{Name: "TEST", Value: "value2"},
			},
		},
		{
			name:    "replace existing",
			initial: []corev1.EnvVar{{Name: "TEST", Value: "oldvalue"}},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddEnvVars: []openclawv1alpha1.SelfConfigEnvVar{{Name: "TEST", Value: "newvalue"}},
				},
			},
			expected: []corev1.EnvVar{{Name: "TEST", Value: "newvalue"}},
		},
		{
			name:    "remove env var",
			initial: []corev1.EnvVar{{Name: "TEST", Value: "value"}, {Name: "KEEP", Value: "value"}},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveEnvVars: []string{"TEST"},
				},
			},
			expected: []corev1.EnvVar{{Name: "KEEP", Value: "value"}},
		},
		{
			name:    "remove non-existent env var",
			initial: []corev1.EnvVar{{Name: "KEEP", Value: "value"}},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveEnvVars: []string{"NONEXISTENT"},
				},
			},
			expected: []corev1.EnvVar{{Name: "KEEP", Value: "value"}},
		},
		{
			name:        "add protected env var",
			initial:     []corev1.EnvVar{},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					AddEnvVars: []openclawv1alpha1.SelfConfigEnvVar{{Name: "HOME", Value: "/tmp"}},
				},
			},
			expectError: true,
			errorMsg:    "environment variable \"HOME\" is protected",
		},
		{
			name:    "remove protected env var",
			initial: []corev1.EnvVar{{Name: "HOME", Value: "/home/user"}},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveEnvVars: []string{"HOME"},
				},
			},
			expectError: true,
			errorMsg:    "environment variable \"HOME\" is protected",
		},
		{
			name:    "add and remove in same operation",
			initial: []corev1.EnvVar{{Name: "OLD", Value: "value"}, {Name: "KEEP", Value: "value"}},
			sc: &openclawv1alpha1.OpenClawSelfConfig{
				Spec: openclawv1alpha1.OpenClawSelfConfigSpec{
					RemoveEnvVars: []string{"OLD"},
					AddEnvVars:    []openclawv1alpha1.SelfConfigEnvVar{{Name: "NEW", Value: "value"}},
				},
			},
			expected: []corev1.EnvVar{
				{Name: "KEEP", Value: "value"},
				{Name: "NEW", Value: "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instance := &openclawv1alpha1.OpenClawInstance{
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Env: make([]corev1.EnvVar, len(tt.initial)),
				},
			}
			copy(instance.Spec.Env, tt.initial)

			err := applyEnvVarChanges(instance, tt.sc)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(instance.Spec.Env) != len(tt.expected) {
				t.Errorf("expected %d env vars, got %d", len(tt.expected), len(instance.Spec.Env))
				return
			}

			for i, expected := range tt.expected {
				if instance.Spec.Env[i].Name != expected.Name || instance.Spec.Env[i].Value != expected.Value {
					t.Errorf("expected env var %v at index %d, got %v", expected, i, instance.Spec.Env[i])
				}
			}
		})
	}
}

func TestDeepMerge(t *testing.T) {
	tests := []struct {
		name     string
		dst      map[string]interface{}
		src      map[string]interface{}
		expected map[string]interface{}
	}{
		{
			name: "simple merge",
			dst:  map[string]interface{}{"a": 1},
			src:  map[string]interface{}{"b": 2},
			expected: map[string]interface{}{
				"a": 1,
				"b": 2,
			},
		},
		{
			name: "override existing",
			dst:  map[string]interface{}{"a": 1},
			src:  map[string]interface{}{"a": 2},
			expected: map[string]interface{}{
				"a": 2,
			},
		},
		{
			name: "nested merge",
			dst: map[string]interface{}{
				"config": map[string]interface{}{
					"setting1": "value1",
					"setting2": "value2",
				},
			},
			src: map[string]interface{}{
				"config": map[string]interface{}{
					"setting2": "newvalue2",
					"setting3": "value3",
				},
			},
			expected: map[string]interface{}{
				"config": map[string]interface{}{
					"setting1": "value1",
					"setting2": "newvalue2",
					"setting3": "value3",
				},
			},
		},
		{
			name: "array replacement",
			dst:  map[string]interface{}{"arr": []interface{}{1, 2}},
			src:  map[string]interface{}{"arr": []interface{}{3, 4}},
			expected: map[string]interface{}{
				"arr": []interface{}{3, 4},
			},
		},
		{
			name: "mixed types",
			dst: map[string]interface{}{
				"str":  "hello",
				"num":  42,
				"bool": true,
				"obj":  map[string]interface{}{"nested": "value"},
			},
			src: map[string]interface{}{
				"str": "world",
				"obj": map[string]interface{}{"nested": "newvalue", "added": "field"},
			},
			expected: map[string]interface{}{
				"str":  "world",
				"num":  42,
				"bool": true,
				"obj":  map[string]interface{}{"nested": "newvalue", "added": "field"},
			},
		},
		{
			name:     "empty dst",
			dst:      map[string]interface{}{},
			src:      map[string]interface{}{"key": "value"},
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "empty src",
			dst:      map[string]interface{}{"key": "value"},
			src:      map[string]interface{}{},
			expected: map[string]interface{}{"key": "value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deepMerge(tt.dst, tt.src)
			
			// Convert to JSON for easy comparison
			expectedJSON, _ := json.Marshal(tt.expected)
			resultJSON, _ := json.Marshal(result)
			
			if string(expectedJSON) != string(resultJSON) {
				t.Errorf("expected %s, got %s", string(expectedJSON), string(resultJSON))
			}
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > len(substr) && s[:len(substr)] == substr) || 
		(len(s) > len(substr) && s[len(s)-len(substr):] == substr) || 
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
