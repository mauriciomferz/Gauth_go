package multisig

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/gauth_rfc_001"
)

// mockVerifier implements VerificationProvider for testing
type mockVerifier struct {
	keys map[string]ed25519.PublicKey
}

func (m *mockVerifier) PublicKey(keyID string) ([]byte, string, error) {
	pub, ok := m.keys[keyID]
	if !ok {
		return nil, "", errors.New("key not found")
	}
	return []byte(pub), "ed25519", nil
}

func (m *mockVerifier) VerifySignature(digest []byte, signature []byte, publicKey []byte) error {
	if len(publicKey) != ed25519.PublicKeySize {
		return errors.New("invalid public key size")
	}
	if len(signature) != ed25519.SignatureSize {
		return errors.New("invalid signature size")
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), digest, signature) {
		return errors.New("signature verification failed")
	}
	return nil
}

// generateTestKey creates an Ed25519 keypair for testing
func generateTestKey(t *testing.T) (ed25519.PrivateKey, ed25519.PublicKey, string) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	keyID := "key-" + hex.EncodeToString(pub[:8])
	return priv, pub, keyID
}

// signDigest signs a digest with a private key
func signDigest(t *testing.T, digest string, priv ed25519.PrivateKey) string {
	digestBytes, err := hex.DecodeString(digest)
	if err != nil {
		t.Fatalf("Failed to decode digest: %v", err)
	}
	signature := ed25519.Sign(priv, digestBytes)
	return base64.StdEncoding.EncodeToString(signature)
}

func TestSignatureManager_InitiateCollection(t *testing.T) {
	verifier := &mockVerifier{keys: make(map[string]ed25519.PublicKey)}
	manager := NewSignatureManager(verifier)

	poa := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-1",
		Grantor:   "alice",
		Grantee:   "agent",
		Threshold: 2,
		Signers:   []string{"alice", "bob", "carol"},
	}

	ctx := context.Background()
	err := manager.InitiateCollection(ctx, poa, 24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to initiate collection: %v", err)
	}

	// Check state was created
	state, err := manager.GetStatus(ctx, "poa-1")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}

	if state.PoAID != "poa-1" {
		t.Errorf("Expected PoAID poa-1, got %s", state.PoAID)
	}
	if state.Threshold != 2 {
		t.Errorf("Expected threshold 2, got %d", state.Threshold)
	}
	if len(state.RequiredSigners) != 3 {
		t.Errorf("Expected 3 required signers, got %d", len(state.RequiredSigners))
	}
	if state.Status != StatusPending {
		t.Errorf("Expected status pending, got %s", state.Status)
	}
	if state.ExpiresAt == nil {
		t.Error("Expected expiration time to be set")
	}
}

func TestSignatureManager_SubmitSignature(t *testing.T) {
	privA, pubA, keyIDA := generateTestKey(t)
	privB, pubB, keyIDB := generateTestKey(t)
	_, pubC, keyIDC := generateTestKey(t)

	verifier := &mockVerifier{
		keys: map[string]ed25519.PublicKey{
			keyIDA: pubA,
			keyIDB: pubB,
			keyIDC: pubC,
		},
	}
	manager := NewSignatureManager(verifier)

	poa := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-2",
		Grantor:   "alice",
		Grantee:   "agent",
		Threshold: 2,
		Signers:   []string{"alice", "bob", "carol"},
	}

	ctx := context.Background()
	if err := manager.InitiateCollection(ctx, poa, 1*time.Hour); err != nil {
		t.Fatalf("Failed to initiate: %v", err)
	}

	// Get canonical digest
	state, _ := manager.GetStatus(ctx, "poa-2")
	digest := state.CanonicalDigest

	// Submit first signature
	sig1 := signDigest(t, digest, privA)
	err := manager.SubmitSignature(ctx, "poa-2", "alice", keyIDA, sig1, nil)
	if err != nil {
		t.Fatalf("Failed to submit signature 1: %v", err)
	}

	// Check status - should still be pending (1/2)
	state, _ = manager.GetStatus(ctx, "poa-2")
	if state.Status != StatusPending {
		t.Errorf("Expected pending status after 1 signature, got %s", state.Status)
	}
	if len(state.Signatures) != 1 {
		t.Errorf("Expected 1 signature, got %d", len(state.Signatures))
	}

	// Submit second signature
	sig2 := signDigest(t, digest, privB)
	err = manager.SubmitSignature(ctx, "poa-2", "bob", keyIDB, sig2, nil)
	if err != nil {
		t.Fatalf("Failed to submit signature 2: %v", err)
	}

	// Check status - should be completed (2/2)
	state, _ = manager.GetStatus(ctx, "poa-2")
	if state.Status != StatusCompleted {
		t.Errorf("Expected completed status after 2 signatures, got %s", state.Status)
	}
	if len(state.Signatures) != 2 {
		t.Errorf("Expected 2 signatures, got %d", len(state.Signatures))
	}
	if state.CompletedAt == nil {
		t.Error("Expected completion time to be set")
	}
}

