package gauth

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAdvancedClaims_ValidateSemantics(t *testing.T) {
	tests := []struct {
		name    string
		claims  *AdvancedClaims
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid_claims_with_metadata",
			claims: &AdvancedClaims{
				Subject:   "user123",
				Issuer:    "https://auth.example.com",
				Audience:  []string{"api.example.com"},
				ExpiresAt: time.Now().Add(time.Hour).Unix(),
				IssuedAt:  time.Now().Unix(),
				NotBefore: time.Now().Unix(),
				JWTID:     "unique-jwt-id-123",
				TokenType: "JWT",
				ClaimsMetadata: &ClaimsMetadata{
					Version:    "1.0",
					Source:     "internal",
					Confidence: 0.95,
				},
			},
			wantErr: false,
		},
		{
			name: "expired_token",
			claims: &AdvancedClaims{
				Subject:   "user123",
				Audience:  []string{"api.example.com"},
				ExpiresAt: time.Now().Add(-time.Hour).Unix(), // Expired
				IssuedAt:  time.Now().Unix(),
			},
			wantErr: true,
			errMsg:  "token expired",
		},
		{
			name: "future_issued_token",
			claims: &AdvancedClaims{
				Subject:  "user123",
				Audience: []string{"api.example.com"},
				IssuedAt: time.Now().Add(time.Hour).Unix(), // Future
			},
			wantErr: true,
			errMsg:  "token issued in future",
		},
		{
			name: "missing_audience",
			claims: &AdvancedClaims{
				Subject: "user123",
				// Missing audience
			},
			wantErr: true,
			errMsg:  "audience (aud) claim is required",
		},
		{
			name: "missing_subject",
			claims: &AdvancedClaims{
				Audience: []string{"api.example.com"},
				// Missing subject
			},
			wantErr: true,
			errMsg:  "subject (sub) claim is required",
		},
		{
			name: "invalid_token_type",
			claims: &AdvancedClaims{
				Subject:   "user123",
				Audience:  []string{"api.example.com"},
				TokenType: "INVALID_TYPE",
			},
			wantErr: true,
			errMsg:  "invalid token type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.claims.ValidateSemantics()
			if tt.wantErr && err == nil {
				t.Errorf("ValidateSemantics() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("ValidateSemantics() unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateSemantics() error = %v, want to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestClaimsMetadata_Validate(t *testing.T) {
	tests := []struct {
		name     string
		metadata *ClaimsMetadata
		wantErr  bool
		errMsg   string
	}{
		{
			name: "valid_metadata",
			metadata: &ClaimsMetadata{
				Version:    "1.0",
				Source:     "internal",
				Confidence: 0.95,
			},
			wantErr: false,
		},
		{
			name: "missing_version",
			metadata: &ClaimsMetadata{
				Source:     "internal",
				Confidence: 0.95,
			},
			wantErr: true,
			errMsg:  "version is required",
		},
		{
			name: "invalid_confidence_low",
			metadata: &ClaimsMetadata{
				Version:    "1.0",
				Confidence: -0.1,
			},
			wantErr: true,
			errMsg:  "confidence must be between 0.0 and 1.0",
		},
		{
			name: "invalid_confidence_high",
			metadata: &ClaimsMetadata{
				Version:    "1.0",
				Confidence: 1.1,
			},
			wantErr: true,
			errMsg:  "confidence must be between 0.0 and 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("Validate() expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
			if tt.wantErr && err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want to contain %v", err, tt.errMsg)
				}
			}
		})
	}
}

func TestClaimsRestrictions_IsInTimeWindow(t *testing.T) {
	tests := []struct {
		name         string
		restrictions *ClaimsRestrictions
		want         bool
	}{
		{
			name:         "no_restrictions",
			restrictions: &ClaimsRestrictions{},
			want:         true,
		},
		{
			name: "no_time_window",
			restrictions: &ClaimsRestrictions{
				UsageLimit: 100,
			},
			want: true,
		},
		{
			name: "always_allowed_time_window",
			restrictions: &ClaimsRestrictions{
				TimeWindow: &TimeWindow{
					StartHour: 0,
					EndHour:   23,
					Weekdays:  []int{0, 1, 2, 3, 4, 5, 6}, // All days
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.restrictions.IsInTimeWindow()
			if got != tt.want {
				t.Errorf("IsInTimeWindow() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAdvancedClaims_ToMapFromMap(t *testing.T) {
	original := &AdvancedClaims{
		Subject:   "user123",
		Issuer:    "https://auth.example.com",
		Audience:  []string{"api.example.com", "web.example.com"},
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
		IssuedAt:  time.Now().Unix(),
		NotBefore: time.Now().Unix(),
		JWTID:     "unique-jwt-id-123",
		Scope:     []string{"read", "write"},
		TokenType: "JWT",
		ClientID:  "client-123",
		ClaimsMetadata: &ClaimsMetadata{
			Version:    "1.0",
			Source:     "internal",
			Confidence: 0.95,
		},
		Custom: map[string]interface{}{
			"tenant_id":  "tenant-123",
			"session_id": "session-456",
		},
	}

	// Convert to map
	claimsMap := original.ToMap()

	// Verify standard fields are present
	if claimsMap["sub"] != original.Subject {
		t.Errorf("ToMap() sub = %v, want %v", claimsMap["sub"], original.Subject)
	}
	if claimsMap["iss"] != original.Issuer {
		t.Errorf("ToMap() iss = %v, want %v", claimsMap["iss"], original.Issuer)
	}

	// Verify custom fields are present
	if claimsMap["tenant_id"] != "tenant-123" {
		t.Errorf("ToMap() tenant_id = %v, want %v", claimsMap["tenant_id"], "tenant-123")
	}

	// Convert back from map
	reconstructed := &AdvancedClaims{}
	if err := reconstructed.FromMap(claimsMap); err != nil {
		t.Fatalf("FromMap() error = %v", err)
	}

	// Verify reconstruction
	if reconstructed.Subject != original.Subject {
		t.Errorf("FromMap() subject = %v, want %v", reconstructed.Subject, original.Subject)
	}
	if reconstructed.TokenType != original.TokenType {
		t.Errorf("FromMap() token_type = %v, want %v", reconstructed.TokenType, original.TokenType)
	}
	if reconstructed.Custom["tenant_id"] != "tenant-123" {
		t.Errorf("FromMap() custom tenant_id = %v, want %v", reconstructed.Custom["tenant_id"], "tenant-123")
	}
}

func TestCreatePASETOFooter(t *testing.T) {
	keyID := "key-123"
	algorithm := "Ed25519"
	issuer := "https://auth.example.com"
	metadata := map[string]interface{}{
		"version": "1.0",
		"purpose": "authentication",
	}

	footer, err := CreatePASETOFooter(keyID, algorithm, issuer, metadata)
	if err != nil {
		t.Fatalf("CreatePASETOFooter() error = %v", err)
	}

	if footer.KeyID != keyID {
		t.Errorf("CreatePASETOFooter() KeyID = %v, want %v", footer.KeyID, keyID)
	}
	if footer.Algorithm != algorithm {
		t.Errorf("CreatePASETOFooter() Algorithm = %v, want %v", footer.Algorithm, algorithm)
	}
	if footer.Issuer != issuer {
		t.Errorf("CreatePASETOFooter() Issuer = %v, want %v", footer.Issuer, issuer)
	}

	// Test JSON conversion
	jsonStr, err := footer.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON() error = %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("JSON unmarshal error = %v", err)
	}

	if parsed["kid"] != keyID {
		t.Errorf("JSON kid = %v, want %v", parsed["kid"], keyID)
	}
}

func TestExampleAdvancedClaims(t *testing.T) {
	example := ExampleAdvancedClaims()

	// Test that example validates successfully
	if err := example.ValidateSemantics(); err != nil {
		t.Errorf("ExampleAdvancedClaims() validation error = %v", err)
	}

	// Test specific fields
	if example.Subject != "user123" {
		t.Errorf("ExampleAdvancedClaims() Subject = %v, want %v", example.Subject, "user123")
	}
	if example.TokenType != "JWT" {
		t.Errorf("ExampleAdvancedClaims() TokenType = %v, want %v", example.TokenType, "JWT")
	}
	if len(example.Scope) != 3 {
		t.Errorf("ExampleAdvancedClaims() Scope length = %v, want %v", len(example.Scope), 3)
	}

	// Test metadata
	if example.ClaimsMetadata == nil {
		t.Error("ExampleAdvancedClaims() ClaimsMetadata is nil")
	} else {
		if example.ClaimsMetadata.Version != "1.0" {
			t.Errorf("ExampleAdvancedClaims() Version = %v, want %v", example.ClaimsMetadata.Version, "1.0")
		}
		if example.ClaimsMetadata.Confidence != 0.95 {
			t.Errorf("ExampleAdvancedClaims() Confidence = %v, want %v", example.ClaimsMetadata.Confidence, 0.95)
		}
	}

	// Test custom claims
	if len(example.Custom) == 0 {
		t.Error("ExampleAdvancedClaims() Custom claims is empty")
	}
	if example.Custom["tenant_id"] != "tenant-123" {
		t.Errorf("ExampleAdvancedClaims() tenant_id = %v, want %v", example.Custom["tenant_id"], "tenant-123")
	}
}

func TestTokenTypeValidation(t *testing.T) {
	validTypes := []string{"JWT", "PASETO", "access_token", "refresh_token", "id_token", "at+jwt", "rt+jwt"}
	invalidTypes := []string{"INVALID", "custom", "bearer", ""}

	for _, tokenType := range validTypes {
		t.Run("valid_"+tokenType, func(t *testing.T) {
			if !isValidTokenType(tokenType) {
				t.Errorf("isValidTokenType(%v) = false, want true", tokenType)
			}
		})
	}

	for _, tokenType := range invalidTypes {
		t.Run("invalid_"+tokenType, func(t *testing.T) {
			if isValidTokenType(tokenType) {
				t.Errorf("isValidTokenType(%v) = true, want false", tokenType)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
