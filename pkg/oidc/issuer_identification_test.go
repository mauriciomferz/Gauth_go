package oidc

import (
	"context"
	"strings"
	"testing"
)

// TestNewIssuerIdentifier tests issuer identifier creation.
func TestNewIssuerIdentifier(t *testing.T) {
	tests := []struct {
		name      string
		issuerURL string
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "valid HTTPS issuer",
			issuerURL: "https://auth.example.com",
			wantErr:   false,
		},
		{
			name:      "valid HTTPS issuer with path",
			issuerURL: "https://auth.example.com/oauth2",
			wantErr:   false,
		},
		{
			name:      "valid localhost HTTP (for testing)",
			issuerURL: "http://localhost:8080",
			wantErr:   false,
		},
		{
			name:      "valid 127.0.0.1 HTTP (for testing)",
			issuerURL: "http://127.0.0.1:8080",
			wantErr:   false,
		},
		{
			name:      "empty issuer URL",
			issuerURL: "",
			wantErr:   true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "invalid URL",
			issuerURL: "not a valid url",
			wantErr:   true,
			errMsg:    "must use HTTPS",
		},
		{
			name:      "HTTP non-localhost",
			issuerURL: "http://auth.example.com",
			wantErr:   true,
			errMsg:    "must use HTTPS",
		},
		{
			name:      "issuer with query parameter",
			issuerURL: "https://auth.example.com?param=value",
			wantErr:   true,
			errMsg:    "must not contain query",
		},
		{
			name:      "issuer with fragment",
			issuerURL: "https://auth.example.com#fragment",
			wantErr:   true,
			errMsg:    "must not contain query or fragment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ii, err := NewIssuerIdentifier(tt.issuerURL)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %q, want substring %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if ii == nil {
				t.Error("issuer identifier is nil")
				return
			}

			// Verify trailing slash is removed
			expectedIssuer := strings.TrimSuffix(tt.issuerURL, "/")
			if ii.GetIssuer() != expectedIssuer {
				t.Errorf("GetIssuer() = %q, want %q", ii.GetIssuer(), expectedIssuer)
			}
		})
	}
}

// TestAddIssuerToAuthorizationResponse tests adding 'iss' parameter to redirect URI.
func TestAddIssuerToAuthorizationResponse(t *testing.T) {
	issuer := "https://auth.example.com"
	ii, err := NewIssuerIdentifier(issuer)
	if err != nil {
		t.Fatalf("failed to create issuer identifier: %v", err)
	}

	tests := []struct {
		name        string
		redirectURI string
		wantErr     bool
		checkFunc   func(t *testing.T, result string)
	}{
		{
			name:        "redirect URI without query parameters",
			redirectURI: "https://client.example.com/callback",
			wantErr:     false,
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "iss=https%3A%2F%2Fauth.example.com") {
					t.Errorf("result does not contain encoded iss parameter: %s", result)
				}
			},
		},
		{
			name:        "redirect URI with existing query parameters",
			redirectURI: "https://client.example.com/callback?code=abc123&state=xyz",
			wantErr:     false,
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "code=abc123") {
					t.Error("existing code parameter missing")
				}
				if !strings.Contains(result, "state=xyz") {
					t.Error("existing state parameter missing")
				}
				if !strings.Contains(result, "iss=https%3A%2F%2Fauth.example.com") {
					t.Error("iss parameter missing")
				}
			},
		},
		{
			name:        "redirect URI with fragment",
			redirectURI: "https://client.example.com/callback#fragment",
			wantErr:     false,
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "iss=https%3A%2F%2Fauth.example.com") {
					t.Error("iss parameter missing")
				}
				if !strings.Contains(result, "#fragment") {
					t.Error("fragment missing")
				}
			},
		},
		{
			name:        "empty redirect URI",
			redirectURI: "",
			wantErr:     true,
		},
		{
			name:        "invalid redirect URI",
			redirectURI: "not a valid uri",
			wantErr:     false, // url.Parse is lenient and treats this as a relative URL
			checkFunc: func(t *testing.T, result string) {
				if !strings.Contains(result, "iss=") {
					t.Error("iss parameter should be added even to relative URLs")
				}
			},
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ii.AddIssuerToAuthorizationResponse(ctx, tt.redirectURI)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, result)
			}
		})
	}
}

