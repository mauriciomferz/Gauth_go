package gauth

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/poa"
)

// TestExtendedToken_Validate tests extended token validation per RFC-0111 §3
func TestExtendedToken_Validate(t *testing.T) {
	now := time.Now()
	futureTime := now.Add(365 * 24 * time.Hour)

	t.Run("Valid complete token", func(t *testing.T) {
		token := &ExtendedToken{
			AccessToken: "test_token_123",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			Scope:       []string{"read", "write"},
			IssuedAt:    now,
			PowerOfAttorney: &poa.PoADefinition{
				Parties: poa.Parties{
					Principal: poa.Principal{
						Type:     "Organization",
						Identity: "DE-HRB-12345",
					},
					AuthorizedClient: poa.AuthorizedClient{
						TypeEnum: poa.ClientTypeLLM,
						Identity: "gpt-4",
						Version:  "2024.1",
					},
				},
			},
			AuthorizationChain: &AuthorizationChain{
				OwnersAuthorizer: &AuthorizationLink{
					EntityID:           "auth_001",
					EntityName:         "John Smith",
					EntityType:         "natural_person",
					Role:               "authorizer",
					AuthorizedBy:       "DE-HRB-12345",
					AuthorizationDate:  now,
					AuthorizationType:  "statutory",
					LegalBasis:         &LegalBasis{BasisType: "company_law", Jurisdiction: "DE"},
					StatutoryAuthority: "§35 GmbHG",
					IdentityVerified:   true,
					VerificationMethod: "eIDAS",
					VerificationProof:  "cert_123",
					ScopeOfAuthority:   []string{"Full authorization"},
					ValidFrom:          now,
					ValidUntil:         futureTime,
					Revocable:          true,
					Status:             "active",
				},
				ClientOwner: &AuthorizationLink{
					EntityID:              "owner_001",
					EntityName:            "TechCorp GmbH",
					EntityType:            "organization",
					Role:                  "owner",
					AuthorizedBy:          "auth_001",
					AuthorizationDate:     now,
					AuthorizationType:     "contractual",
					LegalBasis:            &LegalBasis{BasisType: "power_of_attorney", Jurisdiction: "DE"},
					CommercialRegisterRef: "DE-HRB-12345",
					IdentityVerified:      true,
					VerificationMethod:    "CommercialRegister",
					VerificationProof:     "DE-HRB-12345",
					ScopeOfAuthority:      []string{"AI system ownership"},
					ValidFrom:             now,
					ValidUntil:            futureTime,
					Revocable:             false,
					Status:                "active",
				},
				Client: &AuthorizationLink{
					EntityID:           "client_123",
					EntityName:         "GPT-4 Assistant",
					EntityType:         "ai_system",
					Role:               "client",
					AuthorizedBy:       "owner_001",
					AuthorizationDate:  now,
					AuthorizationType:  "delegated",
					LegalBasis:         &LegalBasis{BasisType: "contractual", Jurisdiction: "DE"},
					IdentityVerified:   true,
					VerificationMethod: "Certificate",
					VerificationProof:  "cert_client_123",
					ScopeOfAuthority:   []string{"AI operations"},
					ValidFrom:          now,
					ValidUntil:         futureTime,
					Revocable:          true,
					Status:             "active",
				},
			},
			ClientOwner: &ClientOwnerInfo{
				OwnerID:                   "owner_001",
				OwnerName:                 "TechCorp GmbH",
				OwnerType:                 "organization",
				OrganizationLegalName:     "TechCorp GmbH",
				OrganizationRegistration:  "HRB 12345",
				JurisdictionOfIncorp:      "DE",
				RegisteredPowerOfAttorney: true,
				CommercialRegisterEntry:   true,
				CommercialRegisterID:      "DE-HRB-12345",
				IdentityVerified:          true,
				VerificationDate:          now,
				VerificationMethod:        "eIDAS",
			},
			OwnersAuthorizer: &OwnersAuthorizerInfo{
				AuthorizerID:            "auth_001",
				AuthorizerName:          "John Smith",
				AuthorizerType:          "managing_director",
				StatutoryAuthority:      "§35 GmbHG",
				AuthorityType:           "individual",
				AuthorityScope:          []string{"Full"},
				CommercialRegisterEntry: true,
				CommercialRegisterID:    "DE-HRB-12345",
				RegisterJurisdiction:    "DE",
				IdentityVerified:        true,
				VerificationMethod:      "eIDAS",
				VerificationDate:        now,
				VerificationProof:       "cert_auth_001",
				RelationshipToOwner:     "Managing Director",
				AuthorizationBasis:      "Commercial Law",
			},
			ResourceOwner: &ResourceOwnerInfo{
				OwnerID:          "user_123",
				OwnerName:        "End User",
				OwnerType:        "individual",
				Jurisdiction:     "DE",
				IdentityVerified: true,
				VerificationDate: now,
			},
			LegalFramework: &LegalFrameworkInfo{
				Jurisdiction:        "DE",
				ApplicableLaws:      []string{"GDPR", "GmbHG"},
				ComplianceFramework: "ISO 27001",
				FiduciaryDuties: []FiduciaryDuty{
					{DutyType: "care", Description: "Duty of care", Scope: []string{"All operations"}},
				},
			},
			VerificationProof: &IdentityVerificationChain{
				ChainID:             "chain_001",
				OverallVerification: "verified",
				VerificationTime:    now,
				VerifierEntity:      "PVP-001",
				VerificationLevels: []VerificationLevel{
					{Level: 1, EntityID: "auth_001", EntityRole: "authorizer", VerificationMethod: "eIDAS", VerificationStatus: "verified", VerificationDate: now, AssuranceLevel: "high"},
					{Level: 2, EntityID: "owner_001", EntityRole: "owner", VerificationMethod: "CommercialRegister", VerificationStatus: "verified", VerificationDate: now, AssuranceLevel: "high"},
					{Level: 3, EntityID: "client_123", EntityRole: "client", VerificationMethod: "Certificate", VerificationStatus: "verified", VerificationDate: now, AssuranceLevel: "substantial"},
				},
			},
			IssuedBy: &AuthorizationServerInfo{
				ServerID:   "auth-server-001",
				ServerURL:  "https://auth.example.com",
				ServerName: "AgentAuth Authorization Server",
				Issuer:     "https://auth.example.com",
				IssueTime:  now,
			},
		}

		err := token.Validate()
		if err != nil {
			t.Errorf("Validate() unexpected error: %v", err)
		}
	})

	t.Run("Missing authorization chain", func(t *testing.T) {
		token := &ExtendedToken{
			AccessToken: "test_token_123",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
		}

		err := token.Validate()
		if err == nil {
			t.Error("Validate() expected error for missing authorization chain, got nil")
		}
	})

	t.Run("Missing client owner", func(t *testing.T) {
		token := &ExtendedToken{
			AccessToken: "test_token_123",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			AuthorizationChain: &AuthorizationChain{
				OwnersAuthorizer: &AuthorizationLink{
					EntityID:   "auth_001",
					EntityName: "John Smith",
				},
			},
		}

		err := token.Validate()
		if err == nil {
			t.Error("Validate() expected error for missing client owner, got nil")
		}
	})

	t.Run("Missing owner's authorizer", func(t *testing.T) {
		token := &ExtendedToken{
			AccessToken: "test_token_123",
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			AuthorizationChain: &AuthorizationChain{
				ClientOwner: &AuthorizationLink{
					EntityID:   "owner_001",
					EntityName: "TechCorp",
				},
			},
		}

		err := token.Validate()
		if err == nil {
			t.Error("Validate() expected error for missing owner's authorizer, got nil")
		}
	})
}

