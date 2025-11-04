package test

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	notary "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

// TestRotationV2EmbeddedECDSAPublicKeyVerification ensures embedded uncompressed P-256 point enables auditor-style verification.
//nolint:gocyclo // ECDSA verification test with multiple cases
func TestRotationV2EmbeddedECDSAPublicKeyVerification(t *testing.T) {
	os.Setenv("GAUTH_ROTATIONS_V2_EMBED_PUBS", "1")
	defer os.Unsetenv("GAUTH_ROTATIONS_V2_EMBED_PUBS")
	// We'll inject ECDSA key via GAUTH_ROTATIONS_V2_ECDSA_KEYS env.
	cfg := &notary.WeightsConfig{SchemaVersion: 1, ActiveKeySetID: "embed-ec-set", ThresholdWeight: 9,
		Signers: []struct {
			ID     string `json:"id"`
			Alg    string `json:"alg,omitempty"`
			Weight int    `json:"weight"`
		}{
			{ID: "edA", Alg: "ED25519", Weight: 4}, {ID: "ecA", Alg: "ECDSA-P256", Weight: 5},
		}, AlgorithmSuite: []string{"ed25519", "ecdsa-p256"}}
	art, err := notary.BuildArtifactFromConfig(cfg, "", time.Now())
	if err != nil {
		t.Fatalf("artifact build: %v", err)
	}
	// Keys
	pubEd, privEd, _ := ed25519.GenerateKey(rand.Reader)
	privEc, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ecdsa key: %v", err)
	}
	if err := notary.AttachEd25519Signature(&art, privEd, "edA", "ED25519", 4); err != nil {
		t.Fatalf("attach ed: %v", err)
	}
	if err := notary.AttachECDSASignature(&art, privEc, "ecA", "ECDSA-P256", 5); err != nil {
		t.Fatalf("attach ec: %v", err)
	}
	// Embed public keys manually: Ed25519 via direct assignment since registry may not serve; ECDSA via env variable mapping
	// Use helper encoding
	encoded := notary.EncodeECDSAP256Uncompressed(&privEc.PublicKey)
	if encoded == "" {
		t.Fatalf("helper returned empty encoding")
	}
	os.Setenv("GAUTH_ROTATIONS_V2_ECDSA_KEYS", "ecA:"+encoded)
	// Rebuild artifact to trigger embedding paths.
	art2, err := notary.BuildArtifactFromConfig(cfg, "", time.Now())
	if err != nil {
		t.Fatalf("artifact rebuild: %v", err)
	}
	// Transfer signatures from initial artifact (embedding build does not sign)
	for _, s := range art.Signers {
		for i := range art2.Signers {
			if art2.Signers[i].ID == s.ID {
				art2.Signers[i].Signature = s.Signature
			}
		}
	}
	// Auditor-style verification: preimage domain separation
	preimage := []byte("GAUTH_ROTATION_V2:" + art2.CanonicalDigest)
	verified := 0
	failures := []string{}
	for _, s := range art2.Signers {
		if s.Signature == "" {
			continue
		}
		sigBytes, err := base64.RawURLEncoding.DecodeString(s.Signature)
		if err != nil {
			failures = append(failures, "sig_decode:"+s.ID)
			continue
		}
		switch strings.ToUpper(s.Alg) {
		case "ED25519":
			pubBytes := pubEd
			if s.Public != "" {
				if pb, err := base64.RawURLEncoding.DecodeString(s.Public); err == nil && len(pb) == ed25519.PublicKeySize {
					pubBytes = ed25519.PublicKey(pb)
				}
			}
			if len(sigBytes) != ed25519.SignatureSize || !ed25519.Verify(pubBytes, preimage, sigBytes) {
				failures = append(failures, "sig_invalid:"+s.ID)
				continue
			}
			verified += s.Weight
		case "ECDSA-P256", "ECDSA_P256", "ECDSA-P256-SHA256":
			if s.Public == "" {
				failures = append(failures, "public_missing:"+s.ID)
				continue
			}
			pb, err := base64.RawURLEncoding.DecodeString(s.Public)
			if err != nil || len(pb) != 65 || pb[0] != 0x04 {
				failures = append(failures, "public_decode:"+s.ID)
				continue
			}
			x := new(big.Int).SetBytes(pb[1:33])
			y := new(big.Int).SetBytes(pb[33:65])
			curve := elliptic.P256()
			if !curve.IsOnCurve(x, y) {
				failures = append(failures, "public_point:"+s.ID)
				continue
			}
			var rs struct{ R, S *big.Int }
			if _, err := asn1.Unmarshal(sigBytes, &rs); err != nil || rs.R == nil || rs.S == nil {
				failures = append(failures, "sig_decode_asn1:"+s.ID)
				continue
			}
			h := sha256.Sum256(preimage)
			pk := ecdsa.PublicKey{Curve: curve, X: x, Y: y}
			if !ecdsa.Verify(&pk, h[:], rs.R, rs.S) {
				failures = append(failures, "sig_invalid:"+s.ID)
				continue
			}
			verified += s.Weight
		}
	}
	if verified != 9 {
		t.Fatalf("expected verified weight 9 got %d failures=%v", verified, failures)
	}
	if len(failures) != 0 {
		t.Fatalf("unexpected failures: %v", failures)
	}
	if art2.ThresholdWeight != 9 {
		t.Fatalf("threshold mismatch")
	}
	if art2.CanonicalDigest == "" {
		t.Fatalf("digest empty")
	}
}
