package handlers

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/julienstroheker/AzHexGate/gateway/relay"
	azureRelay "github.com/julienstroheker/AzHexGate/internal/azure/relay"
	"github.com/julienstroheker/AzHexGate/internal/logging"
	internalRelay "github.com/julienstroheker/AzHexGate/internal/relay"
)

// ProxyConfig holds configuration for the proxy handler
type ProxyConfig struct {
	RelayNamespace string
	RelayKeyName   string
	RelayKey       string
	BaseDomain     string
	Logger         *logging.Logger
}

var proxyConfig *ProxyConfig

// SetProxyConfig sets the proxy configuration
func SetProxyConfig(cfg *ProxyConfig) {
	proxyConfig = cfg
}

// ProxyHandler handles incoming tunnel requests by forwarding them through Azure Relay
func ProxyHandler(w http.ResponseWriter, r *http.Request) {
	logger := proxyConfig.Logger
	if logger == nil {
		logger = logging.New(logging.InfoLevel)
	}

	// Extract subdomain from Host header
	host := r.Host
	// Remove port if present
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	// Extract subdomain (e.g., "c12aaac4" from "c12aaac4.azhexgate.com")
	subdomain := extractSubdomain(host, proxyConfig.BaseDomain)
	if subdomain == "" {
		logger.Warn("Invalid subdomain in request", logging.String("host", r.Host))
		http.Error(w, "Invalid subdomain", http.StatusBadRequest)
		return
	}

	// Derive Hybrid Connection name from subdomain
	hcName := fmt.Sprintf("hc-%s", subdomain)

	logger.Debug("Proxying request through relay",
		logging.String("subdomain", subdomain),
		logging.String("hc_name", hcName),
		logging.String("method", r.Method),
		logging.String("path", r.URL.Path))

	// Generate SAS token for relay connection
	token, err := azureRelay.GenerateSASToken(
		proxyConfig.RelayNamespace,
		hcName,
		proxyConfig.RelayKeyName,
		proxyConfig.RelayKey,
		1*time.Hour, // Token valid for 1 hour
	)
	if err != nil {
		logger.Error("Failed to generate sender token", logging.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create Azure Relay sender
	relayEndpoint := fmt.Sprintf("%s.servicebus.windows.net", proxyConfig.RelayNamespace)
	azureSender, err := internalRelay.NewAzureSender(&internalRelay.AzureSenderOptions{
		RelayEndpoint:        relayEndpoint,
		HybridConnectionName: hcName,
		Token:                token,
	})
	if err != nil {
		logger.Error("Failed to create Azure sender", logging.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer azureSender.Close()

	// Create gateway sender wrapper
	sender := relay.NewSender(&relay.Options{
		Relay: azureSender,
	})
	defer sender.Close()

	// Dial the relay first to get the connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	relayConn, err := azureSender.Dial(ctx)
	if err != nil {
		logger.Error("Failed to dial relay", logging.Error(err))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer relayConn.Close()

	logger.Debug("Connected to relay, writing HTTP request")

	// Write the HTTP request to the relay connection (so it reaches the client)
	if err := r.Write(relayConn); err != nil {
		logger.Error("Failed to write request to relay", logging.Error(err))
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	logger.Debug("HTTP request written to relay, hijacking connection for response")

	// Now hijack the client connection to stream the response back
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		logger.Error("Response writer does not support hijacking")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	clientConn, bufrw, err := hijacker.Hijack()
	if err != nil {
		logger.Error("Failed to hijack connection", logging.Error(err))
		return
	}
	defer clientConn.Close()

	// Flush any buffered data
	if err := bufrw.Flush(); err != nil {
		logger.Error("Failed to flush buffer", logging.Error(err))
		return
	}

	// Bidirectional copy between client and relay
	done := make(chan error, 2)

	// Copy from relay to client (response)
	go func() {
		_, err := io.Copy(clientConn, relayConn)
		done <- err
	}()

	// Copy from client to relay (any additional data like POST body)
	go func() {
		_, err := io.Copy(relayConn, clientConn)
		done <- err
	}()

	// Wait for one direction to complete
	err = <-done

	// Close connections to terminate the other goroutine
	_ = relayConn.Close()
	_ = clientConn.Close()

	// Wait for the other goroutine
	<-done

	if err != nil && err != io.EOF {
		logger.Debug("Connection closed with error", logging.Error(err))
	} else {
		logger.Debug("Request forwarded successfully through relay")
	}
}

// extractSubdomain extracts the subdomain from a host
// For example: "c12aaac4.azhexgate.com" with baseDomain "azhexgate.com" returns "c12aaac4"
func extractSubdomain(host, baseDomain string) string {
	// Normalize to lowercase
	host = strings.ToLower(host)
	baseDomain = strings.ToLower(baseDomain)

	// Check if host ends with baseDomain
	if !strings.HasSuffix(host, "."+baseDomain) {
		// Check if it's exactly the base domain (no subdomain)
		if host == baseDomain {
			return ""
		}
		return ""
	}

	// Extract subdomain by removing the base domain suffix
	subdomain := strings.TrimSuffix(host, "."+baseDomain)

	// Validate subdomain (only alphanumeric and hyphens)
	for _, ch := range subdomain {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return ""
		}
	}

	return subdomain
}

// isWebSocketUpgrade checks if the request is a WebSocket upgrade
func isWebSocketUpgrade(r *http.Request) bool {
	return strings.ToLower(r.Header.Get("Connection")) == "upgrade" &&
		strings.ToLower(r.Header.Get("Upgrade")) == "websocket"
}

// isManagementPath checks if the path is a management API path
func isManagementPath(path string) bool {
	return strings.HasPrefix(path, "/api/") || strings.HasPrefix(path, "/healthz")
}

// shouldProxyRequest determines if a request should be proxied through the relay
// Logic: If subdomain exists → proxy; If base domain → don't proxy (management API)
func shouldProxyRequest(r *http.Request, baseDomain string) bool {
	// Extract subdomain
	host := r.Host
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	subdomain := extractSubdomain(host, baseDomain)

	// If there's a subdomain, ALWAYS proxy (even /api/* paths on the subdomain)
	// This ensures local apps with /api routes work through the tunnel
	return subdomain != ""
}

// ProxyMiddleware wraps an http.Handler and proxies requests with valid subdomains
func ProxyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If no proxy config, skip proxying
		if proxyConfig == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Check if this request should be proxied
		if shouldProxyRequest(r, proxyConfig.BaseDomain) {
			ProxyHandler(w, r)
			return
		}

		// Otherwise, pass to next handler
		next.ServeHTTP(w, r)
	})
}

// connWrapper wraps a net.Conn to provide additional functionality
type connWrapper struct {
	net.Conn
}
