package attest

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"
	"time"

	internalcrypto "github.com/mauriciomferz/Gauth_go/internal/crypto"
)

const testSnapshotHash = "sha256:demo"

// memoryReplay simple in-memory replay strategy for tests
type memoryReplay struct{ seen map[string]struct{} }

func (m *memoryReplay) Seen(nonce string) bool {
	if m.seen == nil {
		return false
	}
	_, ok := m.seen[nonce]
	return ok
}
func (m *memoryReplay) Record(nonce string) {
	if m.seen == nil {
		m.seen = make(map[string]struct{})
	}
	m.seen[nonce] = struct{}{}
}

func TestVerifyModelLimitsAttestationValid(t *testing.T) {
	// Generate ephemeral key and manager
	pub, priv, _ := ed25519.GenerateKey(nil)
	// Manager not required; we use stubKeyRegistry with public key only.
	att := &Attestation{Success: true, Configured: true, Nonce: "n1"}
	att.Snapshot.Hash = testSnapshotHash
	// Reconstruct unsigned struct exactly like verify.go logic
	unsigned := struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Reason     string `json:"reason,omitempty"`
		Nonce      string `json:"nonce,omitempty"`
		Snapshot   struct {
			Hash        string `json:"hash"`
			GeneratedAt string `json:"generated_at"`
		} `json:"snapshot"`
		Audit *struct {
			HeadHash string `json:"head_hash"`
			Entries  int    `json:"entries"`
		} `json:"audit,omitempty"`
		Anchor *struct {
			LatestHash string `json:"latest_hash"`
			Entries    int    `json:"entries"`
			Interval   int    `json:"interval"`
		} `json:"anchor,omitempty"`
		StrictUnknown bool `json:"strict_unknown"`
		Surge         *struct {
			ModelID   string  `json:"model_id"`
			Last10Sec int     `json:"last_10s_exceed_events"`
			AvgActive float64 `json:"avg_active_seconds"`
			Factor    float64 `json:"factor"`
			MinEvents int     `json:"min_events"`
			Triggered bool    `json:"triggered"`
			At        string  `json:"triggered_at,omitempty"`
		} `json:"surge,omitempty"`
		Notarization *struct {
			Provider       string  `json:"provider"`
			Timestamp      string  `json:"timestamp"`
			LatencySeconds float64 `json:"latency_seconds"`
			Success        bool    `json:"success"`
		} `json:"notarization,omitempty"`
	}{Success: att.Success, Configured: att.Configured, Reason: att.Reason, Nonce: att.Nonce, Snapshot: att.Snapshot}
	raw, _ := json.Marshal(unsigned)
	msg := append([]byte(AttestationDomainPrefix), raw...)
	sig := ed25519.Sign(priv, msg)
	att.Signature = base64.RawStdEncoding.EncodeToString(sig)
	att.SigKid = "k1"
	att.SigMode = sigModeEdDSA
	// Provide fake key registry struct implementing FindByID
	fakeRegistry := &stubKeyRegistry{kid: "k1", pub: pub}
	res, err := VerifyModelLimitsAttestation(att, fakeRegistry, &memoryReplay{}, time.Now())
	if err != nil || !res.Valid || res.Kid != "k1" || res.CombinedHash == "" {
		// Note combined hash requires audit/anchor heads; empty snapshot hash uses only snapshot hash component
		// res.CombinedHash should still be non-empty based on seed derivation.
		if err != nil {
			t.Fatalf("verify error: %v", err)
		}
		if !res.Valid {
			t.Fatalf("expected valid signature")
		}
		if res.Kid != "k1" {
			t.Fatalf("kid mismatch")
		}
		if res.CombinedHash == "" {
			t.Fatalf("combined hash empty")
		}
	}
}

func TestVerifyModelLimitsAttestationInvalidSignature(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	att := &Attestation{Success: true, Configured: true, Nonce: "n2"}
	att.Snapshot.Hash = testSnapshotHash
	// Provide bogus signature
	att.Signature = base64.RawStdEncoding.EncodeToString([]byte("invalid_signature"))
	att.SigKid = "k2"
	att.SigMode = sigModeEdDSA
	fakeRegistry := &stubKeyRegistry{kid: "k2", pub: pub}
	res, _ := VerifyModelLimitsAttestation(att, fakeRegistry, &memoryReplay{}, time.Now())
	if res.Valid || !res.SoftInvalid || res.FailureCode != "signature_invalid" {
		// Soft invalid expected
		if res.Valid {
			t.Fatalf("expected invalid signature")
		}
		if !res.SoftInvalid {
			t.Fatalf("expected SoftInvalid flag true")
		}
		if res.FailureCode != "signature_invalid" {
			t.Fatalf("expected failure_code signature_invalid got %s", res.FailureCode)
		}
	}
}

// stubKeyRegistry minimal implementation of FindByID
type stubKeyRegistry struct {
	kid string
	pub ed25519.PublicKey
}

func (s *stubKeyRegistry) FindByID(id string) *internalcrypto.Key {
	if id == s.kid {
		return &internalcrypto.Key{ID: s.kid, Public: s.pub}
	}
	return nil
}

