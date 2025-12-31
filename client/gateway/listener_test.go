package gateway

import (
	"context"
	"testing"

	"github.com/julienstroheker/AzHexGate/internal/api"
	"github.com/julienstroheker/AzHexGate/internal/config"
	"github.com/julienstroheker/AzHexGate/internal/logging"
)

func TestStartListeningRemoteMode(t *testing.T) {
	logger := logging.New(logging.DebugLevel)

	// Create client with remote mode
	client := &Client{
		mode: config.ModeRemote,
	}

	ctx := context.Background()
	tunnelResp := &api.TunnelResponse{
		HybridConnectionName: "hc-test",
	}

	err := client.StartListening(ctx, logger, 3000, tunnelResp)

	// Should return "not yet implemented" error
	if err == nil {
		t.Fatal("Expected error for remote mode, got nil")
	}

	expectedMsg := "remote mode listening not yet implemented"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

// Note: Testing local mode listening requires a running gateway server
// Integration tests should cover the full local mode flow
