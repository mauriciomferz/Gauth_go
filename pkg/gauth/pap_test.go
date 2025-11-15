// Package gauth - Power Administration Point (PAP) Tests
package gauth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewPowerAdministrationPoint tests PAP creation
func TestNewPowerAdministrationPoint(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test power administration point")

	assert.NotNil(t, pap)
	assert.Equal(t, "pap-001", pap.ID)
	assert.Equal(t, "Test PAP", pap.Name)
	assert.Equal(t, "Test power administration point", pap.Description)
	assert.NotZero(t, pap.CreatedAt)
	assert.NotNil(t, pap.policyStore)
	assert.Empty(t, pap.policyStore)
}

// TestPAP_CreatePolicy tests policy creation
func TestPAP_CreatePolicy(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	t.Run("ValidPolicyCreation", func(t *testing.T) {
		expiresAt := time.Now().Add(365 * 24 * time.Hour)
		request := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Test Policy",
			Description:      "Test policy description",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions:   []string{"read", "write"},
				ResourcePatterns: []string{"/api/*"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US", "CA"},
				Sectors:   []string{"healthcare"},
			},
			ExpiresAt: &expiresAt,
			Tags:      []string{"test", "healthcare"},
			Metadata: map[string]interface{}{
				"department": "IT",
			},
		}

		policy, err := pap.CreatePolicy(ctx, request)

		require.NoError(t, err)
		assert.NotNil(t, policy)
		assert.NotEmpty(t, policy.PolicyID)
		assert.Equal(t, PolicyTypePoA, policy.PolicyType)
		assert.Equal(t, "Test Policy", policy.PolicyName)
		assert.Equal(t, "Test policy description", policy.Description)
		assert.Equal(t, PolicyStatusDraft, policy.Status)
		assert.Equal(t, "client-001", policy.ClientOwner)
		assert.Equal(t, "authorizer-001", policy.OwnersAuthorizer)
		assert.Equal(t, 1, policy.PolicyVersion)
		assert.NotZero(t, policy.CreatedAt)
		assert.NotZero(t, policy.UpdatedAt)
		assert.Equal(t, []string{"read", "write"}, policy.PolicyRules.AllowedActions)
		assert.Equal(t, []string{"US", "CA"}, policy.Scope.Countries)
		assert.Equal(t, []string{"test", "healthcare"}, policy.Tags)
		assert.Equal(t, "IT", policy.Metadata["department"])
	})

	t.Run("NilRequest", func(t *testing.T) {
		policy, err := pap.CreatePolicy(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, policy)
		assert.Contains(t, err.Error(), "cannot be nil")
	})

	t.Run("MissingPolicyName", func(t *testing.T) {
		request := &PolicyCreateRequest{
			PolicyType:  PolicyTypePoA,
			PolicyName:  "",
			ClientOwner: "client-001",
		}

		policy, err := pap.CreatePolicy(ctx, request)

		assert.Error(t, err)
		assert.Nil(t, policy)
		assert.Contains(t, err.Error(), "policy name is required")
	})

	t.Run("MissingPolicyType", func(t *testing.T) {
		request := &PolicyCreateRequest{
			PolicyName:  "Test Policy",
			ClientOwner: "client-001",
		}

		policy, err := pap.CreatePolicy(ctx, request)

		assert.Error(t, err)
		assert.Nil(t, policy)
		assert.Contains(t, err.Error(), "policy type is required")
	})
}

// TestPAP_GetPolicy tests policy retrieval
func TestPAP_GetPolicy(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	// Create a policy first
	request := &PolicyCreateRequest{
		PolicyType:       PolicyTypePoA,
		PolicyName:       "Test Policy",
		ClientOwner:      "client-001",
		OwnersAuthorizer: "authorizer-001",
		PolicyRules: PolicyRules{
			AllowedActions: []string{"read"},
		},
	}

	created, err := pap.CreatePolicy(ctx, request)
	require.NoError(t, err)

	t.Run("PolicyExists", func(t *testing.T) {
		policy, err := pap.GetPolicy(ctx, created.PolicyID)

		require.NoError(t, err)
		assert.NotNil(t, policy)
		assert.Equal(t, created.PolicyID, policy.PolicyID)
		assert.Equal(t, "Test Policy", policy.PolicyName)
	})

	t.Run("PolicyNotFound", func(t *testing.T) {
		policy, err := pap.GetPolicy(ctx, "non-existent-policy")

		assert.Error(t, err)
		assert.Nil(t, policy)
		assert.Contains(t, err.Error(), "policy not found")
	})
}

