package main

import (
	"context"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc0111"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/testutil"
)

// BenchmarkDelegationLifecycle benchmarks create -> validate -> revoke for a single delegation.
func BenchmarkDelegationLifecycle(b *testing.B) {
	authorizer := authz.NewMemoryAuthorizer()
	authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	authorizer.AddPolicy(authz.Policy{ID: "allow-revoke", Subject: "alice@example.com", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
	svc := rfc0111.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := rfc0111.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: 5 * time.Minute}
		resp, err := svc.CreateDelegationCtx(context.Background(), req)
		if err != nil {
			b.Fatalf("create failed: %v", err)
		}
		if err := svc.ValidateDelegationCtx(context.Background(), resp.POA.ID, req.Grantee, "transaction:execute"); err != nil {
			b.Fatalf("validate failed: %v", err)
		}
		if err := svc.RevokeDelegationCtx(context.Background(), resp.POA.ID, req.Grantor); err != nil {
			b.Fatalf("revoke failed: %v", err)
		}
	}
}

// BenchmarkDelegationValidateOnly benchmarks repeated validation of an active delegation.
func BenchmarkDelegationValidateOnly(b *testing.B) {
	const batch = 10000 // recreate service every 10k iterations to bound audit log growth
	b.ReportAllocs()
	for i := 0; i < b.N; i += batch {
		remaining := b.N - i
		n := batch
		if remaining < batch {
			n = remaining
		}
		authorizer := authz.NewMemoryAuthorizer()
		authorizer.AddPolicy(authz.Policy{ID: "allow-create", Subject: "alice@example.com", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
		authorizer.AddPolicy(authz.Policy{ID: "allow-revoke", Subject: "alice@example.com", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})
		svc := rfc0111.NewService(audit.NewMemoryLogger(testutil.NoopLogger{}), authorizer)
		req := rfc0111.DelegationRequest{Grantor: "alice@example.com", Grantee: "bob@example.com", Scope: []string{"transaction:execute"}, Duration: time.Hour}
		resp, err := svc.CreateDelegation(req)
		if err != nil {
			b.Fatalf("setup create failed: %v", err)
		}
		for j := 0; j < n; j++ {
			if err := svc.ValidateDelegationCtx(context.Background(), resp.POA.ID, req.Grantee, "transaction:execute"); err != nil {
				b.Fatalf("validate failed: %v", err)
			}
		}
	}
}
