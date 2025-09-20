package gauth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// generateGrantID creates a cryptographically secure random grant ID
func generateGrantID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random grant ID: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
