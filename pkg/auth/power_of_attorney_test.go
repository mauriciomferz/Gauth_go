package auth

import (
	"context"
	"testing"
)

func TestAuthorizePowerOfAttorneyValidation(t *testing.T) {
	svc := NewRFCCompliantService()
	ctx := context.Background()

	base := PowerOfAttorneyRequest{
		ClientID:     "client1",
		ResponseType: "code",
		Scope:        "ai_power_of_attorney,financial_transactions",
		RedirectURI:  "https://cb.example.com",
		PowerType:    "financial_transactions",
		PrincipalID:  "principal-xyz",
		AIAgentID:    "agent-123",
		Jurisdiction: "US",
		LegalBasis:   "law2024",
	}

	t.Run("valid request", func(t *testing.T) {
		if _, err := svc.AuthorizePowerOfAttorney(ctx, base); err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
	})

	t.Run("invalid jurisdiction", func(t *testing.T) {
		req := base
		req.Jurisdiction = "ZZZ"
		if _, err := svc.AuthorizePowerOfAttorney(ctx, req); err == nil {
			t.Fatalf("expected error for invalid jurisdiction")
		}
	})

	t.Run("disallowed scope", func(t *testing.T) {
		req := base
		req.Scope = base.Scope + ",nuclear_launch_codes"
		if _, err := svc.AuthorizePowerOfAttorney(ctx, req); err == nil {
			t.Fatalf("expected error for disallowed capability")
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		req := base
		req.ClientID = ""
		if _, err := svc.AuthorizePowerOfAttorney(ctx, req); err == nil {
			t.Fatalf("expected error for missing required fields")
		}
	})
}

func TestCreateAdvancedDelegationValidation(t *testing.T) {
	svc := NewRFCCompliantService()
	ctx := context.Background()

	base := DelegationRequest{
		PrincipalID:    "principal-xyz",
		DelegateID:     "delegate-abc",
		ValidityPeriod: ValidityPeriod{Days: 10},
		AttestationRequirement: AttestationRequirement{
			Attesters: []string{"attester1"},
		},
	}

	t.Run("valid request", func(t *testing.T) {
		if _, err := svc.CreateAdvancedDelegation(ctx, base); err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
	})

	t.Run("missing principal", func(t *testing.T) {
		req := base
		req.PrincipalID = ""
		if _, err := svc.CreateAdvancedDelegation(ctx, req); err == nil {
			t.Fatalf("expected error for missing principal")
		}
	})

	t.Run("invalid period", func(t *testing.T) {
		req := base
		req.ValidityPeriod.Days = 0
		if _, err := svc.CreateAdvancedDelegation(ctx, req); err == nil {
			t.Fatalf("expected error for invalid period")
		}
	})

	t.Run("no attesters", func(t *testing.T) {
		req := base
		req.AttestationRequirement.Attesters = nil
		if _, err := svc.CreateAdvancedDelegation(ctx, req); err == nil {
			t.Fatalf("expected error for missing attesters")
		}
	})
}