// TestAuthorizationChain_Validate tests authorization chain validation
func TestAuthorizationChain_Validate(t *testing.T) {
	now := time.Now()

	t.Run("Valid authorization chain", func(t *testing.T) {
		chain := &AuthorizationChain{
			OwnersAuthorizer: &AuthorizationLink{
				EntityID:          "auth_001",
				EntityName:        "John Smith",
				EntityType:        "natural_person",
				Role:              "authorizer",
				AuthorizedBy:      "root",
				AuthorizationDate: now,
				LegalBasis:        &LegalBasis{BasisType: "company_law", Jurisdiction: "DE"},
				IdentityVerified:  true,
				ScopeOfAuthority:  []string{"Full"},
				ValidFrom:         now,
				ValidUntil:        now.Add(365 * 24 * time.Hour),
				Status:            "active",
			},
			ClientOwner: &AuthorizationLink{
				EntityID:          "owner_001",
				EntityName:        "TechCorp GmbH",
				EntityType:        "organization",
				Role:              "owner",
				AuthorizedBy:      "auth_001",
				AuthorizationDate: now,
				LegalBasis:        &LegalBasis{BasisType: "power_of_attorney", Jurisdiction: "DE"},
				IdentityVerified:  true,
				ScopeOfAuthority:  []string{"AI operations"},
				ValidFrom:         now,
				ValidUntil:        now.Add(365 * 24 * time.Hour),
				Status:            "active",
			},
			Client: &AuthorizationLink{
				EntityID:          "client_123",
				EntityName:        "GPT-4",
				EntityType:        "ai_system",
				Role:              "client",
				AuthorizedBy:      "owner_001",
				AuthorizationDate: now,
				LegalBasis:        &LegalBasis{BasisType: "contractual", Jurisdiction: "DE"},
				IdentityVerified:  true,
				ScopeOfAuthority:  []string{"Operations"},
				ValidFrom:         now,
				ValidUntil:        now.Add(365 * 24 * time.Hour),
				Status:            "active",
			},
		}

		err := chain.Validate()
		if err != nil {
			t.Errorf("Validate() unexpected error: %v", err)
		}
	})

	t.Run("Broken chain - client not authorized by owner", func(t *testing.T) {
		chain := &AuthorizationChain{
			OwnersAuthorizer: &AuthorizationLink{
				EntityID:     "auth_001",
				AuthorizedBy: "root",
				Status:       "active",
			},
			ClientOwner: &AuthorizationLink{
				EntityID:     "owner_001",
				AuthorizedBy: "auth_001",
				Status:       "active",
			},
			Client: &AuthorizationLink{
				EntityID:     "client_123",
				AuthorizedBy: "someone_else", // Should be owner_001
				Status:       "active",
			},
		}

		err := chain.Validate()
		if err == nil {
			t.Error("Validate() expected error for broken chain, got nil")
		}
	})

	t.Run("Broken chain - owner not authorized by authorizer", func(t *testing.T) {
		chain := &AuthorizationChain{
			OwnersAuthorizer: &AuthorizationLink{
				EntityID:     "auth_001",
				AuthorizedBy: "root",
				Status:       "active",
			},
			ClientOwner: &AuthorizationLink{
				EntityID:     "owner_001",
				AuthorizedBy: "wrong_auth", // Should be auth_001
				Status:       "active",
			},
			Client: &AuthorizationLink{
				EntityID:     "client_123",
				AuthorizedBy: "owner_001",
				Status:       "active",
			},
		}

		err := chain.Validate()
		if err == nil {
			t.Error("Validate() expected error for broken chain, got nil")
		}
	})
}

