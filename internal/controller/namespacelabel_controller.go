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
	loadprotectedlabels "github.com/matanamar10/namespacelabel-operator/internal/utility-packages"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NamespacelabelReconciler reconciles a Namespacelabel object
type NamespacelabelReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *NamespacelabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var namespaceLabel labelsv1.Namespacelabel
	if err := r.Get(ctx, req.NamespacedName, &namespaceLabel); err != nil {
		r.Log.Error(err, "failed to get Namespacelabel")
		return ctrl.Result{}, client.IgnoreNotFound(fmt.Errorf("failed to get namespace label: %w", err))
	}

	if result, err := r.handleDeletion(ctx, &namespaceLabel); err != nil {
		r.Log.Error(err, "error during deletion handling")
		return ctrl.Result{}, err
	} else if result != nil {
		return *result, nil
	}

	// Ensure finalizer is added
	if err := r.ensureFinalizer(ctx, &namespaceLabel); err != nil {
		r.Log.Error(err, "failed to ensure finalizer")
		return ctrl.Result{}, err
	}

	// Process and update namespace labels
	updatedLabels, skippedLabels, duplicateLabels, err := r.processLabels(ctx, &namespaceLabel)
	if err != nil {
		r.Log.Error(err, "failed to process labels")
		return ctrl.Result{}, err
	}

	// Update status based on label processing results
	if err := r.updateStatus(ctx, &namespaceLabel, updatedLabels, skippedLabels, duplicateLabels); err != nil {
		r.Log.Error(err, "failed to update Namespacelabel status")
		return ctrl.Result{}, fmt.Errorf("failed to update Namespacelabel status: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *NamespacelabelReconciler) SetCondition(namespaceLabel *labelsv1.Namespacelabel, conditionType string, status metav1.ConditionStatus, reason, message string) {
	r.Log.Info("Setting condition", "type", conditionType, "status", status, "reason", reason)

	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	for i, cond := range namespaceLabel.Status.Conditions {
		if cond.Type == conditionType {
			namespaceLabel.Status.Conditions[i] = condition
			return
		}
	}

	namespaceLabel.Status.Conditions = append(namespaceLabel.Status.Conditions, condition)
}

// CleanupNamespaceLabels removes labels from a namespace as specified by the Namespacelabel CR.
// This function is part of the finalizer process to ensure resources are cleaned up.
func CleanupNamespaceLabels(ctx context.Context, c client.Client, namespaceLabel labelsv1.Namespacelabel, logger logr.Logger) error {
	logger.Info("Starting cleanup of labels from namespace", "namespace", namespaceLabel.Namespace)

	var namespace corev1.Namespace
	if err := c.Get(ctx, client.ObjectKey{Name: namespaceLabel.Namespace}, &namespace); err != nil {
		logger.Error(err, "Failed to retrieve namespace for cleanup")
		return err
	}

	for key := range namespaceLabel.Spec.Labels {
		delete(namespace.Labels, key)
	}

	if err := c.Update(ctx, &namespace); err != nil {
		logger.Error(err, "Failed to update namespace during label cleanup")
		return err
	}

	logger.Info("Successfully cleaned up labels from namespace", "namespace", namespaceLabel.Namespace)
	return nil
}

func (r *NamespacelabelReconciler) handleDeletion(ctx context.Context, namespaceLabel *labelsv1.Namespacelabel) (*ctrl.Result, error) {
	r.Log.Info("Handling deletion for Namespacelabel", "namespace", namespaceLabel.Namespace)

	if namespaceLabel.ObjectMeta.DeletionTimestamp.IsZero() {
		return nil, nil
	}

	if err := finalizer.CleanupFinalizer(ctx, r.Client, namespaceLabel, r.Log); err != nil {
		r.Log.Error(err, "Error during cleanup finalizer")
		return &ctrl.Result{}, fmt.Errorf("failed to clean up labels during finalizer: %w", err)
	}
	return &ctrl.Result{}, nil
}

func (r *NamespacelabelReconciler) processLabels(ctx context.Context, namespaceLabel *labelsv1.Namespacelabel) (map[string]string, map[string]string, map[string]string, error) {
	r.Log.Info("Processing labels for Namespacelabel", "namespace", namespaceLabel.Namespace)

	var namespace corev1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: namespaceLabel.Namespace}, &namespace); err != nil {
		r.Log.Error(err, "Failed to get namespace")
		return nil, nil, nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	protectedLabels, err := loadprotectedlabels.LoadProtectedLabels(r.Log)
	if err != nil {
		r.Log.Error(err, "Failed to load protected labels")
		return nil, nil, nil, fmt.Errorf("failed to load the protected labels list: %w", err)
	}

	updatedLabels := make(map[string]string)
	skippedLabels := make(map[string]string)
	duplicateLabels := make(map[string]string)

	for key, value := range namespaceLabel.Spec.Labels {
		switch {
		case protectedLabels[key] != "":
			r.Log.Info("Skipping protected label", "key", key, "value", value)
			skippedLabels[key] = value
			r.Recorder.Event(namespaceLabel, corev1.EventTypeWarning, "ProtectedLabelSkipped", fmt.Sprintf("Label %s=%s is protected and was not applied", key, value))

		case namespace.Labels[key] != "":
			r.Log.Info("Skipping duplicate label", "key", key, "value", value)
			duplicateLabels[key] = value
			r.Recorder.Event(namespaceLabel, corev1.EventTypeWarning, "DuplicateLabelSkipped", fmt.Sprintf("Label %s=%s already exists with value %s", key, value, namespace.Labels[key]))

		default:
			r.Log.Info("Adding label", "key", key, "value", value)
			updatedLabels[key] = value
		}
	}

	for key, value := range updatedLabels {
		namespace.Labels[key] = value
	}

	if err := r.Update(ctx, &namespace); err != nil {
		r.Log.Error(err, "Failed to update namespace with new labels")
		return nil, nil, nil, fmt.Errorf("failed to update namespace: %w", err)
	}

	return updatedLabels, skippedLabels, duplicateLabels, nil
}

