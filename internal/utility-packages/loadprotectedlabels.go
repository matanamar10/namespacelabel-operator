package loadprotectedlabels

import (
	"encoding/json"
	"fmt"
	"github.com/go-logr/logr"
	"os"
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