// TestPAP_UpdatePolicy tests policy updates
func TestPAP_UpdatePolicy(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	// Create a policy first
	createRequest := &PolicyCreateRequest{
		PolicyType:       PolicyTypeScope,
		PolicyName:       "Original Name",
		ClientOwner:      "client-001",
		OwnersAuthorizer: "authorizer-001",
		PolicyRules: PolicyRules{
			AllowedActions: []string{"read"},
		},
	}

	created, err := pap.CreatePolicy(ctx, createRequest)
	require.NoError(t, err)

	t.Run("UpdateDraftPolicy", func(t *testing.T) {
		newName := "Updated Name"
		newDesc := "Updated description"
		newRules := PolicyRules{
			AllowedActions: []string{"read", "write", "delete"},
		}
		newTags := []string{"updated", "test"}

		updateRequest := &PolicyUpdateRequest{
			PolicyID:    created.PolicyID,
			PolicyName:  &newName,
			Description: &newDesc,
			PolicyRules: &newRules,
			Tags:        &newTags,
			ChangeLog:   "Updated policy configuration",
		}

		updated, err := pap.UpdatePolicy(ctx, updateRequest)

		require.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, "Updated Name", updated.PolicyName)
		assert.Equal(t, "Updated description", updated.Description)
		assert.Equal(t, []string{"read", "write", "delete"}, updated.PolicyRules.AllowedActions)
		assert.Equal(t, []string{"updated", "test"}, updated.Tags)
		assert.Equal(t, 2, updated.PolicyVersion) // Version incremented
		assert.Equal(t, "Updated policy configuration", updated.ChangeLog)
	})

	t.Run("UpdateNonExistentPolicy", func(t *testing.T) {
		newName := "Updated Name"
		updateRequest := &PolicyUpdateRequest{
			PolicyID:   "non-existent",
			PolicyName: &newName,
		}

		updated, err := pap.UpdatePolicy(ctx, updateRequest)

		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "policy not found")
	})

	t.Run("UpdateActivePolicy", func(t *testing.T) {
		// Create and activate a policy
		activeRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Active Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
		}

		activePolicy, err := pap.CreatePolicy(ctx, activeRequest)
		require.NoError(t, err)

		err = pap.ActivatePolicy(ctx, activePolicy.PolicyID, "approver-001")
		require.NoError(t, err)

		// Try to update active policy
		newName := "Updated Name"
		updateRequest := &PolicyUpdateRequest{
			PolicyID:   activePolicy.PolicyID,
			PolicyName: &newName,
		}

		updated, err := pap.UpdatePolicy(ctx, updateRequest)

		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "cannot update policy in status")
	})

	t.Run("NilRequest", func(t *testing.T) {
		updated, err := pap.UpdatePolicy(ctx, nil)

		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.Contains(t, err.Error(), "cannot be nil")
	})
}

// TestPAP_ActivatePolicy tests policy activation
func TestPAP_ActivatePolicy(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	t.Run("ActivateDraftPolicy", func(t *testing.T) {
		// Create a valid policy
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Test Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")

		require.NoError(t, err)

		// Verify policy is active
		activated, err := pap.GetPolicy(ctx, policy.PolicyID)
		require.NoError(t, err)
		assert.Equal(t, PolicyStatusActive, activated.Status)
		assert.NotNil(t, activated.ActivatedAt)
		assert.Equal(t, "approver-001", activated.Metadata["approved_by"])
	})

	t.Run("ActivateNonDraftPolicy", func(t *testing.T) {
		// Create and activate a policy
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Already Active",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
		require.NoError(t, err)

		// Try to activate again
		err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-002")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only draft policies can be activated")
	})

	t.Run("ActivateNonExistentPolicy", func(t *testing.T) {
		err := pap.ActivatePolicy(ctx, "non-existent", "approver-001")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy not found")
	})

	t.Run("ActivateInvalidPolicy", func(t *testing.T) {
		// Create a policy with no rules (invalid)
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Invalid Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules:      PolicyRules{}, // No allowed or denied actions
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy validation failed")
	})
}

