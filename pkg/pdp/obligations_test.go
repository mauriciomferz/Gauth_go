package pdp

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	imetrics "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	iobligations "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/obligations"
)

// simpleFailExecutor wraps SimpleExecutor to force a known failure name for testing.
type simpleFailExecutor struct{ *iobligations.SimpleExecutor }

func TestObligationsExecutionSuccessAndAudit(t *testing.T) {
	m := imetrics.NewMemory()
	auditPath := filepath.Join(t.TempDir(), "test_obligations_audit_success.jsonl")
	exec := iobligations.NewSimpleExecutor(1, 2, nil) // no failures
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).WithMetrics(m).WithObligations(exec, auditPath)
	engine.AddPolicy(Policy{ID: "p1", Subjects: []string{"alice"}, Rules: []Rule{{ID: "r1", Actions: []string{"read"}, Resources: []string{"doc"}, Effect: outcomeAllow}}, Obligations: []Obligation{{ID: "log_access"}, {ID: "notify"}}})
	dec, err := engine.Evaluate(context.Background(), Request{Subject: "alice", Action: "read", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.Allow {
		t.Fatalf("expected allow decision")
	}
	if len(dec.Obligations) != 2 {
		t.Fatalf("expected 2 obligations, got %d", len(dec.Obligations))
	}
	// metrics: 2 obligations executed
	if m.ObligationsExecuted() != 2 {
		t.Fatalf("expected obligations executed = 2, got %d", m.ObligationsExecuted())
	}
	if m.ObligationsFailed() != 0 {
		t.Fatalf("expected obligations failed = 0, got %d", m.ObligationsFailed())
	}
	// audit file lines
	f, err := os.Open(auditPath)
	if err != nil {
		t.Fatalf("failed reading audit file: %v", err)
	}
	scanner := bufio.NewScanner(f)
	count := 0
	for scanner.Scan() {
		count++
		line := scanner.Text()
		if !strings.Contains(line, "duration_ms") {
			f.Close()
			t.Fatalf("audit line missing duration_ms: %s", line)
		}
		var rec struct {
			Duration float64 `json:"duration_ms"`
			Index    int     `json:"index"`
		}
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			f.Close()
			t.Fatalf("failed to unmarshal audit line: %v", err)
		}
		if rec.Duration < 0 {
			f.Close()
			t.Fatalf("negative duration_ms")
		}
	}
	f.Close()
	if count != 2 {
		t.Fatalf("expected 2 audit lines, got %d", count)
	}
}

