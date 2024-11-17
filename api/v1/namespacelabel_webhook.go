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

package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var namespacelabellog = logf.Log.WithName("namespacelabel-resource")

// webhookClient is used to query Kubernetes resources in the webhook.
var webhookClient client.Client

// SetupWebhookWithManager will set up the webhook with the manager.
func (r *Namespacelabel) SetupWebhookWithManager(mgr ctrl.Manager) error {
	webhookClient = mgr.GetClient()
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// +kubebuilder:webhook:path=/validate-labels-dana-io-v1-namespacelabel,mutating=false,failurePolicy=fail,sideEffects=None,groups=labels.dana.io,resources=namespacelabels,verbs=create;update,versions=v1,name=vnamespacelabel.kb.io,admissionReviewVersions=v1

// Ensure the Namespacelabel implements webhook.Validator.
var _ webhook.Validator = &Namespacelabel{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *Namespacelabel) ValidateCreate() (admission.Warnings, error) {
	namespacelabellog.Info("validate create", "name", r.Name)

	namespaceLabelList := &NamespacelabelList{}
	err := webhookClient.List(context.TODO(), namespaceLabelList, client.InNamespace(r.Namespace))
	if err != nil {
		namespacelabellog.Error(err, "Failed to list NamespaceLabel resources")
		return nil, fmt.Errorf("internal error: unable to validate NamespaceLabel creation")
	}

	if len(namespaceLabelList.Items) > 0 {
		return nil, fmt.Errorf("only one NamespaceLabel resource is allowed per namespace")
	}

	return nil, nil
}

func (r *Namespacelabel) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	namespacelabellog.Info("validate update", "name", r.Name)

	return nil, nil
}

func (r *Namespacelabel) ValidateDelete() (admission.Warnings, error) {
	namespacelabellog.Info("validate delete", "name", r.Name)

	return nil, nil
}
