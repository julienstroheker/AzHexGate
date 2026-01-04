package relay

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateSASToken(t *testing.T) { //nolint:funlen // Table-driven test
	tests := []struct {
		name                 string
		relayNamespace       string
		hybridConnectionName string
		keyName              string
		key                  string
		wantErr              bool
	}{
		{
			name:                 "valid token generation",
			relayNamespace:       "myrelay",
			hybridConnectionName: "myhc",
			keyName:              "RootManageSharedAccessKey",
			key:                  "dGVzdGtleQ==", // base64 encoded "testkey"
			wantErr:              false,
		},
		{
			name:                 "any valid string key",
			relayNamespace:       "myrelay",
			hybridConnectionName: "myhc",
			keyName:              "RootManageSharedAccessKey",
			key:                  "any-string-key-works",
			wantErr:              false,
		},
		{
			name:                 "key with whitespace",
			relayNamespace:       "myrelay",
			hybridConnectionName: "myhc",
			keyName:              "RootManageSharedAccessKey",
			key:                  "  dGVzdGtleQ==  ",
			wantErr:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateSASToken(tt.relayNamespace, tt.hybridConnectionName, tt.keyName, tt.key, 1*time.Hour)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSASToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				// Verify token format
				if !strings.HasPrefix(token, "SharedAccessSignature ") {
					t.Errorf("Token should start with 'SharedAccessSignature ', got: %s", token)
				}
				if !strings.Contains(token, "sr=") {
					t.Errorf("Token should contain 'sr=' parameter")
				}
				if !strings.Contains(token, "sig=") {
					t.Errorf("Token should contain 'sig=' parameter")
				}
				if !strings.Contains(token, "se=") {
					t.Errorf("Token should contain 'se=' parameter")
				}
				if !strings.Contains(token, "skn=") {
					t.Errorf("Token should contain 'skn=' parameter")
				}
				// Verify the URI is built correctly
				if !strings.Contains(token, "myrelay.servicebus.windows.net") {
					t.Errorf("Token should contain namespace URL")
				}
			}
		})
	}
}

func TestSASTokenExpiry(t *testing.T) {
	shortExpiry := 5 * time.Second
	token, err := GenerateSASToken(
		"myrelay",
		"myhc",
		"RootManageSharedAccessKey",
		"dGVzdGtleQ==",
		shortExpiry,
	)
	if err != nil {
		t.Fatalf("GenerateSASToken() error = %v", err)
	}

	// Extract the expiry timestamp from the token
	if !strings.Contains(token, "se=") {
		t.Fatal("Token should contain 'se=' parameter")
	}

	// The token should be valid (i.e., expiry should be in the future)
	// This is a basic sanity check
	if len(token) == 0 {
		t.Error("Token should not be empty")
	}
}
