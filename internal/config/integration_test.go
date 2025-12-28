package config

import (
	"os"
	"testing"
)

// TestClientCanImportConfig verifies that client code can import and use the config package
func TestClientCanImportConfig(t *testing.T) {
	// Save and restore
	originalAPIURL := os.Getenv("AZHEXGATE_API_URL")
	originalAPIKey := os.Getenv("AZHEXGATE_API_KEY")
	defer func() {
		_ = os.Setenv("AZHEXGATE_API_URL", originalAPIURL)
		_ = os.Setenv("AZHEXGATE_API_KEY", originalAPIKey)
	}()

	_ = os.Setenv("AZHEXGATE_API_URL", "https://api.example.com")
	_ = os.Setenv("AZHEXGATE_API_KEY", "test-key")

	cfg := Load()
	if cfg.APIBaseURL != "https://api.example.com" {
		t.Errorf("Expected APIBaseURL to be loaded from env")
	}
}
