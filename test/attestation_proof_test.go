package test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/aap"
	"github.com/mauriciomferz/AgentAuth/pkg/aap/errs"
	"github.com/mauriciomferz/AgentAuth/pkg/attest"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	cr "github.com/mauriciomferz/AgentAuth/pkg/crypto"
	gauth_aap_001 "github.com/mauriciomferz/AgentAuth/pkg/gauth_aap_001"
)

// simpleAllowAuthorizer always allows
type simpleAllowAuthorizer struct{}

func (s simpleAllowAuthorizer) Authorize(ctx context.Context, r authz.Request) (authz.Decision, error) {
	return authz.Decision{Allow: true}, nil
}

func (s simpleAllowAuthorizer) GetPermissions(ctx context.Context, subject string) ([]authz.Permission, error) {
	// Return empty permissions list for test authorizer
	return []authz.Permission{}, nil
}

func newServiceWithSigner(t *testing.T) *gauth_aap_001.Service {
	kms, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("kms init: %v", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	svc := gauth_aap_001.NewService(auditLogger, simpleAllowAuthorizer{}, gauth_aap_001.WithKMS(kms))
	return svc
}

// newServiceWithSignerAndAnchors constructs a service with a supplied (possibly empty)
// trust anchor registry for attestation enforcement tests.
func newServiceWithSignerAndAnchors(t *testing.T, reg *attest.TrustAnchorRegistry, m metrics.Metrics) (*gauth_aap_001.Service, cr.KMS) {
	kms, err := cr.NewInMemoryEd25519Provider()
	if err != nil {
		t.Fatalf("kms init: %v", err)
	}
	auditLogger := audit.NewMemoryLogger(nil)
	opts := []gauth_aap_001.Option{gauth_aap_001.WithKMS(kms), gauth_aap_001.WithAttestationTrustAnchors(reg)}
	if m != nil {
		opts = append(opts, gauth_aap_001.WithMetrics(m))
	}
	svc := gauth_aap_001.NewService(auditLogger, simpleAllowAuthorizer{}, opts...)
	return svc, kms
}

// withAttestationEnforcement sets GAUTH_ATTEST_REQUIRE_TRUST_ANCHOR=1 for the duration
// of the test and restores previous value afterwards.
func withAttestationEnforcement(t *testing.T) func() {
	prev := os.Getenv("GAUTH_ATTEST_REQUIRE_TRUST_ANCHOR")
	if err := os.Setenv("GAUTH_ATTEST_REQUIRE_TRUST_ANCHOR", "1"); err != nil {
		t.Fatalf("failed to set env: %v", err)
	}
	return func() { _ = os.Setenv("GAUTH_ATTEST_REQUIRE_TRUST_ANCHOR", prev) }
}

func TestAttestationProofIssueAndVerify_Success(t *testing.T) {
	svc := newServiceWithSigner(t)
	p, err := svc.IssueAttestationProof(context.Background(), "subject has completed onboarding", "user123", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if p.DigestHex == "" || p.Signature == "" {
		t.Fatalf("missing digest/signature")
	}
	if err := svc.VerifyAttestationProof(context.Background(), p); err != nil {
		t.Fatalf("verify: %v", err)
	}
}

func TestAttestationProofDigestMismatch(t *testing.T) {
	svc := newServiceWithSigner(t)
	p, err := svc.IssueAttestationProof(context.Background(), "A", "user123", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	p.DigestHex = "deadbeef" // force mismatch
	if err := svc.VerifyAttestationProof(context.Background(), p); err == nil {
		t.Fatalf("expected digest mismatch error")
	}
}

func TestAttestationProofSignatureFailureUnknownKey(t *testing.T) {
	svc := newServiceWithSigner(t)
	p, err := svc.IssueAttestationProof(context.Background(), "B", "user123", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	// Replace key id with unknown one to force public key missing
	p.KeyID = "ffffffffffff"
	if err := svc.VerifyAttestationProof(context.Background(), p); err == nil {
		t.Fatalf("expected public key missing error")
	}
}

func TestAttestationProofExpiry(t *testing.T) {
	svc := newServiceWithSigner(t)
	p, err := svc.IssueAttestationProof(context.Background(), "C", "user123", time.Millisecond)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	time.Sleep(2 * time.Millisecond)
	err = svc.VerifyAttestationProof(context.Background(), p)
	if err == nil {
		t.Fatalf("expected expired error, got nil")
	}
	if e, ok := err.(errs.RFCError); !ok || e.Code != aap.ErrExpired {
		t.Fatalf("expected expired RFC error, got %v", err)
	}
}

// Benchmark canonical digest path for attestation proofs (micro performance signal)
func BenchmarkCanonicalAttestationDigest(b *testing.B) {
	p := &attest.AttestationProof{Version: "att/v1", Statement: "bench", Subject: "user123", Issuer: "user123", IssuedAt: time.Now().UTC()}
	for i := 0; i < b.N; i++ {
		if _, _, err := attest.CanonicalAttestationDigest(p); err != nil {
			b.Fatalf("err: %v", err)
		}
	}
}

func TestAttestationProofTrustAnchorMissingEnforced(t *testing.T) {
	reg := attest.NewTrustAnchorRegistry() // empty
	memory := metrics.NewMemory()
	svc, _ := newServiceWithSignerAndAnchors(t, reg, memory)
	cleanup := withAttestationEnforcement(t)
	defer cleanup()
	p, err := svc.IssueAttestationProof(context.Background(), "onboard", "issuerA", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if err := svc.VerifyAttestationProof(context.Background(), p); err == nil {
		t.Fatalf("expected trust anchor missing failure")
	}
	// Verify granular metric increment
	ss := memory.SnapshotEx()
	if ss.AttestationProofTrustAnchorMissing == 0 {
		t.Fatalf("expected trust anchor missing counter increment (got %d)", ss.AttestationProofTrustAnchorMissing)
	}
}

func TestAttestationProofTrustAnchorAlgorithmMismatch(t *testing.T) {
	reg := attest.NewTrustAnchorRegistry()
	memory := metrics.NewMemory()
	svc, _ := newServiceWithSignerAndAnchors(t, reg, memory)
	cleanup := withAttestationEnforcement(t)
	defer cleanup()
	p, err := svc.IssueAttestationProof(context.Background(), "stmt", "issuerB", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	reg.Add(attest.TrustAnchor{Issuer: p.Issuer, Algorithm: "ecdsa-p256", KeyID: p.KeyID})
	if err := svc.VerifyAttestationProof(context.Background(), p); err == nil {
		t.Fatalf("expected algorithm mismatch failure")
	}
	ss := memory.SnapshotEx()
	if ss.AttestationProofTrustAnchorAlgorithmMismatch == 0 {
		t.Fatalf("expected algorithm mismatch counter increment (got %d)", ss.AttestationProofTrustAnchorAlgorithmMismatch)
	}
}

func TestAttestationProofTrustAnchorKeyMismatch(t *testing.T) {
	reg := attest.NewTrustAnchorRegistry()
	memory := metrics.NewMemory()
	svc, _ := newServiceWithSignerAndAnchors(t, reg, memory)
	cleanup := withAttestationEnforcement(t)
	defer cleanup()
	p, err := svc.IssueAttestationProof(context.Background(), "stmt2", "issuerC", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	reg.Add(attest.TrustAnchor{Issuer: p.Issuer, Algorithm: p.Algorithm, KeyID: "wrongkey"})
	if err := svc.VerifyAttestationProof(context.Background(), p); err == nil {
		t.Fatalf("expected key mismatch failure")
	}
	ss := memory.SnapshotEx()
	if ss.AttestationProofTrustAnchorKeyMismatch == 0 {
		t.Fatalf("expected key mismatch counter increment (got %d)", ss.AttestationProofTrustAnchorKeyMismatch)
	}
}

func TestAttestationProofTrustAnchorSuccess(t *testing.T) {
	reg := attest.NewTrustAnchorRegistry()
	memory := metrics.NewMemory()
	svc, _ := newServiceWithSignerAndAnchors(t, reg, memory)
	cleanup := withAttestationEnforcement(t)
	defer cleanup()
	p, err := svc.IssueAttestationProof(context.Background(), "stmt3", "issuerD", time.Minute)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	reg.Add(attest.TrustAnchor{Issuer: p.Issuer, Algorithm: p.Algorithm, KeyID: p.KeyID})
	if err := svc.VerifyAttestationProof(context.Background(), p); err != nil {
		t.Fatalf("expected success with matching anchor, got: %v", err)
	}
	ss := memory.SnapshotEx()
	// Ensure no trust anchor failure counters incremented
	if ss.AttestationProofTrustAnchorMissing+ss.AttestationProofTrustAnchorAlgorithmMismatch+ss.AttestationProofTrustAnchorKeyMismatch != 0 {
		t.Fatalf("expected zero trust anchor failure counters on success, got missing=%d algo=%d key=%d", ss.AttestationProofTrustAnchorMissing, ss.AttestationProofTrustAnchorAlgorithmMismatch, ss.AttestationProofTrustAnchorKeyMismatch)
	}
}
