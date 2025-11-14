package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	"github.com/golang-jwt/jwt/v5"
)

func TestNewIdentityBridge(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	idTokenService, _ := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:  "https://gauth.example.com",
		SigningKey: privateKey,
	})

	bridge := NewIdentityBridge(idTokenService)
	if bridge == nil {
		t.Fatal("NewIdentityBridge returned nil")
	}
	if bridge.idTokenService == nil {
		t.Error("Bridge has nil idTokenService")
	}
	if bridge.trustMapper == nil {
		t.Error("Bridge has nil trustMapper")
	}
}

func TestIdentityBridge_ConvertIDTokenToIdentityProof(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	idTokenService, _ := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:  "https://gauth.example.com",
		SigningKey: privateKey,
	})
	bridge := NewIdentityBridge(idTokenService)

	// Create a valid ID token
	validClaims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "user123",
			Audience: jwt.ClaimStrings{"client123"},
		},
		Name:       "John Doe",
		Email:      "john@example.com",
		ACR:        "substantial",
		EntityType: "natural_person",
	}
	validToken, _ := idTokenService.IssueIDToken(context.Background(), validClaims)

	tests := []struct {
		name             string
		token            string
		expectedAudience string
		wantValid        bool
		wantSubject      string
	}{
		{
			name:             "Valid ID token",
			token:            validToken,
			expectedAudience: "client123",
			wantValid:        true,
			wantSubject:      "user123",
		},
		{
			name:             "Invalid token",
			token:            "invalid.token.here",
			expectedAudience: "client123",
			wantValid:        false,
			wantSubject:      "",
		},
		{
			name:             "Empty token",
			token:            "",
			expectedAudience: "client123",
			wantValid:        false,
			wantSubject:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := bridge.ConvertIDTokenToIdentityProof(
				context.Background(),
				tt.token,
				tt.expectedAudience,
			)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result.Valid != tt.wantValid {
				t.Errorf("Expected Valid=%v, got %v", tt.wantValid, result.Valid)
			}

			if tt.wantValid {
				if result.SubjectID != tt.wantSubject {
					t.Errorf("Expected SubjectID=%s, got %s", tt.wantSubject, result.SubjectID)
				}
				if result.Identity == "" {
					t.Error("Identity should not be empty for valid token")
				}
				if result.TrustLevel == "" {
					t.Error("TrustLevel should not be empty for valid token")
				}
			} else {
				if result.FailureReason == "" {
					t.Error("FailureReason should not be empty for invalid token")
				}
			}
		})
	}
}

func TestIdentityBridge_ConvertIdentityProofToIDToken(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	idTokenService, _ := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:  "https://gauth.example.com",
		SigningKey: privateKey,
	})
	bridge := NewIdentityBridge(idTokenService)

	tests := []struct {
		name         string
		proof        *gauth.IdentityProofResult
		audience     []string
		identityType string
		wantErr      bool
	}{
		{
			name: "Valid identity proof",
			proof: &gauth.IdentityProofResult{
				Valid:      true,
				SubjectID:  "user123",
				Identity:   "John Doe",
				TrustLevel: "substantial",
			},
			audience:     []string{"client123"},
			identityType: "natural_person",
			wantErr:      false,
		},
		{
			name: "Invalid identity proof",
			proof: &gauth.IdentityProofResult{
				Valid:         false,
				FailureReason: "verification failed",
			},
			audience:     []string{"client123"},
			identityType: "natural_person",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := bridge.ConvertIdentityProofToIDToken(
				context.Background(),
				tt.proof,
				tt.audience,
				tt.identityType,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("ConvertIdentityProofToIDToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && token == "" {
				t.Error("Expected non-empty token")
			}
		})
	}
}

