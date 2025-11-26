package test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	cryptoReg "github.com/mauriciomferz/Gauth_go/internal/crypto"
	auditor "github.com/mauriciomferz/Gauth_go/pkg/auditor"
	poa "github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// buildSigningPayload mirrors unexported buildPoASigningPayload for test signature generation.
func buildSigningPayload(p *poa.ProofOfAuthorization) []byte {
	type canon struct {
		ID        string    `json:"id"`
		Subject   string    `json:"subject"`
		Resource  string    `json:"resource"`
		Action    string    `json:"action"`
		Issuer    string    `json:"issuer"`
		IssuedAt  time.Time `json:"issued_at"`
		ExpiresAt time.Time `json:"expires_at"`
		Scope     []string  `json:"scope"`
	}
	c := canon{ID: p.ID, Subject: p.Subject, Resource: p.Resource, Action: p.Action, Issuer: p.Issuer, IssuedAt: p.IssuedAt, ExpiresAt: p.ExpiresAt, Scope: append([]string(nil), p.Scope...)}
	raw, _ := json.Marshal(c)
	return append([]byte("GAUTH_POA:"), raw...)
}

func TestAuditorVerifyPOA(t *testing.T) {
	reg := cryptoReg.GlobalEdDSARegistry
	if reg == nil || reg.Active() == nil {
		t.Skip("eddsa registry not initialized; skip auditor poa test")
	}
	ak := reg.Active()
	poaDoc := &poa.ProofOfAuthorization{
		ID:        "poa_demo_1",
		Subject:   "alice",
		Resource:  "vault:secret",
		Action:    "read",
		Issuer:    "auditor-test",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scope:     []string{"basic"},
	}
	poaDoc.Digest = poa.CanonicalDigest(poaDoc)
	poaDoc.SignerKids = []string{ak.ID}
	poaDoc.Threshold = 1
	msg := buildSigningPayload(poaDoc)
	sig := ed25519.Sign(ak.Private, msg)
	poaDoc.Signatures = []string{base64.RawStdEncoding.EncodeToString(sig)}

	res := auditor.VerifyPOA(poaDoc)
	if !res["digest_valid"].(bool) {
		t.Fatalf("expected digest_valid true")
	}
	if !res["threshold_met"].(bool) {
		t.Fatalf("expected threshold_met true")
	}
	if res["valid_signatures"].(int) != 1 {
		t.Fatalf("expected 1 valid signature")
	}
}
