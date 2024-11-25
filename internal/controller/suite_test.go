package controller

import (
	"context"
	"fmt"
	"k8s.io/client-go/rest"
	"path/filepath"
	"testing"

	labelsv1alpha1 "github.com/matanamar10/namespacelabel-operator/api/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	cfg       *rest.Config
	k8sClient client.Client
	testEnv   *envtest.Environment
	ctx       context.Context
	cancel    context.CancelFunc
)

func TestAPIs(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "Controller Suite")
}

var _ = ginkgo.BeforeSuite(func() {
	ginkgo.By("Setting up the test environment")

	// Initialize the logger
	ctrl.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))

	// Set up a context for managing test lifecycle
	ctx, cancel = context.WithCancel(context.TODO())

	// Initialize envtest environment
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,

		// Optional: Configure binary assets directory if needed
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	cfg, err = testEnv.Start()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to start envtest environment")
	gomega.Expect(cfg).NotTo(gomega.BeNil())

	err = labelsv1alpha1.AddToScheme(scheme.Scheme)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to add Namespacelabel schema")
	// +kubebuilder:scaffold:scheme

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to create controller manager")

	err = (&NamespacelabelReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
		Log:    ctrl.Log.WithName("controllers").WithName("Namespacelabel"),
	}).SetupWithManager(k8sManager)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to set up Namespacelabel reconciler")

	go func() {
		defer ginkgo.GinkgoRecover()
		err = k8sManager.Start(ctx)
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to start manager")
	}()

	k8sClient = k8sManager.GetClient()
	gomega.Expect(k8sClient).NotTo(gomega.BeNil(), "k8sClient is nil")
})

var _ = ginkgo.AfterSuite(func() {
	ginkgo.By("Tearing down the test environment")

	cancel()

	err := testEnv.Stop()
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Failed to stop envtest environment")
})
