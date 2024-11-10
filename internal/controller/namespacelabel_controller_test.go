// controller_test.go
package controller_test

import (
	"context"
	"path/filepath"

	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	"github.com/matanamar10/namespacelabel-operator/internal/controller"
	"github.com/matanamar10/namespacelabel-operator/internal/finalizer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var (
	reconciler     *controller.NamespacelabelReconciler
	namespace      *corev1.Namespace
	namespacelabel *labelsv1.Namespacelabel
	req            ctrl.Request
)

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	var err error
	cfg, err := testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = labelsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	// Initialize the reconciler with k8sClient, scheme, and a logger
	reconciler = &controller.NamespacelabelReconciler{
		Client: k8sClient,
		Scheme: scheme.Scheme,
		Log:    ctrl.Log.WithName("controller").WithName("Namespacelabel"),
	}
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("NamespacelabelReconciler", func() {
	BeforeEach(func() {
		namespace = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-namespace",
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())

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
		Expect(k8sClient.Create(ctx, namespacelabel)).To(Succeed())

		req = ctrl.Request{NamespacedName: types.NamespacedName{Name: namespacelabel.Name, Namespace: namespace.Name}}
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, namespacelabel)).To(Succeed())
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
	})

	Context("Reconcile Namespacelabel", func() {
		It("should add a finalizer if not present", func() {
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			fetchedLabel := &labelsv1.Namespacelabel{}
			Expect(k8sClient.Get(ctx, req.NamespacedName, fetchedLabel)).To(Succeed())
			Expect(fetchedLabel.GetFinalizers()).To(ContainElement(finalizer.FinalizerName))
		})

		It("should apply labels to the namespace", func() {
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			fetchedNamespace := &corev1.Namespace{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespace.Name}, fetchedNamespace)).To(Succeed())
			Expect(fetchedNamespace.Labels).To(HaveKeyWithValue("test-key", "test-value"))
		})

		It("should update status with applied and skipped labels", func() {
			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			fetchedLabel := &labelsv1.Namespacelabel{}
			Expect(k8sClient.Get(ctx, req.NamespacedName, fetchedLabel)).To(Succeed())

			Expect(fetchedLabel.Status.AppliedLabels).To(HaveKeyWithValue("test-key", "test-value"))
			Expect(fetchedLabel.Status.SkippedLabels).To(BeEmpty())
			Expect(fetchedLabel.Status.LastUpdated).NotTo(BeNil())
		})

		It("should skip protected labels and emit an event", func() {
			namespacelabel.Spec.Labels["protected-key"] = "protected-value"
			Expect(k8sClient.Update(ctx, namespacelabel)).To(Succeed())

			_, err := reconciler.Reconcile(ctx, req)
			Expect(err).NotTo(HaveOccurred())

			fetchedLabel := &labelsv1.Namespacelabel{}
			Expect(k8sClient.Get(ctx, req.NamespacedName, fetchedLabel)).To(Succeed())
			Expect(fetchedLabel.Status.SkippedLabels).To(HaveKeyWithValue("protected-key", "protected-value"))
		})
	})
})
