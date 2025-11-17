package web

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestModelLimitAuditChain ensures audit entries are chained for input + rate exceed.
func TestModelLimitAuditChain(t *testing.T) {
	limitsFile, _ := os.CreateTemp(t.TempDir(), "limits.json")
	_, _ = limitsFile.WriteString(`{"model_limits":{"demo-model":{"max_input_tokens":10,"max_requests_per_minute":1}}}`)
	limitsFile.Close()
	auditFile, _ := os.CreateTemp(t.TempDir(), "audit.jsonl")
	auditFile.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })
	// exceed input
	doAuditReq(bs, map[string]any{"model_id": "demo-model", "input_tokens": 50})
	// exceed rate (first allowed, second exceeds)
	doAuditReq(bs, map[string]any{"model_id": "demo-model", "input_tokens": 5})
	doAuditReq(bs, map[string]any{"model_id": "demo-model", "input_tokens": 5})
	// read audit file
	f, _ := os.Open(auditFile.Name())
	scanner := bufio.NewScanner(f)
	var prev string
	count := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var entry struct {
			PrevHash string `json:"prev_hash"`
			Hash     string `json:"hash"`
		}
		if json.Unmarshal(line, &entry) != nil {
			t.Fatalf("invalid json line=%s", line)
		}
		if entry.PrevHash != prev {
			t.Fatalf("prev hash mismatch expected=%s got=%s", prev, entry.PrevHash)
		}
		prev = entry.Hash
		count++
	}
	if count < 2 {
		t.Fatalf("expected at least 2 audit entries got %d", count)
	}
}

func doAuditReq(bs *BetaServer, body map[string]any) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	bs.router.ServeHTTP(w, req)
	return w
}
