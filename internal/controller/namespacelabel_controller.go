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

// NamespacelabelReconciler reconciles a Namespacelabel object
type NamespacelabelReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// RBAC permissions required by the reconciler to manage Namespacelabels and namespaces.
// +kubebuilder:rbac:groups=labels.dana.io,resources=namespacelabels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=labels.dana.io,resources=namespacelabels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=labels.dana.io,resources=namespacelabels/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile reconciles a Namespacelabel object and manages finalizers and namespace labels.
func (r *NamespacelabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Obtain a logger for this context
	_ = log.FromContext(ctx)

	// Fetch the Namespacelabel instance by its name in the specified namespace
	var namespaceLabel labelsv1.Namespacelabel
	if err := r.Get(ctx, req.NamespacedName, &namespaceLabel); err != nil {
		// Ignore the error if the resource was not found
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if deletion timestamp is set, indicating a deletion request
	if !namespaceLabel.ObjectMeta.DeletionTimestamp.IsZero() {
		// Clean up resources before deletion
		if err := finalizer.CleanupFinalizer(ctx, r.Client, &namespaceLabel); err != nil {
			r.Log.Error(err, "Failed to clean up labels during finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Add a finalizer if it is not already present to handle future deletion events
	if err := finalizer.EnsureFinalizer(ctx, r.Client, &namespaceLabel); err != nil {
		r.Log.Error(err, "Failed to add finalizer")
		return ctrl.Result{}, err
	}

	// Fetch the namespace associated with the Namespacelabel resource
	var namespace corev1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: req.Namespace}, &namespace); err != nil {
		return ctrl.Result{}, err
	}

	// Load protected labels that should not be modified
	protectedLabels, err := utils.LoadProtectedLabels()
	if err != nil {
		r.Log.Error(err, "Failed to load protected labels")
		return ctrl.Result{}, err
	}

	// Initialize maps for tracking labels to update, skip, or report as duplicates
	updatedLabels := make(map[string]string)
	skippedLabels := make(map[string]string)
	duplicateLabels := make(map[string]string)

	// Process each label in the Namespacelabel spec
	for key, value := range namespaceLabel.Spec.Labels {
		// Skip if the label is protected
		if _, exists := protectedLabels[key]; exists {
			r.Log.Info("Skipping protected label", "key", key, "value", value)
			skippedLabels[key] = value
			r.Recorder.Event(&namespaceLabel, corev1.EventTypeWarning, "ProtectedLabelSkipped",
				fmt.Sprintf("Label %s=%s is protected and was not applied", key, value))
		} else if existingValue, exists := namespace.Labels[key]; exists {
			// Skip and log if the label is already applied (duplicate)
			r.Log.Info("Skipping duplicate label", "key", key, "value", value, "existingValue", existingValue)
			duplicateLabels[key] = value
			r.Recorder.Event(&namespaceLabel, corev1.EventTypeWarning, "DuplicateLabelSkipped",
				fmt.Sprintf("Label %s=%s was not applied because it already exists with value %s", key, value, existingValue))
		} else {
			// Otherwise, mark the label for updating in the namespace
			updatedLabels[key] = value
		}
	}

	// Apply updated labels to the namespace
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

	// Update the Namespacelabel status to reflect which labels were applied, skipped, or duplicated
	namespaceLabel.Status.AppliedLabels = updatedLabels
	namespaceLabel.Status.SkippedLabels = skippedLabels
	namespaceLabel.Status.LastUpdated = metav1.Now()
	namespaceLabel.Status.Message = "Labels reconciled with skipped and duplicate protected labels"
	if len(duplicateLabels) > 0 {
		namespaceLabel.Status.Message += "; some labels were duplicates and not added."
	}

	// Commit the status update to the Namespacelabel resource
	if err := r.Status().Update(ctx, &namespaceLabel); err != nil {
		r.Log.Error(err, "Failed to update Namespacelabel status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the specified manager and starts it.
func (r *NamespacelabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Obtain an event recorder for this controller
	r.Recorder = mgr.GetEventRecorderFor("NamespacelabelController")

	// Set up a new controller managed by the provided manager
	return ctrl.NewControllerManagedBy(mgr).
		// Watch for changes in Namespacelabel resources
		For(&labelsv1.Namespacelabel{}).
		// Watch for changes in namespaces owned by Namespacelabel resources
		Owns(&corev1.Namespace{}).
		Complete(r)
}
