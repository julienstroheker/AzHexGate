package tunnel

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/relay"
)

const testHTTPRequest = "GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n"

// setupTestEnvironment creates a test environment with local server, relay, and listener
func setupTestEnvironment(t *testing.T, handler http.HandlerFunc) (
	localServer *httptest.Server,
	listener *Listener,
	memorySender *relay.MemorySender,
	ctx context.Context,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
) {
	t.Helper()

	// Create local HTTP server
	localServer = httptest.NewServer(handler)

	// Create in-memory relay
	memoryListener := relay.NewMemoryListener()
	memorySender = relay.NewMemorySender(memoryListener)

	// Create tunnel listener
	listener = NewListener(&Options{
		Relay:     memoryListener,
		LocalAddr: strings.TrimPrefix(localServer.URL, "http://"),
	})

	// Start listener in background
	ctx, cancel = context.WithCancel(context.Background())
	wg = &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = listener.Start(ctx)
	}()

	return
}

// cleanupTestEnvironment cleans up test resources
func cleanupTestEnvironment(localServer *httptest.Server, listener *Listener,
	memorySender *relay.MemorySender, cancel context.CancelFunc, wg *sync.WaitGroup) {
	cancel()
	wg.Wait()
	_ = listener.Close()
	_ = memorySender.Close()
	localServer.Close()
}

func TestListener_HandleHTTPRequest(t *testing.T) {
	localServer, listener, memorySender, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Hello from local server"))
		}))
	defer cleanupTestEnvironment(localServer, listener, memorySender, cancel, wg)

	// Simulate sending an HTTP request through the relay
	conn, err := memorySender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Write HTTP request to the connection
	_, err = conn.Write([]byte(testHTTPRequest))
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// Read and verify HTTP response
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	expectedBody := "Hello from local server"
	if string(body) != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, string(body))
	}
}

func TestListener_HandlePOSTRequest(t *testing.T) {
	localServer, listener, memorySender, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
	defer cleanupTestEnvironment(localServer, listener, memorySender, cancel, wg)

	// Simulate sending a POST request
	conn, err := memorySender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	postBody := "test data"
	request := fmt.Sprintf("POST /echo HTTP/1.1\r\n"+
		"Host: example.com\r\n"+
		"Content-Length: %d\r\n"+
		"\r\n"+
		"%s", len(postBody), postBody)

	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// Read and verify response
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}
	if string(body) != postBody {
		t.Errorf("Expected body %q, got %q", postBody, string(body))
	}
}

func TestListener_LocalServerError(t *testing.T) {
	// Create a local HTTP server that returns an error
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Internal server error"))
	}))
	defer localServer.Close()

	// Create in-memory relay
	memoryListener := relay.NewMemoryListener()
	defer func() { _ = memoryListener.Close() }()
	memorySender := relay.NewMemorySender(memoryListener)
	defer func() { _ = memorySender.Close() }()

	// Create tunnel listener
	listener := NewListener(&Options{
		Relay:     memoryListener,
		LocalAddr: strings.TrimPrefix(localServer.URL, "http://"),
	})
	defer func() { _ = listener.Close() }()

	// Start listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = listener.Start(ctx)
	}()

	// Send request
	conn, err := memorySender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	request := "GET /error HTTP/1.1\r\nHost: example.com\r\n\r\n"
	_, err = conn.Write([]byte(request))
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// Read response
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should forward the error status
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	cancel()
	wg.Wait()
}

func TestListener_LocalServerUnreachable(t *testing.T) {
	// Create in-memory relay
	memoryListener := relay.NewMemoryListener()
	defer func() { _ = memoryListener.Close() }()
	memorySender := relay.NewMemorySender(memoryListener)
	defer func() { _ = memorySender.Close() }()

	// Create tunnel listener pointing to non-existent server
	listener := NewListener(&Options{
		Relay:     memoryListener,
		LocalAddr: "localhost:99999", // Invalid port
	})
	defer func() { _ = listener.Close() }()

	// Start listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = listener.Start(ctx)
	}()

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	// Send request
	conn, err := memorySender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// With raw TCP forwarding, when local server is unreachable,
	// the listener will fail to dial and close the relay connection.
	// We should see the connection close when we try to read.

	// Try to write - may succeed if connection hasn't closed yet, or may fail
	_, writeErr := conn.Write([]byte(testHTTPRequest))

	// Try to read - should fail with EOF or connection closed
	buf := make([]byte, 1024)
	_, readErr := conn.Read(buf)

	// Either write or read should fail when local server is unreachable
	if writeErr == nil && readErr == nil {
		t.Error("Expected connection to fail when local server is unreachable")
	}

	cancel()
	wg.Wait()
}

func TestListener_MultipleRequests(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex
	localServer, listener, memorySender, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			count := requestCount
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "Request #%d", count)
		}))
	defer cleanupTestEnvironment(localServer, listener, memorySender, cancel, wg)

	// Send multiple requests
	numRequests := 5
	for i := 0; i < numRequests; i++ {
		conn, err := memorySender.Dial(ctx)
		if err != nil {
			t.Fatalf("Failed to dial for request %d: %v", i, err)
		}

		_, err = conn.Write([]byte(testHTTPRequest))
		if err != nil {
			_ = conn.Close()
			t.Fatalf("Failed to write request %d: %v", i, err)
		}

		// Read response
		resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			_ = conn.Close()
			t.Fatalf("Failed to read response %d: %v", i, err)
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			_ = conn.Close()
			t.Errorf("Request %d: expected status 200, got %d", i, resp.StatusCode)
			continue
		}

		_ = resp.Body.Close()
		_ = conn.Close()
	}

	mu.Lock()
	finalCount := requestCount
	mu.Unlock()

	if finalCount != numRequests {
		t.Errorf("Expected %d requests to be handled, got %d", numRequests, finalCount)
	}
}

func TestListener_ContextCancellation(t *testing.T) {
	// Create a local HTTP server
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer localServer.Close()

	// Create in-memory relay
	memoryListener := relay.NewMemoryListener()
	defer func() { _ = memoryListener.Close() }()

	// Create tunnel listener
	listener := NewListener(&Options{
		Relay:     memoryListener,
		LocalAddr: strings.TrimPrefix(localServer.URL, "http://"),
	})
	defer func() { _ = listener.Close() }()

	// Start listener with cancellable context
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := listener.Start(ctx)
	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestListener_InvalidHTTPRequest(t *testing.T) {
	// Create a local HTTP server
	localServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer localServer.Close()

	// Create in-memory relay
	memoryListener := relay.NewMemoryListener()
	defer func() { _ = memoryListener.Close() }()
	memorySender := relay.NewMemorySender(memoryListener)
	defer func() { _ = memorySender.Close() }()

	// Create tunnel listener
	listener := NewListener(&Options{
		Relay:     memoryListener,
		LocalAddr: strings.TrimPrefix(localServer.URL, "http://"),
	})
	defer func() { _ = listener.Close() }()

	// Start listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_ = listener.Start(ctx)
	}()

	// Send invalid HTTP request
	conn, err := memorySender.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Write invalid data
	_, err = conn.Write([]byte("NOT A VALID HTTP REQUEST\r\n\r\n"))
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	// The listener should handle the error gracefully and close the connection
	// Wait a bit for the handler to process
	time.Sleep(100 * time.Millisecond)

	cancel()
	wg.Wait()
}
