package auth

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestAuthorizePowerOfAttorney_ValidRequest tests successful PoA authorization
func TestAuthorizePowerOfAttorney_ValidRequest(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	req := PowerOfAttorneyRequest{
		ClientID:     "client-123",
		ResponseType: "code",
		Scope:        "read write",
		RedirectURI:  "https://example.com/callback",
		State:        "state-xyz",
		PowerType:    "full",
		PrincipalID:  "principal-456",
		AIAgentID:    "agent-789",
		Jurisdiction: "US",
		LegalBasis:   "consent",
	}

	resp, err := auth.AuthorizePowerOfAttorney(ctx, req)

	if err != nil {
		t.Fatalf("Expected successful authorization, got error: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected PowerOfAttorneyResponse, got nil")
	}
	if resp.AuthorizationCode == "" {
		t.Error("Expected non-empty authorization code")
	}
	if !resp.LegalCompliance {
		t.Error("Expected legal compliance to be true")
	}
	if resp.AuditRecordID == "" {
		t.Error("Expected non-empty audit record ID")
	}
}

// TestAuthorizePowerOfAttorney_InvalidJurisdiction tests invalid jurisdiction handling
func TestAuthorizePowerOfAttorney_InvalidJurisdiction(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	testCases := []struct {
		name         string
		jurisdiction string
	}{
		{"Empty", ""},
		{"InvalidCountry", "XX"},
		{"Lowercase", "us"},
		{"NotSupported", "CN"},
		{"Random", "ABC123"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := PowerOfAttorneyRequest{
				ClientID:     "client-123",
				PrincipalID:  "principal-456",
				AIAgentID:    "agent-789",
				PowerType:    "full",
				Jurisdiction: tc.jurisdiction,
				Scope:        "read",
			}

			resp, err := auth.AuthorizePowerOfAttorney(ctx, req)

			if err == nil {
				t.Errorf("Expected error for jurisdiction '%s', got success", tc.jurisdiction)
			}
			if resp != nil {
				t.Errorf("Expected nil response for invalid jurisdiction, got: %+v", resp)
			}
			if err != nil && !strings.Contains(err.Error(), "jurisdiction") {
				t.Logf("Error message for '%s': %v", tc.jurisdiction, err)
			}
		})
	}
}

// TestAuthorizePowerOfAttorney_ValidJurisdictions tests all valid jurisdictions
func TestAuthorizePowerOfAttorney_ValidJurisdictions(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	validJurisdictions := []string{"US", "EU", "UK", "DE"}

	for _, jurisdiction := range validJurisdictions {
		t.Run(jurisdiction, func(t *testing.T) {
			req := PowerOfAttorneyRequest{
				ClientID:     "client-123",
				PrincipalID:  "principal-456",
				AIAgentID:    "agent-789",
				PowerType:    "full",
				Jurisdiction: jurisdiction,
				Scope:        "read",
			}

			resp, err := auth.AuthorizePowerOfAttorney(ctx, req)

			if err != nil {
				t.Errorf("Expected success for jurisdiction '%s', got error: %v", jurisdiction, err)
			}
			if resp == nil {
				t.Errorf("Expected response for jurisdiction '%s', got nil", jurisdiction)
			}
		})
	}
}

// TestAuthorizePowerOfAttorney_DisallowedScopes tests disallowed scope handling
func TestAuthorizePowerOfAttorney_DisallowedScopes(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	testCases := []struct {
		name  string
		scope string
	}{
		{"NuclearCodes", "nuclear_launch_codes"},
		{"CriticalInfra", "critical_infra_root"},
		{"MixedDisallowed", "read nuclear_launch_codes write"},
		{"CommaDelimited", "read,nuclear_launch_codes,write"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := PowerOfAttorneyRequest{
				ClientID:     "client-123",
				PrincipalID:  "principal-456",
				AIAgentID:    "agent-789",
				PowerType:    "full",
				Jurisdiction: "US",
				Scope:        tc.scope,
			}

			resp, err := auth.AuthorizePowerOfAttorney(ctx, req)

			if err == nil {
				t.Errorf("Expected error for disallowed scope '%s', got success", tc.scope)
			}
			if resp != nil {
				t.Errorf("Expected nil response for disallowed scope, got: %+v", resp)
			}
			if err != nil && !strings.Contains(err.Error(), "scope") {
				t.Logf("Error message for scope '%s': %v", tc.scope, err)
			}
		})
	}
}

