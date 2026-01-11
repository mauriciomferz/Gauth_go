package poa

import (
	"context"
)

// PoARepository defines the interface for PoA persistence operations.
type PoARepository interface {
	CreatePoA(ctx context.Context, poa *PoARecord) error
	ListPoAs(ctx context.Context, tenantID string, limit, offset int) ([]PoARecord, int, error)
	GetPoA(ctx context.Context, tenantID, poaID string) (*PoARecord, error)
	RevokePoA(ctx context.Context, tenantID, poaID, revokedBy, reason string) error
	ApprovePoA(ctx context.Context, tenantID, poaID, approvedBy string) error
	RejectPoA(ctx context.Context, tenantID, poaID, rejectedBy, reason string) error
	ValidatePoA(ctx context.Context, tenantID, grantorID, representativeID, action, resource string) (*PoARecord, bool, string)
	GetPoAStats(ctx context.Context, tenantID string) (*PoAStats, error)
	CreateTemplate(ctx context.Context, template *PoATemplate) error
	ListTemplates(ctx context.Context, tenantID *string) ([]PoATemplate, error)

	// AddMultiSignature adds a signature to the PoA and transitions to active if threshold matched.
	AddMultiSignature(
		ctx context.Context,
		tenantID, poaID string,
		signerID string,
		signature map[string]interface{},
		threshold int,
	) (*PoARecord, error)
}
