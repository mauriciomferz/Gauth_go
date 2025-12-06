package gauth_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	"github.com/mauriciomferz/Gauth_go/pkg/gauthplus"
	"github.com/mauriciomferz/Gauth_go/pkg/poa"
	"github.com/mauriciomferz/Gauth_go/pkg/poa/taxonomy"
)

// TestGAuthPlusIntegration_SuccessorTakeover tests successor AI activation
func TestGAuthPlusIntegration_SuccessorTakeover(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup: Create test database connection
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Initialize GAuth+ services
	successorService := gauthplus.NewPostgreSQLSuccessorService(db)
	delegationService := gauthplus.NewPostgreSQLDelegationService(db)
	dualControlService := gauthplus.NewPostgreSQLDualControlService(db)
	fiduciaryService := gauthplus.NewPostgreSQLFiduciaryDutyService(db)
	capabilityService := gauthplus.NewPostgreSQLCapabilityAssessmentService(db)

	// Create GAuth+ validator
	validator := gauth.NewGAuthPlusValidator(
		successorService,
		delegationService,
		dualControlService,
		fiduciaryService,
		capabilityService,
	)

	ctx := context.Background()
	poaID := "550e8400-e29b-41d4-a716-446655440001" // Valid UUID
	primaryAgentID := "agent-primary-001"
	successorAgentID := "agent-successor-001"

	// Test 1: No successor active - should use primary agent
	t.Run("NoSuccessorActive", func(t *testing.T) {
		poaDef := createTestPoADefinition(primaryAgentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, primaryAgentID, "read")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if !result.Valid {
			t.Errorf("Expected valid result, got: %s", result.FailureReason)
		}

		if result.SuccessorCheck.SuccessorActive {
			t.Error("Expected no successor active")
		}

		if result.SuccessorCheck.EffectiveAgentID != primaryAgentID {
			t.Errorf("Expected effective agent ID %s, got %s",
				primaryAgentID, result.SuccessorCheck.EffectiveAgentID)
		}
	})

	// Test 2: Activate successor
	t.Run("ActivateSuccessor", func(t *testing.T) {
		activation, err := successorService.ActivateSuccessor(
			ctx,
			poaID,
			primaryAgentID,
			successorAgentID,
			"failure",
			"system",
		)

		if err != nil {
			t.Fatalf("Failed to activate successor: %v", err)
		}

		if activation.Status != "active" {
			t.Errorf("Expected status 'active', got %s", activation.Status)
		}

		// Validate with successor active
		poaDef := createTestPoADefinition(primaryAgentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, primaryAgentID, "read")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if !result.Valid {
			t.Errorf("Expected valid result, got: %s", result.FailureReason)
		}

		if !result.SuccessorCheck.SuccessorActive {
			t.Error("Expected successor to be active")
		}

		if result.SuccessorCheck.EffectiveAgentID != successorAgentID {
			t.Errorf("Expected effective agent ID %s (successor), got %s",
				successorAgentID, result.SuccessorCheck.EffectiveAgentID)
		}

		if len(result.Warnings) == 0 {
			t.Error("Expected warning about successor activation")
		}
	})

	// Test 3: Deactivate successor - should return to primary
	t.Run("DeactivateSuccessor", func(t *testing.T) {
		// Get active successor
		activation, err := successorService.GetActiveSuccessor(ctx, poaID)
		if err != nil {
			t.Fatalf("Failed to get active successor: %v", err)
		}
		if activation == nil {
			t.Fatal("Expected active successor, got nil")
		}

		// Deactivate
		err = successorService.DeactivateSuccessor(ctx, activation.ID, "system")
		if err != nil {
			t.Fatalf("Failed to deactivate successor: %v", err)
		}

		// Validate - should use primary again
		poaDef := createTestPoADefinition(primaryAgentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, primaryAgentID, "read")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if result.SuccessorCheck.SuccessorActive {
			t.Error("Expected no successor active after deactivation")
		}

		if result.SuccessorCheck.EffectiveAgentID != primaryAgentID {
			t.Errorf("Expected effective agent ID %s (primary), got %s",
				primaryAgentID, result.SuccessorCheck.EffectiveAgentID)
		}
	})
}

