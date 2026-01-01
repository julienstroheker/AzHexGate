package relay

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/julienstroheker/AzHexGate/internal/relay"
)

const testHTTPRequest = "GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n"

// setupTestEnvironment creates a test environment with local server, relay, and sender
func setupTestEnvironment(t *testing.T, handler http.HandlerFunc) (
	localServer *httptest.Server,
	sender *Sender,
	ctx context.Context,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
) {
	t.Helper()

	// Create local HTTP server
	localServer = httptest.NewServer(handler)

	// Create in-memory relay
	memoryListener := relay.NewMemoryListener()
	memorySender := relay.NewMemorySender(memoryListener)

	// Create relay sender
	sender = NewSender(&Options{
		Relay: memorySender,
	})

	// Create context
	ctx, cancel = context.WithCancel(context.Background())

	// Start listener that simulates local client behavior
	wg = &sync.WaitGroup{}
	wg.Add(1)
	go startListenerLoop(ctx, memoryListener, localServer.URL, wg)

	return
}

// startListenerLoop runs the listener loop that accepts connections and forwards to local server
func startListenerLoop(ctx context.Context, memoryListener *relay.MemoryListener, localURL string, wg *sync.WaitGroup) {
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

		// Handle connection
		go handleRelayConnection(ctx, relayConn, localURL)
	}
}

// handleRelayConnection handles a single relay connection by forwarding to local server
func handleRelayConnection(ctx context.Context, conn relay.Connection, localURL string) {
	defer func() { _ = conn.Close() }()

	// Dial the local TCP server
	var dialer net.Dialer
	localConn, err := dialer.DialContext(ctx, "tcp", strings.TrimPrefix(localURL, "http://"))
	if err != nil {
		return
	}
	defer func() { _ = localConn.Close() }()

	// Bidirectional copy between relay and local server
	done := make(chan error, 2)

	go func() {
		_, err := io.Copy(localConn, conn)
		done <- err
	}()

	go func() {
		_, err := io.Copy(conn, localConn)
		done <- err
	}()

	<-done
	_ = conn.Close()
	_ = localConn.Close()
	<-done
}

// cleanupTestEnvironment cleans up test resources
func cleanupTestEnvironment(
	localServer *httptest.Server,
	sender *Sender,
	cancel context.CancelFunc,
	wg *sync.WaitGroup,
) {
	cancel()
	wg.Wait()
	_ = sender.Close()
	localServer.Close()
}

func TestSender_ForwardRequestGET(t *testing.T) {
	localServer, sender, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Hello from local server"))
		}))
	defer cleanupTestEnvironment(localServer, sender, cancel, wg)

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	// Dial relay
	conn, err := sender.relay.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Write HTTP request
	_, err = conn.Write([]byte(testHTTPRequest))
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// Read HTTP response
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify response
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

func TestSender_ForwardRequestPOST(t *testing.T) {
	localServer, sender, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			body, _ := io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(body)
		}))
	defer cleanupTestEnvironment(localServer, sender, cancel, wg)

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	// Dial relay
	conn, err := sender.relay.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Write POST request
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

	// Read response
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Verify response
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

func TestSender_ForwardRequestLocalServerError(t *testing.T) {
	localServer, sender, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		}))
	defer cleanupTestEnvironment(localServer, sender, cancel, wg)

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	// Dial relay
	conn, err := sender.relay.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Write HTTP request
	_, err = conn.Write([]byte(testHTTPRequest))
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
}

func TestSender_ForwardRequestListenerOffline(t *testing.T) {
	// Create in-memory relay without starting a listener
	memoryListener := relay.NewMemoryListener()
	_ = memoryListener.Close() // Close immediately to simulate offline
	memorySender := relay.NewMemorySender(memoryListener)
	defer func() { _ = memorySender.Close() }()

	// Create relay sender
	sender := NewSender(&Options{
		Relay: memorySender,
	})
	defer func() { _ = sender.Close() }()

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Try to dial - should fail
	_, err := sender.relay.Dial(ctx)
	if err == nil {
		t.Error("Expected error when listener is offline, got nil")
	}
}

func TestSender_ForwardRequestMultipleRequests(t *testing.T) {
	requestCount := 0
	var mu sync.Mutex
	localServer, sender, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			requestCount++
			count := requestCount
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, "Request #%d", count)
		}))
	defer cleanupTestEnvironment(localServer, sender, cancel, wg)

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	// Send multiple requests
	numRequests := 5
	for i := 0; i < numRequests; i++ {
		conn, err := sender.relay.Dial(ctx)
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

func TestSender_ForwardRequestWithHeaders(t *testing.T) {
	localServer, sender, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Echo the custom header back
			customHeader := r.Header.Get("X-Custom-Header")
			w.Header().Set("X-Echo-Header", customHeader)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}))
	defer cleanupTestEnvironment(localServer, sender, cancel, wg)

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	// Dial relay
	conn, err := sender.relay.Dial(ctx)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	// Write request with custom headers
	request := "GET /headers HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"X-Custom-Header: test-value\r\n" +
		"\r\n"

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

	// Verify response
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	echoHeader := resp.Header.Get("X-Echo-Header")
	if echoHeader != "test-value" {
		t.Errorf("Expected echo header %q, got %q", "test-value", echoHeader)
	}
}

func TestSender_ForwardRequestRaw(t *testing.T) {
	localServer, sender, ctx, cancel, wg := setupTestEnvironment(t,
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Hello via raw forwarding"))
		}))
	defer cleanupTestEnvironment(localServer, sender, cancel, wg)

	// Give listener time to start
	time.Sleep(50 * time.Millisecond)

	// Create a pipe to simulate client connection
	clientReader, clientWriter := net.Pipe()
	defer func() { _ = clientReader.Close() }()
	defer func() { _ = clientWriter.Close() }()

	// Start ForwardRequestRaw in a goroutine
	go func() {
		_ = sender.ForwardRequestRaw(ctx, clientReader, nil)
	}()

	// Write request from client side
	_, err := clientWriter.Write([]byte(testHTTPRequest))
	if err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}

	// Read response from client side
	resp, err := http.ReadResponse(bufio.NewReader(clientWriter), nil)
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

	expectedBody := "Hello via raw forwarding"
	if string(body) != expectedBody {
		t.Errorf("Expected body %q, got %q", expectedBody, string(body))
	}
}

func TestSender_Close(t *testing.T) {
	// Create in-memory relay
	memoryListener := relay.NewMemoryListener()
	defer func() { _ = memoryListener.Close() }()
	memorySender := relay.NewMemorySender(memoryListener)

	// Create relay sender
	sender := NewSender(&Options{
		Relay: memorySender,
	})

	// Close sender
	err := sender.Close()
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}

	// Close again should be safe
	err = sender.Close()
	if err != nil {
		t.Errorf("Expected no error on second close, got %v", err)
	}
}

func TestSender_NewSenderWithNilOptions(t *testing.T) {
	// Should not panic with nil options
	sender := NewSender(nil)
	if sender == nil {
		t.Error("Expected non-nil sender")
	}
}
