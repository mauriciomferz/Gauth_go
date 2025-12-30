package test

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	auditor "github.com/mauriciomferz/AgentAuth/pkg/auditor"
	cryptoReg "github.com/mauriciomferz/AgentAuth/pkg/crypto"
	poa "github.com/mauriciomferz/AgentAuth/pkg/poa"
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
	// Create local key manager
	km, err := cryptoReg.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("failed to create key manager: %v", err)
	}
	ak := km.Active()
	if ak == nil {
		t.Fatalf("no active key")
	}

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

	// Pass local manager to VerifyPOA
	res := auditor.VerifyPOA(poaDoc, km)
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
