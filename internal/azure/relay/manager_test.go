package relay

import (
	"testing"
)

func TestNewManager(t *testing.T) {
	tests := []struct {
		name    string
		opts    *ManagerOptions
		wantErr bool
	}{
		{
			name:    "nil options",
			opts:    nil,
			wantErr: true,
		},
		{
			name: "missing subscription ID",
			opts: &ManagerOptions{
				ResourceGroupName: "test-rg",
				NamespaceName:     "test-ns",
			},
			wantErr: true,
		},
		{
			name: "missing resource group",
			opts: &ManagerOptions{
				SubscriptionID: "test-sub",
				NamespaceName:  "test-ns",
			},
			wantErr: true,
		},
		{
			name: "missing namespace name",
			opts: &ManagerOptions{
				SubscriptionID:    "test-sub",
				ResourceGroupName: "test-rg",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewManager(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
