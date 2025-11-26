package test

import (
	"testing"

	policy "github.com/mauriciomferz/Gauth_go/pkg/policy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStubPolicyEngineAuthorization(t *testing.T) {
	eng := policy.NewStubEngine()
	dec, err := eng.EvaluateAuthorization(policy.AuthzInput{Subject: "user1", Action: "read", Resource: "res1"})
	require.NoError(t, err)
	assert.True(t, dec.Allow)
	assert.Equal(t, "ALLOW_STUB", dec.ReasonCode)
	require.NotNil(t, eng.LastAuthzInput)
	assert.Equal(t, "user1", eng.LastAuthzInput.Subject)
}

func TestStubPolicyEngineDelegation(t *testing.T) {
	eng := policy.NewStubEngine()
	dec, err := eng.EvaluateDelegation(policy.DelegationInput{PrincipalID: "p", DelegateID: "d", Scopes: []string{"x"}})
	require.NoError(t, err)
	assert.True(t, dec.Allow)
	assert.Equal(t, "ALLOW_STUB", dec.ReasonCode)
	require.NotNil(t, eng.LastDelegationInput)
	assert.Equal(t, "d", eng.LastDelegationInput.DelegateID)
}