func (r *NamespacelabelReconciler) updateStatus(ctx context.Context, namespaceLabel *labelsv1.Namespacelabel, updatedLabels, skippedLabels, duplicateLabels map[string]string) error {
	namespaceLabel.Status.AppliedLabels = updatedLabels
	namespaceLabel.Status.SkippedLabels = skippedLabels
	namespaceLabel.Status.LastUpdated = metav1.Now()

	if len(skippedLabels) > 0 {
		r.SetCondition(namespaceLabel, "LabelsSkipped", metav1.ConditionTrue, "ProtectedLabelsSkipped", "Some labels were skipped due to being protected.")
	} else {
		r.SetCondition(namespaceLabel, "LabelsSkipped", metav1.ConditionFalse, "NoLabelsSkipped", "No labels were skipped.")
	}

	if len(duplicateLabels) > 0 {
		r.SetCondition(namespaceLabel, "DuplicateLabels", metav1.ConditionTrue, "DuplicateLabelsFound", "Some labels were duplicates and not added.")
	} else {
		r.SetCondition(namespaceLabel, "DuplicateLabels", metav1.ConditionFalse, "NoDuplicateLabels", "No duplicate labels were found.")
	}

	r.SetCondition(namespaceLabel, "LabelsApplied", metav1.ConditionTrue, "LabelsReconciled", "Labels reconciled successfully.")
	return r.Status().Update(ctx, namespaceLabel)
}

func (r *NamespacelabelReconciler) ensureFinalizer(ctx context.Context, namespaceLabel *labelsv1.Namespacelabel) error {
	if err := finalizer.EnsureFinalizer(ctx, r.Client, namespaceLabel, r.Log); err != nil {
		r.Log.Error(err, "Failed to ensure finalizer", "namespaceLabel", namespaceLabel.Name)
		return fmt.Errorf("failed to add finalizer to the namespacelabel: %w", err)
	}
	return nil
}

func (r *NamespacelabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("NamespacelabelController")

	return ctrl.NewControllerManagedBy(mgr).
		For(&labelsv1.Namespacelabel{}).
		Owns(&corev1.Namespace{}).
		Complete(r)
}
