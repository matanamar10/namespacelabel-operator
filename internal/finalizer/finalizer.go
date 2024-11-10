package finalizer

import (
	"context"
	"fmt"

	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FinalizerName is a constant used to define the finalizer added to Namespacelabel CRs.
// This prevents Kubernetes from deleting the CR until the cleanup function completes.
const FinalizerName = "namespacelabels.finalizers.dana.io"

// EnsureFinalizer ensures that the specified finalizer is added to the Namespacelabel CR if it’s missing.
// This makes sure that cleanup operations are triggered before deletion.
func EnsureFinalizer(ctx context.Context, c client.Client, obj *labelsv1.Namespacelabel) error {
	// Check if the finalizer is already set; if not, add it.
	if !containsString(obj.GetFinalizers(), FinalizerName) {
		obj.SetFinalizers(append(obj.GetFinalizers(), FinalizerName))
		return c.Update(ctx, obj)
	}
	return nil
}

// CleanupFinalizer performs cleanup actions, removing labels from the namespace associated with
// the Namespacelabel CR, and then removes the finalizer itself.
func CleanupFinalizer(ctx context.Context, c client.Client, obj *labelsv1.Namespacelabel) error {
	if err := cleanupNamespaceLabels(ctx, c, *obj); err != nil {
		return fmt.Errorf("failed to clean up labels: %w", err)
	}

	// Remove the finalizer from the CR, allowing Kubernetes to delete it.
	obj.SetFinalizers(removeString(obj.GetFinalizers(), FinalizerName))
	return c.Update(ctx, obj)
}

// cleanupNamespaceLabels removes labels from a namespace as specified by the Namespacelabel CR.
// This function is part of the finalizer process to ensure resources are cleaned up.
func cleanupNamespaceLabels(ctx context.Context, c client.Client, namespaceLabel labelsv1.Namespacelabel) error {
	var namespace corev1.Namespace
	if err := c.Get(ctx, client.ObjectKey{Name: namespaceLabel.Namespace}, &namespace); err != nil {
		return err
	}

	for key := range namespaceLabel.Spec.Labels {
		delete(namespace.Labels, key)
	}

	return c.Update(ctx, &namespace)
}

// containsString checks if a given string exists in a slice of strings.
// Used to verify if the finalizer already exists.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// removeString removes a specified string from a slice of strings.
// Used to remove the finalizer from the list in a Namespacelabel CR.
func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
