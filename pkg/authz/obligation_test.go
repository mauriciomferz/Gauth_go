package authz

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/common"
)

func TestAuditObligationExecutor(t *testing.T) {
	ml := audit.NewMemoryLogger(common.NewSimpleLogger())
	t.Cleanup(func() { _ = ml.Close() })

	exec := &AuditObligationExecutor{Audit: ml}

	t.Run("MandatorySuccess", func(t *testing.T) {
		ob := Obligation{ID: "ob1", Type: "log", Mandatory: true}
		err := exec.Execute(ob, map[string]interface{}{"request_subject": "user1"})
		if err != nil {
			t.Errorf("expected success, got %v", err)
		}

		time.Sleep(100 * time.Millisecond) // Wait for async audit log
		// Since we can't easily sync, we just query and check
		// or use a small delay if needed.
		// Query is consistent if processEvents is fast.
		events, _ := ml.Query(context.Background(), &audit.Filter{EventTypes: []audit.EventType{audit.EventTypeObligation}})
		if len(events) == 0 {
			t.Errorf("expected audit event, got none")
		}
	})

	t.Run("MandatoryFailure", func(t *testing.T) {
		ob := Obligation{ID: "ob2", Type: "notify", Mandatory: true, Params: map[string]string{"simulate_failure": "true"}}
		err := exec.Execute(ob, map[string]interface{}{"request_subject": "user2"})
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		time.Sleep(100 * time.Millisecond) // Wait for async audit log

		events, _ := ml.Query(context.Background(), &audit.Filter{Subject: "user2"})
		if len(events) == 0 {
			t.Errorf("expected audit event for user2, got none")
		}
		if events[0].Result != "failure" {
			t.Errorf("expected failure result, got %s", events[0].Result)
		}
		if events[0].Severity != "warning" {
			t.Errorf("expected severity warning for mandatory failure, got %s", events[0].Severity)
		}
	})
}

func TestEnforcerWithObligations(t *testing.T) {
	ml := audit.NewMemoryLogger(common.NewSimpleLogger())
	t.Cleanup(func() { _ = ml.Close() })
	exec := &AuditObligationExecutor{Audit: ml}

	ma := NewMemoryAuthorizer()
	ma.SetObligationExecutor(exec)

	p := Policy{
		ID:       "p1",
		Subject:  "user1",
		Resource: "res1",
		Actions:  []string{"read"},
		Effect:   Allow,
		Obligations: []Obligation{
			{ID: "ob_mand", Type: "verify", Mandatory: true, Params: map[string]string{"simulate_failure": "true"}},
		},
	}
	ma.AddPolicy(p)

	t.Run("MandatoryFailureFlipsDecision", func(t *testing.T) {
		dec, err := ma.Authorize(context.Background(), Request{Subject: "user1", Resource: "res1", Action: "read"})
		if err != nil {
			t.Fatalf("auth failed: %v", err)
		}
		if dec.Allow {
			t.Errorf("expected decision to be flipped to Deny due to mandatory obligation failure")
		}
		if dec.Metadata["obligation_failure"] != "ob_mand" {
			t.Errorf("expected metadata obligation_failure=ob_mand, got %v", dec.Metadata["obligation_failure"])
		}
	})

	t.Run("AdviceFailureDoesNotFlipDecision", func(t *testing.T) {
		p2 := Policy{
			ID:       "p2",
			Subject:  "user2",
			Resource: "res2",
			Actions:  []string{"read"},
			Effect:   Allow,
			Advice: []Advice{
				{ID: "adv1", Type: "notify", Params: map[string]string{"simulate_failure": "true"}},
			},
		}
		ma.AddPolicy(p2)

		dec, err := ma.Authorize(context.Background(), Request{Subject: "user2", Resource: "res2", Action: "read"})
		if err != nil {
			t.Fatalf("auth failed: %v", err)
		}
		if !dec.Allow {
			t.Errorf("expected decision to remain Allow for non-mandatory advice failure")
		}
		if dec.Metadata["advice_failure"] != "adv1" {
			t.Errorf("expected metadata advice_failure=adv1, got %v", dec.Metadata["advice_failure"])
		}
	})
}
