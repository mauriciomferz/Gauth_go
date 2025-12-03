package poa

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"testing"
	"time"

	internalCrypto "github.com/mauriciomferz/Gauth_go/internal/crypto"
)

func TestMemoryService_WithKeyManager(t *testing.T) {
	// Create a key manager
	km, err := internalCrypto.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	activeKey := km.Active()
	if activeKey == nil {
		t.Fatal("manager has no active key")
	}
	keyID := activeKey.ID

	// Create service with injected manager
	svc := NewMemoryService(WithKeyManager(km))

	// Verify manager is set
	if svc.keyManager != km {
		t.Error("key manager not injected correctly")
	}

	// Issue a POA (should use injected manager)
	// We need to set env vars to trigger multisig logic if we want to test that path
	t.Setenv("GAUTH_POA_MULTISIG_KIDS", keyID)
	t.Setenv("GAUTH_POA_MULTISIG_SIGN", "1")

	req := &Request{
		Subject:  "alice",
		Resource: "data",
		Action:   "read",
	}

	poa, err := svc.Issue(context.Background(), req)
	if err != nil {
		t.Fatalf("Issue failed: %v", err)
	}

	// Verify signature using the manager
	valid, satisfied, threshold := VerifyMultiSig(poa, km)
	if valid != 1 {
		t.Errorf("expected 1 valid signature, got %d", valid)
	}
	if !satisfied {
		t.Error("expected threshold satisfied")
	}
	if threshold != 1 {
		t.Errorf("expected threshold 1, got %d", threshold)
	}
}

func TestVerifyMultiSig_WithInjectedManager(t *testing.T) {
	// Create a key manager
	km, err := internalCrypto.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	activeKey := km.Active()
	if activeKey == nil {
		t.Fatal("manager has no active key")
	}
	keyID := activeKey.ID
	priv := activeKey.Private

	// Create a POA manually
	poa := &ProofOfAuthorization{
		ID:         "test-poa",
		SignerKids: []string{keyID},
		Threshold:  1,
		Signatures: []string{},
	}

	// Sign it
	msg := buildPoASigningPayload(poa)
	sig := ed25519.Sign(priv, msg)
	poa.Signatures = append(poa.Signatures, base64.RawStdEncoding.EncodeToString(sig))

	// Verify with manager
	valid, satisfied, threshold := VerifyMultiSig(poa, km)
	if valid != 1 {
		t.Errorf("expected 1 valid signature, got %d", valid)
	}
	if !satisfied {
		t.Error("expected threshold satisfied")
	}
	if threshold != 1 {
		t.Errorf("expected threshold 1, got %d", threshold)
	}

	// Verify with nil manager (should fail or fallback to global if set)
	// Ensure global is nil for this test
	oldGlobal := internalCrypto.GlobalEdDSARegistry
	internalCrypto.GlobalEdDSARegistry = nil
	defer func() { internalCrypto.GlobalEdDSARegistry = oldGlobal }()

	valid, satisfied, _ = VerifyMultiSig(poa, nil)
	if valid != 0 {
		t.Errorf("expected 0 valid signatures with nil manager, got %d", valid)
	}
}
