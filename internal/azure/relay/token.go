package relay

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
)

// ManagedIdentityTokenProvider provides Azure AD tokens using Managed Identity
// It caches tokens and refreshes them proactively before expiry
type ManagedIdentityTokenProvider struct {
	credential azcore.TokenCredential
	scope      string
	mu         sync.RWMutex
	token      *azcore.AccessToken
}

// NewManagedIdentityTokenProvider creates a new token provider using Managed Identity
func NewManagedIdentityTokenProvider() (*ManagedIdentityTokenProvider, error) {
	// Create a DefaultAzureCredential which will try multiple authentication methods
	// including Managed Identity, Azure CLI, Environment variables, etc.
	credential, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	// Azure Service Bus / Relay scope for authentication
	scope := "https://servicebus.azure.net/.default"

	return &ManagedIdentityTokenProvider{
		credential: credential,
		scope:      scope,
	}, nil
}

// GetToken returns a valid Azure AD access token, using cache when possible
// It proactively refreshes the token if it's about to expire
func (p *ManagedIdentityTokenProvider) GetToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	if p.token != nil && time.Until(p.token.ExpiresOn) > 5*time.Minute {
		// Token is still valid with at least 5 minutes remaining
		token := p.token.Token
		p.mu.RUnlock()
		return token, nil
	}
	p.mu.RUnlock()

	// Need to refresh token
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if p.token != nil && time.Until(p.token.ExpiresOn) > 5*time.Minute {
		return p.token.Token, nil
	}

	// Request new token
	tokenResponse, err := p.credential.GetToken(ctx, policy.TokenRequestOptions{
		Scopes: []string{p.scope},
	})
	if err != nil {
		return "", fmt.Errorf("failed to get token: %w", err)
	}

	p.token = &tokenResponse
	return tokenResponse.Token, nil
}
