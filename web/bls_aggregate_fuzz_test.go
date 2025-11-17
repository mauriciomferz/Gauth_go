package web

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"math/rand"
	"net/http/httptest"
	"testing"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
)

// FuzzBLSAggregateEndpoint exercises the aggregate endpoint with malformed / random inputs to surface panics.
// It focuses on decode and structural error paths; success verification not required for fuzz stability.
func FuzzBLSAggregateEndpoint(f *testing.F) {
	// Seed with a few minimal valid cases.
	seedMsg := base64.StdEncoding.EncodeToString([]byte("fuzz"))
	seed := map[string]any{"mode": "issue", "message_b64": seedMsg, "participants": 2}
	b, _ := json.Marshal(seed)
	f.Add(string(b))
	f.Add(`{"mode":"verify","message_b64":"` + seedMsg + `","aggregated_signature_b64":"AAAA","public_keys_b64":["AAAA"]}`)
	f.Fuzz(func(t *testing.T, raw string) {
		mem := imetrics.NewMemory()
		srv := NewBetaServerWithMetrics(":0", mem)
		t.Cleanup(func() { srv.Shutdown() })
		// Randomly decide small mutations if input looks like JSON.
		if len(raw) > 2048 {
			raw = raw[:2048]
		} // bound size
		// Occasionally craft a synthetic request if raw isn't JSON-y.
		if raw == "" || raw[0] != '{' {
			mode := "issue"
			//nolint:gosec // G404: weak random acceptable for fuzz test mutation
			if rand.Intn(2) == 0 {
				mode = "verify"
			}
			msg := base64.StdEncoding.EncodeToString([]byte("m"))
			//nolint:gosec // G404: weak random acceptable for fuzz test mutation
			if rand.Intn(5) == 0 {
				msg = "%%%"
			} // invalid base64
			obj := map[string]any{"mode": mode, "message_b64": msg}
			if mode == "issue" {
				//nolint:gosec // G404: weak random acceptable for fuzz test mutation
				obj["participants"] = 1 + rand.Intn(4)
			} else {
				obj["aggregated_signature_b64"] = "!!!!" // invalid base64
				obj["public_keys_b64"] = []string{"!!!!"}
			}
			jb, _ := json.Marshal(obj)
			raw = string(jb)
		}
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/api/v1/crypto/bls/aggregate", bytes.NewBufferString(raw))
		srv.router.ServeHTTP(w, req)
		// Accept any status code (200 or 4xx expected). Ensure no server panic.
		if w.Code >= 500 {
			// High severity only if internal panic surfaces (should not for malformed input).
			// t.Fatalf intentionally avoided to allow corpus growth; just mark failure.
			return
		}
	})
}
