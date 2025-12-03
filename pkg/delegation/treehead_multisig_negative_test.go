package delegation

import (
	"os"
	"testing"
	"time"

	cryptoInt "github.com/mauriciomferz/Gauth_go/internal/crypto"
)

// TestMultiSignatureTreeHeadUnknownKid ensures verification fails when a signature references a kid
// not present in the key manager (JWKS / registry). This simulates a client verifying an STH where one
// signature cannot be cryptographically validated, forcing overall multi-sig failure even if threshold
// weight would otherwise be met.
func TestMultiSignatureTreeHeadUnknownKid(t *testing.T) {
	km, err := cryptoInt.NewManager(24 * time.Hour)
	if err != nil {
		t.Fatalf("manager init: %v", err)
	}
	// Create two additional keys so we have at least 3 signers.
	if _, err2 := km.Rotate(); err2 != nil {
		t.Fatalf("rotate1: %v", err2)
	}
	if _, err2 := km.Rotate(); err2 != nil {
		t.Fatalf("rotate2: %v", err2)
	}
	cryptoInt.GlobalEdDSARegistry = km
	// Threshold requires 2 signatures (count fallback) – will be met initially.
	os.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "2")
	chain := NewRevocationChain()
	if _, err2 := chain.Append(RevocationEvent{ID: "rev-neg-1", DelegationID: "del-neg"}); err2 != nil {
		t.Fatalf("append: %v", err2)
	}
	sth, err := chain.SignTreeHead()
	if err != nil {
		t.Fatalf("sign tree head: %v", err)
	}
	if len(sth.Signatures) < 2 {
		t.Fatalf("need at least two signatures to run negative test")
	}
	// Introduce unknown kid by modifying first signature's Kid to a value not in manager.
	sth.Signatures[0].Kid = "unknown-kid-xyz"
	// Attempt multi-sig verification – should fail at signature validation.
	if err = VerifyTreeHeadMultiSig(sth, km); err == nil {
		t.Fatalf("expected verification failure for unknown kid, got success")
	}
}
