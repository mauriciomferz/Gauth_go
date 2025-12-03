package poa

import (
	"context"
	"os"
	"testing"
	"time"

	internalCrypto "github.com/mauriciomferz/Gauth_go/internal/crypto"
)

// helper to ensure eddsa registry seeded
func ensureRegistry(t *testing.T) {
	if internalCrypto.GlobalEdDSARegistry == nil {
		km, err := internalCrypto.NewManager(time.Hour)
		if err != nil {
			t.Fatalf("new manager: %v", err)
		}
		// Produce multiple rotations to accumulate history
		for i := 0; i < 5; i++ {
			if _, err := km.Rotate(); err != nil {
				t.Fatalf("rotate %d: %v", i+1, err)
			}
		}
		internalCrypto.GlobalEdDSARegistry = km
	}
}

func TestPoAMultiSigThresholdSatisfied(t *testing.T) {
	ensureRegistry(t)
	// After rotations, active is first; map unique IDs
	keys := internalCrypto.GlobalEdDSARegistry.ListCurrent()
	// Deduplicate potential identical active/history ids (defensive)
	uniq := []string{}
	seen := map[string]struct{}{}
	for _, k := range keys {
		if k != nil {
			if _, ok := seen[k.ID]; !ok {
				uniq = append(uniq, k.ID)
				seen[k.ID] = struct{}{}
			}
		}
	}
	if len(uniq) < 2 {
		t.Fatalf("need at least 2 unique keys for test have=%d", len(uniq))
	}
	// Use first two kids
	os.Setenv("GAUTH_POA_MULTISIG_SIGN", "1")
	os.Setenv("GAUTH_POA_MULTISIG_KIDS", uniq[0]+","+uniq[1])
	os.Setenv("GAUTH_POA_MULTISIG_THRESHOLD", "2")
	svc := NewMemoryService()
	poaObj, err := svc.Issue(context.TODO(), &Request{Subject: "alice", Resource: "vault", Action: "read"})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if len(poaObj.Signatures) != 2 || len(poaObj.SignerKids) != 2 {
		t.Fatalf("expected 2 signatures, got %d", len(poaObj.Signatures))
	}
	vcount, satisfied, req := VerifyMultiSig(poaObj, internalCrypto.GlobalEdDSARegistry)
	if vcount != 2 || !satisfied || req != 2 {
		t.Fatalf("multisig verify mismatch valid=%d satisfied=%v req=%d", vcount, satisfied, req)
	}
}

func TestPoAMultiSigThresholdUnsatisfied(t *testing.T) {
	ensureRegistry(t)
	keys := internalCrypto.GlobalEdDSARegistry.ListCurrent()
	uniq := []string{}
	seen := map[string]struct{}{}
	for _, k := range keys {
		if k != nil {
			if _, ok := seen[k.ID]; !ok {
				uniq = append(uniq, k.ID)
				seen[k.ID] = struct{}{}
			}
		}
	}
	if len(uniq) < 2 {
		t.Fatalf("need at least 2 unique keys for unsatisfied test have=%d", len(uniq))
	}
	os.Setenv("GAUTH_POA_MULTISIG_SIGN", "1")
	os.Setenv("GAUTH_POA_MULTISIG_KIDS", uniq[0]+","+uniq[1])
	os.Setenv("GAUTH_POA_MULTISIG_THRESHOLD", "3") // impossible threshold
	svc := NewMemoryService()
	poaObj, err := svc.Issue(context.TODO(), &Request{Subject: "bob", Resource: "ledger", Action: "append"})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	vcount, satisfied, req := VerifyMultiSig(poaObj, internalCrypto.GlobalEdDSARegistry)
	if vcount >= req || satisfied || req != 3 {
		t.Fatalf("expected unsatisfied threshold vcount=%d req=%d satisfied=%v", vcount, req, satisfied)
	}
}

func TestPoAMultiSigTamperSignature(t *testing.T) {
	ensureRegistry(t)
	keys := internalCrypto.GlobalEdDSARegistry.ListCurrent()
	if len(keys) < 1 {
		t.Fatalf("need at least 1 key")
	}
	os.Setenv("GAUTH_POA_MULTISIG_SIGN", "1")
	os.Setenv("GAUTH_POA_MULTISIG_KIDS", keys[0].ID)
	os.Setenv("GAUTH_POA_MULTISIG_THRESHOLD", "1")
	svc := NewMemoryService()
	poaObj, err := svc.Issue(context.TODO(), &Request{Subject: "carol", Resource: "db", Action: "query"})
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if len(poaObj.Signatures) != 1 {
		t.Fatalf("expected one signature")
	}
	// Tamper first signature
	poaObj.Signatures[0] = poaObj.Signatures[0][:10] + "AAAA" + poaObj.Signatures[0][10:]
	vcount, satisfied, _ := VerifyMultiSig(poaObj, internalCrypto.GlobalEdDSARegistry)
	if vcount != 0 || satisfied {
		t.Fatalf("tamper should invalidate signature valid=%d satisfied=%v", vcount, satisfied)
	}
}
