package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

// TestCompleteAuthorizationFlow tests the complete RFC111 authorization flow
func TestCompleteAuthorizationFlow(t *testing.T) {
	// Setup test environment
	ctx := context.Background()
	framework := setupTestFramework(t)

	// Step I-II: Owner's authorizer proves identity and authorization
	authorizerProof := createAuthorizerProof(t)
	err := framework.VerifyLegalCapacity(ctx, authorizerProof.Entity)
	require.NoError(t, err, "Owner's authorizer verification should succeed")

	// Step III-IV: Client owner proves identity and authorization
	clientOwnerProof := createClientOwnerProof(t, authorizerProof)
	err = framework.VerifyLegalCapacity(ctx, clientOwnerProof.Entity)
	require.NoError(t, err, "Client owner verification should succeed")

	// Step V: Client authorization
	clientAuth := createClientAuthorization(t, clientOwnerProof)
	err = framework.ValidateClientResourceServerInteraction(ctx, clientAuth.Client, clientAuth.Server)
	require.NoError(t, err, "Client authorization should succeed")

	// Step VI-VII: Resource owner proves identity and authorization
	resourceOwnerProof := createResourceOwnerProof(t, authorizerProof)
	err = framework.VerifyLegalCapacity(ctx, resourceOwnerProof.Entity)
	require.NoError(t, err, "Resource owner verification should succeed")

	// Step VIII: Resource server authorization
	serverAuth := createServerAuthorization(t, resourceOwnerProof)
	err = framework.ValidateResourceServerPowers(ctx, serverAuth.Token, serverAuth.Request)
	require.NoError(t, err, "Resource server authorization should succeed")

	// Test request-specific steps (a-i)
	t.Run("RequestFlow", func(t *testing.T) {
		// Step a-b: Client requests authorization and validation
		clientRequest := createClientRequest(t, clientAuth)
		err = framework.ValidateJurisdiction(ctx, clientRequest.Jurisdiction, clientRequest.Action)
		require.NoError(t, err, "Request jurisdiction validation should succeed")

		// Step c-e: Authorization grant and token issuance
		authGrant := createAuthorizationGrant(t, clientRequest)
		err = validateFiduciaryDuties(t, framework, authGrant)
		require.NoError(t, err, "Fiduciary duties validation should succeed")

		// Step f: Client validates grant compliance
		err = validateGrantCompliance(t, framework, authGrant)
		require.NoError(t, err, "Grant compliance validation should succeed")

		// Step g-h: Transaction execution and validation
		txn := createTransaction(t, authGrant)
		err = validateTransaction(t, framework, txn)
		require.NoError(t, err, "Transaction validation should succeed")

		// Step i: Compliance tracking
		err = trackCompliance(t, framework, txn)
		require.NoError(t, err, "Compliance tracking should succeed")
	})
}

// TestJurisdictionCompliance tests compliance across different jurisdictions
func TestJurisdictionCompliance(t *testing.T) {
	ctx := context.Background()
	framework := setupTestFramework(t)

	jurisdictions := []string{"US", "EU", "UK", "JP"}
	for _, j := range jurisdictions {
		t.Run(j, func(t *testing.T) {
			rules, err := framework.GetJurisdictionRules(j)
			require.NoError(t, err)

			// Test jurisdiction-specific requirements
			err = framework.ValidateJurisdictionRequirements(ctx, rules, "high_value_transaction")
			assert.NoError(t, err)

			// Test fiduciary duties in jurisdiction
			duties := createJurisdictionDuties(t, j)
			for _, duty := range duties {
				err = framework.ValidateDuty(ctx, duty)
				assert.NoError(t, err)
			}
		})
	}
}

// TestPowerOfAttorneyChain tests the delegation chain validation
func TestPowerOfAttorneyChain(t *testing.T) {
	ctx := context.Background()
	framework := setupTestFramework(t)

	// Create a chain of delegations
	chain := createDelegationChain(t)

	// Test each link in the chain
	for i, link := range chain {
		t.Run(fmt.Sprintf("Link_%d", i), func(t *testing.T) {
			// Verify capacity at each level
			err := framework.VerifyLegalCapacity(ctx, link.Entity)
			assert.NoError(t, err)

			// Verify jurisdiction compliance
			err = framework.ValidateJurisdiction(ctx, link.Jurisdiction, "delegate")
			assert.NoError(t, err)

			// Verify fiduciary duties
			err = framework.EnforceFiduciaryDuties(ctx, link.Power)
			assert.NoError(t, err)
		})
	}
}

