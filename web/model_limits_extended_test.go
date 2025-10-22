package web

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// TestModelExtendedLimits validates output token and rate limiting enforcement.
func TestModelExtendedLimits(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "model_limits_ext_*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	limitsJSON := `{"model_limits":{"demo-model":{"max_input_tokens":200,"max_output_tokens":150,"max_requests_per_minute":5}}}`
	if _, err := f.Write([]byte(limitsJSON)); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", f.Name())
	bs := NewBetaServer("")
	// 1. Within all limits
	resp := doModelReq(bs, map[string]any{"model_id": "demo-model", "input_tokens": 50, "output_tokens": 75})
	if resp.Code != 200 {
		t.Fatalf("expected 200 within limits got %d body=%s", resp.Code, resp.Body.String())
	}
	// 2. Output tokens exceed
	resp = doModelReq(bs, map[string]any{"model_id": "demo-model", "input_tokens": 50, "output_tokens": 200})
	if resp.Code != 400 || !bytes.Contains(resp.Body.Bytes(), []byte("model_output_limit_exceeded")) {
		t.Fatalf("expected output exceed 400 got %d body=%s", resp.Code, resp.Body.String())
	}
	// 3. Input tokens exceed (ensure distinct path)
	resp = doModelReq(bs, map[string]any{"model_id": "demo-model", "input_tokens": 500, "output_tokens": 10})
	if resp.Code != 400 || !bytes.Contains(resp.Body.Bytes(), []byte("model_limit_exceeded")) {
		t.Fatalf("expected input exceed 400 got %d body=%s", resp.Code, resp.Body.String())
	}
	// 4. Rate limiting: perform 2 allowed then exceed on configured limit=3 for demonstration (set lower limit explicitly)
	// Reconfigure internal rate limit map for test determinism
	bs.modelRateMu.Lock()
	bs.modelRateLimits["demo-model"] = 3
	bs.modelRateMu.Unlock()
	for i := 0; i < 3; i++ {
		r := doModelReq(bs, map[string]any{"model_id": "demo-model", "input_tokens": 10, "output_tokens": 10})
		if i < 2 && r.Code != 200 {
			t.Fatalf("setup rate allow #%d failed code=%d body=%s", i, r.Code, r.Body.String())
		}
		if i == 2 && r.Code != 429 {
			t.Fatalf("expected 3rd request rate limited code=%d body=%s", r.Code, r.Body.String())
		}
	}
	// 5. Wait window reset and confirm allowed again
	// Force window reset by advancing internal state (sleep slightly >1s then manually call enough to exceed minute if logic broken)
	// Sleep beyond a minute boundary not feasible in test; instead manually reset internal window to simulate expiry.
	bs.modelRateStateMu.Lock()
	if st, ok := bs.modelRateState["demo-model"]; ok {
		st.WindowStart = time.Now().Add(-time.Minute)
		st.Count = 0
		bs.modelRateState["demo-model"] = st
	}
	bs.modelRateStateMu.Unlock()
	resp = doModelReq(bs, map[string]any{"model_id": "demo-model", "input_tokens": 10, "output_tokens": 10})
	// Could still be in same window if <60s; we only verify not rate limited immediately after short wait (should still exceed if window unchanged and count>limit). Accept 200.
	if resp.Code == 429 {
		t.Fatalf("unexpected immediate rate limit after short wait body=%s", resp.Body.String())
	}
}

func doModelReq(bs *BetaServer, body map[string]any) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	return w
}
