package rfc0111

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/sunset"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/audit"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
)

const testDeadbeefCafebabe = "deadbeefcafebabe"

// TestDigestMismatchReasonHeuristics simulates domain_conflict vs tamper_suspected classification.
func TestDigestMismatchReasonHeuristics(t *testing.T) {
	mem := metrics.NewMemory()
	ma := authz.NewMemoryAuthorizer()
	ma.AddPolicy(authz.Policy{ID: "allow_create", Subject: "*", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), ma, WithMetrics(mem))
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	// Issue delegation
	req := DelegationRequest{Grantor: "r@example.com", Grantee: "s@example.com", Scope: []string{"x"}, Duration: time.Minute}
	resp, err := svc.CreateDelegationCtx(context.Background(), req)
	if err != nil {
		t.Fatalf("create delegation failed: %v", err)
	}
	// Tamper digest length -> tamper_suspected
	poa, _ := svc.repo.Get(resp.POA.ID)
	// Ensure signature exists (service may not have signer provider configured in this test)
	if poa.Signature == nil {
		// Generate ephemeral ed25519 key and sign canonical digest
		pub, priv, _ := ed25519.GenerateKey(rand.Reader)
		_ = pub // unused; kept for clarity
		dig, canon, derr := CanonicalPOADigest(poa)
		if derr != nil {
			t.Fatalf("canonical digest failed: %v", derr)
		}
		sig := ed25519.Sign(priv, canon)
		poa.Signature = &POASignature{Algorithm: algEd25519, KeyID: "test", DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(sig), Canonical: canon}
	}
	poa.Signature.DigestHex = testDeadbeefCafebabe // change length (likely different from canonical length)
	_, _ = svc.VerifyToken(context.Background(), resp.AuthToken)
	reasons := mem.EnvelopeDigestMismatchReasonsSnapshot()
	if reasons["tamper_suspected"] == 0 {
		t.Fatalf("expected tamper_suspected mismatch counter >0")
	}
	// Reset digest to same-length different value -> domain_conflict
	if dig, _, derr := CanonicalPOADigest(poa); derr == nil {
		// produce different same-length digest by altering first char if possible
		if len(dig) > 0 {
			alt := dig
			if alt[0] == 'a' {
				alt = "b" + alt[1:]
			} else {
				alt = "a" + alt[1:]
			}
			poa.Signature.DigestHex = alt
		}
	}
	_, _ = svc.VerifyToken(context.Background(), resp.AuthToken)
	reasons2 := mem.EnvelopeDigestMismatchReasonsSnapshot()
	if reasons2["domain_conflict"] == 0 {
		t.Fatalf("expected domain_conflict mismatch counter >0")
	}
}

// TestSunsetControllerPromotion ensures controller advances phase when adoption & mismatch criteria satisfied over window.
func TestSunsetControllerPromotion(t *testing.T) {
	mem := metrics.NewMemory()
	ma := authz.NewMemoryAuthorizer()
	ma.AddPolicy(authz.Policy{ID: "allow_create", Subject: "*", Resource: "poa", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	svc := NewService(audit.NewMemoryLogger(nil), ma, WithMetrics(mem))
	// Seed initial phase Pilot (1)
	mem.SetEnvelopeV1SunsetPhase(1)
	// Simulate high adoption: issue V2 tokens only
	os.Setenv("GAUTH_POA_ENVELOPE_V2", "1")
	for i := 0; i < 10; i++ {
		req := DelegationRequest{Grantor: "p@example.com", Grantee: "q@example.com", Scope: []string{"x"}, Duration: time.Minute}
		if _, err := svc.CreateDelegationCtx(context.Background(), req); err != nil {
			t.Fatalf("issue v2 failed: %v", err)
		}
	}
	// Manually set adoption ratio near 1.0 (already done via issuance) and ensure no mismatches
	cfg := sunset.ControllerConfig{Enable: true, Interval: 50 * time.Millisecond, Window: 200 * time.Millisecond, PilotToBroadAdoption: 0.6, MaxMismatchRatio: 0.01}
	ctrl := sunset.NewController(cfg, sunset.MemoryMetricsView{M: mem})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	go ctrl.Start(ctx)
	// Wait for expected promotion
	timeout := time.After(2 * time.Second)
	for {
		select {
		case <-timeout:
			t.Fatalf("phase promotion timeout (current=%d)", mem.EnvelopeV1SunsetPhase())
		default:
			if mem.EnvelopeV1SunsetPhase() >= 2 { // Broad phase
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
	}
}