// TestPAP_SuspendPolicy tests policy suspension
func TestPAP_SuspendPolicy(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	t.Run("SuspendActivePolicy", func(t *testing.T) {
		// Create and activate a policy
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Active Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
		require.NoError(t, err)

		// Suspend the policy
		err = pap.SuspendPolicy(ctx, policy.PolicyID, "Security incident detected")

		require.NoError(t, err)

		// Verify policy is suspended
		suspended, err := pap.GetPolicy(ctx, policy.PolicyID)
		require.NoError(t, err)
		assert.Equal(t, PolicyStatusSuspended, suspended.Status)
		assert.Equal(t, "Security incident detected", suspended.Metadata["suspension_reason"])
		assert.NotNil(t, suspended.Metadata["suspended_at"])
	})

	t.Run("SuspendNonActivePolicy", func(t *testing.T) {
		// Create a draft policy
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Draft Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.SuspendPolicy(ctx, policy.PolicyID, "Test reason")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only active policies can be suspended")
	})

	t.Run("SuspendNonExistentPolicy", func(t *testing.T) {
		err := pap.SuspendPolicy(ctx, "non-existent", "Test reason")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy not found")
	})
}

// TestPAP_RevokePolicy tests policy revocation
func TestPAP_RevokePolicy(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	t.Run("RevokeActivePolicy", func(t *testing.T) {
		// Create and activate a policy
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Active Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
		require.NoError(t, err)

		// Revoke the policy
		err = pap.RevokePolicy(ctx, policy.PolicyID, "Policy no longer needed")

		require.NoError(t, err)

		// Verify policy is revoked
		revoked, err := pap.GetPolicy(ctx, policy.PolicyID)
		require.NoError(t, err)
		assert.Equal(t, PolicyStatusRevoked, revoked.Status)
		assert.Equal(t, "Policy no longer needed", revoked.Metadata["revocation_reason"])
		assert.NotNil(t, revoked.Metadata["revoked_at"])
	})

	t.Run("RevokeAlreadyRevokedPolicy", func(t *testing.T) {
		// Create, activate, and revoke a policy
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Revoked Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
		require.NoError(t, err)

		err = pap.RevokePolicy(ctx, policy.PolicyID, "First revocation")
		require.NoError(t, err)

		// Try to revoke again
		err = pap.RevokePolicy(ctx, policy.PolicyID, "Second revocation")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy already revoked")
	})

	t.Run("RevokeNonExistentPolicy", func(t *testing.T) {
		err := pap.RevokePolicy(ctx, "non-existent", "Test reason")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy not found")
	})
}

// TestPAP_DeletePolicy tests policy deletion
func TestPAP_DeletePolicy(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	t.Run("DeleteDraftPolicy", func(t *testing.T) {
		// Create a draft policy
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Draft Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.DeletePolicy(ctx, policy.PolicyID)

		require.NoError(t, err)

		// Verify policy is deleted
		deleted, err := pap.GetPolicy(ctx, policy.PolicyID)
		assert.Error(t, err)
		assert.Nil(t, deleted)
	})

	t.Run("DeleteRevokedPolicy", func(t *testing.T) {
		// Create, activate, and revoke a policy
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Revoked Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
		require.NoError(t, err)

		err = pap.RevokePolicy(ctx, policy.PolicyID, "Test revocation")
		require.NoError(t, err)

		// Delete the revoked policy
		err = pap.DeletePolicy(ctx, policy.PolicyID)

		require.NoError(t, err)

		// Verify policy is deleted
		deleted, err := pap.GetPolicy(ctx, policy.PolicyID)
		assert.Error(t, err)
		assert.Nil(t, deleted)
	})

	t.Run("DeleteActivePolicy", func(t *testing.T) {
		// Create and activate a policy
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Active Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
		require.NoError(t, err)

		// Try to delete active policy
		err = pap.DeletePolicy(ctx, policy.PolicyID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete policy in status")
	})

	t.Run("DeleteNonExistentPolicy", func(t *testing.T) {
		err := pap.DeletePolicy(ctx, "non-existent")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "policy not found")
	})
}