// TestAuthorizePowerOfAttorney_AllowedScopes tests allowed scope handling
func TestAuthorizePowerOfAttorney_AllowedScopes(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	testCases := []struct {
		name  string
		scope string
	}{
		{"Single", "read"},
		{"Multiple", "read write"},
		{"Complex", "read write execute admin"},
		{"CommaDelimited", "read,write,execute"},
		{"MixedDelimiters", "read, write execute"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := PowerOfAttorneyRequest{
				ClientID:     "client-123",
				PrincipalID:  "principal-456",
				AIAgentID:    "agent-789",
				PowerType:    "full",
				Jurisdiction: "US",
				Scope:        tc.scope,
			}

			resp, err := auth.AuthorizePowerOfAttorney(ctx, req)

			if err != nil {
				t.Errorf("Expected success for scope '%s', got error: %v", tc.scope, err)
			}
			if resp == nil {
				t.Errorf("Expected response for scope '%s', got nil", tc.scope)
			}
		})
	}
}

// TestAuthorizePowerOfAttorney_MissingRequiredFields tests missing field validation
func TestAuthorizePowerOfAttorney_MissingRequiredFields(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	testCases := []struct {
		name string
		req  PowerOfAttorneyRequest
	}{
		{
			name: "MissingClientID",
			req: PowerOfAttorneyRequest{
				PrincipalID:  "principal-456",
				AIAgentID:    "agent-789",
				PowerType:    "full",
				Jurisdiction: "US",
			},
		},
		{
			name: "MissingPrincipalID",
			req: PowerOfAttorneyRequest{
				ClientID:     "client-123",
				AIAgentID:    "agent-789",
				PowerType:    "full",
				Jurisdiction: "US",
			},
		},
		{
			name: "MissingAIAgentID",
			req: PowerOfAttorneyRequest{
				ClientID:     "client-123",
				PrincipalID:  "principal-456",
				PowerType:    "full",
				Jurisdiction: "US",
			},
		},
		{
			name: "MissingPowerType",
			req: PowerOfAttorneyRequest{
				ClientID:     "client-123",
				PrincipalID:  "principal-456",
				AIAgentID:    "agent-789",
				Jurisdiction: "US",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := auth.AuthorizePowerOfAttorney(ctx, tc.req)

			if err == nil {
				t.Errorf("Expected error for missing fields in '%s', got success", tc.name)
			}
			if resp != nil {
				t.Errorf("Expected nil response for missing fields, got: %+v", resp)
			}
		})
	}
}

// TestCreateAdvancedDelegation_ValidRequest tests successful delegation creation
func TestCreateAdvancedDelegation_ValidRequest(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	req := DelegationRequest{
		PrincipalID: "principal-123",
		DelegateID:  "delegate-456",
		ValidityPeriod: ValidityPeriod{
			Days: 30,
		},
		AttestationRequirement: AttestationRequirement{
			Attesters: []string{"attester1", "attester2"},
		},
	}

	resp, err := auth.CreateAdvancedDelegation(ctx, req)

	if err != nil {
		t.Fatalf("Expected successful delegation, got error: %v", err)
	}
	if resp == nil {
		t.Fatal("Expected DelegationResponse, got nil")
	}
	if resp.DelegationID == "" {
		t.Error("Expected non-empty delegation ID")
	}
	if resp.Status != "active" {
		t.Errorf("Expected status 'active', got '%s'", resp.Status)
	}
	if resp.ValidUntil.IsZero() {
		t.Error("Expected non-zero ValidUntil timestamp")
	}
	if len(resp.Attestations) == 0 {
		t.Error("Expected non-empty attestations list")
	}
	if resp.ComplianceStatus != "compliant" {
		t.Errorf("Expected compliance status 'compliant', got '%s'", resp.ComplianceStatus)
	}
}

// TestCreateAdvancedDelegation_MissingPrincipalID tests missing principal ID handling
func TestCreateAdvancedDelegation_MissingPrincipalID(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	req := DelegationRequest{
		DelegateID: "delegate-456",
		ValidityPeriod: ValidityPeriod{
			Days: 30,
		},
		AttestationRequirement: AttestationRequirement{
			Attesters: []string{"attester1"},
		},
	}

	resp, err := auth.CreateAdvancedDelegation(ctx, req)

	if err == nil {
		t.Error("Expected error for missing principal ID, got success")
	}
	if resp != nil {
		t.Errorf("Expected nil response, got: %+v", resp)
	}
	if err != nil && !strings.Contains(err.Error(), "principal") {
		t.Logf("Error message: %v", err)
	}
}

// TestCreateAdvancedDelegation_MissingDelegateID tests missing delegate ID handling
func TestCreateAdvancedDelegation_MissingDelegateID(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	req := DelegationRequest{
		PrincipalID: "principal-123",
		ValidityPeriod: ValidityPeriod{
			Days: 30,
		},
		AttestationRequirement: AttestationRequirement{
			Attesters: []string{"attester1"},
		},
	}

	resp, err := auth.CreateAdvancedDelegation(ctx, req)

	if err == nil {
		t.Error("Expected error for missing delegate ID, got success")
	}
	if resp != nil {
		t.Errorf("Expected nil response, got: %+v", resp)
	}
	if err != nil && !strings.Contains(err.Error(), "delegate") {
		t.Logf("Error message: %v", err)
	}
}

