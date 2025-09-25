package gauth

import (
	"fmt"
)

// ResourceServer represents a server that provides protected resources
type ResourceServer struct {
	name string
	auth *GAuth
}

// NewResourceServer creates a new resource server instance
func NewResourceServer(name string, auth *GAuth) *ResourceServer {
	return &ResourceServer{
		name: name,
		auth: auth,
	}
}

// ProcessTransaction processes a transaction with the given token
func (s *ResourceServer) ProcessTransaction(tx TransactionDetails, token string) (string, error) {
	// NUCLEAR SOLUTION: Direct token validation to bypass any type alias issues
	var tokenData *TokenResponse
	var err error
	
	// Since GAuth = ServiceAuth, we can directly cast and call the method
	serviceAuth := (*ServiceAuth)(s.auth)
	tokenData, err = serviceAuth.ValidateToken(token)
	
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
