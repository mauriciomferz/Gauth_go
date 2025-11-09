package poa

import (
	"fmt"
	"testing"
	"time"
)

// TestCreateRFC0115CompliantConfig tests the CreateRFC0115CompliantConfig function
func TestCreateRFC0115CompliantConfig(t *testing.T) {
	config := CreateRFC0115CompliantConfig()

	// Verify it returns RFC0115Config type
	rfcConfig, ok := config.(RFC0115Config)
	if !ok {
		t.Fatalf("CreateRFC0115CompliantConfig() returned type %T, want RFC0115Config", config)
	}

	// Verify all exclusion flags are true
	if !rfcConfig.ExcludeWeb3 {
		t.Errorf("ExcludeWeb3 = false, want true")
	}
	if !rfcConfig.ExcludeAIOperators {
		t.Errorf("ExcludeAIOperators = false, want true")
	}
	if !rfcConfig.ExcludeDNAIdentities {
		t.Errorf("ExcludeDNAIdentities = false, want true")
	}

	// Verify MaxValidityDays is 365
	if rfcConfig.MaxValidityDays != 365 {
		t.Errorf("MaxValidityDays = %d, want 365", rfcConfig.MaxValidityDays)
	}
}

// TestValidateRFC0115Compliance_Config tests ValidateRFC0115Compliance with RFC0115Config
func TestValidateRFC0115Compliance_Config(t *testing.T) {
	tests := []struct {
		name    string
		config  RFC0115Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid config - all flags true, valid days",
			config: RFC0115Config{
				ExcludeWeb3:         true,
				ExcludeAIOperators:  true,
				ExcludeDNAIdentities: true,
				MaxValidityDays:     365,
			},
			wantErr: false,
		},
		{
			name: "Invalid - ExcludeWeb3 false",
			config: RFC0115Config{
				ExcludeWeb3:         false,
				ExcludeAIOperators:  true,
				ExcludeDNAIdentities: true,
				MaxValidityDays:     365,
			},
			wantErr: true,
			errMsg:  "all exclusion flags must be true",
		},
		{
			name: "Invalid - ExcludeAIOperators false",
			config: RFC0115Config{
				ExcludeWeb3:         true,
				ExcludeAIOperators:  false,
				ExcludeDNAIdentities: true,
				MaxValidityDays:     365,
			},
			wantErr: true,
			errMsg:  "all exclusion flags must be true",
		},
		{
			name: "Invalid - ExcludeDNAIdentities false",
			config: RFC0115Config{
				ExcludeWeb3:         true,
				ExcludeAIOperators:  true,
				ExcludeDNAIdentities: false,
				MaxValidityDays:     365,
			},
			wantErr: true,
			errMsg:  "all exclusion flags must be true",
		},
		{
			name: "Invalid - MaxValidityDays zero",
			config: RFC0115Config{
				ExcludeWeb3:         true,
				ExcludeAIOperators:  true,
				ExcludeDNAIdentities: true,
				MaxValidityDays:     0,
			},
			wantErr: true,
			errMsg:  "max validity days out of acceptable bounds",
		},
		{
			name: "Invalid - MaxValidityDays negative",
			config: RFC0115Config{
				ExcludeWeb3:         true,
				ExcludeAIOperators:  true,
				ExcludeDNAIdentities: true,
				MaxValidityDays:     -10,
			},
			wantErr: true,
			errMsg:  "max validity days out of acceptable bounds",
		},
		{
			name: "Invalid - MaxValidityDays exceeds 730",
			config: RFC0115Config{
				ExcludeWeb3:         true,
				ExcludeAIOperators:  true,
				ExcludeDNAIdentities: true,
				MaxValidityDays:     731,
			},
			wantErr: true,
			errMsg:  "max validity days out of acceptable bounds",
		},
		{
			name: "Valid - MaxValidityDays at boundary 730",
			config: RFC0115Config{
				ExcludeWeb3:         true,
				ExcludeAIOperators:  true,
				ExcludeDNAIdentities: true,
				MaxValidityDays:     730,
			},
			wantErr: false,
		},
		{
			name: "Valid - MaxValidityDays minimum boundary 1",
			config: RFC0115Config{
				ExcludeWeb3:         true,
				ExcludeAIOperators:  true,
				ExcludeDNAIdentities: true,
				MaxValidityDays:     1,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRFC0115Compliance(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRFC0115Compliance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				if len(err.Error()) == 0 || err.Error()[:len(tt.errMsg)] != tt.errMsg[:len(tt.errMsg)] {
					// Simple substring check
					found := false
					for i := 0; i < len(err.Error())-len(tt.errMsg)+1; i++ {
						if err.Error()[i:i+len(tt.errMsg)] == tt.errMsg {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("ValidateRFC0115Compliance() error = %v, want substring %v", err, tt.errMsg)
					}
				}
			}
		})
	}
}

