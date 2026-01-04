package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds shared configuration values for AzHexGate components
type Config struct {
	// APIBaseURL is the base URL for the Management API
	APIBaseURL string

	// APIKey is the authentication key for the Management API
	APIKey string

	// RelayNamespace is the Azure Relay namespace name (e.g., "myrelay")
	RelayNamespace string

	// RelayKeyName is the name of the Relay shared access key
	RelayKeyName string

	// RelayKey is the Relay shared access key value (base64 encoded)
	RelayKey string

	// LogLevel controls logging verbosity (debug, info, warn, error)
	LogLevel string
}

// Load creates a Config by reading from environment variables
// and applying defaults where values are not set
func Load() *Config {
	return &Config{
		APIBaseURL:     getEnvOrDefault("AZHEXGATE_API_URL", ""),
		APIKey:         getEnvOrDefault("AZHEXGATE_API_KEY", ""),
		RelayNamespace: getEnvOrDefault("AZHEXGATE_RELAY_NAMESPACE", ""),
		RelayKeyName:   getEnvOrDefault("AZHEXGATE_RELAY_KEY_NAME", ""),
		RelayKey:       getEnvOrDefault("AZHEXGATE_RELAY_KEY", ""),
		LogLevel:       getEnvOrDefault("AZHEXGATE_LOG_LEVEL", "info"),
	}
}

// Validate checks that required configuration values are present
func (c *Config) Validate() error {
	var missing []string

	if c.APIBaseURL == "" {
		missing = append(missing, "AZHEXGATE_API_URL")
	}
	if c.APIKey == "" {
		missing = append(missing, "AZHEXGATE_API_KEY")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	return nil
}

// getEnvOrDefault retrieves an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
