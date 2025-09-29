package auth

import (
	"context"
	"errors"
	"time"
)

// RegistryVerifier is a no-op stub for compilation.
type RegistryVerifier interface {
	VerifyRegistration(ctx context.Context, info interface{}) error
	ValidateLegalStatus(ctx context.Context, ownerInfo interface{}) error
}



// IdentityVerificationService is a no-op stub for compilation.
type IdentityVerificationService interface {
	VerifyIdentity(ctx context.Context, id string) error
}

// ErrTokenNotFound is a stub error for compilation.
var ErrTokenNotFound = errors.New("token not found")

// --- Minimal no-op types for stubs ---





type EnhancedToken struct{}





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
func (t *StandardComplianceTracker) RecordAuditEvent(_ context.Context, _ *AuditEvent) error {
	return nil
}

// Fix: ApprovalRule pointer compatibility
// Remove ApprovalRule redeclaration from stubs.go (should only exist in one place)

// Fix: checkCustomLimits expects map[string]interface{}, update call site in compliance.go
// (No need to add stub here, fix should be in compliance.go)

// Remove redeclared HumanVerification, DelegationLink, and SecondLevelApproval from stubs.go
// These types are already defined in enhanced_authorization.go
// Only keep the noopEnhancedStore methods
