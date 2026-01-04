package relay

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/relay/armrelay"
)

// Manager handles Azure Relay management operations
type Manager struct {
	client            *armrelay.HybridConnectionsClient
	subscriptionID    string
	resourceGroupName string
	namespaceName     string
}

// ManagerOptions contains configuration for the Relay Manager
type ManagerOptions struct {
	// SubscriptionID is the Azure subscription ID
	SubscriptionID string

	// ResourceGroupName is the name of the resource group containing the Relay namespace
	ResourceGroupName string

	// NamespaceName is the name of the Azure Relay namespace
	NamespaceName string

	// Credential is the Azure credential to use (optional, defaults to DefaultAzureCredential)
	Credential azcore.TokenCredential
}

// NewManager creates a new Azure Relay Manager
func NewManager(opts *ManagerOptions) (*Manager, error) {
	if opts == nil {
		return nil, fmt.Errorf("options cannot be nil")
	}
	if opts.SubscriptionID == "" {
		return nil, fmt.Errorf("subscription ID is required")
	}
	if opts.ResourceGroupName == "" {
		return nil, fmt.Errorf("resource group name is required")
	}
	if opts.NamespaceName == "" {
		return nil, fmt.Errorf("namespace name is required")
	}

	credential := opts.Credential
	if credential == nil {
		// Use DefaultAzureCredential if no credential provided
		cred, err := azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create default credential: %w", err)
		}
		credential = cred
	}

	hcClient, err := armrelay.NewHybridConnectionsClient(opts.SubscriptionID, credential, &arm.ClientOptions{
		ClientOptions: policy.ClientOptions{Logging: policy.LogOptions{IncludeBody: true}},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create hybrid connections client: %w", err)
	}

	return &Manager{
		client:            hcClient,
		subscriptionID:    opts.SubscriptionID,
		resourceGroupName: opts.ResourceGroupName,
		namespaceName:     opts.NamespaceName,
	}, nil
}

// CreateHybridConnection creates a new Hybrid Connection in the Relay namespace
func (m *Manager) CreateHybridConnection(ctx context.Context, name string) error {
	// Always create or update to ensure correct settings
	// Setting RequiresClientAuthorization to false allows namespace-level SAS tokens
	props := armrelay.HybridConnection{
		Properties: &armrelay.HybridConnectionProperties{
			RequiresClientAuthorization: ptr(false), // Allow namespace-level SAS tokens
		},
	}

	_, err := m.client.CreateOrUpdate(ctx, m.resourceGroupName, m.namespaceName, name, props, nil)
	if err != nil {
		return fmt.Errorf("failed to create hybrid connection: %w", err)
	}

	return nil
}

// DeleteHybridConnection deletes a Hybrid Connection from the Relay namespace
// TODO : Needs to add a story to handle cleanup of HCs
func (m *Manager) DeleteHybridConnection(ctx context.Context, name string) error {
	_, err := m.client.Delete(ctx, m.resourceGroupName, m.namespaceName, name, nil)
	if err != nil {
		return fmt.Errorf("failed to delete hybrid connection: %w", err)
	}
	return nil
}

// ptr is a helper function to get a pointer to a value
func ptr[T any](v T) *T {
	return &v
}
