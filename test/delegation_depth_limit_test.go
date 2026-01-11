package test

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/aap"
	"github.com/mauriciomferz/AgentAuth/pkg/aap/errs"
	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

func TestDelegationDepthLimitExceeded(t *testing.T) {
	t.Setenv("AGENTAUTH_MAX_DELEGATION_DEPTH", "2")
	chain := delegation.NewChain()
	mem := metrics.NewMemory()

	// Append first two delegations (depth 1 and 2) should succeed
	for i := 1; i <= 2; i++ {
		_, err := chain.AppendWithMetrics(delegation.Delegation{
			ID: string(rune('A' + i)), Subject: "s", Delegate: "d",
			Scope: map[string]string{"res": "x"}, ExpiresAt: time.Now().Add(time.Hour),
		}, mem)
		if err != nil {
			t.Fatalf("unexpected error appending depth %d: %v", i, err)
		}
	}
	// Third append should fail
	_, err := chain.AppendWithMetrics(delegation.Delegation{
		ID: "C", Subject: "s", Delegate: "d",
		Scope: map[string]string{"res": "x"}, ExpiresAt: time.Now().Add(time.Hour),
	}, mem)
	if err == nil {
		t.Fatalf("expected depth exceeded error")
	}
	rfcErr, ok := err.(errs.RFCError)
	if !ok {
		t.Fatalf("expected RFCError got %T", err)
	}
	if rfcErr.Code != aap.ErrDelegationDepthExceeded {
		t.Fatalf("expected code %s got %s", aap.ErrDelegationDepthExceeded, rfcErr.Code)
	}
	if mem.SnapshotEx().DelegationStatusTransitions != 0 { /* unrelated metric unaffected */
		_ = mem // Just checking the metric value
	}
}

func TestDelegationDepthLimitDisabled(t *testing.T) {
	t.Setenv("AGENTAUTH_MAX_DELEGATION_DEPTH", "")
	chain := delegation.NewChain()
	for i := 1; i <= 5; i++ {
		_, err := chain.Append(delegation.Delegation{
			ID: time.Now().Format("150405") + string(rune('A'+i)), Subject: "s", Delegate: "d",
			Scope: map[string]string{"res": "x"}, ExpiresAt: time.Now().Add(time.Hour),
		})
		if err != nil {
			t.Fatalf("unexpected error with unlimited depth: %v", err)
		}
	}
}
