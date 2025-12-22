package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	// Test root command help
	rootCmd.SetArgs([]string{"--help"})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "gateway") {
		t.Errorf("Expected output to contain 'gateway', got: %s", output)
	}

	if !strings.Contains(output, "start") {
		t.Errorf("Expected output to contain 'start' command, got: %s", output)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}

func TestRootCommandNoArgs(t *testing.T) {
	// Test root command with no args (should show help)
	rootCmd.SetArgs([]string{})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Reset for next test
	rootCmd.SetArgs(nil)
}
