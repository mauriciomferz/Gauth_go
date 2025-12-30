package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	imetrics "github.com/mauriciomferz/AgentAuth/internal/metrics"
)

// TestBLSAggregateIssueAndVerify exercises successful issuance and verification paths.
func TestBLSAggregateIssueAndVerify(t *testing.T) {
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	t.Cleanup(func() { srv.Shutdown() })
	msg := []byte("hello aggregated bls")
	msgB64 := base64.StdEncoding.EncodeToString(msg)

	// Issue aggregated signature for 5 participants
	issueReqBody := map[string]interface{}{
		"mode":         "issue",
		"message_b64":  msgB64,
		"participants": 5,
	}
	ib, err := json.Marshal(issueReqBody)
	if err != nil {
		t.Fatalf("marshal issue body: %v", err)
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(ib))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("issue expected 200 got %d body=%s", w.Code, w.Body.String())
	}
	var issueResp struct {
		Success                bool     `json:"success"`
		Mode                   string   `json:"mode"`
		AggregatedSignatureB64 string   `json:"aggregated_signature_b64"`
		PublicKeysB64          []string `json:"public_keys_b64"`
		ParticipantCount       int      `json:"participant_count"`
	}
	if err2 := json.Unmarshal(w.Body.Bytes(), &issueResp); err2 != nil {
		t.Fatalf("parse issue resp: %v", err2)
	}
	if !issueResp.Success || issueResp.Mode != "issue" {
		t.Fatalf("unexpected issue response: %+v", issueResp)
	}
	if issueResp.ParticipantCount != 5 || len(issueResp.PublicKeysB64) != 5 {
		t.Fatalf("expected 5 participants, got %d/%d", issueResp.ParticipantCount, len(issueResp.PublicKeysB64))
	}
	if issueResp.AggregatedSignatureB64 == "" {
		t.Fatalf("empty aggregated signature")
	}

	snapAfterIssue := mem.SnapshotEx()
	if snapAfterIssue.MultiSignatureBatchSizeCount == 0 {
		t.Fatalf("batch size count not recorded on issue")
	}
	if snapAfterIssue.MultiSignatureBatchSizeMax < 5 {
		t.Fatalf("batch size max < 5 got %d", snapAfterIssue.MultiSignatureBatchSizeMax)
	}
	if snapAfterIssue.MultiSignatureAggregateLatencyCount == 0 {
		t.Fatalf("aggregate latency count not recorded")
	}

	// Verify path using returned signature & keys
	verifyReqBody := map[string]interface{}{
		"mode":                     "verify",
		"message_b64":              msgB64,
		"aggregated_signature_b64": issueResp.AggregatedSignatureB64,
		"public_keys_b64":          issueResp.PublicKeysB64,
	}
	vb, err := json.Marshal(verifyReqBody)
	if err != nil {
		t.Fatalf("marshal verify body: %v", err)
	}
	wv := httptest.NewRecorder()
	reqv := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(vb))
	srv.router.ServeHTTP(wv, reqv)
	if wv.Code != 200 {
		t.Fatalf("verify expected 200 got %d body=%s", wv.Code, wv.Body.String())
	}
	var verifyResp struct {
		Success          bool   `json:"success"`
		Mode             string `json:"mode"`
		Valid            bool   `json:"valid"`
		ParticipantCount int    `json:"participant_count"`
	}
	if err := json.Unmarshal(wv.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("parse verify resp: %v", err)
	}
	if !verifyResp.Success || verifyResp.Mode != "verify" || !verifyResp.Valid {
		t.Fatalf("unexpected verify response: %+v", verifyResp)
	}
	if verifyResp.ParticipantCount != 5 {
		t.Fatalf("expected verify participant_count=5 got %d", verifyResp.ParticipantCount)
	}

	snapAfterVerify := mem.SnapshotEx()
	if snapAfterVerify.MultiSignatureVerifications != 1 {
		t.Fatalf("expected MultiSignatureVerifications=1 got %d", snapAfterVerify.MultiSignatureVerifications)
	}
	if snapAfterVerify.MultiSignatureVerificationFailures != 0 {
		t.Fatalf("unexpected verification failures: %d", snapAfterVerify.MultiSignatureVerificationFailures)
	}
	if snapAfterVerify.MultiSignatureAggregateLatencyCount < 2 {
		t.Fatalf("expected aggregate latency count >=2 got %d", snapAfterVerify.MultiSignatureAggregateLatencyCount)
	}
}

