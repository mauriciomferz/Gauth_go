package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// generateTestRSAKey generates an RSA key pair for testing
func generateTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}
	return key
}

func TestNewIDTokenService(t *testing.T) {
	privateKey := generateTestRSAKey(t)

	tests := []struct {
		name    string
		config  *IDTokenServiceConfig
		wantErr bool
	}{
		{
			name: "Valid configuration with RS256",
			config: &IDTokenServiceConfig{
				IssuerURL:     "https://agentauth.example.com",
				SigningKey:    privateKey,
				SigningKeyID:  "test-key-1",
				SigningMethod: "RS256",
				TokenExpiry:   time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Valid configuration with RS384",
			config: &IDTokenServiceConfig{
				IssuerURL:     "https://agentauth.example.com",
				SigningKey:    privateKey,
				SigningKeyID:  "test-key-1",
				SigningMethod: "RS384",
				TokenExpiry:   time.Hour,
			},
			wantErr: false,
		},
		{
			name: "Missing signing key",
			config: &IDTokenServiceConfig{
				IssuerURL:     "https://agentauth.example.com",
				SigningKey:    nil,
				SigningKeyID:  "test-key-1",
				SigningMethod: "RS256",
			},
			wantErr: true,
		},
		{
			name: "Invalid signing method",
			config: &IDTokenServiceConfig{
				IssuerURL:     "https://agentauth.example.com",
				SigningKey:    privateKey,
				SigningKeyID:  "test-key-1",
				SigningMethod: "HS256",
			},
			wantErr: true,
		},
		{
			name: "Default signing method (RS256)",
			config: &IDTokenServiceConfig{
				IssuerURL:  "https://agentauth.example.com",
				SigningKey: privateKey,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service, err := NewIDTokenService(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewIDTokenService() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && service == nil {
				t.Error("NewIDTokenService() returned nil service")
			}
		})
	}
}

func TestIDTokenService_IssueIDToken(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	service, err := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:     "https://agentauth.example.com",
		SigningKey:    privateKey,
		SigningKeyID:  "test-key-1",
		SigningMethod: "RS256",
		TokenExpiry:   time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	tests := []struct {
		name    string
		claims  *IDTokenClaims
		wantErr bool
	}{
		{
			name: "Valid ID token with minimal claims",
			claims: &IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:  "user123",
					Audience: jwt.ClaimStrings{"client123"},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid ID token with full claims",
			claims: &IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:  "user123",
					Audience: jwt.ClaimStrings{"client123"},
				},
				Name:          "John Doe",
				Email:         "john@example.com",
				EmailVerified: true,
				ACR:           "substantial",
				EntityType:    "natural_person",
				Nonce:         "test-nonce-123",
			},
			wantErr: false,
		},
		{
			name: "Valid ID token for legal entity",
			claims: &IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject:  "entity123",
					Audience: jwt.ClaimStrings{"client123"},
				},
				LegalEntityName: "Acme Corporation",
				EntityType:      "legal_entity",
				EntityID:        "DE123456789",
				Jurisdiction:    "DE",
				ACR:             "high",
			},
			wantErr: false,
		},
		{
			name: "Missing subject",
			claims: &IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Audience: jwt.ClaimStrings{"client123"},
				},
			},
			wantErr: true,
		},
		{
			name: "Missing audience",
			claims: &IDTokenClaims{
				RegisteredClaims: jwt.RegisteredClaims{
					Subject: "user123",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.IssueIDToken(context.Background(), tt.claims)
			if (err != nil) != tt.wantErr {
				t.Errorf("IssueIDToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if token == "" {
					t.Error("IssueIDToken() returned empty token")
				}
				// Verify token is JWT format (3 parts separated by dots)
				parts := 0
				for _, c := range token {
					if c == '.' {
						parts++
					}
				}
				if parts != 2 {
					t.Errorf("Token does not have 3 parts (header.payload.signature), got %d dots", parts)
				}
			}
		})
	}
}

