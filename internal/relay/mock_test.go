package relay

import (
	"context"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

//nolint:dupl // Test duplication is acceptable
func TestMemoryConnection_ReadWrite(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)
	defer func() { _ = sender.Close() }()

	ctx := context.Background()

	// Dial to create a connection
	senderConn, err := sender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	// Accept the connection on the listener side
	listenerConn, err := listener.Accept(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to accept: %v", err)
	}
	defer func() { _ = listenerConn.Close() }()

	// Test write from sender to listener (async write, sync read)
	message := []byte("hello from sender")
	go func() {
		n, err := senderConn.Write(message)
		if err != nil {
			t.Logf("Failed to write: %v", err)
			return
		}
		if n != len(message) {
			t.Logf("Expected to write %d bytes, wrote %d", len(message), n)
		}
	}()

	// Read on listener side
	buf := make([]byte, len(message))
	n, err := io.ReadFull(listenerConn, buf)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	if n != len(message) {
		t.Errorf("Expected to read %d bytes, read %d", len(message), n)
	}
	if string(buf) != string(message) {
		t.Errorf("Expected message %q, got %q", string(message), string(buf))
	}
}

//nolint:dupl // Test duplication is acceptable
func TestMemoryConnection_BidirectionalReadWrite(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)
	defer func() { _ = sender.Close() }()

	ctx := context.Background()

	// Dial to create a connection
	senderConn, err := sender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	// Accept the connection on the listener side
	listenerConn, err := listener.Accept(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to accept: %v", err)
	}
	defer func() { _ = listenerConn.Close() }()

	// Test write from listener to sender (async write, sync read)
	response := []byte("hello from listener")
	go func() {
		n, err := listenerConn.Write(response)
		if err != nil {
			t.Logf("Failed to write response: %v", err)
			return
		}
		if n != len(response) {
			t.Logf("Expected to write %d bytes, wrote %d", len(response), n)
		}
	}()

	// Read on sender side
	buf := make([]byte, len(response))
	n, err := io.ReadFull(senderConn, buf)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	if n != len(response) {
		t.Errorf("Expected to read %d bytes, read %d", len(response), n)
	}
	if string(buf) != string(response) {
		t.Errorf("Expected response %q, got %q", string(response), string(buf))
	}
}

func TestMemoryConnection_Close(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)
	defer func() { _ = sender.Close() }()

	ctx := context.Background()

	senderConn, err := sender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}

	listenerConn, err := listener.Accept(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to accept: %v", err)
	}

	// Close sender connection
	if err := senderConn.Close(); err != nil {
		t.Fatalf("Failed to close sender connection: %v", err)
	}

	// Try to write to closed connection
	_, err = senderConn.Write([]byte("test"))
	if !errors.Is(err, ErrConnectionClosed) {
		t.Errorf("Expected ErrConnectionClosed, got %v", err)
	}

	// Try to read from closed connection
	_, err = senderConn.Read(make([]byte, 10))
	if !errors.Is(err, ErrConnectionClosed) {
		t.Errorf("Expected ErrConnectionClosed, got %v", err)
	}

	// Close again should not error
	if err := senderConn.Close(); err != nil {
		t.Errorf("Close on already closed connection should not error: %v", err)
	}

	_ = listenerConn.Close()
}

func TestMemoryListener_Accept(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)
	defer func() { _ = sender.Close() }()

	ctx := context.Background()

	// Dial in a goroutine
	go func() {
		conn, err := sender.Dial(ctx)
		if err != nil {
			t.Logf("Dial error: %v", err)
			return
		}
		defer func() { _ = conn.Close() }()
	}()

	// Accept should receive the connection
	conn, err := listener.Accept(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to accept: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if conn == nil {
		t.Error("Expected non-nil connection")
	}
}

func TestMemoryListener_AcceptWithContext(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	// Create a context that times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Accept should return context error
	_, err := listener.Accept(ctx, nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestMemoryListener_AcceptAfterClose(t *testing.T) {
	listener := NewMemoryListener()

	// Close the listener
	if err := listener.Close(); err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}

	// Try to accept after close
	ctx := context.Background()
	_, err := listener.Accept(ctx, nil)
	if !errors.Is(err, ErrListenerClosed) {
		t.Errorf("Expected ErrListenerClosed, got %v", err)
	}
}

func TestMemoryListener_CloseMultipleTimes(t *testing.T) {
	listener := NewMemoryListener()

	// First close
	if err := listener.Close(); err != nil {
		t.Fatalf("First close failed: %v", err)
	}

	// Second close should not error
	if err := listener.Close(); err != nil {
		t.Errorf("Second close should not error: %v", err)
	}
}

func TestMemorySender_Dial(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)
	defer func() { _ = sender.Close() }()

	ctx := context.Background()

	// Dial should succeed
	conn, err := sender.Dial(ctx)
	if err != nil {
		t.Fatalf("Dial failed: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if conn == nil {
		t.Error("Expected non-nil connection")
	}
}

func TestMemorySender_DialAfterClose(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)

	// Close the sender
	if err := sender.Close(); err != nil {
		t.Fatalf("Failed to close sender: %v", err)
	}

	// Try to dial after close
	ctx := context.Background()
	_, err := sender.Dial(ctx)
	if !errors.Is(err, ErrSenderClosed) {
		t.Errorf("Expected ErrSenderClosed, got %v", err)
	}
}