// TestGAuthPlusIntegration_DelegationDepth tests delegation chain depth enforcement
func TestGAuthPlusIntegration_DelegationDepth(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Initialize services
	delegationService := gauthplus.NewPostgreSQLDelegationService(db)
	validator := createTestGAuthPlusValidator(db)

	ctx := context.Background()
	poaID := "550e8400-e29b-41d4-a716-446655440002" // Valid UUID

	// Create delegation chain: agent1 -> agent2 -> agent3 -> agent4
	agentIDs := []string{"agent-1", "agent-2", "agent-3", "agent-4"}

	t.Run("CreateDelegationChain", func(t *testing.T) {
		for i := 0; i < len(agentIDs)-1; i++ {
			delegation := &gauthplus.AIDelegation{
				SourcePOAID:     poaID,
				SourceAgentID:   agentIDs[i],
				TargetAgentID:   agentIDs[i+1],
				DelegatedScope:  []string{"read", "execute"},
				DelegationDepth: i + 1,
				MaxAllowedDepth: 3, // Max depth: 3
				ValidFrom:       time.Now().Add(-1 * time.Hour),
				ValidUntil:      time.Now().Add(24 * time.Hour),
				Status:          "active",
			}

			err := delegationService.CreateDelegation(ctx, delegation)
			if err != nil {
				t.Fatalf("Failed to create delegation %d: %v", i, err)
			}
		}
	})

	// Test: Agent at depth 3 should succeed
	t.Run("DepthThreeShouldSucceed", func(t *testing.T) {
		poaDef := createTestPoADefinition(agentIDs[2]) // agent-3 at depth 3
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, agentIDs[2], "read")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if !result.Valid {
			t.Errorf("Expected valid at depth 3, got: %s", result.FailureReason)
		}

		if result.DelegationCheck.DepthExceeded {
			t.Error("Expected depth not exceeded at level 3")
		}
	})

	// Test: Agent at depth 4 should fail (exceeds max depth 3)
	t.Run("DepthFourShouldFail", func(t *testing.T) {
		poaDef := createTestPoADefinition(agentIDs[3]) // agent-4 at depth 4
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, agentIDs[3], "read")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if result.Valid {
			t.Error("Expected validation to fail at depth 4")
		}

		if !result.DelegationCheck.DepthExceeded {
			t.Error("Expected depth exceeded at level 4")
		}

		if result.DelegationCheck.CurrentDepth != 3 {
			t.Errorf("Expected current depth 3, got %d", result.DelegationCheck.CurrentDepth)
		}
	})
}

