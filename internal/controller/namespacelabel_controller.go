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
	"fmt"
	"github.com/go-logr/logr"
	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	"github.com/matanamar10/namespacelabel-operator/internal/finalizer"
	"github.com/matanamar10/namespacelabel-operator/internal/utils"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NamespacelabelReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=labels.dana.io,resources=namespacelabels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=labels.dana.io,resources=namespacelabels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=labels.dana.io,resources=namespacelabels/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Namespacelabel object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.0/pkg/reconcile

// Reconcile function with finalizer logic integrated

func (r *NamespacelabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)
	var namespaceLabel labelsv1.Namespacelabel
	if err := r.Get(ctx, req.NamespacedName, &namespaceLabel); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !namespaceLabel.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := finalizer.CleanupFinalizer(ctx, r.Client, &namespaceLabel); err != nil {
			r.Log.Error(err, "Failed to clean up labels during finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := finalizer.EnsureFinalizer(ctx, r.Client, &namespaceLabel); err != nil {
		r.Log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}

	var namespace corev1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: req.Namespace}, &namespace); err != nil {
		return ctrl.Result{}, err
	}

	protectedLabels, err := utils.LoadProtectedLabels()
	if err != nil {
		r.Log.Error(err, "Failed to load protected labels")
		return ctrl.Result{}, err
	}

	updatedLabels := make(map[string]string)
	skippedLabels := make(map[string]string)
	duplicateLabels := make(map[string]string)

	for key, value := range namespaceLabel.Spec.Labels {
		if _, exists := protectedLabels[key]; exists {
			r.Log.Info("Skipping protected label", "key", key, "value", value)
			skippedLabels[key] = value
			r.Recorder.Event(&namespaceLabel, corev1.EventTypeWarning, "ProtectedLabelSkipped",
				fmt.Sprintf("Label %s=%s is protected and was not applied", key, value))
		} else if existingValue, exists := namespace.Labels[key]; exists {
			r.Log.Info("Skipping duplicate label", "key", key, "value", value, "existingValue", existingValue)
			duplicateLabels[key] = value
			r.Recorder.Event(&namespaceLabel, corev1.EventTypeWarning, "DuplicateLabelSkipped",
				fmt.Sprintf("Label %s=%s was not applied because it already exists with value %s", key, value, existingValue))
		} else {
			updatedLabels[key] = value
		}
	}

	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}
	for key, value := range updatedLabels {
		namespace.Labels[key] = value
	}

	// Update the namespace with the new labels
	if err := r.Update(ctx, &namespace); err != nil {
		return ctrl.Result{}, err
	}

	// Update the NamespaceLabel status to reflect applied, skipped, and duplicate labels
	namespaceLabel.Status.AppliedLabels = updatedLabels
	namespaceLabel.Status.SkippedLabels = skippedLabels
	namespaceLabel.Status.LastUpdated = metav1.Now()
	namespaceLabel.Status.Message = "Labels reconciled with skipped and duplicate protected labels"
	if len(duplicateLabels) > 0 {
		namespaceLabel.Status.Message += "; some labels were duplicates and not added."
	}

	if err := r.Status().Update(ctx, &namespaceLabel); err != nil {
		r.Log.Error(err, "Failed to update NamespaceLabel status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *NamespacelabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("NamespacelabelController")

	return ctrl.NewControllerManagedBy(mgr).
		For(&labelsv1.Namespacelabel{}).
		Owns(&corev1.Namespace{}).
		Complete(r)
}
