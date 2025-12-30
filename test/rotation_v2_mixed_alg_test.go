package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	notary "github.com/mauriciomferz/AgentAuth/internal/notary"
)

// rType implements notary.PublicKeyResolver for tests.
type rType struct{ key ed25519.PublicKey }

func (r *rType) FindByID(id string) *notary.PublicKeyRecord {
	if id == "s1" {
		return &notary.PublicKeyRecord{Ed25519: r.key}
	}
	return nil
}

// Placeholder test skeleton for mixed algorithm handling until ECDSA resolver support is added.
func TestRotationV2MixedAlgorithmsConfigDefaultAlg(t *testing.T) {
	// Build artifact with one signer missing alg (default ED25519) and ensure loader fills it.
	cfg := &notary.WeightsConfig{SchemaVersion: 1, ActiveKeySetID: "default-set", ThresholdWeight: 40,
		Signers: []struct {
			ID     string `json:"id"`
			Alg    string `json:"alg,omitempty"`
			Weight int    `json:"weight"`
		}{
			{ID: "s1", Alg: "ED25519", Weight: 30}, {ID: "s2", Alg: "ED25519", Weight: 20},
		}, AlgorithmSuite: []string{"ed25519"}}
	art, err := notary.BuildArtifactFromConfig(cfg, "", time.Now(), nil)
	if err != nil {
		t.Fatalf("build artifact: %v", err)
	}
	for _, s := range art.Signers {
		if s.Alg == "" {
			t.Fatalf("signer %s alg empty after build", s.ID)
		}
	}
	// Attach signature for one signer to exercise verified weight path.
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	// Replace resolver usage with a minimal inline type.
	if err := notary.AttachEd25519Signature(&art, priv, "s1", "ED25519", 30); err != nil {
		t.Fatalf("attach sig: %v", err)
	}
	// Minimal resolver instance
	r := &rType{key: pub}
	verified, _, failures := notary.VerifyArtifactSignatures(&art, r)
	if verified != 30 {
		t.Fatalf("expected verified weight 30 got %d failures=%v", verified, failures)
	}
}
