package gnap

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/gauthplus"
	"github.com/mauriciomferz/Gauth_go/pkg/gnap"
)

type mockVerifService struct {
	gauthplus.VerificationService
	report *gauthplus.VerificationReport
}

func (m *mockVerifService) GenerateVerificationReport(ctx context.Context, poaID string, action gauthplus.Action) (*gauthplus.VerificationReport, error) {
	return m.report, nil
}

func TestHandler_LinkAgentAuthContext_Enrichment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Setup mock report
	report := &gauthplus.VerificationReport{
		PoAID: "poa-123",
		PoAVerification: &gauthplus.VerificationResult{
			IssuerID:  "issuer-001",
			GranteeID: "grantee-001",
			Scope:     []string{"read"},
		},
		ChainOfAuthority: []gauthplus.AuthorityLink{
			{
				GranteeID: "grantee-001",
				IssuerID:  "issuer-001",
				IsHuman:   false, // AI Agent
			},
			{
				GranteeID: "issuer-001",
				IssuerID:  "human-ceo",
				IsHuman:   true, // Human
			},
		},
		OverallValid: true,
		FiduciaryCompliance: &gauthplus.FiduciaryComplianceCheck{
			Compliant: true,
		},
		CapabilityCheck: &gauthplus.CapabilityVerificationCheck{
			Sufficient: true,
		},
	}

	mockVerif := &mockVerifService{report: report}
	store := gnap.NewMemoryGrantStore()
	handler := NewHandler(store, mockVerif, "http://localhost")

	t.Run("FullEnrichment_Success", func(t *testing.T) {
		resp := &gnap.GrantResponse{}
		req := &gnap.GrantRequest{
			PoACredentialRef: "poa-123",
			AccessToken: &gnap.AccessTokenRequest{
				Access: []gnap.AccessRight{
					{Type: "api", Identifier: "vault", Actions: []string{"read"}},
				},
			},
		}

		handler.linkAgentAuthContext(context.Background(), resp, req)

		if resp.ComplianceLevel != "high" {
			t.Errorf("Expected compliance_level 'high', got '%s'", resp.ComplianceLevel)
		}

		if len(resp.AuthorizationChain) != 2 {
			t.Errorf("Expected 2 chain links, got %d", len(resp.AuthorizationChain))
		}

		if resp.AuthorizationChain[0].EntityType != "ai_agent" {
			t.Errorf("Expected link[0] entity_type 'ai_agent', got '%s'", resp.AuthorizationChain[0].EntityType)
		}
		if resp.AuthorizationChain[1].EntityType != "human" {
			t.Errorf("Expected link[1] entity_type 'human', got '%s'", resp.AuthorizationChain[1].EntityType)
		}
	})

	t.Run("Compliance_Degraded_Fiduciary", func(t *testing.T) {
		report.FiduciaryCompliance.Compliant = false
		resp := &gnap.GrantResponse{}
		req := &gnap.GrantRequest{PoACredentialRef: "poa-123"}

		handler.linkAgentAuthContext(context.Background(), resp, req)

		if resp.ComplianceLevel != "degraded" {
			t.Errorf("Expected compliance_level 'degraded', got '%s'", resp.ComplianceLevel)
		}
	})

	t.Run("Compliance_Conditional_Capability", func(t *testing.T) {
		report.FiduciaryCompliance.Compliant = true
		report.CapabilityCheck.Sufficient = false
		resp := &gnap.GrantResponse{}
		req := &gnap.GrantRequest{PoACredentialRef: "poa-123"}

		handler.linkAgentAuthContext(context.Background(), resp, req)

		if resp.ComplianceLevel != "conditional" {
			t.Errorf("Expected compliance_level 'conditional', got '%s'", resp.ComplianceLevel)
		}
	})
}
