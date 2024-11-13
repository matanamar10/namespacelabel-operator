package namespacelabels

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	labelsv1 "github.com/matanamar10/namespacelabel-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LoadProtectedLabels loads a set of "protected" labels from an environment variable.
func LoadProtectedLabels(logger logr.Logger) (map[string]string, error) {
	protectedLabelsJSON := os.Getenv("PROTECTED_LABELS")
	if protectedLabelsJSON == "" {
		logger.Info("PROTECTED_LABELS environment variable is not set")
		return nil, fmt.Errorf("PROTECTED_LABELS environment variable is not set")
	}

	protectedLabels := make(map[string]string)
	if err := json.Unmarshal([]byte(protectedLabelsJSON), &protectedLabels); err != nil {
		logger.Error(err, "failed to parse PROTECTED_LABELS")
		return nil, err
	}

	return protectedLabels, nil
}

// Cleanup removes labels from a namespace as specified by the Namespacelabel CR.
// This function is part of the finalizer process to ensure resources are cleaned up.
func Cleanup(ctx context.Context, c client.Client, namespaceLabel labelsv1.Namespacelabel, logger logr.Logger) error {
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
