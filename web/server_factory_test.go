package web

import (
	"testing"

	"github.com/mauriciomferz/AgentAuth/pkg/crypto/keys"
	"github.com/stretchr/testify/assert"
)

func TestNewBetaServer_KeyManagerSelection(t *testing.T) {
	// Test Default (Local)
	t.Setenv("AGENTAUTH_KMS_PROVIDER", "")
	s1 := NewBetaServer("8081")
	assert.NotNil(t, s1.tokenHandler)
	assert.IsType(t, &keys.LocalKeyManager{}, s1.tokenHandler.JWTKeyManager)

	// Test External (Simulated)
	t.Setenv("AGENTAUTH_KMS_PROVIDER", "external")
	defer t.Setenv("AGENTAUTH_KMS_PROVIDER", "")

	s2 := NewBetaServer("8082")
	assert.NotNil(t, s2.tokenHandler)
	assert.IsType(t, &keys.ExternalKeyManager{}, s2.tokenHandler.JWTKeyManager)
}