// helper to sign an attestation using a private key matching verifier reconstruction logic
func signAtt(t *testing.T, priv ed25519.PrivateKey, att *Attestation) {
	unsigned := struct {
		Success    bool   `json:"success"`
		Configured bool   `json:"configured"`
		Reason     string `json:"reason,omitempty"`
		Nonce      string `json:"nonce,omitempty"`
		Snapshot   struct {
			Hash        string `json:"hash"`
			GeneratedAt string `json:"generated_at"`
		} `json:"snapshot"`
		Audit *struct {
			HeadHash string `json:"head_hash"`
			Entries  int    `json:"entries"`
		} `json:"audit,omitempty"`
		Anchor *struct {
			LatestHash string `json:"latest_hash"`
			Entries    int    `json:"entries"`
			Interval   int    `json:"interval"`
		} `json:"anchor,omitempty"`
		StrictUnknown bool `json:"strict_unknown"`
		Surge         *struct {
			ModelID   string  `json:"model_id"`
			Last10Sec int     `json:"last_10s_exceed_events"`
			AvgActive float64 `json:"avg_active_seconds"`
			Factor    float64 `json:"factor"`
			MinEvents int     `json:"min_events"`
			Triggered bool    `json:"triggered"`
			At        string  `json:"triggered_at,omitempty"`
		} `json:"surge,omitempty"`
		Notarization *struct {
			Provider       string  `json:"provider"`
			Timestamp      string  `json:"timestamp"`
			LatencySeconds float64 `json:"latency_seconds"`
			Success        bool    `json:"success"`
		} `json:"notarization,omitempty"`
	}{Success: att.Success, Configured: att.Configured, Reason: att.Reason, Nonce: att.Nonce, Snapshot: att.Snapshot, Audit: att.Audit, Anchor: att.Anchor, StrictUnknown: att.StrictUnknown, Surge: att.Surge, Notarization: att.Notarization}
	raw, _ := json.Marshal(unsigned)
	msg := append([]byte(AttestationDomainPrefix), raw...)
	sig := ed25519.Sign(priv, msg)
	att.Signature = base64.RawStdEncoding.EncodeToString(sig)
}

func TestVerifyModelLimitsAttestationReplay(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	att := &Attestation{Success: true, Configured: true, Nonce: "nonce-replay"}
	att.Snapshot.Hash = testSnapshotHash
	signAtt(t, priv, att)
	att.SigKid = "rkid"
	att.SigMode = sigModeEdDSA
	reg := &stubKeyRegistry{kid: "rkid", pub: pub}
	// Pre-seed replay strategy so nonce is seen
	mr := &memoryReplay{seen: map[string]struct{}{att.Nonce: {}}}
	res, _ := VerifyModelLimitsAttestation(att, reg, mr, time.Now())
	if res.FailureCode != "nonce_replay" || res.HTTPStatus != 409 {
		t.Fatalf("expected nonce_replay 409 got %+v", res)
	}
}

func TestVerifyModelLimitsAttestationNonceMissing(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	att := &Attestation{Success: true, Configured: true, Nonce: ""}
	att.Snapshot.Hash = testSnapshotHash
	signAtt(t, priv, att)
	att.SigKid = "nkid"
	att.SigMode = sigModeEdDSA
	reg := &stubKeyRegistry{kid: "nkid", pub: pub}
	res, _ := VerifyModelLimitsAttestation(att, reg, &memoryReplay{}, time.Now())
	if res.FailureCode != "nonce_missing" || res.HTTPStatus != 400 {
		t.Fatalf("expected nonce_missing 400 got %+v", res)
	}
}

func TestVerifyModelLimitsAttestationNotarizationInconsistent(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	att := &Attestation{Success: true, Configured: true, Nonce: "n3"}
	att.Snapshot.Hash = testSnapshotHash
	att.Notarization = &struct {
		Provider       string  `json:"provider"`
		Timestamp      string  `json:"timestamp"`
		LatencySeconds float64 `json:"latency_seconds"`
		Success        bool    `json:"success"`
	}{Provider: "stub", Timestamp: time.Now().UTC().Format(time.RFC3339Nano), LatencySeconds: 0.01, Success: false}
	signAtt(t, priv, att)
	att.SigKid = "nkid2"
	att.SigMode = sigModeEdDSA
	reg := &stubKeyRegistry{kid: "nkid2", pub: pub}
	res, _ := VerifyModelLimitsAttestation(att, reg, &memoryReplay{}, time.Now())
	if res.FailureCode != "notarization_inconsistent" || res.HTTPStatus != 422 {
		t.Fatalf("expected notarization_inconsistent 422 got %+v", res)
	}
}

func TestVerifyModelLimitsAttestationSignatureFieldsMissing(t *testing.T) {
	att := &Attestation{Success: true, Configured: true, Nonce: "n4"}
	att.Snapshot.Hash = testSnapshotHash
	// No signature fields populated
	res, _ := VerifyModelLimitsAttestation(att, nil, &memoryReplay{}, time.Now())
	if res.FailureCode != "signature_fields_missing" || res.HTTPStatus != 400 {
		t.Fatalf("expected signature_fields_missing 400 got %+v", res)
	}
}

