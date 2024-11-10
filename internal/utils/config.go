package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
)

// LoadProtectedLabels loads a set of "protected" labels from an environment variable.
// These labels should be preserved, as they might be essential for system operation or security policies.
func LoadProtectedLabels() (map[string]string, error) {
	// Fetch the protected labels JSON from the environment variable.
	protectedLabelsJSON := os.Getenv("PROTECTED_LABELS")
	if protectedLabelsJSON == "" {
		log.Println("PROTECTED_LABELS environment variable is not set")
		return nil, fmt.Errorf("PROTECTED_LABELS environment variable is not set")
	}

	// Parse the JSON string into a map to facilitate lookup and use within the reconciler.
	protectedLabels := make(map[string]string)
	if err := json.Unmarshal([]byte(protectedLabelsJSON), &protectedLabels); err != nil {
		log.Printf("Failed to parse PROTECTED_LABELS: %v", err)
		return nil, err
	}

	return protectedLabels, nil
}
