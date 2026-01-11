package test

import (
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

func TestDelegationChainIntegrity(t *testing.T) {
	chain := delegation.NewChain()
	// Append two valid delegations
	d1, err := chain.Append(delegation.Delegation{
		ID: "d1", Subject: "alice", Delegate: "bob",
		Scope: map[string]string{"resource": "doc"}, ExpiresAt: time.Now().UTC().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("append d1: %v", err)
	}
	if d1.Hash == "" {
		t.Fatalf("expected hash for d1")
	}
	if _, err := chain.Append(delegation.Delegation{
		ID: "d2", Subject: "alice", Delegate: "carol",
		Scope: map[string]string{"resource": "doc"}, ExpiresAt: time.Now().UTC().Add(2 * time.Hour),
	}); err != nil {
		t.Fatalf("append d2: %v", err)
	}
	if err := chain.VerifyChain(); err != nil {
		t.Fatalf("verify: %v", err)
	}

	// Tamper prev hash of second
	head := chain.Head()
	const tamperedHash = "deadbeef"
	head.PrevHash = tamperedHash
	if err := chain.VerifyChain(); err == nil {
		t.Fatalf("expected verification failure after tamper")
	}
}

func TestDelegationScopeNarrowing(t *testing.T) {
	parent := delegation.Delegation{
		ID: "parent", Subject: "alice", Delegate: "bob",
		Scope:     map[string]string{"resource": "doc", "action": "read"},
		ExpiresAt: time.Now().UTC().Add(time.Hour),
	}
	childValid := delegation.Delegation{
		ID: "child", Subject: "bob", Delegate: "carol",
		Scope:     map[string]string{"resource": "doc", "action": "read"},
		ExpiresAt: time.Now().UTC().Add(time.Minute * 30),
	}
	if err := delegation.ValidateScopeNarrowing(parent, childValid); err != nil {
		t.Fatalf("expected valid narrowing: %v", err)
	}
	childWiden := delegation.Delegation{
		ID: "child2", Subject: "bob", Delegate: "carol",
		Scope:     map[string]string{"resource": "doc", "action": "write"},
		ExpiresAt: time.Now().UTC().Add(time.Minute * 30),
	}
	if err := delegation.ValidateScopeNarrowing(parent, childWiden); err == nil {
		t.Fatalf("expected widening error")
	}
}

func TestDelegationExpiryVerification(t *testing.T) {
	chain := delegation.NewChain()
	// Expired delegation should be rejected on append
	_, err := chain.Append(delegation.Delegation{
		ID: "expired", Subject: "alice", Delegate: "bob",
		Scope: map[string]string{"resource": "doc"}, ExpiresAt: time.Now().UTC().Add(-time.Minute),
	})
	if err == nil {
		t.Fatalf("expected error appending expired delegation")
	}

	// Append valid then artificially expire it and expect verification failure
	d1, err := chain.Append(delegation.Delegation{
		ID: "valid", Subject: "alice", Delegate: "bob",
		Scope: map[string]string{"resource": "doc"}, ExpiresAt: time.Now().UTC().Add(time.Second * 1),
	})
	if err != nil {
		t.Fatalf("append valid: %v", err)
	}
	if err := chain.VerifyChain(); err != nil {
		t.Fatalf("verify should pass initially: %v", err)
	}
	// Wait for expiry
	time.Sleep(time.Second * 2)
	if err := chain.VerifyChain(); err == nil {
		t.Fatalf("expected verification failure due to expiry")
	}
	_ = d1
}
