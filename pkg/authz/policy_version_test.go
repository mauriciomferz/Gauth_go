package authz

import "testing"

func TestPolicyVersioningAndRollback(t *testing.T) {
	ma := NewMemoryAuthorizer()
	ma.AddPolicy(Policy{ID: "p1", Subject: "alice", Resource: "doc", Actions: []string{"read"}, Effect: Allow})
	v1 := ma.Snapshot()
	if v1 != 1 {
		t.Fatalf("expected first snapshot version=1 got %d", v1)
	}
	ma.AddPolicy(Policy{ID: "p2", Subject: "alice", Resource: "doc", Actions: []string{"write"}, Effect: Allow})
	v2 := ma.Snapshot()
	if v2 != 2 {
		t.Fatalf("expected second snapshot version=2 got %d", v2)
	}
	if len(ma.policies) != 2 {
		t.Fatalf("expected 2 policies before rollback")
	}
	if err := ma.Rollback(v1); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if len(ma.policies) != 1 || ma.policies[0].ID != "p1" {
		t.Fatalf("rollback did not restore snapshot p1 only: %+v", ma.policies)
	}
	// Snapshot after rollback should produce version 2 again
	// (monotonic increment after setting ma.version=v1+1 then snapshot increments to 2)
	_ = ma.Snapshot()
	versions := ma.ListVersions()
	if len(versions) < 2 {
		t.Fatalf("expected >=2 versions got %v", versions)
	}
}
