// Package gauth - Tests for geographic scope validation
package gauth

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/poa"
	"github.com/mauriciomferz/Gauth_go/pkg/poa/taxonomy"
)

// TestGeographicScopeValidation_Success tests successful validation when jurisdiction is authorized
func TestGeographicScopeValidation_Success(t *testing.T) {
	// Create validator
	validator := NewComplianceValidator(nil, nil, nil)

	// Create PoA with German scope
	poaDef := &poa.PoADefinition{
		Parties: poa.Parties{
			AuthorizedClient: poa.AuthorizedClient{
				StatusEnum: poa.OperationalStatusActive,
			},
		},
		Authorization: poa.AuthorizationScope{
			ApplicableRegions: []poa.GeographicScope{
				{
					Type:       poa.GeoTypeNational,
					Identifier: "DE",
					Name:       "Germany",
				},
			},
			AuthorizedActions: poa.AuthorizedActions{
				NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
					taxonomy.ActionNonPhysicalAnalyzing,
				},
			},
		},
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(24 * time.Hour),
			},
		},
	}

	// Create request for German jurisdiction
	request := &ExtendedAuthorizationRequest{
		AuthorizationRequest: &AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"read"},
		},
		PowerOfAttorney: poaDef,
		Jurisdiction:    "DE",
		RequestTime:     time.Now(),
	}

	// Validate
	result := &RequestComplianceResult{
		Valid:    true,
		Checks:   make(map[string]bool),
		Warnings: []string{},
	}

	err := validator.validateGeographicScope(context.Background(), request, result)
	if err != nil {
		t.Errorf("Expected no error for authorized region, got: %v", err)
	}

	if !result.Checks["geographic_scope"] {
		t.Error("Expected geographic_scope check to pass")
	}
}

// TestGeographicScopeValidation_Failure tests rejection when jurisdiction is not authorized
func TestGeographicScopeValidation_Failure(t *testing.T) {
	// Create validator
	validator := NewComplianceValidator(nil, nil, nil)

	// Create PoA with only German scope
	poaDef := &poa.PoADefinition{
		Parties: poa.Parties{
			AuthorizedClient: poa.AuthorizedClient{
				StatusEnum: poa.OperationalStatusActive,
			},
		},
		Authorization: poa.AuthorizationScope{
			ApplicableRegions: []poa.GeographicScope{
				{
					Type:       poa.GeoTypeNational,
					Identifier: "DE",
					Name:       "Germany",
				},
			},
			AuthorizedActions: poa.AuthorizedActions{
				NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
					taxonomy.ActionNonPhysicalAnalyzing,
				},
			},
		},
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(24 * time.Hour),
			},
		},
	}

	// Create request for US jurisdiction (not authorized)
	request := &ExtendedAuthorizationRequest{
		AuthorizationRequest: &AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"read"},
		},
		PowerOfAttorney: poaDef,
		Jurisdiction:    "US",
		RequestTime:     time.Now(),
	}

	// Validate
	result := &RequestComplianceResult{
		Valid:    true,
		Checks:   make(map[string]bool),
		Warnings: []string{},
	}

	err := validator.validateGeographicScope(context.Background(), request, result)
	if err == nil {
		t.Error("Expected error for unauthorized region, got nil")
	}

	gauthErr, ok := err.(*AgentAuthError)
	if !ok {
		t.Errorf("Expected AgentAuthError, got %T", err)
	} else if gauthErr.Code != "geographic_scope_violation" {
		t.Errorf("Expected error code 'geographic_scope_violation', got '%s'", gauthErr.Code)
	}

	if result.Checks["geographic_scope"] {
		t.Error("Expected geographic_scope check to fail")
	}
}

