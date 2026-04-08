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
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	openclawv1alpha1 "github.com/openclawrocks/openclaw-operator/api/v1alpha1"
	"github.com/openclawrocks/openclaw-operator/internal/plans"
)

const envtestK8sVersion = "1.31.0"

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func resolveEnvtestBinaryAssetsDirectory() (string, error) {
	if assetsDir := os.Getenv("KUBEBUILDER_ASSETS"); assetsDir != "" {
		return assetsDir, nil
	}

	assetsDir := filepath.Join("..", "..", "bin", "k8s", fmt.Sprintf("%s-%s-%s", envtestK8sVersion, runtime.GOOS, runtime.GOARCH))
	for _, binary := range []string{"etcd", "kube-apiserver", "kubectl"} {
		if _, err := os.Stat(filepath.Join(assetsDir, binary)); err != nil {
			return "", fmt.Errorf(
				"envtest binaries not found at %q and KUBEBUILDER_ASSETS is not set; run `make envtest` once, then rerun `go test ./internal/controller/...`, or use `make test-controller`: missing %s",
				assetsDir,
				binary,
			)
		}
	}

	return assetsDir, nil
}

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	assetsDir, err := resolveEnvtestBinaryAssetsDirectory()
	Expect(err).NotTo(HaveOccurred())

	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: assetsDir,
	}

	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = openclawv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Start the controller manager
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).NotTo(HaveOccurred())

	testPlanRegistry := plans.NewRegistryFromMap(map[string]plans.ServicePlan{
		"test-plan-small": {
			DisplayName: "Small Test Plan",
			Description: "Small plan for envtest",
			Resources: plans.PlanResources{
				Requests: plans.PlanResourceList{CPU: "200m", Memory: "256Mi"},
				Limits:   plans.PlanResourceList{CPU: "500m", Memory: "512Mi"},
			},
			Storage: plans.PlanStorage{Size: "2Gi"},
			Config: map[string]interface{}{
				"agents": map[string]interface{}{
					"defaults": map[string]interface{}{
						"model": "test/small-model",
					},
				},
			},
			Overridable: []string{"config", "storage.size"},
		},
		"test-plan-locked": {
			DisplayName: "Locked Test Plan",
			Description: "Locked plan for envtest - nothing overridable",
			Resources: plans.PlanResources{
				Requests: plans.PlanResourceList{CPU: "1", Memory: "2Gi"},
				Limits:   plans.PlanResourceList{CPU: "2", Memory: "4Gi"},
			},
			Storage: plans.PlanStorage{Size: "10Gi"},
			Config: map[string]interface{}{
				"agents": map[string]interface{}{
					"defaults": map[string]interface{}{
						"model": "test/locked-model",
					},
				},
			},
			Overridable: []string{},
		},
	})

	err = (&OpenClawInstanceReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Recorder:          mgr.GetEventRecorderFor("openclawinstance-controller"),
		OperatorNamespace: "default",
		PlanRegistry:      testPlanRegistry,
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	err = (&OpenClawSelfConfigReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("openclawselfconfig-controller"),
	}).SetupWithManager(mgr)
	Expect(err).NotTo(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	}()
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