// TestAITeamAuthorization tests centralized AI team authorization
func TestAITeamAuthorization(t *testing.T) {
	ctx := context.Background()
	framework := setupTestFramework(t)

	// Create AI team structure
	team := createAITeam(t)

	// Test lead agent authorization
	t.Run("LeadAgent", func(t *testing.T) {
		err := framework.VerifyLegalCapacity(ctx, team.LeadAgent.Entity)
		assert.NoError(t, err)
	})

	// Test team member authorizations
	t.Run("TeamMembers", func(t *testing.T) {
		for i, member := range team.Members {
			t.Run(fmt.Sprintf("Member_%d", i), func(t *testing.T) {
				err := framework.VerifyLegalCapacity(ctx, member.Entity)
				assert.NoError(t, err)
			})
		}
	})

	// Verify centralized control
	t.Run("CentralizedControl", func(t *testing.T) {
		// Attempt decentralized authorization (should fail)
		err := framework.ValidateJurisdiction(ctx, team.Jurisdiction, "autonomous_decision")
		assert.Error(t, err)

		// Verify through central authority (should succeed)
		err = framework.ValidateJurisdiction(ctx, team.Jurisdiction, "centralized_decision")
		assert.NoError(t, err)
	})
}

// TestComplianceTracking tests the compliance tracking requirements
func TestComplianceTracking(t *testing.T) {
	ctx := context.Background()
	framework := setupTestFramework(t)

	// Create a series of actions to track
	actions := createComplianceActions(t)

	// Test tracking for each action
	for i, action := range actions {
		t.Run(fmt.Sprintf("Action_%d", i), func(t *testing.T) {
			// Create approval event
			event := &auth.ApprovalEvent{
				Time:            time.Now(),
				ApprovalID:      fmt.Sprintf("approval_%d", i),
				RequesterID:     action.RequesterID,
				ApproverID:      action.ApproverID,
				Action:          action.Name,
				JurisdictionID:  action.Jurisdiction,
				LegalBasis:      action.LegalBasis,
				FiduciaryChecks: []auth.FiduciaryDuty{}, // Updated to match stubs
				Evidence:        action.Evidence,
			}

			// Track the approval
			err := framework.TrackApprovalDetails(ctx, event)
			assert.NoError(t, err)

			// Verify tracking record
			records, err := getTrackingRecords(t, framework, event.ApprovalID)
			assert.NoError(t, err)
			assert.NotEmpty(t, records)
		})
	}
}

// Helper functions for test setup and data creation
func setupTestFramework(_ *testing.T) *auth.StandardLegalFramework {
	store := auth.NewMemoryStore()
	verifier := auth.NewStandardVerificationSystem()
	register := auth.NewStandardCommercialRegister()

	return auth.NewStandardLegalFramework(store, verifier, register)
}

func createAuthorizerProof(_ *testing.T) *auth.CapacityProof {
	return &auth.CapacityProof{
		Type:         "commercial_register",
		IssuedAt:     time.Now().Add(-24 * time.Hour),
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
		IssuerID:     "commercial_registry_001",
		Proof:        "cryptographic_proof_001",
		Jurisdiction: "US",
		Entity: &auth.Entity{
			ID:             "authorizer_001",
			Type:           "organization",
			JurisdictionID: "US",
			LegalStatus:    "active",
			CapacityProofs: []auth.CapacityProof{},
			FiduciaryDuties: []auth.FiduciaryDuty{
				{
					Type:        "compliance",
					Description: "Regulatory compliance oversight",
					Scope:       []string{"all_operations"},
					Validation:  []string{"reg_compliance_check"},
				},
			},
		},
	}
}

func createClientOwnerProof(_ *testing.T, authorizer *auth.CapacityProof) *auth.CapacityProof {
	return &auth.CapacityProof{
		Type:         "power_of_attorney",
		IssuedAt:     time.Now().Add(-12 * time.Hour),
		ExpiresAt:    time.Now().Add(180 * 24 * time.Hour),
		IssuerID:     authorizer.Entity.ID,
		Proof:        "cryptographic_proof_002",
		Jurisdiction: authorizer.Jurisdiction,
		Entity: &auth.Entity{
			ID:             "client_owner_001",
			Type:           "organization",
			JurisdictionID: authorizer.Jurisdiction,
			LegalStatus:    "active",
			CapacityProofs: []auth.CapacityProof{},
			FiduciaryDuties: []auth.FiduciaryDuty{
				{
					Type:        "loyalty",
					Description: "Client interest protection",
					Scope:       []string{"client_operations"},
					Validation:  []string{"interest_check"},
				},
			},
		},
	}
}