// TestGeographicScopeValidation_GlobalScope tests that global scope allows any jurisdiction
func TestGeographicScopeValidation_GlobalScope(t *testing.T) {
	// Create validator
	validator := NewComplianceValidator(nil, nil, nil)

	// Create PoA with global scope
	poaDef := &poa.PoADefinition{
		Parties: poa.Parties{
			AuthorizedClient: poa.AuthorizedClient{
				StatusEnum: poa.OperationalStatusActive,
			},
		},
		Authorization: poa.AuthorizationScope{
			ApplicableRegions: []poa.GeographicScope{
				{
					Type: poa.GeoTypeGlobal,
				},
			},
			AuthorizedActions: poa.AuthorizedActions{
				NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
					taxonomy.ActionNonPhysicalAnalyzing,
				},
			},
		},
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(24 * time.Hour),
			},
		},
	}

	// Test with various jurisdictions
	jurisdictions := []string{"US", "DE", "JP", "AU", "CN", "BR"}

	for _, jurisdiction := range jurisdictions {
		request := &ExtendedAuthorizationRequest{
			AuthorizationRequest: &AuthorizationRequest{
				ClientID: "test-client",
				Scopes:   []string{"read"},
			},
			PowerOfAttorney: poaDef,
			Jurisdiction:    jurisdiction,
			RequestTime:     time.Now(),
		}

		result := &RequestComplianceResult{
			Valid:    true,
			Checks:   make(map[string]bool),
			Warnings: []string{},
		}

		err := validator.validateGeographicScope(context.Background(), request, result)
		if err != nil {
			t.Errorf("Global scope should allow jurisdiction %s, got error: %v", jurisdiction, err)
		}

		if !result.Checks["geographic_scope"] {
			t.Errorf("Geographic scope check should pass for jurisdiction %s with global scope", jurisdiction)
		}
	}
}

// TestGeographicScopeValidation_NoScopeStrict tests strict mode rejects PoA without geographic scope
func TestGeographicScopeValidation_NoScopeStrict(t *testing.T) {
	// Create validator in strict mode
	validator := NewComplianceValidator(nil, nil, nil)
	validator.strictMode = true

	// Create PoA with NO geographic scope
	poaDef := &poa.PoADefinition{
		Parties: poa.Parties{
			AuthorizedClient: poa.AuthorizedClient{
				StatusEnum: poa.OperationalStatusActive,
			},
		},
		Authorization: poa.AuthorizationScope{
			ApplicableRegions: []poa.GeographicScope{}, // Empty!
			AuthorizedActions: poa.AuthorizedActions{
				NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
					taxonomy.ActionNonPhysicalAnalyzing,
				},
			},
		},
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(24 * time.Hour),
			},
		},
	}

	request := &ExtendedAuthorizationRequest{
		AuthorizationRequest: &AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"read"},
		},
		PowerOfAttorney: poaDef,
		Jurisdiction:    "DE",
		RequestTime:     time.Now(),
	}

	result := &RequestComplianceResult{
		Valid:    true,
		Checks:   make(map[string]bool),
		Warnings: []string{},
	}

	err := validator.validateGeographicScope(context.Background(), request, result)
	if err == nil {
		t.Error("Expected error in strict mode for PoA without geographic scope, got nil")
	}

	gauthErr, ok := err.(*AgentAuthError)
	if !ok {
		t.Errorf("Expected AgentAuthError, got %T", err)
	} else if gauthErr.Code != "no_geographic_scope" {
		t.Errorf("Expected error code 'no_geographic_scope', got '%s'", gauthErr.Code)
	}
}

