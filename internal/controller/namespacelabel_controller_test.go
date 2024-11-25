package controller

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	labelsv1alpha1 "github.com/matanamar10/namespacelabel-operator/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
)

var _ = Describe("Namespacelabel Controller", func() {
	const (
		NamespaceName    = "test-namespace"
		NamespaceLabelCR = "test-namespacelabel"
		timeout          = time.Second * 30
		interval         = time.Second * 1
	)

	var (
		ctx      context.Context
		recorder *record.FakeRecorder
	)

	deleteNamespace := func(name string) {
		namespace := &corev1.Namespace{}
		Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: name}, namespace)
			return errors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue(), "Namespace was not fully deleted")
	}

	createNamespace := func(name string) {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		}
		err := k8sClient.Create(ctx, namespace)
		if errors.IsAlreadyExists(err) {
			By("Namespace already exists, ensuring it's clean")
			deleteNamespace(name)
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
		} else {
			Expect(err).To(Succeed())
		}
	}

	deleteAllNamespaceLabels := func() {
		namespaceLabelList := &labelsv1alpha1.NamespacelabelList{}
		Expect(k8sClient.List(ctx, namespaceLabelList)).To(Succeed())
		for _, nl := range namespaceLabelList.Items {
			Expect(k8sClient.Delete(ctx, &nl)).To(Succeed())
		}
		Eventually(func() int {
			Expect(k8sClient.List(ctx, namespaceLabelList)).To(Succeed())
			return len(namespaceLabelList.Items)
		}, timeout, interval).Should(BeZero())
	}

	getNextEvent := func() string {
		select {
		case event := <-recorder.Events:
			return event
		case <-time.After(timeout):
			return ""
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		recorder = record.NewFakeRecorder(100)
		By("Creating a fresh namespace for the test")
		createNamespace(NamespaceName)
	})

	AfterEach(func() {
		By("Cleaning up test resources")
		deleteAllNamespaceLabels()
		deleteNamespace(NamespaceName)
	})

	Context("Full CRUD operations with events", func() {
		It("should create, update, and delete a Namespacelabel CR while emitting events", func() {
			By("Creating a Namespacelabel CR")
			labelsCR := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NamespaceLabelCR,
					Namespace: NamespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "value1"},
				},
			}
			Expect(k8sClient.Create(ctx, labelsCR)).To(Succeed())

			By("Verifying the labels are applied to the namespace")
			Eventually(func() map[string]string {
				namespace := &corev1.Namespace{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)).To(Succeed())
				return namespace.Labels
			}, timeout, interval).Should(HaveKeyWithValue("key1", "value1"))

			By("Verifying an event is emitted for label application")
			Eventually(getNextEvent, timeout, interval).Should(ContainSubstring("AppliedLabels"))

			By("Updating the Namespacelabel CR")
			updatedLabels := map[string]string{"key1": "updated-value", "key2": "value2"}
			Eventually(func() error {
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceLabelCR, Namespace: NamespaceName}, labelsCR)).To(Succeed())
				labelsCR.Spec.Labels = updatedLabels
				return k8sClient.Update(ctx, labelsCR)
			}, timeout, interval).Should(Succeed())

			By("Verifying the updated labels are applied")
			Eventually(func() map[string]string {
				namespace := &corev1.Namespace{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)).To(Succeed())
				return namespace.Labels
			}, timeout, interval).Should(SatisfyAll(
				HaveKeyWithValue("key1", "updated-value"),
				HaveKeyWithValue("key2", "value2"),
			))

			By("Verifying an event is emitted for label update")
			Eventually(getNextEvent, timeout, interval).Should(ContainSubstring("UpdatedLabels"))

			By("Deleting the Namespacelabel CR")
			Expect(k8sClient.Delete(ctx, labelsCR)).To(Succeed())

			By("Verifying the labels are removed from the namespace")
			Eventually(func() map[string]string {
				namespace := &corev1.Namespace{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)).To(Succeed())
				return namespace.Labels
			}, timeout, interval).ShouldNot(HaveKey("key1"))

			By("Verifying an event is emitted for CR deletion")
			Eventually(getNextEvent, timeout, interval).Should(ContainSubstring("DeletedNamespacelabel"))
		})
	})

	Context("Handling protected labels with events", func() {
		It("should skip applying protected labels and emit events", func() {
			By("Creating a Namespacelabel CR with protected labels")
			labelsCR := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NamespaceLabelCR,
					Namespace: NamespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{
						"protected-label": "value",
						"key1":            "value1",
					},
				},
			}
			Expect(k8sClient.Create(ctx, labelsCR)).To(Succeed())

			By("Verifying the protected label was not applied")
			Eventually(func() map[string]string {
				namespace := &corev1.Namespace{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)).To(Succeed())
				return namespace.Labels
			}, timeout, interval).Should(SatisfyAll(
				HaveKeyWithValue("key1", "value1"),
				Not(HaveKey("protected-label")),
			))

			By("Verifying an event for the skipped label")
			Eventually(getNextEvent, timeout, interval).Should(ContainSubstring("ProtectedLabelSkipped"))
		})
	})

	Context("Multiple CRs with overlapping keys", func() {
		It("should not override existing labels and emit events", func() {
			By("Creating the first Namespacelabel CR")
			labelsCR1 := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: "label-1", Namespace: NamespaceName},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "value1"},
				},
			}
			Expect(k8sClient.Create(ctx, labelsCR1)).To(Succeed())

			By("Creating the second Namespacelabel CR")
			labelsCR2 := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{Name: "label-2", Namespace: NamespaceName},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "new-value", "key2": "value2"},
				},
			}
			Expect(k8sClient.Create(ctx, labelsCR2)).To(Succeed())

			By("Verifying labels are applied correctly")
			Eventually(func() map[string]string {
				namespace := &corev1.Namespace{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: NamespaceName}, namespace)).To(Succeed())
				return namespace.Labels
			}, timeout, interval).Should(SatisfyAll(
				HaveKeyWithValue("key1", "value1"),
				HaveKeyWithValue("key2", "value2"),
			))

			By("Verifying an event for the duplicate label")
			Eventually(getNextEvent, timeout, interval).Should(ContainSubstring("DuplicateLabelSkipped"))
		})
	})
})
