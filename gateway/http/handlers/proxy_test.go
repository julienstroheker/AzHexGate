package handlers

import (
	"net/http"
	"testing"
)

func TestExtractSubdomain(t *testing.T) { //nolint:funlen // Table-driven test
	tests := []struct {
		name       string
		host       string
		baseDomain string
		want       string
	}{
		{
			name:       "valid subdomain",
			host:       "c12aaac4.azhexgate.com",
			baseDomain: "azhexgate.com",
			want:       "c12aaac4",
		},
		{
			name:       "uppercase host",
			host:       "C12AAAC4.AZHEXGATE.COM",
			baseDomain: "azhexgate.com",
			want:       "c12aaac4",
		},
		{
			name:       "mixed case subdomain",
			host:       "Test123.azhexgate.com",
			baseDomain: "azhexgate.com",
			want:       "test123",
		},
		{
			name:       "subdomain with hyphens",
			host:       "test-123-abc.azhexgate.com",
			baseDomain: "azhexgate.com",
			want:       "test-123-abc",
		},
		{
			name:       "no subdomain (base domain)",
			host:       "azhexgate.com",
			baseDomain: "azhexgate.com",
			want:       "",
		},
		{
			name:       "wrong base domain",
			host:       "c12aaac4.example.com",
			baseDomain: "azhexgate.com",
			want:       "",
		},
		{
			name:       "invalid characters in subdomain",
			host:       "test_123.azhexgate.com",
			baseDomain: "azhexgate.com",
			want:       "",
		},
		{
			name:       "empty host",
			host:       "",
			baseDomain: "azhexgate.com",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractSubdomain(tt.host, tt.baseDomain)
			if got != tt.want {
				t.Errorf("extractSubdomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldProxyRequest(t *testing.T) { //nolint:funlen // Table-driven test
	tests := []struct {
		name       string
		host       string
		path       string
		baseDomain string
		want       bool
	}{
		{
			name:       "subdomain with root path - should proxy",
			host:       "c12aaac4.azhexgate.com",
			path:       "/",
			baseDomain: "azhexgate.com",
			want:       true,
		},
		{
			name:       "subdomain with api path - should proxy (local app may have /api)",
			host:       "c12aaac4.azhexgate.com",
			path:       "/api/users",
			baseDomain: "azhexgate.com",
			want:       true,
		},
		{
			name:       "subdomain with healthz path - should proxy",
			host:       "c12aaac4.azhexgate.com",
			path:       "/healthz",
			baseDomain: "azhexgate.com",
			want:       true,
		},
		{
			name:       "base domain with api path - should not proxy (management API)",
			host:       "azhexgate.com",
			path:       "/api/tunnels",
			baseDomain: "azhexgate.com",
			want:       false,
		},
		{
			name:       "base domain with healthz - should not proxy",
			host:       "azhexgate.com",
			path:       "/healthz",
			baseDomain: "azhexgate.com",
			want:       false,
		},
		{
			name:       "base domain with root - should not proxy",
			host:       "azhexgate.com",
			path:       "/",
			baseDomain: "azhexgate.com",
			want:       false,
		},
		{
			name:       "subdomain with port - should proxy",
			host:       "c12aaac4.azhexgate.com:8080",
			path:       "/",
			baseDomain: "azhexgate.com",
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "http://"+tt.host+tt.path, nil)
			req.Host = tt.host
			got := shouldProxyRequest(req, tt.baseDomain)
			if got != tt.want {
				t.Errorf("shouldProxyRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}