func TestSignatureManager_DuplicateSignature(t *testing.T) {
	privA, pubA, keyIDA := generateTestKey(t)

	verifier := &mockVerifier{
		keys: map[string]ed25519.PublicKey{
			keyIDA: pubA,
		},
	}
	manager := NewSignatureManager(verifier)

	poa := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-3",
		Grantor:   "alice",
		Grantee:   "agent",
		Threshold: 2,
		Signers:   []string{"alice", "bob"},
	}

	ctx := context.Background()
	if err := manager.InitiateCollection(ctx, poa, 1*time.Hour); err != nil {
		t.Fatalf("InitiateCollection failed: %v", err)
	}

	state, _ := manager.GetStatus(ctx, "poa-3")
	digest := state.CanonicalDigest
	sig := signDigest(t, digest, privA)

	// Submit first time - should succeed
	err := manager.SubmitSignature(ctx, "poa-3", "alice", keyIDA, sig, nil)
	if err != nil {
		t.Fatalf("First submission failed: %v", err)
	}

	// Submit second time - should fail with duplicate error
	err = manager.SubmitSignature(ctx, "poa-3", "alice", keyIDA, sig, nil)
	if err == nil {
		t.Fatal("Expected error for duplicate signature")
	}
	if !contains(err.Error(), "already submitted") {
		t.Errorf("Expected 'already submitted' error, got: %v", err)
	}
}

func TestSignatureManager_InvalidSignature(t *testing.T) {
	_, pubA, keyIDA := generateTestKey(t)
	privWrong, _, _ := generateTestKey(t)

	verifier := &mockVerifier{
		keys: map[string]ed25519.PublicKey{
			keyIDA: pubA,
		},
	}
	manager := NewSignatureManager(verifier)

	poa := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-4",
		Grantor:   "alice",
		Grantee:   "agent",
		Threshold: 2,
		Signers:   []string{"alice", "bob"},
	}

	ctx := context.Background()
	if err := manager.InitiateCollection(ctx, poa, 1*time.Hour); err != nil {
		t.Fatalf("InitiateCollection failed: %v", err)
	}

	state, _ := manager.GetStatus(ctx, "poa-4")
	digest := state.CanonicalDigest

	// Sign with wrong key
	wrongSig := signDigest(t, digest, privWrong)

	err := manager.SubmitSignature(ctx, "poa-4", "alice", keyIDA, wrongSig, nil)
	if err == nil {
		t.Fatal("Expected signature verification to fail")
	}
	if !contains(err.Error(), "verification failed") {
		t.Errorf("Expected 'verification failed' error, got: %v", err)
	}
}

