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

func TestGetInt(t *testing.T) {
	// Save and restore original env var
	originalVal := os.Getenv("TEST_INT")
	defer func() { _ = os.Setenv("TEST_INT", originalVal) }()

	t.Run("default value when not set", func(t *testing.T) {
		_ = os.Unsetenv("TEST_INT")
		val := GetInt("TEST_INT", 42)
		if val != 42 {
			t.Errorf("Expected default value 42, got: %d", val)
		}
	})

	t.Run("parse valid integer", func(t *testing.T) {
		_ = os.Setenv("TEST_INT", "123")
		val := GetInt("TEST_INT", 42)
		if val != 123 {
			t.Errorf("Expected 123, got: %d", val)
		}
	})

	t.Run("default on invalid integer", func(t *testing.T) {
		_ = os.Setenv("TEST_INT", "not-a-number")
		val := GetInt("TEST_INT", 42)
		if val != 42 {
			t.Errorf("Expected default value 42 on invalid input, got: %d", val)
		}
	})
}

func TestGetBool(t *testing.T) {
	// Save and restore original env var
	originalVal := os.Getenv("TEST_BOOL")
	defer func() { _ = os.Setenv("TEST_BOOL", originalVal) }()

	tests := []struct {
		name         string
		envValue     string
		defaultValue bool
		expected     bool
	}{
		{"default when not set", "", true, true},
		{"default when not set false", "", false, false},
		{"true", "true", false, true},
		{"True", "True", false, true},
		{"1", "1", false, true},
		{"yes", "yes", false, true},
		{"YES", "YES", false, true},
		{"on", "on", false, true},
		{"ON", "ON", false, true},
		{"false", "false", true, false},
		{"False", "False", true, false},
		{"0", "0", true, false},
		{"no", "no", true, false},
		{"NO", "NO", true, false},
		{"off", "off", true, false},
		{"OFF", "OFF", true, false},
		{"invalid", "invalid", true, true},
		{"invalid default false", "invalid", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue == "" {
				_ = os.Unsetenv("TEST_BOOL")
			} else {
				_ = os.Setenv("TEST_BOOL", tt.envValue)
			}

			result := GetBool("TEST_BOOL", tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got: %v", tt.expected, result)
			}
		})
	}
}
