package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	bls "github.com/herumi/bls-eth-go-binary/bls"
)

// TestBLSPoPIssueAndVerify exercises PoP challenge issuance via require_pop and subsequent verification.
func TestBLSPoPIssueAndVerify(t *testing.T) {
	os.Setenv("GAUTH_ALLOW_POP_PRIV_EXPORT", "1")
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	msgB64 := base64.StdEncoding.EncodeToString([]byte("pop flow"))
	issueReq := map[string]interface{}{"mode": "issue", "message_b64": msgB64, "participants": 3, "require_pop": true}
	ib, _ := json.Marshal(issueReq)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(ib))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("issue_pop expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var ir struct {
		Mode             string   `json:"mode"`
		PublicKeysB64    []string `json:"public_keys_b64"`
		PrivateKeysB64   []string `json:"private_keys_b64"`
		ChallengesB64    []string `json:"challenges_b64"`
		ParticipantCount int      `json:"participant_count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &ir); err != nil {
		t.Fatalf("parse issue_pop resp: %v", err)
	}
	if ir.Mode != "issue_pop" || ir.ParticipantCount != 3 {
		t.Fatalf("unexpected pop issue resp: %+v", ir)
	}
	if len(ir.PrivateKeysB64) != 3 || len(ir.PublicKeysB64) != 3 || len(ir.ChallengesB64) != 3 {
		t.Fatalf("expected 3 keys/challenges")
	}

	// Sign each challenge
	if err := bls.Init(bls.BLS12_381); err != nil {
		t.Fatalf("bls init: %v", err)
	}
	pairs := make([]map[string]string, 0, 3)
	for i := 0; i < 3; i++ {
		pkRaw, err := base64.StdEncoding.DecodeString(ir.PublicKeysB64[i])
		if err != nil {
			t.Fatalf("decode pub: %v", err)
		}
		var sk bls.SecretKey
		privRaw, err := base64.StdEncoding.DecodeString(ir.PrivateKeysB64[i])
		if err != nil {
			t.Fatalf("decode priv: %v", err)
		}
		if err := sk.Deserialize(privRaw); err != nil {
			t.Fatalf("deserialize priv: %v", err)
		}
		chRaw, err := base64.StdEncoding.DecodeString(ir.ChallengesB64[i])
		if err != nil {
			t.Fatalf("decode challenge: %v", err)
		}
		sig := sk.SignByte(chRaw)
		sigB64 := base64.StdEncoding.EncodeToString(sig.Serialize())
		_ = pkRaw // not needed directly here; verify endpoint will deserialize from provided public key
		pairs = append(pairs, map[string]string{"public_key_b64": ir.PublicKeysB64[i], "signature_b64": sigB64, "challenge_b64": ir.ChallengesB64[i]})
	}
	verifyReq := map[string]interface{}{"pairs": pairs}
	vb, _ := json.Marshal(verifyReq)
	wv := httptest.NewRecorder()
	reqv := httptest.NewRequest("POST", "/api/v1/crypto/bls/pop/verify", bytes.NewReader(vb))
	srv.router.ServeHTTP(wv, reqv)
	if wv.Code != 200 {
		t.Fatalf("pop verify expected 200 got %d body=%s", wv.Code, wv.Body.String())
	}
	var vr struct {
		Success  bool `json:"success"`
		Valid    bool `json:"valid"`
		Failures int  `json:"failures"`
	}
	if err := json.Unmarshal(wv.Body.Bytes(), &vr); err != nil {
		t.Fatalf("parse verify pop resp: %v", err)
	}
	if !vr.Success || !vr.Valid || vr.Failures != 0 {
		t.Fatalf("unexpected pop verify resp: %+v", vr)
	}
	snap := mem.SnapshotEx()
	if snap.BLSPoPChallengesIssued < 3 {
		t.Fatalf("expected BLSPoPChallengesIssued>=3 got %d", snap.BLSPoPChallengesIssued)
	}
	if snap.BLSPoPVerifications != 3 {
		t.Fatalf("expected BLSPoPVerifications=3 got %d", snap.BLSPoPVerifications)
	}
	if snap.BLSPoPVerificationFailures != 0 {
		t.Fatalf("expected BLSPoPVerificationFailures=0 got %d", snap.BLSPoPVerificationFailures)
	}
}

// TestBLSPoPFailure modifies a signature to force verification failure.
func TestBLSPoPFailure(t *testing.T) {
	os.Setenv("GAUTH_ALLOW_POP_PRIV_EXPORT", "1")
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	msgB64 := base64.StdEncoding.EncodeToString([]byte("pop fail"))
	issueReq := map[string]interface{}{"mode": "issue", "message_b64": msgB64, "participants": 2, "require_pop": true}
	ib, _ := json.Marshal(issueReq)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(ib))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("issue_pop expected 200 got %d", w.Code)
	}
	var ir struct {
		PublicKeysB64  []string `json:"public_keys_b64"`
		PrivateKeysB64 []string `json:"private_keys_b64"`
		ChallengesB64  []string `json:"challenges_b64"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &ir); err != nil {
		t.Fatalf("parse issue_pop resp: %v", err)
	}
	if err := bls.Init(bls.BLS12_381); err != nil {
		t.Fatalf("bls init: %v", err)
	}
	pairs := make([]map[string]string, 0, 2)
	for i := 0; i < 2; i++ {
		var sk bls.SecretKey
		privRaw, _ := base64.StdEncoding.DecodeString(ir.PrivateKeysB64[i])
		if err := sk.Deserialize(privRaw); err != nil {
			t.Fatalf("deserialize priv: %v", err)
		}
		chRaw, _ := base64.StdEncoding.DecodeString(ir.ChallengesB64[i])
		sig := sk.SignByte(chRaw)
		sigRaw := sig.Serialize()
		if i == 1 { // tamper second signature
			sigRaw[0] ^= 0xFF
		}
		pairs = append(pairs, map[string]string{"public_key_b64": ir.PublicKeysB64[i], "signature_b64": base64.StdEncoding.EncodeToString(sigRaw), "challenge_b64": ir.ChallengesB64[i]})
	}
	verifyReq := map[string]interface{}{"pairs": pairs}
	vb, _ := json.Marshal(verifyReq)
	wv := httptest.NewRecorder()
	reqv := httptest.NewRequest("POST", "/api/v1/crypto/bls/pop/verify", bytes.NewReader(vb))
	srv.router.ServeHTTP(wv, reqv)
	if wv.Code != 200 {
		t.Fatalf("pop verify expected 200 got %d", wv.Code)
	}
	var vr struct {
		Valid    bool `json:"valid"`
		Failures int  `json:"failures"`
	}
	if err := json.Unmarshal(wv.Body.Bytes(), &vr); err != nil {
		t.Fatalf("parse verify pop resp: %v", err)
	}
	if vr.Valid || vr.Failures == 0 {
		t.Fatalf("expected failures >0 resp=%+v", vr)
	}
	snap := mem.SnapshotEx()
	if snap.BLSPoPVerificationFailures == 0 {
		t.Fatalf("expected BLSPoPVerificationFailures >0")
	}
}

