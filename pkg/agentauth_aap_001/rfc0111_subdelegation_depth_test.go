package agentauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/aap"
	"github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/authz"
	"github.com/mauriciomferz/AgentAuth/pkg/common"
)

// MockAllowAllAuthorizer implements authz.Authorizer and allows everything
type MockAllowAllAuthorizer struct{}

func (m *MockAllowAllAuthorizer) Authorize(ctx context.Context, request authz.Request) (authz.Decision, error) {
	return authz.Decision{Allow: true}, nil
}

func (m *MockAllowAllAuthorizer) GetPermissions(ctx context.Context, subject string) ([]authz.Permission, error) {
	// Return a wildcard permission to allow scope validation checks to pass
	return []authz.Permission{
		{Resource: "*", Actions: []string{"*"}, Granted: true},
	}, nil
}

func TestSubDelegation_MaxDepth_Enforcement(t *testing.T) {
	// Setup service with in-memory repo and mocks
	auditLogger := audit.NewMemoryLogger(common.NewSimpleLogger())
	authorizer := &MockAllowAllAuthorizer{}
	svc := NewService(auditLogger, authorizer)
	ctx := context.Background()

	// 1. Create Root PoA with max_delegation_depth = 2
	rootReq := DelegationRequest{
		Grantor: "alice",
		Grantee: "bob",
		Scope:   []string{"read:data"},
		Restrictions: map[string]string{
			"max_delegation_depth": "2",
		},
		Duration: 24 * time.Hour,
	}
	rootResp, err := svc.CreateDelegationCtx(ctx, rootReq)
	if err != nil {
		t.Fatalf("Failed to create root delegation: %v", err)
	}

	// 2. Create Level 1 Child (Depth 1) - Should succeed
	childReq := DelegationRequest{
		Grantor:     "bob",
		Grantee:     "carol",
		Scope:       []string{"read:data"},
		ParentPOAID: rootResp.POA.ID,
		Duration:    12 * time.Hour,
	}
	childResp, err := svc.CreateDelegationCtx(ctx, childReq)
	if err != nil {
		t.Fatalf("Failed to create child delegation (Level 1): %v", err)
	}
	if childResp.POA.Depth != 1 {
		t.Errorf("Expected child depth 1, got %d", childResp.POA.Depth)
	}

	// 3. Create Level 2 Grandchild (Depth 2) - Should succeed (limit is <= 2)
	grandChildReq := DelegationRequest{
		Grantor:     "carol",
		Grantee:     "dave",
		Scope:       []string{"read:data"},
		ParentPOAID: childResp.POA.ID,
		Duration:    6 * time.Hour,
	}
	grandChildResp, err := svc.CreateDelegationCtx(ctx, grandChildReq)
	if err != nil {
		t.Fatalf("Failed to create grandchild delegation (Level 2): %v", err)
	}
	if grandChildResp.POA.Depth != 2 {
		t.Errorf("Expected grandchild depth 2, got %d", grandChildResp.POA.Depth)
	}

	// 4. Create Level 3 GreatGrandchild (Depth 3) - Should fail (exceeds max 2)
	greatGrandChildReq := DelegationRequest{
		Grantor:     "dave",
		Grantee:     "eve",
		Scope:       []string{"read:data"},
		ParentPOAID: grandChildResp.POA.ID,
		Duration:    1 * time.Hour,
	}
	_, err = svc.CreateDelegationCtx(ctx, greatGrandChildReq)
	if err == nil {
		t.Error("Expected error for delegation depth 3 (max 2), got nil")
	} else if errRes, ok := err.(aap.RFCError); !ok || errRes.Code != aap.ErrInvalidRequest {
		t.Errorf("Expected ErrInvalidRequest, got %v", err)
	}
}

func TestSubDelegation_MaxDepth_Tightening(t *testing.T) {
	// Setup service with in-memory repo and mocks
	auditLogger := audit.NewMemoryLogger(common.NewSimpleLogger())
	authorizer := &MockAllowAllAuthorizer{}
	svc := NewService(auditLogger, authorizer)
	ctx := context.Background()

	// Root allows depth 5
	rootReq := DelegationRequest{
		Grantor: "root",
		Grantee: "admin",
		Scope:   []string{"admin"},
		Restrictions: map[string]string{
			"max_delegation_depth": "5",
		},
		Duration: 24 * time.Hour,
	}
	rootResp, _ := svc.CreateDelegationCtx(ctx, rootReq)

	// Child tries to relax limit to 10 - Should fail
	relaxReq := DelegationRequest{
		Grantor:     "admin",
		Grantee:     "user",
		Scope:       []string{"admin"},
		ParentPOAID: rootResp.POA.ID,
		Restrictions: map[string]string{
			"max_delegation_depth": "10",
		},
		Duration: 12 * time.Hour,
	}
	_, err := svc.CreateDelegationCtx(ctx, relaxReq)
	if err == nil {
		t.Error("Child allowed to relax max_delegation_depth from 5 to 10")
	}

	// Child tightens limit to 3 - Should succeed
	tightenReq := DelegationRequest{
		Grantor:     "admin",
		Grantee:     "user",
		Scope:       []string{"admin"},
		ParentPOAID: rootResp.POA.ID,
		Restrictions: map[string]string{
			"max_delegation_depth": "3",
		},
		Duration: 12 * time.Hour,
	}
	childResp, err := svc.CreateDelegationCtx(ctx, tightenReq)
	if err != nil {
		t.Errorf("Failed to tighten max_delegation_depth: %v", err)
	}

	// Grandchild creation logic check (simulated)
	// We verify that effective limit for grandchild respects tighter bound if we were to continue
	// Since CreateDelegationCtx reads from immediate parent, the grandchild will see '3' as the constraint.
	if val, ok := childResp.POA.Restrictions["max_delegation_depth"]; !ok || val != "3" {
		t.Errorf("Child restriction did not persist: %v", childResp.POA.Restrictions)
	}
}
