package attest

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/notary"
	internalCrypto "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// helper to setup fresh key state each test
func createKeyManager(t *testing.T) *internalCrypto.Manager {
	km, err := internalCrypto.NewManager(1 * time.Hour)
	if err != nil {
		t.Fatalf("fresh key manager: %v", err)
	}
	if km.Active() == nil {
		t.Fatalf("expected active key")
	}
	return km
}

func TestBuildUnsignedGeneratesNonce(t *testing.T) {
	km := createKeyManager(t)
	svc := NewAttestationService(WithKeyProvider(km))
	raw := []byte(`{"demo":true}`)
	_, nonce1 := svc.BuildUnsigned(raw, "")
	_, nonce2 := svc.BuildUnsigned(raw, "")
	if nonce1 == "" || nonce2 == "" {
		t.Fatalf("nonce empty")
	}
	if nonce1 == nonce2 {
		t.Fatalf("expected distinct nonces got same: %s", nonce1)
	}
	// Provided nonce preserved
	_, given := svc.BuildUnsigned(raw, "fixednonce")
	if given != "fixednonce" {
		t.Fatalf("expected provided nonce preserved got %s", given)
	}
}

func TestSignDomainSeparatedAndVerify(t *testing.T) {
	km := createKeyManager(t)
	svc := NewAttestationService(WithKeyProvider(km))
	raw := []byte(`{"x":1}`)
	prefix := "GAUTH_ATTEST:"
	sig, kid, err := svc.SignDomainSeparated(prefix, raw)
	if err != nil {
		t.Fatalf("sign err: %v", err)
	}
	if kid == "" {
		t.Fatalf("kid empty")
	}
	active := km.Active()
	if active == nil || len(active.Public) != ed25519.PublicKeySize {
		t.Fatalf("active public invalid")
	}
	msg := append([]byte(prefix), raw...)
	if !ed25519.Verify(active.Public, msg, sig) {
		t.Fatalf("signature failed verification")
	}
}

func TestSignDomainSeparatedB64(t *testing.T) {
	km := createKeyManager(t)
	svc := NewAttestationService(WithKeyProvider(km))
	raw := []byte(`{"y":2}`)
	prefix := "ATTEST:"
	b64, kid, err := svc.SignDomainSeparatedB64(prefix, raw)
	if err != nil {
		t.Fatalf("sign b64 err: %v", err)
	}
	if b64 == "" {
		t.Fatalf("empty b64 signature")
	}
	sigBytes, decErr := base64.RawStdEncoding.DecodeString(b64)
	if decErr != nil {
		t.Fatalf("decode err: %v", decErr)
	}
	active := km.Active()
	msg := append([]byte(prefix), raw...)
	if !ed25519.Verify(active.Public, msg, sigBytes) {
		t.Fatalf("b64 signature fails verify")
	}
	if kid == "" {
		t.Fatalf("kid empty")
	}
}

func TestNotarizeAndSignModelLimits_WithNotarizationAndSigning(t *testing.T) {
	km := createKeyManager(t)
	svc := NewAttestationService(WithKeyProvider(km))
	// unsigned JSON canonical
	unsigned := []byte(`{"limits":{"max_tokens":100}}`)
	snapshotHash := "sha256:abcd" // arbitrary non-empty
	auditHead := "audit123"
	anchorHead := "anchorXYZ"
	mem := notary.NewMemory()
	res, err := svc.NotarizeAndSignModelLimits(unsigned, snapshotHash, auditHead, anchorHead, true, true, mem)
	if err != nil {
		t.Fatalf("notarize+sign err: %v", err)
	}
	if res.Notarization == nil {
		t.Fatalf("expected notarization present")
	}
	if res.Notarization.Provider != "memory" {
		t.Fatalf("provider mismatch: %s", res.Notarization.Provider)
	}
	if res.Signature == "" || res.SigKid == "" {
		t.Fatalf("signature fields empty")
	}
	sigBytes, decErr := base64.RawStdEncoding.DecodeString(res.Signature)
	if decErr != nil {
		t.Fatalf("sig decode: %v", decErr)
	}
	active := km.Active()
	if active == nil {
		t.Fatalf("active nil")
	}
	prefixed := append([]byte(AttestationDomainPrefix), unsigned...)
	if !ed25519.Verify(active.Public, prefixed, sigBytes) {
		t.Fatalf("primary signature fails verification over domain-prefixed payload")
	}
}

type mockKeyProvider struct{}

