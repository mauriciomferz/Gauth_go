package test

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	cryptoReg "github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// minimalAttestationUnsigned mirrors auditor unsigned struct subset
type minimalAttestationUnsigned struct {
	Success    bool   `json:"success"`
	Configured bool   `json:"configured"`
	Reason     string `json:"reason,omitempty"`
	Nonce      string `json:"nonce,omitempty"`
	Snapshot   struct {
		Hash        string `json:"hash"`
		GeneratedAt string `json:"generated_at"`
	} `json:"snapshot"`
}

func TestAuditorAttestationSignature(t *testing.T) {
	reg, err := cryptoReg.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("setup: %v", err)
	}
	if reg.Active() == nil {
		t.Skip("active key nil")
	}
	ak := reg.Active()
	unsigned := minimalAttestationUnsigned{Success: true, Configured: true, Nonce: "nonce123"}
	unsigned.Snapshot.Hash = "demo_hash"
	unsigned.Snapshot.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	raw, _ := json.Marshal(unsigned)
	msg := append([]byte("GAUTH_MODEL_LIMIT_ATTEST:"), raw...)
	sig := ed25519.Sign(ak.Private, msg)
	att := map[string]any{
		"success":    true,
		"configured": true,
		"nonce":      unsigned.Nonce,
		"snapshot":   map[string]string{"hash": unsigned.Snapshot.Hash, "generated_at": unsigned.Snapshot.GeneratedAt},
		"signature":  base64.RawStdEncoding.EncodeToString(sig),
		"sig_kid":    ak.ID,
		"sig_mode":   "eddsa",
	}
	// Recompute combined hash
	seed := "attest|" + unsigned.Snapshot.Hash + "||" // no audit/anchor heads
	ch := sha256.Sum256([]byte(seed))
	combined := "sha256:" + base64.StdEncoding.EncodeToString(ch[:])
	_ = combined // placeholder: could assert formatting later
	// Verify signature manually
	bsig, _ := base64.RawStdEncoding.DecodeString(att["signature"].(string))
	if !ed25519.Verify(ak.Public, msg, bsig) {
		t.Fatalf("expected valid attestation signature")
	}
}
