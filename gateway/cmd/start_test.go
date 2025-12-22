package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartCommandHelp(t *testing.T) {
	// Test start command help
	rootCmd.SetArgs([]string{"start", "--help"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "start") {
		t.Errorf("Expected output to contain 'start', got: %s", output)
	}

	if !strings.Contains(output, "--port") {
		t.Errorf("Expected output to contain '--port' flag, got: %s", output)
	}

	if !strings.Contains(output, "--shutdown-timeout") {
		t.Errorf("Expected output to contain '--shutdown-timeout' flag, got: %s", output)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestStartCommandDefaultValues(t *testing.T) {
	// Reset flags to defaults
	portFlag = defaultPort
	shutdownTimeoutFlag = defaultShutdownTimeout

	// Verify default values
	if portFlag != 8080 {
		t.Errorf("Expected default port to be 8080, got: %d", portFlag)
	}

	if shutdownTimeoutFlag != 30 {
		t.Errorf("Expected default shutdown timeout to be 30, got: %d", shutdownTimeoutFlag)
	}
}