// TestValidateRFC0115Compliance_Definition tests ValidateRFC0115Compliance with PoADefinition
// Note: This test uses the actual PoADefinition structure from the implementation
func TestValidateRFC0115Compliance_Definition(t *testing.T) {
	// ValidateRFC0115Compliance for PoADefinition calls ValidatePoADefinition first,
	// then performs additional semantic checks. Most coverage comes from the config tests.
	// This test focuses on ensuring the PoADefinition path is exercised.
	
	t.Run("PoADefinition validation path coverage", func(t *testing.T) {
		// This test ensures the PoADefinition switch case is covered
		// Even an invalid definition will exercise the code path
		def := PoADefinition{
			Parties: Parties{
				Principal: Principal{
					Identity: "", // Will fail ValidatePoADefinition
				},
				AuthorizedClient: AuthorizedClient{
					Identity: "",
				},
			},
		}
		
		err := ValidateRFC0115Compliance(def)
		if err == nil {
			t.Error("ValidateRFC0115Compliance() expected error for invalid definition, got nil")
		}
	})
}

// TestValidateRFC0115Compliance_CompositeMap tests ValidateRFC0115Compliance with map[string]interface{}
func TestValidateRFC0115Compliance_CompositeMap(t *testing.T) {
	validConfig := RFC0115Config{
		ExcludeWeb3:         true,
		ExcludeAIOperators:  true,
		ExcludeDNAIdentities: true,
		MaxValidityDays:     365,
	}

	tests := []struct {
		name    string
		input   map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid composite with only config",
			input: map[string]interface{}{
				"config": validConfig,
			},
			wantErr: false,
		},
		{
			name: "Invalid config in composite",
			input: map[string]interface{}{
				"config": RFC0115Config{
					ExcludeWeb3:         false,
					ExcludeAIOperators:  true,
					ExcludeDNAIdentities: true,
					MaxValidityDays:     365,
				},
			},
			wantErr: true,
			errMsg:  "config invalid",
		},
		{
			name: "Empty composite map",
			input: map[string]interface{}{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRFC0115Compliance(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRFC0115Compliance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && err != nil {
				found := false
				errStr := err.Error()
				for i := 0; i <= len(errStr)-len(tt.errMsg); i++ {
					if errStr[i:i+len(tt.errMsg)] == tt.errMsg {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ValidateRFC0115Compliance() error = %v, want substring %v", err, tt.errMsg)
				}
			}
		})
	}
}

// TestValidateRFC0115Compliance_UnsupportedType tests ValidateRFC0115Compliance with unsupported types
func TestValidateRFC0115Compliance_UnsupportedType(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr bool
	}{
		{
			name:    "Unsupported type - string",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "Unsupported type - int",
			input:   42,
			wantErr: true,
		},
		{
			name:    "Unsupported type - struct",
			input:   struct{ Field string }{Field: "test"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRFC0115Compliance(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRFC0115Compliance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil {
				errStr := err.Error()
				// Check for "unsupported" substring
				found := false
				target := "unsupported"
				for i := 0; i <= len(errStr)-len(target); i++ {
					if errStr[i:i+len(target)] == target {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ValidateRFC0115Compliance() error = %v, want to contain 'unsupported'", err)
				}
			}
		})
	}
}

// TestCanonicalDigest tests the CanonicalDigest function
func TestCanonicalDigest(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		poa      *ProofOfAuthorization
		wantEmpty bool
		checkPrefix string
	}{
		{
			name:     "Nil POA returns empty string",
			poa:      nil,
			wantEmpty: true,
		},
		{
			name: "Basic POA without delegation or attestation",
			poa: &ProofOfAuthorization{
				ID:        "poa_001",
				Subject:   "user_001",
				Resource:  "resource_001",
				Action:    "read",
				Issuer:    "issuer_001",
				IssuedAt:  baseTime,
				ExpiresAt: baseTime.Add(24 * time.Hour),
				Scope:     []string{"read", "list"},
			},
			wantEmpty: false,
			checkPrefix: "sha256:",
		},
		{
			name: "POA with delegation",
			poa: &ProofOfAuthorization{
				ID:        "poa_002",
				Subject:   "user_002",
				Resource:  "resource_002",
				Action:    "write",
				Issuer:    "issuer_002",
				IssuedAt:  baseTime,
				ExpiresAt: baseTime.Add(24 * time.Hour),
				Scope:     []string{"write"},
				Delegation: &Delegation{
					DelegatedBy: "admin",
					DelegatedTo: "user_002",
					DelegatedAt: baseTime,
					ExpiresAt:   baseTime.Add(12 * time.Hour),
					Scope:       []string{"write"},
					Revocable:   true,
				},
			},
			wantEmpty: false,
			checkPrefix: "sha256:",
		},
		{
			name: "POA with attestation",
			poa: &ProofOfAuthorization{
				ID:        "poa_003",
				Subject:   "user_003",
				Resource:  "resource_003",
				Action:    "delete",
				Issuer:    "issuer_003",
				IssuedAt:  baseTime,
				ExpiresAt: baseTime.Add(24 * time.Hour),
				Scope:     []string{"delete"},
				Attestation: &Attestation{
					AttestedBy:    "verifier",
					AttestedAt:    baseTime,
					Confidence:    0.95,
					ValidityScore: 0.90,
					Evidence: map[string]interface{}{
						"proof": "signature",
					},
				},
			},
			wantEmpty: false,
			checkPrefix: "sha256:",
		},
		{
			name: "POA with both delegation and attestation",
			poa: &ProofOfAuthorization{
				ID:        "poa_004",
				Subject:   "user_004",
				Resource:  "resource_004",
				Action:    "execute",
				Issuer:    "issuer_004",
				IssuedAt:  baseTime,
				ExpiresAt: baseTime.Add(24 * time.Hour),
				Scope:     []string{"execute"},
				Delegation: &Delegation{
					DelegatedBy: "supervisor",
					DelegatedTo: "user_004",
					DelegatedAt: baseTime,
					ExpiresAt:   baseTime.Add(12 * time.Hour),
					Scope:       []string{"execute"},
					Revocable:   false,
				},
				Attestation: &Attestation{
					AttestedBy:    "auditor",
					AttestedAt:    baseTime,
					Confidence:    0.99,
					ValidityScore: 0.98,
					Evidence: map[string]interface{}{
						"log": "audit_trail",
					},
				},
			},
			wantEmpty: false,
			checkPrefix: "sha256:",
		},
		{
			name: "Empty POA",
			poa: &ProofOfAuthorization{},
			wantEmpty: false,
			checkPrefix: "sha256:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			digest := CanonicalDigest(tt.poa)
			if tt.wantEmpty {
				if digest != "" {
					t.Errorf("CanonicalDigest() = %v, want empty string", digest)
				}
				return
			}
			if digest == "" {
				t.Errorf("CanonicalDigest() returned empty string, want non-empty")
				return
			}
			if tt.checkPrefix != "" {
				// Check prefix
				if len(digest) < len(tt.checkPrefix) {
					t.Errorf("CanonicalDigest() = %v, too short for prefix %v", digest, tt.checkPrefix)
					return
				}
				if digest[:len(tt.checkPrefix)] != tt.checkPrefix {
					t.Errorf("CanonicalDigest() = %v, want prefix %v", digest, tt.checkPrefix)
				}
			}
		})
	}
}