// TestBLSAggregateVerifyTamper ensures tampered aggregated signature fails verification and increments failure counter.
func TestBLSAggregateVerifyTamper(t *testing.T) {
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	t.Cleanup(func() { srv.Shutdown() })
	msgB64 := base64.StdEncoding.EncodeToString([]byte("tamper case"))
	// Issue
	ib, _ := json.Marshal(map[string]interface{}{"mode": "issue", "message_b64": msgB64, "participants": 3})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(ib))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("issue expected 200 got %d", w.Code)
	}
	var ir struct {
		AggregatedSignatureB64 string   `json:"aggregated_signature_b64"`
		PublicKeysB64          []string `json:"public_keys_b64"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &ir); err != nil {
		t.Fatalf("parse issue resp: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(ir.AggregatedSignatureB64)
	if err != nil {
		t.Fatalf("decode agg sig: %v", err)
	}
	if len(raw) == 0 {
		t.Fatalf("empty agg sig bytes")
	}
	raw[0] ^= 0xFF // mutate
	tampered := base64.StdEncoding.EncodeToString(raw)
	vb, _ := json.Marshal(map[string]interface{}{"mode": "verify", "message_b64": msgB64, "aggregated_signature_b64": tampered, "public_keys_b64": ir.PublicKeysB64})
	wv := httptest.NewRecorder()
	reqv := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(vb))
	srv.router.ServeHTTP(wv, reqv)
	switch wv.Code {
	case 200:
		var vr struct {
			Valid bool `json:"valid"`
		}
		if err := json.Unmarshal(wv.Body.Bytes(), &vr); err != nil {
			t.Fatalf("parse verify resp: %v", err)
		}
		if vr.Valid {
			t.Fatalf("expected invalid after tamper")
		}
		snap := mem.SnapshotEx()
		if snap.MultiSignatureVerificationFailures == 0 {
			t.Fatalf("expected verification failure counter increment")
		}
	case 400:
		// Accept structural failure (deserialize) - metrics not incremented in this path.
		// No assertion on counters; ensure error marker present.
		body := wv.Body.String()
		if !(strings.Contains(body, "aggregated_signature_deserialize_failed") || strings.Contains(body, "aggregated_signature_decode_failed")) {
			t.Fatalf("expected deserialize or decode failure marker in body=%s", body)
		}
	default:
		t.Fatalf("unexpected status code %d body=%s", wv.Code, wv.Body.String())
	}
}

// TestBLSAggregateBadMode validates invalid mode handling.
func TestBLSAggregateBadMode(t *testing.T) {
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	t.Cleanup(func() { srv.Shutdown() })
	msgB64 := base64.StdEncoding.EncodeToString([]byte("x"))
	badReq := map[string]interface{}{"mode": "foo", "message_b64": msgB64, "participants": 1}
	b, _ := json.Marshal(badReq)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(b))
	srv.router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 invalid mode got %d", w.Code)
	}
}

// TestBLSAggregateMissingMessage ensures missing message field is rejected.
func TestBLSAggregateMissingMessage(t *testing.T) {
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	t.Cleanup(func() { srv.Shutdown() })
	badReq := map[string]interface{}{"mode": "issue", "participants": 2}
	b, _ := json.Marshal(badReq)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(b))
	srv.router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 missing_message got %d", w.Code)
	}
}

// Basic duration sanity check ensuring latency counters advance (non-zero) after issue + verify sequence.
func TestBLSAggregateLatencyAdvances(t *testing.T) {
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	t.Cleanup(func() { srv.Shutdown() })
	msgB64 := base64.StdEncoding.EncodeToString([]byte("latency check"))
	// issue
	ib, _ := json.Marshal(map[string]interface{}{"mode": "issue", "message_b64": msgB64, "participants": 2})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(ib))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("issue expected 200 got %d", w.Code)
	}
	var ir struct {
		AggregatedSignatureB64 string   `json:"aggregated_signature_b64"`
		PublicKeysB64          []string `json:"public_keys_b64"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &ir); err != nil {
		t.Fatalf("parse issue resp: %v", err)
	}
	// verify
	vb, _ := json.Marshal(map[string]interface{}{"mode": "verify", "message_b64": msgB64, "aggregated_signature_b64": ir.AggregatedSignatureB64, "public_keys_b64": ir.PublicKeysB64})
	wv := httptest.NewRecorder()
	reqv := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(vb))
	start := time.Now()
	srv.router.ServeHTTP(wv, reqv)
	if wv.Code != 200 {
		t.Fatalf("verify expected 200 got %d", wv.Code)
	}
	_ = start // placeholder in case future latency assertions compare wall-clock
	snap := mem.SnapshotEx()
	if snap.MultiSignatureAggregateLatencyCount < 2 {
		t.Fatalf("expected >=2 aggregate latency samples got %d", snap.MultiSignatureAggregateLatencyCount)
	}
	if snap.MultiSignatureAggregateLatencyTotalNS == 0 {
		t.Fatalf("expected non-zero aggregate latency total")
	}
}

