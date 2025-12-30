// Package agentauth - Policy Store Tests
package agentauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInMemoryPolicyStore_CRUD tests basic CRUD operations
func TestInMemoryPolicyStore_CRUD(t *testing.T) {
	store := NewInMemoryPolicyStore()
	ctx := context.Background()

	// Create a test policy
	policy := &AuthorizationPolicy{
		PolicyID:    "test-policy-001",
		PolicyType:  PolicyTypePoA,
		PolicyName:  "Test Policy",
		Description: "Test policy for CRUD operations",
		Status:      PolicyStatusDraft,
		ClientOwner: "client-001",
		PolicyRules: PolicyRules{
			AllowedActions: []string{"read", "write"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Test Create
	err := store.Create(ctx, policy)
	require.NoError(t, err, "Create should succeed")

	// Test duplicate Create
	err = store.Create(ctx, policy)
	assert.Error(t, err, "Creating duplicate policy should fail")

	// Test Get
	retrieved, err := store.Get(ctx, "test-policy-001")
	require.NoError(t, err, "Get should succeed")
	assert.Equal(t, policy.PolicyID, retrieved.PolicyID)
	assert.Equal(t, policy.PolicyName, retrieved.PolicyName)

	// Test Update
	retrieved.Description = "Updated description"
	err = store.Update(ctx, retrieved)
	require.NoError(t, err, "Update should succeed")

	updated, err := store.Get(ctx, "test-policy-001")
	require.NoError(t, err)
	assert.Equal(t, "Updated description", updated.Description)

	// Test Exists
	exists, err := store.Exists(ctx, "test-policy-001")
	require.NoError(t, err)
	assert.True(t, exists)

	// Test Delete
	err = store.Delete(ctx, "test-policy-001")
	require.NoError(t, err, "Delete should succeed")

	// Verify deletion
	_, err = store.Get(ctx, "test-policy-001")
	assert.Error(t, err, "Get after delete should fail")
}

// TestInMemoryPolicyStore_List tests listing with filtering
func TestInMemoryPolicyStore_List(t *testing.T) {
	store := NewInMemoryPolicyStore()
	ctx := context.Background()

	// Create multiple policies with different statuses
	policies := []*AuthorizationPolicy{
		{
			PolicyID:   "policy-001",
			PolicyType: PolicyTypePoA,
			PolicyName: "Active Policy 1",
			Status:     PolicyStatusActive,
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			PolicyID:   "policy-002",
			PolicyType: PolicyTypePoA,
			PolicyName: "Active Policy 2",
			Status:     PolicyStatusActive,
			PolicyRules: PolicyRules{
				AllowedActions: []string{"write"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			PolicyID:   "policy-003",
			PolicyType: PolicyTypePoA,
			PolicyName: "Draft Policy",
			Status:     PolicyStatusDraft,
			PolicyRules: PolicyRules{
				AllowedActions: []string{"delete"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, p := range policies {
		err := store.Create(ctx, p)
		require.NoError(t, err)
	}

	// Test listing all policies
	allPolicies, err := store.List(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, allPolicies, 3, "Should have 3 policies")

	// Test filtering by status
	activeStatus := PolicyStatusActive
	activePolicies, err := store.List(ctx, &activeStatus)
	require.NoError(t, err)
	assert.Len(t, activePolicies, 2, "Should have 2 active policies")

	draftStatus := PolicyStatusDraft
	draftPolicies, err := store.List(ctx, &draftStatus)
	require.NoError(t, err)
	assert.Len(t, draftPolicies, 1, "Should have 1 draft policy")
}

// TestInMemoryPolicyStore_Search tests search functionality
func TestInMemoryPolicyStore_Search(t *testing.T) {
	store := NewInMemoryPolicyStore()
	ctx := context.Background()

	// Create test policies
	policies := []*AuthorizationPolicy{
		{
			PolicyID:         "policy-001",
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Client A Policy",
			Status:           PolicyStatusActive,
			ClientOwner:      "client-a",
			OwnersAuthorizer: "authorizer-1",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			PolicyID:         "policy-002",
			PolicyType:       PolicyTypeAuthorizationChain,
			PolicyName:       "Client B Policy",
			Status:           PolicyStatusDraft,
			ClientOwner:      "client-b",
			OwnersAuthorizer: "authorizer-2",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"write"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, p := range policies {
		err := store.Create(ctx, p)
		require.NoError(t, err)
	}

	// Test search by client owner
	criteria := &PolicySearchCriteria{
		ClientOwner: "client-a",
	}
	results, err := store.Search(ctx, criteria)
	require.NoError(t, err)
	assert.Len(t, results, 1, "Should find 1 policy for client-a")
	assert.Equal(t, "policy-001", results[0].PolicyID)

	// Test search by policy type
	criteria = &PolicySearchCriteria{
		PolicyTypes: []PolicyType{PolicyTypeAuthorizationChain},
	}
	results, err = store.Search(ctx, criteria)
	require.NoError(t, err)
	assert.Len(t, results, 1, "Should find 1 authorization chain policy")

	// Test search with limit
	criteria = &PolicySearchCriteria{
		Limit: 1,
	}
	results, err = store.Search(ctx, criteria)
	require.NoError(t, err)
	assert.Len(t, results, 1, "Should respect limit")
}

// TestInMemoryPolicyStore_Count tests count functionality
func TestInMemoryPolicyStore_Count(t *testing.T) {
	store := NewInMemoryPolicyStore()
	ctx := context.Background()

	// Create multiple policies
	for i := 0; i < 5; i++ {
		status := PolicyStatusActive
		if i%2 == 0 {
			status = PolicyStatusDraft
		}

		policy := &AuthorizationPolicy{
			PolicyID:   string(rune('A' + i)),
			PolicyType: PolicyTypePoA,
			PolicyName: "Test Policy",
			Status:     status,
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}

		err := store.Create(ctx, policy)
		require.NoError(t, err)
	}

	// Test total count
	totalCount, err := store.Count(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, 5, totalCount, "Should have 5 total policies")

	// Test count by status
	activeStatus := PolicyStatusActive
	activeCount, err := store.Count(ctx, &activeStatus)
	require.NoError(t, err)
	assert.Equal(t, 2, activeCount, "Should have 2 active policies")

	draftStatus := PolicyStatusDraft
	draftCount, err := store.Count(ctx, &draftStatus)
	require.NoError(t, err)
	assert.Equal(t, 3, draftCount, "Should have 3 draft policies")
}

// TestInMemoryPolicyStore_IsolationCopy tests that store returns copies, not references
func TestInMemoryPolicyStore_IsolationCopy(t *testing.T) {
	store := NewInMemoryPolicyStore()
	ctx := context.Background()

	// Create a policy
	policy := &AuthorizationPolicy{
		PolicyID:    "test-isolation",
		PolicyType:  PolicyTypePoA,
		PolicyName:  "Original Name",
		Description: "Original Description",
		Status:      PolicyStatusDraft,
		PolicyRules: PolicyRules{
			AllowedActions: []string{"read"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err := store.Create(ctx, policy)
	require.NoError(t, err)

	// Get the policy
	retrieved, err := store.Get(ctx, "test-isolation")
	require.NoError(t, err)

	// Modify the retrieved policy (should not affect stored version)
	retrieved.PolicyName = "Modified Name"
	retrieved.Description = "Modified Description"

	// Get the policy again
	retrieved2, err := store.Get(ctx, "test-isolation")
	require.NoError(t, err)

	// Verify original values are preserved
	assert.Equal(t, "Original Name", retrieved2.PolicyName, "Stored policy should not be affected by external modifications")
	assert.Equal(t, "Original Description", retrieved2.Description, "Stored policy should not be affected by external modifications")
}

// TestPowerAdministrationPoint_WithPolicyStore tests PAP using the PolicyStore interface
func TestPowerAdministrationPoint_WithPolicyStore(t *testing.T) {
	// Test with in-memory store
	store := NewInMemoryPolicyStore()
	pap := NewPowerAdministrationPointWithStore("pap-001", "Test PAP", "Test with custom store", store)
	ctx := context.Background()

	// Create a policy via PAP
	request := &PolicyCreateRequest{
		PolicyType:       PolicyTypePoA,
		PolicyName:       "Test Policy",
		ClientOwner:      "client-001",
		OwnersAuthorizer: "authorizer-001",
		PolicyRules: PolicyRules{
			AllowedActions: []string{"read", "write"},
		},
	}

	created, err := pap.CreatePolicy(ctx, request)
	require.NoError(t, err)
	assert.NotEmpty(t, created.PolicyID)

	// Retrieve via PAP
	retrieved, err := pap.GetPolicy(ctx, created.PolicyID)
	require.NoError(t, err)
	assert.Equal(t, created.PolicyID, retrieved.PolicyID)
	assert.Equal(t, "Test Policy", retrieved.PolicyName)

	// List via PAP
	policies, err := pap.ListPolicies(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, policies, 1)

	// Delete via PAP
	err = pap.DeletePolicy(ctx, created.PolicyID)
	require.NoError(t, err)

	// Verify deletion
	_, err = pap.GetPolicy(ctx, created.PolicyID)
	assert.Error(t, err)
}