func TestIDTokenService_ValidateIDToken(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	service, err := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:     "https://agentauth.example.com",
		SigningKey:    privateKey,
		SigningKeyID:  "test-key-1",
		SigningMethod: "RS256",
		TokenExpiry:   time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	// Issue a valid token
	validClaims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "user123",
			Audience: jwt.ClaimStrings{"client123"},
		},
		Name:  "John Doe",
		Email: "john@example.com",
		ACR:   "substantial",
	}
	validToken, err := service.IssueIDToken(context.Background(), validClaims)
	if err != nil {
		t.Fatalf("Failed to issue valid token: %v", err)
	}

	// Issue a token with different audience
	wrongAudienceClaims := &IDTokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  "user123",
			Audience: jwt.ClaimStrings{"wrong-client"},
		},
	}
	wrongAudienceToken, err := service.IssueIDToken(context.Background(), wrongAudienceClaims)
	if err != nil {
		t.Fatalf("Failed to issue token with wrong audience: %v", err)
	}

	tests := []struct {
		name             string
		token            string
		expectedAudience string
		wantErr          bool
	}{
		{
			name:             "Valid token with correct audience",
			token:            validToken,
			expectedAudience: "client123",
			wantErr:          false,
		},
		{
			name:             "Valid token with wrong audience",
			token:            wrongAudienceToken,
			expectedAudience: "client123",
			wantErr:          true,
		},
		{
			name:             "Invalid token format",
			token:            "not.a.valid.jwt",
			expectedAudience: "client123",
			wantErr:          true,
		},
		{
			name:             "Empty token",
			token:            "",
			expectedAudience: "client123",
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateIDToken(context.Background(), tt.token, tt.expectedAudience)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIDToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if claims == nil {
					t.Error("ValidateIDToken() returned nil claims")
					return
				}
				if claims.Subject != "user123" {
					t.Errorf("Expected subject user123, got %s", claims.Subject)
				}
				if claims.Issuer != "https://agentauth.example.com" {
					t.Errorf("Expected issuer https://agentauth.example.com, got %s", claims.Issuer)
				}
			}
		})
	}
}

func TestIDTokenService_CreateIDTokenFromIdentity(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	service, err := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:     "https://agentauth.example.com",
		SigningKey:    privateKey,
		SigningKeyID:  "test-key-1",
		SigningMethod: "RS256",
		TokenExpiry:   time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create ID token service: %v", err)
	}

	tests := []struct {
		name             string
		subjectID        string
		audience         []string
		identityType     string
		trustLevel       string
		additionalClaims map[string]interface{}
		wantErr          bool
	}{
		{
			name:         "Natural person identity",
			subjectID:    "user123",
			audience:     []string{"client123"},
			identityType: "natural_person",
			trustLevel:   "substantial",
			additionalClaims: map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			wantErr: false,
		},
		{
			name:         "Legal entity identity",
			subjectID:    "entity123",
			audience:     []string{"client123"},
			identityType: "legal_entity",
			trustLevel:   "high",
			additionalClaims: map[string]interface{}{
				"legal_entity_name": "Acme Corp",
				"entity_id":         "DE123456789",
				"jurisdiction":      "DE",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.CreateIDTokenFromIdentity(
				context.Background(),
				tt.subjectID,
				tt.audience,
				tt.identityType,
				tt.trustLevel,
				tt.additionalClaims,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateIDTokenFromIdentity() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if token == "" {
					t.Error("CreateIDTokenFromIdentity() returned empty token")
				}

				// Validate the token
				claims, err := service.ValidateIDToken(context.Background(), token, tt.audience[0])
				if err != nil {
					t.Errorf("Failed to validate created token: %v", err)
					return
				}

				if claims.Subject != tt.subjectID {
					t.Errorf("Expected subject %s, got %s", tt.subjectID, claims.Subject)
				}
				if claims.EntityType != tt.identityType {
					t.Errorf("Expected entity_type %s, got %s", tt.identityType, claims.EntityType)
				}
			}
		})
	}
}

func TestIDTokenService_MapTrustLevelToACR(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	service, _ := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:  "https://agentauth.example.com",
		SigningKey: privateKey,
	})

	tests := []struct {
		name       string
		trustLevel string
		want       string
	}{
		{
			name:       "low trust level",
			trustLevel: "low",
			want:       "1",
		},
		{
			name:       "substantial trust level",
			trustLevel: "substantial",
			want:       "substantial",
		},
		{
			name:       "high trust level",
			trustLevel: "high",
			want:       "high",
		},
		{
			name:       "unknown trust level",
			trustLevel: "unknown",
			want:       "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := service.mapTrustLevelToACR(tt.trustLevel); got != tt.want {
				t.Errorf("mapTrustLevelToACR(%s) = %s, want %s", tt.trustLevel, got, tt.want)
			}
		})
	}
}

func TestIDTokenService_GetMethods(t *testing.T) {
	privateKey := generateTestRSAKey(t)
	service, _ := NewIDTokenService(&IDTokenServiceConfig{
		IssuerURL:     "https://agentauth.example.com",
		SigningKey:    privateKey,
		SigningKeyID:  "test-key-1",
		SigningMethod: "RS256",
		TokenExpiry:   2 * time.Hour,
	})

	if got := service.GetSigningKeyID(); got != "test-key-1" {
		t.Errorf("GetSigningKeyID() = %s, want test-key-1", got)
	}

	if got := service.GetSigningAlgorithm(); got != "RS256" {
		t.Errorf("GetSigningAlgorithm() = %s, want RS256", got)
	}

	if got := service.GetDefaultExpiry(); got != 2*time.Hour {
		t.Errorf("GetDefaultExpiry() = %v, want 2h", got)
	}
}
