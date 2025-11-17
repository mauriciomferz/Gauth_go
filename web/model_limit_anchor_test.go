package web

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestModelLimitAuditAnchoring ensures periodic anchor entries are created and the chain verifies.
func TestModelLimitAuditAnchoring(t *testing.T) {
	// Prepare temp files
	limitsFile, err := os.CreateTemp(t.TempDir(), "model_limits_*.json")
	if err != nil {
		t.Fatalf("temp limits: %v", err)
	}
	// Very small input limit to trigger exceed easily
	limitsJSON := `{"model_limits":{"anchor-model":{"max_input_tokens":10}},"user_limits":{}}`
	if _, err2 := limitsFile.Write([]byte(limitsJSON)); err2 != nil {
		t.Fatalf("write limits: %v", err2)
	}
	limitsFile.Close()
	auditFile, err := os.CreateTemp(t.TempDir(), "model_limit_audit_*.jsonl")
	if err != nil {
		t.Fatalf("temp audit: %v", err)
	}
	auditFile.Close()
	anchorFile, err := os.CreateTemp(t.TempDir(), "model_limit_anchor_*.jsonl")
	if err != nil {
		t.Fatalf("temp anchor: %v", err)
	}
	anchorFile.Close()
	os.Setenv("GAUTH_MODEL_LIMITS_PATH", limitsFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_PATH", anchorFile.Name())
	os.Setenv("GAUTH_MODEL_LIMIT_ANCHOR_INTERVAL", "2") // anchor every 2 audit entries

	bs := NewBetaServer("")
	t.Cleanup(func() { bs.Shutdown() })

	exceed := func() {
		w := httptest.NewRecorder()
		body := map[string]any{"model_id": "anchor-model", "input_tokens": 50, "output_tokens": 1}
		b, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/model/validate", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		bs.router.ServeHTTP(w, req)
		if w.Code != 400 {
			t.Fatalf("expected 400 exceed got %d body=%s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "model_limit_exceeded") {
			t.Fatalf("unexpected body: %s", w.Body.String())
		}
	}

	// Two exceeds -> two audit entries -> one anchor
	exceed()
	exceed()

	// Read anchor file
	data, err := os.ReadFile(anchorFile.Name())
	if err != nil {
		t.Fatalf("read anchor: %v", err)
	}
	lines := bufio.NewScanner(bytes.NewReader(data))
	var lineCount int
	var first string
	for lines.Scan() {
		ln := strings.TrimSpace(lines.Text())
		if ln == "" {
			continue
		}
		if first == "" {
			first = ln
		}
		lineCount++
	}
	if lineCount != 1 {
		t.Fatalf("expected 1 anchor entry got %d", lineCount)
	}
	if !strings.Contains(first, "\"audit_entries\":2") {
		t.Fatalf("anchor missing audit_entries=2: %s", first)
	}
	if !strings.Contains(first, "audit_last_hash") {
		t.Fatalf("anchor missing last hash field: %s", first)
	}

	// Verify endpoint
	vw := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/api/v1/model/limits/audit/anchor/verify", nil)
	bs.router.ServeHTTP(vw, req)
	if vw.Code != 200 {
		t.Fatalf("verify endpoint status %d body=%s", vw.Code, vw.Body.String())
	}
	if !strings.Contains(vw.Body.String(), "\"valid\":true") {
		t.Fatalf("verify endpoint invalid: %s", vw.Body.String())
	}
}
