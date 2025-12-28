package logging

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got: %s", tt.expected, result)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"DEBUG", DebugLevel},
		{"Debug", DebugLevel},
		{"info", InfoLevel},
		{"INFO", InfoLevel},
		{"Info", InfoLevel},
		{"warn", WarnLevel},
		{"WARN", WarnLevel},
		{"warning", WarnLevel},
		{"WARNING", WarnLevel},
		{"error", ErrorLevel},
		{"ERROR", ErrorLevel},
		{"Error", ErrorLevel},
		{"invalid", InfoLevel},
		{"", InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ParseLevel(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got: %v", tt.expected, result)
			}
		})
	}
}

func TestNew(t *testing.T) {
	logger := New(InfoLevel)

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	if logger.level != InfoLevel {
		t.Errorf("Expected level InfoLevel, got: %v", logger.level)
	}
}

func TestNewWithOutput(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(DebugLevel, buf)

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	if logger.level != DebugLevel {
		t.Errorf("Expected level DebugLevel, got: %v", logger.level)
	}

	if logger.output != buf {
		t.Error("Expected output to be custom buffer")
	}
}

func TestSetLevel(t *testing.T) {
	logger := New(InfoLevel)
	logger.SetLevel(ErrorLevel)

	if logger.level != ErrorLevel {
		t.Errorf("Expected level ErrorLevel, got: %v", logger.level)
	}
}

func TestLogger_Debug(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(DebugLevel, buf)

	logger.Debug("test debug message")

	output := buf.String()
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("Expected output to contain DEBUG, got: %s", output)
	}
	if !strings.Contains(output, "test debug message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

func TestLogger_Info(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(InfoLevel, buf)

	logger.Info("test info message")

	output := buf.String()
	if !strings.Contains(output, "INFO") {
		t.Errorf("Expected output to contain INFO, got: %s", output)
	}
	if !strings.Contains(output, "test info message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

func TestLogger_Warn(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(WarnLevel, buf)

	logger.Warn("test warn message")

	output := buf.String()
	if !strings.Contains(output, "WARN") {
		t.Errorf("Expected output to contain WARN, got: %s", output)
	}
	if !strings.Contains(output, "test warn message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

func TestLogger_Error(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(ErrorLevel, buf)

	logger.Error("test error message")

	output := buf.String()
	if !strings.Contains(output, "ERROR") {
		t.Errorf("Expected output to contain ERROR, got: %s", output)
	}
	if !strings.Contains(output, "test error message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(WarnLevel, buf)

	// These should be filtered out
	logger.Debug("debug message")
	logger.Info("info message")

	// These should appear
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	if strings.Contains(output, "debug message") {
		t.Error("Debug message should not appear at WARN level")
	}
	if strings.Contains(output, "info message") {
		t.Error("Info message should not appear at WARN level")
	}
	if !strings.Contains(output, "warn message") {
		t.Error("Warn message should appear at WARN level")
	}
	if !strings.Contains(output, "error message") {
		t.Error("Error message should appear at WARN level")
	}
}

func TestLogger_WithFields(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(InfoLevel, buf)

	logger.Info("test message",
		String("key1", "value1"),
		Int("key2", 42),
		Bool("key3", true),
	)

	output := buf.String()

	if !strings.Contains(output, "test message") {
		t.Errorf("Expected output to contain message, got: %s", output)
	}
	if !strings.Contains(output, "key1=value1") {
		t.Errorf("Expected output to contain key1=value1, got: %s", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("Expected output to contain key2=42, got: %s", output)
	}
	if !strings.Contains(output, "key3=true") {
		t.Errorf("Expected output to contain key3=true, got: %s", output)
	}
}

func TestField_String(t *testing.T) {
	field := String("key", "value")

	if field.Key != "key" {
		t.Errorf("Expected key 'key', got: %s", field.Key)
	}
	if field.Value != "value" {
		t.Errorf("Expected value 'value', got: %v", field.Value)
	}
}

func TestField_Int(t *testing.T) {
	field := Int("count", 123)

	if field.Key != "count" {
		t.Errorf("Expected key 'count', got: %s", field.Key)
	}
	if field.Value != 123 {
		t.Errorf("Expected value 123, got: %v", field.Value)
	}
}

func TestField_Bool(t *testing.T) {
	field := Bool("enabled", true)

	if field.Key != "enabled" {
		t.Errorf("Expected key 'enabled', got: %s", field.Key)
	}
	if field.Value != true {
		t.Errorf("Expected value true, got: %v", field.Value)
	}
}

func TestField_Error(t *testing.T) {
	err := errors.New("test error")
	field := Error(err)

	if field.Key != "error" {
		t.Errorf("Expected key 'error', got: %s", field.Key)
	}
	if field.Value != "test error" {
		t.Errorf("Expected value 'test error', got: %v", field.Value)
	}
}

func TestField_Any(t *testing.T) {
	type customStruct struct {
		Name string
		Age  int
	}

	value := customStruct{Name: "test", Age: 30}
	field := Any("custom", value)

	if field.Key != "custom" {
		t.Errorf("Expected key 'custom', got: %s", field.Key)
	}
	if field.Value != value {
		t.Errorf("Expected value %v, got: %v", value, field.Value)
	}
}

func TestLogger_OutputFormat(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewWithOutput(InfoLevel, buf)

	logger.Info("test message", String("key", "value"))

	output := buf.String()

	// Check format: timestamp level message key=value
	parts := strings.Fields(output)
	if len(parts) < 4 {
		t.Errorf("Expected at least 4 parts in output, got: %d", len(parts))
	}

	// Check that timestamp is present (first part should be ISO 8601 format)
	if len(parts[0]) < 10 {
		t.Errorf("Expected timestamp in first part, got: %s", parts[0])
	}

	// Check level
	if parts[1] != "INFO" {
		t.Errorf("Expected level INFO, got: %s", parts[1])
	}

	// Check message
	if parts[2] != "test" {
		t.Errorf("Expected message part 'test', got: %s", parts[2])
	}
}
