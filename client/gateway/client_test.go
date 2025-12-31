package gateway

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient(nil)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.baseURL == "" {
		t.Error("Expected baseURL to be set")
	}
}

func TestNewClientWithOptions(t *testing.T) {
	opts := &Options{
		BaseURL: "http://test.example.com",
	}

	client := NewClient(opts)
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	if client.baseURL != opts.BaseURL {
		t.Errorf("Expected baseURL '%s', got '%s'", opts.BaseURL, client.baseURL)
	}
}
