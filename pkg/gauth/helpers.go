package gauth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Helper functions for grant generation and validation

// generateGrantID creates a cryptographically secure random grant ID
func generateGrantID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("failed to generate random grant ID: %v", err))
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// validateScope checks if all requested scopes are allowed
//
//nolint:unused // reserved for OAuth2 scope validation
func validateScope(requested, allowed []string) bool {
	if len(requested) == 0 {
		return false
	}

	allowedMap := make(map[string]bool)
	for _, scope := range allowed {
		allowedMap[scope] = true
	}

	for _, scope := range requested {
		if !allowedMap[scope] {
			return false
		}
	}

	return true
}

// validateRedirectURI validates that the redirect URI is allowed for the client
//
//nolint:unused // reserved for OAuth2 redirect URI validation
func validateRedirectURI(redirectURI string, allowedURIs []string) bool {
	if redirectURI == "" || len(allowedURIs) == 0 {
		return false
	}

	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return false
	}

	// Normalize the URI by removing the fragment
	parsed.Fragment = ""
	normalizedURI := parsed.String()

	for _, allowed := range allowedURIs {
		parsedAllowed, err := url.Parse(allowed)
		if err != nil {
			continue
		}

		// Normalize the allowed URI
		parsedAllowed.Fragment = ""
		normalizedAllowed := parsedAllowed.String()

		if strings.HasSuffix(normalizedAllowed, "*") {
			// Wildcard matching
			prefix := normalizedAllowed[:len(normalizedAllowed)-1]
			if strings.HasPrefix(normalizedURI, prefix) {
				return true
			}
		} else if normalizedURI == normalizedAllowed {
			// Exact match
			return true
		}
	}

	return false
}

// Helper functions for response generation

// generateError creates a standardized error response for internal use only.
// NOTE: map[string]interface{} is used here only for error response formatting (not public API).
// All public APIs use type-safe alternatives. Do not expose in new APIs.
//
//nolint:unused // reserved for OAuth2 error responses
func generateError(code string, description string) map[string]interface{} {
	return map[string]interface{}{
		"error":             code,
		"error_description": description,
	}
}

// sanitizeScope removes any invalid characters from scope strings
//
//nolint:unused // reserved for scope sanitization
func sanitizeScope(scope string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= 'A' && r <= 'Z':
			return r
		case r >= '0' && r <= '9':
			return r
		case r == '_' || r == '.' || r == '-':
			return r
		default:
			return -1
		}
	}, scope)
}

// Helper functions for security

// validateClientCredentials validates client credentials securely
//
//nolint:unused // reserved for client credential validation
func validateClientCredentials(providedSecret, storedHash string) bool {
	// Use constant time comparison to prevent timing attacks
	if len(providedSecret) != len(storedHash) {
		return false
	}
	// Use crypto/subtle for constant time comparison
	return subtle.ConstantTimeCompare([]byte(providedSecret), []byte(storedHash)) == 1
}

// sanitizeRedirectURI sanitizes and validates a redirect URI
//
//nolint:unused // reserved for redirect URI sanitization
func sanitizeRedirectURI(uri string) (string, error) {
	if uri == "" {
		return "", fmt.Errorf("empty redirect URI")
	}

	parsed, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid redirect URI: %w", err)
	}

	// Ensure the scheme is either http or https
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("invalid URI scheme: must be http or https")
	}

	// Remove any fragments
	parsed.Fragment = ""

	return parsed.String(), nil
}

// Helper functions for token management

// isTokenExpired checks if a token has expired with a safety margin
//
//nolint:unused // reserved for token expiry checking
func isTokenExpired(expiryTime int64, safetyMargin int64) bool {
	now := TimeNow().Unix()
	return now >= (expiryTime - safetyMargin)
}

// TimeNow is a replaceable function to get the current time (useful for testing)
var TimeNow = func() TimeFunc {
	return timeNow{}
}

type TimeFunc interface {
	Unix() int64
}

type timeNow struct{}

func (timeNow) Unix() int64 {
	return time.Now().Unix()
}
