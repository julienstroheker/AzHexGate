package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Format represents the output format for logs
type Format int

const (
	// FormatConsole is human-readable console output
	FormatConsole Format = iota
	// FormatJSON is structured JSON output
	FormatJSON
)

// Level represents a logging level
type Level int

const (
	// DebugLevel is for debug messages
	DebugLevel Level = iota
	// InfoLevel is for informational messages
	InfoLevel
	// WarnLevel is for warning messages
	WarnLevel
	// ErrorLevel is for error messages
	ErrorLevel
)

// String returns the string representation of a Level
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// ParseLevel converts a string to a Level
func ParseLevel(s string) Level {
	switch strings.ToLower(s) {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// Logger provides structured logging capabilities
type Logger struct {
	level  Level
	format Format
	output io.Writer
}

// New creates a new Logger with the specified level and console format
func New(level Level) *Logger {
	return &Logger{
		level:  level,
		format: FormatConsole,
		output: os.Stdout,
	}
}

// NewWithFormat creates a new Logger with the specified level and format
func NewWithFormat(level Level, format Format) *Logger {
	return &Logger{
		level:  level,
		format: format,
		output: os.Stdout,
	}
}

// NewWithOutput creates a new Logger with the specified level and output writer
func NewWithOutput(level Level, output io.Writer) *Logger {
	return &Logger{
		level:  level,
		format: FormatConsole,
		output: output,
	}
}

// SetLevel changes the logging level
func (l *Logger) SetLevel(level Level) {
	l.level = level
}

// Debug logs a debug message with optional fields
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(DebugLevel, msg, fields...)
}

// Info logs an informational message with optional fields
func (l *Logger) Info(msg string, fields ...Field) {
	l.log(InfoLevel, msg, fields...)
}

// Warn logs a warning message with optional fields
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(WarnLevel, msg, fields...)
}

// Error logs an error message with optional fields
func (l *Logger) Error(msg string, fields ...Field) {
	l.log(ErrorLevel, msg, fields...)
}

// log is the internal logging method
func (l *Logger) log(level Level, msg string, fields ...Field) {
	if level < l.level {
		return
	}

	if l.format == FormatJSON {
		l.logJSON(level, msg, fields...)
	} else {
		l.logConsole(level, msg, fields...)
	}
}

// logConsole outputs logs in human-readable console format
func (l *Logger) logConsole(level Level, msg string, fields ...Field) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	levelStr := level.String()

	// Build the log line
	var output strings.Builder
	output.WriteString(timestamp)
	output.WriteString(" ")
	output.WriteString(levelStr)
	output.WriteString(" ")
	output.WriteString(msg)

	// Add fields if present
	if len(fields) > 0 {
		for _, field := range fields {
			output.WriteString(" ")
			output.WriteString(field.Key)
			output.WriteString("=")
			output.WriteString(fmt.Sprintf("%v", field.Value))
		}
	}

	output.WriteString("\n")

	// Write to output
	_, _ = fmt.Fprint(l.output, output.String())
}

// logJSON outputs logs in JSON format
func (l *Logger) logJSON(level Level, msg string, fields ...Field) {
	logEntry := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     level.String(),
		"message":   msg,
	}

	// Add fields if present
	if len(fields) > 0 {
		for _, field := range fields {
			logEntry[field.Key] = field.Value
		}
	}

	jsonBytes, err := json.Marshal(logEntry)
	if err != nil {
		// Fallback to console output if JSON marshaling fails
		l.logConsole(level, msg, fields...)
		return
	}

	_, _ = fmt.Fprintf(l.output, "%s\n", jsonBytes)
}

// Field represents a structured logging field
type Field struct {
	Key   string
	Value any
}

// String creates a Field with a string value
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates a Field with an integer value
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a Field with a boolean value
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error creates a Field with an error value
func Error(err error) Field {
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a Field with any value
func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}
