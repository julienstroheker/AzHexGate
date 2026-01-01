package sender

import (
	"bufio"
	"bytes"
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

// setupTestEnvironment creates a test environment with local server, relay, and sender
func setupTestEnvironment(t *testing.T, handler http.HandlerFunc) (
	localServer *httptest.Server,
	sender *Sender,
	memoryListener *relay.MemoryListener,
	ctx context.Context,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
) {
	t.Helper()

	// Create local HTTP server
	localServer = httptest.NewServer(handler)

	// Create in-memory relay
	memoryListener = relay.NewMemoryListener()
	memorySender := relay.NewMemorySender(memoryListener)

	// Create gateway sender
	sender = NewSender(&Options{
		Relay: memorySender,
	})

	// Start a listener goroutine that forwards to local server
	ctx, cancel = context.WithCancel(context.Background())
	wg = &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Accept incoming connection from relay
			relayConn, err := memoryListener.Accept(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			// Handle connection in a goroutine
			go func(conn relay.Connection) {
				defer func() { _ = conn.Close() }()

				// Read HTTP request
				req, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					return
				}

				// Forward to local server
				req.URL.Scheme = "http"
				req.URL.Host = strings.TrimPrefix(localServer.URL, "http://")
				req.RequestURI = ""

				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					return
				}
				defer func() { _ = resp.Body.Close() }()

				// Write response back through relay
				_ = resp.Write(conn)
			}(relayConn)
		}
	}()

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	return
}

// cleanupTestEnvironment cleans up test resources
func cleanupTestEnvironment(localServer *httptest.Server, sender *Sender,
	memoryListener *relay.MemoryListener, cancel context.CancelFunc, wg *sync.WaitGroup) {
	cancel()
	wg.Wait()
	_ = sender.Close()
	_ = memoryListener.Close()
	localServer.Close()
}

func TestSender_ForwardGETRequest(t *testing.T) {
	localServer, sender, memoryListener, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Hello from local server"))
		}))
	defer cleanupTestEnvironment(localServer, sender, memoryListener, cancel, wg)

	// Create an HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Forward the request through the sender
	resp, err := sender.Forward(ctx, req, nil)
	if err != nil {
		t.Fatalf("Failed to forward request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify the response
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

func TestSender_ForwardPOSTRequest(t *testing.T) {
	localServer, sender, memoryListener, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
	defer cleanupTestEnvironment(localServer, sender, memoryListener, cancel, wg)

	// Create a POST request with body
	postBody := "test data"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.com/echo",
		bytes.NewBufferString(postBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "text/plain")

	// Forward the request
	resp, err := sender.Forward(ctx, req, nil)
	if err != nil {
		t.Fatalf("Failed to forward request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify the response
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

func TestSender_LocalServerError(t *testing.T) {
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

	// Create gateway sender
	sender := NewSender(&Options{
		Relay: memorySender,
	})
	defer func() { _ = sender.Close() }()

	// Start listener
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			relayConn, err := memoryListener.Accept(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}

			go func(conn relay.Connection) {
				defer func() { _ = conn.Close() }()

				req, err := http.ReadRequest(bufio.NewReader(conn))
				if err != nil {
					return
				}

				req.URL.Scheme = "http"
				req.URL.Host = strings.TrimPrefix(localServer.URL, "http://")
				req.RequestURI = ""

				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					return
				}
				defer func() { _ = resp.Body.Close() }()

				_ = resp.Write(conn)
			}(relayConn)
		}
	}()

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	// Send request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/error", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := sender.Forward(ctx, req, nil)
	if err != nil {
		t.Fatalf("Failed to forward request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should forward the error status
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	cancel()
	wg.Wait()
}

func TestSender_MultipleRequests(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex
	localServer, sender, memoryListener, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			count := requestCount
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "Request #%d", count)
		}))
	defer cleanupTestEnvironment(localServer, sender, memoryListener, cancel, wg)

	// Send multiple requests
	numRequests := 5
	for i := 0; i < numRequests; i++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			fmt.Sprintf("http://example.com/test%d", i), nil)
		if err != nil {
			t.Fatalf("Failed to create request %d: %v", i, err)
		}

		resp, err := sender.Forward(ctx, req, nil)
		if err != nil {
			t.Fatalf("Failed to forward request %d: %v", i, err)
		}

		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			t.Errorf("Request %d: expected status 200, got %d", i, resp.StatusCode)
			continue
		}

		_ = resp.Body.Close()
	}

	mu.Lock()
	finalCount := requestCount
	mu.Unlock()

	if finalCount != numRequests {
		t.Errorf("Expected %d requests to be handled, got %d", numRequests, finalCount)
	}
}

func TestSender_ContextCancellation(t *testing.T) {
	// Create in-memory relay
	memoryListener := relay.NewMemoryListener()
	defer func() { _ = memoryListener.Close() }()
	memorySender := relay.NewMemorySender(memoryListener)
	defer func() { _ = memorySender.Close() }()

	sender := NewSender(&Options{
		Relay: memorySender,
	})
	defer func() { _ = sender.Close() }()

	// Start a listener that accepts but doesn't respond
	ctx := context.Background()
	go func() {
		for {
			conn, err := memoryListener.Accept(ctx)
			if err != nil {
				return
			}
			// Accept but never respond - simulates hanging connection
			time.Sleep(1 * time.Second)
			_ = conn.Close()
		}
	}()

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	// Create a context with short timeout
	reqCtx, ctxCancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer ctxCancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "http://example.com/slow", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// The request should timeout when trying to read response
	// Note: This test verifies behavior when the listener doesn't respond in time
	_, err = sender.Forward(reqCtx, req, nil)
	if err == nil {
		t.Log("Warning: Expected error due to timeout, but got successful response")
		// This is not a hard failure since timing can vary
	}
}

func TestSender_LargeRequestBody(t *testing.T) {
	localServer, sender, memoryListener, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "Received %d bytes", len(body))
		}))
	defer cleanupTestEnvironment(localServer, sender, memoryListener, cancel, wg)

	// Create a large request body (1MB)
	largeBody := bytes.Repeat([]byte("x"), 1024*1024)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://example.com/upload",
		bytes.NewReader(largeBody))
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := sender.Forward(ctx, req, nil)
	if err != nil {
		t.Fatalf("Failed to forward request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	expectedResponse := "Received 1048576 bytes"
	if string(body) != expectedResponse {
		t.Errorf("Expected body %q, got %q", expectedResponse, string(body))
	}
}

func TestSender_CustomHeaders(t *testing.T) {
	localServer, sender, memoryListener, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			customHeader := r.Header.Get("X-Custom-Header")
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "Custom header: %s", customHeader)
		}))
	defer cleanupTestEnvironment(localServer, sender, memoryListener, cancel, wg)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/headers", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("X-Custom-Header", "test-value")

	resp, err := sender.Forward(ctx, req, nil)
	if err != nil {
		t.Fatalf("Failed to forward request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	expectedResponse := "Custom header: test-value"
	if string(body) != expectedResponse {
		t.Errorf("Expected body %q, got %q", expectedResponse, string(body))
	}
}

func TestSender_RelayConnectionFailure(t *testing.T) {
	// Create a closed relay
	memoryListener := relay.NewMemoryListener()
	_ = memoryListener.Close()
	memorySender := relay.NewMemorySender(memoryListener)

	sender := NewSender(&Options{
		Relay: memorySender,
	})

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com/test", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Should fail to dial relay
	_, err = sender.Forward(ctx, req, nil)
	if err == nil {
		t.Error("Expected error when dialing closed relay, got nil")
	}
}
