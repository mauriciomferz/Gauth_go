package gauth

import (
	"context"
	"time"
)

// PowerEnforcementPoint represents the enforcement point in the GAuth protocol
type PowerEnforcementPoint struct {
	GAuth *GAuth
}

// PowerDecisionPoint represents the decision point in the GAuth protocol
type PowerDecisionPoint struct {
	GAuth *GAuth
}

// PowerInformationPoint represents the information point in the GAuth protocol
type PowerInformationPoint struct {
	GAuth *GAuth
}

// PowerAdministrationPoint represents the administration point in the GAuth protocol
type PowerAdministrationPoint struct {
	GAuth *GAuth
}

// PowerVerificationPoint represents the verification point in the GAuth protocol
type PowerVerificationPoint struct {
	GAuth *GAuth
}

// TokenType represents the type of token issued
// (access_token, refresh_token, etc.)
type TokenType string

const (
	// AccessToken represents a short-lived token for resource access
	AccessToken TokenType = "access_token"
	// RefreshToken represents a long-lived token for obtaining new access tokens
	RefreshToken TokenType = "refresh_token"
)

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

// AuditEventType represents the type of audit event recorded
type AuditEventType string

const (
	// AuditEventAuthSuccess represents a successful authentication
	AuditEventAuthSuccess AuditEventType = "auth_success"
	// AuditEventAuthFailure represents a failed authentication
	AuditEventAuthFailure AuditEventType = "auth_failure"
	// AuditEventTokenIssued represents a token issuance event
	AuditEventTokenIssued AuditEventType = "token_issued"
	// AuditEventTokenRevoked represents a token revocation event
	AuditEventTokenRevoked AuditEventType = "token_revoked"
	// AuditEventTransactionStarted represents a transaction start event
	AuditEventTransactionStarted AuditEventType = "transaction_started"
	// AuditEventTransactionCompleted represents a transaction completion event
	AuditEventTransactionCompleted AuditEventType = "transaction_completed"
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
	Context      context.Context
}

// TokenResponse represents the response to a token request
type TokenResponse struct {
	Token        string
	ValidUntil   time.Time
	Scope        []string
	Restrictions []Restriction
}

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