func TestSignatureManager_UnauthorizedSigner(t *testing.T) {
	privX, pubX, keyIDX := generateTestKey(t)

	verifier := &mockVerifier{
		keys: map[string]ed25519.PublicKey{
			keyIDX: pubX,
		},
	}
	manager := NewSignatureManager(verifier)

	poa := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-5",
		Grantor:   "alice",
		Grantee:   "agent",
		Threshold: 2,
		Signers:   []string{"alice", "bob", "carol"},
	}

	ctx := context.Background()
	if err := manager.InitiateCollection(ctx, poa, 1*time.Hour); err != nil {
		t.Fatalf("InitiateCollection failed: %v", err)
	}

	state, _ := manager.GetStatus(ctx, "poa-5")
	digest := state.CanonicalDigest
	sig := signDigest(t, digest, privX)

	// Try to submit signature from unauthorized signer
	err := manager.SubmitSignature(ctx, "poa-5", "charlie", keyIDX, sig, nil)
	if err == nil {
		t.Fatal("Expected error for unauthorized signer")
	}
	if !contains(err.Error(), "not in required signers") {
		t.Errorf("Expected 'not in required signers' error, got: %v", err)
	}
}

func TestSignatureManager_ActivatePoA(t *testing.T) {
	privA, pubA, keyIDA := generateTestKey(t)
	privB, pubB, keyIDB := generateTestKey(t)

	verifier := &mockVerifier{
		keys: map[string]ed25519.PublicKey{
			keyIDA: pubA,
			keyIDB: pubB,
		},
	}
	manager := NewSignatureManager(verifier)

	poa := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-6",
		Grantor:   "alice",
		Grantee:   "agent",
		Threshold: 2,
		Signers:   []string{"alice", "bob"},
	}

	ctx := context.Background()
	if err := manager.InitiateCollection(ctx, poa, 1*time.Hour); err != nil {
		t.Fatalf("Failed to initiate collection: %v", err)
	}

	// Try to activate before threshold met - should fail
	err := manager.ActivatePoA(ctx, "poa-6")
	if err == nil {
		t.Fatal("Expected error activating before threshold")
	}

	// Submit signatures to meet threshold
	state, _ := manager.GetStatus(ctx, "poa-6")
	digest := state.CanonicalDigest

	sig1 := signDigest(t, digest, privA)
	if err2 := manager.SubmitSignature(ctx, "poa-6", "alice", keyIDA, sig1, nil); err2 != nil {
		t.Fatalf("SubmitSignature failed: %v", err)
	}

	sig2 := signDigest(t, digest, privB)
	if err2 := manager.SubmitSignature(ctx, "poa-6", "bob", keyIDB, sig2, nil); err2 != nil {
		t.Fatalf("SubmitSignature failed: %v", err)
	}

	// Now activation should succeed
	err = manager.ActivatePoA(ctx, "poa-6")
	if err != nil {
		t.Fatalf("Failed to activate PoA: %v", err)
	}

	// Check status is active
	state, _ = manager.GetStatus(ctx, "poa-6")
	if state.Status != StatusActive {
		t.Errorf("Expected active status, got %s", state.Status)
	}
	if state.ActivatedAt == nil {
		t.Error("Expected activation time to be set")
	}
}