// TestValidateIssuerInTokenRequest tests issuer validation in token requests.
func TestValidateIssuerInTokenRequest(t *testing.T) {
	issuer := "https://auth.example.com"
	ii, err := NewIssuerIdentifier(issuer)
	if err != nil {
		t.Fatalf("failed to create issuer identifier: %v", err)
	}

	tests := []struct {
		name              string
		issuerFromRequest string
		wantErr           bool
		expectedErrorCode string
		expectedErrorMsg  string
	}{
		{
			name:              "valid issuer",
			issuerFromRequest: "https://auth.example.com",
			wantErr:           false,
		},
		{
			name:              "missing issuer",
			issuerFromRequest: "",
			wantErr:           true,
			expectedErrorCode: ErrorInvalidRequest,
			expectedErrorMsg:  "missing 'iss' parameter",
		},
		{
			name:              "wrong issuer",
			issuerFromRequest: "https://evil.example.com",
			wantErr:           true,
			expectedErrorCode: ErrorInvalidRequest,
			expectedErrorMsg:  "issuer mismatch",
		},
		{
			name:              "issuer with trailing slash",
			issuerFromRequest: "https://auth.example.com/",
			wantErr:           true,
			expectedErrorCode: ErrorInvalidRequest,
			expectedErrorMsg:  "issuer mismatch",
		},
		{
			name:              "issuer case mismatch",
			issuerFromRequest: "https://AUTH.EXAMPLE.COM",
			wantErr:           true,
			expectedErrorCode: ErrorInvalidRequest,
			expectedErrorMsg:  "issuer mismatch",
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ii.ValidateIssuerInTokenRequest(ctx, tt.issuerFromRequest)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}

				oidcErr, ok := err.(*OIDCError)
				if !ok {
					t.Errorf("error type = %T, want *OIDCError", err)
					return
				}

				if oidcErr.ErrorCode != tt.expectedErrorCode {
					t.Errorf("error code = %q, want %q", oidcErr.ErrorCode, tt.expectedErrorCode)
				}

				if tt.expectedErrorMsg != "" && !strings.Contains(oidcErr.ErrorDescription, tt.expectedErrorMsg) {
					t.Errorf("error description = %q, want substring %q", oidcErr.ErrorDescription, tt.expectedErrorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestExtractIssuerFromRedirect tests issuer extraction from redirect URI.
func TestExtractIssuerFromRedirect(t *testing.T) {
	tests := []struct {
		name        string
		redirectURI string
		wantIssuer  string
		wantErr     bool
	}{
		{
			name:        "valid redirect with iss parameter",
			redirectURI: "https://client.example.com/callback?code=abc123&iss=https%3A%2F%2Fauth.example.com",
			wantIssuer:  "https://auth.example.com",
			wantErr:     false,
		},
		{
			name:        "iss parameter only",
			redirectURI: "https://client.example.com/callback?iss=https%3A%2F%2Fauth.example.com",
			wantIssuer:  "https://auth.example.com",
			wantErr:     false,
		},
		{
			name:        "missing iss parameter",
			redirectURI: "https://client.example.com/callback?code=abc123",
			wantErr:     true,
		},
		{
			name:        "empty redirect URI",
			redirectURI: "",
			wantErr:     true,
		},
		{
			name:        "invalid redirect URI",
			redirectURI: "not a valid uri",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issuer, err := ExtractIssuerFromRedirect(tt.redirectURI)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if issuer != tt.wantIssuer {
				t.Errorf("issuer = %q, want %q", issuer, tt.wantIssuer)
			}
		})
	}
}

// TestIssuerIdentificationMiddleware tests middleware functionality.
func TestIssuerIdentificationMiddleware(t *testing.T) {
	issuer := "https://auth.example.com"
	ii, err := NewIssuerIdentifier(issuer)
	if err != nil {
		t.Fatalf("failed to create issuer identifier: %v", err)
	}

	tests := []struct {
		name        string
		enabled     bool
		issuerParam string
		wantErr     bool
	}{
		{
			name:        "middleware enabled, valid issuer",
			enabled:     true,
			issuerParam: "https://auth.example.com",
			wantErr:     false,
		},
		{
			name:        "middleware enabled, invalid issuer",
			enabled:     true,
			issuerParam: "https://evil.example.com",
			wantErr:     true,
		},
		{
			name:        "middleware enabled, missing issuer",
			enabled:     true,
			issuerParam: "",
			wantErr:     true,
		},
		{
			name:        "middleware disabled, missing issuer",
			enabled:     false,
			issuerParam: "",
			wantErr:     false,
		},
		{
			name:        "middleware disabled, invalid issuer",
			enabled:     false,
			issuerParam: "https://evil.example.com",
			wantErr:     false,
		},
	}

	ctx := context.Background()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := NewIssuerIdentificationMiddleware(ii, tt.enabled)
			err := middleware.ValidateRequest(ctx, tt.issuerParam)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// TestMixUpAttackPrevention tests that RFC 9207 prevents mix-up attacks.
func TestMixUpAttackPrevention(t *testing.T) {
	// Scenario: Client is registered with two authorization servers
	as1Issuer := "https://as1.example.com"
	as2Issuer := "https://as2.example.com"

	as1, err := NewIssuerIdentifier(as1Issuer)
	if err != nil {
		t.Fatalf("failed to create AS1 identifier: %v", err)
	}

	as2, err := NewIssuerIdentifier(as2Issuer)
	if err != nil {
		t.Fatalf("failed to create AS2 identifier: %v", err)
	}

	ctx := context.Background()

	// Step 1: Client initiates flow with AS1
	redirectURI := "https://client.example.com/callback?code=from_as1&state=xyz"
	as1RedirectWithIss, err := as1.AddIssuerToAuthorizationResponse(ctx, redirectURI)
	if err != nil {
		t.Fatalf("AS1 failed to add iss: %v", err)
	}

	// Step 2: Extract issuer from redirect (client side)
	extractedIssuer, err := ExtractIssuerFromRedirect(as1RedirectWithIss)
	if err != nil {
		t.Fatalf("failed to extract issuer: %v", err)
	}

	if extractedIssuer != as1Issuer {
		t.Errorf("extracted issuer = %q, want %q", extractedIssuer, as1Issuer)
	}

	// Step 3: Attacker tries to replay AS1 code to AS2
	// AS2 should reject because issuer doesn't match
	err = as2.ValidateIssuerInTokenRequest(ctx, as1Issuer)
	if err == nil {
		t.Error("AS2 should reject AS1 issuer (mix-up attack prevented)")
	}

	oidcErr, ok := err.(*OIDCError)
	if !ok {
		t.Fatalf("error type = %T, want *OIDCError", err)
	}

	if oidcErr.ErrorCode != ErrorInvalidRequest {
		t.Errorf("error code = %q, want %q", oidcErr.ErrorCode, ErrorInvalidRequest)
	}

	if !strings.Contains(oidcErr.ErrorDescription, "issuer mismatch") {
		t.Errorf("error should mention issuer mismatch, got: %s", oidcErr.ErrorDescription)
	}

	// Step 4: Verify AS1 accepts its own issuer
	err = as1.ValidateIssuerInTokenRequest(ctx, as1Issuer)
	if err != nil {
		t.Errorf("AS1 should accept its own issuer: %v", err)
	}

	// Step 5: Verify AS2 accepts its own issuer
	err = as2.ValidateIssuerInTokenRequest(ctx, as2Issuer)
	if err != nil {
		t.Errorf("AS2 should accept its own issuer: %v", err)
	}
}

// TestIssuerTrailingSlashNormalization tests trailing slash handling.
func TestIssuerTrailingSlashNormalization(t *testing.T) {
	tests := []struct {
		name      string
		issuerURL string
		expected  string
	}{
		{
			name:      "no trailing slash",
			issuerURL: "https://auth.example.com",
			expected:  "https://auth.example.com",
		},
		{
			name:      "single trailing slash",
			issuerURL: "https://auth.example.com/",
			expected:  "https://auth.example.com",
		},
		{
			name:      "with path, no trailing slash",
			issuerURL: "https://auth.example.com/oauth2",
			expected:  "https://auth.example.com/oauth2",
		},
		{
			name:      "with path, trailing slash",
			issuerURL: "https://auth.example.com/oauth2/",
			expected:  "https://auth.example.com/oauth2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ii, err := NewIssuerIdentifier(tt.issuerURL)
			if err != nil {
				t.Fatalf("failed to create issuer identifier: %v", err)
			}

			if ii.GetIssuer() != tt.expected {
				t.Errorf("GetIssuer() = %q, want %q", ii.GetIssuer(), tt.expected)
			}
		})
	}
}

// TestMultipleAuthorizationServers tests handling of multiple AS scenario.
func TestMultipleAuthorizationServers(t *testing.T) {
	authServers := []string{
		"https://as1.example.com",
		"https://as2.example.com",
		"https://as3.example.com",
	}

	// Create identifiers for all AS
	identifiers := make(map[string]*IssuerIdentifier)
	for _, issuer := range authServers {
		ii, err := NewIssuerIdentifier(issuer)
		if err != nil {
			t.Fatalf("failed to create identifier for %s: %v", issuer, err)
		}
		identifiers[issuer] = ii
	}

	ctx := context.Background()

	// Test that each AS accepts only its own issuer
	for i, issuer := range authServers {
		ii := identifiers[issuer]

		// Should accept own issuer
		err := ii.ValidateIssuerInTokenRequest(ctx, issuer)
		if err != nil {
			t.Errorf("AS %d should accept its own issuer: %v", i+1, err)
		}

		// Should reject other issuers
		for j, otherIssuer := range authServers {
			if i == j {
				continue
			}

			err := ii.ValidateIssuerInTokenRequest(ctx, otherIssuer)
			if err == nil {
				t.Errorf("AS %d should reject AS %d's issuer", i+1, j+1)
			}
		}
	}
}
