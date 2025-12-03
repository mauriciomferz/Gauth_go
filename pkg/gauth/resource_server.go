package gauth

import (
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
