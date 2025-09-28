package auth

import (
	"context"
	"errors"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/common"
)

// RegistryVerifier is a no-op stub for compilation.
type RegistryVerifier interface {
	VerifyRegistration(ctx context.Context, info interface{}) error
	ValidateLegalStatus(ctx context.Context, ownerInfo interface{}) error
}

//nolint:unused // stub implementation for registry verification
type noopRegistryVerifier struct{}

//nolint:unused // stub implementation for registry verification
func (n *noopRegistryVerifier) VerifyRegistration(ctx context.Context, info interface{}) error {
	return nil
}

//nolint:unused // stub implementation for registry verification
func (n *noopRegistryVerifier) ValidateLegalStatus(ctx context.Context, ownerInfo interface{}) error {
	return nil
}

// IdentityVerificationService is a no-op stub for compilation.
type IdentityVerificationService interface {
	VerifyIdentity(ctx context.Context, id string) error
}

// ErrTokenNotFound is a stub error for compilation.
var ErrTokenNotFound = errors.New("token not found")

// --- Minimal no-op types for stubs ---

//nolint:unused // stub implementation for enhanced tokens
type noopEnhancedToken struct{}

//nolint:unused // stub implementation for enhanced token store
type noopEnhancedStore struct{}

type EnhancedToken struct{}

// Add IsExpired to noopEnhancedToken
//
//nolint:unused // stub implementation for enhanced tokens
func (t *noopEnhancedToken) IsExpired() bool { return false }

// Update noopEnhancedStore methods to use common types
//
//nolint:unused // stub implementation for enhanced token store
func (s *noopEnhancedStore) GetHumanVerification(ctx context.Context, token *EnhancedToken) (*common.HumanVerification, error) {
	return &common.HumanVerification{
		UltimateHumanID:          "stub-human",
		Role:                     "stub-role",
		LegalCapacityVerified:    true,
		CapacityVerificationTime: time.Now(),
		CapacityVerifier:         "stub-verifier",
		DelegationChain:          []common.DelegationLink{},
	}, nil
}

//nolint:unused // stub implementation for enhanced token store
func (s *noopEnhancedStore) GetSecondLevelApproval(ctx context.Context, token *EnhancedToken) (*common.SecondLevelApproval, error) {
	return &common.SecondLevelApproval{
		PrimaryApprover:       "stub-primary",
		PrimaryApprovalTime:   time.Now(),
		PrimaryRole:           "stub-role",
		SecondaryApprover:     "stub-secondary",
		SecondaryApprovalTime: time.Now(),
		SecondaryRole:         "stub-role",
		ApprovalLevel:         1,
		ApprovalScope:         []string{"stub-scope"},
		ApprovalDuration:      0,
		JurisdictionRules:     nil,
	}, nil
}

// AuditEvent stub for extended_controls.go
// (Fields: Time, Type, RuleID, Result, Details, Evidence)
type AuditEvent struct {
	Time     time.Time
	Type     string
	RuleID   string
	Result   bool
	Details  string
	Evidence map[string]interface{}
}

// Add RecordAuditEvent to StandardComplianceTracker stub
func (t *StandardComplianceTracker) RecordAuditEvent(ctx context.Context, event *AuditEvent) error {
	return nil
}

// Fix: ApprovalRule pointer compatibility
// Remove ApprovalRule redeclaration from stubs.go (should only exist in one place)

// Fix: checkCustomLimits expects map[string]interface{}, update call site in compliance.go
// (No need to add stub here, fix should be in compliance.go)

// Remove redeclared HumanVerification, DelegationLink, and SecondLevelApproval from stubs.go
// These types are already defined in enhanced_authorization.go
// Only keep the noopEnhancedStore methods
