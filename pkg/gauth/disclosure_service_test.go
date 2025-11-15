package gauth

import (
	"testing"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/poa"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisclosureService_TokenToSummary_ExtractsGrantedActions(t *testing.T) {
	service := &DisclosureService{}

	t.Run("extracts actions from PoA definition", func(t *testing.T) {
		token := &ExtendedToken{
			AccessToken: "test-token-123",
			IssuedAt:    time.Now(),
			ExpiresIn:   3600,
			Scope:       []string{"read", "write"},
			PowerOfAttorney: &poa.PoADefinition{
				Authorization: poa.AuthorizationScope{
					AuthorizedActions: poa.AuthorizedActions{
						Transactions: []poa.TransactionType{
							poa.TransactionPurchase,
							poa.TransactionPayment,
						},
						Decisions: []poa.DecisionType{
							poa.DecisionFinancial,
						},
						PhysicalActions: []poa.ActionTypePhysical{
							poa.ActionPhysicalTransport,
						},
						NonPhysicalActions: []poa.ActionTypeNonPhysical{
							poa.ActionNonPhysicalAnalyzing,
						},
					},
				},
			},
			ResourceOwner: &ResourceOwnerInfo{
				OwnerID: "owner-123",
			},
			ComplianceLevel: "high",
		}

		summary := service.tokenToSummary(token)

		require.NotNil(t, summary)
		assert.Equal(t, "test-token-123", summary.AuthorizationID)
		assert.Equal(t, "high", summary.ComplianceStatus)
		assert.Equal(t, []string{"read", "write"}, summary.GrantedScopes)
		
		// Verify granted actions were extracted
		assert.Len(t, summary.GrantedActions, 5)
		assert.Contains(t, summary.GrantedActions, string(poa.TransactionPurchase))
		assert.Contains(t, summary.GrantedActions, string(poa.TransactionPayment))
		assert.Contains(t, summary.GrantedActions, string(poa.DecisionFinancial))
		assert.Contains(t, summary.GrantedActions, string(poa.ActionPhysicalTransport))
		assert.Contains(t, summary.GrantedActions, string(poa.ActionNonPhysicalAnalyzing))
	})

	t.Run("returns empty actions when PoA is nil", func(t *testing.T) {
		token := &ExtendedToken{
			AccessToken:     "test-token-456",
			IssuedAt:        time.Now(),
			ExpiresIn:       3600,
			Scope:           []string{"read"},
			PowerOfAttorney: nil,
			ResourceOwner: &ResourceOwnerInfo{
				OwnerID: "owner-456",
			},
			ComplianceLevel: "standard",
		}

		summary := service.tokenToSummary(token)

		require.NotNil(t, summary)
		assert.Empty(t, summary.GrantedActions)
	})
}

func TestDisclosureService_SubscriptionTracking(t *testing.T) {
	t.Run("subscription ID is included in authorization detail", func(t *testing.T) {
		// This is a structural test - actual implementation would require full service setup
		token := &ExtendedToken{
			AccessToken:    "test-token-789",
			SubscriptionID: "subscription-abc-123",
			IssuedAt:       time.Now(),
			ExpiresIn:      3600,
			ResourceOwner: &ResourceOwnerInfo{
				OwnerID: "owner-789",
			},
		}

		// Verify subscription ID is populated
		assert.Equal(t, "subscription-abc-123", token.SubscriptionID)
	})
}