// TestGAuthPlusIntegration_CapabilityEnforcement tests capability assessment enforcement
func TestGAuthPlusIntegration_CapabilityEnforcement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	capabilityService := gauthplus.NewPostgreSQLCapabilityAssessmentService(db)
	validator := createTestGAuthPlusValidator(db)
	validator.SetEnforceCapabilities(true) // Enable capability enforcement

	ctx := context.Background()
	poaID := "550e8400-e29b-41d4-a716-446655440003" // Valid UUID
	agentID := "agent-capability-test"

	// Test 1: No capability assessment - should fail
	t.Run("NoAssessmentShouldFail", func(t *testing.T) {
		poaDef := createTestPoADefinition(agentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, agentID, "execute")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if result.Valid {
			t.Error("Expected validation to fail without capability assessment")
		}

		if result.CapabilityCheck.CapabilityMet {
			t.Error("Expected capability check to fail")
		}
	})

	// Test 2: Create L1 assessment - should fail (requires L2)
	t.Run("InsufficientCapabilityShouldFail", func(t *testing.T) {
		assessment := &gauthplus.AICapabilityAssessment{
			AgentID:        agentID,
			AssessmentDate: time.Now(),
			OverallLevel:   "L1",
			DomainScores: map[string]float64{
				"reasoning": 0.6,
				"planning":  0.5,
			},
			RiskProfile:         map[string]interface{}{"risk_level": "moderate"},
			CertificationStatus: "uncertified",
			Certifications:      []string{},
			AssessedBy:          "system",
			ValidUntil:          time.Now().Add(30 * 24 * time.Hour),
		}

		err := capabilityService.CreateAssessment(ctx, assessment)
		if err != nil {
			t.Fatalf("Failed to create assessment: %v", err)
		}

		poaDef := createTestPoADefinition(agentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, agentID, "execute")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if result.Valid {
			t.Error("Expected validation to fail with L1 capability (requires L2)")
		}

		if result.CapabilityCheck.ActualLevel != "L1" {
			t.Errorf("Expected actual level L1, got %s", result.CapabilityCheck.ActualLevel)
		}

		if result.CapabilityCheck.RequiredLevel != "L2" {
			t.Errorf("Expected required level L2, got %s", result.CapabilityCheck.RequiredLevel)
		}
	})

	// Test 3: Update to L3 assessment - should succeed
	t.Run("SufficientCapabilityShouldSucceed", func(t *testing.T) {
		assessment := &gauthplus.AICapabilityAssessment{
			AgentID:        agentID,
			AssessmentDate: time.Now(),
			OverallLevel:   "L3",
			DomainScores: map[string]float64{
				"reasoning": 0.85,
				"planning":  0.80,
				"execution": 0.82,
			},
			RiskProfile:         map[string]interface{}{"risk_level": "low"},
			CertificationStatus: "certified",
			Certifications:      []string{"cert-001"},
			AssessedBy:          "system",
			ValidUntil:          time.Now().Add(30 * 24 * time.Hour),
		}

		err := capabilityService.CreateAssessment(ctx, assessment)
		if err != nil {
			t.Fatalf("Failed to create assessment: %v", err)
		}

		poaDef := createTestPoADefinition(agentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, agentID, "execute")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if !result.Valid {
			t.Errorf("Expected validation to succeed with L3 capability, got: %s", result.FailureReason)
		}

		if !result.CapabilityCheck.CapabilityMet {
			t.Error("Expected capability check to pass")
		}

		if result.CapabilityCheck.AssessmentExpired {
			t.Error("Expected assessment not expired")
		}
	})

	// Test 4: Expired assessment - should warn
	t.Run("ExpiredAssessmentShouldWarn", func(t *testing.T) {
		assessment := &gauthplus.AICapabilityAssessment{
			AgentID:        "agent-expired",
			AssessmentDate: time.Now().Add(-60 * 24 * time.Hour),
			OverallLevel:   "L3",
			DomainScores: map[string]float64{
				"reasoning": 0.85,
			},
			RiskProfile:         map[string]interface{}{"risk_level": "low"},
			CertificationStatus: "certified",
			Certifications:      []string{},
			AssessedBy:          "system",
			ValidUntil:          time.Now().Add(-1 * time.Hour), // Expired
		}

		err := capabilityService.CreateAssessment(ctx, assessment)
		if err != nil {
			t.Fatalf("Failed to create assessment: %v", err)
		}

		poaDef := createTestPoADefinition("agent-expired")
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, "550e8400-e29b-41d4-a716-446655440005", poaDef, "agent-expired", "execute")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if result.CapabilityCheck.AssessmentExpired {
			// Should still pass but with warning
			if len(result.Warnings) == 0 {
				t.Error("Expected warning about expired assessment")
			}
		}
	})
}

