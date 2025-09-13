package gauth

import (
	"crypto/rand"
	"encoding/base64"
	"time"
)

// TokenData represents metadata associated with a token
type TokenData struct {
	TokenID      string
	UserID       string
	ClientID     string
	Scopes       []string
	Restrictions []Restriction
	IssuedAt     time.Time
	ExpiresAt    time.Time
}

// generateGrantID creates a unique grant identifier

// generateToken creates a new secure token
func generateToken() (string, error) {
	b := make([]byte, 48)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
