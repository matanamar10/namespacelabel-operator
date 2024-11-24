package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	labelsv1alpha1 "github.com/matanamar10/namespacelabel-operator/api/v1alpha1"
)

const (
	namespaceName = "test-namespace"
	timeout       = time.Second * 5
	interval      = time.Millisecond * 250
)

var _ = Describe("Namespacelabel Controller", func() {
	BeforeEach(func() {
		By("Creating a test namespace")
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
		}
		Expect(k8sClient.Create(context.Background(), namespace)).To(Succeed())
	})

	AfterEach(func() {
		By("Cleaning up Namespacelabel resources")
		deleteAllNamespacelabels()

		By("Cleaning up the test namespace")
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: namespaceName},
		}
		Expect(k8sClient.Delete(context.Background(), namespace)).To(Succeed())

		Eventually(func() error {
			return k8sClient.Get(context.Background(), types.NamespacedName{Name: namespaceName}, namespace)
		}, timeout, interval).ShouldNot(Succeed())

	})

	Context("CRUD operations", func() {
		It("should create, update, and delete a Namespacelabel", func() {
			ctx := context.Background()

			By("Creating a Namespacelabel")
			namespacelabel := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-label",
					Namespace: namespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "value1"},
				},
			}
			Expect(k8sClient.Create(ctx, namespacelabel)).To(Succeed())

			By("Verifying that the Namespacelabel was created")
			createdNamespacelabel := &labelsv1alpha1.Namespacelabel{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: "test-label", Namespace: namespaceName}, createdNamespacelabel)
			}, timeout, interval).Should(Succeed())

			By("Updating the Namespacelabel")
			Eventually(func() error {
				latest := &labelsv1alpha1.Namespacelabel{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-label", Namespace: namespaceName}, latest); err != nil {
					return err
				}
				latest.Spec.Labels["key2"] = "value2"
				return k8sClient.Update(ctx, latest)
			}, timeout, interval).Should(Succeed())

			By("Verifying the Namespacelabel was updated")
			Eventually(func() string {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: "test-label", Namespace: namespaceName}, createdNamespacelabel)
				return createdNamespacelabel.Spec.Labels["key2"]
			}, timeout, interval).Should(Equal("value2"))

			By("Deleting the Namespacelabel")
			Expect(k8sClient.Delete(ctx, createdNamespacelabel)).To(Succeed())
		})
	})

	Context("Edge cases", func() {
		It("should not allow overriding existing keys", func() {
			ctx := context.Background()

			By("Creating the first Namespacelabel")
			firstLabel := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "label-1",
					Namespace: namespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "value1"},
				},
			}
			Expect(k8sClient.Create(ctx, firstLabel)).To(Succeed())

			By("Creating the second Namespacelabel with overlapping keys")
			secondLabel := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "label-2",
					Namespace: namespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "new-value", "key2": "value2"},
				},
			}
			Expect(k8sClient.Create(ctx, secondLabel)).To(Succeed())

			By("Verifying the namespace labels are not overridden")
			namespace := &corev1.Namespace{}
			Eventually(func() map[string]string {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)
				return namespace.Labels
			}, timeout, interval).Should(Equal(map[string]string{"key1": "value1", "key2": "value2"}))
		})

		It("should skip applying protected labels", func() {
			ctx := context.Background()

			By("Creating a Namespacelabel with protected labels")
			namespacelabel := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "protected-label",
					Namespace: namespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{
						"protected-key": "protected-value",
						"key1":          "value1",
					},
				},
			}
			Expect(k8sClient.Create(ctx, namespacelabel)).To(Succeed())

			By("Verifying only non-protected labels are applied")
			namespace := &corev1.Namespace{}
			Eventually(func() map[string]string {
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, namespace)
				return namespace.Labels
			}, timeout, interval).Should(Equal(map[string]string{"key1": "value1"}))
		})
	})
})

func deleteAllNamespacelabels() {
	namespaceLabelList := &labelsv1alpha1.NamespacelabelList{}
	Expect(k8sClient.List(context.Background(), namespaceLabelList)).To(Succeed())
	for _, nl := range namespaceLabelList.Items {
		Expect(k8sClient.Delete(context.Background(), &nl)).To(Succeed())
	}
}