// TestBLSAggregatePublicKeyDecodeFailure ensures invalid base64 for public key yields 400 public_key_decode_failed.
func TestBLSAggregatePublicKeyDecodeFailure(t *testing.T) {
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	t.Cleanup(func() { srv.Shutdown() })
	msgB64 := base64.StdEncoding.EncodeToString([]byte("decode fail"))
	// Issue a valid aggregated signature with 2 participants to get a signature to reuse.
	ib, _ := json.Marshal(map[string]interface{}{"mode": "issue", "message_b64": msgB64, "participants": 2})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(ib))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("issue expected 200 got %d", w.Code)
	}
	var ir struct {
		AggregatedSignatureB64 string   `json:"aggregated_signature_b64"`
		PublicKeysB64          []string `json:"public_keys_b64"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &ir); err != nil {
		t.Fatalf("parse issue resp: %v", err)
	}
	// Corrupt first public key with invalid base64 characters.
	badKeys := append([]string{"!!!!"}, ir.PublicKeysB64[1:]...)
	vb, _ := json.Marshal(map[string]interface{}{"mode": "verify", "message_b64": msgB64, "aggregated_signature_b64": ir.AggregatedSignatureB64, "public_keys_b64": badKeys})
	wv := httptest.NewRecorder()
	reqv := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(vb))
	srv.router.ServeHTTP(wv, reqv)
	if wv.Code != 400 {
		t.Fatalf("expected 400 for public_key_decode_failed got %d body=%s", wv.Code, wv.Body.String())
	}
	if !strings.Contains(wv.Body.String(), "public_key_decode_failed") {
		t.Fatalf("expected public_key_decode_failed marker, body=%s", wv.Body.String())
	}
}

// TestBLSAggregateAggregatedSignatureDecodeFailure ensures invalid base64 aggregated signature returns 400 aggregated_signature_decode_failed.
func TestBLSAggregateAggregatedSignatureDecodeFailure(t *testing.T) {
	mem := imetrics.NewMemory()
	srv := NewBetaServerWithMetrics(":0", mem)
	t.Cleanup(func() { srv.Shutdown() })
	msgB64 := base64.StdEncoding.EncodeToString([]byte("sig decode fail"))
	// Issue to obtain public keys (ignore signature returned; we'll replace with invalid string).
	ib, _ := json.Marshal(map[string]interface{}{"mode": "issue", "message_b64": msgB64, "participants": 3})
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(ib))
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("issue expected 200 got %d", w.Code)
	}
	var ir struct {
		PublicKeysB64 []string `json:"public_keys_b64"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &ir); err != nil {
		t.Fatalf("parse issue resp: %v", err)
	}
	vb, _ := json.Marshal(map[string]interface{}{"mode": "verify", "message_b64": msgB64, "aggregated_signature_b64": "!!!", "public_keys_b64": ir.PublicKeysB64})
	wv := httptest.NewRecorder()
	reqv := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewReader(vb))
	srv.router.ServeHTTP(wv, reqv)
	if wv.Code != 400 {
		t.Fatalf("expected 400 aggregated_signature_decode_failed got %d body=%s", wv.Code, wv.Body.String())
	}
	if !strings.Contains(wv.Body.String(), "aggregated_signature_decode_failed") {
		t.Fatalf("expected aggregated_signature_decode_failed marker, body=%s", wv.Body.String())
	}
}
