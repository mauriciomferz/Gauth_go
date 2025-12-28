package authz

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectCycles(t *testing.T) {
	tests := []struct {
		name      string
		graph     map[string][]string
		wantError bool
	}{
		{
			name: "No Cycle",
			graph: map[string][]string{
				"A": {"B"},
				"B": {"C"},
				"C": {},
			},
			wantError: false,
		},
		{
			name: "Cycle A-B-A",
			graph: map[string][]string{
				"A": {"B"},
				"B": {"A"},
			},
			wantError: true,
		},
		{
			name: "Cycle A-B-C-A",
			graph: map[string][]string{
				"A": {"B"},
				"B": {"C"},
				"C": {"A"},
			},
			wantError: true,
		},
		{
			name: "Disconnected Cycle",
			graph: map[string][]string{
				"A": {"B"},
				"C": {"D"},
				"D": {"C"},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DetectCycles(tt.graph)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRoleInheritance(t *testing.T) {
	// Setup Authorizer
	az := NewMemoryAuthorizer()

	// Policy requires "editor" role
	az.AddPolicy(Policy{
		ID:       "p1",
		Effect:   Allow,
		Actions:  []string{"edit"},
		Resource: "doc",
		Roles:    []string{"editor"},
	})

	// Hierarchy: admin -> editor (admin includes editor permissions)
	hierarchy := map[string][]string{
		"admin": {"editor"},
	}
	require.NoError(t, az.SetRoleHierarchy(hierarchy))

	ctx := context.Background()

	// Test 1: User with "editor" role should pass
	req1 := Request{
		Subject:  "u1",
		Action:   "edit",
		Resource: "doc",
		Context:  map[string]string{"roles": "editor"},
	}
	dec1, err := az.Authorize(ctx, req1)
	require.NoError(t, err)
	assert.True(t, dec1.Allow, "Direct role assignment failed")

	// Test 2: User with "admin" role should pass (inherited)
	req2 := Request{
		Subject:  "u2",
		Action:   "edit",
		Resource: "doc",
		Context:  map[string]string{"roles": "admin"},
	}
	dec2, err := az.Authorize(ctx, req2)
	require.NoError(t, err)
	assert.True(t, dec2.Allow, "Inherited role assignment failed")

	// Test 3: User with "viewer" role should fail
	req3 := Request{
		Subject:  "u3",
		Action:   "edit",
		Resource: "doc",
		Context:  map[string]string{"roles": "viewer"},
	}
	dec3, err := az.Authorize(ctx, req3)
	require.NoError(t, err)
	assert.False(t, dec3.Allow, "Unrelated role should fail")
}

func TestSetRoleHierarchy_Cycle(t *testing.T) {
	az := NewMemoryAuthorizer()
	hierarchy := map[string][]string{
		"A": {"B"},
		"B": {"A"},
	}
	err := az.SetRoleHierarchy(hierarchy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cycle detected")
}
