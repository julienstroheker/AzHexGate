package relay

import (
	"testing"
	"time"
)

// TestIntegration_SASTokenGeneration demonstrates real SAS token generation
func TestIntegration_SASTokenGeneration(t *testing.T) {
	// This test demonstrates that we can generate real SAS tokens
	// It does not require a real Azure Relay namespace to run

	relayNamespace := "test-relay"
	hybridConnectionName := "hc-12345"
	keyName := "RootManageSharedAccessKey"
	// This is a test key - base64 encoded "testkey"
	key := "dGVzdGtleQ=="

	// Generate token with 24h expiry
	listenerToken, err := GenerateSASToken(
		relayNamespace,
		hybridConnectionName,
		keyName,
		key,
		24*time.Hour,
	)
	if err != nil {
		t.Fatalf("Failed to generate listener token: %v", err)
	}

	if listenerToken == "" {
		t.Error("Listener token should not be empty")
	}

	t.Logf("Generated listener token: %s", listenerToken[:50]+"...")

	// Generate token with 1h expiry
	senderToken, err := GenerateSASToken(
		relayNamespace,
		hybridConnectionName,
		keyName,
		key,
		1*time.Hour,
	)
	if err != nil {
		t.Fatalf("Failed to generate sender token: %v", err)
	}

	if senderToken == "" {
		t.Error("Sender token should not be empty")
	}

	t.Logf("Generated sender token: %s", senderToken[:50]+"...")

	// Tokens should be different because they have different expiry times
	if listenerToken == senderToken {
		t.Error("Tokens with different expiry should be different")
	}
}

// TestIntegration_TokenExpiry demonstrates token expiry handling
func TestIntegration_TokenExpiry(t *testing.T) {
	relayNamespace := "test-relay"
	hybridConnectionName := "hc-12345"
	keyName := "RootManageSharedAccessKey"
	key := "dGVzdGtleQ=="

	// Generate token with short expiry
	token1, err := GenerateSASToken(
		relayNamespace,
		hybridConnectionName,
		keyName,
		key,
		1*time.Second,
	)
	if err != nil {
		t.Fatalf("Failed to generate token1: %v", err)
	}

	// Wait for token to expire
	time.Sleep(2 * time.Second)

	// Generate new token
	token2, err := GenerateSASToken(
		relayNamespace,
		hybridConnectionName,
		keyName,
		key,
		1*time.Second,
	)
	if err != nil {
		t.Fatalf("Failed to generate token2: %v", err)
	}

	// Tokens should be different because they were generated at different times
	if token1 == token2 {
		t.Error("Tokens generated at different times should be different")
	}

	t.Log("Token expiry test passed - tokens are properly time-based")
}
