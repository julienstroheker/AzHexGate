package relay

import (
	"context"
	"testing"
	"time"
)

func TestNewManagedIdentityTokenProvider(t *testing.T) {
	// This test will fail in environments without Azure credentials
	// That's expected behavior - it's an integration test
	provider, err := NewManagedIdentityTokenProvider()
	if err != nil {
		// Skip if credentials not available (not a real failure)
		t.Skipf("Skipping test - Azure credentials not available: %v", err)
	}

	if provider == nil {
		t.Fatal("Provider should not be nil")
	}
	if provider.credential == nil {
		t.Error("Credential should not be nil")
	}
	if provider.scope == "" {
		t.Error("Scope should not be empty")
	}
}

func TestManagedIdentityTokenProvider_GetToken(t *testing.T) {
	provider, err := NewManagedIdentityTokenProvider()
	if err != nil {
		t.Skipf("Skipping test - Azure credentials not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Try to get a token
	token, err := provider.GetToken(ctx)
	if err != nil {
		t.Skipf("Skipping test - could not acquire token: %v", err)
	}

	if token == "" {
		t.Error("Token should not be empty")
	}

	// Try to get token again (should use cache)
	token2, err := provider.GetToken(ctx)
	if err != nil {
		t.Fatalf("GetToken() on cached token failed: %v", err)
	}

	if token != token2 {
		t.Error("Cached token should be the same")
	}
}
