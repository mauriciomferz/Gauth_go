package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"
	"time"

	notary "github.com/mauriciomferz/Gauth_go/internal/notary"
)

// TestRotationV2EmbeddedPublicKeyVerification ensures auditor-style verification logic can validate
// embedded public keys' signatures and achieve threshold satisfaction. Since the auditor logic
// is implemented separately, we replicate its digest and signature preimage computation here.
func TestRotationV2EmbeddedPublicKeyVerification(t *testing.T) {
	// Set env to embed public keys when building artifact from config.
	os.Setenv("GAUTH_ROTATIONS_V2_EMBED_PUBS", "1")
	defer os.Unsetenv("GAUTH_ROTATIONS_V2_EMBED_PUBS")

	// Create a config with two Ed25519 signers that together meet threshold.
	cfg := &notary.WeightsConfig{SchemaVersion: 1, ActiveKeySetID: "embed-set", ThresholdWeight: 6,
		Signers: []struct {
			ID     string `json:"id"`
			Alg    string `json:"alg,omitempty"`
			Weight int    `json:"weight"`
		}{
			{ID: "sA", Alg: "ED25519", Weight: 3}, {ID: "sB", Alg: "ED25519", Weight: 3},
		}, AlgorithmSuite: []string{"ed25519"}}

	// Build initial artifact (no public keys yet because global resolver must supply them).
	// We will manually embed public keys to simulate server embedding to avoid dependency on global registry state.
	art, err := notary.BuildArtifactFromConfig(cfg, "", time.Now(), nil)
	if err != nil {
		t.Fatalf("artifact build: %v", err)
	}

	// Generate keys for both signers and attach signatures.
	pubA, privA, _ := ed25519.GenerateKey(rand.Reader)
	pubB, privB, _ := ed25519.GenerateKey(rand.Reader)
	if err := notary.AttachEd25519Signature(&art, privA, "sA", "ED25519", 3); err != nil {
		t.Fatalf("attach A: %v", err)
	}
	if err := notary.AttachEd25519Signature(&art, privB, "sB", "ED25519", 3); err != nil {
		t.Fatalf("attach B: %v", err)
	}

	// Manually embed public keys (since cryptoGlobalEdDSAResolve may not have them in test environment).
	for i := range art.Signers {
		switch art.Signers[i].ID {
		case "sA":
			art.Signers[i].Public = base64.RawURLEncoding.EncodeToString(pubA)
		case "sB":
			art.Signers[i].Public = base64.RawURLEncoding.EncodeToString(pubB)
		}
	}

	// Recompute digest should remain unchanged after embedding.
	originalDigest := art.CanonicalDigest
	recomputed := recomputeDigestNoEmbedded(&art)
	if originalDigest != recomputed {
		t.Fatalf("digest changed after embedding public keys: orig=%s new=%s", originalDigest, recomputed)
	}

	// Auditor-style verification.
	verifiedWeight, failures, thresholdMet := auditorStyleVerify(&art)
	if verifiedWeight != 6 {
		t.Fatalf("expected verified weight 6 got %d failures=%v", verifiedWeight, failures)
	}
	if !thresholdMet {
		t.Fatalf("expected threshold met")
	}
	if len(failures) != 0 {
		t.Fatalf("unexpected failures: %v", failures)
	}
}

// recomputeDigestNoEmbedded replicates digest logic excluding signatures/public keys.
func recomputeDigestNoEmbedded(art *notary.WeightedRotationArtifact) string {
	if art == nil {
		return ""
	}
	return notary.ComputeDebugDigestForTests(art) // helper added for tests (exposed via build tag) or we inline if unavailable.
}

// auditorStyleVerify replicates subset of auditor verification for embedded Ed25519.
func auditorStyleVerify(art *notary.WeightedRotationArtifact) (int, []string, bool) {
	if art == nil {
		return 0, []string{"artifact_nil"}, false
	}
	preimage := []byte("GAUTH_ROTATION_V2:" + art.CanonicalDigest)
	verified := 0
	failures := []string{}
	for _, s := range art.Signers {
		if s.Signature == "" {
			continue
		}
		if s.Public == "" {
			failures = append(failures, "public_missing:"+s.ID)
			continue
		}
		pubBytes, err := base64.RawURLEncoding.DecodeString(s.Public)
		if err != nil || len(pubBytes) != ed25519.PublicKeySize {
			failures = append(failures, "public_decode:"+s.ID)
			continue
		}
		sigBytes, err := base64.RawURLEncoding.DecodeString(s.Signature)
		if err != nil || len(sigBytes) != ed25519.SignatureSize {
			failures = append(failures, "signature_decode:"+s.ID)
			continue
		}
		if !ed25519.Verify(ed25519.PublicKey(pubBytes), preimage, sigBytes) {
			failures = append(failures, "signature_invalid:"+s.ID)
			continue
		}
		verified += s.Weight
	}
	return verified, failures, verified >= art.ThresholdWeight
}
