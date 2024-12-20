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
	"fmt"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	labelsv1alpha1 "github.com/matanamar10/namespacelabel-operator/api/v1alpha1"
)

// nolint:unused
// log is for logging in this package.
var namespacelabellog = logf.Log.WithName("namespacelabel-resource")

// SetupNamespacelabelWebhookWithManager registers the webhook for Namespacelabel in the manager.
func SetupNamespacelabelWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&labelsv1alpha1.Namespacelabel{}).
		WithValidator(&NamespacelabelCustomValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-labels-dana-io-v1alpha1-namespacelabel,mutating=false,failurePolicy=fail,sideEffects=None,groups=labels.dana.io,resources=namespacelabels,verbs=create;update,versions=v1alpha1,name=vnamespacelabel-v1alpha1.kb.io,admissionReviewVersions=v1

// NamespacelabelCustomValidator struct is responsible for validating the Namespacelabel resource
// when it is created, updated, or deleted.
type NamespacelabelCustomValidator struct {
	Client   client.Client
	decoder  *admission.Decoder
	Logger   logr.Logger
	Recorder record.EventRecorder
}

var _ webhook.CustomValidator = &NamespacelabelCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Namespacelabel.
func (v *NamespacelabelCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	namespaceLabel, ok := obj.(*labelsv1alpha1.Namespacelabel)
	if !ok {
		return nil, fmt.Errorf("unexpected object type: %T", obj)
	}

	existingnamespaceLabels := &labelsv1alpha1.NamespacelabelList{}
	if err := v.Client.List(ctx, existingnamespaceLabels, client.InNamespace(namespaceLabel.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list NamespaceLabels: %v", err)
	}

	if len(existingnamespaceLabels.Items) > 0 {
		v.Recorder.Eventf(namespaceLabel, corev1.EventTypeWarning, "FailedCreate",
			"only one NamespaceLabel is allowed per namespace; found %d existing", len(existingnamespaceLabels.Items))

		return nil, fmt.Errorf("only one NamespaceLabel is allowed per namespace; found %d existing", len(existingnamespaceLabels.Items))
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Namespacelabel.
func (v *NamespacelabelCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	namespacelabel, ok := newObj.(*labelsv1alpha1.Namespacelabel)
	if !ok {
		return nil, fmt.Errorf("expected a Namespacelabel object for the newObj but got %T", newObj)
	}
	namespacelabellog.Info("Validation for Namespacelabel upon update", "name", namespacelabel.GetName())
	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Namespacelabel.
func (v *NamespacelabelCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	namespacelabel, ok := obj.(*labelsv1alpha1.Namespacelabel)
	if !ok {
		return nil, fmt.Errorf("expected a Namespacelabel object but got %T", obj)
	}
	namespacelabellog.Info("Validation for Namespacelabel upon deletion", "name", namespacelabel.GetName())
	return nil, nil
}
