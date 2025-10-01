package gauth

import (
	"fmt"
)

// ResourceServer represents a server that provides protected resources
// SUPER ULTIMATE FIX: Use interface directly, not pointer to interface
type ResourceServer struct {
	name string
	auth GAuth // Direct interface - NOT pointer to interface
}

// NewResourceServer creates a new resource server instance
// SUPER ULTIMATE FIX: Accept interface directly and dereference if needed
func NewResourceServer(name string, auth GAuth) *ResourceServer {
	return &ResourceServer{
		name: name,
		auth: auth,
	}
}

// ProcessTransaction processes a transaction with the given token
func (s *ResourceServer) ProcessTransaction(tx TransactionDetails, token string) (string, error) {
	// SUPER ULTIMATE NUCLEAR SOLUTION: Direct interface method call
	// Since GAuth interface explicitly defines ValidateToken, this MUST work
	tokenData, err := s.auth.ValidateToken(token)

	if err != nil {
		return "", err
	}

	// Check if token has required scope
	hasScope := false
	for _, scope := range tokenData.Scope {
		if scope == "transaction:execute" {
			hasScope = true
			break
		}
	}
	if !hasScope {
		return "", fmt.Errorf("insufficient scope: token lacks transaction:execute scope")
	}

	// In a real implementation, this would process the transaction
	// For this example, we just return a success message
	return "Transaction processed successfully", nil
}
