package relay

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestMockConnection_ReadWrite(t *testing.T) {
	conn := NewMockConnection()

	// Test writing
	data := []byte("hello world")
	n, err := conn.Write(data)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Test reading written data
	written := conn.GetWrittenData()
	if string(written) != string(data) {
		t.Errorf("Expected written data %q, got %q", string(data), string(written))
	}
}

func TestMockConnection_WriteToReadBuffer(t *testing.T) {
	conn := NewMockConnection()

	// Simulate incoming data
	data := []byte("incoming data")
	conn.WriteToReadBuffer(data)

	// Read the data
	buf := make([]byte, len(data))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if n != len(data) {
		t.Errorf("Expected to read %d bytes, read %d", len(data), n)
	}
	if string(buf) != string(data) {
		t.Errorf("Expected to read %q, got %q", string(data), string(buf))
	}
}

func TestMockConnection_Close(t *testing.T) {
	conn := NewMockConnection()

	// Close the connection
	if err := conn.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Write should fail after close
	_, err := conn.Write([]byte("test"))
	if err == nil {
		t.Error("Expected write to fail after close")
	}

	// Read should return EOF after close
	buf := make([]byte, 10)
	_, err = conn.Read(buf)
	if err != io.EOF {
		t.Errorf("Expected EOF after close, got: %v", err)
	}
}

func TestMockListener_Accept(t *testing.T) {
	listener := NewMockListener("test-address")

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send a connection in a goroutine
	mockConn := NewMockConnection()
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := listener.SendConnection(mockConn); err != nil {
			t.Errorf("SendConnection failed: %v", err)
		}
	}()

	// Accept the connection
	conn, err := listener.Accept(ctx)
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}
	if conn != mockConn {
		t.Error("Expected to receive the same connection")
	}
}

func TestMockListener_AcceptTimeout(t *testing.T) {
	listener := NewMockListener("test-address")

	// Create a context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Accept should timeout
	_, err := listener.Accept(ctx)
	if err == nil {
		t.Error("Expected Accept to timeout")
	}
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
	}
}

func TestMockListener_Close(t *testing.T) {
	listener := NewMockListener("test-address")

	// Close the listener
	if err := listener.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Accept should fail after close
	ctx := context.Background()
	_, err := listener.Accept(ctx)
	if err == nil {
		t.Error("Expected Accept to fail after close")
	}

	// SendConnection should fail after close
	conn := NewMockConnection()
	err = listener.SendConnection(conn)
	if err == nil {
		t.Error("Expected SendConnection to fail after close")
	}
}

func TestMockListener_Addr(t *testing.T) {
	addr := "test-relay-address"
	listener := NewMockListener(addr)

	if listener.Addr() != addr {
		t.Errorf("Expected address %q, got %q", addr, listener.Addr())
	}
}

func TestMockSender_Dial(t *testing.T) {
	listener := NewMockListener("test-address")

	sender := NewMockSender(listener)

	// Dial should create a connection
	ctx := context.Background()
	conn, err := sender.Dial(ctx)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	if conn == nil {
		t.Fatal("Expected connection to be created")
	}

	// Listener should receive the connection
	ctx2, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	receivedConn, err := listener.Accept(ctx2)
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}
	if receivedConn != conn {
		t.Error("Expected listener to receive the same connection")
	}
}

func TestMockSender_Close(t *testing.T) {
	listener := NewMockListener("test-address")
	sender := NewMockSender(listener)

	// Close the sender
	if err := sender.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Dial should fail after close
	ctx := context.Background()
	_, err := sender.Dial(ctx)
	if err == nil {
		t.Error("Expected Dial to fail after close")
	}
}

func TestMockSender_DialClosedListener(t *testing.T) {
	listener := NewMockListener("test-address")
	sender := NewMockSender(listener)

	// Close the listener
	if err := listener.Close(); err != nil {
		t.Fatalf("Close listener failed: %v", err)
	}

	// Dial should fail with closed listener
	ctx := context.Background()
	_, err := sender.Dial(ctx)
	if err == nil {
		t.Error("Expected Dial to fail with closed listener")
	}
}

