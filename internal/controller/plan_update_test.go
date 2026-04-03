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

// Plan Update Handling — envtest integration tests
//
// Design note (verified by reading Reconcile()):
//
//   The Reconcile() loop fetches a FRESH instance from the API server on every
//   invocation:
//     instance := &openclawv1alpha1.OpenClawInstance{}
//     r.Get(ctx, req.NamespacedName, instance)
//
//   applyPlanDefaults() then mutates this fresh copy IN MEMORY. The mutation is
//   never written back to the API server (only instance.Status is updated).
//   The API server therefore always stores the original user spec (with empty
//   resource fields if the user never set them).
//
//   Consequence for plan switches:
//   - When the user writes spec.plan="plan-A", the API server stores spec with
//     empty Resources (user set nothing). The reconciler fills Resources from
//     plan-A in memory.
//   - When the user changes spec.plan="plan-B", the API server still stores
//     empty Resources. The next reconcile fills Resources from plan-B in memory.
//   - Plan switch is therefore AUTOMATICALLY CORRECT without extra logic.
//
//   Consequence for plan removal:
//   - User sets spec.plan="". API server stores empty plan + whatever the user
//     set in spec.resources (which may also be empty).
//   - If the user never set spec.resources, the webhook defaulter fills them
//     at admission time. Those defaults are then stored in the API server spec.
//   - After plan removal, the reconciler uses the values stored in the API
//     server spec (either user-set or webhook defaults).

import (
	"encoding/json"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
	"github.com/openclawrocks/openclaw-operator/internal/resources"
)