func TestMemorySender_DialWithClosedListener(t *testing.T) {
	listener := NewMemoryListener()
	sender := NewMemorySender(listener)
	defer func() { _ = sender.Close() }()

	// Close the listener
	if err := listener.Close(); err != nil {
		t.Fatalf("Failed to close listener: %v", err)
	}

	// Try to dial with closed listener
	ctx := context.Background()
	_, err := sender.Dial(ctx)
	if !errors.Is(err, ErrListenerClosed) {
		t.Errorf("Expected ErrListenerClosed, got %v", err)
	}
}

func TestMemorySender_CloseMultipleTimes(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)

	// First close
	if err := sender.Close(); err != nil {
		t.Fatalf("First close failed: %v", err)
	}

	// Second close should not error
	if err := sender.Close(); err != nil {
		t.Errorf("Second close should not error: %v", err)
	}
}

func TestMemoryConnection_ConcurrentReadWrite(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)
	defer func() { _ = sender.Close() }()

	ctx := context.Background()

	senderConn, err := sender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	listenerConn, err := listener.Accept(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to accept: %v", err)
	}
	defer func() { _ = listenerConn.Close() }()

	// Send multiple messages concurrently
	numMessages := 10
	var wg sync.WaitGroup

	// Sender goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			msg := []byte("message")
			if _, err := senderConn.Write(msg); err != nil {
				t.Logf("Write error: %v", err)
				return
			}
		}
	}()

	// Receiver goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			buf := make([]byte, 7)
			if _, err := io.ReadFull(listenerConn, buf); err != nil {
				t.Logf("Read error: %v", err)
				return
			}
		}
	}()

	wg.Wait()
}

func TestMemoryConnection_LargeData(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)
	defer func() { _ = sender.Close() }()

	ctx := context.Background()

	senderConn, err := sender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = senderConn.Close() }()

	listenerConn, err := listener.Accept(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to accept: %v", err)
	}
	defer func() { _ = listenerConn.Close() }()

	// Send large data (1MB)
	size := 1024 * 1024
	largeData := make([]byte, size)

	var wg sync.WaitGroup

	// Write in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		n, err := senderConn.Write(largeData)
		if err != nil {
			t.Logf("Write error: %v", err)
		}
		if n != size {
			t.Logf("Expected to write %d bytes, wrote %d", size, n)
		}
	}()

	// Read in goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		received := make([]byte, size)
		n, err := io.ReadFull(listenerConn, received)
		if err != nil {
			t.Logf("Read error: %v", err)
			return
		}
		if n != size {
			t.Errorf("Expected to read %d bytes, read %d", size, n)
		}
	}()

	wg.Wait()
}

func TestMemoryConnection_MultipleConnections(t *testing.T) {
	listener := NewMemoryListener()
	defer func() { _ = listener.Close() }()

	sender := NewMemorySender(listener)
	defer func() { _ = sender.Close() }()

	ctx := context.Background()
	numConnections := 5

	var wg sync.WaitGroup

	// Create multiple connections
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := sender.Dial(ctx)
			if err != nil {
				t.Logf("Dial error for connection %d: %v", id, err)
				return
			}
			defer func() { _ = conn.Close() }()

			// Send a message
			msg := []byte("test")
			if _, err := conn.Write(msg); err != nil {
				t.Logf("Write error for connection %d: %v", id, err)
			}
		}(i)
	}

	// Accept all connections
	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			conn, err := listener.Accept(ctx, nil)
			if err != nil {
				t.Logf("Accept error for connection %d: %v", id, err)
				return
			}
			defer func() { _ = conn.Close() }()

			// Read the message
			buf := make([]byte, 4)
			if _, err := io.ReadFull(conn, buf); err != nil {
				t.Logf("Read error for connection %d: %v", id, err)
			}
		}(i)
	}

	wg.Wait()
}