// TestCreateAdvancedDelegation_InvalidValidityPeriod tests invalid validity period handling
func TestCreateAdvancedDelegation_InvalidValidityPeriod(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	testCases := []struct {
		name string
		days int
	}{
		{"Zero", 0},
		{"Negative", -10},
		{"VeryNegative", -1000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := DelegationRequest{
				PrincipalID: "principal-123",
				DelegateID:  "delegate-456",
				ValidityPeriod: ValidityPeriod{
					Days: tc.days,
				},
				AttestationRequirement: AttestationRequirement{
					Attesters: []string{"attester1"},
				},
			}

			resp, err := auth.CreateAdvancedDelegation(ctx, req)

			if err == nil {
				t.Errorf("Expected error for validity period %d days, got success", tc.days)
			}
			if resp != nil {
				t.Errorf("Expected nil response, got: %+v", resp)
			}
		})
	}
}

// TestCreateAdvancedDelegation_NoAttesters tests missing attesters handling
func TestCreateAdvancedDelegation_NoAttesters(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	req := DelegationRequest{
		PrincipalID: "principal-123",
		DelegateID:  "delegate-456",
		ValidityPeriod: ValidityPeriod{
			Days: 30,
		},
		AttestationRequirement: AttestationRequirement{
			Attesters: []string{},
		},
	}

	resp, err := auth.CreateAdvancedDelegation(ctx, req)

	if err == nil {
		t.Error("Expected error for no attesters, got success")
	}
	if resp != nil {
		t.Errorf("Expected nil response, got: %+v", resp)
	}
	if err != nil && !strings.Contains(err.Error(), "attester") {
		t.Logf("Error message: %v", err)
	}
}

// TestCreateAdvancedDelegation_ValidityPeriodCalculation tests validity until calculation
func TestCreateAdvancedDelegation_ValidityPeriodCalculation(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	testCases := []struct {
		name string
		days int
	}{
		{"OneDay", 1},
		{"OneWeek", 7},
		{"OneMonth", 30},
		{"OneYear", 365},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := DelegationRequest{
				PrincipalID: "principal-123",
				DelegateID:  "delegate-456",
				ValidityPeriod: ValidityPeriod{
					Days: tc.days,
				},
				AttestationRequirement: AttestationRequirement{
					Attesters: []string{"attester1"},
				},
			}

			before := time.Now()
			resp, err := auth.CreateAdvancedDelegation(ctx, req)
			after := time.Now()

			if err != nil {
				t.Fatalf("Expected success, got error: %v", err)
			}

			expectedMin := before.Add(time.Duration(tc.days) * 24 * time.Hour)
			expectedMax := after.Add(time.Duration(tc.days) * 24 * time.Hour)

			if resp.ValidUntil.Before(expectedMin) {
				t.Errorf("ValidUntil %v is before expected minimum %v", resp.ValidUntil, expectedMin)
			}
			if resp.ValidUntil.After(expectedMax) {
				t.Errorf("ValidUntil %v is after expected maximum %v", resp.ValidUntil, expectedMax)
			}
		})
	}
}

// TestCreateAdvancedDelegation_AttestationList tests attestation list handling
func TestCreateAdvancedDelegation_AttestationList(t *testing.T) {
	auth := NewAuthenticator(nil)
	ctx := context.Background()

	testCases := []struct {
		name      string
		attesters []string
	}{
		{"Single", []string{"attester1"}},
		{"Multiple", []string{"attester1", "attester2", "attester3"}},
		{"Many", []string{"att1", "att2", "att3", "att4", "att5"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := DelegationRequest{
				PrincipalID: "principal-123",
				DelegateID:  "delegate-456",
				ValidityPeriod: ValidityPeriod{
					Days: 30,
				},
				AttestationRequirement: AttestationRequirement{
					Attesters: tc.attesters,
				},
			}

			resp, err := auth.CreateAdvancedDelegation(ctx, req)

			if err != nil {
				t.Fatalf("Expected success, got error: %v", err)
			}
			if len(resp.Attestations) != len(tc.attesters) {
				t.Errorf("Expected %d attestations, got %d", len(tc.attesters), len(resp.Attestations))
			}
			for i, expected := range tc.attesters {
				if i < len(resp.Attestations) && resp.Attestations[i] != expected {
					t.Errorf("Attestation %d: expected '%s', got '%s'", i, expected, resp.Attestations[i])
				}
			}
		})
	}
}
