package gauth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper function to create a test PAP instance
func newTestPAP() *PowerAdministrationPoint {
	return NewPowerAdministrationPoint("test-pap-001", "Test PAP", "PAP for integration testing")
}

func TestSimplePDP_PAPIntegration(t *testing.T) {
	t.Run("PDP without PAP fails policy operations", func(t *testing.T) {
		pdp := NewSimplePDP()
		
		policy := &AuthorizationPolicy{
			PolicyName:       "test",
			PolicyType:       PolicyTypePoA,
			ClientOwner:      "owner",
			OwnersAuthorizer: "authorizer",
			PolicyRules:      PolicyRules{AllowedActions: []string{"test"}},
		}
		
		err := pdp.AddPolicy("", policy)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "PAP not configured")
	})
	
	t.Run("PDP with PAP successfully manages policies", func(t *testing.T) {
		pap := newTestPAP()
		pdp := NewSimplePDPWithPAP(pap)
		
		policy := &AuthorizationPolicy{
			PolicyName:       "test-policy",
			PolicyType:       PolicyTypePoA,
			ClientOwner:      "owner-123",
			OwnersAuthorizer: "authorizer-456",
			PolicyRules: PolicyRules{
				AllowedActions: []string{"execute"},
			},
			Scope: &PolicyScope{
				Countries: []string{"US"},
			},
		}
		
		// Add policy
		err := pdp.AddPolicy("", policy)
		require.NoError(t, err)
		assert.NotEmpty(t, policy.PolicyID)
		
		// Get policy
		retrieved, err := pdp.GetPolicy(policy.PolicyID)
		require.NoError(t, err)
		assert.Equal(t, "test-policy", retrieved.PolicyName)
		
		// Activate via PAP
		err = pap.ActivatePolicy(context.Background(), policy.PolicyID, "approver")
		require.NoError(t, err)
		
		// List active
		active, err := pdp.ListActivePolicies()
		require.NoError(t, err)
		assert.Len(t, active, 1)
		
		// Revoke via PAP
		err = pap.RevokePolicy(context.Background(), policy.PolicyID, "test")
		require.NoError(t, err)
		
		// Remove
		err = pdp.RemovePolicy(policy.PolicyID)
		require.NoError(t, err)
	})
}
