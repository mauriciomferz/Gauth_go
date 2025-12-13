package gauth

import (
	"context"
	"fmt"
	"time"
)

// ResourceServer represents a resource server
type ResourceServer struct {
	name    string
	service *Service
}

// NewResourceServer creates a new resource server
func NewResourceServer(name string, service *Service) *ResourceServer {
	return &ResourceServer{
		name:    name,
		service: service,
	}
}

// ProcessTransaction processes a transaction
func (rs *ResourceServer) ProcessTransaction(transaction TransactionDetails, token string) (string, error) {
	// Validate token first
	if _, err := rs.service.ValidateToken(token); err != nil {
		return "", err
	}

	// Simulate transaction processing
	return "Transaction processed successfully", nil
}

// RateLimit represents rate limit configuration
type RateLimit struct {
	RequestsPerSecond int
	BurstSize         int
	Window            time.Duration
}

// SetRateLimit sets rate limiting for the resource server with multiple parameter support
func (rs *ResourceServer) SetRateLimit(args ...interface{}) {
	// Handle different argument patterns for backwards compatibility
	if len(args) == 2 {
		// SetRateLimit(100, time.Second) pattern
		if rps, ok := args[0].(int); ok {
			if window, ok := args[1].(time.Duration); ok {
				// Create rate limit config
				_ = RateLimit{
					RequestsPerSecond: rps,
					Window:            window,
				}
			}
		}
	} else if len(args) == 1 {
		// SetRateLimit(interface{}) pattern
		_ = args[0]
	}
	// Placeholder implementation - store the rate limit config if needed
}

// ValidateExtendedTokenWithRAR validates an extended token and checks RAR permissions
func (rs *ResourceServer) ValidateExtendedTokenWithRAR(ctx context.Context, tokenString string, resource string, action string) error {
	if rs.service == nil || rs.service.protocolOrchestrator == nil || rs.service.protocolOrchestrator.extendedTokenService == nil {
		return fmt.Errorf("service not fully initialized")
	}

	ets := rs.service.protocolOrchestrator.extendedTokenService

	// Validate token
	result, err := ets.ValidateExtendedToken(ctx, tokenString)
	if err != nil {
		return fmt.Errorf("invalid token: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("token invalid")
	}

	// Check RAR
	details := result.ExtendedToken.AuthorizationDetails
	if len(details) == 0 {
		return fmt.Errorf("no authorization details found in token")
	}

	rarValidator := NewRARValidator()
	return rarValidator.EvaluateAccess(details, resource, action)
}
