package tunnel

import (
	"context"
	"strings"
	"testing"

	"github.com/julienstroheker/AzHexGate/internal/config"
)

func TestNewManager(t *testing.T) {
	manager := NewManager(&Options{
		Mode: config.ModeLocal,
	})

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.mode != config.ModeLocal {
		t.Errorf("Expected mode to be local, got: %v", manager.mode)
	}
}

func TestManager_CreateTunnel_LocalMode(t *testing.T) {
	manager := NewManager(&Options{
		Mode: config.ModeLocal,
	})

	ctx := context.Background()
	resp, err := manager.CreateTunnel(ctx, 3000)
	if err != nil {
		t.Fatalf("CreateTunnel failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response to be non-nil")
	}

	if !strings.HasPrefix(resp.PublicURL, "http://localhost:8080/tunnel/") {
		t.Errorf("Expected local URL, got: %s", resp.PublicURL)
	}

	if resp.RelayEndpoint != "in-memory" {
		t.Errorf("Expected in-memory endpoint, got: %s", resp.RelayEndpoint)
	}

	if !strings.HasPrefix(resp.HybridConnectionName, "hc-") {
		t.Errorf("Expected hc- prefix, got: %s", resp.HybridConnectionName)
	}
}

func TestManager_CreateTunnel_RemoteMode(t *testing.T) {
	manager := NewManager(&Options{
		Mode: config.ModeRemote,
	})

	ctx := context.Background()
	resp, err := manager.CreateTunnel(ctx, 3000)
	if err != nil {
		t.Fatalf("CreateTunnel failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected response to be non-nil")
	}

	// Remote mode returns placeholder data for now
	if resp.PublicURL == "" {
		t.Error("Expected non-empty public URL")
	}
}

func TestManager_GetListener_LocalMode(t *testing.T) {
	manager := NewManager(&Options{
		Mode: config.ModeLocal,
	})

	ctx := context.Background()
	resp, err := manager.CreateTunnel(ctx, 3000)
	if err != nil {
		t.Fatalf("CreateTunnel failed: %v", err)
	}

	listener, err := manager.GetListener(resp.HybridConnectionName)
	if err != nil {
		t.Fatalf("GetListener failed: %v", err)
	}

	if listener == nil {
		t.Fatal("Expected listener to be non-nil")
	}

	if listener.Addr() != resp.HybridConnectionName {
		t.Errorf("Expected listener addr %s, got: %s", resp.HybridConnectionName, listener.Addr())
	}
}

func TestManager_GetListener_RemoteMode(t *testing.T) {
	manager := NewManager(&Options{
		Mode: config.ModeRemote,
	})

	_, err := manager.GetListener("hc-test")
	if err == nil {
		t.Error("Expected error for GetListener in remote mode")
	}
}

func TestManager_GetListener_NotFound(t *testing.T) {
	manager := NewManager(&Options{
		Mode: config.ModeLocal,
	})

	_, err := manager.GetListener("hc-nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent listener")
	}
}

func TestManager_GetSender_LocalMode(t *testing.T) {
	manager := NewManager(&Options{
		Mode: config.ModeLocal,
	})

	ctx := context.Background()
	resp, err := manager.CreateTunnel(ctx, 3000)
	if err != nil {
		t.Fatalf("CreateTunnel failed: %v", err)
	}

	sender, err := manager.GetSender(resp.HybridConnectionName)
	if err != nil {
		t.Fatalf("GetSender failed: %v", err)
	}

	if sender == nil {
		t.Fatal("Expected sender to be non-nil")
	}
}

func TestManager_GetSender_RemoteMode(t *testing.T) {
	manager := NewManager(&Options{
		Mode: config.ModeRemote,
	})

	_, err := manager.GetSender("hc-test")
	if err == nil {
		t.Error("Expected error for GetSender in remote mode")
	}
}
