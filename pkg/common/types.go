// Package common provides common types and utilities for the GAuth implementation
package common

import (
	"time"
)

// EventType represents different types of events in the GAuth system
type EventType string

const (
	// TokenCreated represents a token creation event
	TokenCreated EventType = "token.created"
	// TokenRevoked represents a token revocation event
	TokenRevoked EventType = "token.revoked"
	// TokenValidated represents a token validation event
	TokenValidated EventType = "token.validated"
	// AuthorizationGranted represents an authorization grant event
	AuthorizationGranted EventType = "authorization.granted"
	// AuthorizationDenied represents an authorization denial event
	AuthorizationDenied EventType = "authorization.denied"

	// Delegation events (RFC 0111)
	DelegationCreated   EventType = "delegation.created"
	DelegationRevoked   EventType = "delegation.revoked"
	DelegationValidated EventType = "delegation.validated"
)

// DefaultRateLimitConfig returns the default rate limiting configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 100,
		BurstSize:         200,
		WindowSize:        int(time.Minute / time.Second), // convert to seconds for int field
	}
}

// ValidationResult represents the result of a validation operation
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Reason string            `json:"reason,omitempty"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// IsValid returns true if the validation result is valid
func (vr ValidationResult) IsValid() bool {
	return vr.Valid && len(vr.Errors) == 0
}

// AddError adds a validation error to the result
func (vr *ValidationResult) AddError(field, message string) {
	vr.Valid = false
	vr.Errors = append(vr.Errors, ValidationError{
		Field:   field,
		Message: message,
	})
}
