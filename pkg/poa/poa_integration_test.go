package poa

import (
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/poa/taxonomy"
)

// TestPoADefinition_CompleteValidation tests full PoA definition validation
func TestPoADefinition_CompleteValidation(t *testing.T) {
	t.Run("Valid complete PoA definition", func(t *testing.T) {
		def := PoADefinition{
			Parties: Parties{
				Principal: Principal{
					Type:     PrincipalTypeOrganization,
					Identity: "DE:HRB:12345",
					Organization: &Organization{
						Type:                "GmbH",
						Name:                "Tech Solutions GmbH",
						RegisterEntry:       "HRB12345",
						ManagingDirector:    "Dr. Max Mustermann",
						RegisteredAuthority: true,
					},
				},
				Representative: &Representative{
					Identity:          "AI-OPS-GmbH",
					LegalRelationship: RelationshipOwner,
					RegistrationInfo: &RegistrationInfo{
						RegisteredName:        "AI Operations GmbH",
						RegistrationNumber:    "HRB67890",
						RegisteringAuthority:  "Amtsgericht München",
						RegistrationDate:      "2023-01-15",
						Jurisdiction:          "DE",
						BusinessType:          "GmbH",
						TaxIdentifier:         "DE123456789",
						CommercialRegister:    true,
						PowerOfAttorneyOnFile: true,
					},
					AuthorizationChain: []AuthorizationLink{
						{
							FromParty:     "Tech Solutions GmbH",
							ToParty:       "AI Operations GmbH",
							GrantedDate:   "2024-01-01",
							ExpiryDate:    "2025-12-31",
							DocumentRef:   "POA-2024-001",
							Scope:         "AI client management",
							Revocable:     true,
							SubDelegation: true,
						},
					},
					ContactInformation: &ContactInformation{
						PrimaryContact:    "John Smith",
						Email:             "contact@ai-ops.example.com",
						Phone:             "+49-89-12345678",
						PreferredLanguage: "en",
						Address: &Address{
							Street:     "Hauptstraße 123",
							City:       "München",
							State:      "Bayern",
							PostalCode: "80331",
							Country:    "DE",
						},
					},
				},
				AuthorizedClient: AuthorizedClient{
					TypeEnum:        ClientTypeLLM,
					Identity:        "gpt-4-client-001",
					Version:         "4.0",
					StatusEnum:      OperationalStatusActive,
					CapabilityLevel: CapabilityL3,
					ModelAttributes: &ModelAttributes{
						Architecture:     "transformer",
						ParameterCount:   1750000000000,
						Modalities:       []string{"text", "code"},
						ContextWindow:    128000,
						ReasoningMethods: []string{"chain-of-thought", "reasoning"},
					},
				},
			},
			Authorization: AuthorizationScope{
				AuthorizationType: AuthorizationType{
					RepresentationType: RepresentationSole,
					SignatureType:      SignatureSingle,
					SubProxyAuthority:  false,
					Restrictions:       []string{"no_financial_transactions_above_10000_EUR"},
				},
				ApplicableSectors: []taxonomy.IndustrySector{
					{
						Code:        taxonomy.SectorInfoCommunication,
						Description: "Information and Communication",
						Authorized:  true,
					},
					{
						Code:        taxonomy.SectorProfessionalScience,
						Description: "Professional, Scientific and Technical Activities",
						Authorized:  true,
					},
				},
				ApplicableRegions: []GeographicScope{
					{
						Type:       GeoTypeNational,
						Identifier: "DE",
						Name:       "Germany",
					},
					{
						Type:       GeoTypeNational,
						Identifier: "FR",
						Name:       "France",
					},
				},
				AuthorizedActions: AuthorizedActions{
					Transactions: []taxonomy.TransactionType{
						taxonomy.TransactionPurchase,
						taxonomy.TransactionPayment,
					},
					Decisions: []taxonomy.DecisionType{
						taxonomy.DecisionOperational,
					},
					NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
						taxonomy.ActionNonPhysicalResearching,
						taxonomy.ActionNonPhysicalDataAggregation,
					},
				},
			},
			Requirements: Requirements{
				ValidityPeriod: ValidityPeriod{
					StartTime: time.Now().Add(-24 * time.Hour),
					EndTime:   time.Now().Add(365 * 24 * time.Hour),
					TerminationConditions: []string{
						"revocation_by_principal",
						"expiration",
					},
				},
				FormalRequirements: FormalRequirements{
					NotarialCertification:  true,
					IDVerificationRequired: true,
					DigitalSignatures:      true,
				},
				SecurityCompliance: SecurityCompliance{
					CommunicationProtocols: []string{"TLS 1.3", "mTLS"},
					SecurityProperties:     []string{"end-to-end-encryption", "audit-logging"},
					ComplianceInfo:         []string{"GDPR", "NIS2"},
					UpdateMechanism:        "automated",
				},
				JurisdictionLaw: JurisdictionLaw{
					Language:            "de",
					GoverningLaw:        "German Commercial Code (HGB)",
					PlaceOfJurisdiction: "München, Germany",
				},
			},
		}

		err := ValidatePoADefinition(def)
		if err != nil {
			t.Errorf("ValidatePoADefinition() error = %v, want nil", err)
		}
	})

	t.Run("Missing principal identity", func(t *testing.T) {
		def := PoADefinition{
			Parties: Parties{
				Principal:        Principal{Identity: "", Type: PrincipalTypeOrganization},
				AuthorizedClient: AuthorizedClient{Identity: "client-1", TypeEnum: ClientTypeLLM},
			},
			Requirements: Requirements{
				ValidityPeriod: ValidityPeriod{
					StartTime: time.Now(),
					EndTime:   time.Now().Add(time.Hour),
				},
			},
		}

		err := ValidatePoADefinition(def)
		if err == nil {
			t.Error("ValidatePoADefinition() expected error for missing principal identity, got nil")
		}
	})

	t.Run("Missing authorized client identity", func(t *testing.T) {
		def := PoADefinition{
			Parties: Parties{
				Principal:        Principal{Identity: "principal-1", Type: PrincipalTypeOrganization},
				AuthorizedClient: AuthorizedClient{Identity: "", TypeEnum: ClientTypeLLM},
			},
			Requirements: Requirements{
				ValidityPeriod: ValidityPeriod{
					StartTime: time.Now(),
					EndTime:   time.Now().Add(time.Hour),
				},
			},
		}

		err := ValidatePoADefinition(def)
		if err == nil {
			t.Error("ValidatePoADefinition() expected error for missing client identity, got nil")
		}
	})

	t.Run("Invalid validity period (end before start)", func(t *testing.T) {
		now := time.Now()
		def := PoADefinition{
			Parties: Parties{
				Principal:        Principal{Identity: "principal-1", Type: PrincipalTypeOrganization},
				AuthorizedClient: AuthorizedClient{Identity: "client-1", TypeEnum: ClientTypeLLM},
			},
			Requirements: Requirements{
				ValidityPeriod: ValidityPeriod{
					StartTime: now,
					EndTime:   now.Add(-time.Hour),
				},
			},
		}

		err := ValidatePoADefinition(def)
		if err == nil {
			t.Error("ValidatePoADefinition() expected error for invalid validity period, got nil")
		}
	})
}

