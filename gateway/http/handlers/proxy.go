package handlers

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/julienstroheker/AzHexGate/gateway/tunnel"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

// NewProxyHandler creates a handler that proxies traffic through the relay
// This works for both remote (Azure Relay) and local (Mock Relay) modes
func NewProxyHandler(manager *tunnel.Manager, logger *logging.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract tunnel info and target path
		hcName, targetPath, err := extractTunnelInfo(r)
		if err != nil {
			if logger != nil {
				logger.Error("Failed to extract tunnel info", logging.Error(err))
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if logger != nil {
			logger.Info("Proxying request",
				logging.String("hc_name", hcName),
				logging.String("path", targetPath),
				logging.String("method", r.Method))
		}

		// Get sender for this tunnel (works for both mock and real relay)
		sender, err := manager.GetSender(hcName)
		if err != nil {
			if logger != nil {
				logger.Error("Tunnel not found",
					logging.String("hc_name", hcName),
					logging.Error(err))
			}
			http.Error(w, fmt.Sprintf("Tunnel not found: %s", hcName), http.StatusNotFound)
			return
		}

		// Dial the relay connection (Azure Relay or Mock)
		conn, err := sender.Dial(r.Context())
		if err != nil {
			if logger != nil {
				logger.Error("Failed to dial relay", logging.Error(err))
			}
			http.Error(w, "Failed to connect to tunnel", http.StatusBadGateway)
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		// Create proxied request with modified path
		proxyReq := r.Clone(r.Context())
		proxyReq.URL.Path = targetPath
		proxyReq.RequestURI = targetPath
		if r.URL.RawQuery != "" {
			proxyReq.RequestURI += "?" + r.URL.RawQuery
		}

		if logger != nil {
			logger.Debug("Sending request through relay",
				logging.String("uri", proxyReq.RequestURI))
		}

		// Write request to relay
		if err := proxyReq.Write(conn); err != nil {
			if logger != nil {
				logger.Error("Failed to write request to relay", logging.Error(err))
			}
			http.Error(w, "Failed to send request", http.StatusInternalServerError)
			return
		}

		// Read response from relay
		reader := bufio.NewReader(conn)
		resp, err := http.ReadResponse(reader, proxyReq)
		if err != nil {
			if logger != nil {
				logger.Error("Failed to read response from relay", logging.Error(err))
			}
			http.Error(w, "Failed to receive response", http.StatusBadGateway)
			return
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if logger != nil {
			logger.Info("Received response from tunnel",
				logging.Int("status", resp.StatusCode))
		}

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Write status code
		w.WriteHeader(resp.StatusCode)

		// Copy response body
		if _, err := io.Copy(w, resp.Body); err != nil {
			if logger != nil {
				logger.Error("Failed to copy response body", logging.Error(err))
			}
		}
	}
}

// extractTunnelInfo extracts tunnel ID and target path from request
func extractTunnelInfo(r *http.Request) (hcName, targetPath string, err error) {
	// Check if this is a local mode path-based request
	if strings.HasPrefix(r.URL.Path, "/tunnel/") {
		// Local mode: /tunnel/hc-abc123/api/users
		path := strings.TrimPrefix(r.URL.Path, "/tunnel/")
		parts := strings.SplitN(path, "/", 2)

		if len(parts) < 1 || parts[0] == "" {
			return "", "", fmt.Errorf("invalid tunnel path")
		}

		hcName = parts[0]
		targetPath = "/"
		if len(parts) > 1 {
			targetPath = "/" + parts[1]
		}
		return hcName, targetPath, nil
	}

	// Production mode: extract from Host header
	// 63873749.azhexgate.com -> hc-63873749
	host := r.Host
	if host == "" {
		host = r.Header.Get("Host")
	}

	// Extract subdomain
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return "", "", fmt.Errorf("invalid host: %s (expected subdomain.domain.tld)", host)
	}

	subdomain := parts[0]
	hcName = fmt.Sprintf("hc-%s", subdomain)
	targetPath = r.URL.Path

	return hcName, targetPath, nil
}