// TestExtendedToken_HasCommercialRegisterProof tests commercial register proof verification
func TestExtendedToken_HasCommercialRegisterProof(t *testing.T) {
	t.Run("Has commercial register proof", func(t *testing.T) {
		token := &ExtendedToken{
			OwnersAuthorizer: &OwnersAuthorizerInfo{
				CommercialRegisterEntry: true,
				CommercialRegisterID:    "HRB 12345",
			},
		}

		if !token.HasCommercialRegisterProof() {
			t.Error("HasCommercialRegisterProof() = false, want true")
		}
	})

	t.Run("No commercial register proof", func(t *testing.T) {
		token := &ExtendedToken{
			OwnersAuthorizer: &OwnersAuthorizerInfo{
				AuthorizerName: "John Smith",
			},
		}

		if token.HasCommercialRegisterProof() {
			t.Error("HasCommercialRegisterProof() = true, want false")
		}
	})

	t.Run("No owner's authorizer", func(t *testing.T) {
		token := &ExtendedToken{}

		if token.HasCommercialRegisterProof() {
			t.Error("HasCommercialRegisterProof() = true, want false")
		}
	})
}

// TestExtendedToken_Serialization tests token field access
func TestExtendedToken_Serialization(t *testing.T) {
	now := time.Now()
	original := &ExtendedToken{
		AccessToken: "test_token_123",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Scope:       []string{"read", "write"},
		IssuedAt:    now,
		ClientOwner: &ClientOwnerInfo{
			OwnerID:                  "owner_001",
			OwnerName:                "TechCorp GmbH",
			OrganizationRegistration: "HRB 12345",
		},
		AuthorizationChain: &AuthorizationChain{
			OwnersAuthorizer: &AuthorizationLink{
				EntityID:   "auth_001",
				EntityName: "John Smith",
			},
			ClientOwner: &AuthorizationLink{
				EntityID:   "owner_001",
				EntityName: "TechCorp GmbH",
			},
			Client: &AuthorizationLink{
				EntityID:   "client_123",
				EntityName: "GPT-4",
			},
		},
	}

	// Test that all fields are accessible
	if original.AccessToken != "test_token_123" {
		t.Errorf("AccessToken = %v, want test_token_123", original.AccessToken)
	}
	if original.ClientOwner.OwnerName != "TechCorp GmbH" {
		t.Errorf("ClientOwner.OwnerName = %v, want TechCorp GmbH", original.ClientOwner.OwnerName)
	}
	if original.AuthorizationChain.OwnersAuthorizer.EntityName != "John Smith" {
		t.Errorf("AuthorizationChain.OwnersAuthorizer.EntityName = %v, want John Smith", original.AuthorizationChain.OwnersAuthorizer.EntityName)
	}
	if len(original.Scope) != 2 {
		t.Errorf("Scope length = %d, want 2", len(original.Scope))
	}
}