func TestVerifyModelLimitsAttestationUnknownKid(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	att := &Attestation{Success: true, Configured: true, Nonce: "n5"}
	att.Snapshot.Hash = testSnapshotHash
	signAtt(t, priv, att)
	att.SigKid = "kid-real" // registry will not contain this kid
	att.SigMode = sigModeEdDSA
	reg := &stubKeyRegistry{kid: "different", pub: pub}
	res, _ := VerifyModelLimitsAttestation(att, reg, &memoryReplay{}, time.Now())
	if res.FailureCode != "unknown_kid" || res.HTTPStatus != 404 {
		t.Fatalf("expected unknown_kid 404 got %+v", res)
	}
}

func TestVerifyModelLimitsAttestationSignatureBase64Invalid(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	att := &Attestation{Success: true, Configured: true, Nonce: "n6"}
	att.Snapshot.Hash = testSnapshotHash
	att.Signature = "@@@" // invalid base64
	att.SigKid = "kid1"
	att.SigMode = sigModeEdDSA
	reg := &stubKeyRegistry{kid: "kid1", pub: pub}
	res, _ := VerifyModelLimitsAttestation(att, reg, &memoryReplay{}, time.Now())
	if res.FailureCode != "signature_base64_invalid" || res.HTTPStatus != 400 {
		t.Fatalf("expected signature_base64_invalid 400 got %+v", res)
	}
}

func TestVerifyModelLimitsAttestationCombinedHash(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	att := &Attestation{Success: true, Configured: true, Nonce: "n7"}
	att.Snapshot.Hash = "sha256:snap"
	att.Audit = &struct {
		HeadHash string `json:"head_hash"`
		Entries  int    `json:"entries"`
	}{HeadHash: "sha256:audit", Entries: 10}
	att.Anchor = &struct {
		LatestHash string `json:"latest_hash"`
		Entries    int    `json:"entries"`
		Interval   int    `json:"interval"`
	}{LatestHash: "sha256:anchor", Entries: 5, Interval: 1}
	signAtt(t, priv, att)
	att.SigKid = "kidCH"
	att.SigMode = sigModeEdDSA
	reg := &stubKeyRegistry{kid: "kidCH", pub: pub}
	res, _ := VerifyModelLimitsAttestation(att, reg, &memoryReplay{}, time.Now())
	if !res.Valid || res.CombinedHash == "" {
		t.Fatalf("expected valid combined hash got %+v", res)
	}
	expectedSeed := "attest|sha256:snap|sha256:audit|sha256:anchor"
	sum := sha256.Sum256([]byte(expectedSeed))
	hexExpected := "sha256:" + hex.EncodeToString(sum[:])
	if res.CombinedHash != hexExpected {
		t.Fatalf("combined hash mismatch expected %s got %s", hexExpected, res.CombinedHash)
	}
}

// durableReplay simulates a persistent replay store by retaining nonces across calls
type durableReplay struct{ seen map[string]struct{} }

func (d *durableReplay) Seen(nonce string) bool {
	if d.seen == nil {
		return false
	}
	_, ok := d.seen[nonce]
	return ok
}
func (d *durableReplay) Record(nonce string) {
	if d.seen == nil {
		d.seen = make(map[string]struct{})
	}
	d.seen[nonce] = struct{}{}
}

// Test tampering with snapshot hash after signing causes soft invalid signature
func TestVerifyModelLimitsAttestationTamperedSnapshot(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	att := &Attestation{Success: true, Configured: true, Nonce: "tamper1"}
	att.Snapshot.Hash = "sha256:orig"
	signAtt(t, priv, att)
	att.SigKid = "tkid"
	att.SigMode = sigModeEdDSA
	// Tamper after signing
	att.Snapshot.Hash = "sha256:modified"
	reg := &stubKeyRegistry{kid: "tkid", pub: pub}
	res, _ := VerifyModelLimitsAttestation(att, reg, &memoryReplay{}, time.Now())
	if res.Valid || !res.SoftInvalid || res.FailureCode != "signature_invalid" {
		t.Fatalf("expected soft invalid signature_invalid after tamper got %+v", res)
	}
}

// Test durable replay: first verification records nonce, second triggers replay failure
func TestVerifyModelLimitsAttestationDurableReplay(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	att := &Attestation{Success: true, Configured: true, Nonce: "durable-nonce"}
	att.Snapshot.Hash = testSnapshotHash
	signAtt(t, priv, att)
	att.SigKid = "dkid"
	att.SigMode = sigModeEdDSA
	reg := &stubKeyRegistry{kid: "dkid", pub: pub}
	dr := &durableReplay{}
	res1, _ := VerifyModelLimitsAttestation(att, reg, dr, time.Now())
	if !res1.Valid {
		t.Fatalf("expected first durable verification valid got %+v", res1)
	}
	res2, _ := VerifyModelLimitsAttestation(att, reg, dr, time.Now())
	if res2.FailureCode != "nonce_replay" || res2.HTTPStatus != 409 {
		t.Fatalf("expected nonce_replay 409 on second durable verify got %+v", res2)
	}
}