func (m *mockKeyProvider) ActiveSigner() (internalCrypto.Signer, error) {
	return nil, errors.New("no active key")
}
func (m *mockKeyProvider) PublicKey(keyID string) ([]byte, string, error) { return nil, "", nil }
func (m *mockKeyProvider) VerifyWith(msg, sig []byte, keyID string) error { return nil }

func TestNotarizeAndSignModelLimits_NoKey(t *testing.T) {
	// Use mock provider that returns error
	svc := NewAttestationService(WithKeyProvider(&mockKeyProvider{}))
	unsigned := []byte(`{"limits":{"max_tokens":50}}`)
	mem := notary.NewMemory()
	res, err := svc.NotarizeAndSignModelLimits(unsigned, "sha256:z", "a", "b", true, true, mem)
	if err == nil {
		t.Fatalf("expected error no_active_key")
	}
	if res.Signature != "" {
		t.Fatalf("signature should be empty on error")
	}
}

func TestNotarizeAndSignModelLimits_NoPrivateMaterial(t *testing.T) {
	km := createKeyManager(t)
	// Corrupt private key length
	active := km.Active()
	if active == nil {
		t.Fatalf("active nil")
	}
	active.Private = active.Private[:10] // truncate to trigger error path

	svc := NewAttestationService(WithKeyProvider(km))
	unsigned := []byte(`{"limits":{"max_tokens":10}}`)
	mem := notary.NewMemory()
	_, err := svc.NotarizeAndSignModelLimits(unsigned, "sha256:q", "a", "b", false, true, mem)
	if err == nil {
		t.Fatalf("expected error no_private_material")
	}
}

func TestNotarizeAndSignModelLimits_DomainPrefixDualSignature(t *testing.T) {
	km := createKeyManager(t)
	os.Setenv("GAUTH_ATTEST_DOMAIN_PREFIX", "GAUTH_ATTEST:")
	defer os.Unsetenv("GAUTH_ATTEST_DOMAIN_PREFIX")
	svc := NewAttestationService(WithKeyProvider(km))
	unsigned := []byte(`{"limits":{"max_tokens":77}}`)
	mem := notary.NewMemory()
	res, err := svc.NotarizeAndSignModelLimits(unsigned, "sha256:zz", "a", "b", false, true, mem)
	if err != nil {
		t.Fatalf("sign err: %v", err)
	}
	if res.Signature == "" {
		t.Fatalf("raw signature missing")
	}
	if res.DomainSignature == "" {
		t.Fatalf("domain signature missing")
	}
	if res.DomainPrefix != "GAUTH_ATTEST:" {
		t.Fatalf("domain prefix mismatch: %s", res.DomainPrefix)
	}
	rawBytes, _ := base64.RawStdEncoding.DecodeString(res.Signature)
	domBytes, _ := base64.RawStdEncoding.DecodeString(res.DomainSignature)
	active := km.Active()
	prefixed := append([]byte(AttestationDomainPrefix), unsigned...)
	if !ed25519.Verify(active.Public, prefixed, rawBytes) {
		t.Fatalf("primary signature verify fail (domain prefix)")
	}
	msg := append([]byte(res.DomainPrefix), unsigned...)
	if !ed25519.Verify(active.Public, msg, domBytes) {
		t.Fatalf("domain verify fail")
	}
}

func TestCanonicalizeModelLimitsUnsigned_StripsSignatureFields(t *testing.T) {
	svc := NewAttestationService()
	payload := map[string]any{
		"success":          true,
		"signature":        "abc",
		"sig_kid":          "kid",
		"sig_mode":         "eddsa",
		"domain_signature": "xyz",
		"domain_prefix":    "PFX:",
		"snapshot":         map[string]any{"hash": "sha256:123", "generated_at": "2025-10-27T00:00:00Z"},
	}
	enc, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	canon, err := svc.CanonicalizeModelLimitsUnsigned(enc)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}
	var out map[string]any
	if json.Unmarshal(canon, &out) != nil {
		t.Fatalf("unmarshal canon")
	}
	for _, k := range []string{"signature", "sig_kid", "sig_mode", "domain_signature", "domain_prefix"} {
		if _, ok := out[k]; ok {
			t.Fatalf("expected %s removed", k)
		}
	}
	if out["success"] != true {
		t.Fatalf("success field lost")
	}
	if _, ok := out["snapshot"].(map[string]any); !ok {
		t.Fatalf("snapshot missing")
	}
}
