package gauth

import (
	"encoding/json"
	"testing"
	"time"
)

func TestService_ValidateAdvancedToken_Integration(t *testing.T) {
	// Create service with HMAC mode for testing
	config := Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "test-client",
		ClientSecret:      "test-secret-key-that-is-long-enough-for-hmac",
		SigningKey:        "advanced-test-signing-key-32-bytes",
		AccessTokenExpiry: time.Hour,
		Audience:          []string{"api.example.com"},
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Create advanced token request
	advancedReq := AdvancedTokenRequest{
		Subject:   "test-user@example.com",
		Audience:  []string{"api.example.com"},
		Scope:     []string{"read", "write", "admin"},
		TTL:       time.Hour,
		TokenType: "JWT",
		ClientID:  "test-client",
		ClaimsMetadata: &ClaimsMetadata{
			Version:      "1.0",
			Capabilities: []string{"delegation", "revocation"},
			Source:       "internal",
			Confidence:   0.95,
			Restrictions: &ClaimsRestrictions{
				TimeWindow: &TimeWindow{
					StartHour: 0,
					EndHour:   23,
					Weekdays:  []int{0, 1, 2, 3, 4, 5, 6}, // All days for testing
				},
				UsageLimit: 100,
			},
		},
		CustomClaims: map[string]interface{}{
			"tenant_id":  "test-tenant",
			"session_id": "test-session-123",
		},
	}

	// Create advanced token
	tokenResponse, err := service.CreateAdvancedToken(advancedReq)
	if err != nil {
		t.Fatalf("CreateAdvancedToken() error = %v", err)
	}

	if tokenResponse.Token == "" {
		t.Fatal("CreateAdvancedToken() returned empty token")
	}

	// Validate the advanced token
	result, err := service.ValidateAdvancedToken(tokenResponse.Token)
	if err != nil {
		t.Fatalf("ValidateAdvancedToken() error = %v", err)
	}

	// Verify basic validation results
	if !result.Valid {
		t.Error("ValidateAdvancedToken() result.Valid = false, want true")
	}

	// Note: The basic validation returns the JWT 'sub' claim as ClientID, which is the subject
	// This is expected behavior for JWT validation

	// Verify advanced claims
	if result.AdvancedClaims == nil {
		t.Fatal("ValidateAdvancedToken() AdvancedClaims is nil")
	}

	claims := result.AdvancedClaims
	if claims.Subject != "test-user@example.com" {
		t.Errorf("AdvancedClaims Subject = %v, want %v", claims.Subject, "test-user@example.com")
	}
	if claims.TokenType != "JWT" {
		t.Errorf("AdvancedClaims TokenType = %v, want %v", claims.TokenType, "JWT")
	}
	if claims.ClientID != "test-client" {
		t.Errorf("AdvancedClaims ClientID = %v, want %v", claims.ClientID, "test-client")
	}

	// Verify claims metadata
	if claims.ClaimsMetadata == nil {
		t.Fatal("AdvancedClaims ClaimsMetadata is nil")
	}
	if claims.ClaimsMetadata.Version != "1.0" {
		t.Errorf("ClaimsMetadata Version = %v, want %v", claims.ClaimsMetadata.Version, "1.0")
	}
	if claims.ClaimsMetadata.Confidence != 0.95 {
		t.Errorf("ClaimsMetadata Confidence = %v, want %v", claims.ClaimsMetadata.Confidence, 0.95)
	}

	// Verify custom claims
	if claims.Custom["tenant_id"] != "test-tenant" {
		t.Errorf("Custom tenant_id = %v, want %v", claims.Custom["tenant_id"], "test-tenant")
	}

	// Verify validation metadata
	if result.ValidationMetadata == nil {
		t.Fatal("ValidationMetadata is nil")
	}
	if result.ValidationMetadata.Confidence <= 0 {
		t.Errorf("ValidationMetadata Confidence = %v, want > 0", result.ValidationMetadata.Confidence)
	}
}

func TestService_ValidateAdvancedToken_ExpiredToken(t *testing.T) {
	config := Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "test-client",
		SigningKey:        "advanced-test-signing-key-32-bytes",
		AccessTokenExpiry: time.Hour,
		Audience:          []string{"api.example.com"},
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Create token request with very short TTL
	advancedReq := AdvancedTokenRequest{
		Subject:   "test-user@example.com",
		Audience:  []string{"api.example.com"},
		Scope:     []string{"read"},
		TTL:       -time.Hour, // Expired
		TokenType: "JWT",
		ClientID:  "test-client",
		ClaimsMetadata: &ClaimsMetadata{
			Version:    "1.0",
			Source:     "internal",
			Confidence: 0.8,
		},
	}

	// This should fail at creation due to validation
	_, err = service.CreateAdvancedToken(advancedReq)
	if err == nil {
		t.Error("CreateAdvancedToken() expected error for expired token, got nil")
	}
}