// TestRepresentative_Validation tests representative validation
func TestRepresentative_Validation(t *testing.T) {
	t.Run("Valid representative with full information", func(t *testing.T) {
		rep := &Representative{
			Identity:          "AI-Operator-123",
			LegalRelationship: RelationshipOwner,
			RegistrationInfo: &RegistrationInfo{
				RegisteredName:        "AI Operator GmbH",
				RegistrationNumber:    "HRB12345",
				RegisteringAuthority:  "Amtsgericht Berlin",
				RegistrationDate:      "2023-06-01",
				Jurisdiction:          "DE",
				BusinessType:          "GmbH",
				CommercialRegister:    true,
				PowerOfAttorneyOnFile: true,
			},
			AuthorizationChain: []AuthorizationLink{
				{
					FromParty:     "Principal Corp",
					ToParty:       "AI Operator GmbH",
					GrantedDate:   "2024-01-01",
					Scope:         "AI operations",
					Revocable:     true,
					SubDelegation: false,
				},
			},
			ContactInformation: &ContactInformation{
				PrimaryContact: "Jane Doe",
				Email:          "contact@ai-operator.example.com",
				Phone:          "+49-30-12345678",
			},
		}

		err := rep.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("Missing identity", func(t *testing.T) {
		rep := &Representative{
			Identity:          "",
			LegalRelationship: RelationshipOwner,
		}

		err := rep.Validate()
		if err == nil {
			t.Error("Validate() expected error for missing identity, got nil")
		}
	})

	t.Run("Invalid legal relationship", func(t *testing.T) {
		rep := &Representative{
			Identity:          "AI-Operator-123",
			LegalRelationship: "InvalidRelationship",
		}

		err := rep.Validate()
		if err == nil {
			t.Error("Validate() expected error for invalid legal relationship, got nil")
		}
	})

	t.Run("Missing registration info fields", func(t *testing.T) {
		rep := &Representative{
			Identity:          "AI-Operator-123",
			LegalRelationship: RelationshipOwner,
			RegistrationInfo: &RegistrationInfo{
				RegisteredName:     "AI Operator GmbH",
				RegistrationNumber: "", // Missing
				Jurisdiction:       "DE",
			},
		}

		err := rep.Validate()
		if err == nil {
			t.Error("Validate() expected error for missing registration number, got nil")
		}
	})

	t.Run("Invalid authorization chain link", func(t *testing.T) {
		rep := &Representative{
			Identity:          "AI-Operator-123",
			LegalRelationship: RelationshipOwner,
			AuthorizationChain: []AuthorizationLink{
				{
					FromParty:   "Principal",
					ToParty:     "", // Missing
					GrantedDate: "2024-01-01",
					Scope:       "operations",
				},
			},
		}

		err := rep.Validate()
		if err == nil {
			t.Error("Validate() expected error for invalid authorization chain, got nil")
		}
	})

	t.Run("Missing contact information fields", func(t *testing.T) {
		rep := &Representative{
			Identity:          "AI-Operator-123",
			LegalRelationship: RelationshipOwner,
			ContactInformation: &ContactInformation{
				PrimaryContact: "", // Missing
				Email:          "test@example.com",
			},
		}

		err := rep.Validate()
		if err == nil {
			t.Error("Validate() expected error for missing primary contact, got nil")
		}
	})
}

// TestAuthorizationChain_Validation tests authorization chain validation
func TestAuthorizationChain_Validation(t *testing.T) {
	t.Run("Valid continuous chain", func(t *testing.T) {
		chain := []AuthorizationLink{
			{
				FromParty:     "PartyA",
				ToParty:       "PartyB",
				GrantedDate:   "2024-01-01",
				Scope:         "operations",
				SubDelegation: true,
			},
			{
				FromParty:     "PartyB",
				ToParty:       "PartyC",
				GrantedDate:   "2024-01-15",
				Scope:         "AI management",
				SubDelegation: false,
			},
		}

		err := ValidateAuthorizationChain(chain)
		if err != nil {
			t.Errorf("ValidateAuthorizationChain() error = %v, want nil", err)
		}
	})

	t.Run("Empty chain is valid", func(t *testing.T) {
		chain := []AuthorizationLink{}

		err := ValidateAuthorizationChain(chain)
		if err != nil {
			t.Errorf("ValidateAuthorizationChain() error = %v, want nil for empty chain", err)
		}
	})

	t.Run("Broken chain continuity", func(t *testing.T) {
		chain := []AuthorizationLink{
			{
				FromParty:     "PartyA",
				ToParty:       "PartyB",
				GrantedDate:   "2024-01-01",
				Scope:         "operations",
				SubDelegation: true,
			},
			{
				FromParty:     "PartyC", // Doesn't match previous ToParty
				ToParty:       "PartyD",
				GrantedDate:   "2024-01-15",
				Scope:         "management",
				SubDelegation: false,
			},
		}

		err := ValidateAuthorizationChain(chain)
		if err == nil {
			t.Error("ValidateAuthorizationChain() expected error for broken chain, got nil")
		}
	})

	t.Run("Unauthorized sub-delegation", func(t *testing.T) {
		chain := []AuthorizationLink{
			{
				FromParty:     "PartyA",
				ToParty:       "PartyB",
				GrantedDate:   "2024-01-01",
				Scope:         "operations",
				SubDelegation: false, // Sub-delegation not allowed
			},
			{
				FromParty:     "PartyB",
				ToParty:       "PartyC", // But PartyB tries to delegate
				GrantedDate:   "2024-01-15",
				Scope:         "management",
				SubDelegation: false,
			},
		}

		err := ValidateAuthorizationChain(chain)
		if err == nil {
			t.Error("ValidateAuthorizationChain() expected error for unauthorized sub-delegation, got nil")
		}
	})
}

// TestGeographicScope_Validation tests geographic scope validation
func TestGeographicScope_Validation(t *testing.T) {
	t.Run("Valid global scope", func(t *testing.T) {
		scope := &GeographicScope{
			Type: GeoTypeGlobal,
			Name: "Global",
		}

		err := scope.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("Valid national scope (Germany)", func(t *testing.T) {
		scope := &GeographicScope{
			Type:       GeoTypeNational,
			Identifier: "DE",
			Name:       "Germany",
		}

		err := scope.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("Valid subnational scope (Bavaria)", func(t *testing.T) {
		scope := &GeographicScope{
			Type:       GeoTypeSubnational,
			Identifier: "DE-BY",
			Name:       "Bavaria",
		}

		err := scope.Validate()
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("Invalid geographic type", func(t *testing.T) {
		scope := &GeographicScope{
			Type:       "InvalidType",
			Identifier: "DE",
		}

		err := scope.Validate()
		if err == nil {
			t.Error("Validate() expected error for invalid geographic type, got nil")
		}
	})

	t.Run("Missing identifier for national scope", func(t *testing.T) {
		scope := &GeographicScope{
			Type:       GeoTypeNational,
			Identifier: "",
		}

		err := scope.Validate()
		if err == nil {
			t.Error("Validate() expected error for missing identifier, got nil")
		}
	})

	t.Run("Invalid ISO 3166-1 format", func(t *testing.T) {
		scope := &GeographicScope{
			Type:       GeoTypeNational,
			Identifier: "DEU", // Should be 2 chars
		}

		err := scope.Validate()
		if err == nil {
			t.Error("Validate() expected error for invalid ISO 3166-1 format, got nil")
		}
	})

	t.Run("Lowercase country code", func(t *testing.T) {
		scope := &GeographicScope{
			Type:       GeoTypeNational,
			Identifier: "de", // Should be uppercase
		}

		err := scope.Validate()
		if err == nil {
			t.Error("Validate() expected error for lowercase country code, got nil")
		}
	})

	t.Run("Invalid ISO 3166-2 format", func(t *testing.T) {
		scope := &GeographicScope{
			Type:       GeoTypeSubnational,
			Identifier: "DEBY", // Missing hyphen
		}

		err := scope.Validate()
		if err == nil {
			t.Error("Validate() expected error for invalid ISO 3166-2 format, got nil")
		}
	})
}

// TestGeographicScope_IsAuthorizedInRegion tests regional authorization
func TestGeographicScope_IsAuthorizedInRegion(t *testing.T) {
	t.Run("Global scope authorizes all regions", func(t *testing.T) {
		scopes := []GeographicScope{
			{Type: GeoTypeGlobal, Name: "Global"},
		}

		if !IsAuthorizedInRegion(scopes, "DE") {
			t.Error("Expected global scope to authorize DE")
		}
		if !IsAuthorizedInRegion(scopes, "US") {
			t.Error("Expected global scope to authorize US")
		}
		if !IsAuthorizedInRegion(scopes, "JP") {
			t.Error("Expected global scope to authorize JP")
		}
	})

	t.Run("National scope exact match", func(t *testing.T) {
		scopes := []GeographicScope{
			{Type: GeoTypeNational, Identifier: "DE", Name: "Germany"},
			{Type: GeoTypeNational, Identifier: "FR", Name: "France"},
		}

		if !IsAuthorizedInRegion(scopes, "DE") {
			t.Error("Expected DE to be authorized")
		}
		if !IsAuthorizedInRegion(scopes, "FR") {
			t.Error("Expected FR to be authorized")
		}
		if IsAuthorizedInRegion(scopes, "GB") {
			t.Error("Expected GB to NOT be authorized")
		}
	})

	t.Run("Subnational scope with subdivisions", func(t *testing.T) {
		scopes := []GeographicScope{
			{
				Type:                GeoTypeNational,
				Identifier:          "DE",
				IncludeSubdivisions: true,
			},
		}

		if !IsAuthorizedInRegion(scopes, "DE") {
			t.Error("Expected DE to be authorized")
		}
		if !IsAuthorizedInRegion(scopes, "DE-BY") {
			t.Error("Expected DE-BY to be authorized (subdivision)")
		}
		if !IsAuthorizedInRegion(scopes, "DE-NW") {
			t.Error("Expected DE-NW to be authorized (subdivision)")
		}
	})

	t.Run("No matching scope", func(t *testing.T) {
		scopes := []GeographicScope{
			{Type: GeoTypeNational, Identifier: "FR", Name: "France"},
		}

		if IsAuthorizedInRegion(scopes, "DE") {
			t.Error("Expected DE to NOT be authorized")
		}
	})
}

// TestAuthorizedClient_Validations tests authorized client helper methods
func TestAuthorizedClient_Validations(t *testing.T) {
	t.Run("CanOperate with active status", func(t *testing.T) {
		client := &AuthorizedClient{
			StatusEnum: OperationalStatusActive,
		}

		if !client.CanOperate() {
			t.Error("Expected active client to be able to operate")
		}
	})

	t.Run("CanOperate with testing status", func(t *testing.T) {
		client := &AuthorizedClient{
			StatusEnum: OperationalStatusTesting,
		}

		if !client.CanOperate() {
			t.Error("Expected testing client to be able to operate")
		}
	})

	t.Run("Cannot operate when revoked", func(t *testing.T) {
		client := &AuthorizedClient{
			StatusEnum: OperationalStatusRevoked,
		}

		if client.CanOperate() {
			t.Error("Expected revoked client to NOT be able to operate")
		}
	})

	t.Run("Cannot operate when suspended", func(t *testing.T) {
		client := &AuthorizedClient{
			StatusEnum: OperationalStatusSuspended,
		}

		if client.CanOperate() {
			t.Error("Expected suspended client to NOT be able to operate")
		}
	})

	t.Run("IsPhysicalSystem for humanoid robot", func(t *testing.T) {
		client := &AuthorizedClient{
			TypeEnum: ClientTypeHumanoidRobot,
		}

		if !client.IsPhysicalSystem() {
			t.Error("Expected humanoid robot to be physical system")
		}
	})

	t.Run("IsPhysicalSystem for robotic system", func(t *testing.T) {
		client := &AuthorizedClient{
			TypeEnum: ClientTypeRoboticSystem,
		}

		if !client.IsPhysicalSystem() {
			t.Error("Expected robotic system to be physical system")
		}
	})

	t.Run("IsDigitalSystem for LLM", func(t *testing.T) {
		client := &AuthorizedClient{
			TypeEnum: ClientTypeLLM,
		}

		if !client.IsDigitalSystem() {
			t.Error("Expected LLM to be digital system")
		}
		if client.IsPhysicalSystem() {
			t.Error("Expected LLM to NOT be physical system")
		}
	})

	t.Run("RequiresTeamCoordination for agentic AI", func(t *testing.T) {
		client := &AuthorizedClient{
			TypeEnum: ClientTypeAgenticAI,
		}

		if !client.RequiresTeamCoordination() {
			t.Error("Expected agentic AI to require team coordination")
		}
	})

	t.Run("GetRiskLevel for high automation physical system", func(t *testing.T) {
		client := &AuthorizedClient{
			TypeEnum:        ClientTypeHumanoidRobot,
			CapabilityLevel: CapabilityL5,
		}

		risk := client.GetRiskLevel()
		if risk != "high" {
			t.Errorf("Expected risk level 'high', got '%s'", risk)
		}
	})
}

// TestClientType_Validation tests client type validation
func TestClientType_Validation(t *testing.T) {
	validTypes := []ClientType{
		ClientTypeLLM,
		ClientTypeDigitalAgent,
		ClientTypeAgenticAI,
		ClientTypeHumanoidRobot,
		ClientTypeRoboticSystem,
		ClientTypeOther,
	}

	for _, ct := range validTypes {
		t.Run(string(ct), func(t *testing.T) {
			err := ValidateClientType(ct)
			if err != nil {
				t.Errorf("ValidateClientType(%s) error = %v, want nil", ct, err)
			}
		})
	}

	t.Run("Invalid client type", func(t *testing.T) {
		err := ValidateClientType("InvalidType")
		if err == nil {
			t.Error("ValidateClientType() expected error for invalid type, got nil")
		}
	})
}

// TestOperationalStatus_Validation tests operational status validation
func TestOperationalStatus_Validation(t *testing.T) {
	validStatuses := []OperationalStatus{
		OperationalStatusActive,
		OperationalStatusSuspended,
		OperationalStatusRevoked,
		OperationalStatusMaintenance,
		OperationalStatusTesting,
		OperationalStatusDecommissioned,
	}

	for _, status := range validStatuses {
		t.Run(string(status), func(t *testing.T) {
			err := ValidateOperationalStatus(status)
			if err != nil {
				t.Errorf("ValidateOperationalStatus(%s) error = %v, want nil", status, err)
			}
		})
	}

	t.Run("Invalid operational status", func(t *testing.T) {
		err := ValidateOperationalStatus("InvalidStatus")
		if err == nil {
			t.Error("ValidateOperationalStatus() expected error for invalid status, got nil")
		}
	})
}

// TestCapabilityLevel_Validation tests capability level validation
func TestCapabilityLevel_Validation(t *testing.T) {
	validLevels := []CapabilityLevel{
		CapabilityL0,
		CapabilityL1,
		CapabilityL2,
		CapabilityL3,
		CapabilityL4,
		CapabilityL5,
	}

	for _, level := range validLevels {
		t.Run(string(level), func(t *testing.T) {
			err := ValidateCapabilityLevel(level)
			if err != nil {
				t.Errorf("ValidateCapabilityLevel(%s) error = %v, want nil", level, err)
			}
		})
	}

	t.Run("Empty capability level is valid (optional)", func(t *testing.T) {
		err := ValidateCapabilityLevel("")
		if err != nil {
			t.Error("ValidateCapabilityLevel(\"\") expected nil for empty level, got error")
		}
	})

	t.Run("Invalid capability level", func(t *testing.T) {
		err := ValidateCapabilityLevel("InvalidLevel")
		if err == nil {
			t.Error("ValidateCapabilityLevel() expected error for invalid level, got nil")
		}
	})
}

// TestLegalRelationship_Validation tests legal relationship validation
func TestLegalRelationship_Validation(t *testing.T) {
	validRelationships := []LegalRelationship{
		RelationshipOwner,
		RelationshipOperator,
		RelationshipLicensee,
		RelationshipContractor,
		RelationshipServiceProvider,
		RelationshipManufacturer,
		RelationshipDistributor,
		RelationshipAgent,
		RelationshipOther,
	}

	for _, lr := range validRelationships {
		t.Run(string(lr), func(t *testing.T) {
			err := ValidateLegalRelationship(lr)
			if err != nil {
				t.Errorf("ValidateLegalRelationship(%s) error = %v, want nil", lr, err)
			}
		})
	}

	t.Run("Invalid legal relationship", func(t *testing.T) {
		err := ValidateLegalRelationship("InvalidRelationship")
		if err == nil {
			t.Error("ValidateLegalRelationship() expected error for invalid relationship, got nil")
		}
	})
}

// TestValidityPeriod_TemporalConstraints tests temporal constraint validation
func TestValidityPeriod_TemporalConstraints(t *testing.T) {
	now := time.Now()

	t.Run("Valid future validity period", func(t *testing.T) {
		vp := ValidityPeriod{
			StartTime: now,
			EndTime:   now.Add(365 * 24 * time.Hour),
		}

		// Create a minimal PoA definition to test validity period
		def := PoADefinition{
			Parties: Parties{
				Principal:        Principal{Identity: "test-principal", Type: PrincipalTypeOrganization},
				AuthorizedClient: AuthorizedClient{Identity: "test-client", TypeEnum: ClientTypeLLM},
			},
			Requirements: Requirements{
				ValidityPeriod: vp,
			},
		}

		err := ValidatePoADefinition(def)
		if err != nil {
			t.Errorf("Expected valid future period, got error: %v", err)
		}
	})

	t.Run("Valid past start with future end", func(t *testing.T) {
		vp := ValidityPeriod{
			StartTime: now.Add(-30 * 24 * time.Hour),
			EndTime:   now.Add(30 * 24 * time.Hour),
		}

		def := PoADefinition{
			Parties: Parties{
				Principal:        Principal{Identity: "test-principal", Type: PrincipalTypeOrganization},
				AuthorizedClient: AuthorizedClient{Identity: "test-client", TypeEnum: ClientTypeLLM},
			},
			Requirements: Requirements{
				ValidityPeriod: vp,
			},
		}

		err := ValidatePoADefinition(def)
		if err != nil {
			t.Errorf("Expected valid period with past start, got error: %v", err)
		}
	})

	t.Run("Invalid: end before start", func(t *testing.T) {
		vp := ValidityPeriod{
			StartTime: now,
			EndTime:   now.Add(-24 * time.Hour),
		}

		def := PoADefinition{
			Parties: Parties{
				Principal:        Principal{Identity: "test-principal", Type: PrincipalTypeOrganization},
				AuthorizedClient: AuthorizedClient{Identity: "test-client", TypeEnum: ClientTypeLLM},
			},
			Requirements: Requirements{
				ValidityPeriod: vp,
			},
		}

		err := ValidatePoADefinition(def)
		if err == nil {
			t.Error("Expected error for end before start, got nil")
		}
	})
}

// BenchmarkValidatePoADefinition benchmarks PoA validation
func BenchmarkValidatePoADefinition(b *testing.B) {
	def := PoADefinition{
		Parties: Parties{
			Principal: Principal{
				Type:     PrincipalTypeOrganization,
				Identity: "DE:HRB:12345",
			},
			AuthorizedClient: AuthorizedClient{
				TypeEnum: ClientTypeLLM,
				Identity: "client-001",
			},
		},
		Requirements: Requirements{
			ValidityPeriod: ValidityPeriod{
				StartTime: time.Now(),
				EndTime:   time.Now().Add(365 * 24 * time.Hour),
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePoADefinition(def)
	}
}

// BenchmarkRepresentativeValidate benchmarks representative validation
func BenchmarkRepresentativeValidate(b *testing.B) {
	rep := &Representative{
		Identity:          "AI-Operator-123",
		LegalRelationship: RelationshipOwner,
		RegistrationInfo: &RegistrationInfo{
			RegisteredName:     "AI Operator GmbH",
			RegistrationNumber: "HRB12345",
			Jurisdiction:       "DE",
		},
		ContactInformation: &ContactInformation{
			PrimaryContact: "Contact Name",
			Email:          "contact@example.com",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rep.Validate()
	}
}

// BenchmarkValidateAuthorizationChain benchmarks authorization chain validation
func BenchmarkValidateAuthorizationChain(b *testing.B) {
	chain := []AuthorizationLink{
		{
			FromParty:     "PartyA",
			ToParty:       "PartyB",
			GrantedDate:   "2024-01-01",
			Scope:         "operations",
			SubDelegation: true,
		},
		{
			FromParty:     "PartyB",
			ToParty:       "PartyC",
			GrantedDate:   "2024-01-15",
			Scope:         "management",
			SubDelegation: false,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateAuthorizationChain(chain)
	}
}

// BenchmarkGeographicScopeValidate benchmarks geographic scope validation
func BenchmarkGeographicScopeValidate(b *testing.B) {
	scope := &GeographicScope{
		Type:       GeoTypeNational,
		Identifier: "DE",
		Name:       "Germany",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = scope.Validate()
	}
}
