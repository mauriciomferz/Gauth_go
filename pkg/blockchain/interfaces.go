package blockchain

import (
	"context"
)

// BlockchainRegistry defines the interface for blockchain-based PoA registration
type BlockchainRegistry interface {
	RegisterPoA(ctx context.Context, poa *PoARecord) (transactionHash string, err error)
	UpdatePoAStatus(ctx context.Context, poaID string, status string) (transactionHash string, err error)
	RevokePoA(ctx context.Context, poaID string, revokedBy string, reason string) (transactionHash string, err error)
	VerifyPoAOnChain(ctx context.Context, poaID string) (*BlockchainPoARecord, error)
	GetPublicVerificationURL(poaID string) string
	GetTransactionStatus(ctx context.Context, txHash string) (*TransactionStatus, error)
	ListPoAsByIssuer(ctx context.Context, issuerID string) ([]*BlockchainPoARecord, error)
	ListPoAsByGrantee(ctx context.Context, granteeID string) ([]*BlockchainPoARecord, error)
	GetBlockchainHeight(ctx context.Context) (int64, error)
	HealthCheck(ctx context.Context) error
}