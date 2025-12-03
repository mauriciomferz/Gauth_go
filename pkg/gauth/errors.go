package gauth

import (
	"errors"
	"fmt"
)

// GAuthError represents a GAuth error
type GAuthError struct {
	Code    string
	Message string
}

func (e *GAuthError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Error definitions
var (
	ErrInvalidToken          = &GAuthError{Code: "invalid_token", Message: "Invalid token"}
	ErrUnauthorized          = &GAuthError{Code: "unauthorized", Message: "Unauthorized access"}
	ErrTokenExpired          = &GAuthError{Code: "token_expired", Message: "Token has expired"}
	ErrInvalidGrant          = &GAuthError{Code: "invalid_grant", Message: "Invalid authorization grant"}
	ErrInvalidClient         = &GAuthError{Code: "invalid_client", Message: "Invalid client credentials"}
	ErrStrictAuthKeyRequired = errors.New("strict auth mode requires explicit SigningKey of >=32 bytes")
)
