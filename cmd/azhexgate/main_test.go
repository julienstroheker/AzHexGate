package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestStartCommandWithDefaultPort(t *testing.T) {
	// Save original args and stdout
	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
	}()

	// Redirect stdout to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set args for start command without port flag
	os.Args = []string{"azhexgate", "start"}

	// Run in a goroutine to capture output
	done := make(chan bool)
	var output string
	go func() {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r)
		output = buf.String()
		done <- true
	}()

	// Execute startCmd
	startCmd()

	// Close writer and wait for output
	_ = w.Close()
	<-done

	// Verify output contains default port
	if !strings.Contains(output, "Starting tunnel on port 3000") {
		t.Errorf("Expected output to contain 'Starting tunnel on port 3000', got: %s", output)
	}
}

func TestStartCommandWithCustomPort(t *testing.T) {
	// Save original args and stdout
	oldArgs := os.Args
	oldStdout := os.Stdout
	defer func() {
		os.Args = oldArgs
		os.Stdout = oldStdout
	}()

	// Redirect stdout to capture output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set args for start command with custom port
	os.Args = []string{"azhexgate", "start", "--port", "8080"}

	// Run in a goroutine to capture output
	done := make(chan bool)
	var output string
	go func() {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r)
		output = buf.String()
		done <- true
	}()

	// Execute startCmd
	startCmd()

	// Close writer and wait for output
	_ = w.Close()
	<-done

	// Verify output contains custom port
	if !strings.Contains(output, "Starting tunnel on port 8080") {
		t.Errorf("Expected output to contain 'Starting tunnel on port 8080', got: %s", output)
	}
}

func TestUsageText(t *testing.T) {
	// Test that usage text contains expected content
	if !strings.Contains(usageText, "azhexgate - Azure Hybrid Connection reverse tunnel") {
		t.Error("Expected usage text to contain description")
	}

	if !strings.Contains(usageText, "start") {
		t.Error("Expected usage text to contain 'start' command")
	}

	if !strings.Contains(usageText, "--port") {
		t.Error("Expected usage text to contain '--port' flag")
	}
}
