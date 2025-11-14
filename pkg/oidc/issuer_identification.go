package oidc

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// RFC 9207: OAuth 2.0 Authorization Server Issuer Identification
//
// This implementation prevents mix-up attacks by ensuring that:
// 1. The authorization endpoint returns an 'iss' parameter in the response
// 2. The token endpoint validates that the 'iss' parameter matches the expected issuer
//
// This is particularly important in scenarios where:
// - Multiple authorization servers are used
// - Clients interact with multiple ASs
// - Authorization responses may be intercepted and replayed to different ASs

// IssuerIdentifier manages issuer identification for RFC 9207 compliance.
type IssuerIdentifier struct {
	issuerURL string // The issuer identifier (e.g., "https://auth.example.com")
}

// NewIssuerIdentifier creates a new issuer identifier manager.
//
// The issuerURL should be the canonical URL of the authorization server,
// matching the 'issuer' value in the discovery document.
//
// Example: "https://auth.example.com"
func NewIssuerIdentifier(issuerURL string) (*IssuerIdentifier, error) {
	if issuerURL == "" {
		return nil, fmt.Errorf("issuer URL cannot be empty")
	}

	// Validate issuer URL format
	parsed, err := url.Parse(issuerURL)
	if err != nil {
		return nil, fmt.Errorf("invalid issuer URL: %w", err)
	}

	// RFC 9207 requires HTTPS (except localhost for testing)
	if parsed.Scheme != "https" && !isLocalhost(parsed.Host) {
		return nil, fmt.Errorf("issuer URL must use HTTPS (got %s)", parsed.Scheme)
	}

	// Issuer URL must not contain query or fragment
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("issuer URL must not contain query or fragment")
	}

	return &IssuerIdentifier{
		issuerURL: strings.TrimSuffix(issuerURL, "/"),
	}, nil
}

// GetIssuer returns the issuer identifier.
func (ii *IssuerIdentifier) GetIssuer() string {
	return ii.issuerURL
}

// AddIssuerToAuthorizationResponse adds the 'iss' parameter to an authorization response.
//
// This should be called by the authorization endpoint before redirecting back to the client.
// The 'iss' parameter is added to the redirect URI as a query parameter.
//
// Parameters:
//   - ctx: Context for the operation
//   - redirectURI: The client's redirect URI (may already contain query parameters)
//
// Returns:
//   - string: The redirect URI with 'iss' parameter added
//   - error: If the redirect URI is invalid
//
// Example:
//
//	Input:  "https://client.example.com/callback?code=abc123"
//	Output: "https://client.example.com/callback?code=abc123&iss=https://auth.example.com"
func (ii *IssuerIdentifier) AddIssuerToAuthorizationResponse(ctx context.Context, redirectURI string) (string, error) {
	if redirectURI == "" {
		return "", fmt.Errorf("redirect URI cannot be empty")
	}

	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return "", fmt.Errorf("invalid redirect URI: %w", err)
	}

	// Parse existing query parameters
	query := parsed.Query()

	// Add 'iss' parameter (RFC 9207 Section 2.1)
	query.Set("iss", ii.issuerURL)

	// Reconstruct the URL with the new query parameters
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

// ValidateIssuerInTokenRequest validates the 'iss' parameter in a token request.
//
// This should be called by the token endpoint to ensure that the authorization
// response came from the expected authorization server.
//
// Parameters:
//   - ctx: Context for the operation
//   - issuerFromRequest: The 'iss' parameter received in the token request
//
// Returns:
//   - error: If the issuer doesn't match or is missing
//
// RFC 9207 Section 2.2 requires:
//   - The 'iss' parameter MUST be present
//   - The 'iss' value MUST match the expected issuer
//   - If validation fails, return 'invalid_request' error
func (ii *IssuerIdentifier) ValidateIssuerInTokenRequest(ctx context.Context, issuerFromRequest string) error {
	// Check if 'iss' parameter is present
	if issuerFromRequest == "" {
		return &OIDCError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: "missing 'iss' parameter (RFC 9207)",
		}
	}

	// Validate that 'iss' matches the expected issuer
	if issuerFromRequest != ii.issuerURL {
		return &OIDCError{
			ErrorCode:        ErrorInvalidRequest,
			ErrorDescription: fmt.Sprintf("issuer mismatch: expected '%s', got '%s'", ii.issuerURL, issuerFromRequest),
		}
	}

	return nil
}

// ExtractIssuerFromRedirect extracts the 'iss' parameter from a redirect URI.
//
// This is a utility function for clients to extract the issuer from the
// authorization response redirect.
//
// Parameters:
//   - redirectURI: The full redirect URI received by the client
//
// Returns:
//   - string: The issuer identifier
//   - error: If the redirect URI is invalid or 'iss' is missing
func ExtractIssuerFromRedirect(redirectURI string) (string, error) {
	if redirectURI == "" {
		return "", fmt.Errorf("redirect URI cannot be empty")
	}

	parsed, err := url.Parse(redirectURI)
	if err != nil {
		return "", fmt.Errorf("invalid redirect URI: %w", err)
	}

	// Extract 'iss' parameter
	issuer := parsed.Query().Get("iss")
	if issuer == "" {
		return "", fmt.Errorf("missing 'iss' parameter in redirect URI")
	}

	return issuer, nil
}

// isLocalhost checks if a host is localhost (for testing purposes).
func isLocalhost(host string) bool {
	// Remove port if present
	if idx := strings.Index(host, ":"); idx >= 0 {
		host = host[:idx]
	}

	return host == "localhost" ||
		host == "127.0.0.1" ||
		host == "::1" ||
		host == "[::1]"
}

// IssuerIdentificationMiddleware is HTTP middleware that validates issuer identification
// for token requests.
//
// This middleware should be applied to the token endpoint to enforce RFC 9207 compliance.
// It extracts the 'iss' parameter from the request and validates it against the expected issuer.
type IssuerIdentificationMiddleware struct {
	identifier *IssuerIdentifier
	enabled    bool // Allow disabling for backward compatibility
}

// NewIssuerIdentificationMiddleware creates a new issuer identification middleware.
func NewIssuerIdentificationMiddleware(identifier *IssuerIdentifier, enabled bool) *IssuerIdentificationMiddleware {
	return &IssuerIdentificationMiddleware{
		identifier: identifier,
		enabled:    enabled,
	}
}

// ValidateRequest validates the issuer in an HTTP request.
//
// This extracts the 'iss' parameter from the request (form or query) and validates it.
func (m *IssuerIdentificationMiddleware) ValidateRequest(ctx context.Context, issuerParam string) error {
	// If disabled, skip validation (for backward compatibility)
	if !m.enabled {
		return nil
	}

	// Validate issuer
	return m.identifier.ValidateIssuerInTokenRequest(ctx, issuerParam)
}
