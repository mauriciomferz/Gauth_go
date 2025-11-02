package test

import (
	"crypto/ed25519"
	"testing"
	"time"

	internalCrypto "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
)

// TestSignatureAgilityBackwardCompat ensures RB6 interface-based signing produces identical
// bytes to direct ed25519.Sign using the underlying active key, preserving digest stability.
func TestSignatureAgilityBackwardCompat(t *testing.T) {
    m, err := internalCrypto.NewManager(12 * time.Hour)
    if err != nil { t.Fatalf("manager init: %v", err) }
    internalCrypto.RegisterGlobalEdDSAManager(m)
    active := m.Active()
    if active == nil || len(active.Private) != ed25519.PrivateKeySize { t.Fatalf("no active key") }
    msg := []byte("rb6-agility-test-message-1234")
    sigDirect := ed25519.Sign(active.Private, msg)
    sigGlobal, err := internalCrypto.SignWithGlobal(msg)
    if err != nil { t.Fatalf("SignWithGlobal error: %v", err) }
    if len(sigDirect) != len(sigGlobal) { t.Fatalf("size mismatch direct=%d global=%d", len(sigDirect), len(sigGlobal)) }
    if string(sigDirect) != string(sigGlobal) { t.Fatalf("signature bytes differ; agility layer changed digest invariants") }
    // Verify helper path
    if !internalCrypto.VerifyWithGlobal(msg, sigDirect) { t.Fatalf("VerifyWithGlobal failed for direct signature") }
}
