package auth

import (
	"context"
	"time"
)

// RegistryVerifier is a no-op stub for compilation.

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
func (t *StandardComplianceTracker) RecordAuditEvent(ctx context.Context, event *AuditEvent) error { return nil }

// Fix: ApprovalRule pointer compatibility
// Remove ApprovalRule redeclaration from stubs.go (should only exist in one place)

// Fix: checkCustomLimits expects map[string]interface{}, update call site in compliance.go
// (No need to add stub here, fix should be in compliance.go)

// Remove redeclared HumanVerification, DelegationLink, and SecondLevelApproval from stubs.go
// These types are already defined in enhanced_authorization.go
// Only keep the noopEnhancedStore methods
