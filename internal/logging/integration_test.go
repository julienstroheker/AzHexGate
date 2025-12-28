package logging

import (
	"bytes"
	"strings"
	"testing"
)

// TestClientCanImportLogging verifies that client code can import and use the logging package
func TestClientCanImportLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(InfoLevel, buf)

	logger.Info("test message", String("component", "client"))

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected log output to contain message")
	}
	if !strings.Contains(output, "component=client") {
		t.Errorf("Expected log output to contain field")
	}
}
