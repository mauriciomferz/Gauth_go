// Package oidc - OIDC PowerVerificationPoint Tests
package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth"
)

// TestNewOIDCPowerVerificationPoint tests OIDC PVP creation
func TestNewOIDCPowerVerificationPoint(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	idTokenService, err := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:    "https://gauth.example.com",
		SigningKey:   privateKey,
		SigningKeyID: "key1",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	tests := []struct {
		name        string
		config      OIDCPVPConfig
		wantErr     bool
		expectedACR string
	}{
		{
			name: "valid config with custom ACR",
			config: OIDCPVPConfig{
				IDTokenService: idTokenService,
				RequiredACR:    "high",
			},
			wantErr:     false,
			expectedACR: "high",
		},
		{
			name: "valid config with default ACR",
			config: OIDCPVPConfig{
				IDTokenService: idTokenService,
			},
			wantErr:     false,
			expectedACR: "substantial",
		},
		{
			name: "missing ID token service",
			config: OIDCPVPConfig{
				RequiredACR: "substantial",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pvp, err := NewOIDCPowerVerificationPoint(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if pvp == nil {
				t.Error("Expected PVP, got nil")
				return
			}
			if pvp.GetRequiredACR() != tt.expectedACR {
				t.Errorf("Expected required ACR %s, got %s", tt.expectedACR, pvp.GetRequiredACR())
			}
		})
	}
}

// TestOIDCPowerVerificationPoint_VerifyIdentityProof tests identity proof verification
func TestOIDCPowerVerificationPoint_VerifyIdentityProof(t *testing.T) {
	// Setup
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	idTokenService, err := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:    "https://gauth.example.com",
		SigningKey:   privateKey,
		SigningKeyID: "key1",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	pvp, err := NewOIDCPowerVerificationPoint(OIDCPVPConfig{
		IDTokenService: idTokenService,
		RequiredACR:    "substantial",
	})
	if err != nil {
		t.Fatalf("Failed to create PVP: %v", err)
	}

	ctx := context.Background()

	// Create a valid ID token
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user123",
			Audience:  jwt.ClaimStrings{"client456"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		Name:       "John Doe",
		Email:      "john@example.com",
		ACR:        "substantial",
		EntityType: "natural_person",
	}
	validToken, err := idTokenService.IssueIDToken(ctx, claims)
	if err != nil {
		t.Fatalf("Failed to issue ID token: %v", err)
	}

	// Create an ID token with insufficient trust level
	lowTrustClaims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user789",
			Audience:  jwt.ClaimStrings{"client456"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		Name:       "Jane Doe",
		ACR:        "0", // Low trust level
		EntityType: "natural_person",
	}
	lowTrustToken, err := idTokenService.IssueIDToken(ctx, lowTrustClaims)
	if err != nil {
		t.Fatalf("Failed to issue low trust ID token: %v", err)
	}

	tests := []struct {
		name         string
		request      *gauth.IdentityProofRequest
		expectValid  bool
		expectError  bool
		expectReason string
	}{
		{
			name: "valid ID token with sufficient trust",
			request: &gauth.IdentityProofRequest{
				SubjectID:    "user123",
				IdentityType: "natural_person",
				ProofMethod:  ProofMethodOIDCIDToken,
				ProofData: map[string]interface{}{
					"id_token": validToken,
					"audience": "client456",
				},
				RequiredLevel: "substantial",
			},
			expectValid: true,
			expectError: false,
		},
		{
			name: "valid ID token with default trust requirement",
			request: &gauth.IdentityProofRequest{
				SubjectID:    "user123",
				IdentityType: "natural_person",
				ProofMethod:  ProofMethodOIDCIDToken,
				ProofData: map[string]interface{}{
					"id_token": validToken,
					"audience": "client456",
				},
			},
			expectValid: true,
			expectError: false,
		},
		{
			name: "insufficient trust level",
			request: &gauth.IdentityProofRequest{
				SubjectID:    "user789",
				IdentityType: "natural_person",
				ProofMethod:  ProofMethodOIDCIDToken,
				ProofData: map[string]interface{}{
					"id_token": lowTrustToken,
					"audience": "client456",
				},
				RequiredLevel: "substantial",
			},
			expectValid:  false,
			expectError:  false,
			expectReason: "insufficient trust level",
		},
		{
			name: "missing id_token",
			request: &gauth.IdentityProofRequest{
				SubjectID:    "user123",
				IdentityType: "natural_person",
				ProofMethod:  ProofMethodOIDCIDToken,
				ProofData: map[string]interface{}{
					"audience": "client456",
				},
			},
			expectValid:  false,
			expectError:  false,
			expectReason: "id_token not found",
		},
		{
			name: "missing audience",
			request: &gauth.IdentityProofRequest{
				SubjectID:    "user123",
				IdentityType: "natural_person",
				ProofMethod:  ProofMethodOIDCIDToken,
				ProofData: map[string]interface{}{
					"id_token": validToken,
				},
			},
			expectValid:  false,
			expectError:  false,
			expectReason: "audience not found",
		},
		{
			name: "invalid token format",
			request: &gauth.IdentityProofRequest{
				SubjectID:    "user123",
				IdentityType: "natural_person",
				ProofMethod:  ProofMethodOIDCIDToken,
				ProofData: map[string]interface{}{
					"id_token": "not.a.valid.token",
					"audience": "client456",
				},
			},
			expectValid:  false,
			expectError:  false,
			expectReason: "ID token validation failed",
		},
		{
			name: "wrong audience",
			request: &gauth.IdentityProofRequest{
				SubjectID:    "user123",
				IdentityType: "natural_person",
				ProofMethod:  ProofMethodOIDCIDToken,
				ProofData: map[string]interface{}{
					"id_token": validToken,
					"audience": "wrong_client",
				},
			},
			expectValid:  false,
			expectError:  false,
			expectReason: "ID token validation failed",
		},
		{
			name: "subject ID mismatch",
			request: &gauth.IdentityProofRequest{
				SubjectID:    "wrong_user",
				IdentityType: "natural_person",
				ProofMethod:  ProofMethodOIDCIDToken,
				ProofData: map[string]interface{}{
					"id_token": validToken,
					"audience": "client456",
				},
			},
			expectValid:  false,
			expectError:  false,
			expectReason: "subject ID mismatch",
		},
		{
			name: "unsupported proof method",
			request: &gauth.IdentityProofRequest{
				SubjectID:    "user123",
				IdentityType: "natural_person",
				ProofMethod:  "eIDAS",
				ProofData: map[string]interface{}{
					"id_token": validToken,
					"audience": "client456",
				},
			},
			expectValid: false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := pvp.VerifyIdentityProof(ctx, tt.request)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected result, got nil")
				return
			}

			if result.Valid != tt.expectValid {
				t.Errorf("Expected Valid=%v, got %v. Reason: %s",
					tt.expectValid, result.Valid, result.FailureReason)
			}

			if !tt.expectValid && tt.expectReason != "" {
				if result.FailureReason == "" {
					t.Error("Expected failure reason, got empty string")
				}
				// Check if failure reason contains expected substring
				if len(result.FailureReason) > 0 && len(tt.expectReason) > 0 {
					found := false
					for i := 0; i <= len(result.FailureReason)-len(tt.expectReason); i++ {
						if result.FailureReason[i:i+len(tt.expectReason)] == tt.expectReason {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected failure reason to contain '%s', got '%s'",
							tt.expectReason, result.FailureReason)
					}
				}
			}

			if tt.expectValid {
				if result.SubjectID != tt.request.SubjectID {
					t.Errorf("Expected SubjectID=%s, got %s",
						tt.request.SubjectID, result.SubjectID)
				}
				if result.TrustLevel == "" {
					t.Error("Expected TrustLevel to be set")
				}
			}
		})
	}
}