func TestObligationsExecutionFailure(t *testing.T) {
	m := imetrics.NewMemory()
	auditPath := filepath.Join(t.TempDir(), "test_obligations_audit_fail.jsonl")
	// Force failure for obligation id "fail_me"
	exec := iobligations.NewSimpleExecutor(0, 0, []string{"fail_me"})
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).WithMetrics(m).WithObligations(exec, auditPath)
	engine.AddPolicy(Policy{ID: "p2", Subjects: []string{"bob"}, Rules: []Rule{{ID: "r2", Actions: []string{"write"}, Resources: []string{"doc"}, Effect: outcomeAllow}}, Obligations: []Obligation{{ID: "fail_me"}, {ID: "ok"}}})
	dec, err := engine.Evaluate(context.Background(), Request{Subject: "bob", Action: "write", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.Allow {
		t.Fatalf("expected allow decision")
	}
	if m.ObligationsExecuted() != 1 {
		t.Fatalf("expected executed=1 (only ok), got %d", m.ObligationsExecuted())
	}
	if m.ObligationsFailed() != 1 {
		t.Fatalf("expected failed=1 (fail_me), got %d", m.ObligationsFailed())
	}
	f, err := os.Open(auditPath)
	if err != nil {
		t.Fatalf("failed reading audit file: %v", err)
	}
	scanner := bufio.NewScanner(f)
	count := 0
	failureSeen := false
	for scanner.Scan() {
		count++
		line := scanner.Text()
		if !strings.Contains(line, "duration_ms") {
			f.Close()
			t.Fatalf("missing duration_ms: %s", line)
		}
		if strings.Contains(line, "fail_me") && strings.Contains(line, "\"success\":false") {
			failureSeen = true
		}
	}
	f.Close()
	if count != 2 {
		t.Fatalf("expected 2 audit lines, got %d", count)
	}
	if !failureSeen {
		t.Fatalf("expected failure audit entry for fail_me")
	}
}

func TestMandatoryFailureDeniesWhenConfigured(t *testing.T) {
	m := imetrics.NewMemory()
	auditPath := filepath.Join(t.TempDir(), "test_obligations_audit_mandatory_deny.jsonl")
	// fail mandatory obligation
	exec := iobligations.NewSimpleExecutor(0, 0, []string{"critical_log"})
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).WithMetrics(m).WithObligations(exec, auditPath).WithObligationFailureDenies(true)
	engine.AddPolicy(Policy{ID: "p_mand", Subjects: []string{"dave"}, Rules: []Rule{{ID: "r_mand", Actions: []string{"read"}, Resources: []string{"doc"}, Effect: outcomeAllow}}, Obligations: []Obligation{{ID: "critical_log", Mandatory: true}}})
	dec, err := engine.Evaluate(context.Background(), Request{Subject: "dave", Action: "read", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dec.Allow {
		t.Fatalf("expected deny due to mandatory failure; got allow with reason %s", dec.Reason)
	}
	if !strings.Contains(dec.Reason, "mandatory obligation") {
		t.Fatalf("reason should mention mandatory failure: %s", dec.Reason)
	}
	if dec.Metadata["mandatory_obligation_failures"] != "critical_log" {
		t.Fatalf("metadata missing failures list: %+v", dec.Metadata)
	}
}

func TestMandatoryFailureDoesNotDenyWhenDisabled(t *testing.T) {
	m := imetrics.NewMemory()
	auditPath := filepath.Join(t.TempDir(), "test_obligations_audit_mandatory_no_deny.jsonl")
	exec := iobligations.NewSimpleExecutor(0, 0, []string{"critical_log"})
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).WithMetrics(m).WithObligations(exec, auditPath).WithObligationFailureDenies(false)
	engine.AddPolicy(Policy{ID: "p_nomand", Subjects: []string{"erin"}, Rules: []Rule{{ID: "r_nomand", Actions: []string{"read"}, Resources: []string{"doc"}, Effect: outcomeAllow}}, Obligations: []Obligation{{ID: "critical_log", Mandatory: true}}})
	dec, err := engine.Evaluate(context.Background(), Request{Subject: "erin", Action: "read", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.Allow {
		t.Fatalf("expected allow; engine not configured to deny on mandatory failure")
	}
	if _, ok := dec.Metadata["mandatory_obligation_failures"]; ok {
		t.Fatalf("metadata should not include mandatory failures when outcome not flipped")
	}
}

func TestObligationLatencyAndMandatoryMetrics(t *testing.T) {
	m := imetrics.NewMemory()
	auditPath := filepath.Join(t.TempDir(), "test_obligations_latency_metrics.jsonl")
	// One mandatory failing, one succeeding to exercise both latency and mandatory failure counter
	exec := iobligations.NewSimpleExecutor(1, 3, []string{"must_fail"})
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).WithMetrics(m).WithObligations(exec, auditPath).WithObligationFailureDenies(true)
	engine.AddPolicy(Policy{ID: "p_lat", Subjects: []string{"frank"}, Rules: []Rule{{ID: "r_lat", Actions: []string{"read"}, Resources: []string{"doc"}, Effect: outcomeAllow}}, Obligations: []Obligation{{ID: "must_fail", Mandatory: true}, {ID: "ok", Mandatory: false}}})
	dec, err := engine.Evaluate(context.Background(), Request{Subject: "frank", Action: "read", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should be denied due to mandatory failure
	if dec.Allow {
		t.Fatalf("expected deny due to mandatory failure")
	}
	// Metrics: mandatory failure counter should be >=1
	if m.MandatoryObligationFailures() < 1 {
		t.Fatalf("expected mandatory obligation failures >=1, got %d", m.MandatoryObligationFailures())
	}
	// Latency metrics: at least 2 observations
	snap := m.SnapshotEx()
	if snap.ObligationLatencyCount < 2 {
		t.Fatalf("expected at least 2 obligation latency observations, got %d", snap.ObligationLatencyCount)
	}
	if snap.ObligationLatencyTotalNS == 0 {
		t.Fatalf("expected non-zero total latency")
	}
}

func TestObligationsContextCancellation(t *testing.T) {
	m := imetrics.NewMemory()
	auditPath := filepath.Join(t.TempDir(), "test_obligations_audit_cancel.jsonl")
	// executor with artificial small latency to allow cancellation mid-way
	exec := iobligations.NewSimpleExecutor(5, 5, nil)
	engine := NewInMemoryEngine(DenyOverridesStrategy{}).WithMetrics(m).WithObligations(exec, auditPath)
	engine.AddPolicy(Policy{ID: "p3", Subjects: []string{"carol"}, Rules: []Rule{{ID: "r3", Actions: []string{"read"}, Resources: []string{"doc"}, Effect: outcomeAllow}}, Obligations: []Obligation{{ID: "one"}, {ID: "two"}, {ID: "three"}}})
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel quickly
	go func() { time.Sleep(2 * time.Millisecond); cancel() }()
	dec, err := engine.Evaluate(ctx, Request{Subject: "carol", Action: "read", Resource: "doc", Time: time.Now()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dec.Allow {
		t.Fatalf("expected allow decision")
	}
	// We can't guarantee exact split due to timing, but at least one should fail
	if m.ObligationsFailed() == 0 {
		t.Fatalf("expected at least one failed due to cancellation")
	}
	if m.ObligationsExecuted()+m.ObligationsFailed() < 1 {
		t.Fatalf("expected some obligations processed")
	}
	f, err := os.Open(auditPath)
	if err != nil {
		t.Fatalf("failed reading audit file: %v", err)
	}
	scanner := bufio.NewScanner(f)
	found := false
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "duration_ms") {
			found = true
			break
		}
	}
	f.Close()
	if !found {
		t.Fatalf("expected at least one line with duration_ms")
	}
}
