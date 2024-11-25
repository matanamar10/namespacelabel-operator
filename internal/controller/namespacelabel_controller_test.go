package controller

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	labelsv1alpha1 "github.com/matanamar10/namespacelabel-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Namespacelabel Controller", func() {
	const (
		NamespaceName    = "test-namespace"
		NamespaceLabelCR = "test-namespacelabel"
		timeout          = time.Second * 10
		interval         = time.Millisecond * 250
	)

	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()

		By("Creating a test namespace")
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: NamespaceName,
			},
		}
		Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
	})

	AfterEach(func() {
		By("Cleaning up the test namespace")
		namespace := &corev1.Namespace{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)).To(Succeed())
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
	})

	Context("CRUD operations", func() {
		It("Should handle creation, update, and deletion of Namespacelabel", func() {
			By("Creating a Namespacelabel CR")
			labelsCR := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NamespaceLabelCR,
					Namespace: NamespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			}
			Expect(k8sClient.Create(ctx, labelsCR)).To(Succeed())

			By("Verifying the labels are applied to the namespace")
			Eventually(func() map[string]string {
				namespace := &corev1.Namespace{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)
				return namespace.Labels
			}, timeout, interval).Should(HaveKeyWithValue("key1", "value1"))

			By("Deleting the Namespacelabel CR")
			Expect(k8sClient.Delete(ctx, labelsCR)).To(Succeed())
		})
	})

	Context("Handling multiple Namespacelabel CRs", func() {
		It("Should not override existing labels with multiple CRs", func() {
			By("Creating multiple Namespacelabel CRs")
			labelsCR1 := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "namespacelabel1",
					Namespace: NamespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{
						"key1": "value1",
					},
				},
			}
			Expect(k8sClient.Create(ctx, labelsCR1)).To(Succeed())

			labelsCR2 := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "namespacelabel2",
					Namespace: NamespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{
						"key2": "value2",
					},
				},
			}
			Expect(k8sClient.Create(ctx, labelsCR2)).To(Succeed())

			By("Verifying both labels are applied to the namespace")
			Eventually(func() map[string]string {
				namespace := &corev1.Namespace{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)
				return namespace.Labels
			}, timeout, interval).Should(SatisfyAll(
				HaveKeyWithValue("key1", "value1"),
				HaveKeyWithValue("key2", "value2"),
			))
		})
	})

	Context("Skipping protected labels", func() {
		It("Should skip applying protected labels", func() {
			By("Creating a Namespacelabel CR with a protected label")
			labelsCR := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NamespaceLabelCR,
					Namespace: NamespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{
						"protected-label": "new-value",
					},
				},
			}
			Expect(k8sClient.Create(ctx, labelsCR)).To(Succeed())

			By("Verifying the protected label was not applied")
			Eventually(func() map[string]string {
				namespace := &corev1.Namespace{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)
				return namespace.Labels
			}, timeout, interval).ShouldNot(HaveKeyWithValue("protected-label", "new-value"))

			By("Ensuring the namespace still has the original protected label value")
			Eventually(func() map[string]string {
				namespace := &corev1.Namespace{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)
				return namespace.Labels
			}, timeout, interval).ShouldNot(HaveKey("protected-label"))
		})
	})
})