// TestCanonicalDigest_Deterministic tests that same POA produces same digest
func TestCanonicalDigest_Deterministic(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	poa1 := &ProofOfAuthorization{
		ID:        "poa_deterministic",
		Subject:   "user_det",
		Resource:  "resource_det",
		Action:    "read",
		Issuer:    "issuer_det",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"read", "write"},
	}

	// Create identical POA
	poa2 := &ProofOfAuthorization{
		ID:        "poa_deterministic",
		Subject:   "user_det",
		Resource:  "resource_det",
		Action:    "read",
		Issuer:    "issuer_det",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"read", "write"},
	}

	digest1 := CanonicalDigest(poa1)
	digest2 := CanonicalDigest(poa2)

	if digest1 != digest2 {
		t.Errorf("CanonicalDigest() not deterministic: %v != %v", digest1, digest2)
	}
}

// TestCanonicalDigest_Different tests that different POAs produce different digests
func TestCanonicalDigest_Different(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	poa1 := &ProofOfAuthorization{
		ID:        "poa_001",
		Subject:   "user_001",
		Resource:  "resource_001",
		Action:    "read",
		Issuer:    "issuer_001",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"read"},
	}

	poa2 := &ProofOfAuthorization{
		ID:        "poa_002",
		Subject:   "user_002",
		Resource:  "resource_002",
		Action:    "write",
		Issuer:    "issuer_002",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"write"},
	}

	digest1 := CanonicalDigest(poa1)
	digest2 := CanonicalDigest(poa2)

	if digest1 == digest2 {
		t.Errorf("CanonicalDigest() produced same digest for different POAs: %v", digest1)
	}
}

