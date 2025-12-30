package agentauth_aap_001

import (
	"encoding/base64"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// BenchmarkCreateDelegation establishes a baseline for creation throughput.
func BenchmarkCreateDelegation(b *testing.B) {
	authzMem := authz.NewMemoryAuthorizer()
	authzMem.AddPolicy(authz.Policy{ID: "bench-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), authzMem)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Duration: time.Minute}
		if _, err := svc.CreateDelegation(req); err != nil {
			b.Fatalf("create: %v", err)
		}
	}
}

// BenchmarkValidateDelegation measures validation latency (hot path).
func BenchmarkValidateDelegation(b *testing.B) {
	authzMem := authz.NewMemoryAuthorizer()
	authzMem.AddPolicy(authz.Policy{ID: "bench-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), authzMem)
	dr, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Duration: time.Minute})
	if err != nil {
		b.Fatalf("setup create: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := svc.ValidateDelegation(dr.POA.ID, "bob", "transaction:execute"); err != nil {
			b.Fatalf("validate: %v", err)
		}
	}
}

// BenchmarkValidateDelegationWithMetrics measures validation latency when in-memory metrics collection is enabled.
func BenchmarkValidateDelegationWithMetrics(b *testing.B) {
	authzMem := authz.NewMemoryAuthorizer()
	authzMem.AddPolicy(authz.Policy{ID: "bench-create", Subject: "alice", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), authzMem, WithMetrics(imetrics.NewMemory()))
	dr, err := svc.CreateDelegation(DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, Duration: time.Minute})
	if err != nil {
		b.Fatalf("setup create: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := svc.ValidateDelegation(dr.POA.ID, "bob", "transaction:execute"); err != nil {
			b.Fatalf("validate: %v", err)
		}
	}
}

// BenchmarkSignCanonicalPOA isolates cost of canonicalization+signing (Ed25519) absent service overhead.
func BenchmarkSignCanonicalPOA(b *testing.B) {
	provider, err := crypto.NewInMemoryEd25519Provider()
	if err != nil {
		b.Fatalf("provider: %v", err)
	}
	signer, _ := provider.ActiveSigner()
	now := time.Now().UTC()
	poa := &PowerOfAttorney{ID: "bench", Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, ValidFrom: now, ValidUntil: now.Add(time.Hour), Status: POAStatusActive, CreatedAt: now, UpdatedAt: now}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d, canon, err := CanonicalPOADigest(poa)
		if err != nil {
			b.Fatalf("digest: %v", err)
		}
		sig, err := signer.Sign(canon)
		if err != nil {
			b.Fatalf("sign: %v", err)
		}
		if len(sig) == 0 || len(d) == 0 {
			b.Fatalf("unexpected empty results")
		}
	}
}

// BenchmarkVerifyCanonicalPOA isolates cost of canonicalization+verification (Ed25519).
func BenchmarkVerifyCanonicalPOA(b *testing.B) {
	provider, err := crypto.NewInMemoryEd25519Provider()
	if err != nil {
		b.Fatalf("provider: %v", err)
	}
	signer, _ := provider.ActiveSigner()
	now := time.Now().UTC()
	poa := &PowerOfAttorney{ID: "bench", Grantor: "alice", Grantee: "bob", Scope: []string{"transaction:execute"}, ValidFrom: now, ValidUntil: now.Add(time.Hour), Status: POAStatusActive, CreatedAt: now, UpdatedAt: now}
	digest, canon, err := CanonicalPOADigest(poa)
	if err != nil {
		b.Fatalf("digest: %v", err)
	}
	sig, err := signer.Sign(canon)
	if err != nil {
		b.Fatalf("sign: %v", err)
	}
	poa.Signature = &POASignature{Algorithm: signer.Algorithm(), KeyID: signer.KeyID(), DigestHex: digest, SigBase64: base64.StdEncoding.EncodeToString(sig), Canonical: canon}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d2, canon2, err := CanonicalPOADigest(poa)
		if err != nil {
			b.Fatalf("digest2: %v", err)
		}
		if d2 != poa.Signature.DigestHex {
			b.Fatalf("digest mismatch")
		}
		if err := provider.VerifyWith(canon2, sig, signer.KeyID()); err != nil {
			b.Fatalf("verify: %v", err)
		}
	}
}
