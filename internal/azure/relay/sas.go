package relay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// GenerateSASToken generates a Shared Access Signature token for Azure Relay
// This token can be used by clients to authenticate with Azure Relay Hybrid Connections
func GenerateSASToken(uri, keyName, key string, expiry time.Duration) (string, error) {
	// Ensure URI is properly formatted (no trailing slash, lowercase scheme)
	uri = strings.TrimSuffix(uri, "/")

	// Calculate expiry timestamp (seconds since Unix epoch)
	expiryTimestamp := time.Now().Add(expiry).Unix()

	// Create the string to sign: <url>\n<expiry>
	stringToSign := fmt.Sprintf("%s\n%d", url.QueryEscape(uri), expiryTimestamp)

	// Decode the key from base64
	decodedKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("failed to decode key: %w", err)
	}

	// Create HMAC-SHA256 signature
	h := hmac.New(sha256.New, decodedKey)
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// Build the SAS token
	// Format: SharedAccessSignature sr=<url>&sig=<signature>&se=<expiry>&skn=<keyname>
	token := fmt.Sprintf("SharedAccessSignature sr=%s&sig=%s&se=%d&skn=%s",
		url.QueryEscape(uri),
		url.QueryEscape(signature),
		expiryTimestamp,
		url.QueryEscape(keyName),
	)

	return token, nil
}

// GenerateListenerSASToken generates a SAS token with Listen rights for a Hybrid Connection
func GenerateListenerSASToken(
	relayNamespace, hybridConnectionName, keyName, key string, expiry time.Duration,
) (string, error) {
	// Build the resource URI for the hybrid connection
	// Format: https://<namespace>.servicebus.windows.net/<hybridConnectionName>
	uri := fmt.Sprintf("https://%s.servicebus.windows.net/%s", relayNamespace, hybridConnectionName)

	return GenerateSASToken(uri, keyName, key, expiry)
}

// GenerateSenderSASToken generates a SAS token with Send rights for a Hybrid Connection
func GenerateSenderSASToken(
	relayNamespace, hybridConnectionName, keyName, key string, expiry time.Duration,
) (string, error) {
	// Build the resource URI for the hybrid connection
	// Format: https://<namespace>.servicebus.windows.net/<hybridConnectionName>
	uri := fmt.Sprintf("https://%s.servicebus.windows.net/%s", relayNamespace, hybridConnectionName)

	return GenerateSASToken(uri, keyName, key, expiry)
}
