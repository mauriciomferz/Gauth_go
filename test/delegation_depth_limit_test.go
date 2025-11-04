package test

import (
	"os"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/metrics"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/delegation"
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/rfc"
)

func TestDelegationDepthLimitExceeded(t *testing.T) {
	os.Setenv("GAUTH_MAX_DELEGATION_DEPTH", "2")
	defer os.Unsetenv("GAUTH_MAX_DELEGATION_DEPTH")
	chain := delegation.NewChain()
	mem := metrics.NewMemory()

	// Append first two delegations (depth 1 and 2) should succeed
	for i := 1; i <= 2; i++ {
		_, err := chain.AppendWithMetrics(delegation.Delegation{ID: string(rune('A' + i)), Subject: "s", Delegate: "d", Scope: map[string]string{"res": "x"}, ExpiresAt: time.Now().Add(time.Hour)}, mem)
		if err != nil {
			t.Fatalf("unexpected error appending depth %d: %v", i, err)
		}
	}
	// Third append should fail
	_, err := chain.AppendWithMetrics(delegation.Delegation{ID: "C", Subject: "s", Delegate: "d", Scope: map[string]string{"res": "x"}, ExpiresAt: time.Now().Add(time.Hour)}, mem)
	if err == nil {
		t.Fatalf("expected depth exceeded error")
	}
	rfcErr, ok := err.(rfc.RFCError)
	if !ok {
		t.Fatalf("expected RFCError got %T", err)
	}
	if rfcErr.Code != rfc.ErrDelegationDepthExceeded {
		t.Fatalf("expected code %s got %s", rfc.ErrDelegationDepthExceeded, rfcErr.Code)
	}
	if mem.SnapshotEx().DelegationStatusTransitions != 0 { /* unrelated metric unaffected */
	}
}

func TestDelegationDepthLimitDisabled(t *testing.T) {
	os.Unsetenv("GAUTH_MAX_DELEGATION_DEPTH")
	chain := delegation.NewChain()
	for i := 1; i <= 5; i++ {
		_, err := chain.Append(delegation.Delegation{ID: time.Now().Format("150405") + string(rune('A'+i)), Subject: "s", Delegate: "d", Scope: map[string]string{"res": "x"}, ExpiresAt: time.Now().Add(time.Hour)})
		if err != nil {
			t.Fatalf("unexpected error with unlimited depth: %v", err)
		}
	}
}