func TestTrustLevelMapper_MapACRToTrustLevel(t *testing.T) {
	mapper := NewTrustLevelMapper()

	tests := []struct {
		name string
		acr  string
		want string
	}{
		{
			name: "ACR 0",
			acr:  "0",
			want: "low",
		},
		{
			name: "ACR 1",
			acr:  "1",
			want: "low",
		},
		{
			name: "ACR 2",
			acr:  "2",
			want: "substantial",
		},
		{
			name: "ACR substantial",
			acr:  "substantial",
			want: "substantial",
		},
		{
			name: "ACR high",
			acr:  "high",
			want: "high",
		},
		{
			name: "ACR loa-4",
			acr:  "loa-4",
			want: "high",
		},
		{
			name: "Unknown ACR",
			acr:  "unknown",
			want: "low",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapper.MapACRToTrustLevel(tt.acr); got != tt.want {
				t.Errorf("MapACRToTrustLevel(%s) = %s, want %s", tt.acr, got, tt.want)
			}
		})
	}
}

func TestTrustLevelMapper_MapTrustLevelToACR(t *testing.T) {
	mapper := NewTrustLevelMapper()

	tests := []struct {
		name       string
		trustLevel string
		want       string
	}{
		{
			name:       "Low trust level",
			trustLevel: "low",
			want:       "1",
		},
		{
			name:       "Substantial trust level",
			trustLevel: "substantial",
			want:       "substantial",
		},
		{
			name:       "High trust level",
			trustLevel: "high",
			want:       "high",
		},
		{
			name:       "Unknown trust level",
			trustLevel: "unknown",
			want:       "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapper.MapTrustLevelToACR(tt.trustLevel); got != tt.want {
				t.Errorf("MapTrustLevelToACR(%s) = %s, want %s", tt.trustLevel, got, tt.want)
			}
		})
	}
}

func TestTrustLevelMapper_ValidateMinimumTrustLevel(t *testing.T) {
	mapper := NewTrustLevelMapper()

	tests := []struct {
		name     string
		acr      string
		required string
		want     bool
	}{
		{
			name:     "High ACR meets substantial requirement",
			acr:      "high",
			required: "substantial",
			want:     true,
		},
		{
			name:     "Substantial ACR meets substantial requirement",
			acr:      "substantial",
			required: "substantial",
			want:     true,
		},
		{
			name:     "Low ACR does not meet substantial requirement",
			acr:      "1",
			required: "substantial",
			want:     false,
		},
		{
			name:     "High ACR meets high requirement",
			acr:      "high",
			required: "high",
			want:     true,
		},
		{
			name:     "Substantial ACR does not meet high requirement",
			acr:      "substantial",
			required: "high",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mapper.ValidateMinimumTrustLevel(tt.acr, tt.required); got != tt.want {
				t.Errorf("ValidateMinimumTrustLevel(%s, %s) = %v, want %v", tt.acr, tt.required, got, tt.want)
			}
		})
	}
}

func TestTrustLevelMapper_AddCustomMapping(t *testing.T) {
	mapper := NewTrustLevelMapper()

	// Add custom mapping
	mapper.AddCustomMapping("custom_acr", "high")

	// Verify custom mapping works
	if got := mapper.MapACRToTrustLevel("custom_acr"); got != "high" {
		t.Errorf("Custom ACR mapping failed: got %s, want high", got)
	}
}

