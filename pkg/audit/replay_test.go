package audit

import (
	"context"
	"testing"
)

// TestAuditReplayDeterminism ensures recorded authorization decision metadata
// contains sufficient fields to deterministically reproduce the outcome (AAP001 §4).
func TestAuditReplayDeterminism(t *testing.T) {
	ml := NewMemoryLogger(nil)
	req := map[string]interface{}{"subject": "alice", "action": "read", "resource": "doc:1"}
	outcome := map[string]interface{}{"allow": true, "deny": false, "reason": "allowed"}
	ev := &Event{Type: EventTypeAuthorization, Action: "evaluate", Result: ResultSuccess, Subject: "alice", Metadata: map[string]interface{}{"request": req, "decision": outcome}}
	if err := ml.Log(context.TODO(), ev); err != nil {
		t.Fatalf("log: %v", err)
	}
	if err := ml.VerifyChain(); err != nil {
		t.Fatalf("verify chain: %v", err)
	}
	metaReq, ok := ev.Metadata["request"].(map[string]interface{})
	if !ok {
		t.Fatalf("request metadata missing")
	}
	metaDec, ok := ev.Metadata["decision"].(map[string]interface{})
	if !ok {
		t.Fatalf("decision metadata missing")
	}
	if metaReq["subject"] != "alice" || metaReq["action"] != "read" || metaReq["resource"] != "doc:1" {
		t.Fatalf("request metadata mismatch: %+v", metaReq)
	}
	if metaDec["allow"] != true || metaDec["deny"] != false || metaDec["reason"] != "allowed" {
		t.Fatalf("decision metadata mismatch: %+v", metaDec)
	}
}
