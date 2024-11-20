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

package controller

import (
	"context"
	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	if err := os.Unsetenv("PROTECTED_LABELS"); err != nil {
		log.Fatalf("Failed to unset PROTECTED_LABELS environment variable: %v", err)
	}

})

var _ = BeforeSuite(func() {
	By("Initializing test environment")
	scheme = runtime.NewScheme()
	Expect(labelsv1.AddToScheme(scheme)).To(Succeed())
	Expect(corev1.AddToScheme(scheme)).To(Succeed())
	k8sClient = fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx = context.Background()
	if err := os.Setenv("PROTECTED_LABELS", `{"protected-key": "value1", "another-protected-key": "value2"}`); err != nil {
		log.Fatalf("Failed to set PROTECTED_LABELS environment variable: %v", err)
	}
})
