package delegation

import (
	"os"
	"testing"
	"time"

	cryptoInt "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

func TestMultiSignatureTreeHeadThreshold(t *testing.T) {
	// Setup key manager with multiple keys by manual rotation
	km, err := cryptoInt.NewManager(24 * time.Hour)
	if err != nil {
		t.Fatalf("manager init: %v", err)
	}
	if _, err2 := km.Rotate(); err2 != nil {
		t.Fatalf("rotate1: %v", err)
	}
	if _, err2 := km.Rotate(); err2 != nil {
		t.Fatalf("rotate2: %v", err)
	}
	// Configure threshold=2
	os.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "2")
	chain := NewRevocationChain(WithKeyProvider(km))
	// Append some events
	for i := 0; i < 3; i++ {
		if _, err2 := chain.Append(RevocationEvent{ID: "rev-" + time.Now().Format("150405") + string(rune('a'+i)), DelegationID: "del-"}); err != nil {
			t.Fatalf("append: %v", err2)
		}
		time.Sleep(5 * time.Millisecond) // ensure unique timestamp for hash determinism
	}
	sth, err := chain.SignTreeHead()
	if err != nil {
		t.Fatalf("sign tree head: %v", err)
	}
	if sth.Threshold != 2 {
		t.Fatalf("expected threshold 2, got %d", sth.Threshold)
	}
	if len(sth.Signatures) < 2 {
		t.Fatalf("expected >=2 signatures, got %d", len(sth.Signatures))
	}
	if err := VerifyTreeHeadMultiSig(sth, km); err != nil {
		t.Fatalf("multi-sig verification failed: %v", err)
	}
}

func TestMultiSignatureTreeHeadWeights(t *testing.T) {
	km, err := cryptoInt.NewManager(24 * time.Hour)
	if err != nil {
		t.Fatalf("manager init: %v", err)
	}
	// Rotate twice to have 3 keys total (active + 2 history)
	if _, err2 := km.Rotate(); err2 != nil {
		t.Fatalf("rotate1: %v", err)
	}
	if _, err2 := km.Rotate(); err2 != nil {
		t.Fatalf("rotate2: %v", err)
	}
	// Build weights mapping
	keys := km.ListCurrent()
	if len(keys) < 2 {
		t.Fatalf("need at least two keys")
	}
	wEnv := keys[0].ID + "=2," + keys[1].ID + "=1"
	os.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "3")
	os.Setenv("GAUTH_MULTI_SIG_WEIGHTS", wEnv)
	chain := NewRevocationChain(WithKeyProvider(km))
	if _, err2 := chain.Append(RevocationEvent{ID: "rev-x", DelegationID: "del-x"}); err != nil {
		t.Fatalf("append: %v", err2)
	}
	sth, err := chain.SignTreeHead()
	if err != nil {
		t.Fatalf("sign tree head: %v", err)
	}
	if sth.Version < 2 {
		t.Fatalf("expected version >=2 for multi-sig, got %d", sth.Version)
	}
	if sth.Threshold != 3 {
		t.Fatalf("threshold mismatch: %d", sth.Threshold)
	}
	if sth.SatisfiedWeight < 3 {
		t.Fatalf("satisfied weight insufficient: %d", sth.SatisfiedWeight)
	}
	if err := VerifyTreeHeadMultiSig(sth, km); err != nil {
		t.Fatalf("verification failed: %v", err)
	}
}
