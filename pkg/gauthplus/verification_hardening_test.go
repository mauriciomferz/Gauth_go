package gauthplus

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/registry"
)

func TestVerificationService_Hardened(t *testing.T) {
	// Setup mock services
	regSvc := registry.NewMockCommercialRegisterService()
	principalVerifier := NewDefaultPrincipalVerifier()

	// Setup Attestation Signer (Phase 11)
	_, privKey, _ := ed25519.GenerateKey(rand.Reader)
	signer := NewDefaultAttestationSigner("test-signer", privKey)
	verifier := NewDefaultAttestationVerifier()
	verifier.RegisterKey("test-signer", privKey.Public().(ed25519.PublicKey))

	// Mock Store (simplified)
	poaStore := &verifMockPoAStore{}

	svc := NewVerificationService(
		poaStore,
		&verifMockDelegationService{},
		&verifMockCapabilityService{},
		&verifMockFiduciaryService{},
		principalVerifier,
		verifier,
		signer,
		regSvc,
	)

	ctx := context.Background()

	t.Run("VerifyRepresentativePosition_Success", func(t *testing.T) {
		// "Dr. Max Mustermann" is in mock register for "HRB 12345" in "DE"
		res, err := svc.VerifyRepresentativePosition(ctx, "Dr. Max Mustermann", "HRB 12345:DE")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if !res.Valid {
			t.Error("Expected representative to be valid")
		}
		if res.Position != "Geschäftsführer" {
			t.Errorf("Expected position 'Geschäftsführer', got '%s'", res.Position)
		}
		if res.Attestation == nil {
			t.Fatal("Expected attestation to be generated")
		}

		// Verify the proof
		valid, err := verifier.Verify(ctx, *res.Attestation)
		if err != nil || !valid {
			t.Errorf("Failed to verify generated attestation: %v", err)
		}
	})

	t.Run("VerifyPrincipalStatus_TypeCheck", func(t *testing.T) {
		// Human
		res, err := svc.VerifyPrincipalStatus(ctx, "human-user-123")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.EntityType != "human" {
			t.Errorf("Expected EntityType 'human', got '%s'", res.EntityType)
		}
		if res.Attestation == nil {
			t.Fatal("Expected attestation for human principal")
		}

		// AI
		res, err = svc.VerifyPrincipalStatus(ctx, "AI-AGENT-007")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if res.EntityType != "ai_agent" {
			t.Errorf("Expected EntityType 'ai_agent', got '%s'", res.EntityType)
		}
	})

	t.Run("GenerateVerificationReport_AttestationAggregation", func(t *testing.T) {
		action := Action{Type: "transaction", Resource: "bank-account", Operation: "transfer"}
		report, err := svc.GenerateVerificationReport(ctx, "poa-123", action)
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if len(report.Attestations) == 0 {
			t.Error("Expected report to aggregate attestations")
		}

		// Check for PrincipalStatus attestation in the list
		found := false
		for _, att := range report.Attestations {
			if att.Type == "compliance" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Did not find compliance attestation in report")
		}
	})
}

// Minimal mock services for testing
type verifMockPoAStore struct{ PoAStore }

func (m *verifMockPoAStore) GetPoA(ctx context.Context, id string) (*EnhancedPoA, error) {
	return &EnhancedPoA{
		ID:         id,
		IssuerID:   "human-1",
		GranteeID:  "agent-1",
		ValidFrom:  time.Now().Add(-1 * time.Hour),
		ValidUntil: time.Now().Add(1 * time.Hour),
		Status:     "active",
		Attestations: []Attestation{
			{ID: "att-1", Status: "verified", Verified: true, Provider: "test-signer"},
		},
	}, nil
}
func (m *verifMockPoAStore) IsRevoked(ctx context.Context, id string) (bool, *RevocationInfo, error) {
	return false, nil, nil
}
func (m *verifMockPoAStore) GetPoAsByGrantee(ctx context.Context, granteeID string) ([]*EnhancedPoA, error) {
	return nil, nil // Return empty to stay safe
}

type verifMockDelegationService struct{ DelegationService }

func (m *verifMockDelegationService) ValidateDelegation(ctx context.Context, s, t string, sc []string, d int) error {
	return nil
}

type verifMockCapabilityService struct{ CapabilityAssessmentService }

func (m *verifMockCapabilityService) GetLatestAssessment(ctx context.Context, id string) (*AICapabilityAssessment, error) {
	return &AICapabilityAssessment{OverallLevel: "L4", AssessmentDate: time.Now()}, nil
}
func (m *verifMockCapabilityService) CheckCapabilityMatch(ctx context.Context, id string, reqs *CapabilityRequirements) (bool, []string, error) {
	return true, nil, nil
}

type verifMockFiduciaryService struct{ FiduciaryDutyService }

func (m *verifMockFiduciaryService) GetViolations(ctx context.Context, p, a string) ([]*FiduciaryDutyViolation, error) {
	return nil, nil // No violations
}
