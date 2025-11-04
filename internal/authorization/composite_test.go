package authorization

import (
	"testing"
	"time"
)

func minimalArtifact() *CompositeAuthorizationArtifact {
	now := time.Now().UTC()
	vf := now.Add(1 * time.Hour)
	vu := now.Add(2 * time.Hour)
	return &CompositeAuthorizationArtifact{
		AISystemID:           "ai_v1",
		AuthorizationGrant:   &AuthorizationGrant{Type: "general", Scope: []string{"financial_operations"}, ValidFrom: &vf, ValidUntil: &vu, Revocable: true},
		PowersGranted:        &PowersGranted{BasicPowers: []string{"financial_operations"}},
		DecisionAuthority:    &DecisionAuthority{AutonomousDecisions: []string{"routine_invoice_approval"}},
		TransactionRights:    &TransactionRights{AllowedTransactionTypes: []string{"vendor_payments"}},
		ActionPermissions:    &ActionPermissions{SystemActions: []string{"generate_reports"}},
		DualControlPrinciple: &DualControlPrinciple{Enabled: true},
		AuthorizationCascade: &AuthorizationCascade{AccountabilityChain: []string{"ceo_001", "cfo_001", "ai_v1"}},
		ExpiresAt:            now.Add(4 * time.Hour),
	}
}

func TestActivateSuccess(t *testing.T) {
	m := &Manager{}
	art := minimalArtifact()
	state, err := m.Activate(art)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if state.CanonicalHash == "" {
		t.Fatalf("canonical hash empty")
	}
	if state.PreviousArtifactHash != "" {
		t.Fatalf("expected empty previous hash on first activation")
	}
}

func TestActivateConflict(t *testing.T) {
	m := &Manager{}
	first := minimalArtifact()
	if _, err := m.Activate(first); err != nil {
		t.Fatalf("first activation failed: %v", err)
	}
	// Second artifact valid_from before first expires -> conflict
	second := minimalArtifact()
	if _, err := m.Activate(second); err == nil {
		t.Fatalf("expected conflict error")
	} else if err != ErrConflict {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestActivateInvalidExpired(t *testing.T) {
	m := &Manager{}
	art := minimalArtifact()
	art.ExpiresAt = time.Now().Add(-1 * time.Hour)
	if _, err := m.Activate(art); err == nil {
		t.Fatalf("expected invalid (expired) error")
	}
}

func TestActivateInvalidMissing(t *testing.T) {
	m := &Manager{}
	art := &CompositeAuthorizationArtifact{AISystemID: "ai_v1", ExpiresAt: time.Now().Add(2 * time.Hour)}
	if _, err := m.Activate(art); err == nil {
		t.Fatalf("expected invalid error for missing required nested blocks")
	}
}
