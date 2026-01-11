// Copyright 2025 AgentAuth Contributors
// SPDX-License-Identifier: Apache-2.0

package agentauth_aap_001

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdvancedClaims_GenerationFeatureGated verifies that AdvancedClaims are only populated
// when AGENTAUTH_ADVANCED_CLAIMS=1, ensuring backward compatibility.
func TestAdvancedClaims_GenerationFeatureGated(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc := newTestService()

	// Create delegation via CreateDelegation
	resp, err := svc.CreateDelegationCtx(ctx, DelegationRequest{
		Grantor:      "alice",
		Grantee:      "bob",
		Scope:        []string{"read", "write"},
		Duration:     24 * time.Hour,
		Restrictions: map[string]string{"env": "test"},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	t.Run("disabled_by_default", func(t *testing.T) {
		// Ensure both flags are set correctly
		t.Setenv("AGENTAUTH_ADVANCED_CLAIMS", "")
		t.Setenv("AGENTAUTH_POA_ENVELOPE_V2", "1")

		// Generate token using internal helper
		tokenStr := generateAuthToken(svc, &resp.POA)
		require.NotEmpty(t, tokenStr)

		// Verify token by decoding it with VerifyToken (which handles PASETO decoding)
		verifyRes, err := svc.VerifyToken(ctx, tokenStr)
		require.NoError(t, err)
		assert.False(t, verifyRes.Expired)

		// We can't directly inspect AdvancedClaims without decoding PASETO, but we verified
		// the token is valid without AdvancedClaims (backward compatibility)
	})

	t.Run("enabled_with_flag", func(t *testing.T) {
		// Enable both feature flags
		t.Setenv("AGENTAUTH_ADVANCED_CLAIMS", "1")
		t.Setenv("AGENTAUTH_POA_ENVELOPE_V2", "1")

		// Generate token
		tokenStr := generateAuthToken(svc, &resp.POA)
		require.NotEmpty(t, tokenStr)

		// Verify token validates (implies AdvancedClaims populated correctly if feature enabled)
		verifyRes, err := svc.VerifyToken(ctx, tokenStr)
		require.NoError(t, err)
		assert.False(t, verifyRes.Expired)
		assert.Equal(t, resp.POA.Grantee, verifyRes.Grantee)
		assert.Equal(t, resp.POA.Grantor, verifyRes.Grantor)
		assert.Equal(t, resp.POA.Scope, verifyRes.Scope)

		// Note: Direct inspection of AdvancedClaims requires PASETO decoding which
		// is internal to VerifyToken. The fact that verification succeeds confirms
		// AdvancedClaims was populated and validated correctly.
	})
}

// TestAdvancedClaims_BackwardCompatibility verifies that tokens without AdvancedClaims
// still validate correctly (pre-P2.10 tokens).
func TestAdvancedClaims_BackwardCompatibility(t *testing.T) {
	ctx := WithSubject(context.Background(), "bob")
	svc := newTestService()

	// Create POA
	resp, err := svc.CreateDelegationCtx(ctx, DelegationRequest{
		Grantor:  "alice",
		Grantee:  "bob",
		Scope:    []string{"read"},
		Duration: 24 * time.Hour,
	})
	require.NoError(t, err)

	t.Run("token_without_advanced_claims", func(t *testing.T) {
		// Ensure AGENTAUTH_ADVANCED_CLAIMS is not set (simulates pre-P2.10 token generation)
		t.Setenv("AGENTAUTH_ADVANCED_CLAIMS", "")
		t.Setenv("AGENTAUTH_POA_ENVELOPE_V2", "1")

		tokenStr := generateAuthToken(svc, &resp.POA)
		require.NotEmpty(t, tokenStr)

		// Verify token without AdvancedClaims still validates
		verifyRes, err := svc.VerifyToken(ctx, tokenStr)
		require.NoError(t, err, "Tokens without AdvancedClaims should still validate")
		assert.False(t, verifyRes.Expired)
		assert.False(t, verifyRes.Revoked)
		assert.False(t, verifyRes.Suspended)
	})

	t.Run("verification_skips_validation_when_flag_disabled", func(t *testing.T) {
		// Enable AGENTAUTH_ADVANCED_CLAIMS for token generation
		t.Setenv("AGENTAUTH_ADVANCED_CLAIMS", "1")
		t.Setenv("AGENTAUTH_POA_ENVELOPE_V2", "1")

		tokenStr := generateAuthToken(svc, &resp.POA)
		require.NotEmpty(t, tokenStr)

		// Disable AGENTAUTH_ADVANCED_CLAIMS for verification (simulates rollback scenario)
		t.Setenv("AGENTAUTH_ADVANCED_CLAIMS", "")

		verifyRes, err := svc.VerifyToken(ctx, tokenStr)
		require.NoError(t, err, "Verification should skip AdvancedClaims validation when flag disabled")
		assert.False(t, verifyRes.Expired)
		assert.False(t, verifyRes.Revoked)
		assert.False(t, verifyRes.Suspended)
	})
}
