package gauth

import (
	"crypto/rand"
	"encoding/hex"
)

// generateToken creates a random token string for demonstration/testing purposes.
func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
