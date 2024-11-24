package finalizer

import (
	"fmt"
	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	corev1 "k8s.io/api/core/v1"

	"context"

	"github.com/matanamar10/namespacelabel-operator/internal/labels"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// FinalizerName is a constant used to define the finalizer added to Namespacelabel CRs.
// This prevents Kubernetes from deleting the CR until the cleanup function completes.
const finalizerName = "namespacelabels.finalizers.dana.io"

// Ensure ensures that the specified finalizer is added to the Namespacelabel CR if it’s missing.
// This makes sure that cleanup operations are triggered before deletion.
func Ensure(ctx context.Context, c client.Client, obj client.Object, logger logr.Logger) error {
	namespaceLabel, ok := obj.(*labelsv1.Namespacelabel)
	if !ok {
		return fmt.Errorf("unexpected type: expected *labelsv1.Namespacelabel, got %T", obj)
	}
	if !controllerutil.ContainsFinalizer(obj, finalizerName) {
		controllerutil.AddFinalizer(obj, finalizerName)
		if err := c.Update(ctx, obj); err != nil {
			return fmt.Errorf("failed to add finalizer: %w", err)
		}
		logger.Info("Finalizer added successfully", "finalizer", finalizerName, "namespaceLabel", namespaceLabel.Name)
	}
	return nil
}

// Cleanup actions, removing labels from the namespace associated with
// the Namespacelabel CR, and then removes the finalizer itself.
// Cleanup performs finalizer actions, cleaning up namespace labels and removing the finalizer.
func Cleanup(ctx context.Context, c client.Client, obj client.Object, logger logr.Logger) error {
	namespaceLabel, ok := obj.(*labelsv1.Namespacelabel)
	if !ok {
		return fmt.Errorf("unexpected type: expected *labelsv1.Namespacelabel, got %T", obj)
	}

	logger.Info("Starting cleanup for Namespacelabel", "namespaceLabel", namespaceLabel.Name)

	var namespace corev1.Namespace
	if err := c.Get(ctx, client.ObjectKey{Name: namespaceLabel.Namespace}, &namespace); err != nil {
		logger.Error(err, "Failed to retrieve namespace for cleanup", "namespaceLabel", namespaceLabel.Name)
		return fmt.Errorf("failed to retrieve namespace: %w", err)
	}

	labels.Cleanup(&namespace, namespaceLabel.Spec.Labels, logger)

	if err := c.Update(ctx, &namespace); err != nil {
		logger.Error(err, "Failed to update namespace after cleanup", "namespaceLabel", namespaceLabel.Name)
		return fmt.Errorf("failed to update namespace: %w", err)
	}

	controllerutil.RemoveFinalizer(obj, finalizerName)
	if err := c.Update(ctx, obj); err != nil {
		logger.Error(err, "Failed to remove finalizer", "namespaceLabel", namespaceLabel.Name)
		return fmt.Errorf("failed to remove finalizer: %w", err)
	}

	logger.Info("Finalizer removed successfully", "finalizer", finalizerName, "namespaceLabel", namespaceLabel.Name)
	return nil
}