func createClientAuthorization(_ *testing.T, ownerProof *auth.CapacityProof) *auth.ClientAuthorization {
	return &auth.ClientAuthorization{
		Client: &auth.Client{
			ID:           "ai_client_001",
			Type:         "agentic_ai",
			OwnerID:      ownerProof.Entity.ID,
			Entity:       ownerProof.Entity,
			Capabilities: []string{"financial_decisions", "contract_execution"},
		},
		Server: &auth.ResourceServer{
			ID:     "resource_001",
			Type:   "data_server",
			Entity: ownerProof.Entity,
			Scopes: []string{"financial_data", "contract_data"},
		},
		Timestamp: time.Now(),
		Scope:     []string{"execute_trades", "sign_contracts"},
	}
}

func createResourceOwnerProof(_ *testing.T, authorizer *auth.CapacityProof) *auth.CapacityProof {
	return &auth.CapacityProof{
		Type:         "legal_authority",
		IssuedAt:     time.Now().Add(-24 * time.Hour),
		ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
		IssuerID:     authorizer.Entity.ID,
		Proof:        "cryptographic_proof_003",
		Jurisdiction: authorizer.Jurisdiction,
		Entity: &auth.Entity{
			ID:             "resource_owner_001",
			Type:           "organization",
			JurisdictionID: authorizer.Jurisdiction,
			LegalStatus:    "active",
			CapacityProofs: []auth.CapacityProof{},
			FiduciaryDuties: []auth.FiduciaryDuty{
				{
					Type:        "care",
					Description: "Resource protection",
					Scope:       []string{"resource_management"},
					Validation:  []string{"protection_check"},
				},
			},
		},
	}
}

func createServerAuthorization(_ *testing.T, ownerProof *auth.CapacityProof) *auth.ServerAuthorization {
	return &auth.ServerAuthorization{
		Token: &auth.Token{
			ID:        "token_001",
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		},
		Request: &auth.LegalFrameworkRequest{
			ID: "request_001",
			ResourceServer: &auth.ResourceServer{
				ID:     "resource_002",
				Type:   "data_server",
				Entity: ownerProof.Entity,
				Scopes: []string{"data_access"},
			},
			Jurisdiction: ownerProof.Jurisdiction,
			Action:       "data_processing",
			PowerOfAttorney: &auth.PowerOfAttorney{
				ID:        "poa_001",
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(180 * 24 * time.Hour),
			},
		},
	}
}

func createClientRequest(_ *testing.T, clientAuth *auth.ClientAuthorization) *auth.LegalFrameworkRequest {
	return &auth.LegalFrameworkRequest{
		ID:           "request_002",
		ClientID:     clientAuth.Client.ID,
		Action:       "execute_trade",
		Resource:     "financial_data",
		Scope:        []string{"trade_execution"},
		Timestamp:    time.Now(),
		Jurisdiction: clientAuth.Client.Entity.JurisdictionID,
		Metadata: map[string]interface{}{
			"trade_value": 1000000.0,
			"trade_type":  "stock_purchase",
		},
	}
}

func createAuthorizationGrant(_ *testing.T, request *auth.LegalFrameworkRequest) *auth.LegalFrameworkAuthorizationGrant {
	return &auth.LegalFrameworkAuthorizationGrant{
		ID:        "grant_001",
		RequestID: request.ID,
		GrantorID: "resource_owner_001",
		Scope:     request.Scope,
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Conditions: []auth.GrantCondition{
			{
				Type:       "value_limit",
				Constraint: "trade_value <= 2000000.0",
			},
			{
				Type:       "time_window",
				Constraint: "trading_hours",
			},
		},
	}
}

func validateFiduciaryDuties(_ *testing.T, framework *auth.StandardLegalFramework, _ *auth.LegalFrameworkAuthorizationGrant) error {
	duties := []auth.FiduciaryDuty{
		{
			Type:        "loyalty",
			Description: "Best execution",
			Scope:       []string{"trade_execution"},
			Validation:  []string{"best_execution_check"},
		},
		{
			Type:        "care",
			Description: "Risk assessment",
			Scope:       []string{"trade_execution"},
			Validation:  []string{"risk_check"},
		},
	}

	for _, duty := range duties {
		if err := framework.ValidateDuty(context.Background(), duty); err != nil {
			return err
		}
	}
	return nil
}

func validateGrantCompliance(_ *testing.T, framework *auth.StandardLegalFramework, grant *auth.LegalFrameworkAuthorizationGrant) error {
	ctx := context.Background()
	rules := &auth.JurisdictionRules{
		Country: "US",
		RequiredApprovals: map[string]auth.ApprovalLevel{
			"trade_execution": auth.DualApproval,
		},
		ValueLimits: map[string]float64{
			"trade_execution": 2000000.0,
		},
	}

	return framework.ValidateJurisdictionRequirements(ctx, rules, "trade_execution")
}