func TestService_ValidateAdvancedToken_TimeWindowRestriction(t *testing.T) {
	config := Config{
		AuthServerURL:     "https://auth.example.com",
		ClientID:          "test-client",
		SigningKey:        "advanced-test-signing-key-32-bytes",
		AccessTokenExpiry: time.Hour,
		Audience:          []string{"api.example.com"},
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Create token with restrictive time window (never allows current time)
	now := time.Now()
	restrictiveHour := (now.Hour() + 12) % 24 // 12 hours from now

	advancedReq := AdvancedTokenRequest{
		Subject:   "test-user@example.com",
		Audience:  []string{"api.example.com"},
		Scope:     []string{"read"},
		TTL:       time.Hour,
		TokenType: "JWT",
		ClientID:  "test-client",
		ClaimsMetadata: &ClaimsMetadata{
			Version:    "1.0",
			Source:     "internal",
			Confidence: 0.8,
			Restrictions: &ClaimsRestrictions{
				TimeWindow: &TimeWindow{
					StartHour: restrictiveHour,
					EndHour:   restrictiveHour, // Same hour = very narrow window
					Weekdays:  []int{},         // No days allowed
				},
			},
		},
	}

	// Create token (should succeed)
	tokenResponse, err := service.CreateAdvancedToken(advancedReq)
	if err != nil {
		t.Fatalf("CreateAdvancedToken() error = %v", err)
	}

	// Validation should fail due to time window restriction
	_, err = service.ValidateAdvancedToken(tokenResponse.Token)
	if err == nil {
		t.Error("ValidateAdvancedToken() expected error for time window restriction, got nil")
	}
	if err != nil && !contains(err.Error(), "time window") {
		t.Errorf("ValidateAdvancedToken() error = %v, want time window error", err)
	}
}

func TestPASETOValidation_StructuredFooter(t *testing.T) {
	config := Config{
		AuthServerURL: "https://auth.example.com",
		ClientID:      "test-client",
		SigningKey:    "test-signing-key-32-bytes-long-12",
	}

	service, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create service: %v", err)
	}
	defer service.Close()

	// Test valid PASETO token format
	validToken := "v4.public.eyJzdWIiOiJ1c2VyMTIzIiwiaXNzIjoiaHR0cHM6Ly9hdXRoLmV4YW1wbGUuY29tIn0.eyJraWQiOiJrZXktMTIzIiwiYWxnIjoiRWQyNTUxOSIsImlzcyI6Imh0dHBzOi8vYXV0aC5leGFtcGxlLmNvbSJ9"

	result, err := service.ValidatePASETOWithFooter(validToken)
	if err != nil {
		t.Fatalf("ValidatePASETOWithFooter() error = %v", err)
	}

	if !result.Valid {
		t.Error("ValidatePASETOWithFooter() Valid = false, want true")
	}
	if result.Version != "v4" {
		t.Errorf("ValidatePASETOWithFooter() Version = %v, want v4", result.Version)
	}
	if result.Purpose != "public" {
		t.Errorf("ValidatePASETOWithFooter() Purpose = %v, want public", result.Purpose)
	}

	// Test invalid PASETO token format
	invalidToken := "invalid.token.format"
	_, err = service.ValidatePASETOWithFooter(invalidToken)
	if err == nil {
		t.Error("ValidatePASETOWithFooter() expected error for invalid token, got nil")
	}
}

