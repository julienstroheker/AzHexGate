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

**File**: `internal/config/mode.go`

Already implemented - defines `ModeLocal` and `ModeRemote` enums with validation.

### 2. Gateway Implementation

#### 2.1 Tunnel Manager

**File**: `gateway/tunnel/manager.go`

The tunnel manager encapsulates all tunnel creation and management logic. In local mode, it creates and stores MockListener instances in an in-memory registry.

**Key Methods**:

- `CreateTunnel()`: Routes to `createLocalTunnel()` or `createRemoteTunnel()` based on mode
- `GetListener()`: Returns the mock listener for a given hybrid connection (local mode only)
- `GetSender()`: Creates a mock sender for proxying requests (local mode only)

**Local Mode Behavior**:
- Creates `MockListener` instances with unique hybrid connection names
- Stores them in an in-memory map (`listeners`)
- Returns tunnel metadata with `/tunnel/{hcName}` URLs for proxying

**Remote Mode Behavior**:
- TODO: Connect to Azure Relay
- TODO: Obtain real listener tokens
- Returns placeholder production tunnel metadata

#### 2.2 HTTP Traffic Proxy Handler

**File**: `gateway/http/handlers/proxy.go` ✅ IMPLEMENTED

The proxy handler routes incoming HTTP traffic through the relay to the client. This works for **both local and remote modes** - it's not just a testing feature, it's the core production functionality.

**Key Features**:
- Extracts tunnel ID from URL path (local mode: `/tunnel/hc-abc123/path`) or Host header (remote mode: `subdomain.domain.tld`)
- Gets the relay sender from the tunnel manager
- Dials a relay connection using `sender.Dial()`
- Forwards the HTTP request through the relay connection
- Reads the HTTP response back and returns it to the client

**Request Flow**:
```
Browser → Gateway :8080/tunnel/hc-abc123/api/users
         ↓
    Extract hc-abc123
         ↓
    manager.GetSender(hc-abc123)
         ↓
    sender.Dial(ctx) → Connects to MockListener
         ↓
    Write HTTP request to connection
         ↓
    Read HTTP response from connection
         ↓
    Return response to browser
```

#### 2.3 Internal Listener Connection Endpoint

**File**: `gateway/http/handlers/listen.go` ✅ IMPLEMENTED

This endpoint allows clients to connect and accept relay connections, mimicking Azure Relay's listener behavior.

**Endpoint**: `GET /internal/listen/{hcName}`

**Behavior**:
- Client makes a GET request to this endpoint with the hybrid connection name
- Gateway looks up the MockListener from its registry
- Gateway calls `listener.Accept(ctx)` to wait for an incoming connection
- When a connection arrives (via proxy handler's `sender.Dial()`), the Accept returns
- Gateway streams the connection through the HTTP response (simplified implementation)

**Note**: The current implementation is simplified - a production version would need bidirectional streaming using WebSockets, HTTP/2, or Server-Sent Events.

#### 2.4 HTTP Server Updates

**File**: `gateway/http/server.go` ✅ UPDATED

Added two new routes:
- `/tunnel/*` - Proxy handler for routing traffic through relay
- `/internal/listen/*` - Listener endpoint for clients to accept connections

Both handlers receive the tunnel manager as a dependency for accessing the relay registry.

#### 2.5 Tunnels Handler Updates

**File**: `gateway/http/handlers/tunnels.go` ✅ ALREADY UPDATED

Uses factory pattern `NewTunnelsHandler(manager)` to delegate tunnel creation to the manager.

#### 2.6 Gateway CLI Updates

**File**: `gateway/cmd/start.go` ✅ ALREADY UPDATED

Added `--mode` flag with validation and wires up the tunnel manager with the selected mode.

### 3. Client Implementation

#### 3.1 Gateway Client Updates

**File**: `client/gateway/client.go` ✅ ALREADY UPDATED

Added mode support to the client with `Mode` field in Options.

#### 3.2 Tunnel Creation

**File**: `client/gateway/tunnel.go` ✅ UPDATED

In local mode, the client now **calls the gateway API** (same as remote mode) instead of creating its own listener. This ensures:
- Gateway creates and owns the MockListener
- Client gets back the correct tunnel ID
- Both processes are actually communicating
- Mimics Azure Relay behavior

```go
func (c *Client) createLocalTunnel(ctx, logger, localPort) {
    // Call gateway API - gateway creates MockListener
    return c.createRemoteTunnel(ctx, logger, localPort)
}
```

#### 3.3 Listener Connection

**File**: `client/gateway/listener.go` ✅ IMPLEMENTED

The client connects to the gateway's listener endpoint to accept relay connections.

**Key Implementation**:

```go
func (c *Client) StartListening(ctx, logger, localPort, tunnelResp) error {
    // In local mode, continuously poll gateway's listener endpoint
    for {
        // Make GET request to /internal/listen/{hcName}
        // This mimics Azure Relay's Accept() behavior
        acceptOneConnection(ctx, logger, localPort, hcName)
    }
}
```

**Connection Flow**:
1. Client calls `GET /internal/listen/{hcName}`
2. Gateway's listener endpoint calls `listener.Accept()`
3. When traffic comes in, proxy handler's `sender.Dial()` sends a connection
4. Gateway's Accept() returns that connection
5. Client reads HTTP request from the response stream
6. Client forwards request to `localhost:{localPort}`
7. Client reads response from localhost
8. (In current simplified implementation, response doesn't flow back)

#### 3.4 Connection Handler

**File**: `client/gateway/listener.go` ✅ IMPLEMENTED

```go
func (c *Client) handleConnection(req, connReader, logger, localPort) {
    // Update request URL to point to localhost
    req.URL.Scheme = "http"
    req.URL.Host = fmt.Sprintf("localhost:%d", localPort)
    
    // Forward to localhost using standard http.Client
    resp, err := client.Do(req)
    
    // Note: Response streaming back through the HTTP connection
    // is simplified in this implementation
}
```

#### 3.5 Client CLI Updates

**File**: `client/cmd/start.go` ✅ UPDATED

Added `--mode` flag and passes tunnel response to `StartListening()`.

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