// TestPAP_SearchPolicies tests policy search
func TestPAP_SearchPolicies(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	// Create multiple policies
	policies := []struct {
		policyType PolicyType
		name       string
		owner      string
		authorizer string
		tags       []string
		status     PolicyStatus
	}{
		{PolicyTypePoA, "PoA Policy 1", "client-001", "auth-001", []string{"healthcare", "test"}, PolicyStatusDraft},
		{PolicyTypeScope, "Scope Policy", "client-001", "auth-001", []string{"finance", "test"}, PolicyStatusDraft},
		{PolicyTypePoA, "PoA Policy 2", "client-002", "auth-002", []string{"healthcare", "prod"}, PolicyStatusDraft},
		{PolicyTypeRestriction, "Restriction Policy", "client-003", "auth-001", []string{"legal"}, PolicyStatusDraft},
	}

	for _, p := range policies {
		createRequest := &PolicyCreateRequest{
			PolicyType:       p.policyType,
			PolicyName:       p.name,
			ClientOwner:      p.owner,
			OwnersAuthorizer: p.authorizer,
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Tags: p.tags,
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		// Activate some policies
		if p.status == PolicyStatusActive {
			err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
			require.NoError(t, err)
		}
	}

	t.Run("SearchByPolicyType", func(t *testing.T) {
		criteria := &PolicySearchCriteria{
			PolicyTypes: []PolicyType{PolicyTypePoA},
		}

		results, err := pap.SearchPolicies(ctx, criteria)

		require.NoError(t, err)
		assert.Len(t, results, 2) // Two PoA policies
	})

	t.Run("SearchByClientOwner", func(t *testing.T) {
		criteria := &PolicySearchCriteria{
			ClientOwner: "client-001",
		}

		results, err := pap.SearchPolicies(ctx, criteria)

		require.NoError(t, err)
		assert.Len(t, results, 2) // Two policies for client-001
	})

	t.Run("SearchByOwnersAuthorizer", func(t *testing.T) {
		criteria := &PolicySearchCriteria{
			OwnersAuthorizer: "auth-001",
		}

		results, err := pap.SearchPolicies(ctx, criteria)

		require.NoError(t, err)
		assert.Len(t, results, 3) // Three policies with auth-001
	})

	t.Run("SearchByTags", func(t *testing.T) {
		criteria := &PolicySearchCriteria{
			Tags: []string{"healthcare"},
		}

		results, err := pap.SearchPolicies(ctx, criteria)

		require.NoError(t, err)
		assert.Len(t, results, 2) // Two healthcare policies
	})

	t.Run("SearchByMultipleTags", func(t *testing.T) {
		criteria := &PolicySearchCriteria{
			Tags: []string{"healthcare", "test"},
		}

		results, err := pap.SearchPolicies(ctx, criteria)

		require.NoError(t, err)
		assert.Len(t, results, 1) // Only one policy has both tags
	})

	t.Run("SearchByStatus", func(t *testing.T) {
		criteria := &PolicySearchCriteria{
			Statuses: []PolicyStatus{PolicyStatusDraft},
		}

		results, err := pap.SearchPolicies(ctx, criteria)

		require.NoError(t, err)
		assert.Len(t, results, 4) // All four are draft
	})

	t.Run("SearchWithLimit", func(t *testing.T) {
		criteria := &PolicySearchCriteria{
			Limit: 2,
		}

		results, err := pap.SearchPolicies(ctx, criteria)

		require.NoError(t, err)
		assert.LessOrEqual(t, len(results), 2) // At most 2 results
	})

	t.Run("SearchNoResults", func(t *testing.T) {
		criteria := &PolicySearchCriteria{
			ClientOwner: "non-existent-client",
		}

		results, err := pap.SearchPolicies(ctx, criteria)

		require.NoError(t, err)
		assert.Empty(t, results)
	})
}

// TestPAP_ListPolicies tests policy listing
func TestPAP_ListPolicies(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	// Create multiple policies with different statuses
	for i := 1; i <= 3; i++ {
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Policy " + string(rune('A'+i-1)),
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		// Activate the first two
		if i <= 2 {
			err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
			require.NoError(t, err)
		}
	}

	t.Run("ListAllPolicies", func(t *testing.T) {
		policies, err := pap.ListPolicies(ctx, nil)

		require.NoError(t, err)
		assert.Len(t, policies, 3)
	})

	t.Run("ListDraftPolicies", func(t *testing.T) {
		status := PolicyStatusDraft
		policies, err := pap.ListPolicies(ctx, &status)

		require.NoError(t, err)
		assert.Len(t, policies, 1) // Only one draft policy
	})

	t.Run("ListActivePolicies", func(t *testing.T) {
		status := PolicyStatusActive
		policies, err := pap.ListPolicies(ctx, &status)

		require.NoError(t, err)
		assert.Len(t, policies, 2) // Two active policies
	})

	t.Run("ListRevokedPolicies", func(t *testing.T) {
		status := PolicyStatusRevoked
		policies, err := pap.ListPolicies(ctx, &status)

		require.NoError(t, err)
		assert.Empty(t, policies) // No revoked policies
	})
}

// TestPAP_ValidatePolicy tests policy validation
func TestPAP_ValidatePolicy(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	t.Run("ValidPolicy", func(t *testing.T) {
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Valid Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read", "write"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		result, err := pap.ValidatePolicy(ctx, policy.PolicyID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.True(t, result.Valid)
		assert.Empty(t, result.Errors)
	})

	t.Run("InvalidPolicy_NoRules", func(t *testing.T) {
		createRequest := &PolicyCreateRequest{
			PolicyType:       PolicyTypePoA,
			PolicyName:       "Invalid Policy",
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules:      PolicyRules{}, // No allowed or denied actions
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		result, err := pap.ValidatePolicy(ctx, policy.PolicyID)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0], "at least one allowed or denied action")
	})

	t.Run("NonExistentPolicy", func(t *testing.T) {
		result, err := pap.ValidatePolicy(ctx, "non-existent")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "policy not found")
	})
}

