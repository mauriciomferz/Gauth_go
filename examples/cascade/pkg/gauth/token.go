package gauth

import (
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