// TestExtendedToken_LegalFrameworkValidation tests legal framework validation
func TestExtendedToken_LegalFrameworkValidation(t *testing.T) {
	token := &ExtendedToken{
		AccessToken: "test_token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		LegalFramework: &LegalFrameworkInfo{
			Jurisdiction:        "DE",
			ApplicableLaws:      []string{"GDPR", "GmbHG"},
			ComplianceFramework: "ISO 27001",
		},
		AuthorizationChain: &AuthorizationChain{
			OwnersAuthorizer: &AuthorizationLink{EntityID: "auth_001", AuthorizedBy: "root", Status: "active"},
			ClientOwner:      &AuthorizationLink{EntityID: "owner_001", AuthorizedBy: "auth_001", Status: "active"},
			Client:           &AuthorizationLink{EntityID: "client_001", AuthorizedBy: "owner_001", Status: "active"},
		},
		ClientOwner:      &ClientOwnerInfo{OwnerID: "owner_001"},
		OwnersAuthorizer: &OwnersAuthorizerInfo{AuthorizerID: "auth_001"},
	}

	if token.LegalFramework == nil {
		t.Error("LegalFramework should not be nil")
	}

	if token.LegalFramework.Jurisdiction != "DE" {
		t.Errorf("Jurisdiction = %v, want DE", token.LegalFramework.Jurisdiction)
	}

	if len(token.LegalFramework.ApplicableLaws) != 2 {
		t.Errorf("ApplicableLaws count = %v, want 2", len(token.LegalFramework.ApplicableLaws))
	}
}