// TestPAP_GetPolicyStatistics tests policy statistics
func TestPAP_GetPolicyStatistics(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	// Create policies with different types and statuses
	testPolicies := []struct {
		policyType PolicyType
		activate   bool
		suspend    bool
		revoke     bool
	}{
		{PolicyTypePoA, true, false, false},        // Active PoA
		{PolicyTypePoA, false, false, false},       // Draft PoA
		{PolicyTypeScope, true, false, false},      // Active Scope
		{PolicyTypeScope, true, true, false},       // Suspended Scope
		{PolicyTypeRestriction, true, false, true}, // Revoked Restriction
	}

	for i, tp := range testPolicies {
		createRequest := &PolicyCreateRequest{
			PolicyType:       tp.policyType,
			PolicyName:       "Policy " + string(rune('A'+i)),
			ClientOwner:      "client-001",
			OwnersAuthorizer: "authorizer-001",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"read"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}

		policy, err := pap.CreatePolicy(ctx, createRequest)
		require.NoError(t, err)

		if tp.activate {
			err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
			require.NoError(t, err)
		}

		if tp.suspend {
			err = pap.SuspendPolicy(ctx, policy.PolicyID, "Test suspension")
			require.NoError(t, err)
		}

		if tp.revoke {
			err = pap.RevokePolicy(ctx, policy.PolicyID, "Test revocation")
			require.NoError(t, err)
		}
	}

	stats, err := pap.GetPolicyStatistics(ctx)

	require.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, 5, stats.TotalPolicies)
	assert.Equal(t, 2, stats.ActivePolicies)    // 2 active policies
	assert.Equal(t, 1, stats.DraftPolicies)     // 1 draft policy
	assert.Equal(t, 1, stats.SuspendedPolicies) // 1 suspended policy
	assert.Equal(t, 1, stats.RevokedPolicies)   // 1 revoked policy

	// Check policies by type
	assert.Equal(t, 2, stats.PoliciesByType[PolicyTypePoA])
	assert.Equal(t, 2, stats.PoliciesByType[PolicyTypeScope])
	assert.Equal(t, 1, stats.PoliciesByType[PolicyTypeRestriction])
}

