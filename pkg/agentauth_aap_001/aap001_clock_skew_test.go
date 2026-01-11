package agentauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/aap"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// TestClockSkewTolerance validates not-before and expiry grace window behavior controlled by AGENTAUTH_CLOCK_SKEW_SECONDS.
func TestClockSkewTolerance(t *testing.T) {
	t.Setenv("AGENTAUTH_CLOCK_SKEW_SECONDS", "120") // 2 minute tolerance
	repo := newMemoryRepository()
	svc := &Service{repo: repo, audit: audit.NewMemoryLogger(nil), authz: authz.NewMemoryAuthorizer(), nowFn: time.Now}

	now := time.Now().UTC()
	// POA future within skew (ValidFrom 60s ahead) should validate successfully.
	poaFuture := &PowerOfAttorney{
		ID: "poa_future", Grantor: "alice", Grantee: "bob", Scope: []string{"read"},
		ValidFrom: now.Add(60 * time.Second), ValidUntil: now.Add(10 * time.Minute),
		Status: POAStatusActive, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	if err := repo.Create(poaFuture); err != nil {
		t.Fatalf("create future poa: %v", err)
	}
	if err := svc.ValidateDelegationRich(context.Background(), poaFuture.ID, "bob", ValidationContext{Action: "read"}); err != nil {
		t.Fatalf("expected validation success within skew for future valid_from: %v", err)
	}

	// POA future outside skew (ValidFrom 200s ahead) should fail.
	poaOutside := &PowerOfAttorney{
		ID: "poa_outside", Grantor: "alice", Grantee: "bob", Scope: []string{"read"},
		ValidFrom: now.Add(200 * time.Second), ValidUntil: now.Add(10 * time.Minute),
		Status: POAStatusActive, CreatedAt: now, UpdatedAt: now, Version: 1,
	}
	if err := repo.Create(poaOutside); err != nil {
		t.Fatalf("create outside poa: %v", err)
	}
	if err := svc.ValidateDelegationRich(context.Background(), poaOutside.ID, "bob", ValidationContext{Action: "read"}); err == nil {
		t.Fatalf("expected failure for not-yet-valid delegation outside skew tolerance")
	} else if e, ok := err.(aap.RFCError); !ok || e.Code != aap.ErrInvalidRequest {
		t.Fatalf("expected invalid_request code for not-yet-valid, got: %v", err)
	}

	// POA expired within skew (expired 30s ago) should still validate.
	poaGrace := &PowerOfAttorney{
		ID: "poa_grace", Grantor: "alice", Grantee: "bob", Scope: []string{"read"},
		ValidFrom: now.Add(-10 * time.Minute), ValidUntil: now.Add(-30 * time.Second),
		Status: POAStatusActive, CreatedAt: now.Add(-10 * time.Minute),
		UpdatedAt: now.Add(-30 * time.Second), Version: 1,
	}
	if err := repo.Create(poaGrace); err != nil {
		t.Fatalf("create grace poa: %v", err)
	}
	if err := svc.ValidateDelegationRich(context.Background(), poaGrace.ID, "bob", ValidationContext{Action: "read"}); err != nil {
		t.Fatalf("expected success for expired-within-skew delegation: %v", err)
	}

	// POA expired outside skew (expired 5 minutes ago) should fail with expired.
	poaExpired := &PowerOfAttorney{
		ID: "poa_expired", Grantor: "alice", Grantee: "bob", Scope: []string{"read"},
		ValidFrom: now.Add(-10 * time.Minute), ValidUntil: now.Add(-5 * time.Minute),
		Status: POAStatusActive, CreatedAt: now.Add(-10 * time.Minute),
		UpdatedAt: now.Add(-5 * time.Minute), Version: 1,
	}
	if err := repo.Create(poaExpired); err != nil {
		t.Fatalf("create expired poa: %v", err)
	}
	if err := svc.ValidateDelegationRich(context.Background(), poaExpired.ID, "bob", ValidationContext{Action: "read"}); err == nil {
		t.Fatalf("expected failure for expired delegation outside skew tolerance")
	} else if e, ok := err.(aap.RFCError); !ok || e.Code != aap.ErrExpired {
		t.Fatalf("expected expired code, got: %v", err)
	}
}
