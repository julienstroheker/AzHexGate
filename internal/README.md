# Internal Packages

This directory contains shared packages used by both the CLI and Gateway components of AzHexGate.

## Packages

### config

The `config` package provides configuration loading from environment variables.

**Usage:**

```go
import "github.com/julienstroheker/AzHexGate/internal/config"

// Load configuration from environment variables
cfg := config.Load()

// Validate that required fields are set
if err := cfg.Validate(); err != nil {
    log.Fatal(err)
}

// Access configuration values
apiURL := cfg.APIBaseURL
apiKey := cfg.APIKey
relayNamespace := cfg.RelayNamespace
logLevel := cfg.LogLevel

// Use helper functions for typed environment variables
port := config.GetInt("PORT", 8080)
enableDebug := config.GetBool("DEBUG", false)
```

**Environment Variables:**

- `AZHEXGATE_API_URL` - Base URL for the Management API (required)
- `AZHEXGATE_API_KEY` - Authentication key for the Management API (required)
- `AZHEXGATE_RELAY_NAMESPACE` - Azure Relay namespace URL (optional)
- `AZHEXGATE_LOG_LEVEL` - Logging verbosity: debug, info, warn, error (default: info)

### logging

The `logging` package provides structured logging capabilities.

**Usage:**

```go
import "github.com/julienstroheker/AzHexGate/internal/logging"

// Create a logger with a specific level
logger := logging.New(logging.InfoLevel)

// Or parse level from string
level := logging.ParseLevel("debug")
logger := logging.New(level)

// Log messages
logger.Debug("Debug message")
logger.Info("Info message")
logger.Warn("Warning message")
logger.Error("Error message")

// Log with structured fields
logger.Info("Request processed",
    logging.String("path", "/api/tunnels"),
    logging.Int("status", 200),
    logging.Bool("success", true),
)

// Log errors
if err != nil {
    logger.Error("Operation failed", logging.Error(err))
}

// Change log level dynamically
logger.SetLevel(logging.ErrorLevel)
```

**Log Levels:**

- `DebugLevel` - Detailed debugging information
- `InfoLevel` - General informational messages (default)
- `WarnLevel` - Warning messages
- `ErrorLevel` - Error messages

**Field Types:**

- `logging.String(key, value)` - String field
- `logging.Int(key, value)` - Integer field
- `logging.Bool(key, value)` - Boolean field
- `logging.Error(err)` - Error field (key is "error")
- `logging.Any(key, value)` - Any value field

## Design Principles

These packages follow the AzHexGate architecture principles:

1. **Simple and explicit** - No clever abstractions, straightforward APIs
2. **No external dependencies** - Only Go standard library
3. **No Azure dependencies** - Generic utilities, not Azure-specific
4. **No business logic** - Pure utility functions
5. **Well-tested** - Comprehensive unit tests with edge case coverage

## Testing

Run tests for internal packages:

```bash
go test ./internal/...
```

Run tests with coverage:

```bash
go test ./internal/... -cover
```
