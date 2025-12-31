// Package relay provides abstractions for Azure Relay Hybrid Connection interactions.
//
// This package defines interfaces for working with Azure Relay without directly
// depending on the Azure SDK. It enables testing with mock implementations and
// isolates Azure SDK usage to specific adapters.
//
// # Core Interfaces
//
// Connection represents a bidirectional stream between a sender and listener.
// It implements io.Reader, io.Writer, and io.Closer for standard I/O operations.
//
// Listener accepts incoming connections from senders. The local client uses this
// to receive requests from the cloud gateway through Azure Relay.
//
// Sender establishes connections to listeners. The cloud gateway uses this to
// forward requests to the local client through Azure Relay.
//
// # Mock Implementation
//
// This package includes in-memory mock implementations (MockConnection, MockListener,
// MockSender) for testing without real Azure Relay infrastructure. These mocks
// enable fast, deterministic unit tests.
//
// # Usage Example
//
//	// Create a mock listener (simulates client side)
//	listener := relay.NewMockListener("relay-endpoint")
//	defer listener.Close()
//
//	// Create a mock sender (simulates gateway side)
//	sender := relay.NewMockSender(listener)
//	defer sender.Close()
//
//	// Gateway dials the listener
//	conn, err := sender.Dial(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer conn.Close()
//
//	// Client accepts the connection
//	clientConn, err := listener.Accept(context.Background())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer clientConn.Close()
//
//	// Bidirectional communication is now possible
//	conn.Write([]byte("request data"))
//	// ... clientConn can read the data
package relay