// TestOIDCPowerVerificationPoint_GetSupportedProofMethods tests supported proof methods
func TestOIDCPowerVerificationPoint_GetSupportedProofMethods(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	idTokenService, err := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:    "https://gauth.example.com",
		SigningKey:   privateKey,
		SigningKeyID: "key1",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	pvp, err := NewOIDCPowerVerificationPoint(OIDCPVPConfig{
		IDTokenService: idTokenService,
	})
	if err != nil {
		t.Fatalf("Failed to create PVP: %v", err)
	}

	methods := pvp.GetSupportedProofMethods()
	if len(methods) != 2 {
		t.Errorf("Expected 2 supported methods, got %d", len(methods))
	}

	expectedMethods := map[string]bool{
		ProofMethodOIDCIDToken:  true,
		ProofMethodOIDCExternal: true,
	}

	for _, method := range methods {
		if !expectedMethods[method] {
			t.Errorf("Unexpected method: %s", method)
		}
	}
}

// TestOIDCPowerVerificationPoint_ACRManagement tests ACR requirement management
func TestOIDCPowerVerificationPoint_ACRManagement(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	idTokenService, err := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:    "https://gauth.example.com",
		SigningKey:   privateKey,
		SigningKeyID: "key1",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	pvp, err := NewOIDCPowerVerificationPoint(OIDCPVPConfig{
		IDTokenService: idTokenService,
		RequiredACR:    "substantial",
	})
	if err != nil {
		t.Fatalf("Failed to create PVP: %v", err)
	}

	// Test initial ACR
	if pvp.GetRequiredACR() != "substantial" {
		t.Errorf("Expected ACR 'substantial', got '%s'", pvp.GetRequiredACR())
	}

	// Test updating ACR
	pvp.SetRequiredACR("high")
	if pvp.GetRequiredACR() != "high" {
		t.Errorf("Expected ACR 'high' after update, got '%s'", pvp.GetRequiredACR())
	}
}

