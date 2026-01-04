package relay

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// GenerateSASToken generates a Shared Access Signature token for Azure Relay Hybrid Connections.
// The token format is the same for both Listener and Sender connections - the access rights
// are determined by the key used, not by the token format itself.
//
// Parameters:
//   - relayNamespace: The Azure Relay namespace name (e.g., "myrelay")
//   - hybridConnectionName: The Hybrid Connection name (e.g., "hc-12345")
//   - keyName: The name of the shared access key (e.g., "RootManageSharedAccessKey")
//   - key: The shared access key value (as copied from Azure Portal)
//   - expiry: How long the token should be valid
//
// Returns a SAS token string that can be used in the ServiceBusAuthorization header.
func GenerateSASToken(relayNamespace, hybridConnectionName, keyName, key string, expiry time.Duration) (string, error) {
	// Build the resource URI for the hybrid connection
	// Format: https://<namespace>.servicebus.windows.net/<hybridConnectionName>
	uri := fmt.Sprintf("https://%s.servicebus.windows.net/%s", relayNamespace, hybridConnectionName)

	return generateSASTokenFromURI(uri, keyName, key, expiry)
}

// generateSASTokenFromURI is the internal implementation that generates a SAS token from a full URI.
// This is kept for flexibility if we need to generate tokens for other resource types in the future.
func generateSASTokenFromURI(uri, keyName, key string, expiry time.Duration) (string, error) {
	// 1. Ensure URI is properly formatted and lowercased
	uri = strings.TrimSuffix(uri, "/")
	uri = strings.ToLower(uri)

	// 2. Trim any whitespace from the key (common issue when copying from portal)
	key = strings.TrimSpace(key)

	// 3. Calculate expiry timestamp (seconds since Unix epoch)
	expiryTimestamp := time.Now().Add(expiry).Unix()
	expiryStr := strconv.FormatInt(expiryTimestamp, 10)

	// 4. URL-encode the URI using standard library
	encodedURI := url.QueryEscape(uri)

	// 5. Create the string to sign: <url-encoded-uri>\n<expiry>
	stringToSign := encodedURI + "\n" + expiryStr

	// 6. Use the key AS IS (do NOT decode base64)
	// The key from Azure Portal is already the raw key string
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	// 7. Build the SAS token with URL-encoded signature
	// Format: SharedAccessSignature sr=<url-encoded-uri>&sig=<url-encoded-sig>&se=<expiry>&skn=<keyname>
	token := fmt.Sprintf("SharedAccessSignature sr=%s&sig=%s&se=%s&skn=%s",
		encodedURI,
		url.QueryEscape(signature), // Must escape the signature!
		expiryStr,
		keyName,
	)

	return token, nil
}
