//go:build go1.18

package test

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/testutil"
)

// FuzzCreateDelegation validates that CreateDelegation never panics with arbitrary input and enforces invariants.
func FuzzCreateDelegation(f *testing.F) {
	// Seed corpus with a few valid baselines
	f.Add("alice@example.com", "bob@example.com", int64(3600), "transaction:execute")
	f.Add("", "bob@example.com", int64(0), "") // clearly invalid baseline

	f.Fuzz(func(t *testing.T, grantor, grantee string, durationSeconds int64, action string) {
		authorizer := authz.NewMemoryAuthorizer()
		// Grant create for whatever grantor string appears (unless empty)
		if grantor != "" {
			authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: grantor, Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
		}
		logger := audit.NewMemoryLogger(testutil.NoopLogger{})
		svc := rfc0111.NewService(logger, authorizer)

		// Normalize duration
		if durationSeconds < 0 {
			durationSeconds = -durationSeconds
		}
		if durationSeconds > 86400 {
			durationSeconds = 86400
		} // clamp at 24h
		dur := time.Duration(durationSeconds) * time.Second
		req := rfc0111.DelegationRequest{Grantor: grantor, Grantee: grantee, Scope: []string{}, Duration: dur}
		if action != "" {
			req.Scope = []string{action}
		}

		resp, err := svc.CreateDelegationCtx(context.Background(), req)
		if err == nil {
			// Basic invariants on successful creation
			if resp.POA.ID == "" {
				t.Fatalf("empty POA ID on success")
			}
			if resp.POA.Grantor != grantor {
				t.Fatalf("grantor mismatch")
			}
			if resp.POA.Grantee != grantee {
				t.Fatalf("grantee mismatch")
			}
			if resp.POA.ValidUntil.Before(resp.POA.ValidFrom) {
				t.Fatalf("validity window inverted")
			}
			// Validate allowed action if present
			if action != "" {
				if vErr := svc.ValidateDelegationCtx(context.Background(), resp.POA.ID, grantee, action); vErr != nil {
					t.Fatalf("expected validate succeed: %v", vErr)
				}
			}
		}
	})
}