// TestOIDCPowerVerificationPoint_ValidateProofData tests proof data validation
func TestOIDCPowerVerificationPoint_ValidateProofData(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	idTokenService, err := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:    "https://gauth.example.com",
		SigningKey:   privateKey,
		SigningKeyID: "key1",
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	pvp, err := NewOIDCPowerVerificationPoint(OIDCPVPConfig{
		IDTokenService: idTokenService,
	})
	if err != nil {
		t.Fatalf("Failed to create PVP: %v", err)
	}

	tests := []struct {
		name      string
		proofData map[string]interface{}
		wantErr   bool
		errMsg    string
	}{
		{
			name: "valid proof data",
			proofData: map[string]interface{}{
				"id_token": "eyJhbGciOiJSUzI1NiJ9...",
				"audience": "client123",
			},
			wantErr: false,
		},
		{
			name:      "nil proof data",
			proofData: nil,
			wantErr:   true,
			errMsg:    "proof_data is required",
		},
		{
			name: "missing id_token",
			proofData: map[string]interface{}{
				"audience": "client123",
			},
			wantErr: true,
			errMsg:  "id_token is required in proof_data",
		},
		{
			name: "empty id_token",
			proofData: map[string]interface{}{
				"id_token": "",
				"audience": "client123",
			},
			wantErr: true,
			errMsg:  "id_token is required in proof_data",
		},
		{
			name: "missing audience",
			proofData: map[string]interface{}{
				"id_token": "eyJhbGciOiJSUzI1NiJ9...",
			},
			wantErr: true,
			errMsg:  "audience is required in proof_data",
		},
		{
			name: "empty audience",
			proofData: map[string]interface{}{
				"id_token": "eyJhbGciOiJSUzI1NiJ9...",
				"audience": "",
			},
			wantErr: true,
			errMsg:  "audience is required in proof_data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := pvp.ValidateProofData(tt.proofData)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