func TestEndToEnd_Communication(t *testing.T) {
	// Setup
	listener := NewMockListener("relay-endpoint")
	sender := NewMockSender(listener)

	// Gateway dials the listener
	ctx := context.Background()
	senderConn, err := sender.Dial(ctx)
	if err != nil {
		t.Fatalf("Sender dial failed: %v", err)
	}

	// Client accepts the connection
	ctx2, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	listenerConn, err := listener.Accept(ctx2)
	if err != nil {
		t.Fatalf("Listener accept failed: %v", err)
	}

	// Test bidirectional communication
	testMessage := "Hello from sender"

	// Sender writes to connection
	_, err = senderConn.Write([]byte(testMessage))
	if err != nil {
		t.Fatalf("Sender write failed: %v", err)
	}

	// Listener reads from connection (they share the same connection)
	mockConn := listenerConn.(*MockConnection)
	receivedData := mockConn.GetWrittenData()
	if string(receivedData) != testMessage {
		t.Errorf("Expected to receive %q, got %q", testMessage, string(receivedData))
	}

	// Test response
	responseMessage := "Hello from listener"
	mockConn.WriteToReadBuffer([]byte(responseMessage))

	// Sender reads response
	buf := make([]byte, len(responseMessage))
	_, err = senderConn.Read(buf)
	if err != nil {
		t.Fatalf("Sender read failed: %v", err)
	}
	if string(buf) != responseMessage {
		t.Errorf("Expected to read %q, got %q", responseMessage, string(buf))
	}

	// Cleanup
	if err := senderConn.Close(); err != nil {
		t.Errorf("Close sender connection failed: %v", err)
	}
	if err := listenerConn.Close(); err != nil {
		t.Errorf("Close listener connection failed: %v", err)
	}
	if err := sender.Close(); err != nil {
		t.Errorf("Close sender failed: %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Errorf("Close listener failed: %v", err)
	}
}

func TestMockConnection_ReadEOF(t *testing.T) {
	conn := NewMockConnection()

	// Try to read when buffer is empty - should return EOF
	buf := make([]byte, 10)
	n, err := conn.Read(buf)
	if err != io.EOF {
		t.Errorf("Expected EOF on empty buffer, got: %v", err)
	}
	if n != 0 {
		t.Errorf("Expected to read 0 bytes, read %d", n)
	}
}

func TestMockConnection_MultipleWrites(t *testing.T) {
	conn := NewMockConnection()

	messages := []string{"first", "second", "third"}
	expected := strings.Join(messages, "")

	for _, msg := range messages {
		_, err := conn.Write([]byte(msg))
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}
	}

	written := string(conn.GetWrittenData())
	if written != expected {
		t.Errorf("Expected %q, got %q", expected, written)
	}
}

func TestMockListener_MultipleConnections(t *testing.T) {
	listener := NewMockListener("test-address")
	sender := NewMockSender(listener)

	ctx := context.Background()

	// Create multiple connections
	numConns := 3
	for i := 0; i < numConns; i++ {
		_, err := sender.Dial(ctx)
		if err != nil {
			t.Fatalf("Dial %d failed: %v", i, err)
		}
	}

	// Accept all connections
	for i := 0; i < numConns; i++ {
		ctx2, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_, err := listener.Accept(ctx2)
		cancel()
		if err != nil {
			t.Fatalf("Accept %d failed: %v", i, err)
		}
	}
}

func TestMockListener_CloseMultipleTimes(t *testing.T) {
	listener := NewMockListener("test-address")

	if err := listener.Close(); err != nil {
		t.Fatalf("First close failed: %v", err)
	}

	// Closing again should not fail
	if err := listener.Close(); err != nil {
		t.Errorf("Second close failed: %v", err)
	}
}
