package agentauth_aap_001

import (
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

// fake signer / key provider for testing rotation of signature public keys.
// We reuse the real in-memory ed25519 key provider from crypto package for authenticity.

// TestRotationRobustness ensures that after rotating token symmetric keys, existing tokens still validate
// (handled in decryptWithAnyKey) and that removal of a signing public key increments the signature_public_key_missing metric.
func TestRotationRobustness(t *testing.T) {
	// Disable strict authenticity for soft skip expectation.
	os.Setenv("AGENTAUTH_STRICT_AUTHENTICITY", "0")
	defer os.Unsetenv("AGENTAUTH_STRICT_AUTHENTICITY")
	memMetrics := metrics.NewMemory()
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	// allow grantor create
	authorizer.AddPolicy(authz.Policy{ID: "p1", Subject: "alice", Resource: "*", Actions: []string{"create_delegation"}, Effect: authz.Allow})
	authorizer.AddPolicy(authz.Policy{ID: "p2", Subject: "alice", Resource: "*", Actions: []string{"revoke_delegation"}, Effect: authz.Allow})

	// Simple signer provider (not real crypto) – just testing missing key path logic.
	// No signerProvider: focus purely on token key rotation & missing public key metric simulation logic.
	svc := NewService(auditLogger, authorizer, WithMetrics(memMetrics))

	// Create delegation (signature issued)
	resp, err := svc.CreateDelegationCtx(context.Background(), DelegationRequest{Grantor: "alice", Grantee: "bob", Scope: []string{"read"}, Duration: time.Minute})
	if err != nil {
		t.Fatalf("create delegation: %v", err)
	}
	// Signature is nil (no signer); the first two validations should not increment missing public key.

	// First validation should succeed; missing public key metric should remain zero.
	if err := svc.ValidateDelegationCtx(context.Background(), resp.POA.ID, "bob", "read"); err != nil {
		t.Fatalf("validate: %v", err)
	}
	snap := memMetrics.SnapshotEx()
	if snap.SignaturePublicKeyMissing != 0 {
		t.Fatalf("expected missing public key metric 0 got %d", snap.SignaturePublicKeyMissing)
	}

	// Simulate rotation (new active key) so previous key list contains old active (with signer kid) to ensure still found.
	svc.keyRing.Rotate()
	// Keep signer key id on first previous key
	if len(svc.keyRing.Previous()) == 0 {
		t.Fatalf("expected previous key after rotation")
	}
	if err := svc.ValidateDelegationCtx(context.Background(), resp.POA.ID, "bob", "read"); err != nil {
		t.Fatalf("validate after rotation: %v", err)
	}
	snap = memMetrics.SnapshotEx()
	if snap.SignaturePublicKeyMissing != 0 {
		t.Fatalf("expected missing public key metric still 0 got %d", snap.SignaturePublicKeyMissing)
	}

	// Simulate key removal (clear previous keys) -> should trigger missing key path.
	// NOTE: keyRing does not expose mutation; for test we directly set previous slice to empty via reflection impossible here.
	// Instead, change signer.kid to an unknown ID to force lookup miss.
	// Simulate a signed POA with unknown key id to trigger missing key metric (digest must match to avoid early integrity failure)
	stored, ok := svc.repo.Get(resp.POA.ID)
	if !ok || stored == nil {
		t.Fatalf("could not retrieve stored poa")
	}
	dig, _, derr := CanonicalPOADigest(stored)
	if derr != nil {
		t.Fatalf("canonical digest: %v", derr)
	}
	// Provide a 64-byte base64 signature placeholder (all zeros) to satisfy size checks without real verification.
	zeroSig := make([]byte, 64)
	stored.Signature = &POASignature{Algorithm: algEd25519, KeyID: "unknown_kid", DigestHex: dig, SigBase64: base64.StdEncoding.EncodeToString(zeroSig)}
	if err := svc.ValidateDelegationCtx(context.Background(), resp.POA.ID, "bob", "read"); err != nil {
		t.Fatalf("validation after injecting unknown key signature should soft skip, got error: %v", err)
	}
	snap = memMetrics.SnapshotEx()
	if snap.SignaturePublicKeyMissing == 0 {
		t.Fatalf("expected missing public key metric >0 after forced unknown key validation")
	}
}
