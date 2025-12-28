package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds shared configuration values for AzHexGate components
type Config struct {
	// APIBaseURL is the base URL for the Management API
	APIBaseURL string

	// APIKey is the authentication key for the Management API
	APIKey string

	// RelayNamespace is the Azure Relay namespace URL
	RelayNamespace string

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

// GetInt retrieves an environment variable as an integer with a default fallback
func GetInt(key string, defaultValue int) int {
	val := os.Getenv(key)
	if val == "" {
		return defaultValue
	}

	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}

	return intVal
}

// GetBool retrieves an environment variable as a boolean with a default fallback
// Truthy values: "true", "1", "yes", "on" (case-insensitive)
func GetBool(key string, defaultValue bool) bool {
	val := strings.ToLower(os.Getenv(key))
	if val == "" {
		return defaultValue
	}

	switch val {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// getEnvOrDefault retrieves an environment variable or returns a default value
func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
