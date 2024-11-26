/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"time"

	labelsv1alpha1 "github.com/matanamar10/namespacelabel-operator/api/v1alpha1"
)

var _ = Describe("Namespacelabel Webhook", func() {
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
		deleteNamespace(NamespaceName)
	})

	Context("Webhook validation with events", func() {
		It("should reject a second Namespacelabel in the same namespace and emit an event", func() {
			By("Creating the first Namespacelabel CR")
			firstLabel := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "namespacelabel1",
					Namespace: NamespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{"key1": "value1"},
				},
			}
			Expect(k8sClient.Create(ctx, firstLabel)).To(Succeed())

			By("Attempting to create a second Namespacelabel CR")
			secondLabel := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "namespacelabel2",
					Namespace: NamespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{"key2": "value2"},
				},
			}
			err := k8sClient.Create(ctx, secondLabel)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("only one NamespaceLabel is allowed per namespace"))

			By("Verifying an event for the webhook rejection")
			Eventually(getNextEvent, timeout, interval).Should(ContainSubstring("FailedCreate"))
		})

		It("should allow a single Namespacelabel in the namespace and emit no failure events", func() {
			By("Creating a Namespacelabel CR")
			labelsCR := &labelsv1alpha1.Namespacelabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      NamespaceLabelCR,
					Namespace: NamespaceName,
				},
				Spec: labelsv1alpha1.NamespacelabelSpec{
					Labels: map[string]string{"key3": "value3"},
				},
			}
			Expect(k8sClient.Create(ctx, labelsCR)).To(Succeed())

			By("Verifying no failure events are emitted")
			Consistently(getNextEvent, timeout, interval).ShouldNot(ContainSubstring("FailedCreate"))
		})
	})
})
