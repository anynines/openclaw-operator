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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
	"github.com/openclawrocks/openclaw-operator/internal/resources"
)

var _ = Describe("Service Plan Integration", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	Context("When creating an instance with a plan", func() {
		It("Should apply plan resource defaults to the StatefulSet", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "plan-defaults-test",
					Namespace: "default",
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Plan: "test-plan-small",
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secret",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			stsKey := types.NamespacedName{
				Name:      resources.StatefulSetName(instance),
				Namespace: "default",
			}

			By("Verifying StatefulSet container resources match plan defaults")
			Eventually(func(g Gomega) {
				sts := &appsv1.StatefulSet{}
				g.Expect(k8sClient.Get(ctx, stsKey, sts)).To(Succeed())
				container := sts.Spec.Template.Spec.Containers[0]

				g.Expect(container.Resources.Requests[corev1.ResourceCPU]).To(Equal(resource.MustParse("200m")))
				g.Expect(container.Resources.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("256Mi")))
				g.Expect(container.Resources.Limits[corev1.ResourceCPU]).To(Equal(resource.MustParse("500m")))
				g.Expect(container.Resources.Limits[corev1.ResourceMemory]).To(Equal(resource.MustParse("512Mi")))
			}, timeout, interval).Should(Succeed())

			By("Verifying status.activePlan is set")
			instanceKey := types.NamespacedName{Name: "plan-defaults-test", Namespace: "default"}
			Eventually(func(g Gomega) {
				inst := &openclawv1alpha1.OpenClawInstance{}
				g.Expect(k8sClient.Get(ctx, instanceKey, inst)).To(Succeed())
				g.Expect(inst.Status.ActivePlan).To(Equal("test-plan-small"))
			}, timeout, interval).Should(Succeed())

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When creating an instance with a plan and resource overrides", func() {
		It("Should use instance overrides where specified and plan defaults elsewhere", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "plan-override-test",
					Namespace: "default",
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Plan: "test-plan-small",
					Resources: openclawv1alpha1.ResourcesSpec{
						Limits: openclawv1alpha1.ResourceList{
							Memory: "1Gi",
						},
					},
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secret",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			stsKey := types.NamespacedName{
				Name:      resources.StatefulSetName(instance),
				Namespace: "default",
			}

			By("Verifying instance memory limit override is used")
			Eventually(func(g Gomega) {
				sts := &appsv1.StatefulSet{}
				g.Expect(k8sClient.Get(ctx, stsKey, sts)).To(Succeed())
				container := sts.Spec.Template.Spec.Containers[0]

				// Instance override for memory limit
				g.Expect(container.Resources.Limits[corev1.ResourceMemory]).To(Equal(resource.MustParse("1Gi")))
				// Plan defaults for the rest
				g.Expect(container.Resources.Requests[corev1.ResourceCPU]).To(Equal(resource.MustParse("200m")))
				g.Expect(container.Resources.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("256Mi")))
				g.Expect(container.Resources.Limits[corev1.ResourceCPU]).To(Equal(resource.MustParse("500m")))
			}, timeout, interval).Should(Succeed())

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When creating an instance with a plan that has config defaults", func() {
		It("Should include plan config in the ConfigMap", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "plan-config-test",
					Namespace: "default",
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Plan: "test-plan-small",
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secret",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			cmKey := types.NamespacedName{
				Name:      resources.ConfigMapName(instance),
				Namespace: "default",
			}

			By("Verifying ConfigMap contains plan config")
			Eventually(func(g Gomega) {
				cm := &corev1.ConfigMap{}
				g.Expect(k8sClient.Get(ctx, cmKey, cm)).To(Succeed())

				configJSON, ok := cm.Data["openclaw.json"]
				g.Expect(ok).To(BeTrue(), "openclaw.json key should exist in ConfigMap data")

				var cfg map[string]interface{}
				g.Expect(json.Unmarshal([]byte(configJSON), &cfg)).To(Succeed())

				agents, ok := cfg["agents"].(map[string]interface{})
				g.Expect(ok).To(BeTrue(), "agents key should exist in config")
				defaults, ok := agents["defaults"].(map[string]interface{})
				g.Expect(ok).To(BeTrue(), "agents.defaults key should exist in config")
				g.Expect(defaults["model"]).To(Equal("test/small-model"))
			}, timeout, interval).Should(Succeed())

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})

	Context("When creating an instance without a plan", func() {
		It("Should use instance resources directly with no activePlan", func() {
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "no-plan-test",
					Namespace: "default",
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Resources: openclawv1alpha1.ResourcesSpec{
						Requests: openclawv1alpha1.ResourceList{
							CPU:    "300m",
							Memory: "768Mi",
						},
						Limits: openclawv1alpha1.ResourceList{
							CPU:    "750m",
							Memory: "1536Mi",
						},
					},
					EnvFrom: []corev1.EnvFromSource{
						{
							SecretRef: &corev1.SecretEnvSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: "test-secret",
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			stsKey := types.NamespacedName{
				Name:      resources.StatefulSetName(instance),
				Namespace: "default",
			}

			By("Verifying StatefulSet uses instance resources directly")
			Eventually(func(g Gomega) {
				sts := &appsv1.StatefulSet{}
				g.Expect(k8sClient.Get(ctx, stsKey, sts)).To(Succeed())
				container := sts.Spec.Template.Spec.Containers[0]

				g.Expect(container.Resources.Requests[corev1.ResourceCPU]).To(Equal(resource.MustParse("300m")))
				g.Expect(container.Resources.Requests[corev1.ResourceMemory]).To(Equal(resource.MustParse("768Mi")))
				g.Expect(container.Resources.Limits[corev1.ResourceCPU]).To(Equal(resource.MustParse("750m")))
				g.Expect(container.Resources.Limits[corev1.ResourceMemory]).To(Equal(resource.MustParse("1536Mi")))
			}, timeout, interval).Should(Succeed())

			By("Verifying status.activePlan is empty")
			instanceKey := types.NamespacedName{Name: "no-plan-test", Namespace: "default"}
			Eventually(func(g Gomega) {
				inst := &openclawv1alpha1.OpenClawInstance{}
				g.Expect(k8sClient.Get(ctx, instanceKey, inst)).To(Succeed())
				g.Expect(inst.Status.ActivePlan).To(BeEmpty())
			}, timeout, interval).Should(Succeed())

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
		})
	})
})