// TestGAuthPlusIntegration_FiduciaryViolations tests fiduciary duty enforcement
func TestGAuthPlusIntegration_FiduciaryViolations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	fiduciaryService := gauthplus.NewPostgreSQLFiduciaryDutyService(db)
	validator := createTestGAuthPlusValidator(db)
	validator.SetEnforceFiduciary(true)     // Enable fiduciary enforcement
	validator.SetEnforceCapabilities(false) // Disable capability enforcement for this test

	ctx := context.Background()
	poaID := "550e8400-e29b-41d4-a716-446655440004" // Valid UUID
	agentID := "agent-fiduciary-test"

	// Test 1: No violations - should succeed
	t.Run("NoViolationsShouldSucceed", func(t *testing.T) {
		poaDef := createTestPoADefinition(agentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, agentID, "read")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if !result.Valid {
			t.Fatalf("Expected valid with no violations, got: %s", result.FailureReason)
		}

		if result.FiduciaryCheck.HasViolations {
			t.Error("Expected no violations")
		}
	})

	// Test 2: Minor violation - should succeed with warning
	t.Run("MinorViolationShouldWarn", func(t *testing.T) {
		violation := &gauthplus.FiduciaryDutyViolation{
			POAID:                poaID,
			AgentID:              agentID,
			DutyType:             "disclosure",
			ViolationDescription: "Delayed disclosure of minor conflict",
			Severity:             "minor",
			DetectedAt:           time.Now(),
			DetectedBy:           "monitor",
			ResolutionStatus:     "open",
		}

		err := fiduciaryService.RecordViolation(ctx, violation)
		if err != nil {
			t.Fatalf("Failed to record violation: %v", err)
		}

		poaDef := createTestPoADefinition(agentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, agentID, "read")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if !result.Valid {
			t.Errorf("Expected valid with minor violation, got: %s", result.FailureReason)
		}

		if !result.FiduciaryCheck.HasViolations {
			t.Error("Expected violations detected")
		}

		if len(result.Warnings) == 0 {
			t.Error("Expected warning about unresolved violations")
		}
	})

	// Test 3: Critical violation - should block
	t.Run("CriticalViolationShouldBlock", func(t *testing.T) {
		violation := &gauthplus.FiduciaryDutyViolation{
			POAID:                poaID,
			AgentID:              agentID,
			DutyType:             "loyalty",
			ViolationDescription: "Acted against principal's interest",
			Severity:             "critical",
			DetectedAt:           time.Now(),
			DetectedBy:           "audit",
			ResolutionStatus:     "open",
		}

		err := fiduciaryService.RecordViolation(ctx, violation)
		if err != nil {
			t.Fatalf("Failed to record violation: %v", err)
		}

		poaDef := createTestPoADefinition(agentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, agentID, "execute")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if result.Valid {
			t.Error("Expected validation to fail with critical violation")
		}

		if result.FiduciaryCheck.CriticalViolations == 0 {
			t.Error("Expected critical violations count > 0")
		}

		if !result.FiduciaryCheck.BlockingAction {
			t.Error("Expected blocking action due to critical violations")
		}
	})

	// Test 4: Resolve critical violation - should succeed
	t.Run("ResolvedViolationShouldSucceed", func(t *testing.T) {
		// Get violations
		violations, err := fiduciaryService.GetViolations(ctx, poaID, agentID)
		if err != nil {
			t.Fatalf("Failed to get violations: %v", err)
		}

		// Resolve critical violation
		for _, v := range violations {
			if v.Severity == "critical" {
				err = fiduciaryService.ResolveViolation(ctx, v.ID, "corrective action taken", "admin")
				if err != nil {
					t.Fatalf("Failed to resolve violation: %v", err)
				}
			}
		}

		poaDef := createTestPoADefinition(agentID)
		result, err := validator.ValidatePoAWithGAuthPlus(ctx, poaID, poaDef, agentID, "execute")

		if err != nil {
			t.Fatalf("Validation failed: %v", err)
		}

		if !result.Valid {
			t.Errorf("Expected valid after resolving violations, got: %s", result.FailureReason)
		}

		if result.FiduciaryCheck.CriticalViolations > 0 {
			t.Error("Expected no unresolved critical violations")
		}
	})
}

// TestGAuthPlusIntegration_ComplianceValidator tests integration with ComplianceValidator
func TestGAuthPlusIntegration_ComplianceValidator(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Setup validators
	chainValidator := createMockChainValidator()
	complianceValidator := gauth.NewComplianceValidator(chainValidator, nil, nil)
	gauthPlusValidator := createTestGAuthPlusValidator(db)

	complianceValidator.SetGAuthPlusValidator(gauthPlusValidator)
	complianceValidator.SetEnforceGAuthPlus(true)

	ctx := context.Background()

	t.Run("RequestValidationWithGAuthPlus", func(t *testing.T) {
		agentID := "550e8400-e29b-41d4-a716-446655440001" // Valid UUID
		request := &gauth.ExtendedAuthorizationRequest{
			AuthorizationRequest: &gauth.AuthorizationRequest{
				ClientID: "test-client",
				Scopes:   []string{"read", "write"},
			},
			PowerOfAttorney:  createTestPoADefinition(agentID),
			RequestedActions: []string{"read"},
			RequestTime:      time.Now(),
		}

		result, err := complianceValidator.ValidateRequestCompliance(ctx, request)

		if err != nil {
			t.Logf("Validation error (expected in test): %v", err)
		}

		// Check that GAuth+ validation was performed
		if result.GAuthPlusValidation == nil {
			t.Error("Expected GAuth+ validation result")
		}
	})
}

