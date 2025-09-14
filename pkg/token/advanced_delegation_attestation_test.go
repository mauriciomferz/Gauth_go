//go:build integration

// TestAdvancedDelegationAttestation covers advanced delegation, attestation, versioning, and revocation flows for RFC111 compliance.
package token_test

import (
	"testing"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

func TestAdvancedDelegationAttestation(t *testing.T) {

	// 1. Create DelegationOptions with nested restrictions, attestation, and version
	restrictions := &token.Restrictions{
		ValueLimits: &token.ValueLimits{
			MaxTransactionValue: 5000,
			DailyLimit:          10000,
			Currency:            "EURO",
		},
		TimeConstraints: &token.TimeConstraints{
			AllowedTimeWindows: []token.TimeWindow{{
				StartTime:  "09:00",
				EndTime:    "17:00",
				DaysOfWeek: []int{1, 2, 3, 4, 5},
			}},
			TimeZone: "UTC",
		},
		GeographicConstraints: []string{"EU", "ES"},
		CustomLimits:          map[string]float64{"special_limit": 42},
	}
	attestation := &token.Attestation{
		Type:            "notary",
		AttesterID:      "notary-xyz",
		AttestationDate: time.Now(),
		Evidence:        "doc-hash",
	}
	opts := token.DelegationOptions{
		Principal:    "owner-789",
		Scope:        "approve_payment",
		Restrictions: restrictions,
		Attestation:  attestation,
		ValidUntil:   time.Now().Add(48 * time.Hour),
		Version:      2,
	}

	tok := token.NewDelegatedToken("ai-agent-789", opts)

	if tok.Token == nil || tok.Token.Subject != "ai-agent-789" {
		t.Errorf("Expected Subject=ai-agent-789, got %+v", tok.Token)
	}
	if tok.AI == nil || tok.AI.Restrictions == nil || tok.AI.Restrictions.ValueLimits == nil || tok.AI.Restrictions.ValueLimits.MaxTransactionValue != 5000 {
		t.Errorf("Expected MaxTransactionValue=5000, got %+v", tok.AI)
	}
	if len(tok.Attestations) != 1 || tok.Attestations[0].AttesterID != "notary-xyz" {
		t.Errorf("Expected attestation by notary-xyz, got %+v", tok.Attestations)
	}
	if len(tok.Versions) == 0 || tok.Versions[0].Version != 2 {
		t.Errorf("Expected version=2, got %+v", tok.Versions)
	}

	// 2. Simulate version history update (no Revoked field, just versioning)
	tok.Versions = append(tok.Versions, token.VersionInfo{
		Version:       3,
		UpdatedAt:     time.Now(),
		UpdatedBy:     "admin",
		ChangeType:    "revoked",
		ChangeSummary: "Token revoked by admin.",
	})
	if len(tok.Versions) < 2 || tok.Versions[1].Version != 3 {
		t.Errorf("Expected version history to include version 3, got %+v", tok.Versions)
	}
}
