package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	ctx       context.Context
	k8sClient client.Client
	scheme    *runtime.Scheme
)

var _ = BeforeSuite(func() {
	By("Initializing test environment")
	scheme = runtime.NewScheme()
	Expect(labelsv1.AddToScheme(scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx = context.Background()
})

func createNamespace(name string) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
}

func deleteNamespace(name string) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
}

func deleteAllNamespaceLabels() {
	namespaceLabelList := &labelsv1.NamespacelabelList{}
	Expect(k8sClient.List(ctx, namespaceLabelList)).To(Succeed())
	for _, nl := range namespaceLabelList.Items {
		Expect(k8sClient.Delete(ctx, &nl)).To(Succeed())
	}
}

func getNextEvent(recorder *record.FakeRecorder) string {
	select {
	case event := <-recorder.Events:
		return event
	default:
		return ""
	}
}

var _ = Describe("NamespaceLabel Controller", func() {
	BeforeEach(func() {
		By("Creating default namespace")
		createNamespace("default")
	})

	AfterEach(func() {
		By("Cleaning up test resources")
		deleteAllNamespaceLabels()
		deleteNamespace("default")
	})

	Context("When reconciling multiple NamespaceLabel resources in the same namespace", func() {
		const namespaceName = "default"
		const firstResourceName = "test-resource-1"
		const secondResourceName = "test-resource-2"

		It("should apply only non-overlapping labels from multiple NamespaceLabel resources", func() {
			By("creating the first NamespaceLabel resource")
			firstNamespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: firstResourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "value1", "key2": "value2"},
				},
			}
			Expect(k8sClient.Create(ctx, firstNamespaceLabel)).To(Succeed())

			By("creating the second NamespaceLabel resource with overlapping labels")
			secondNamespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: secondResourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"key2": "new-value", "key3": "value3"},
				},
			}
			Expect(k8sClient.Create(ctx, secondNamespaceLabel)).To(Succeed())

			recorder := record.NewFakeRecorder(100)
			reconciler := &NamespacelabelReconciler{
				Client:   k8sClient,
				Scheme:   scheme,
				Log:      zap.New(zap.UseDevMode(true)),
				Recorder: recorder,
			}

			By("reconciling the first NamespaceLabel")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: firstResourceName, Namespace: namespaceName}})
			Expect(err).NotTo(HaveOccurred())

			By("reconciling the second NamespaceLabel")
			_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: secondResourceName, Namespace: namespaceName}})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the Namespace contains only unique keys")
			namespace := &corev1.Namespace{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)).To(Succeed())
			Expect(namespace.Labels).To(HaveKeyWithValue("key1", "value1"))
			Expect(namespace.Labels).To(HaveKeyWithValue("key2", "value2")) // Original value retained
			Expect(namespace.Labels).To(HaveKeyWithValue("key3", "value3"))

			By("verifying an event for the skipped label in the second NamespaceLabel")
			event := getNextEvent(recorder)
			Expect(event).To(ContainSubstring("DuplicateLabelSkipped"))
			Expect(event).To(ContainSubstring("key2=new-value"))
		})

		It("should remove only the labels associated with a deleted NamespaceLabel resource", func() {
			By("creating two NamespaceLabel resources")
			firstNamespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: firstResourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "value1", "key2": "value2"},
				},
			}
			Expect(k8sClient.Create(ctx, firstNamespaceLabel)).To(Succeed())

			secondNamespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: secondResourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"key3": "value3"},
				},
			}
			Expect(k8sClient.Create(ctx, secondNamespaceLabel)).To(Succeed())

			recorder := record.NewFakeRecorder(100)
			reconciler := &NamespacelabelReconciler{
				Client:   k8sClient,
				Scheme:   scheme,
				Log:      zap.New(zap.UseDevMode(true)),
				Recorder: recorder,
			}

			By("reconciling both NamespaceLabels")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: firstResourceName, Namespace: namespaceName}})
			Expect(err).NotTo(HaveOccurred())
			_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: secondResourceName, Namespace: namespaceName}})
			Expect(err).NotTo(HaveOccurred())

			By("deleting the first NamespaceLabel resource")
			Expect(k8sClient.Delete(ctx, firstNamespaceLabel)).To(Succeed())
			_, err = reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: firstResourceName, Namespace: namespaceName}})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the remaining labels in the Namespace")
			namespace := &corev1.Namespace{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)).To(Succeed())
			Expect(namespace.Labels).NotTo(HaveKey("key1"))
			Expect(namespace.Labels).NotTo(HaveKey("key2"))
			Expect(namespace.Labels).To(HaveKeyWithValue("key3", "value3"))
		})
	})
})