func TestSignatureManager_Expiration(t *testing.T) {
	privA, pubA, keyIDA := generateTestKey(t)

	verifier := &mockVerifier{
		keys: map[string]ed25519.PublicKey{
			keyIDA: pubA,
		},
	}
	manager := NewSignatureManager(verifier)

	poa := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-7",
		Grantor:   "alice",
		Grantee:   "agent",
		Threshold: 2,
		Signers:   []string{"alice", "bob"},
	}

	ctx := context.Background()
	// Set very short expiration
	if err := manager.InitiateCollection(ctx, poa, 1*time.Millisecond); err != nil {
		t.Fatalf("Failed to initiate collection: %v", err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	state, _ := manager.GetStatus(ctx, "poa-7")
	digest := state.CanonicalDigest
	sig := signDigest(t, digest, privA)

	// Try to submit after expiration
	err := manager.SubmitSignature(ctx, "poa-7", "alice", keyIDA, sig, nil)
	if err == nil {
		t.Fatal("Expected error for expired collection")
	}
	if !contains(err.Error(), "expired") {
		t.Errorf("Expected 'expired' error, got: %v", err)
	}

	// Check status
	state, _ = manager.GetStatus(ctx, "poa-7")
	if state.Status != StatusExpired {
		t.Errorf("Expected expired status, got %s", state.Status)
	}
}

func TestSignatureManager_GetSignatures(t *testing.T) {
	privA, pubA, keyIDA := generateTestKey(t)
	privB, pubB, keyIDB := generateTestKey(t)

	verifier := &mockVerifier{
		keys: map[string]ed25519.PublicKey{
			keyIDA: pubA,
			keyIDB: pubB,
		},
	}
	manager := NewSignatureManager(verifier)

	poa := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-8",
		Grantor:   "alice",
		Grantee:   "agent",
		Threshold: 2,
		Signers:   []string{"alice", "bob"},
	}

	ctx := context.Background()
	if err := manager.InitiateCollection(ctx, poa, 1*time.Hour); err != nil {
		t.Fatalf("Failed to initiate collection: %v", err)
	}

	state, _ := manager.GetStatus(ctx, "poa-8")
	digest := state.CanonicalDigest

	sig1 := signDigest(t, digest, privA)
	if err := manager.SubmitSignature(ctx, "poa-8", "alice", keyIDA, sig1, nil); err != nil {
		t.Fatalf("SubmitSignature failed: %v", err)
	}

	sig2 := signDigest(t, digest, privB)
	if err := manager.SubmitSignature(ctx, "poa-8", "bob", keyIDB, sig2, nil); err != nil {
		t.Fatalf("SubmitSignature failed: %v", err)
	}

	// Get signatures in AAP001 format
	signatures, err := manager.GetSignatures(ctx, "poa-8")
	if err != nil {
		t.Fatalf("Failed to get signatures: %v", err)
	}

	if len(signatures) != 2 {
		t.Errorf("Expected 2 signatures, got %d", len(signatures))
	}

	// Verify alice signature
	if sig, ok := signatures["alice"]; !ok {
		t.Error("Expected alice signature")
	} else {
		if sig.Algorithm != "ed25519" {
			t.Errorf("Expected algorithm ed25519, got %s", sig.Algorithm)
		}
		if sig.KeyID != keyIDA {
			t.Errorf("Expected keyID %s, got %s", keyIDA, sig.KeyID)
		}
		if sig.SigBase64 != sig1 {
			t.Error("Signature mismatch")
		}
	}

	// Verify bob signature
	if sig, ok := signatures["bob"]; !ok {
		t.Error("Expected bob signature")
	} else if sig.KeyID != keyIDB {
		t.Errorf("Expected keyID %s, got %s", keyIDB, sig.KeyID)
	}
}

func TestSignatureManager_ListPending(t *testing.T) {
	verifier := &mockVerifier{keys: make(map[string]ed25519.PublicKey)}
	manager := NewSignatureManager(verifier)

	ctx := context.Background()

	// Create multiple PoAs
	poa1 := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-pending-1",
		Threshold: 2,
		Signers:   []string{"alice", "bob"},
	}
	poa2 := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-pending-2",
		Threshold: 2,
		Signers:   []string{"carol", "dave"},
	}

	_ = manager.InitiateCollection(ctx, poa1, 1*time.Hour)
	_ = manager.InitiateCollection(ctx, poa2, 1*time.Hour)

	pending := manager.ListPending(ctx)
	if len(pending) != 2 {
		t.Errorf("Expected 2 pending, got %d", len(pending))
	}
}

func TestSignatureManager_RejectCollection(t *testing.T) {
	verifier := &mockVerifier{keys: make(map[string]ed25519.PublicKey)}
	manager := NewSignatureManager(verifier)

	poa := &gauth_rfc_001.PowerOfAttorney{
		ID:        "poa-reject",
		Threshold: 2,
		Signers:   []string{"alice", "bob"},
	}

	ctx := context.Background()
	_ = manager.InitiateCollection(ctx, poa, 1*time.Hour)

	// Reject the collection
	err := manager.RejectCollection(ctx, "poa-reject", "PoA withdrawn")
	if err != nil {
		t.Fatalf("Failed to reject collection: %v", err)
	}

	// Check status
	state, _ := manager.GetStatus(ctx, "poa-reject")
	if state.Status != StatusRejected {
		t.Errorf("Expected rejected status, got %s", state.Status)
	}
}
