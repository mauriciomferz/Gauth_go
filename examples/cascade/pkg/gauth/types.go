// Package gauth provides the core functionality for the GAuth authentication and authorization system.
package gauth

import "time"

// EventType represents the type of audit event
type EventType int

const (
	EventAuthRequest EventType = iota
	EventAuthGrant
	EventTokenIssue
	EventTokenRevoke
	EventTransactionStart
	EventTransactionComplete
	EventTransactionFailed
	EventRateLimited
)

// String returns the string representation of an EventType
func (e EventType) String() string {
	return [...]string{
		"auth_request",
		"auth_grant",
		"token_issue",
		"token_revoke",
		"transaction_start",
		"transaction_complete",
		"transaction_failed",
		"rate_limited",
	}[e]
}

// AuthorizationRequest represents a request for authorization
type AuthorizationRequest struct {
	ClientID        string
	ClientOwnerID   string
	ResourceOwnerID string
	RequestDetails  string
	Scopes          []string
	Timestamp       int64
}

// AuthorizationGrant represents the granted authorization
type AuthorizationGrant struct {
	GrantID      string
	ClientID     string
	Scope        []string
	Restrictions []Restriction
	ValidUntil   time.Time
}

// TokenRequest represents a request for a token
type TokenRequest struct {
	GrantID      string
	Scope        []string
	Restrictions []Restriction
}

// TokenResponse represents the response to a token request
type TokenResponse struct {
	Token        string
	ValidUntil   time.Time
	Scope        []string
	Restrictions []Restriction
}

// SUPER ULTIMATE NUCLEAR SOLUTION: ZERO TransactionDetails references in types.go
// TransactionDetails is EXCLUSIVELY defined in transaction.go - NO references here
// This completely eliminates any possibility of redeclaration errors

//go:generate echo "SUPER ULTIMATE FIX: NO TransactionDetails in types.go - ZERO redeclaration risk"

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int `json:"requests_per_second"` // Maximum requests per second
	BurstSize         int `json:"burst_size"`          // Maximum burst size
	WindowSize        int `json:"window_size"`         // Time window in seconds
}

// Config represents the configuration for GAuth
type Config struct {
	AuthServerURL     string          // URL of the authorization server
	ClientID          string          // Client identifier
	ClientSecret      string          // Client secret
	Scopes            []string        // Default scopes
	RateLimit         RateLimitConfig // Rate limiting configuration
	AccessTokenExpiry time.Duration   // Token expiry duration
}
