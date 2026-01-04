package relay

import (
	"testing"
)

func TestNewAzureListener(t *testing.T) {
	tests := []struct {
		name    string
		opts    *AzureListenerOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: &AzureListenerOptions{
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
			opts: &AzureListenerOptions{
				HybridConnectionName: "hc-12345",
				Token:                "test-token",
			},
			wantErr: true,
		},
		{
			name: "missing hybrid connection name",
			opts: &AzureListenerOptions{
				RelayEndpoint: "myrelay.servicebus.windows.net",
				Token:         "test-token",
			},
			wantErr: true,
		},
		{
			name: "missing token",
			opts: &AzureListenerOptions{
				RelayEndpoint:        "myrelay.servicebus.windows.net",
				HybridConnectionName: "hc-12345",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := NewAzureListener(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAzureListener() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && listener == nil {
				t.Error("NewAzureListener() returned nil listener")
			}
		})
	}
}

func TestAzureListener_Close(t *testing.T) {
	listener, err := NewAzureListener(&AzureListenerOptions{
		RelayEndpoint:        "myrelay.servicebus.windows.net",
		HybridConnectionName: "hc-12345",
		Token:                "test-token",
	})
	if err != nil {
		t.Fatalf("NewAzureListener() error = %v", err)
	}

	// Should be able to close multiple times without error
	if err := listener.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if err := listener.Close(); err != nil {
		t.Errorf("Close() second time error = %v", err)
	}
}