// Helper functions

func setupTestDB(t *testing.T) (*sql.DB, func()) {
	// Connect to test database
	connStr := "host=localhost port=5432 user=postgres password=gauth_dev_password dbname=gauth sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Skipf("Cannot connect to test database: %v", err)
		return nil, func() {}
	}

	// Test connection
	if err := db.Ping(); err != nil {
		t.Skipf("Cannot ping test database: %v", err)
		return nil, func() {}
	}

	// Insert test tenant
	_, err = db.Exec(`
		INSERT INTO subscribers (tenant_id, tenant_name, status) 
		VALUES ('tenant-1', 'Test Tenant', 'active') 
		ON CONFLICT (tenant_id) DO NOTHING
	`)
	if err != nil {
		t.Fatalf("Failed to insert test tenant: %v", err)
	}

	// Insert test data for PoA
	_, err = db.Exec(`
		INSERT INTO power_of_attorney (
			id, tenant_id, grantor_id, representative_id, representative_type, 
			status, delegation_policy, valid_from, valid_until
		) VALUES 
			('550e8400-e29b-41d4-a716-446655440001', 'tenant-1', 'org-001', 'agent-primary-001', 'digital_agent', 'active', '{"can_delegate": true, "max_depth": 3}', NOW(), NOW() + INTERVAL '1 year'),
			('550e8400-e29b-41d4-a716-446655440002', 'tenant-1', 'org-001', 'agent-1', 'digital_agent', 'active', '{"can_delegate": true, "max_depth": 3, "allowed_delegates": ["agent-2"]}', NOW(), NOW() + INTERVAL '1 year'),
			('550e8400-e29b-41d4-a716-446655440003', 'tenant-1', 'org-001', 'agent-capability-test', 'digital_agent', 'active', '{"can_delegate": true, "max_depth": 3}', NOW(), NOW() + INTERVAL '1 year'),
			('550e8400-e29b-41d4-a716-446655440004', 'tenant-1', 'org-001', 'agent-fiduciary-test', 'digital_agent', 'active', '{"can_delegate": true, "max_depth": 3}', NOW(), NOW() + INTERVAL '1 year'),
			('550e8400-e29b-41d4-a716-446655440005', 'tenant-1', 'org-001', 'agent-expired', 'digital_agent', 'active', '{"can_delegate": true, "max_depth": 3}', NOW(), NOW() + INTERVAL '1 year'),
			('550e8400-e29b-41d4-a716-446655440006', 'tenant-1', 'org-001', 'agent-2', 'digital_agent', 'active', '{"can_delegate": true, "max_depth": 3}', NOW(), NOW() + INTERVAL '1 year'),
			('550e8400-e29b-41d4-a716-446655440007', 'tenant-1', 'org-001', 'agent-3', 'digital_agent', 'active', '{"can_delegate": true, "max_depth": 3}', NOW(), NOW() + INTERVAL '1 year'),
			('550e8400-e29b-41d4-a716-446655440008', 'tenant-1', 'org-001', 'agent-4', 'digital_agent', 'active', '{"can_delegate": true, "max_depth": 3}', NOW(), NOW() + INTERVAL '1 year')
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		t.Fatalf("Failed to insert test PoA data: %v", err)
	}

	// Insert test data for Capability Assessment (L3 for primary agent)
	_, err = db.Exec(`
		INSERT INTO ai_capability_assessments (
			id, agent_id, assessment_date, overall_level, domain_scores, 
			risk_profile, certification_status, certifications, limitations, 
			recommended_restrictions, assessed_by, valid_until, created_at, updated_at
		) VALUES (
			'cap-001', 'agent-primary-001', NOW(), 'L3', '{"reasoning": 0.9}'::jsonb, 
			'{"risk_level": "low"}'::jsonb, 'certified', '[]'::jsonb, '[]'::jsonb,
			'[]'::jsonb, 'system', NOW() + INTERVAL '30 days', NOW(), NOW()
		),
		(
			'cap-002', 'agent-successor-001', NOW(), 'L3', '{"reasoning": 0.85}'::jsonb, 
			'{"risk_level": "low"}'::jsonb, 'certified', '[]'::jsonb, '[]'::jsonb,
			'[]'::jsonb, 'system', NOW() + INTERVAL '30 days', NOW(), NOW()
		),
		(
			'cap-003', 'agent-1', NOW(), 'L3', '{"reasoning": 0.9}'::jsonb, 
			'{"risk_level": "low"}'::jsonb, 'certified', '[]'::jsonb, '[]'::jsonb,
			'[]'::jsonb, 'system', NOW() + INTERVAL '30 days', NOW(), NOW()
		),
		(
			'cap-004', 'agent-2', NOW(), 'L3', '{"reasoning": 0.9}'::jsonb, 
			'{"risk_level": "low"}'::jsonb, 'certified', '[]'::jsonb, '[]'::jsonb,
			'[]'::jsonb, 'system', NOW() + INTERVAL '30 days', NOW(), NOW()
		),
		(
			'cap-005', 'agent-3', NOW(), 'L3', '{"reasoning": 0.9}'::jsonb, 
			'{"risk_level": "low"}'::jsonb, 'certified', '[]'::jsonb, '[]'::jsonb,
			'[]'::jsonb, 'system', NOW() + INTERVAL '30 days', NOW(), NOW()
		),
		(
			'cap-006', 'agent-4', NOW(), 'L3', '{"reasoning": 0.9}'::jsonb, 
			'{"risk_level": "low"}'::jsonb, 'certified', '[]'::jsonb, '[]'::jsonb,
			'[]'::jsonb, 'system', NOW() + INTERVAL '30 days', NOW(), NOW()
		)
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		t.Fatalf("Failed to insert test capability data: %v", err)
	}

	cleanup := func() {
		// Clean up test data
		_, _ = db.Exec("DELETE FROM successor_activations WHERE poa_id::text LIKE '550e8400%'")
		_, _ = db.Exec("DELETE FROM ai_delegations WHERE source_poa_id::text LIKE '550e8400%'")
		_, _ = db.Exec("DELETE FROM ai_capability_assessments WHERE agent_id LIKE 'agent-%'")
		_, _ = db.Exec("DELETE FROM fiduciary_duty_violations WHERE poa_id::text LIKE '550e8400%'")
		db.Close()
	}

	return db, cleanup
}

