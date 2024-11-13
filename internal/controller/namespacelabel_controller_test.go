package controller_test

import (
	"time"

	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	"github.com/matanamar10/namespacelabel-operator/internal/controller"
	"github.com/matanamar10/namespacelabel-operator/internal/finalizer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	reconciler *controller.NamespacelabelReconciler
	req        ctrl.Request
)

var _ = BeforeSuite(func() {
	Init()

	reconciler = &controller.NamespacelabelReconciler{
		Client:   K8sClient,
		Scheme:   Scheme,
		Log:      ctrl.Log.WithName("controller").WithName("Namespacelabel"),
		Recorder: nil, // Add a mock recorder if needed
	}
})

var _ = AfterSuite(func() {
	Teardown()
})

var _ = Describe("NamespacelabelReconciler", func() {
	var namespace *corev1.Namespace
	var namespacelabel *labelsv1.Namespacelabel

	BeforeEach(func() {
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		}
		Expect(K8sClient.Create(Ctx, namespace)).To(Succeed())

		namespacelabel = &labelsv1.Namespacelabel{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-namespacelabel",
				Namespace: namespace.Name,
			},
			Spec: labelsv1.NamespacelabelSpec{
				Labels: map[string]string{
					"test-key": "test-value",
				},
			},
		}
		Expect(K8sClient.Create(Ctx, namespacelabel)).To(Succeed())

		req = ctrl.Request{NamespacedName: types.NamespacedName{Name: namespacelabel.Name, Namespace: namespace.Name}}
	})

	AfterEach(func() {
		Expect(K8sClient.Delete(Ctx, namespacelabel)).To(Succeed())
		Expect(K8sClient.Delete(Ctx, namespace)).To(Succeed())
	})

	Context("Reconcile Namespacelabel", func() {
		It("should add a finalizer if not present", func() {
			_, err := reconciler.Reconcile(Ctx, req)
			Expect(err).NotTo(HaveOccurred())

			fetchedLabel := &labelsv1.Namespacelabel{}
			Expect(K8sClient.Get(Ctx, req.NamespacedName, fetchedLabel)).To(Succeed())
			Expect(fetchedLabel.GetFinalizers()).To(ContainElement(finalizer.FinalizerName))
		})

		It("should apply labels to the namespace", func() {
			_, err := reconciler.Reconcile(Ctx, req)
			Expect(err).NotTo(HaveOccurred())

			fetchedNamespace := &corev1.Namespace{}
			Expect(K8sClient.Get(Ctx, types.NamespacedName{Name: namespace.Name}, fetchedNamespace)).To(Succeed())
			Expect(fetchedNamespace.Labels).To(HaveKeyWithValue("test-key", "test-value"))
		})

		It("should update status with applied and skipped labels", func() {
			_, err := reconciler.Reconcile(Ctx, req)
			Expect(err).NotTo(HaveOccurred())

			fetchedLabel := &labelsv1.Namespacelabel{}
			Expect(K8sClient.Get(Ctx, req.NamespacedName, fetchedLabel)).To(Succeed())
			Expect(fetchedLabel.Status.AppliedLabels).To(HaveKeyWithValue("test-key", "test-value"))
			Expect(fetchedLabel.Status.SkippedLabels).To(BeEmpty())
			Expect(fetchedLabel.Status.LastUpdated).NotTo(BeNil())
		})

		It("should skip protected labels and emit an event", func() {
			namespacelabel.Spec.Labels["protected-key"] = "protected-value"
			Expect(K8sClient.Update(Ctx, namespacelabel)).To(Succeed())

			_, err := reconciler.Reconcile(Ctx, req)
			Expect(err).NotTo(HaveOccurred())

			fetchedLabel := &labelsv1.Namespacelabel{}
			Expect(K8sClient.Get(Ctx, req.NamespacedName, fetchedLabel)).To(Succeed())
			Expect(fetchedLabel.Status.SkippedLabels).To(HaveKeyWithValue("protected-key", "protected-value"))
		})

		It("should handle deletion of Namespacelabel", func() {
			Expect(K8sClient.Delete(Ctx, namespacelabel)).To(Succeed())

			_, err := reconciler.Reconcile(Ctx, req)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() bool {
				fetchedLabel := &labelsv1.Namespacelabel{}
				err := K8sClient.Get(Ctx, req.NamespacedName, fetchedLabel)
				return client.IgnoreNotFound(err) != nil
			}, time.Second*5).Should(BeTrue())
		})
	})
})
