package http

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	port := 9999
	server := NewServer(port)

	if server == nil {
		t.Fatal("Expected server to be created, got nil")
	}

	if server.Port() != port {
		t.Errorf("Expected port %d, got %d", port, server.Port())
	}

	if server.server == nil {
		t.Error("Expected http.Server to be initialized")
	}

	if server.server.Handler == nil {
		t.Error("Expected handler to be set")
	}
}

func TestServerLifecycle(t *testing.T) {
	port := 9998
	server := NewServer(port)

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		serverErrors <- server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is running by making a request
	resp, err := http.Get("http://localhost:9998/healthz")
	if err != nil {
		t.Fatalf("Expected server to be running, got error: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Logf("Error closing response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, resp.StatusCode)
	}

	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Expected clean shutdown, got error: %v", err)
	}

	// Verify server stopped
	select {
	case <-serverErrors:
		// Server stopped, this is expected
	case <-time.After(1 * time.Second):
		t.Error("Server did not stop within expected time")
	}
}

func TestServerShutdownTimeout(t *testing.T) {
	port := 9997
	server := NewServer(port)

	// Start server
	go func() {
		// Ignore error as this is a test for shutdown behavior
		_ = server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test shutdown with immediate timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// This might return an error if context times out before shutdown completes
	// but that's okay for this test
	_ = server.Shutdown(ctx)

	// Clean up
	_ = server.Close()
}

func TestServerClose(t *testing.T) {
	port := 9996
	server := NewServer(port)

	// Start server
	go func() {
		// Ignore error as this is a test for close behavior
		_ = server.ListenAndServe()
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test immediate close
	if err := server.Close(); err != nil {
		t.Errorf("Expected clean close, got error: %v", err)
	}
}
