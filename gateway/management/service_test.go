package management

import (
	"strings"
	"testing"
)

func TestNewService(t *testing.T) {
	tests := []struct {
		name    string
		opts    *Options
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &Options{
				RelayNamespace: "myrelay",
				RelayKeyName:   "RootManageSharedAccessKey",
				RelayKey:       "dGVzdGtleQ==",
				BaseDomain:     "azhexgate.com",
			},
			wantErr: false,
		},
		{
			name:    "nil options",
			opts:    nil,
			wantErr: true,
		},
		{
			name: "missing relay namespace",
			opts: &Options{
				RelayKeyName: "RootManageSharedAccessKey",
				RelayKey:     "dGVzdGtleQ==",
			},
			wantErr: true,
		},
		{
			name: "missing relay key name",
			opts: &Options{
				RelayNamespace: "myrelay",
				RelayKey:       "dGVzdGtleQ==",
			},
			wantErr: true,
		},
		{
			name: "missing relay key",
			opts: &Options{
				RelayNamespace: "myrelay",
				RelayKeyName:   "RootManageSharedAccessKey",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewService(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && service == nil {
				t.Error("NewService() returned nil service")
			}
		})
	}
}

func TestService_CreateTunnel(t *testing.T) {
	service, err := NewService(&Options{
		RelayNamespace: "myrelay",
		RelayKeyName:   "RootManageSharedAccessKey",
		RelayKey:       "dGVzdGtleQ==",
		BaseDomain:     "azhexgate.com",
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	response, err := service.CreateTunnel(3000)
	if err != nil {
		t.Fatalf("CreateTunnel() error = %v", err)
	}

	// Verify response structure
	if response.PublicURL == "" {
		t.Error("PublicURL should not be empty")
	}
	if !strings.HasSuffix(response.PublicURL, ".azhexgate.com") {
		t.Errorf("PublicURL should end with .azhexgate.com, got: %s", response.PublicURL)
	}
	if !strings.HasPrefix(response.PublicURL, "https://") {
		t.Errorf("PublicURL should start with https://, got: %s", response.PublicURL)
	}

	if response.RelayEndpoint == "" {
		t.Error("RelayEndpoint should not be empty")
	}
	if !strings.HasSuffix(response.RelayEndpoint, ".servicebus.windows.net") {
		t.Errorf("RelayEndpoint should end with .servicebus.windows.net, got: %s", response.RelayEndpoint)
	}

	if response.HybridConnectionName == "" {
		t.Error("HybridConnectionName should not be empty")
	}
	if !strings.HasPrefix(response.HybridConnectionName, "hc-") {
		t.Errorf("HybridConnectionName should start with hc-, got: %s", response.HybridConnectionName)
	}

	if response.ListenerToken == "" {
		t.Error("ListenerToken should not be empty")
	}
	if !strings.HasPrefix(response.ListenerToken, "SharedAccessSignature ") {
		t.Errorf("ListenerToken should start with 'SharedAccessSignature ', got: %s", response.ListenerToken)
	}

	if response.SessionID == "" {
		t.Error("SessionID should not be empty")
	}
}

func TestGenerateSubdomainID(t *testing.T) {
	// Test that subdomain IDs are unique
	id1 := generateSubdomainID()
	id2 := generateSubdomainID()

	if id1 == id2 {
		t.Error("Expected unique subdomain IDs")
	}

	if len(id1) != 8 {
		t.Errorf("Expected subdomain ID length of 8, got: %d", len(id1))
	}

	if len(id2) != 8 {
		t.Errorf("Expected subdomain ID length of 8, got: %d", len(id2))
	}
}
