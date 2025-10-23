package web

import (
	"bufio"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestAttestationStreamAuditTrigger ensures audit append causes an attestation emission with reason=audit_append.
func TestAttestationStreamAuditTrigger(t *testing.T) {
	os.Setenv("GAUTH_ATTEST_STREAM_ENABLE", "1")
	os.Setenv("GAUTH_MODEL_LIMIT_ATTEST_SIGN", "0")
	os.Setenv("GAUTH_MODEL_LIMIT_ATTEST_NOTARIZE", "0")
	// Prepare temp audit path
	auditFile, err := os.CreateTemp(t.TempDir(), "audit-*.jsonl")
	if err != nil {
		t.Fatalf("audit temp: %v", err)
	}
	// seed one entry so configured=true
	_, _ = auditFile.WriteString("{\"hash\":\"h0\"}\n")
	_ = auditFile.Close()
	os.Setenv("GAUTH_MODEL_LIMIT_AUDIT_PATH", auditFile.Name())
	// Create server
	ts := NewBetaServer("0")
	live := httptest.NewServer(ts.router)
	defer live.Close()
	// Open stream
	resp, err := live.Client().Get(live.URL + "/api/v1/model/limits/attestation/stream")
	if err != nil {
		t.Fatalf("stream open: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	scan := bufio.NewScanner(resp.Body)
	deadline := time.Now().Add(3 * time.Second)
	foundOpen := false
	for time.Now().Before(deadline) && scan.Scan() {
		l := scan.Text()
		if strings.HasPrefix(l, "event: attestation") {
			if scan.Scan() {
				data := scan.Text()
				if strings.Contains(data, "\"reason\":\"open\"") {
					foundOpen = true
					break
				}
			}
		}
	}
	if !foundOpen {
		t.Fatalf("did not see initial attestation with reason open")
	}
	// Append an audit exceed event by calling writeModelLimitAudit through validation path causing exceed (simulate input limit exceed)
	// Simplify by directly invoking writeModelLimitAudit (acceptable for unit test) for a new model id.
	ts.writeModelLimitAudit("m1", "input", 999, 10, 0, 0, "")
	// Scan for audit_append emission
	deadline2 := time.Now().Add(3 * time.Second)
	foundAudit := false
	for time.Now().Before(deadline2) && scan.Scan() {
		l := scan.Text()
		if strings.HasPrefix(l, "event: attestation") {
			if scan.Scan() {
				data := scan.Text()
				if strings.Contains(data, "\"reason\":\"audit_append\"") {
					foundAudit = true
					break
				}
			}
		}
	}
	if !foundAudit {
		t.Fatalf("did not observe audit_append attestation event")
	}
}
