# Local Mode Implementation Guide

## Overview

This document describes the implementation of local mode for AzHexGate, which allows the gateway and client to run on the same machine using an in-memory relay instead of Azure Relay. This enables local development and testing without requiring Azure infrastructure.

## Architecture Decision

### Mode Enum Pattern

Instead of using a boolean flag (`--local`), we use an enum-based mode pattern:

- **`ModeLocal`**: Uses in-memory mock relay (no Azure required)
- **`ModeRemote`**: Uses Azure Relay (production mode)

This design is more extensible and allows for future modes (e.g., `ModeHybrid`, `ModeDev`) without breaking changes.

### Package Structure

Both client and gateway delegate business logic to dedicated packages, keeping the CLI layer lean:

#### Gateway Structure
```
gateway/
├── cmd/
│   └── start.go          # CLI layer - flags and orchestration only
├── tunnel/               # NEW - Business logic package
│   └── manager.go        # Tunnel management (local + remote mode)
├── http/
│   ├── server.go         # HTTP server setup
│   └── handlers/
│       ├── health.go
│       └── tunnels.go    # HTTP handler - delegates to manager
└── main.go
```

#### Client Structure
```
client/
├── cmd/
│   └── start.go          # CLI layer - flags and orchestration only
├── gateway/
│   └── client.go         # Gateway API client (local + remote mode)
└── main.go
```

## Implementation Details

### 1. Mode Type Definition

**File**: `internal/config/mode.go` (NEW)

```go
package config

// Mode represents the operational mode of the application
type Mode string

const (
    // ModeLocal runs with in-memory relay (no Azure required)
    ModeLocal Mode = "local"
    
    // ModeRemote runs with Azure Relay
    ModeRemote Mode = "remote"
)

// IsValid checks if the mode is valid
func (m Mode) IsValid() bool {
    return m == ModeLocal || m == ModeRemote
}

// String returns the string representation
func (m Mode) String() string {
    return string(m)
}
```

### 2. Gateway Implementation

#### 2.1 Tunnel Manager

**File**: `gateway/tunnel/manager.go` (NEW)

The tunnel manager encapsulates all tunnel creation and management logic:

```go
package tunnel

type Manager struct {
    mode      config.Mode
    logger    *logging.Logger
    
    // Local mode: in-memory registry
    mu        sync.RWMutex
    listeners map[string]relay.Listener
    
    // Remote mode: Azure Relay credentials
    // TODO: Add Azure Relay client fields
}

type Options struct {
    Mode   config.Mode
    Logger *logging.Logger
}

func NewManager(opts *Options) *Manager
func (m *Manager) CreateTunnel(ctx context.Context, localPort int) (*api.TunnelResponse, error)
func (m *Manager) GetListener(hcName string) (relay.Listener, error)
func (m *Manager) GetSender(hcName string) (relay.Sender, error)
```

**Key Methods**:

- `CreateTunnel()`: Routes to `createLocalTunnel()` or `createRemoteTunnel()` based on mode
- `GetListener()`: Returns the mock listener for a given hybrid connection (local mode only)
- `GetSender()`: Creates a mock sender for proxying requests (local mode only)

**Local Mode Behavior**:
- Creates `MockListener` instances
- Stores them in an in-memory map (`listeners`)
- Returns tunnel metadata with local URLs

**Remote Mode Behavior**:
- TODO: Connect to Azure Relay
- TODO: Obtain real listener tokens
- Returns production tunnel metadata

#### 2.2 HTTP Server Updates

**File**: `gateway/http/server.go` (UPDATED)

Add manager as dependency:

```go
type Server struct {
    server  *http.Server
    port    int
    manager *tunnel.Manager  // NEW
}

type Options struct {
    Port    int
    Manager *tunnel.Manager  // NEW
}

func NewServer(opts *Options) *Server {
    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", handlers.HealthHandler)
    mux.HandleFunc("/api/tunnels", handlers.NewTunnelsHandler(opts.Manager))
    // ...
}
```

#### 2.3 Tunnels Handler Updates

**File**: `gateway/http/handlers/tunnels.go` (UPDATED)

Change from function to factory pattern:

```go
// Before: func TunnelsHandler(w http.ResponseWriter, r *http.Request)
// After:
func NewTunnelsHandler(manager *tunnel.Manager) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodPost {
            w.WriteHeader(http.StatusMethodNotAllowed)
            return
        }

        // TODO: Parse localPort from request body
        response, err := manager.CreateTunnel(r.Context(), 3000)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        data, err := json.Marshal(response)
        if err != nil {
            w.WriteHeader(http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write(data)
    }
}
```

#### 2.4 Gateway CLI Updates

**File**: `gateway/cmd/start.go` (UPDATED)

Add mode flag and wire up manager:

```go
var (
    portFlag            int
    shutdownTimeoutFlag int
    modeFlag            string  // NEW
)

func init() {
    rootCmd.AddCommand(startCmd)
    startCmd.Flags().IntVarP(&portFlag, "port", "p", defaultPort, "Port to listen on")
    startCmd.Flags().IntVar(&shutdownTimeoutFlag, "shutdown-timeout", defaultShutdownTimeout,
        "Graceful shutdown timeout in seconds")
    startCmd.Flags().StringVar(&modeFlag, "mode", string(config.ModeRemote), 
        "Operation mode: local or remote")
}

func runServer() error {
    log := GetLogger()
    
    // Parse and validate mode
    mode := config.Mode(modeFlag)
    if !mode.IsValid() {
        return fmt.Errorf("invalid mode: %s (must be 'local' or 'remote')", modeFlag)
    }
    
    log.Info("Starting gateway server", 
        logging.Int("port", portFlag),
        logging.String("mode", mode.String()))

    // Create tunnel manager with mode
    manager := tunnel.NewManager(&tunnel.Options{
        Mode:   mode,
        Logger: log,
    })

    // Create server with manager
    server := http.NewServer(&http.Options{
        Port:    portFlag,
        Manager: manager,
    })

    // ... rest of signal handling unchanged
}
```

### 3. Client Implementation

#### 3.1 Gateway Client Updates

**File**: `client/gateway/client.go` (UPDATED)

Add mode support to the client:

```go
type Options struct {
    BaseURL    string
    Timeout    time.Duration
    MaxRetries int
    Logger     *logging.Logger
    Mode       config.Mode  // NEW
}

type Client struct {
    baseURL    string
    httpClient *httpclient.Client
    logger     *logging.Logger
    mode       config.Mode   // NEW
    listener   relay.Listener // NEW - for local mode
}

func NewClient(opts *Options) *Client {
    if opts == nil {
        opts = &Options{
            Mode: config.ModeRemote, // Default to remote
        }
    }
    
    return &Client{
        baseURL:    baseURL,
        httpClient: httpclient.NewClient(httpOpts),
        logger:     opts.Logger,
        mode:       opts.Mode,
    }
}

func (c *Client) CreateTunnel(ctx context.Context, localPort int) (*api.TunnelResponse, error) {
    switch c.mode {
    case config.ModeLocal:
        return c.createLocalTunnel(ctx, localPort)
    case config.ModeRemote:
        return c.createRemoteTunnel(ctx, localPort)
    default:
        return nil, fmt.Errorf("unsupported mode: %s", c.mode)
    }
}

func (c *Client) createLocalTunnel(ctx context.Context, localPort int) (*api.TunnelResponse, error) {
    // Create in-memory listener
    c.listener = relay.NewMockListener("hc-local")
    
    // TODO: Register with shared registry (for gateway to access)
    
    return &api.TunnelResponse{
        PublicURL:            "http://localhost:8080/tunnel/local",
        RelayEndpoint:        "in-memory",
        HybridConnectionName: "hc-local",
        ListenerToken:        "local-mode",
        SessionID:            "local-session",
    }, nil
}

func (c *Client) createRemoteTunnel(ctx context.Context, localPort int) (*api.TunnelResponse, error) {
    // Existing HTTP API call to gateway
    // ... (current CreateTunnel implementation)
}

// NEW: Start listening for connections
func (c *Client) StartListening(ctx context.Context, localPort int) error {
    if c.listener == nil {
        return fmt.Errorf("no listener available; call CreateTunnel first")
    }
    
    for {
        conn, err := c.listener.Accept(ctx)
        if err != nil {
            return err
        }
        
        go c.handleConnection(conn, localPort)
    }
}

// NEW: Handle individual connections
func (c *Client) handleConnection(conn relay.Connection, localPort int) {
    // TODO: Implement HTTP request parsing and forwarding to localhost:localPort
    defer conn.Close()
}
```

#### 3.2 Client CLI Updates

**File**: `client/cmd/start.go` (UPDATED)

Add mode flag and call StartListening:

