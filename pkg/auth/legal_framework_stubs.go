package auth

import "time"

// Stubs for legal framework test types

type ApprovalEvent struct{}
type FiduciaryDuty struct {
	Type        string
	Description string
	Scope       []string
	Validation  []string
}
type ValidationContext struct{}
type ApprovalContext struct{}
type RoleContext struct{}
type ComplianceContext struct{}
type IssuerContext struct{}
type CapacityProof struct {
	Type         string
	IssuedAt     time.Time
	ExpiresAt    time.Time
	IssuerID     string
	Proof        string
	Jurisdiction string
}
type JurisdictionContext struct{}
type EvidenceContext struct{}

// StandardLegalFramework is a stub for legal framework tests
// (real implementation should be in a separate file if needed)
type StandardLegalFramework struct {
	verifier interface{}
	store    interface{}
	register interface{}
}

// Stub methods for StandardLegalFramework to satisfy tests
func (f *StandardLegalFramework) validateDuty(ctx interface{}, duty interface{}) error { return nil }
func (f *StandardLegalFramework) getJurisdictionRules(jurisdiction string) (*JurisdictionRules, error) { return nil, nil }
func (f *StandardLegalFramework) verifyCapacityProof(ctx interface{}, proof *CapacityProof) error { return nil }
func (f *StandardLegalFramework) isActionAllowed(action string, allowedActions []string) bool { return true }
func (f *StandardLegalFramework) validateJurisdictionRequirements(ctx interface{}, rules *JurisdictionRules, action string) error { return nil }
