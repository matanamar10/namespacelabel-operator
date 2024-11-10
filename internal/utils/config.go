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
	protectedLabelsJSON := os.Getenv("PROTECTED_LABELS")
	if protectedLabelsJSON == "" {
		log.Println("PROTECTED_LABELS environment variable is not set")
		return nil, fmt.Errorf("PROTECTED_LABELS environment variable is not set")
	}

	protectedLabels := make(map[string]string)
	if err := json.Unmarshal([]byte(protectedLabelsJSON), &protectedLabels); err != nil {
		log.Printf("Failed to parse PROTECTED_LABELS: %v", err)
		return nil, err
	}

	return protectedLabels, nil
}