```go
var (
    portFlag   int
    apiURLFlag string
    modeFlag   string  // NEW
)

var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start the tunnel and forward traffic to localhost",
    Long:  `Start the tunnel and forward traffic to localhost`,
    RunE: func(cmd *cobra.Command, args []string) error {
        log := GetLogger()
        
        // Parse and validate mode
        mode := config.Mode(modeFlag)
        if !mode.IsValid() {
            return fmt.Errorf("invalid mode: %s (must be 'local' or 'remote')", modeFlag)
        }
        
        log.Info("Starting tunnel", 
            logging.Int("port", portFlag),
            logging.String("mode", mode.String()))

        // Create Gateway API client with mode
        gatewayClient := gateway.NewClient(&gateway.Options{
            BaseURL: apiURLFlag,
            Logger:  log,
            Mode:    mode,
        })

        // Call Gateway API to create tunnel
        ctx := context.Background()
        tunnelResp, err := gatewayClient.CreateTunnel(ctx, portFlag)
        if err != nil {
            return fmt.Errorf("failed to create tunnel: %w", err)
        }

        // Print the public URL
        cmd.Println("Tunnel established")
        cmd.Println(fmt.Sprintf("Public URL: %s", tunnelResp.PublicURL))
        cmd.Println(fmt.Sprintf("Forwarding to: http://localhost:%d", portFlag))

        log.Info("Tunnel created successfully",
            logging.String("public_url", tunnelResp.PublicURL),
            logging.String("session_id", tunnelResp.SessionID))

        // Start listening for connections (both local and remote modes)
        if err := gatewayClient.StartListening(ctx, portFlag); err != nil {
            return fmt.Errorf("listener error: %w", err)
        }

        return nil
    },
}

func init() {
    rootCmd.AddCommand(startCmd)
    startCmd.Flags().IntVarP(&portFlag, "port", "p", defaultPort, "Local port to forward traffic to")
    startCmd.Flags().StringVar(&apiURLFlag, "api-url", defaultAPIURL, "Gateway API base URL")
    startCmd.Flags().StringVar(&modeFlag, "mode", string(config.ModeRemote), 
        "Operation mode: local or remote")
}
```

## Usage Examples

### Local Mode (Development)

```bash
# Terminal 1: Start gateway in local mode
cd gateway
go run main.go start --mode local --port 8080

# Terminal 2: Start client in local mode
cd client
go run main.go start --mode local --port 3000

# Terminal 3: Test the tunnel
curl http://localhost:8080/tunnel/local/test
```

### Remote Mode (Production)

```bash
# Gateway (running in Azure or on premise)
gateway start --mode remote --port 8080

# Client (running on developer's machine)
client start --mode remote --port 3000
```

### Default Behavior

When `--mode` is omitted, both gateway and client default to `remote` mode:

```bash
# These are equivalent
gateway start
gateway start --mode remote
```

## Data Flow

### Local Mode Flow

```
Browser/curl
    ↓
Gateway HTTP Server (:8080)
    ↓
tunnel.Manager.GetSender("hc-local")
    ↓
MockSender.Dial() → MockListener.Accept()
    ↓
client.handleConnection()
    ↓
localhost:3000 (user's app)
```

### Remote Mode Flow

```
Browser/curl
    ↓
Gateway HTTP Server (:8080)
    ↓
Azure Relay (Hybrid Connection)
    ↓
Client Relay Listener
    ↓
client.handleConnection()
    ↓
localhost:3000 (user's app)
```

## Shared Registry Pattern (Local Mode)

For local mode to work, the gateway and client need to share access to the same `MockListener` instance. This requires a registry:

### In-Memory Map (Single Process)
If running gateway and client in the same process for testing:

```go
var localRegistry = make(map[string]*relay.MockListener)

// Gateway creates and registers
listener := relay.NewMockListener("hc-local")
localRegistry["hc-local"] = listener

// Client retrieves
listener := localRegistry["hc-local"]
```

## Testing Strategy

### Unit Tests

- `internal/config/mode_test.go`: Test mode validation
- `gateway/tunnel/manager_test.go`: Test local and remote tunnel creation
- `client/gateway/client_test.go`: Test local and remote client behavior

### Integration Tests

Create integration test that runs both gateway and client in local mode:

```go
func TestLocalModeIntegration(t *testing.T) {
    // Start gateway with local mode
    manager := tunnel.NewManager(&tunnel.Options{
        Mode: config.ModeLocal,
    })
    
    // Start client with local mode
    client := gateway.NewClient(&gateway.Options{
        Mode: config.ModeLocal,
    })
    
    // Create tunnel
    resp, err := client.CreateTunnel(ctx, 3000)
    
    // Verify mock relay connection works
    // ...
}
```

## To sum up

- Add `internal/config/mode.go`
- Update gateway and client to support mode flag
- Keep existing behavior as `remote` mode
- Implement `createLocalTunnel()` in both packages
- Add shared registry mechanism
- Add HTTP request forwarding logic

## Benefits of This Design

1. **Extensible**: Easy to add new modes without changing existing code
2. **Type-Safe**: Compile-time validation of modes
3. **Clean Separation**: CLI stays lean, business logic in dedicated packages
4. **Testable**: Each mode can be tested independently
5. **Consistent**: Same pattern in both gateway and client
6. **Explicit**: `--mode remote` is clearer than default behavior
7. **Production-Ready**: Default to safe production mode

## TODO Items

- [ ] Implement `createLocalTunnel()` in both gateway and client
- [ ] Implement shared registry for local mode
- [ ] Add HTTP request forwarding logic in client
- [ ] Implement `createRemoteTunnel()` with Azure Relay
- [ ] Add integration tests for local mode
- [ ] Add documentation for local development workflow
- [ ] Consider adding `--mode` validation at flag parse time
- [ ] Add logging for mode switches

## IMPORTANT
- Do not implement anything related to Azure Relay, just set placeholder func for remote mode.