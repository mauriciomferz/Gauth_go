package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/pdp"
)

// CapturingPDPEngine captures requests for verification
type CapturingPDPEngine struct {
	LastRequest pdp.Request
	Decision    pdp.Decision
}

func (m *CapturingPDPEngine) Evaluate(ctx context.Context, req pdp.Request) (pdp.Decision, error) {
	m.LastRequest = req
	return m.Decision, nil
}

func (m *CapturingPDPEngine) Metrics() pdp.MetricsSnapshot {
	return pdp.MetricsSnapshot{}
}

func TestMCP_Agent_Identity_Propagation(t *testing.T) {
	// Setup
	pdpEngine := &CapturingPDPEngine{
		Decision: pdp.Decision{Allow: true, Reason: "permitted"},
	}
	bridge := NewAuthorizationBridge(pdpEngine)

	// Create Extended Token with Chain
	token := &agentauth.ExtendedToken{
		AccessToken:     "test-token",
		ExpiresIn:       3600,
		IssuedAt:        time.Now(),
		ComplianceLevel: "high",
		Scope:           []string{"mcp:tool:call"},
		AuthorizationChain: &agentauth.AuthorizationChain{
			Client: &agentauth.AuthorizationLink{
				EntityID:         "agent-007",
				EntityName:       "Bond Agent",
				EntityType:       "ai_system",
				AuthorizedBy:     "owner-1",
				Status:           "active",
				ValidUntil:       time.Now().Add(24 * time.Hour),
				IdentityVerified: true,
			},
			OwnersAuthorizer: &agentauth.AuthorizationLink{
				EntityID:         "auth-1",
				Status:           "active",
				ValidUntil:       time.Now().Add(24 * time.Hour),
				IdentityVerified: true,
			},
			ClientOwner: &agentauth.AuthorizationLink{
				EntityID:         "owner-1",
				AuthorizedBy:     "auth-1",
				Status:           "active",
				ValidUntil:       time.Now().Add(24 * time.Hour),
				IdentityVerified: true,
			},
		},
		ClientOwner:      &agentauth.ClientOwnerInfo{OwnerID: "owner-1"},
		OwnersAuthorizer: &agentauth.OwnersAuthorizerInfo{AuthorizerID: "auth-1"},
		VerificationProof: &agentauth.IdentityVerificationChain{
			VerificationLevels: []agentauth.VerificationLevel{
				{
					EntityID:       "agent-007",
					AssuranceLevel: "substantial",
				},
			},
			OverallVerification: "verified",
		},
		LegalFramework: &agentauth.LegalFrameworkInfo{
			Jurisdiction: "US",
		},
		IssuedBy: &agentauth.AuthorizationServerInfo{
			ServerID: "gauth-test",
			Issuer:   "https://agentauth.test",
		},
		JurisdictionContext: &agentauth.JurisdictionContext{
			PrimaryJurisdiction: "US",
		},
	}

	// Test Tool Call
	t.Run("AuthorizeToolCall_PropagatesIdentity", func(t *testing.T) {
		_, err := bridge.AuthorizeToolCall(context.Background(), token, "payment_tool", map[string]interface{}{
			"amount": 100.0,
		})
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		// Verify PDP Request Attributes
		req := pdpEngine.LastRequest
		if req.Attributes["agent_id"] != "agent-007" {
			t.Errorf("Expected agent_id 'agent-007', got '%s'", req.Attributes["agent_id"])
		}
		if req.Attributes["agent_name"] != "Bond Agent" {
			t.Errorf("Expected agent_name 'Bond Agent', got '%s'", req.Attributes["agent_name"])
		}
		if req.Attributes["agent_assurance"] != "substantial" {
			t.Errorf("Expected agent_assurance 'substantial', got '%s'", req.Attributes["agent_assurance"])
		}
	})

	// Test Resource Read
	t.Run("AuthorizeResourceRead_PropagatesIdentity", func(t *testing.T) {
		token.Scope = []string{"mcp:resource:read"}
		_, err := bridge.AuthorizeResourceRead(context.Background(), token, "file:///secret.txt")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		req := pdpEngine.LastRequest
		if req.Attributes["agent_id"] != "agent-007" {
			t.Errorf("Expected agent_id 'agent-007', got '%s'", req.Attributes["agent_id"])
		}
		if req.Attributes["resource_type"] != "file" {
			t.Errorf("Expected resource_type 'file', got '%s'", req.Attributes["resource_type"])
		}
	})
}
