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

	Context("When reconciling a NamespaceLabel resource", func() {
		const namespaceName = "default"
		const resourceName = "test-resource"

		It("should emit an event when a protected label is skipped", func() {
			By("creating a NamespaceLabel resource with a protected label")
			namespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"protected-key": "value", "key2": "value2"},
				},
			}
			Expect(k8sClient.Create(ctx, namespaceLabel)).To(Succeed())

			recorder := record.NewFakeRecorder(100)
			reconciler := &NamespacelabelReconciler{
				Client:   k8sClient,
				Scheme:   scheme,
				Log:      zap.New(zap.UseDevMode(true)),
				Recorder: recorder,
			}

			By("reconciling the NamespaceLabel resource")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: resourceName, Namespace: namespaceName}})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the event for the protected label")
			event := getNextEvent(recorder)
			Expect(event).To(ContainSubstring("ProtectedLabelSkipped"))
			Expect(event).To(ContainSubstring("protected-key=value"))
		})

		It("should emit an event when a duplicate label is skipped", func() {
			By("creating a namespace with an existing label")
			createNamespace(namespaceName)
			namespace := &corev1.Namespace{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)).To(Succeed())
			namespace.Labels = map[string]string{"key1": "existing-value"}
			Expect(k8sClient.Update(ctx, namespace)).To(Succeed())

			By("creating a NamespaceLabel resource with a duplicate label")
			namespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "new-value", "key2": "value2"},
				},
			}
			Expect(k8sClient.Create(ctx, namespaceLabel)).To(Succeed())

			recorder := record.NewFakeRecorder(100)
			reconciler := &NamespacelabelReconciler{
				Client:   k8sClient,
				Scheme:   scheme,
				Log:      zap.New(zap.UseDevMode(true)),
				Recorder: recorder,
			}

			By("reconciling the NamespaceLabel resource")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: resourceName, Namespace: namespaceName}})
			Expect(err).NotTo(HaveOccurred())

			By("verifying the event for the duplicate label")
			event := getNextEvent(recorder)
			Expect(event).To(ContainSubstring("DuplicateLabelSkipped"))
			Expect(event).To(ContainSubstring("key1=new-value"))
		})

		It("should emit events for successful label application", func() {
			By("creating a NamespaceLabel resource")
			namespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "value1", "key2": "value2"},
				},
			}
			Expect(k8sClient.Create(ctx, namespaceLabel)).To(Succeed())

			recorder := record.NewFakeRecorder(100)
			reconciler := &NamespacelabelReconciler{
				Client:   k8sClient,
				Scheme:   scheme,
				Log:      zap.New(zap.UseDevMode(true)),
				Recorder: recorder,
			}

			By("reconciling the NamespaceLabel resource")
			_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: resourceName, Namespace: namespaceName}})
			Expect(err).NotTo(HaveOccurred())

			By("verifying events for label application")
			event := getNextEvent(recorder)
			Expect(event).To(ContainSubstring("Normal"))
			Expect(event).To(ContainSubstring("LabelsApplied"))
			Expect(event).To(ContainSubstring("key1=value1"))
		})
	})
})
