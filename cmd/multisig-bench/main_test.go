package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// runHarness executes `go run .` in the multisig-bench directory with the given args
// and returns stdout lines (non-empty) and stderr (full string).
func runHarness(t *testing.T, args ...string) ([]string, string) {
	t.Helper()
	cmdArgs := append([]string{"run", "."}, args...)
	cmd := exec.Command("go", cmdArgs...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("harness run failed: %v\nstderr: %s", err, stderr.String())
	}
	// Split into lines; filter out empty lines.
	scanner := bufio.NewScanner(&stdout)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scanner error: %v", err)
	}
	return lines, stderr.String()
}

// parseRecords converts JSONL lines into a slice of record maps.
func parseRecords(t *testing.T, lines []string) []map[string]any {
	t.Helper()
	out := make([]map[string]any, 0, len(lines))
	for i, l := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(l), &m); err != nil {
			t.Fatalf("line %d is not valid JSON: %s (err=%v)", i, l, err)
		}
		out = append(out, m)
	}
	return out
}

// TestMultiSigBenchBasicBoth ensures basic output shape for mode=both.
func TestMultiSigBenchBasicBoth(t *testing.T) {
	lines, stderr := runHarness(t,
		"--signers", "1,2",
		"--iterations", "5",
		"--mode", "both",
		"--seed", "12345",
	)
	if len(stderr) != 0 {
		// Non-fatal: just surface unexpected stderr content.
		t.Logf("stderr: %s", stderr)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSON lines (one per signer group); got %d", len(lines))
	}
	recs := parseRecords(t, lines)
	for _, r := range recs {
		// Required keys
		for _, k := range []string{"signers", "iterations", "mode"} {
			if _, ok := r[k]; !ok {
				t.Fatalf("missing key %q in record: %#v", k, r)
			}
		}
		// Check averages > 0 for both signing & verifying.
		signNS, okSign := r["avg_sign_ns"].(float64)
		verifyNS, okVerify := r["avg_verify_ns"].(float64)
		if !okSign || signNS <= 0 {
			t.Fatalf("avg_sign_ns missing or non-positive: %v", r)
		}
		if !okVerify || verifyNS <= 0 {
			t.Fatalf("avg_verify_ns missing or non-positive: %v", r)
		}
	}
}

// TestMultiSigBenchVerifyOnly ensures signing metrics are omitted in verify-only mode.
func TestMultiSigBenchVerifyOnly(t *testing.T) {
	lines, _ := runHarness(t,
		"--signers", "2,3",
		"--iterations", "3",
		"--mode", "verify",
		"--seed", "999",
	)
	if len(lines) != 2 {
		t.Fatalf("expected 2 JSON lines; got %d", len(lines))
	}
	recs := parseRecords(t, lines)
	for _, r := range recs {
		if r["mode"] != "verify" {
			t.Fatalf("expected mode verify; got %v", r["mode"])
		}
		if _, hasSign := r["avg_sign_ns"]; hasSign {
			t.Fatalf("avg_sign_ns should be omitted in verify-only mode: %#v", r)
		}
		verifyNS, okVerify := r["avg_verify_ns"].(float64)
		if !okVerify || verifyNS <= 0 {
			t.Fatalf("avg_verify_ns missing or non-positive: %v", r)
		}
	}
}

// TestMultiSigBenchMetricsFlag ensures that enabling --metrics still yields JSON lines.
// We don't assert detailed metrics content (stderr) yet; just that output remains valid.
func TestMultiSigBenchMetricsFlag(t *testing.T) {
	lines, stderr := runHarness(t,
		"--signers", "4",
		"--iterations", "2",
		"--mode", "sign",
		"--seed", "42",
		"--metrics",
	)
	if len(lines) != 1 {
		t.Fatalf("expected 1 JSON line; got %d", len(lines))
	}
	_ = parseRecords(t, lines) // validates JSON
	if !strings.Contains(stderr, "METRICS") {
		t.Logf("note: no metrics line detected in stderr; content: %s", stderr)
	}
}

// TestMultiSigBenchPercentiles ensures percentile fields are present and ordered.
func TestMultiSigBenchPercentiles(t *testing.T) {
	lines, _ := runHarness(t,
		"--signers", "3",
		"--iterations", "7",
		"--mode", "both",
		"--seed", "77",
	)
	if len(lines) != 1 {
		t.Fatalf("expected 1 JSON line; got %d", len(lines))
	}
	recs := parseRecords(t, lines)
	r := recs[0]
	// Presence checks
	for _, k := range []string{"p50_sign_ns", "p95_sign_ns", "p99_sign_ns", "p50_verify_ns", "p95_verify_ns", "p99_verify_ns"} {
		if _, ok := r[k]; !ok {
			t.Fatalf("missing percentile field %s", k)
		}
	}
	p50s := r["p50_sign_ns"].(float64)
	p95s := r["p95_sign_ns"].(float64)
	p99s := r["p99_sign_ns"].(float64)
	if !(p50s <= p95s && p95s <= p99s) {
		t.Fatalf("sign percentile ordering invalid: p50=%f p95=%f p99=%f", p50s, p95s, p99s)
	}
	p50v := r["p50_verify_ns"].(float64)
	p95v := r["p95_verify_ns"].(float64)
	p99v := r["p99_verify_ns"].(float64)
	if !(p50v <= p95v && p95v <= p99v) {
		t.Fatalf("verify percentile ordering invalid: p50=%f p95=%f p99=%f", p50v, p95v, p99v)
	}
}