func TestExtractEntityTypeFromClaims(t *testing.T) {
	tests := []struct {
		name   string
		claims *IDTokenClaims
		want   string
	}{
		{
			name: "Explicit entity type",
			claims: &IDTokenClaims{
				EntityType: "natural_person",
			},
			want: "natural_person",
		},
		{
			name: "Legal entity with legal_entity_name",
			claims: &IDTokenClaims{
				LegalEntityName: "Acme Corp",
			},
			want: "legal_entity",
		},
		{
			name:   "Default to natural person",
			claims: &IDTokenClaims{},
			want:   "natural_person",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractEntityTypeFromClaims(tt.claims); got != tt.want {
				t.Errorf("ExtractEntityTypeFromClaims() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestExtractProofDataFromClaims(t *testing.T) {
	claims := &IDTokenClaims{
		Name:            "John Doe",
		Email:           "john@example.com",
		EmailVerified:   true,
		EntityID:        "DE123456789",
		LegalEntityName: "Acme Corp",
		Jurisdiction:    "DE",
		TSPName:         "eIDAS Provider",
		TSPID:           "tsp-123",
		ACR:             "substantial",
		AMR:             []string{"mfa", "otp"},
	}

	proofData := ExtractProofDataFromClaims(claims)

	tests := []struct {
		key  string
		want interface{}
	}{
		{"name", "John Doe"},
		{"email", "john@example.com"},
		{"email_verified", true},
		{"entity_id", "DE123456789"},
		{"legal_entity_name", "Acme Corp"},
		{"jurisdiction", "DE"},
		{"tsp_name", "eIDAS Provider"},
		{"tsp_id", "tsp-123"},
		{"acr", "substantial"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got, exists := proofData[tt.key]; !exists {
				t.Errorf("Key %s not found in proof data", tt.key)
			} else if got != tt.want {
				t.Errorf("proofData[%s] = %v, want %v", tt.key, got, tt.want)
			}
		})
	}

	// Verify AMR is extracted
	if amr, exists := proofData["amr"]; !exists {
		t.Error("AMR not found in proof data")
	} else {
		amrSlice, ok := amr.([]string)
		if !ok {
			t.Error("AMR is not []string")
		} else if len(amrSlice) != 2 {
			t.Errorf("Expected 2 AMR values, got %d", len(amrSlice))
		}
	}
}

func TestBuildIdentityProofRequestFromIDToken(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	idTokenService, _ := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:  "https://gauth.example.com",
		SigningKey: privateKey,
	})

	// Create a valid ID token
	claims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "user123",
			Audience: jwt.ClaimStrings{"client123"},
		},
		Name:       "John Doe",
		Email:      "john@example.com",
		ACR:        "substantial",
		EntityType: "natural_person",
	}
	token, _ := idTokenService.IssueIDToken(context.Background(), claims)

	// Build identity proof request
	request, err := BuildIdentityProofRequestFromIDToken(token, idTokenService, "client123")
	if err != nil {
		t.Fatalf("BuildIdentityProofRequestFromIDToken() error = %v", err)
	}

	if request.SubjectID != "user123" {
		t.Errorf("Expected SubjectID user123, got %s", request.SubjectID)
	}
	if request.IdentityType != "natural_person" {
		t.Errorf("Expected IdentityType natural_person, got %s", request.IdentityType)
	}
	if request.ProofMethod != ProofMethodOIDCIDToken {
		t.Errorf("Expected ProofMethod %s, got %s", ProofMethodOIDCIDToken, request.ProofMethod)
	}
	if request.ProofData == nil {
		t.Error("ProofData should not be nil")
	}
}

func TestValidateIDTokenForIdentityProof(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	idTokenService, _ := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:  "https://gauth.example.com",
		SigningKey: privateKey,
	})

	// Create tokens with different ACRs
	highACRToken, _ := idTokenService.IssueIDToken(context.Background(), &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "user123",
			Audience: jwt.ClaimStrings{"client123"},
		},
		ACR: "high",
	})

	lowACRToken, _ := idTokenService.IssueIDToken(context.Background(), &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "user123",
			Audience: jwt.ClaimStrings{"client123"},
		},
		ACR: "1",
	})

	tests := []struct {
		name             string
		token            string
		expectedAudience string
		minTrustLevel    string
		wantErr          bool
	}{
		{
			name:             "High ACR meets substantial requirement",
			token:            highACRToken,
			expectedAudience: "client123",
			minTrustLevel:    "substantial",
			wantErr:          false,
		},
		{
			name:             "Low ACR does not meet substantial requirement",
			token:            lowACRToken,
			expectedAudience: "client123",
			minTrustLevel:    "substantial",
			wantErr:          true,
		},
		{
			name:             "Invalid token",
			token:            "invalid.token.here",
			expectedAudience: "client123",
			minTrustLevel:    "low",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIDTokenForIdentityProof(
				context.Background(),
				tt.token,
				idTokenService,
				tt.expectedAudience,
				tt.minTrustLevel,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIDTokenForIdentityProof() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