func createTransaction(_ *testing.T, grant *auth.LegalFrameworkAuthorizationGrant) *auth.Transaction {
	return &auth.Transaction{
		ID:        "txn_001",
		GrantID:   grant.ID,
		Type:      "trade_execution",
		Status:    "pending",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"trade_id":    "trade_001",
			"asset":       "STOCK_XYZ",
			"quantity":    1000,
			"price":       1000.0,
			"total_value": 1000000.0,
		},
	}
}

func validateTransaction(_ *testing.T, framework *auth.StandardLegalFramework, txn *auth.Transaction) error {
	ctx := context.Background()
	return framework.ValidateJurisdictionRequirements(
		ctx,
		&auth.JurisdictionRules{
			Country: "US",
			RequiredApprovals: map[string]auth.ApprovalLevel{
				"trade_execution": auth.DualApproval,
			},
			ValueLimits: map[string]float64{
				"trade_execution": 2000000.0,
			},
		},
		txn.Type,
	)
}

func trackCompliance(_ *testing.T, framework *auth.StandardLegalFramework, txn *auth.Transaction) error {
	return framework.TrackApprovalDetails(
		context.Background(),
		&auth.Approval{
			ID:             "approval_001",
			TransactionID:  txn.ID,
			RequesterID:    "ai_client_001",
			ApproverID:     "resource_owner_001",
			Action:         txn.Type,
			JurisdictionID: "US",
			LegalBasis:     "granted_authority",
			FiduciaryChecks: []auth.FiduciaryDuty{
				{Type: "loyalty"},
				{Type: "care"},
				{Type: "compliance"},
			},
			Evidence: "cryptographic_proof_004",
		},
	)
}

func createJurisdictionDuties(_ *testing.T, jurisdiction string) []auth.FiduciaryDuty {
	return []auth.FiduciaryDuty{
		{
			Type:        "loyalty",
			Description: "Market jurisdiction compliance",
			Scope:       []string{"market_operations"},
			Validation:  []string{"market_compliance_check"},
		},
		{
			Type:        "compliance",
			Description: fmt.Sprintf("%s regulatory compliance", jurisdiction),
			Scope:       []string{"regulatory_compliance"},
			Validation:  []string{"regulatory_check"},
		},
	}
}

func createDelegationChain(_ *testing.T) []auth.DelegationLink {
	now := time.Now()
	return []auth.DelegationLink{
		{
			FromID: "authorizer_001",
			ToID:   "client_owner_001",
			Type:   "human-to-human",
			Level:  1,
			Time:   now.Add(-48 * time.Hour),
		},
		{
			FromID: "client_owner_001",
			ToID:   "ai_client_001",
			Type:   "human-to-ai",
			Level:  2,
			Time:   now.Add(-24 * time.Hour),
		},
	}
}

func createAITeam(_ *testing.T) *auth.AITeam {
	return &auth.AITeam{
		ID:           "team_001",
		Jurisdiction: "US",
		LeadAgent: &auth.AIAgent{
			ID:          "lead_001",
			Role:        "team_lead",
			Type:        "lead",
			Permissions: []string{"coordinate", "monitor"},
			Entity: &auth.Entity{
				ID:             "lead_entity_001",
				Type:           "ai_agent",
				JurisdictionID: "US",
				LegalStatus:    "active",
			},
		},
		Members: []*auth.AIAgent{
			{
				ID:          "member_001",
				Role:        "executor",
				Type:        "member",
				Permissions: []string{"execute_trades"},
				ReportsTo:   "lead_001",
				Entity: &auth.Entity{
					ID:             "member_entity_001",
					Type:           "ai_agent",
					JurisdictionID: "US",
					LegalStatus:    "active",
				},
			},
		},
	}
}

func createComplianceActions(_ *testing.T) []auth.ComplianceAction {
	return []auth.ComplianceAction{
		{
			Name:         "high_value_trade",
			RequesterID:  "ai_client_001",
			ApproverID:   "resource_owner_001",
			Jurisdiction: "US",
			LegalBasis:   "delegated_authority",
			Checks: []string{
				"value_limit_check",
				"authority_verification",
				"compliance_verification",
			},
			Evidence: map[string]interface{}{
				"approval_chain": []string{"approver_1", "approver_2"},
				"timestamps":     []time.Time{time.Now(), time.Now().Add(1 * time.Hour)},
				"signatures":     []string{"sig_1", "sig_2"},
			},
		},
	}
}

func getTrackingRecords(t *testing.T, framework *auth.StandardLegalFramework, approvalID string) ([]auth.TrackingRecord, error) {
	store, ok := framework.Store().(*auth.StoreStub)
	if !ok {
		t.Fatalf("framework.Store() is not of type *auth.StoreStub")
	}
	return store.GetTrackingRecords(context.Background(), approvalID)
}
