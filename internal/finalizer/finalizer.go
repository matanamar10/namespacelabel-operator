package finalizer

import (
	"fmt"

	"context"

	cleanup "github.com/matanamar10/namespacelabel-operator/internal/labels"

	"github.com/go-logr/logr"
	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// FinalizerName is a constant used to define the finalizer added to Namespacelabel CRs.
// This prevents Kubernetes from deleting the CR until the cleanup function completes.
const FinalizerName = "namespacelabels.finalizers.dana.io"

// Ensure ensures that the specified finalizer is added to the Namespacelabel CR if it’s missing.
// This makes sure that cleanup operations are triggered before deletion.
func Ensure(ctx context.Context, c client.Client, obj *labelsv1.Namespacelabel, logger logr.Logger) error {
	if !controllerutil.ContainsFinalizer(obj, FinalizerName) {
		controllerutil.AddFinalizer(obj, FinalizerName)
		if err := c.Update(ctx, obj); err != nil {
			return fmt.Errorf("failed to add finalizer: %w", err)
		}
		logger.Info("Finalizer added successfully", "finalizer", FinalizerName, "namespaceLabel", obj.Name)
	}
	return nil
}

// Cleanup actions, removing labels from the namespace associated with
// the Namespacelabel CR, and then removes the finalizer itself.
func Cleanup(ctx context.Context, c client.Client, obj *labelsv1.Namespacelabel, logger logr.Logger) error {
	logger.Info("Starting cleanup for Namespacelabel", "namespaceLabel", obj.Name)

	if err := cleanup.Cleanup(ctx, c, *obj, logger); err != nil {
		logger.Error(err, "Failed to clean up labels for Namespacelabel", "namespaceLabel", obj.Name)
		return fmt.Errorf("failed to clean up labels: %w", err)
	}

	controllerutil.RemoveFinalizer(obj, FinalizerName)
	if err := c.Update(ctx, obj); err != nil {
		return fmt.Errorf("failed to remove finalizer: %w", err)
	}

	logger.Info("Finalizer removed successfully", "finalizer", FinalizerName, "namespaceLabel", obj.Name)
	return nil
}
