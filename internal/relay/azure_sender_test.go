package relay

import (
	"testing"
)

func TestNewAzureSender(t *testing.T) {
	tests := []struct {
		name    string
		opts    *AzureSenderOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &AzureSenderOptions{
				RelayEndpoint:        "myrelay.servicebus.windows.net",
				HybridConnectionName: "hc-12345",
				Token:                "test-token",
			},
			wantErr: false,
		},
		{
			name:    "nil options",
			opts:    nil,
			wantErr: true,
		},
		{
			name: "missing relay endpoint",
			opts: &AzureSenderOptions{
				HybridConnectionName: "hc-12345",
				Token:                "test-token",
			},
			wantErr: true,
		},
		{
			name: "missing hybrid connection name",
			opts: &AzureSenderOptions{
				RelayEndpoint: "myrelay.servicebus.windows.net",
				Token:         "test-token",
			},
			wantErr: true,
		},
		{
			name: "missing token",
			opts: &AzureSenderOptions{
				RelayEndpoint:        "myrelay.servicebus.windows.net",
				HybridConnectionName: "hc-12345",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender, err := NewAzureSender(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAzureSender() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && sender == nil {
				t.Error("NewAzureSender() returned nil sender")
			}
		})
	}
}

func TestAzureSender_Close(t *testing.T) {
	sender, err := NewAzureSender(&AzureSenderOptions{
		RelayEndpoint:        "myrelay.servicebus.windows.net",
		HybridConnectionName: "hc-12345",
		Token:                "test-token",
	})
	if err != nil {
		t.Fatalf("NewAzureSender() error = %v", err)
	}

	// Should be able to close multiple times without error
	if err := sender.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if err := sender.Close(); err != nil {
		t.Errorf("Close() second time error = %v", err)
	}
}
