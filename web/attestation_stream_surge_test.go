package web

import (
	"bufio"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestAttestationStreamSurgeTrigger forces a surge condition and expects a surge_trigger attestation.
func TestAttestationStreamSurgeTrigger(t *testing.T) {
	os.Setenv("GAUTH_ATTEST_STREAM_ENABLE", "1")
	os.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "0")
	os.Setenv("GAUTH_MODEL_LIMIT_ATTEST_NOTARIZE", "0")
	// Lower surge thresholds for deterministic trigger
	os.Setenv("GAUTH_MODEL_LIMIT_SURGE_FACTOR", "0.5")
	os.Setenv("GAUTH_MODEL_LIMIT_SURGE_MIN_EVENTS", "2")
	// Prepare audit file (required for configured=true)
	auditFile, err := os.CreateTemp(t.TempDir(), "audit-*.jsonl")
	if err != nil { t.Fatalf("audit temp: %v", err) }
	_, _ = auditFile.WriteString("{\"hash\":\"h0\"}\n")
	_ = auditFile.Close()
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	// Server
	ts := NewBetaServer("0")
	live := httptest.NewServer(ts.router)
	defer live.Close()
	resp, err := live.Client().Get(live.URL + "/api/v1/model/limits/attestation/stream")
	if err != nil { t.Fatalf("stream open: %v", err) }
	defer resp.Body.Close()
	scan := bufio.NewScanner(resp.Body)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) && scan.Scan() { // consume initial open attestation
		if strings.HasPrefix(scan.Text(), "event: attestation") {
			_ = scan.Scan() // data line discard
			break
		}
	}
	// Fire enough exceed events to meet surge (simulate input exceed repeatedly)
	for i := 0; i < 3; i++ { // 3 events exceeds min_events=2 and factor baseline
		ts.writeModelLimitAudit("surge-model", "input", 500+i, 10, 0, 0, "")
	}
	deadline2 := time.Now().Add(3 * time.Second)
	foundSurge := false
	for time.Now().Before(deadline2) && scan.Scan() {
		l := scan.Text()
		if strings.HasPrefix(l, "event: attestation") {
			if scan.Scan() {
				data := scan.Text()
				if strings.Contains(data, "\"reason\":\"surge_trigger\"") { foundSurge = true; break }
			}
		}
	}
	if !foundSurge { t.Fatalf("did not observe surge_trigger attestation event") }
}