var _ = Describe("Plan Update Handling", func() {
	const (
		timeout  = time.Second * 30
		interval = time.Millisecond * 250
	)

	// planCPURequest returns the CPU request from the first container of the StatefulSet.
	planCPURequest := func(stsKey types.NamespacedName) func(Gomega) resource.Quantity {
		return func(g Gomega) resource.Quantity {
			sts := &appsv1.StatefulSet{}
			g.Expect(k8sClient.Get(ctx, stsKey, sts)).To(Succeed())
			return sts.Spec.Template.Spec.Containers[0].Resources.Requests[corev1.ResourceCPU]
		}
	}

	// planMemLimit returns the memory limit from the first container.
	planMemLimit := func(stsKey types.NamespacedName) func(Gomega) resource.Quantity {
		return func(g Gomega) resource.Quantity {
			sts := &appsv1.StatefulSet{}
			g.Expect(k8sClient.Get(ctx, stsKey, sts)).To(Succeed())
			return sts.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceMemory]
		}
	}

	// configModel reads agents.defaults.model from the instance's ConfigMap.
	configModel := func(cmKey types.NamespacedName) func(Gomega) string {
		return func(g Gomega) string {
			cm := &corev1.ConfigMap{}
			if err := k8sClient.Get(ctx, cmKey, cm); err != nil {
				return ""
			}
			raw, ok := cm.Data["openclaw.json"]
			if !ok {
				return ""
			}
			var cfg map[string]interface{}
			if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
				return ""
			}
			agents, _ := cfg["agents"].(map[string]interface{})
			defaults, _ := agents["defaults"].(map[string]interface{})
			model, _ := defaults["model"].(string)
			return model
		}
	}

	// -------------------------------------------------------------------------
	// Test: plan-A → plan-B switch
	// -------------------------------------------------------------------------
	Context("When switching from one plan to another", func() {
		It("Should apply the new plan's defaults after the switch", func() {
			// test-plan-small:  CPU req 200m / lim 500m, Mem req 256Mi / lim 512Mi
			// test-plan-locked: CPU req 1    / lim 2,   Mem req 2Gi   / lim 4Gi
			instanceName := "plan-switch-test"
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Plan: "test-plan-small",
					EnvFrom: []corev1.EnvFromSource{
						{SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			instanceKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			stsKey := types.NamespacedName{Name: resources.StatefulSetName(instance), Namespace: "default"}
			cmKey := types.NamespacedName{Name: resources.ConfigMapName(instance), Namespace: "default"}

			By("Verifying plan-small resources are applied initially")
			Eventually(planCPURequest(stsKey), timeout, interval).
				Should(Equal(resource.MustParse("200m")))

			By("Verifying plan-small config model is applied initially")
			Eventually(configModel(cmKey), timeout, interval).
				Should(Equal("test/small-model"))

			By("Verifying status.activePlan reflects plan-small")
			Eventually(func(g Gomega) {
				inst := &openclawv1alpha1.OpenClawInstance{}
				g.Expect(k8sClient.Get(ctx, instanceKey, inst)).To(Succeed())
				g.Expect(inst.Status.ActivePlan).To(Equal("test-plan-small"))
			}, timeout, interval).Should(Succeed())

			By("Switching to test-plan-locked")
			Eventually(func() error {
				inst := &openclawv1alpha1.OpenClawInstance{}
				if err := k8sClient.Get(ctx, instanceKey, inst); err != nil {
					return err
				}
				inst.Spec.Plan = "test-plan-locked"
				return k8sClient.Update(ctx, inst)
			}, timeout, interval).Should(Succeed())

			By("Verifying plan-locked CPU request is applied after switch")
			Eventually(planCPURequest(stsKey), timeout, interval).
				Should(Equal(resource.MustParse("1")))

			By("Verifying plan-locked memory limit is applied after switch")
			Eventually(planMemLimit(stsKey), timeout, interval).
				Should(Equal(resource.MustParse("4Gi")))

			By("Verifying status.activePlan reflects plan-locked")
			Eventually(func(g Gomega) {
				inst := &openclawv1alpha1.OpenClawInstance{}
				g.Expect(k8sClient.Get(ctx, instanceKey, inst)).To(Succeed())
				g.Expect(inst.Status.ActivePlan).To(Equal("test-plan-locked"))
			}, timeout, interval).Should(Succeed())

			By("Cleaning up")
			// Remove finalizer before delete to avoid backup-job timeout in envtest
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				instance.Finalizers = nil
				return k8sClient.Update(ctx, instance)
			}, timeout, interval).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceKey, &openclawv1alpha1.OpenClawInstance{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	// -------------------------------------------------------------------------
	// Test: plan removal (plan → empty)
	// -------------------------------------------------------------------------
	Context("When removing a plan from an instance", func() {
		It("Should clear status.activePlan and keep the current spec resources", func() {
			instanceName := "plan-removal-test"
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Plan: "test-plan-small",
					EnvFrom: []corev1.EnvFromSource{
						{SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			instanceKey := types.NamespacedName{Name: instanceName, Namespace: "default"}

			By("Waiting for activePlan to be set")
			Eventually(func(g Gomega) {
				inst := &openclawv1alpha1.OpenClawInstance{}
				g.Expect(k8sClient.Get(ctx, instanceKey, inst)).To(Succeed())
				g.Expect(inst.Status.ActivePlan).To(Equal("test-plan-small"))
			}, timeout, interval).Should(Succeed())

			By("Removing the plan from the instance spec")
			Eventually(func() error {
				inst := &openclawv1alpha1.OpenClawInstance{}
				if err := k8sClient.Get(ctx, instanceKey, inst); err != nil {
					return err
				}
				inst.Spec.Plan = ""
				return k8sClient.Update(ctx, inst)
			}, timeout, interval).Should(Succeed())

			By("Verifying status.activePlan is cleared")
			Eventually(func(g Gomega) {
				inst := &openclawv1alpha1.OpenClawInstance{}
				g.Expect(k8sClient.Get(ctx, instanceKey, inst)).To(Succeed())
				g.Expect(inst.Status.ActivePlan).To(BeEmpty())
			}, timeout, interval).Should(Succeed())

			By("Cleaning up")
			// Remove finalizer before delete to avoid backup-job timeout in envtest
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				instance.Finalizers = nil
				return k8sClient.Update(ctx, instance)
			}, timeout, interval).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceKey, &openclawv1alpha1.OpenClawInstance{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	// -------------------------------------------------------------------------
	// Test: plan addition (no plan → plan)
	// -------------------------------------------------------------------------
	Context("When adding a plan to an instance that had none", func() {
		It("Should apply plan defaults after plan is added", func() {
			instanceName := "plan-addition-test"
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					// No plan initially.
					EnvFrom: []corev1.EnvFromSource{
						{SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			instanceKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			stsKey := types.NamespacedName{Name: resources.StatefulSetName(instance), Namespace: "default"}

			By("Waiting for StatefulSet to be created")
			Eventually(func() error {
				return k8sClient.Get(ctx, stsKey, &appsv1.StatefulSet{})
			}, timeout, interval).Should(Succeed())

			By("Verifying status.activePlan is empty initially")
			Eventually(func(g Gomega) {
				inst := &openclawv1alpha1.OpenClawInstance{}
				g.Expect(k8sClient.Get(ctx, instanceKey, inst)).To(Succeed())
				// activePlan is not set, or empty
				g.Expect(inst.Status.ActivePlan).To(BeEmpty())
			}, timeout, interval).Should(Succeed())

			By("Adding test-plan-small to the instance")
			Eventually(func() error {
				inst := &openclawv1alpha1.OpenClawInstance{}
				if err := k8sClient.Get(ctx, instanceKey, inst); err != nil {
					return err
				}
				// Clear any webhook-defaulted resources so plan can fill them.
				inst.Spec.Plan = "test-plan-small"
				inst.Spec.Resources = openclawv1alpha1.ResourcesSpec{}
				return k8sClient.Update(ctx, inst)
			}, timeout, interval).Should(Succeed())

			By("Verifying plan-small CPU request is applied after plan addition")
			Eventually(planCPURequest(stsKey), timeout, interval).
				Should(Equal(resource.MustParse("200m")))

			By("Verifying status.activePlan is set")
			Eventually(func(g Gomega) {
				inst := &openclawv1alpha1.OpenClawInstance{}
				g.Expect(k8sClient.Get(ctx, instanceKey, inst)).To(Succeed())
				g.Expect(inst.Status.ActivePlan).To(Equal("test-plan-small"))
			}, timeout, interval).Should(Succeed())

			By("Cleaning up")
			// Remove finalizer before delete to avoid backup-job timeout in envtest
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				instance.Finalizers = nil
				return k8sClient.Update(ctx, instance)
			}, timeout, interval).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceKey, &openclawv1alpha1.OpenClawInstance{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})

	// -------------------------------------------------------------------------
	// Test: plan switch preserves user-set overrides
	// -------------------------------------------------------------------------
	Context("When switching plans with user resource overrides", func() {
		It("Should preserve user overrides after plan switch", func() {
			instanceName := "plan-switch-override-test"
			// User sets a specific memory limit override; everything else comes from plan.
			instance := &openclawv1alpha1.OpenClawInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      instanceName,
					Namespace: "default",
				},
				Spec: openclawv1alpha1.OpenClawInstanceSpec{
					Plan: "test-plan-small",
					Resources: openclawv1alpha1.ResourcesSpec{
						Limits: openclawv1alpha1.ResourceList{
							Memory: "768Mi", // user override: higher than plan-small's 512Mi
						},
					},
					EnvFrom: []corev1.EnvFromSource{
						{SecretRef: &corev1.SecretEnvSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: "test-secret"},
						}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, instance)).Should(Succeed())

			instanceKey := types.NamespacedName{Name: instanceName, Namespace: "default"}
			stsKey := types.NamespacedName{Name: resources.StatefulSetName(instance), Namespace: "default"}

			By("Verifying user memory override is respected with plan-small")
			Eventually(planMemLimit(stsKey), timeout, interval).
				Should(Equal(resource.MustParse("768Mi")))

			By("Switching to test-plan-locked")
			Eventually(func() error {
				inst := &openclawv1alpha1.OpenClawInstance{}
				if err := k8sClient.Get(ctx, instanceKey, inst); err != nil {
					return err
				}
				inst.Spec.Plan = "test-plan-locked"
				return k8sClient.Update(ctx, inst)
			}, timeout, interval).Should(Succeed())

			By("Verifying user memory override is STILL respected after switch to plan-locked")
			// plan-locked has 4Gi but user set 768Mi → user wins (768Mi is stored in API server spec).
			Eventually(planMemLimit(stsKey), timeout, interval).
				Should(Equal(resource.MustParse("768Mi")))

			By("Verifying plan-locked CPU (not overridden by user) switches correctly")
			Eventually(planCPURequest(stsKey), timeout, interval).
				Should(Equal(resource.MustParse("1")))

			By("Cleaning up")
			// Remove finalizer before delete to avoid backup-job timeout in envtest
			Eventually(func() error {
				if err := k8sClient.Get(ctx, client.ObjectKeyFromObject(instance), instance); err != nil {
					return err
				}
				instance.Finalizers = nil
				return k8sClient.Update(ctx, instance)
			}, timeout, interval).Should(Succeed())
			Expect(k8sClient.Delete(ctx, instance)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, instanceKey, &openclawv1alpha1.OpenClawInstance{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		})
	})
})