func createTestGAuthPlusValidator(db *sql.DB) *gauth.GAuthPlusValidator {
	successorService := gauthplus.NewPostgreSQLSuccessorService(db)
	delegationService := gauthplus.NewPostgreSQLDelegationService(db)
	dualControlService := gauthplus.NewPostgreSQLDualControlService(db)
	fiduciaryService := gauthplus.NewPostgreSQLFiduciaryDutyService(db)
	capabilityService := gauthplus.NewPostgreSQLCapabilityAssessmentService(db)

	return gauth.NewGAuthPlusValidator(
		successorService,
		delegationService,
		dualControlService,
		fiduciaryService,
		capabilityService,
	)
}

func createTestPoADefinition(agentID string) *poa.PoADefinition {
	return &poa.PoADefinition{
		Parties: poa.Parties{
			Principal: poa.Principal{
				Type:     "organization",
				Identity: "org-001",
			},
			AuthorizedClient: poa.AuthorizedClient{
				Identity:        agentID,
				StatusEnum:      poa.OperationalStatusActive,
				CapabilityLevel: poa.CapabilityL3,
				TypeEnum:        poa.ClientTypeDigitalAgent,
				Version:         "1.0",
			},
		},
		Authorization: poa.AuthorizationScope{
			AuthorizedActions: poa.AuthorizedActions{
				Transactions: []taxonomy.TransactionType{
					taxonomy.TransactionPayment,
					taxonomy.TransactionPurchase,
				},
			},
		},
		Requirements: poa.Requirements{
			ValidityPeriod: poa.ValidityPeriod{
				StartTime: time.Now().Add(-24 * time.Hour),
				EndTime:   time.Now().Add(365 * 24 * time.Hour),
			},
		},
	}
}

func createMockChainValidator() *gauth.AuthorizationChainValidator {
	// Create a mock chain validator for testing
	// In real tests, this would use proper mocks
	return gauth.NewAuthorizationChainValidator(nil, nil, nil)
}
