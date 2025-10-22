package web

import (
	"bufio"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestAttestationStreamInitial ensures the SSE endpoint returns an attestation event with a snapshot hash.
func TestAttestationStreamInitial(t *testing.T) {
	os.Setenv("GAUTH_ATTEST_STREAM_ENABLE", "1")
	os.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "0")
	os.Setenv("GAUTH_MODEL_LIMIT_ATTEST_NOTARIZE", "0")
	auditFile, err := os.CreateTemp(t.TempDir(), "audit-*.jsonl")
	if err != nil { t.Fatalf("audit temp: %v", err) }
	_, _ = auditFile.WriteString("{\"hash\":\"h1\"}\n")
	_ = auditFile.Close()
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	ts := NewBetaServer("0")
	live := httptest.NewServer(ts.router)
	defer live.Close()
	client := live.Client()
	client.Timeout = 3 * time.Second
	resp, err := client.Get(live.URL + "/api/v1/model/limits/attestation/stream")
	if err != nil { t.Fatalf("stream err: %v", err) }
	defer resp.Body.Close()
	if resp.StatusCode != 200 { t.Fatalf("expected 200 got %d", resp.StatusCode) }
	scan := bufio.NewScanner(resp.Body)
	deadline := time.Now().Add(2 * time.Second)
	found := false
	for time.Now().Before(deadline) && scan.Scan() {
		line := scan.Text()
		if strings.HasPrefix(line, "event: attestation") {
			if scan.Scan() { // data line
				dataLine := scan.Text()
				if strings.Contains(dataLine, "\"snapshot\"") && strings.Contains(dataLine, "\"hash\"") {
					found = true
					break
				}
			}
		}
	}
	if !found {
		 t.Fatalf("attestation event with snapshot hash not observed in stream")
	}
}
