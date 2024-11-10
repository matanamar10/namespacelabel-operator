package e2e_tests

import (
	"context"

	"github.com/matanamar10/namespacelabel-operator/test/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("NamespaceLabel Operator E2E Tests", func() {
	It("should apply labels to the namespace as specified by Namespacelabel CR", func() {
		By("Creating a test namespace")
		namespace := mocks.CreateBaseNamespace()
		Expect(k8sClient.Create(context.Background(), namespace)).To(Succeed(), "failed to create test namespace")

		By("Creating a Namespacelabel CR with labels")
		nsLabel := mocks.CreateBaseNamespacelabel()
		Expect(k8sClient.Create(context.Background(), nsLabel)).To(Succeed(), "failed to create Namespacelabel CR")

		By("Verifying the label is applied to the namespace")
		Eventually(func() map[string]string {
			err := k8sClient.Get(context.Background(), ctrlclient.ObjectKey{Name: namespace.Name}, namespace)
			Expect(err).NotTo(HaveOccurred(), "failed to get test namespace")
			return namespace.Labels
		}).Should(HaveKeyWithValue("environment", "test"), "namespace should have 'environment=test' label")

		By("Cleaning up the Namespacelabel CR")
		Expect(k8sClient.Delete(context.Background(), nsLabel)).To(Succeed(), "failed to delete Namespacelabel CR")

		By("Verifying the namespace no longer has the label after cleanup")
		Eventually(func() bool {
			// Fetch namespace and check if the label is removed
			err := k8sClient.Get(context.Background(), ctrlclient.ObjectKey{Name: namespace.Name}, namespace)
			Expect(err).NotTo(HaveOccurred(), "failed to get test namespace after cleanup")
			_, exists := namespace.Labels["environment"]
			return exists
		}).Should(BeFalse(), "namespace should no longer have the 'environment' label after deletion")
	})
})
