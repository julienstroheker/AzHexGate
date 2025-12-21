package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestStartCommand(t *testing.T) {
	// Test start command with default port
	rootCmd.SetArgs([]string{"start"})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Starting tunnel on port 3000") {
		t.Errorf("Expected output to contain 'Starting tunnel on port 3000', got: %s", output)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestStartCommandWithCustomPort(t *testing.T) {
	// Test start command with custom port
	rootCmd.SetArgs([]string{"start", "--port", "8080"})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Starting tunnel on port 8080") {
		t.Errorf("Expected output to contain 'Starting tunnel on port 8080', got: %s", output)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestStartCommandWithShortFlag(t *testing.T) {
	// Test start command with short flag
	rootCmd.SetArgs([]string{"start", "-p", "5000"})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Starting tunnel on port 5000") {
		t.Errorf("Expected output to contain 'Starting tunnel on port 5000', got: %s", output)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestRootCommandHelp(t *testing.T) {
	// Test help command
	rootCmd.SetArgs([]string{"--help"})

	// Capture output
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "azhexgate") {
		t.Errorf("Expected output to contain 'azhexgate', got: %s", output)
	}

	if !strings.Contains(output, "start") {
		t.Errorf("Expected output to contain 'start' command, got: %s", output)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}