// TestBLSPoPEdgeCases covers malformed base64 inputs, empty pairs, missing fields, duplicate public keys, and mismatched challenge/signature pairs.
func TestBLSPoPEdgeCases(t *testing.T) {
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	// 1. Empty pairs
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/pop/verify", bytes.NewReader([]byte(`{"pairs":[]}`)))
	srv.router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("empty pairs expected 400 got %d body=%s", w.Code, w.Body.String())
	}

	// 2. Malformed JSON base64 (invalid char) in public_key_b64
	bad := `{"pairs":[{"public_key_b64":"@@@","signature_b64":"@@@","challenge_b64":"@@@"}]}`
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/api/v1/crypto/bls/pop/verify", bytes.NewReader([]byte(bad)))
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("malformed base64 should still 200 processing failures counted got %d", w2.Code)
	}
	var r2 struct {
		Success  bool `json:"success"`
		Valid    bool `json:"valid"`
		Failures int  `json:"failures"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &r2)
	if r2.Failures == 0 || r2.Valid {
		t.Fatalf("expected failures>0 valid=false r2=%+v", r2)
	}

	// Issue proper PoP to craft duplicate public key scenario
	os.Setenv("GAUTH_ALLOW_POP_PRIV_EXPORT", "1")
	msgB64 := base64.StdEncoding.EncodeToString([]byte("edge case msg"))
	issueReq := map[string]interface{}{"mode": "issue", "message_b64": msgB64, "participants": 2, "require_pop": true}
	ib, _ := json.Marshal(issueReq)
	wi := httptest.NewRecorder()
	reqi := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(ib))
	srv.router.ServeHTTP(wi, reqi)
	if wi.Code != 200 {
		t.Fatalf("issue_pop expected 200 got %d", wi.Code)
	}
	var ir struct {
		PublicKeysB64  []string `json:"public_keys_b64"`
		PrivateKeysB64 []string `json:"private_keys_b64"`
		ChallengesB64  []string `json:"challenges_b64"`
	}
	if err := json.Unmarshal(wi.Body.Bytes(), &ir); err != nil {
		t.Fatalf("parse issue_pop resp: %v", err)
	}
	if len(ir.PublicKeysB64) != 2 {
		t.Fatalf("expected 2 public keys")
	}
	if err := bls.Init(bls.BLS12_381); err != nil {
		t.Fatalf("bls init: %v", err)
	}
	// Duplicate public key: use first key for both pairs
	var sk1 bls.SecretKey
	priv1Raw, _ := base64.StdEncoding.DecodeString(ir.PrivateKeysB64[0])
	if err := sk1.Deserialize(priv1Raw); err != nil {
		t.Fatalf("deserialize priv1: %v", err)
	}
	ch0Raw, _ := base64.StdEncoding.DecodeString(ir.ChallengesB64[0])
	ch1Raw, _ := base64.StdEncoding.DecodeString(ir.ChallengesB64[1])
	sig0 := sk1.SignByte(ch0Raw).Serialize()
	sig1 := sk1.SignByte(ch1Raw).Serialize()
	pairsDup := []map[string]string{
		{"public_key_b64": ir.PublicKeysB64[0], "signature_b64": base64.StdEncoding.EncodeToString(sig0), "challenge_b64": ir.ChallengesB64[0]},
		{"public_key_b64": ir.PublicKeysB64[0], "signature_b64": base64.StdEncoding.EncodeToString(sig1), "challenge_b64": ir.ChallengesB64[1]},
	}
	vbDup, _ := json.Marshal(map[string]interface{}{"pairs": pairsDup})
	wDup := httptest.NewRecorder()
	reqDup := httptest.NewRequest("POST", "/api/v1/crypto/bls/pop/verify", bytes.NewReader(vbDup))
	srv.router.ServeHTTP(wDup, reqDup)
	if wDup.Code != 200 {
		t.Fatalf("dup pub verify expected 200 got %d", wDup.Code)
	}
	var vrDup struct {
		Valid    bool `json:"valid"`
		Failures int  `json:"failures"`
	}
	if err := json.Unmarshal(wDup.Body.Bytes(), &vrDup); err != nil {
		t.Fatalf("parse dup verify: %v", err)
	}
	// Duplicate keys still may verify individually; treat as allowed but ensure metrics increment for successes
	snap := mem.SnapshotEx()
	if snap.BLSPoPVerifications < 2 {
		t.Fatalf("expected >=2 verifications got %d", snap.BLSPoPVerifications)
	}

	// Mismatched challenge: tamper second challenge
	var sk2 bls.SecretKey
	priv2Raw, _ := base64.StdEncoding.DecodeString(ir.PrivateKeysB64[1])
	if err := sk2.Deserialize(priv2Raw); err != nil {
		t.Fatalf("deserialize priv2: %v", err)
	}
	tampered := append([]byte(nil), ch1Raw...)
	if len(tampered) > 0 {
		tampered[0] ^= 0xAA
	}
	sigTampered := sk2.SignByte(tampered).Serialize()
	pairsMismatch := []map[string]string{
		{"public_key_b64": ir.PublicKeysB64[1], "signature_b64": base64.StdEncoding.EncodeToString(sigTampered), "challenge_b64": ir.ChallengesB64[1]},
	}
	vbMis, _ := json.Marshal(map[string]interface{}{"pairs": pairsMismatch})
	wMis := httptest.NewRecorder()
	reqMis := httptest.NewRequest("POST", "/api/v1/crypto/bls/pop/verify", bytes.NewReader(vbMis))
	srv.router.ServeHTTP(wMis, reqMis)
	if wMis.Code != 200 {
		t.Fatalf("mismatch verify expected 200 got %d", wMis.Code)
	}
	var vrMis struct {
		Valid    bool `json:"valid"`
		Failures int  `json:"failures"`
	}
	if err := json.Unmarshal(wMis.Body.Bytes(), &vrMis); err != nil {
		t.Fatalf("parse mismatch verify: %v", err)
	}
	if vrMis.Valid || vrMis.Failures == 0 {
		t.Fatalf("expected mismatch failure resp=%+v", vrMis)
	}

	// Missing field (challenge_b64 omitted) -> decoding failure counted
	missingField := `{"pairs":[{"public_key_b64":"` + ir.PublicKeysB64[0] + `","signature_b64":"` + base64.StdEncoding.EncodeToString(sig0) + `"}]}`
	wMF := httptest.NewRecorder()
	reqMF := httptest.NewRequest("POST", "/api/v1/crypto/bls/pop/verify", bytes.NewReader([]byte(missingField)))
	srv.router.ServeHTTP(wMF, reqMF)
	if wMF.Code != 200 {
		t.Fatalf("missing field verify expected 200 got %d", wMF.Code)
	}
	var vrMF struct {
		Valid    bool `json:"valid"`
		Failures int  `json:"failures"`
	}
	_ = json.Unmarshal(wMF.Body.Bytes(), &vrMF)
	if vrMF.Valid || vrMF.Failures == 0 {
		t.Fatalf("expected failures due to missing field resp=%+v", vrMF)
	}
}
