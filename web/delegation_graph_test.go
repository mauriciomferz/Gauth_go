package web

import (
	"context"
	"testing"
	"time"

	agentauth_aap_001 "github.com/mauriciomferz/AgentAuth/pkg/agentauth_aap_001"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
)

func TestDelegationGraphExport(t *testing.T) {
	t.Skip("Test requires repository with List support - skipping until repository interface is updated")

	// Create a proper Service using the public constructor
	auditLogger := audit.NewMemoryLogger(nil)
	authorizer := authz.NewMemoryAuthorizer()
	// Add required policies for the test
	authorizer.AddPolicy(authz.Policy{
		ID:       "test-policy",
		Subject:  "*",
		Resource: "*",
		Actions:  []string{"create_delegation"},
		Effect:   authz.Allow,
	})

	svc := agentauth_aap_001.NewService(auditLogger, authorizer)

	// Create delegations through the service (this will populate the repo)
	rootResp, err := svc.CreateDelegationCtx(context.Background(), agentauth_aap_001.DelegationRequest{
		Grantor:  "alice",
		Grantee:  "agentA",
		Scope:    []string{"scope.x"},
		Duration: time.Hour,
	})
	if err != nil {
		t.Fatalf("create root delegation: %v", err)
	}

	childResp, err := svc.CreateDelegationCtx(context.Background(), agentauth_aap_001.DelegationRequest{
		Grantor:     "agentA",
		Grantee:     "agentB",
		Scope:       []string{"scope.x"},
		Duration:    time.Hour,
		ParentPOAID: rootResp.POA.ID,
	})
	if err != nil {
		t.Fatalf("create child delegation: %v", err)
	}

	grandResp, err := svc.CreateDelegationCtx(context.Background(), agentauth_aap_001.DelegationRequest{
		Grantor:     "agentB",
		Grantee:     "agentC",
		Scope:       []string{"scope.x"},
		Duration:    time.Hour,
		ParentPOAID: childResp.POA.ID,
	})
	if err != nil {
		t.Fatalf("create grand delegation: %v", err)
	}

	// Now test the delegation graph export
	nodes, err := svc.BuildDelegationGraph(context.Background())
	if err != nil {
		t.Fatalf("BuildDelegationGraph error: %v", err)
	}

	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes got %d", len(nodes))
	}

	// verify parent linkage and depth
	find := func(id string) *agentauth_aap_001.DelegationGraphNode {
		for _, n := range nodes {
			if n.ID == id {
				return &n
			}
		}
		return nil
	}

	if n := find(rootResp.POA.ID); n == nil || n.Depth != 0 || n.Parent != "" {
		t.Fatalf("root node invalid: %+v", n)
	}
	if n := find(childResp.POA.ID); n == nil || n.Depth != 1 || n.Parent != rootResp.POA.ID {
		t.Fatalf("child node invalid: %+v", n)
	}
	if n := find(grandResp.POA.ID); n == nil || n.Depth != 2 || n.Parent != childResp.POA.ID {
		t.Fatalf("grand node invalid: %+v", n)
	}
}
