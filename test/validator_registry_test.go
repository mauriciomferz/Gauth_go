package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
)

// TestValidatorRegistrySuccess ensures validator passes and metrics increment.
func TestValidatorRegistrySuccess(t *testing.T) {
	vr := authz.NewValidatorRegistry()
	err := vr.Register("allow_all", func(req authz.Request, policy authz.Policy) error { return nil }, authz.WithDescription("always passes"))
	if err != nil {
		t.Fatalf("register error: %v", err)
	}
	ma := authz.NewMemoryAuthorizer()
	ma.SetValidatorRegistry(vr)
	p := authz.Policy{ID: "v1", Subject: "bob", Resource: "res:1", Actions: []string{"read"}, Effect: authz.Allow, Validators: []string{"allow_all"}}
	ma.AddPolicy(p)
	ma.Snapshot()
	dec, _ := ma.Authorize(context.Background(), authz.Request{Subject: "bob", Resource: "res:1", Action: "read"})
	if !dec.Allow {
		t.Fatalf("expected allow decision")
	}
	metrics := vr.Snapshot()
	if len(metrics) != 1 || metrics[0].Invocations == 0 {
		t.Fatalf("expected invocation metrics, got %#v", metrics)
	}
	if metrics[0].Failures != 0 {
		t.Fatalf("expected zero failures")
	}
}

// TestValidatorRegistryFailure ensures failing validator blocks policy match.
func TestValidatorRegistryFailure(t *testing.T) {
	vr := authz.NewValidatorRegistry()
	_ = vr.Register("deny_region", func(req authz.Request, policy authz.Policy) error {
		if req.Context["region"] == "blocked" {
			return errors.New("region blocked")
		}
		return nil
	}, authz.WithTimeout(100*time.Millisecond))
	ma := authz.NewMemoryAuthorizer()
	ma.SetValidatorRegistry(vr)
	p := authz.Policy{ID: "v2", Subject: "alice", Resource: "obj:*", Actions: []string{"write"}, Effect: authz.Allow, Validators: []string{"deny_region"}}
	ma.AddPolicy(p)
	ma.Snapshot()
	dec1, _ := ma.Authorize(context.Background(), authz.Request{Subject: "alice", Resource: "obj:9", Action: "write", Context: map[string]string{"region": "eu"}})
	if !dec1.Allow {
		t.Fatalf("expected allow when region not blocked")
	}
	dec2, _ := ma.Authorize(context.Background(), authz.Request{Subject: "alice", Resource: "obj:9", Action: "write", Context: map[string]string{"region": "blocked"}})
	if dec2.Allow {
		t.Fatalf("expected deny when validator fails")
	}
	m := vr.Snapshot()[0]
	if m.Failures == 0 {
		t.Fatalf("expected failure count >0")
	}
}

// TestValidatorRegistryTimeout ensures timeout fails closed.
func TestValidatorRegistryTimeout(t *testing.T) {
	vr := authz.NewValidatorRegistry()
	_ = vr.Register("slow", func(req authz.Request, policy authz.Policy) error { time.Sleep(50 * time.Millisecond); return nil }, authz.WithTimeout(1*time.Millisecond))
	ma := authz.NewMemoryAuthorizer()
	ma.SetValidatorRegistry(vr)
	p := authz.Policy{ID: "v3", Subject: "jim", Resource: "acc:1", Actions: []string{"read"}, Effect: authz.Allow, Validators: []string{"slow"}}
	ma.AddPolicy(p)
	ma.Snapshot()
	dec, _ := ma.Authorize(context.Background(), authz.Request{Subject: "jim", Resource: "acc:1", Action: "read"})
	if dec.Allow {
		t.Fatalf("expected deny due to timeout fail-closed")
	}
	m := vr.Snapshot()[0]
	if m.Failures == 0 {
		t.Fatalf("expected failure metric for timeout")
	}
}

// helper context (could be extended later)
// removed testCtx helper (unused)