// TestVerifyDigest tests the VerifyDigest function
func TestVerifyDigest(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		poa  *ProofOfAuthorization
		want bool
	}{
		{
			name: "Nil POA returns false",
			poa:  nil,
			want: false,
		},
		{
			name: "POA with empty Digest returns false",
			poa: &ProofOfAuthorization{
				ID:        "poa_001",
				Subject:   "user_001",
				Resource:  "resource_001",
				Action:    "read",
				Issuer:    "issuer_001",
				IssuedAt:  baseTime,
				ExpiresAt: baseTime.Add(24 * time.Hour),
				Scope:     []string{"read"},
				Digest:    "",
			},
			want: false,
		},
		{
			name: "POA with incorrect Digest returns false",
			poa: &ProofOfAuthorization{
				ID:        "poa_002",
				Subject:   "user_002",
				Resource:  "resource_002",
				Action:    "write",
				Issuer:    "issuer_002",
				IssuedAt:  baseTime,
				ExpiresAt: baseTime.Add(24 * time.Hour),
				Scope:     []string{"write"},
				Digest:    "sha256:incorrect_digest",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := VerifyDigest(tt.poa); got != tt.want {
				t.Errorf("VerifyDigest() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestVerifyDigest_ValidDigest tests VerifyDigest with correct digest
func TestVerifyDigest_ValidDigest(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	poa := &ProofOfAuthorization{
		ID:        "poa_valid",
		Subject:   "user_valid",
		Resource:  "resource_valid",
		Action:    "execute",
		Issuer:    "issuer_valid",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"execute", "read"},
	}

	// Compute and set the correct digest
	poa.Digest = CanonicalDigest(poa)

	if !VerifyDigest(poa) {
		t.Errorf("VerifyDigest() = false, want true for POA with correct digest")
	}
}

// TestVerifyDigest_ModifiedPOA tests that verification fails after POA modification
func TestVerifyDigest_ModifiedPOA(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	poa := &ProofOfAuthorization{
		ID:        "poa_modified",
		Subject:   "user_modified",
		Resource:  "resource_modified",
		Action:    "read",
		Issuer:    "issuer_modified",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"read"},
	}

	// Set correct digest
	poa.Digest = CanonicalDigest(poa)

	// Verify it's valid
	if !VerifyDigest(poa) {
		t.Fatal("VerifyDigest() should be true before modification")
	}

	// Modify the POA
	poa.Action = "write"

	// Verify it's now invalid
	if VerifyDigest(poa) {
		t.Errorf("VerifyDigest() = true after modification, want false")
	}
}

// TestCanonicalDigest_MetadataExcluded tests that metadata doesn't affect digest
func TestCanonicalDigest_MetadataExcluded(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	
	poa1 := &ProofOfAuthorization{
		ID:        "poa_meta",
		Subject:   "user_meta",
		Resource:  "resource_meta",
		Action:    "read",
		Issuer:    "issuer_meta",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"read"},
		Metadata: map[string]interface{}{
			"key1": "value1",
			"key2": 42,
		},
	}

	poa2 := &ProofOfAuthorization{
		ID:        "poa_meta",
		Subject:   "user_meta",
		Resource:  "resource_meta",
		Action:    "read",
		Issuer:    "issuer_meta",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"read"},
		Metadata: map[string]interface{}{
			"key3": "different_value",
			"key4": 99,
		},
	}

	digest1 := CanonicalDigest(poa1)
	digest2 := CanonicalDigest(poa2)

	if digest1 != digest2 {
		t.Errorf("CanonicalDigest() affected by metadata: %v != %v", digest1, digest2)
	}
}

// TestCanonicalDigest_AttestationEvidenceExcluded tests that attestation evidence doesn't affect digest
func TestCanonicalDigest_AttestationEvidenceExcluded(t *testing.T) {
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	
	poa1 := &ProofOfAuthorization{
		ID:        "poa_att_evi",
		Subject:   "user_att_evi",
		Resource:  "resource_att_evi",
		Action:    "read",
		Issuer:    "issuer_att_evi",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"read"},
		Attestation: &Attestation{
			AttestedBy:    "verifier",
			AttestedAt:    baseTime,
			Confidence:    0.95,
			ValidityScore: 0.90,
			Evidence: map[string]interface{}{
				"proof": "signature1",
			},
		},
	}

	poa2 := &ProofOfAuthorization{
		ID:        "poa_att_evi",
		Subject:   "user_att_evi",
		Resource:  "resource_att_evi",
		Action:    "read",
		Issuer:    "issuer_att_evi",
		IssuedAt:  baseTime,
		ExpiresAt: baseTime.Add(24 * time.Hour),
		Scope:     []string{"read"},
		Attestation: &Attestation{
			AttestedBy:    "verifier",
			AttestedAt:    baseTime,
			Confidence:    0.95,
			ValidityScore: 0.90,
			Evidence: map[string]interface{}{
				"proof": "signature2",
				"extra": "data",
			},
		},
	}

	digest1 := CanonicalDigest(poa1)
	digest2 := CanonicalDigest(poa2)

	if digest1 != digest2 {
		t.Errorf("CanonicalDigest() affected by attestation evidence: %v != %v", digest1, digest2)
	}
}

