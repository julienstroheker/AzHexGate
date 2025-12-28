package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Save original env vars and restore after test
	originalAPIURL := os.Getenv("AZHEXGATE_API_URL")
	originalAPIKey := os.Getenv("AZHEXGATE_API_KEY")
	originalRelay := os.Getenv("AZHEXGATE_RELAY_NAMESPACE")
	originalLogLevel := os.Getenv("AZHEXGATE_LOG_LEVEL")
	defer func() {
		_ = os.Setenv("AZHEXGATE_API_URL", originalAPIURL)
		_ = os.Setenv("AZHEXGATE_API_KEY", originalAPIKey)
		_ = os.Setenv("AZHEXGATE_RELAY_NAMESPACE", originalRelay)
		_ = os.Setenv("AZHEXGATE_LOG_LEVEL", originalLogLevel)
	}()

	// Clear env vars
	_ = os.Unsetenv("AZHEXGATE_API_URL")
	_ = os.Unsetenv("AZHEXGATE_API_KEY")
	_ = os.Unsetenv("AZHEXGATE_RELAY_NAMESPACE")
	_ = os.Unsetenv("AZHEXGATE_LOG_LEVEL")

	t.Run("defaults", func(t *testing.T) {
		cfg := Load()

		if cfg.APIBaseURL != "" {
			t.Errorf("Expected empty APIBaseURL, got: %s", cfg.APIBaseURL)
		}
		if cfg.APIKey != "" {
			t.Errorf("Expected empty APIKey, got: %s", cfg.APIKey)
		}
		if cfg.RelayNamespace != "" {
			t.Errorf("Expected empty RelayNamespace, got: %s", cfg.RelayNamespace)
		}
		if cfg.LogLevel != "info" {
			t.Errorf("Expected default LogLevel 'info', got: %s", cfg.LogLevel)
		}
	})

	t.Run("from environment", func(t *testing.T) {
		_ = os.Setenv("AZHEXGATE_API_URL", "https://api.example.com")
		_ = os.Setenv("AZHEXGATE_API_KEY", "test-key-123")
		_ = os.Setenv("AZHEXGATE_RELAY_NAMESPACE", "test-relay.servicebus.windows.net")
		_ = os.Setenv("AZHEXGATE_LOG_LEVEL", "debug")

		cfg := Load()

		if cfg.APIBaseURL != "https://api.example.com" {
			t.Errorf("Expected APIBaseURL from env, got: %s", cfg.APIBaseURL)
		}
		if cfg.APIKey != "test-key-123" {
			t.Errorf("Expected APIKey from env, got: %s", cfg.APIKey)
		}
		if cfg.RelayNamespace != "test-relay.servicebus.windows.net" {
			t.Errorf("Expected RelayNamespace from env, got: %s", cfg.RelayNamespace)
		}
		if cfg.LogLevel != "debug" {
			t.Errorf("Expected LogLevel from env, got: %s", cfg.LogLevel)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		cfg := &Config{
			APIBaseURL:     "https://api.example.com",
			APIKey:         "test-key",
			RelayNamespace: "test-relay.servicebus.windows.net",
			LogLevel:       "info",
		}

		err := cfg.Validate()
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
	})

	t.Run("missing API URL", func(t *testing.T) {
		cfg := &Config{
			APIKey:         "test-key",
			RelayNamespace: "test-relay.servicebus.windows.net",
			LogLevel:       "info",
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected error for missing APIBaseURL")
		}
	})

	t.Run("missing API key", func(t *testing.T) {
		cfg := &Config{
			APIBaseURL:     "https://api.example.com",
			RelayNamespace: "test-relay.servicebus.windows.net",
			LogLevel:       "info",
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected error for missing APIKey")
		}
	})

	t.Run("missing both required fields", func(t *testing.T) {
		cfg := &Config{
			RelayNamespace: "test-relay.servicebus.windows.net",
			LogLevel:       "info",
		}

		err := cfg.Validate()
		if err == nil {
			t.Error("Expected error for missing required fields")
		}
	})
}