// TestPAP_ConcurrentAccess tests thread safety
func TestPAP_ConcurrentAccess(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	// Create multiple policies concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(index int) {
			createRequest := &PolicyCreateRequest{
				PolicyType:       PolicyTypePoA,
				PolicyName:       "Concurrent Policy " + string(rune('A'+index)),
				ClientOwner:      "client-001",
				OwnersAuthorizer: "authorizer-001",
				PolicyRules: PolicyRules{
					AllowedActions: []string{"read"},
				},
			}

			_, err := pap.CreatePolicy(ctx, createRequest)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all policies were created
	policies, err := pap.ListPolicies(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, policies, 10)
}

// TestPAP_PolicyLifecycleFlow tests complete policy lifecycle
func TestPAP_PolicyLifecycleFlow(t *testing.T) {
	pap := NewPowerAdministrationPoint("pap-001", "Test PAP", "Test PAP")
	ctx := context.Background()

	// 1. Create policy (draft)
	createRequest := &PolicyCreateRequest{
		PolicyType:       PolicyTypePoA,
		PolicyName:       "Lifecycle Test Policy",
		Description:      "Testing complete lifecycle",
		ClientOwner:      "client-001",
		OwnersAuthorizer: "authorizer-001",
		PolicyRules: PolicyRules{
			AllowedActions: []string{"read", "write"},
		},
		Scope: &PolicyScope{
			Countries: []string{"US", "CA"},
			Sectors:   []string{"healthcare"},
		},
		Tags: []string{"test", "lifecycle"},
	}

	policy, err := pap.CreatePolicy(ctx, createRequest)
	require.NoError(t, err)
	assert.Equal(t, PolicyStatusDraft, policy.Status)

	// 2. Update policy (still draft)
	updatedName := "Updated Lifecycle Test Policy"
	updateRequest := &PolicyUpdateRequest{
		PolicyID:   policy.PolicyID,
		PolicyName: &updatedName,
		ChangeLog:  "Updated policy name",
	}

	updated, err := pap.UpdatePolicy(ctx, updateRequest)
	require.NoError(t, err)
	assert.Equal(t, "Updated Lifecycle Test Policy", updated.PolicyName)
	assert.Equal(t, 2, updated.PolicyVersion)

	// 3. Validate policy
	validationResult, err := pap.ValidatePolicy(ctx, policy.PolicyID)
	require.NoError(t, err)
	assert.True(t, validationResult.Valid)

	// 4. Activate policy
	err = pap.ActivatePolicy(ctx, policy.PolicyID, "approver-001")
	require.NoError(t, err)

	active, err := pap.GetPolicy(ctx, policy.PolicyID)
	require.NoError(t, err)
	assert.Equal(t, PolicyStatusActive, active.Status)
	assert.NotNil(t, active.ActivatedAt)

	// 5. Suspend policy
	err = pap.SuspendPolicy(ctx, policy.PolicyID, "Temporary suspension for audit")
	require.NoError(t, err)

	suspended, err := pap.GetPolicy(ctx, policy.PolicyID)
	require.NoError(t, err)
	assert.Equal(t, PolicyStatusSuspended, suspended.Status)

	// 6. Revoke policy
	err = pap.RevokePolicy(ctx, policy.PolicyID, "Policy no longer needed")
	require.NoError(t, err)

	revoked, err := pap.GetPolicy(ctx, policy.PolicyID)
	require.NoError(t, err)
	assert.Equal(t, PolicyStatusRevoked, revoked.Status)
	assert.NotNil(t, revoked.RevokedAt)

	// 7. Delete policy (revoked policies can be deleted)
	err = pap.DeletePolicy(ctx, policy.PolicyID)
	require.NoError(t, err)

	// 8. Verify deletion
	deleted, err := pap.GetPolicy(ctx, policy.PolicyID)
	assert.Error(t, err)
	assert.Nil(t, deleted)
}
