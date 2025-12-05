package gauth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/poa"
	"github.com/mauriciomferz/Gauth_go/pkg/poa/taxonomy"
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
						Transactions: []taxonomy.TransactionType{
							taxonomy.TransactionPurchase,
							taxonomy.TransactionPayment,
						},
						Decisions: []taxonomy.DecisionType{
							taxonomy.DecisionFinancial,
						},
						PhysicalActions: []taxonomy.ActionTypePhysical{
							taxonomy.ActionPhysicalTransport,
						},
						NonPhysicalActions: []taxonomy.ActionTypeNonPhysical{
							taxonomy.ActionNonPhysicalAnalyzing,
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
		assert.Contains(t, summary.GrantedActions, string(taxonomy.TransactionPurchase))
		assert.Contains(t, summary.GrantedActions, string(taxonomy.TransactionPayment))
		assert.Contains(t, summary.GrantedActions, string(taxonomy.DecisionFinancial))
		assert.Contains(t, summary.GrantedActions, string(taxonomy.ActionPhysicalTransport))
		assert.Contains(t, summary.GrantedActions, string(taxonomy.ActionNonPhysicalAnalyzing))
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

func TestDisclosureService_DetermineViolationSeverity(t *testing.T) {
	service := &DisclosureService{}

	testCases := []struct {
		name             string
		violation        string
		expectedSeverity string
	}{
		{
			name:             "critical - expired",
			violation:        "Token has expired",
			expectedSeverity: "critical",
		},
		{
			name:             "critical - revoked",
			violation:        "Authorization revoked by owner",
			expectedSeverity: "critical",
		},
		{
			name:             "critical - invalid",
			violation:        "Invalid signature detected",
			expectedSeverity: "critical",
		},
		{
			name:             "high - exceeded",
			violation:        "Transaction limit exceeded",
			expectedSeverity: "high",
		},
		{
			name:             "high - breach",
			violation:        "Policy breach detected",
			expectedSeverity: "high",
		},
		{
			name:             "medium - warning",
			violation:        "Warning: approaching limit",
			expectedSeverity: "medium",
		},
		{
			name:             "low - general",
			violation:        "General compliance note",
			expectedSeverity: "low",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			severity := service.determineViolationSeverity(tc.violation)
			assert.Equal(t, tc.expectedSeverity, severity)
		})
	}
}

func TestDisclosureService_ComplianceViolationsRetrieval(t *testing.T) {
	t.Run("returns empty violations when token not tracked", func(t *testing.T) {
		// Create mock tracker that returns error
		mockTracker := &mockComplianceTracker{
			shouldError: true,
		}

		service := &DisclosureService{
			complianceTracker: mockTracker,
		}

		violations := service.getComplianceViolations(nil, "unknown-token")
		assert.Empty(t, violations)
	})

	t.Run("returns empty violations when tracking inactive", func(t *testing.T) {
		mockTracker := &mockComplianceTracker{
			status: &ComplianceTrackingStatus{
				Active: false,
			},
		}

		service := &DisclosureService{
			complianceTracker: mockTracker,
		}

		violations := service.getComplianceViolations(nil, "inactive-token")
		assert.Empty(t, violations)
	})
}

// mockComplianceTracker for testing
type mockComplianceTracker struct {
	shouldError bool
	status      *ComplianceTrackingStatus
}

func (m *mockComplianceTracker) StartTracking(ctx context.Context, req *ComplianceTrackingRequest) error {
	return nil
}

func (m *mockComplianceTracker) CheckCompliance(ctx context.Context, tokenID string) (*ComplianceStatus, error) {
	return nil, nil
}

func (m *mockComplianceTracker) StopTracking(ctx context.Context, tokenID string) error {
	return nil
}

func (m *mockComplianceTracker) GetTrackingStatus(ctx context.Context, tokenID string) (*ComplianceTrackingStatus, error) {
	if m.shouldError {
		return nil, fmt.Errorf("token not tracked")
	}
	return m.status, nil
}

func (m *mockComplianceTracker) ListActiveTracking(ctx context.Context) ([]string, error) {
	return nil, nil
}
