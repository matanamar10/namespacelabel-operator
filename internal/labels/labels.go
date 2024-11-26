package labels

import (
	"fmt"

	"encoding/json"
	"os"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
)

// The ProtectedLabelsEnv const is represented the protected labels for all the namespaces in the k8s cluster.
// Those labels keys and values can't be overridden by any namespacelabel object in any namespace.
const ProtectedLabelsEnv = "PROTECTED_LABELS"

// LoadProtected loads a set of "protected" labels from an environment variable.
func LoadProtected(logger logr.Logger) (map[string]string, error) {
	protectedLabelsJSON := os.Getenv(ProtectedLabelsEnv)
	if protectedLabelsJSON == "" {
		return nil, fmt.Errorf("PROTECTED_LABELS environment variable is not set")
	}

	protectedLabels := make(map[string]string)
	if err := json.Unmarshal([]byte(protectedLabelsJSON), &protectedLabels); err != nil {
		logger.Error(err, "failed to parse PROTECTED_LABELS")
	}

	return protectedLabels, nil
}

// Cleanup modifies the namespace's labels based on the given label map.
func Cleanup(namespace *corev1.Namespace, labelsToRemove map[string]string, logger logr.Logger) {
	logger.Info("Starting label cleanup", "namespace", namespace.Name)

	for key := range labelsToRemove {
		logger.Info("Removing label", "key", key)
		delete(namespace.Labels, key)
	}

	logger.Info("Label cleanup completed", "namespace", namespace.Name)
}