// TestGeographicScopeValidation_MultipleRegions tests PoA with multiple authorized regions
func TestGeographicScopeValidation_MultipleRegions(t *testing.T) {
	// Create validator
	validator := NewComplianceValidator(nil, nil, nil)

	// Create PoA with EU countries
	poaDef := &poa.PoADefinition{
		Parties: poa.Parties{
			AuthorizedClient: poa.AuthorizedClient{
				StatusEnum: poa.OperationalStatusActive,
			},
		},
		Authorization: poa.AuthorizationScope{
			ApplicableRegions: []poa.GeographicScope{
				{Type: poa.GeoTypeNational, Identifier: "DE", Name: "Germany"},
				{Type: poa.GeoTypeNational, Identifier: "FR", Name: "France"},
				{Type: poa.GeoTypeNational, Identifier: "IT", Name: "Italy"},
			},
			AuthorizedActions: poa.AuthorizedActions{
				NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
					taxonomy.ActionNonPhysicalAnalyzing,
				},
			},
		},
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(24 * time.Hour),
			},
		},
	}

	// Test authorized regions
	authorizedJurisdictions := []string{"DE", "FR", "IT"}
	for _, jurisdiction := range authorizedJurisdictions {
		request := &ExtendedAuthorizationRequest{
			AuthorizationRequest: &AuthorizationRequest{
				ClientID: "test-client",
				Scopes:   []string{"read"},
			},
			PowerOfAttorney: poaDef,
			Jurisdiction:    jurisdiction,
			RequestTime:     time.Now(),
		}

		result := &RequestComplianceResult{
			Valid:    true,
			Checks:   make(map[string]bool),
			Warnings: []string{},
		}

		err := validator.validateGeographicScope(context.Background(), request, result)
		if err != nil {
			t.Errorf("Expected no error for authorized region %s, got: %v", jurisdiction, err)
		}
	}

	// Test unauthorized region
	unauthorizedRequest := &ExtendedAuthorizationRequest{
		AuthorizationRequest: &AuthorizationRequest{
			ClientID: "test-client",
			Scopes:   []string{"read"},
		},
		PowerOfAttorney: poaDef,
		Jurisdiction:    "GB", // UK not authorized
		RequestTime:     time.Now(),
	}

	result := &RequestComplianceResult{
		Valid:    true,
		Checks:   make(map[string]bool),
		Warnings: []string{},
	}

	err := validator.validateGeographicScope(context.Background(), unauthorizedRequest, result)
	if err == nil {
		t.Error("Expected error for unauthorized region GB, got nil")
	}
}

// TestGeographicScopeValidation_SubdivisionSupport tests ISO 3166-2 subdivision codes
func TestGeographicScopeValidation_SubdivisionSupport(t *testing.T) {
	// Create validator
	validator := NewComplianceValidator(nil, nil, nil)

	// Create PoA for Bavaria (DE-BY) with subdivision inclusion
	poaDef := &poa.PoADefinition{
		Parties: poa.Parties{
			AuthorizedClient: poa.AuthorizedClient{
				StatusEnum: poa.OperationalStatusActive,
			},
		},
		Authorization: poa.AuthorizationScope{
			ApplicableRegions: []poa.GeographicScope{
				{
					Type:                poa.GeoTypeNational,
					Identifier:          "DE",
					Name:                "Germany",
					IncludeSubdivisions: true,
				},
			},
			AuthorizedActions: poa.AuthorizedActions{
				NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
					taxonomy.ActionNonPhysicalAnalyzing,
				},
			},
		},
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now().Add(24 * time.Hour),
			},
		},
	}

	// Test both country level and subdivision
	jurisdictions := []string{"DE", "DE-BY", "DE-NW"}

	for _, jurisdiction := range jurisdictions {
		request := &ExtendedAuthorizationRequest{
			AuthorizationRequest: &AuthorizationRequest{
				ClientID: "test-client",
				Scopes:   []string{"read"},
			},
			PowerOfAttorney: poaDef,
			Jurisdiction:    jurisdiction,
			RequestTime:     time.Now(),
		}

		result := &RequestComplianceResult{
			Valid:    true,
			Checks:   make(map[string]bool),
			Warnings: []string{},
		}

		err := validator.validateGeographicScope(context.Background(), request, result)
		if err != nil {
			t.Errorf("Expected no error for jurisdiction %s with subdivision support, got: %v", jurisdiction, err)
		}
	}
}