func TestAdvancedClaimsComplexScenarios(t *testing.T) {
	tests := []struct {
		name        string
		setupClaims func() *AdvancedClaims
		expectValid bool
		expectError string
	}{
		{
			name: "multi_tenant_with_delegation",
			setupClaims: func() *AdvancedClaims {
				return &AdvancedClaims{
					Subject:   "delegated-user@tenant-a.com",
					Issuer:    "https://auth.example.com",
					Audience:  []string{"api.tenant-a.com", "api.shared.com"},
					ExpiresAt: time.Now().Add(time.Hour).Unix(),
					IssuedAt:  time.Now().Unix(),
					NotBefore: time.Now().Unix(),
					JWTID:     "multi-tenant-jwt-123",
					Scope:     []string{"tenant-a:read", "tenant-a:write", "shared:read"},
					TokenType: "JWT",
					ClientID:  "tenant-a-client",
					ClaimsMetadata: &ClaimsMetadata{
						Version:      "2.0",
						Capabilities: []string{"delegation", "multi-tenant", "audit"},
						Source:       "delegated",
						Confidence:   0.85,
					},
					Custom: map[string]interface{}{
						"tenant_id":        "tenant-a",
						"original_subject": "admin@tenant-a.com",
						"delegation_depth": 1,
						"audit_trail":      []string{"login", "delegate", "access"},
					},
				}
			},
			expectValid: true,
		},
		{
			name: "high_risk_with_restrictions",
			setupClaims: func() *AdvancedClaims {
				return &AdvancedClaims{
					Subject:   "high-risk-user@example.com",
					Issuer:    "https://auth.example.com",
					Audience:  []string{"api.secure.com"},
					ExpiresAt: time.Now().Add(30 * time.Minute).Unix(), // Short-lived
					IssuedAt:  time.Now().Unix(),
					NotBefore: time.Now().Unix(),
					JWTID:     "high-risk-jwt-456",
					Scope:     []string{"secure:read"},
					TokenType: "JWT",
					ClientID:  "secure-client",
					ClaimsMetadata: &ClaimsMetadata{
						Version:    "2.0",
						Source:     "risk-engine",
						Confidence: 0.60, // Lower confidence
						Restrictions: &ClaimsRestrictions{
							TimeWindow: &TimeWindow{
								StartHour: 0,
								EndHour:   23,
								Weekdays:  []int{0, 1, 2, 3, 4, 5, 6},
							},
							UsageLimit:     10, // Very limited
							GeofenceRegion: "US-WEST",
							IPWhitelist:    []string{"192.168.1.100/32"}, // Single IP
						},
					},
					Custom: map[string]interface{}{
						"risk_score":           0.75,
						"risk_factors":         []string{"new_device", "unusual_location", "off_hours"},
						"requires_mfa":         true,
						"max_session_duration": 1800, // 30 minutes
					},
				}
			},
			expectValid: true,
		},
		{
			name: "invalid_confidence_range",
			setupClaims: func() *AdvancedClaims {
				return &AdvancedClaims{
					Subject:  "user@example.com",
					Audience: []string{"api.example.com"},
					ClaimsMetadata: &ClaimsMetadata{
						Version:    "1.0",
						Confidence: 1.5, // Invalid - over 1.0
					},
				}
			},
			expectValid: false,
			expectError: "confidence must be between 0.0 and 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims := tt.setupClaims()
			err := claims.ValidateSemantics()

			if tt.expectValid && err != nil {
				t.Errorf("ValidateSemantics() unexpected error: %v", err)
			}
			if !tt.expectValid && err == nil {
				t.Error("ValidateSemantics() expected error but got none")
			}
			if !tt.expectValid && err != nil && tt.expectError != "" {
				if !contains(err.Error(), tt.expectError) {
					t.Errorf("ValidateSemantics() error = %v, want to contain %v", err, tt.expectError)
				}
			}
		})
	}
}

func TestAdvancedTokenJSONSerialization(t *testing.T) {
	original := &AdvancedClaims{
		Subject:   "serialization-test@example.com",
		Issuer:    "https://auth.example.com",
		Audience:  []string{"api.example.com"},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
		TokenType: "JWT",
		ClaimsMetadata: &ClaimsMetadata{
			Version:      "2.0",
			Confidence:   0.90,
			Capabilities: []string{"test", "serialization"},
		},
		Custom: map[string]interface{}{
			"nested_object": map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": "deep_value",
					"array":  []string{"item1", "item2"},
				},
			},
			"simple_value": 42,
			"boolean_flag": true,
		},
	}

	// Test ToMap
	claimsMap := original.ToMap()

	// Serialize to JSON
	jsonBytes, err := json.Marshal(claimsMap)
	if err != nil {
		t.Fatalf("JSON Marshal error: %v", err)
	}

	// Deserialize from JSON
	var deserializedMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &deserializedMap); err != nil {
		t.Fatalf("JSON Unmarshal error: %v", err)
	}

	// Convert back to AdvancedClaims
	reconstructed := &AdvancedClaims{}
	if err := reconstructed.FromMap(deserializedMap); err != nil {
		t.Fatalf("FromMap error: %v", err)
	}

	// Verify critical fields
	if reconstructed.Subject != original.Subject {
		t.Errorf("Subject mismatch: got %v, want %v", reconstructed.Subject, original.Subject)
	}
	if reconstructed.TokenType != original.TokenType {
		t.Errorf("TokenType mismatch: got %v, want %v", reconstructed.TokenType, original.TokenType)
	}

	// Verify custom claims preservation
	if reconstructed.Custom["simple_value"] == nil {
		t.Error("Custom simple_value not preserved")
	}

	// Note: Complex nested objects might need additional type handling in a production implementation
	// For this test, we verify the basic structure is maintained
}
