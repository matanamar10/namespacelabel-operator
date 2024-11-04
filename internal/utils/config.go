package utils

import (
	"encoding/json"
	"fmt"
	"github.com/joho/godotenv"
	"log"
	"os"
)

// LoadProtectedLabels loads and parses the PROTECTED_LABELS environment variable from .env
func LoadProtectedLabels() (map[string]string, error) {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Println("Error loading .env file:", err)
		return nil, err
	}

	protectedLabelsJSON := os.Getenv("PROTECTED_LABELS")
	if protectedLabelsJSON == "" {
		log.Println("PROTECTED_LABELS environment variable is not set")
		return nil, fmt.Errorf("PROTECTED_LABELS environment variable is not set")
	}

	protectedLabels := make(map[string]string)
	if err := json.Unmarshal([]byte(protectedLabelsJSON), &protectedLabels); err != nil {
		log.Fatalf("Failed to parse PROTECTED_LABELS: %v", err)
		return nil, err
	}

	return protectedLabels, nil
}
