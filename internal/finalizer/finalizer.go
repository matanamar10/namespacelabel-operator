package finalizer

import (
	"context"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
)

const FinalizerName = "namespacelabels.finalizers.dana.io"

func EnsureFinalizer(ctx context.Context, c client.Client, obj *labelsv1.Namespacelabel) error {
	if !containsString(obj.GetFinalizers(), FinalizerName) {
		obj.SetFinalizers(append(obj.GetFinalizers(), FinalizerName))
		return c.Update(ctx, obj)
	}
	return nil
}

func CleanupFinalizer(ctx context.Context, c client.Client, obj *labelsv1.Namespacelabel) error {
	if err := cleanupNamespaceLabels(ctx, c, *obj); err != nil {
		return fmt.Errorf("failed to clean up labels: %w", err)
	}

	// Remove the finalizer
	obj.SetFinalizers(removeString(obj.GetFinalizers(), FinalizerName))
	return c.Update(ctx, obj)
}

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

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}
