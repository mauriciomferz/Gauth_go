package compliance

import (
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"
)

func TestDisputeRegistry_Persistence(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "disputes.json")

	// 1. Create registry with persistence
	dr1, err := NewDisputeRegistry(path)
	if err != nil {
		t.Fatalf("failed to create registry 1: %v", err)
	}

	meta := DisputeMetadata{
		ID:           "persist-1",
		FlowID:       "flow-persist",
		Jurisdiction: JurisdictionUS,
		Entity:       EntityTypeIndividual,
		Reason:       "verification failed",
		Status:       "pending",
	}

	if err := dr1.EscalateDispute(meta); err != nil {
		t.Fatalf("failed to escalate: %v", err)
	}

	// 2. Create new registry pointing to same file (Simulate restart)
	dr2, err := NewDisputeRegistry(path)
	if err != nil {
		t.Fatalf("failed to create registry 2: %v", err)
	}

	// 3. Verify dispute loaded
	d, ok := dr2.GetDispute("persist-1")
	if !ok {
		t.Fatal("expected dispute to be persisted and loaded")
	}
	if d.FlowID != "flow-persist" {
		t.Errorf("expected flow ID flow-persist, got %s", d.FlowID)
	}
}

func TestAttestation_Verification(t *testing.T) {
	// Generate keys
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate keys: %v", err)
	}

	signerID := "signer-1"
	signer := NewEd25519Signer(signerID, priv)
	verifier := NewDefaultAttestationVerifier()
	verifier.RegisterKey(signerID, pub)

	// Create attestation
	ts := time.Now().Unix()
	sig, err := signer.Sign("att-1", "flow-1", ts)
	if err != nil {
		t.Fatalf("sign failed: %v", err)
	}

	att := Attestation{
		ID:        "att-1",
		FlowID:    "flow-1",
		SignerID:  signerID,
		Timestamp: ts,
		Proof:     sig,
	}

	// Verify
	valid, err := verifier.Verify(att)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if !valid {
		t.Error("expected valid signature")
	}

	// Tamper check
	att.FlowID = "flow-tampered"
	valid, err = verifier.Verify(att)
	if valid || err == nil {
		t.Error("expected invalid signature check for tampered data")
	}
}
