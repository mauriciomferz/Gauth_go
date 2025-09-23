package auth

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockStore struct {
	mock.Mock
}

func (m *MockStore) GetJurisdictionRules(jurisdiction string) (*JurisdictionRules, error) {
	args := m.Called(jurisdiction)
	if rules, ok := args.Get(0).(*JurisdictionRules); ok {
		return rules, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStore) StoreJurisdictionRules(rules *JurisdictionRules) error {
	args := m.Called(rules)
	return args.Error(0)
}

func (m *MockStore) RecordApprovalEvent(ctx context.Context, event *ApprovalEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

type MockVerifier struct {
	mock.Mock
}

func (m *MockVerifier) ValidateScope(ctx context.Context, scope string) error {
	args := m.Called(ctx, scope)
	return args.Error(0)
}

func (m *MockVerifier) ValidateRule(ctx context.Context, rule string, duty FiduciaryDuty, validationCtx *ValidationContext) error {
	args := m.Called(ctx, rule, duty, validationCtx)
	return args.Error(0)
}

func (m *MockVerifier) VerifyMultiLevelApproval(ctx context.Context, approvalCtx *ApprovalContext) error {
	args := m.Called(ctx, approvalCtx)
	return args.Error(0)
}

func (m *MockVerifier) VerifyDualApproval(ctx context.Context, approvalCtx *ApprovalContext) error {
	args := m.Called(ctx, approvalCtx)
	return args.Error(0)
}

func (m *MockVerifier) GetActionValue(ctx context.Context, action string) (float64, error) {
	args := m.Called(ctx, action)
	return args.Get(0).(float64), args.Error(1)
}

func (m *MockVerifier) VerifyRole(ctx context.Context, role string, roleCtx *RoleContext) error {
	args := m.Called(ctx, role, roleCtx)
	return args.Error(0)
}

func (m *MockVerifier) ValidateComplianceRule(ctx context.Context, rule string, complianceCtx *ComplianceContext) error {
	args := m.Called(ctx, rule, complianceCtx)
	return args.Error(0)
}

func (m *MockVerifier) IsInScope(action, scope string) bool {
	args := m.Called(action, scope)
	return args.Bool(0)
}

func (m *MockVerifier) VerifyIssuer(ctx context.Context, issuerID string, issuerCtx *IssuerContext) error {
	args := m.Called(ctx, issuerID, issuerCtx)
	return args.Error(0)
}

func (m *MockVerifier) VerifyProofJurisdiction(ctx context.Context, proof *CapacityProof, jurisdictionCtx *JurisdictionContext) error {
	args := m.Called(ctx, proof, jurisdictionCtx)
	return args.Error(0)
}

func (m *MockVerifier) VerifyProofEvidence(ctx context.Context, proof string, evidenceCtx *EvidenceContext) error {
	args := m.Called(ctx, proof, evidenceCtx)
	return args.Error(0)
}

type MockRegister struct {
	mock.Mock
}

func (m *MockRegister) GetJurisdictionRules(jurisdiction string) (*JurisdictionRules, error) {
	args := m.Called(jurisdiction)
	if rules, ok := args.Get(0).(*JurisdictionRules); ok {
		return rules, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockRegister) VerifyRegistration(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestValidateDuty(t *testing.T) {
	tests := []struct {
		name    string
		duty    FiduciaryDuty
		setup   func(*MockVerifier)
		wantErr bool
	}{
		{
			name: "valid duty",
			duty: FiduciaryDuty{
				Type:        "loyalty",
				Description: "Primary loyalty duty",
				Scope:       []string{"financial_decisions"},
				Validation:  []string{"rule1"},
			},
			setup: func(v *MockVerifier) {
				v.On("ValidateScope", mock.Anything, "financial_decisions").Return(nil)
				v.On("ValidateRule", mock.Anything, "rule1", mock.Anything, mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "invalid duty type",
			duty: FiduciaryDuty{
				Type:        "invalid",
				Description: "Invalid duty",
				Scope:       []string{"scope"},
				Validation:  []string{"rule1"},
			},
			setup:   func(v *MockVerifier) {},
			wantErr: true,
		},
		{
			name: "empty scope",
			duty: FiduciaryDuty{
				Type:        "loyalty",
				Description: "Empty scope duty",
				Scope:       []string{},
				Validation:  []string{"rule1"},
			},
			setup:   func(v *MockVerifier) {},
			wantErr: true,
		},
		{
			name: "validation rule failure",
			duty: FiduciaryDuty{
				Type:        "loyalty",
				Description: "Failed validation",
				Scope:       []string{"scope1"},
				Validation:  []string{"rule1"},
			},
			setup: func(v *MockVerifier) {
				v.On("ValidateScope", mock.Anything, "scope1").Return(nil)
				v.On("ValidateRule", mock.Anything, "rule1", mock.Anything, mock.Anything).
					Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &MockVerifier{}
			tt.setup(verifier)

			framework := &StandardLegalFramework{
				verifier: verifier,
			}

			ctx := context.WithValue(context.Background(), "entity_id", "test_entity")
			ctx = context.WithValue(ctx, "delegation_chain", []string{"chain1"})

			err := framework.validateDuty(ctx, tt.duty)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			verifier.AssertExpectations(t)
		})
	}
}

func TestGetJurisdictionRules(t *testing.T) {
	tests := []struct {
		name         string
		jurisdiction string
		setupStore   func(*MockStore)
		setupReg     func(*MockRegister)
		want         *JurisdictionRules
		wantErr      bool
	}{
		{
			name:         "cache hit",
			jurisdiction: "US",
			setupStore: func(s *MockStore) {
				rules := &JurisdictionRules{Country: "US"}
				s.On("GetJurisdictionRules", "US").Return(rules, nil)
			},
			setupReg: func(r *MockRegister) {},
			want:     &JurisdictionRules{Country: "US"},
			wantErr:  false,
		},
		{
			name:         "cache miss with registry success",
			jurisdiction: "UK",
			setupStore: func(s *MockStore) {
				s.On("GetJurisdictionRules", "UK").Return(nil, assert.AnError)
				rules := &JurisdictionRules{Country: "UK"}
				s.On("StoreJurisdictionRules", rules).Return(nil)
			},
			setupReg: func(r *MockRegister) {
				rules := &JurisdictionRules{Country: "UK"}
				r.On("GetJurisdictionRules", "UK").Return(rules, nil)
			},
			want:    &JurisdictionRules{Country: "UK"},
			wantErr: false,
		},
		{
			name:         "complete failure",
			jurisdiction: "invalid",
			setupStore: func(s *MockStore) {
				s.On("GetJurisdictionRules", "invalid").Return(nil, assert.AnError)
			},
			setupReg: func(r *MockRegister) {
				r.On("GetJurisdictionRules", "invalid").Return(nil, assert.AnError)
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := &MockStore{}
			register := &MockRegister{}
			tt.setupStore(store)
			tt.setupReg(register)

			framework := &StandardLegalFramework{
				store:    store,
				register: register,
			}

			got, err := framework.getJurisdictionRules(tt.jurisdiction)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}

			store.AssertExpectations(t)
			register.AssertExpectations(t)
		})
	}
}

func TestVerifyCapacityProof(t *testing.T) {
	tests := []struct {
		name      string
		proof     *CapacityProof
		setupMock func(*MockVerifier)
		wantErr   bool
	}{
		{
			name: "valid proof",
			proof: &CapacityProof{
				Type:         "court_order",
				IssuedAt:     time.Now().Add(-24 * time.Hour),
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				IssuerID:     "issuer1",
				Proof:        "valid_proof",
				Jurisdiction: "US",
			},
			setupMock: func(v *MockVerifier) {
				v.On("VerifyIssuer", mock.Anything, "issuer1", mock.Anything).Return(nil)
				v.On("VerifyProofJurisdiction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
				v.On("VerifyProofEvidence", mock.Anything, "valid_proof", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "expired proof",
			proof: &CapacityProof{
				Type:         "court_order",
				IssuedAt:     time.Now().Add(-48 * time.Hour),
				ExpiresAt:    time.Now().Add(-24 * time.Hour),
				IssuerID:     "issuer1",
				Proof:        "expired_proof",
				Jurisdiction: "US",
			},
			setupMock: func(v *MockVerifier) {},
			wantErr:   true,
		},
		{
			name: "future proof",
			proof: &CapacityProof{
				Type:         "court_order",
				IssuedAt:     time.Now().Add(24 * time.Hour),
				ExpiresAt:    time.Now().Add(48 * time.Hour),
				IssuerID:     "issuer1",
				Proof:        "future_proof",
				Jurisdiction: "US",
			},
			setupMock: func(v *MockVerifier) {},
			wantErr:   true,
		},
		{
			name: "invalid issuer",
			proof: &CapacityProof{
				Type:         "court_order",
				IssuedAt:     time.Now().Add(-24 * time.Hour),
				ExpiresAt:    time.Now().Add(24 * time.Hour),
				IssuerID:     "invalid_issuer",
				Proof:        "valid_proof",
				Jurisdiction: "US",
			},
			setupMock: func(v *MockVerifier) {
				v.On("VerifyIssuer", mock.Anything, "invalid_issuer", mock.Anything).Return(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &MockVerifier{}
			tt.setupMock(verifier)

			framework := &StandardLegalFramework{
				verifier: verifier,
			}

			ctx := context.WithValue(context.Background(), "entity_type", "organization")
			err := framework.verifyCapacityProof(ctx, tt.proof)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			verifier.AssertExpectations(t)
		})
	}
}

func TestIsActionAllowed(t *testing.T) {
	tests := []struct {
		name           string
		action         string
		allowedActions []string
		setupVerifier  func(*MockVerifier)
		want           bool
	}{
		{
			name:           "direct match",
			action:         "read:document",
			allowedActions: []string{"read:document"},
			setupVerifier:  func(v *MockVerifier) {},
			want:           true,
		},
		{
			name:           "wildcard match",
			action:         "read:document",
			allowedActions: []string{"read:*"},
			setupVerifier:  func(v *MockVerifier) {},
			want:           true,
		},
		{
			name:           "scope match",
			action:         "write:document",
			allowedActions: []string{"scope:documents"},
			setupVerifier: func(v *MockVerifier) {
				v.On("IsInScope", "write:document", "documents").Return(true)
			},
			want: true,
		},
		{
			name:           "no match",
			action:         "delete:document",
			allowedActions: []string{"read:*", "write:*"},
			setupVerifier:  func(v *MockVerifier) {},
			want:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &MockVerifier{}
			tt.setupVerifier(verifier)

			framework := &StandardLegalFramework{
				verifier: verifier,
			}

			got := framework.isActionAllowed(tt.action, tt.allowedActions)
			assert.Equal(t, tt.want, got)

			verifier.AssertExpectations(t)
		})
	}
}

func TestValidateJurisdictionRequirements(t *testing.T) {
	tests := []struct {
		name       string
		rules      *JurisdictionRules
		action     string
		setupMocks func(*MockVerifier)
		wantErr    bool
	}{
		{
			name: "valid multi-level approval",
			rules: &JurisdictionRules{
				Country: "US",
				RequiredApprovals: map[string]ApprovalLevel{
					"high_value_transaction": MultiLevelApproval,
				},
				ValueLimits: map[string]float64{
					"high_value_transaction": 1000000,
				},
				RequiredRoles:   []string{"executive"},
				ComplianceRules: []string{"rule1"},
			},
			action: "high_value_transaction",
			setupMocks: func(v *MockVerifier) {
				v.On("VerifyMultiLevelApproval", mock.Anything, mock.Anything).Return(nil)
				v.On("GetActionValue", mock.Anything, "high_value_transaction").Return(float64(500000), nil)
				v.On("VerifyRole", mock.Anything, "executive", mock.Anything).Return(nil)
				v.On("ValidateComplianceRule", mock.Anything, "rule1", mock.Anything).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed value limit",
			rules: &JurisdictionRules{
				Country: "US",
				ValueLimits: map[string]float64{
					"high_value_transaction": 1000000,
				},
			},
			action: "high_value_transaction",
			setupMocks: func(v *MockVerifier) {
				v.On("GetActionValue", mock.Anything, "high_value_transaction").Return(float64(2000000), nil)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verifier := &MockVerifier{}
			tt.setupMocks(verifier)

			framework := &StandardLegalFramework{
				verifier: verifier,
			}

			ctx := context.WithValue(context.Background(), "entity_type", "organization")
			err := framework.validateJurisdictionRequirements(ctx, tt.rules, tt.action)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			verifier.AssertExpectations(t)
		})
	}
}
