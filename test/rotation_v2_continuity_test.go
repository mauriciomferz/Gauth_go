package test

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	notary "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web"
	"github.com/gin-gonic/gin"
)

// singleEdResolver implements PublicKeyResolver for a single Ed25519 key.
type singleEdResolver struct{ key ed25519.PublicKey }

func (r *singleEdResolver) FindByID(id string) *notary.PublicKeyRecord {
	if id == "s1" {
		return &notary.PublicKeyRecord{Ed25519: r.key}
	}
	return nil
}

func TestRotationV2ContinuityUpdatesPreviousHash(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := &web.BetaServer{}
	// Continuity relies on srv.rotationLastV2Hash so we simulate directly.
	// Build first artifact manually (prev empty)
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	signers := []notary.WeightedRotationSigner{{ID: "s1", Alg: "ED25519", Weight: 10}}
	art1, err := notary.BuildWeightedRotationArtifact("setA", "", 10, signers, []string{"ed25519"}, time.Now())
	if err != nil {
		t.Fatalf("artifact1: %v", err)
	}
	if err2 := notary.AttachEd25519Signature(&art1, priv, "s1", "ED25519", 10); err2 != nil {
		t.Fatalf("attach1: %v", err2)
	}
	verified1, _, _ := notary.VerifyArtifactSignatures(&art1, &singleEdResolver{key: pub})
	if verified1 != 10 {
		t.Fatalf("verified1 mismatch")
	}
	srv.RotationV2ContinuityUpdate(art1.CanonicalDigest)
	// Second artifact should include previous hash
	art2, err := notary.BuildWeightedRotationArtifact("setA", srv.RotationV2LastHash(), 10, signers, []string{"ed25519"}, time.Now())
	if err != nil {
		t.Fatalf("artifact2: %v", err)
	}
	if art2.PreviousArtifactHash == "" {
		t.Fatalf("expected previous hash present")
	}
	if art2.PreviousArtifactHash != art1.CanonicalDigest {
		t.Fatalf("continuity mismatch prev=%s expected=%s", art2.PreviousArtifactHash, art1.CanonicalDigest)
	}
}
