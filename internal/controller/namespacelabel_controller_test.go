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
	"k8s.io/client-go/util/retry"
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

func initTestEnvironment() {
	scheme = runtime.NewScheme()
	Expect(labelsv1.AddToScheme(scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx = context.Background()
}

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

var _ = Describe("NamespaceLabel Controller", func() {
	BeforeEach(func() {
		initTestEnvironment()
		createNamespace("default")
	})

	AfterEach(func() {
		deleteAllNamespaceLabels()
		deleteNamespace("default")
	})

	Context("When reconciling a NamespaceLabel resource", func() {
		const namespaceName = "default"
		const resourceName = "test-resource"
		const secondresourceName = "test-second-resource"
		namespacedName := types.NamespacedName{Name: resourceName, Namespace: namespaceName}

		It("should successfully create, update, delete labels and delete NamespaceLabels", func() {
			By("creating the NamespaceLabel resource")
			namespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"label_1": "a", "label_2": "b"},
				},
			}
			Expect(k8sClient.Create(ctx, namespaceLabel)).To(Succeed())

			created := &labelsv1.Namespacelabel{}
			Expect(k8sClient.Get(ctx, namespacedName, created)).To(Succeed())

			By("reconciling the created resource")
			controllerReconciler := &NamespacelabelReconciler{
				Client: k8sClient,
				Scheme: scheme,
				Log:    zap.New(zap.UseDevMode(true)),
			}
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			By("checking that the labels were applied to the Namespace")
			namespace := &corev1.Namespace{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)).To(Succeed())
			Expect(namespace.Labels).To(HaveKeyWithValue("label_1", "a"))
			Expect(namespace.Labels).To(HaveKeyWithValue("label_2", "b"))

			By("updating the NamespaceLabel resource")
			retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
				if err := k8sClient.Get(ctx, namespacedName, namespaceLabel); err != nil {
					return err
				}
				namespaceLabel.Spec.Labels["label_1"] = "updated"
				return k8sClient.Update(ctx, namespaceLabel)
			})
			Expect(retryErr).To(Succeed())
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)).To(Succeed())
			Expect(namespace.Labels).To(HaveKeyWithValue("label_1", "updated"))

			By("deleting a single label from the NamespaceLabel resource")
			delete(namespaceLabel.Spec.Labels, "label_2")
			Expect(k8sClient.Update(ctx, namespaceLabel)).To(Succeed())
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)).To(Succeed())
			Expect(namespace.Labels).NotTo(HaveKey("label_2"))

			By("deleting the NamespaceLabel resource")
			Expect(k8sClient.Delete(ctx, namespaceLabel)).To(Succeed())
			_, err = controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)).To(Succeed())
			Expect(namespace.Labels).NotTo(HaveKey("label_1"))
		})

		It("should prevent creating NamespaceLabel with managed labels", func() {
			By("creating the NamespaceLabel with managed labels")
			namespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: "managed-label-resource", Namespace: namespaceName},
				Spec:       labelsv1.NamespacelabelSpec{Labels: map[string]string{"kubernetes.io/managed": "true"}},
			}
			Expect(k8sClient.Create(ctx, namespaceLabel)).To(Succeed())

			By("reconciling the resource")
			controllerReconciler := &NamespacelabelReconciler{
				Client: k8sClient,
				Scheme: scheme,
				Log:    zap.New(zap.UseDevMode(true)),
			}
			_, err := controllerReconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: "managed-label-resource", Namespace: namespaceName}})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot add protected or management label 'kubernetes.io/managed'"))
		})
		It("should allow multiple NamespaceLabels but apply only unique keys", func() {
			By("creating the first NamespaceLabel resource")
			firstResourceName := resourceName
			firstNamespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: firstResourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "value1", "key2": "value2"},
				},
			}
			Expect(k8sClient.Create(ctx, firstNamespaceLabel)).To(Succeed())

			By("creating the second NamespaceLabel resource")
			secondResourceName := secondresourceName
			secondNamespaceLabel := &labelsv1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: secondResourceName, Namespace: namespaceName},
				Spec: labelsv1.NamespacelabelSpec{
					Labels: map[string]string{"key2": "new-value", "key3": "value3"},
				},
			}
			Expect(k8sClient.Create(ctx, secondNamespaceLabel)).To(Succeed())

			reconciler := &NamespacelabelReconciler{
				Client: k8sClient,
				Scheme: scheme,
				Log:    zap.New(zap.UseDevMode(true)),
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
			Expect(namespace.Labels).To(HaveKeyWithValue("key2", "value2"))
			Expect(namespace.Labels).To(HaveKeyWithValue("key3", "value3"))
		})

	})
})
