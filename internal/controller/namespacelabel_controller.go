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
	"fmt"

	"context"

	"github.com/go-logr/logr"
	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	"github.com/matanamar10/namespacelabel-operator/internal/finalizer"
	"github.com/matanamar10/namespacelabel-operator/internal/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NamespacelabelReconciler reconciles a Namespacelabel object
type NamespacelabelReconciler struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func (r *NamespacelabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log.Info("Starting reconciliation", "NamespacedName", req.NamespacedName)
	var namespaceLabel labelsv1.Namespacelabel
	if err := r.Get(ctx, req.NamespacedName, &namespaceLabel); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(fmt.Errorf("failed to get namespace label: %w", err))
	}

	if !namespaceLabel.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.handleDeletion(ctx, &namespaceLabel); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to handle deletion: %w", err)
		}
		return ctrl.Result{}, nil
	}

	if err := r.ensureFinalizer(ctx, &namespaceLabel); err != nil {
		return ctrl.Result{}, err
	}

	protectedLabels, err := labels.LoadProtected(r.Log)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to load the protected labels list: %w", err)
	}

	namespace, err := r.fetchNamespace(ctx, namespaceLabel.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	updatedLabels, skippedLabels, duplicateLabels := r.processLabels(namespace, &namespaceLabel, protectedLabels)

	for key, value := range updatedLabels {
		namespace.Labels[key] = value
	}

	if err := r.updateNamespace(ctx, namespace); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.updateStatus(ctx, &namespaceLabel, updatedLabels, skippedLabels, duplicateLabels); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to update Namespacelabel status: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *NamespacelabelReconciler) setCondition(namespaceLabel *labelsv1.Namespacelabel, conditionType string, status metav1.ConditionStatus, reason, message string) {
	r.Log.Info("Setting condition", "type", conditionType, "status", status, "reason", reason)

	condition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}

	meta.SetStatusCondition(&namespaceLabel.Status.Conditions, condition)
}

func (r *NamespacelabelReconciler) handleDeletion(ctx context.Context, namespaceLabel *labelsv1.Namespacelabel) error {
	r.Log.Info("Handling deletion for Namespacelabel", "namespace", namespaceLabel.Namespace)
	if err := finalizer.Cleanup(ctx, r.Client, namespaceLabel, r.Log); err != nil {
		return fmt.Errorf("failed to clean up labels during finalizer: %w", err)
	}
	return nil
}

func (r *NamespacelabelReconciler) fetchNamespace(ctx context.Context, namespaceName string) (*corev1.Namespace, error) {
	var namespace corev1.Namespace
	if err := r.Get(ctx, types.NamespacedName{Name: namespaceName}, &namespace); err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}
	return &namespace, nil
}

func (r *NamespacelabelReconciler) updateNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	if err := r.Update(ctx, namespace); err != nil {
		return fmt.Errorf("failed to update namespace: %w", err)
	} else {
		return nil
	}
}

func (r *NamespacelabelReconciler) processLabels(namespace *corev1.Namespace, namespaceLabel *labelsv1.Namespacelabel, protectedLabels map[string]string) (updatedLabels map[string]string, skippedLabels map[string]string, duplicateLabels map[string]string) {
	r.Log.Info("Processing labels for Namespacelabel", "namespace", namespaceLabel.Namespace)

	updatedLabels = make(map[string]string)
	skippedLabels = make(map[string]string)
	duplicateLabels = make(map[string]string)

	if namespace.Labels == nil {
		namespace.Labels = make(map[string]string)
	}

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
	return updatedLabels, skippedLabels, duplicateLabels
}

func (r *NamespacelabelReconciler) updateStatus(ctx context.Context, namespaceLabel *labelsv1.Namespacelabel, updatedLabels, skippedLabels, duplicateLabels map[string]string) error {
	namespaceLabel.Status.AppliedLabels = updatedLabels
	namespaceLabel.Status.SkippedLabels = skippedLabels

	if len(skippedLabels) > 0 {
		r.setCondition(namespaceLabel, "LabelsSkipped", metav1.ConditionTrue, "ProtectedLabelsHandled", "Some labels were skipped because they are protected.")
	} else {
		r.setCondition(namespaceLabel, "LabelsSkipped", metav1.ConditionFalse, "ProtectedLabelsHandled", "All labels were applied successfully; no protected labels were skipped.")
	}

	if len(duplicateLabels) > 0 {
		r.setCondition(namespaceLabel, "DuplicateLabels", metav1.ConditionTrue, "DuplicateLabelsHandled", "Some labels were not applied because they are duplicates.")
	} else {
		r.setCondition(namespaceLabel, "DuplicateLabels", metav1.ConditionFalse, "DuplicateLabelsHandled", "All labels were unique and applied successfully.")
	}

	r.setCondition(namespaceLabel, "LabelsApplied", metav1.ConditionTrue, "LabelsReconciled", "Labels reconciled successfully.")

	if err := r.Status().Update(ctx, namespaceLabel); err != nil {
		if client.IgnoreNotFound(err) == nil {
			r.Log.Info("Resource already deleted, skipping status update", "namespaceLabel", namespaceLabel.Name)
			return nil
		}
		return fmt.Errorf("failed to update Namespacelabel status: %w", err)
	}

	return nil
}

func (r *NamespacelabelReconciler) ensureFinalizer(ctx context.Context, namespaceLabel *labelsv1.Namespacelabel) error {
	if err := finalizer.Ensure(ctx, r.Client, namespaceLabel, r.Log); err != nil {
		return err
	}
	return nil
}

func (r *NamespacelabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&labelsv1.Namespacelabel{}).
		Watches(&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueRequestsFromNamespace),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

// enqueueRequestsFromNamespace triggers reconciliation for related Namespacelabel resources when a Namespace changes.
// enqueueRequestsFromNamespace reconciles the Namespacelabel when the associated Namespace changes.
func (r *NamespacelabelReconciler) enqueueRequestsFromNamespace(ctx context.Context, namespace client.Object) []reconcile.Request {
	ns, ok := namespace.(*corev1.Namespace)
	if !ok {
		r.Log.Error(nil, "Failed to cast object to Namespace", "object", namespace)
		return []reconcile.Request{}
	}

	namespaceLabelList := &labelsv1.NamespacelabelList{}
	listOps := &client.ListOptions{
		Namespace: ns.Name,
	}
	if err := r.List(ctx, namespaceLabelList, listOps); err != nil {
		r.Log.Error(err, "Failed to list Namespacelabel resources", "Namespace", ns.Name)
		return []reconcile.Request{}
	}

	requests := make([]reconcile.Request, 0, len(namespaceLabelList.Items))
	for _, item := range namespaceLabelList.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      item.Name,
				Namespace: item.Namespace,
			},
		})
	}

	r.Log.Info("Enqueued reconciliation requests", "Namespace", ns.Name, "RequestCount", len(requests))
	return requests
}
