package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"

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
	logger := getLogger()
	subdomain, hcName, err := extractTunnelInfo(r, logger)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	logger.Debug("Proxying request through relay",
		logging.String("subdomain", subdomain),
		logging.String("hc_name", hcName),
		logging.String("method", r.Method),
		logging.String("path", r.URL.Path))

	if err := forwardRequestThroughRelay(r.Context(), w, r, hcName, logger); err != nil {
		logger.Error("Failed to forward request", logging.Error(err))
	}
}

// getLogger returns the configured logger or a default one
func getLogger() *logging.Logger {
	if proxyConfig != nil && proxyConfig.Logger != nil {
		return proxyConfig.Logger
	}
	return logging.New(logging.InfoLevel)
}

// extractTunnelInfo extracts subdomain and HC name from the request
func extractTunnelInfo(r *http.Request, logger *logging.Logger) (string, string, error) {
	host := r.Host
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	subdomain := extractSubdomain(host, proxyConfig.BaseDomain)
	if subdomain == "" {
		logger.Warn("Invalid subdomain in request", logging.String("host", r.Host))
		return "", "", fmt.Errorf("invalid subdomain")
	}

	hcName := fmt.Sprintf("hc-%s", subdomain)
	return subdomain, hcName, nil
}

// forwardRequestThroughRelay establishes a relay connection and forwards the request
func forwardRequestThroughRelay(
	ctx context.Context, w http.ResponseWriter, r *http.Request, hcName string, logger *logging.Logger,
) error {
	// Generate SAS token
	token, err := azureRelay.GenerateSASToken(
		proxyConfig.RelayNamespace, hcName,
		proxyConfig.RelayKeyName, proxyConfig.RelayKey,
		1*time.Hour,
	)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Create Azure Relay sender
	azureSender, err := createAzureSender(hcName, token)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return err
	}
	defer func() { _ = azureSender.Close() }()

	// Create gateway sender wrapper
	sender := relay.NewSender(&relay.Options{Relay: azureSender})
	defer func() { _ = sender.Close() }()

	// Dial the relay with timeout derived from request context
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	relayConn, err := azureSender.Dial(dialCtx)
	if err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return fmt.Errorf("failed to dial relay: %w", err)
	}
	defer func() { _ = relayConn.Close() }()

	logger.Debug("Connected to relay, writing HTTP request")

	// Write the HTTP request to the relay
	if err := r.Write(relayConn); err != nil {
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return fmt.Errorf("failed to write request: %w", err)
	}

	return streamResponse(w, relayConn, logger)
}

// createAzureSender creates a new Azure Relay sender
func createAzureSender(hcName, token string) (*internalRelay.AzureSender, error) {
	relayEndpoint := fmt.Sprintf("%s.servicebus.windows.net", proxyConfig.RelayNamespace)
	return internalRelay.NewAzureSender(&internalRelay.AzureSenderOptions{
		RelayEndpoint:        relayEndpoint,
		HybridConnectionName: hcName,
		Token:                token,
	})
}

// streamResponse hijacks the connection and streams the response back
func streamResponse(w http.ResponseWriter, relayConn internalRelay.Connection, logger *logging.Logger) error {
	logger.Debug("HTTP request written to relay, hijacking connection for response")

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return fmt.Errorf("response writer does not support hijacking")
	}

	clientConn, bufrw, err := hijacker.Hijack()
	if err != nil {
		return fmt.Errorf("failed to hijack connection: %w", err)
	}
	defer func() { _ = clientConn.Close() }()

	if err := bufrw.Flush(); err != nil {
		return fmt.Errorf("failed to flush buffer: %w", err)
	}

	// Bidirectional copy
	done := make(chan error, 2)
	go func() { _, err := io.Copy(clientConn, relayConn); done <- err }()
	go func() { _, err := io.Copy(relayConn, clientConn); done <- err }()

	err = <-done
	_ = relayConn.Close()
	_ = clientConn.Close()
	<-done

	if err != nil && err != io.EOF {
		logger.Debug("Connection closed with error", logging.Error(err))
	} else {
		logger.Debug("Request forwarded successfully through relay")
	}
	return nil
}

// extractSubdomain extracts the subdomain from a host
// For example: "c12aaac4.azhexgate.com" with baseDomain "azhexgate.com" returns "c12aaac4"
func extractSubdomain(host, baseDomain string) string {
	// Normalize to lowercase
	host = strings.ToLower(host)
	baseDomain = strings.ToLower(baseDomain)

	// Check if host ends with baseDomain
	if !strings.HasSuffix(host, "."+baseDomain) {
		return ""
	}

	// Extract subdomain by removing the base domain suffix
	subdomain := strings.TrimSuffix(host, "."+baseDomain)

	// Validate subdomain (only alphanumeric and hyphens)
	for _, ch := range subdomain {
		if !unicode.IsLetter(ch) && !unicode.IsDigit(ch) && ch != '-' {
			return ""
		}
	}

	return subdomain
}

// shouldProxyRequest determines if a request should be proxied through the relay
// Logic: If subdomain exists → proxy; If base domain → don't proxy (management API)
func shouldProxyRequest(r *http.Request, baseDomain string) bool {
	host := r.Host
	if colonIdx := strings.Index(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}

	// If there's a subdomain, ALWAYS proxy (even /api/* paths on the subdomain)
	// This ensures local apps with /api routes work through the tunnel
	return extractSubdomain(host, baseDomain) != ""
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