// TestExtendedToken_RestrictionsValidation tests power restrictions validation
func TestExtendedToken_RestrictionsValidation(t *testing.T) {
	token := &ExtendedToken{
		AccessToken: "test_token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Restrictions: []PowerRestriction{
			{
				RestrictionType:  "geographic_limit",
				Description:      "Operations limited to EU",
				Value:            "EU",
				EnforcementLevel: "mandatory",
			},
			{
				RestrictionType:  "time_limit",
				Description:      "Business hours only",
				Value:            "09:00-17:00 CET",
				EnforcementLevel: "mandatory",
			},
		},
		AuthorizationChain: &AuthorizationChain{
			OwnersAuthorizer: &AuthorizationLink{EntityID: "auth_001", AuthorizedBy: "root", Status: "active"},
			ClientOwner:      &AuthorizationLink{EntityID: "owner_001", AuthorizedBy: "auth_001", Status: "active"},
			Client:           &AuthorizationLink{EntityID: "client_001", AuthorizedBy: "owner_001", Status: "active"},
		},
		ClientOwner:      &ClientOwnerInfo{OwnerID: "owner_001"},
		OwnersAuthorizer: &OwnersAuthorizerInfo{AuthorizerID: "auth_001"},
	}

	if len(token.Restrictions) != 2 {
		t.Errorf("Restrictions count = %v, want 2", len(token.Restrictions))
	}

	if token.Restrictions[0].RestrictionType != "geographic_limit" {
		t.Errorf("First restriction type = %v, want geographic_limit", token.Restrictions[0].RestrictionType)
	}
}

// Benchmark tests
func BenchmarkExtendedToken_Validate(b *testing.B) {
	now := time.Now()
	token := &ExtendedToken{
		AccessToken: "test_token_123",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		AuthorizationChain: &AuthorizationChain{
			OwnersAuthorizer: &AuthorizationLink{
				EntityID:     "auth_001",
				EntityName:   "John Smith",
				AuthorizedBy: "root",
				ValidFrom:    now,
				ValidUntil:   now.Add(365 * 24 * time.Hour),
				Status:       "active",
			},
			ClientOwner: &AuthorizationLink{
				EntityID:     "owner_001",
				EntityName:   "TechCorp",
				AuthorizedBy: "auth_001",
				ValidFrom:    now,
				ValidUntil:   now.Add(365 * 24 * time.Hour),
				Status:       "active",
			},
			Client: &AuthorizationLink{
				EntityID:     "client_123",
				EntityName:   "GPT-4",
				AuthorizedBy: "owner_001",
				ValidFrom:    now,
				ValidUntil:   now.Add(365 * 24 * time.Hour),
				Status:       "active",
			},
		},
		ClientOwner:      &ClientOwnerInfo{OwnerID: "owner_001"},
		OwnersAuthorizer: &OwnersAuthorizerInfo{AuthorizerID: "auth_001"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = token.Validate()
	}
}

func BenchmarkAuthorizationChain_Validate(b *testing.B) {
	now := time.Now()
	chain := &AuthorizationChain{
		OwnersAuthorizer: &AuthorizationLink{
			EntityID:     "auth_001",
			AuthorizedBy: "root",
			ValidFrom:    now,
			ValidUntil:   now.Add(365 * 24 * time.Hour),
			Status:       "active",
		},
		ClientOwner: &AuthorizationLink{
			EntityID:     "owner_001",
			AuthorizedBy: "auth_001",
			ValidFrom:    now,
			ValidUntil:   now.Add(365 * 24 * time.Hour),
			Status:       "active",
		},
		Client: &AuthorizationLink{
			EntityID:     "client_123",
			AuthorizedBy: "owner_001",
			ValidFrom:    now,
			ValidUntil:   now.Add(365 * 24 * time.Hour),
			Status:       "active",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chain.Validate()
	}
}
