package agentauth

import (
	"errors"
	"fmt"
)

// AgentAuthError represents a AgentAuth error
type AgentAuthError struct {
	Code    string
	Message string
}

func (e *AgentAuthError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Error definitions
var (
	ErrInvalidToken          = &AgentAuthError{Code: "invalid_token", Message: "Invalid token"}
	ErrUnauthorized          = &AgentAuthError{Code: "unauthorized", Message: "Unauthorized access"}
	ErrTokenExpired          = &AgentAuthError{Code: "token_expired", Message: "Token has expired"}
	ErrInvalidGrant          = &AgentAuthError{Code: "invalid_grant", Message: "Invalid authorization grant"}
	ErrInvalidClient         = &AgentAuthError{Code: "invalid_client", Message: "Invalid client credentials"}
	ErrStrictAuthKeyRequired = errors.New("strict auth mode requires explicit SigningKey of >=32 bytes")
)
