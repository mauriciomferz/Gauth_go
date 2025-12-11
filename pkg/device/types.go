// Package device implements OAuth 2.0 Device Authorization Grant (RFC 8628).
package device

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// DeviceAuthRequest represents a device authorization request per RFC 8628 §3.1.
type DeviceAuthRequest struct {
	ClientID string `json:"client_id"`
	Scope    string `json:"scope,omitempty"`
}

// DeviceAuthResponse represents a device authorization response per RFC 8628 §3.2.
type DeviceAuthResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval,omitempty"` // Polling interval in seconds
}

// DeviceCodeStatus represents the status of a device authorization.
type DeviceCodeStatus string

const (
	StatusPending              DeviceCodeStatus = "pending"
	StatusAuthorized           DeviceCodeStatus = "authorized"
	StatusDenied               DeviceCodeStatus = "denied"
	StatusExpired              DeviceCodeStatus = "expired"
	StatusAuthorizationPending DeviceCodeStatus = "authorization_pending" // RFC 8628 error
	StatusSlowDown             DeviceCodeStatus = "slow_down"             // RFC 8628 error
)

// DeviceCode represents a device authorization in progress.
type DeviceCode struct {
	DeviceCode string           `json:"device_code"`
	UserCode   string           `json:"user_code"`
	ClientID   string           `json:"client_id"`
	Scope      string           `json:"scope,omitempty"`
	Status     DeviceCodeStatus `json:"status"`
	UserID     string           `json:"user_id,omitempty"` // Set when authorized
	CreatedAt  time.Time        `json:"created_at"`
	ExpiresAt  time.Time        `json:"expires_at"`
	LastPollAt time.Time        `json:"last_poll_at,omitempty"`
	Interval   int              `json:"interval"` // Seconds between polls
}

// IsExpired returns true if the device code has expired.
func (d *DeviceCode) IsExpired() bool {
	return time.Now().After(d.ExpiresAt)
}

// CanPoll returns true if enough time has passed since the last poll.
func (d *DeviceCode) CanPoll() bool {
	if d.LastPollAt.IsZero() {
		return true
	}
	return time.Since(d.LastPollAt) >= time.Duration(d.Interval)*time.Second
}

// TokenRequest represents a token request for device flow per RFC 8628 §3.4.
type TokenRequest struct {
	GrantType           string `json:"grant_type"` // "urn:ietf:params:oauth:grant-type:device_code"
	DeviceCode          string `json:"device_code"`
	ClientID            string `json:"client_id"`
	ClientAssertion     string `json:"client_assertion,omitempty"`
	ClientAssertionType string `json:"client_assertion_type,omitempty"`
}

// TokenResponse represents a successful token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// ErrorResponse represents an error response per RFC 8628 §3.5.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// GenerateDeviceCode generates a random device code.
func GenerateDeviceCode() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// GenerateUserCode generates a user-friendly code (XXXX-XXXX format).
func GenerateUserCode() string {
	const chars = "BCDFGHJKLMNPQRSTVWXZ" // Avoid confusing chars (I, O, etc.)
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	code := make([]byte, 9) // 8 chars + 1 dash
	for i := 0; i < 4; i++ {
		code[i] = chars[int(b[i])%len(chars)]
	}
	code[4] = '-'
	for i := 5; i < 9; i++ {
		code[i] = chars[int(b[i-1])%len(chars)]
	}
	return string(code)
}

// RFC 8628 error codes
const (
	ErrorAuthorizationPending = "authorization_pending"
	ErrorSlowDown             = "slow_down"
	ErrorAccessDenied         = "access_denied"
	ErrorExpiredToken         = "expired_token"
)

// DeviceFlowGrantType is the grant type for device authorization.
const DeviceFlowGrantType = "urn:ietf:params:oauth:grant-type:device_code"
