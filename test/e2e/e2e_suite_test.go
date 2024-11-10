// e2e_suite_test.go
package e2e_tests

import (
	"context"
	"testing"

	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	"github.com/matanamar10/namespacelabel-operator/test/mocks"
	"github.com/matanamar10/namespacelabel-operator/test/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var k8sClient client.Client

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "E2E Suite")
}

// newScheme registers custom resources and core Kubernetes resources to the scheme.
func newScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = labelsv1.AddToScheme(s)
	return s
}

var _ = SynchronizedBeforeSuite(func() {
	By("Setting up Kubernetes client for E2E tests")
	initClient()
	cleanUp()
	createE2ETestNamespace()
}, func() {
	initClient()
})

var _ = SynchronizedAfterSuite(func() {}, func() {
	cleanUp()
})

// initClient initializes the Kubernetes client for use in tests.
func initClient() {
	cfg, err := config.GetConfig()
	Expect(err).NotTo(HaveOccurred(), "failed to get Kubernetes config")

	k8sClient, err = client.New(cfg, client.Options{Scheme: newScheme()})
	Expect(err).NotTo(HaveOccurred(), "failed to create Kubernetes client")
	Expect(k8sClient).NotTo(BeNil())
}

// createE2ETestNamespace creates a test namespace for running e2e tests.
func createE2ETestNamespace() {
	namespace := mocks.CreateBaseNamespace()

	Expect(k8sClient.Create(context.Background(), namespace)).To(Succeed(), "failed to create test namespace")
	Eventually(func() bool {
		return utils.DoesResourceExist(k8sClient, namespace)
	}).Should(BeTrue(), "The namespace should be created")
}

// cleanUp removes any resources created in the e2e test namespace.
func cleanUp() {
	namespace := mocks.CreateBaseNamespace()
	if utils.DoesResourceExist(k8sClient, namespace) {
		Expect(k8sClient.Delete(context.Background(), namespace)).To(Succeed(), "failed to delete test namespace")
		Eventually(func() error {
			return k8sClient.Get(context.Background(), client.ObjectKey{Name: namespace.Name}, namespace)
		}).Should(HaveOccurred(), "The namespace should be deleted")
	}
}