// Helper function for substring check
func substringCheck(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Custom error check to avoid importing strings package
func errorContains(err error, substr string) bool {
	if err == nil {
		return false
	}
	return substringCheck(fmt.Sprintf("%v", err), substr)
}

// ============================================================================
// Session 34: Additional ValidateRFC0115Compliance edge cases
// ============================================================================

func TestValidateRFC0115Compliance_UnsupportedInputType(t *testing.T) {
	// Test with completely unsupported type (not RFC0115Config, not PoADefinition, not map)
	tests := []interface{}{
		123,
		"string",
		[]string{"array"},
		true,
		struct{ X int }{X: 42},
	}

	for _, input := range tests {
		t.Run(fmt.Sprintf("Type_%T", input), func(t *testing.T) {
			err := ValidateRFC0115Compliance(input)
			if err == nil {
				t.Errorf("Expected error for unsupported type %T", input)
			}
			if !errorContains(err, "unsupported RFC0115 compliance object type") {
				t.Errorf("Error = %v, want 'unsupported RFC0115 compliance object type'", err)
			}
		})
	}
}

func TestValidateRFC0115Compliance_PoADefinition_NoRegions(t *testing.T) {
	// Test PoADefinition validation when ApplicableRegions is empty
	baseTime := time.Now()

	def := PoADefinition{
		Parties: Parties{
			Principal: Principal{
				Identity: "test-principal",
			},
			AuthorizedClient: AuthorizedClient{
				Identity: "test-client",
			},
		},
		Authorization: AuthorizationScope{
			ApplicableSectors: []IndustrySector{SectorFinancialInsurance},
			ApplicableRegions: []GeographicScope{}, // Empty!
			AuthorizedActions: AuthorizedActions{
				Transactions: []Transaction{TransactionLoan},
			},
		},
		Requirements: Requirements{
			ValidityPeriod: ValidityPeriod{
				StartTime: baseTime,
				EndTime:   baseTime.Add(24 * time.Hour),
			},
		},
	}

	err := ValidateRFC0115Compliance(def)
	if err == nil {
		t.Fatal("Expected error for empty ApplicableRegions")
	}
	if !errorContains(err, "at least one region") {
		t.Errorf("Error = %v, want 'at least one region'", err)
	}
}

func TestValidateRFC0115Compliance_PoADefinition_NoActions(t *testing.T) {
	// Test PoADefinition validation when all action arrays are empty
	baseTime := time.Now()

	def := PoADefinition{
		Parties: Parties{
			Principal: Principal{
				Identity: "test-principal",
			},
			AuthorizedClient: AuthorizedClient{
				Identity: "test-client",
			},
		},
		Authorization: AuthorizationScope{
			ApplicableSectors: []IndustrySector{SectorFinancialInsurance},
			ApplicableRegions: []GeographicScope{
				{Type: GeoTypeNational, Identifier: "US"},
			},
			AuthorizedActions: AuthorizedActions{
				Transactions:       []Transaction{},
				Decisions:          []Decision{},
				NonPhysicalActions: []NonPhysicalAction{},
			},
		},
		Requirements: Requirements{
			ValidityPeriod: ValidityPeriod{
				StartTime: baseTime,
				EndTime:   baseTime.Add(24 * time.Hour),
			},
		},
	}

	err := ValidateRFC0115Compliance(def)
	if err == nil {
		t.Fatal("Expected error for no actions")
	}
	if !errorContains(err, "at least one action") {
		t.Errorf("Error = %v, want 'at least one action'", err)
	}
}

func TestValidateRFC0115Compliance_PoADefinition_NegativeDuration(t *testing.T) {
	// Test PoADefinition validation when validity period is negative
	baseTime := time.Now()

	def := PoADefinition{
		Parties: Parties{
			Principal: Principal{
				Identity: "test-principal",
			},
			AuthorizedClient: AuthorizedClient{
				Identity: "test-client",
			},
		},
		Authorization: AuthorizationScope{
			ApplicableSectors: []IndustrySector{SectorFinancialInsurance},
			ApplicableRegions: []GeographicScope{
				{Type: GeoTypeNational, Identifier: "US"},
			},
			AuthorizedActions: AuthorizedActions{
				Transactions: []Transaction{TransactionLoan},
			},
		},
		Requirements: Requirements{
			ValidityPeriod: ValidityPeriod{
				StartTime: baseTime.Add(48 * time.Hour), // After EndTime!
				EndTime:   baseTime,
			},
		},
	}

	err := ValidateRFC0115Compliance(def)
	if err == nil {
		t.Fatal("Expected error for negative duration")
	}
	// Note: ValidatePoADefinition is checked first and reports "end before start"
	// Both errors are acceptable for this test
	if !errorContains(err, "positive duration") && !errorContains(err, "end before start") {
		t.Errorf("Error = %v, want 'positive duration' or 'end before start'", err)
	}
}

func TestValidateRFC0115Compliance_PoADefinition_ExceedsTwoYears(t *testing.T) {
	// Test PoADefinition validation when validity period > 2 years
	baseTime := time.Now()

	def := PoADefinition{
		Parties: Parties{
			Principal: Principal{
				Identity: "test-principal",
			},
			AuthorizedClient: AuthorizedClient{
				Identity: "test-client",
			},
		},
		Authorization: AuthorizationScope{
			ApplicableSectors: []IndustrySector{SectorFinancialInsurance},
			ApplicableRegions: []GeographicScope{
				{Type: GeoTypeNational, Identifier: "US"},
			},
			AuthorizedActions: AuthorizedActions{
				Transactions: []Transaction{TransactionLoan},
			},
		},
		Requirements: Requirements{
			ValidityPeriod: ValidityPeriod{
				StartTime: baseTime,
				EndTime:   baseTime.Add(time.Hour * 24 * 731), // >2 years
			},
		},
	}

	err := ValidateRFC0115Compliance(def)
	if err == nil {
		t.Fatal("Expected error for duration exceeding 2 years")
	}
	if !errorContains(err, "exceeds 2 years") {
		t.Errorf("Error = %v, want 'exceeds 2 years'", err)
	}
}

func TestValidateRFC0115Compliance_PoADefinition_OnlyDecisions(t *testing.T) {
	// Test PoADefinition validation with only Decisions (no Transactions)
	baseTime := time.Now()

	def := PoADefinition{
		Parties: Parties{
			Principal: Principal{
				Identity: "test-principal",
			},
			AuthorizedClient: AuthorizedClient{
				Identity: "test-client",
			},
		},
		Authorization: AuthorizationScope{
			ApplicableSectors: []IndustrySector{SectorFinancialInsurance},
			ApplicableRegions: []GeographicScope{
				{Type: GeoTypeNational, Identifier: "US"},
			},
			AuthorizedActions: AuthorizedActions{
				Decisions: []Decision{DecisionFinancial},
			},
		},
		Requirements: Requirements{
			ValidityPeriod: ValidityPeriod{
				StartTime: baseTime,
				EndTime:   baseTime.Add(24 * time.Hour),
			},
		},
	}

	err := ValidateRFC0115Compliance(def)
	if err != nil {
		t.Errorf("Unexpected error for valid PoADefinition with only Decisions: %v", err)
	}
}

func TestValidateRFC0115Compliance_PoADefinition_OnlyNonPhysicalActions(t *testing.T) {
	// Test PoADefinition validation with only NonPhysicalActions
	baseTime := time.Now()

	def := PoADefinition{
		Parties: Parties{
			Principal: Principal{
				Identity: "test-principal",
			},
			AuthorizedClient: AuthorizedClient{
				Identity: "test-client",
			},
		},
		Authorization: AuthorizationScope{
			ApplicableSectors: []IndustrySector{SectorProfessional},
			ApplicableRegions: []GeographicScope{
				{Type: GeoTypeNational, Identifier: "EU"},
			},
			AuthorizedActions: AuthorizedActions{
				NonPhysicalActions: []NonPhysicalAction{ActionResearching},
			},
		},
		Requirements: Requirements{
			ValidityPeriod: ValidityPeriod{
				StartTime: baseTime,
				EndTime:   baseTime.Add(24 * time.Hour),
			},
		},
	}

	err := ValidateRFC0115Compliance(def)
	if err != nil {
		t.Errorf("Unexpected error for valid PoADefinition with only NonPhysicalActions: %v", err)
	}
}

func TestValidateRFC0115Compliance_CompositeMap_InvalidDefinition(t *testing.T) {
	// Test composite map with invalid definition (missing sectors)
	baseTime := time.Now()

	invalidDef := PoADefinition{
		Parties: Parties{
			Principal: Principal{
				Identity: "test-principal",
			},
			AuthorizedClient: AuthorizedClient{
				Identity: "test-client",
			},
		},
		Authorization: AuthorizationScope{
			ApplicableSectors: []IndustrySector{}, // Empty!
			ApplicableRegions: []GeographicScope{
				{Type: GeoTypeNational, Identifier: "US"},
			},
			AuthorizedActions: AuthorizedActions{
				Transactions: []Transaction{TransactionLoan},
			},
		},
		Requirements: Requirements{
			ValidityPeriod: ValidityPeriod{
				StartTime: baseTime,
				EndTime:   baseTime.Add(24 * time.Hour),
			},
		},
	}

	composite := map[string]interface{}{
		"definition": invalidDef,
	}

	err := ValidateRFC0115Compliance(composite)
	if err == nil {
		t.Fatal("Expected error for invalid definition in composite")
	}
	if !errorContains(err, "definition invalid") {
		t.Errorf("Error = %v, want 'definition invalid'", err)
	}
}

func TestValidateRFC0115Compliance_CompositeMap_BothConfigAndDefinition(t *testing.T) {
	// Test composite map with both config and definition
	baseTime := time.Now()

	validConfig := RFC0115Config{
		ExcludeWeb3:          true,
		ExcludeAIOperators:   true,
		ExcludeDNAIdentities: true,
		MaxValidityDays:      365,
	}

	validDef := PoADefinition{
		Parties: Parties{
			Principal: Principal{
				Identity: "test-principal",
			},
			AuthorizedClient: AuthorizedClient{
				Identity: "test-client",
			},
		},
		Authorization: AuthorizationScope{
			ApplicableSectors: []IndustrySector{SectorFinancialInsurance},
			ApplicableRegions: []GeographicScope{
				{Type: GeoTypeNational, Identifier: "US"},
			},
			AuthorizedActions: AuthorizedActions{
				Transactions: []Transaction{TransactionLoan},
			},
		},
		Requirements: Requirements{
			ValidityPeriod: ValidityPeriod{
				StartTime: baseTime,
				EndTime:   baseTime.Add(24 * time.Hour),
			},
		},
	}

	composite := map[string]interface{}{
		"config":     validConfig,
		"definition": validDef,
	}

	err := ValidateRFC0115Compliance(composite)
	if err != nil {
		t.Errorf("Unexpected error for valid composite with both config and definition: %v", err)
	}
}