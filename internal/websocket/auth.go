package websocket

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"spacelift-intent-mcp/internal"
)

// AuthConfig holds authentication configuration for WebSocket connections
type AuthConfig struct {
	SigningKey *rsa.PrivateKey
	ExecutorID string
}

// ConnectionParams holds parameters for establishing a WebSocket connection
type ConnectionParams struct {
	URL       string
	SessionID string
	Headers   http.Header
}

// AddExecutorCookies adds executor cookies to the cookiejar
func AddExecutorCookies(url *url.URL, jar *cookiejar.Jar, authConfig *AuthConfig) (*cookiejar.Jar, error) {
	if authConfig == nil || authConfig.SigningKey == nil {
		return jar, nil
	}

	signature, err := signExecutorID(authConfig.ExecutorID, authConfig.SigningKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signature: %w", err)
	}
	jar.SetCookies(url, []*http.Cookie{
		{
			Name:  internal.ExecutorSignatureCookieName,
			Value: signature,
		},
	})

	return jar, nil
}

func signExecutorID(executorID string, privateKey *rsa.PrivateKey) (string, error) {
	// Hash the executorID
	hash := sha256.Sum256([]byte(executorID))

	// Sign the hash with RSA-PSS
	signature, err := rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hash[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign executorID: %w", err)
	}

	return hex.EncodeToString(signature), nil
}
