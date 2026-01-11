package blockchain

import (
	"context"
	"fmt"
	"time"
)

// PoAStore defines the interface for database operations
type PoAStore interface {
	GetPoA(ctx context.Context, poaID string) (*EnhancedPoA, error)
	GetPoAsByStatus(ctx context.Context, status string) ([]*EnhancedPoA, error)
	UpdatePoABlockchainInfo(ctx context.Context, poaID string, txHash string, blockNumber int64) error
	GetUnsyncedPoAs(ctx context.Context, limit int) ([]*EnhancedPoA, error)
}

// EnhancedPoA represents the enhanced Proof of Authorization structure
type EnhancedPoA struct {
	ID               string      `json:"id"`
	IssuerID         string      `json:"issuer_id"`
	GranteeID        string      `json:"grantee_id"`
	SuccessorID      *string     `json:"successor_id,omitempty"`
	StructuredScope  interface{} `json:"structured_scope"`
	Restrictions     interface{} `json:"restrictions"`
	Attestations     interface{} `json:"attestations"`
	VersionNumber    int         `json:"version_number"`
	VersionHistory   interface{} `json:"version_history"`
	Status           string      `json:"status"`
	ValidFrom        time.Time   `json:"valid_from"`
	ValidUntil       time.Time   `json:"valid_until"`
	RevokedAt        *time.Time  `json:"revoked_at,omitempty"`
	RevokedBy        *string     `json:"revoked_by,omitempty"`
	RevocationReason *string     `json:"revocation_reason,omitempty"`
	BlockchainTxHash *string     `json:"blockchain_tx_hash,omitempty"`
	BlockchainBlock  *int64      `json:"blockchain_block,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

// PoARecord represents minimal PoA data stored on blockchain
type PoARecord struct {
	ID              string    `json:"id"`
	IssuerID        string    `json:"issuer_id"`
	GranteeID       string    `json:"grantee_id"`
	ScopeHash       string    `json:"scope_hash"`
	ValidFrom       time.Time `json:"valid_from"`
	ValidUntil      time.Time `json:"valid_until"`
	Status          string    `json:"status"`
	AttestationHash string    `json:"attestation_hash"`
	MetadataHash    string    `json:"metadata_hash"`
	MetadataURI     string    `json:"metadata_uri"`
}

// BlockchainPoARecord represents full PoA data from blockchain
type BlockchainPoARecord struct {
	ID               string    `json:"id"`
	IssuerIDHash     string    `json:"issuer_id_hash"`
	GranteeIDHash    string    `json:"grantee_id_hash"`
	ScopeHash        string    `json:"scope_hash"`
	AttestationHash  string    `json:"attestation_hash"`
	MetadataHash     string    `json:"metadata_hash"`
	MetadataURI      string    `json:"metadata_uri"`
	ValidFrom        time.Time `json:"valid_from"`
	ValidUntil       time.Time `json:"valid_until"`
	Status           string    `json:"status"`
	RegisteredAt     time.Time `json:"registered_at"`
	Revoked          bool      `json:"revoked"`
	RevokedAt        time.Time `json:"revoked_at,omitempty"`
	RevokedByHash    string    `json:"revoked_by_hash,omitempty"`
	RevocationReason string    `json:"revocation_reason,omitempty"`
	TxHash           string    `json:"tx_hash"`
	BlockNumber      int64     `json:"block_number"`
}

// VerificationProof represents cryptographic proof of verification
type VerificationProof struct {
	ProofType        string                 `json:"proof_type"` // "merkle", "signature", "zk-proof"
	ProofData        map[string]interface{} `json:"proof_data"`
	Timestamp        time.Time              `json:"timestamp"`
	BlockchainTxHash string                 `json:"blockchain_tx_hash"`
	VerificationURL  string                 `json:"verification_url"`
}

// AIAgentRegistration represents AI agent registration data
type AIAgentRegistration struct {
	AgentID          string                 `json:"agent_id"`
	OwnerID          string                 `json:"owner_id"`
	AgentName        string                 `json:"agent_name"`
	AgentType        string                 `json:"agent_type"` // "llm", "autonomous", "assistant"
	Capabilities     []string               `json:"capabilities"`
	Specifications   map[string]interface{} `json:"specifications"`
	CertificationDoc string                 `json:"certification_doc"`
	RegisteredAt     time.Time              `json:"registered_at"`
}

// RegistrationCertificate represents an AI agent registration certificate
type RegistrationCertificate struct {
	CertificateID    string                 `json:"certificate_id"`
	AgentID          string                 `json:"agent_id"`
	IssuerID         string                 `json:"issuer_id"` // Commercial register authority
	IssuedAt         time.Time              `json:"issued_at"`
	ExpiresAt        time.Time              `json:"expires_at"`
	CertificateType  string                 `json:"certificate_type"`
	Details          map[string]interface{} `json:"details"`
	BlockchainTxHash string                 `json:"blockchain_tx_hash"`
	CertificateURL   string                 `json:"certificate_url"`
}

// AIAgentPowersSummary represents summary of AI agent powers
type AIAgentPowersSummary struct {
	AgentID           string                 `json:"agent_id"`
	TotalPoAs         int                    `json:"total_poas"`
	ActivePoAs        int                    `json:"active_poas"`
	RevokedPoAs       int                    `json:"revoked_poas"`
	AuthorizedActions []string               `json:"authorized_actions"`
	Restrictions      map[string]interface{} `json:"restrictions"`
	LastUpdated       time.Time              `json:"last_updated"`
}

// PublicVerificationResult represents public verification result
type PublicVerificationResult struct {
	PoAID             string             `json:"poa_id"`
	Verified          bool               `json:"verified"`
	Active            bool               `json:"active"`
	IssuerIDHash      string             `json:"issuer_id_hash"`
	GranteeIDHash     string             `json:"grantee_id_hash"`
	ValidFrom         time.Time          `json:"valid_from"`
	ValidUntil        time.Time          `json:"valid_until"`
	Status            string             `json:"status"`
	VerificationProof *VerificationProof `json:"verification_proof"`
	VerifiedAt        time.Time          `json:"verified_at"`
	BlockchainURL     string             `json:"blockchain_url"`
}

// ConsistencyReport represents blockchain-database consistency check result
type ConsistencyReport struct {
	CheckedAt           time.Time             `json:"checked_at"`
	TotalPoAs           int                   `json:"total_poas"`
	ConsistentPoAs      int                   `json:"consistent_poas"`
	InconsistentPoAs    int                   `json:"inconsistent_poas"`
	MissingOnBlockchain int                   `json:"missing_on_blockchain"`
	MissingInDatabase   int                   `json:"missing_in_database"`
	Inconsistencies     []InconsistencyDetail `json:"inconsistencies"`
}

// InconsistencyDetail represents a specific inconsistency
type InconsistencyDetail struct {
	PoAID string `json:"poa_id"`
	// InconsistencyType is one of: missing_blockchain, missing_db, hash_mismatch, status_mismatch.
	InconsistencyType string      `json:"inconsistency_type"`
	DatabaseValue     interface{} `json:"database_value"`
	BlockchainValue   interface{} `json:"blockchain_value"`
	DetectedAt        time.Time   `json:"detected_at"`
}

// SyncStatus represents current synchronization status
type SyncStatus struct {
	LastSyncTime       time.Time `json:"last_sync_time"`
	LastBlockProcessed int64     `json:"last_block_processed"`
	CurrentBlockHeight int64     `json:"current_block_height"`
	SyncProgress       float64   `json:"sync_progress"`
	PendingSyncs       int       `json:"pending_syncs"`
	FailedSyncs        int       `json:"failed_syncs"`
	IsHealthy          bool      `json:"is_healthy"`
}

// CommercialRegisterService defines the interface for commercial register operations
type CommercialRegisterService interface {
	RegisterAIAgent(ctx context.Context, registration *AIAgentRegistration) (registrationID string, txHash string, err error)
	IssueRegistrationCertificate(ctx context.Context, registrationID string) (*RegistrationCertificate, error)
	LookupAIAgentPowers(ctx context.Context, agentID string) (*AIAgentPowersSummary, error)
	VerifyAIPowers(ctx context.Context, agentID string, action string) (*PublicVerificationResult, error)
	LinkPoAToAgent(ctx context.Context, poaID string, agentID string) (txHash string, err error)
	GetAgentCertificate(ctx context.Context, agentID string) (*RegistrationCertificate, error)
}

// SyncService defines the interface for blockchain synchronization
type SyncService interface {
	// Sync operations
	SyncToBlockchain(ctx context.Context) error
	SyncFromBlockchain(ctx context.Context, fromBlock int64) error
	CheckConsistency(ctx context.Context) (*ConsistencyReport, error)

	// Status and monitoring
	GetSyncStatus(ctx context.Context) (*SyncStatus, error)

	// PoA specific sync
	SyncPoARegistration(ctx context.Context, poa *EnhancedPoA) error
	SyncPoARevocation(ctx context.Context, poaID string, revokedBy string, reason string) error

	// Lifecycle
	Start(ctx context.Context) error
	Stop()
}

// HashingService defines the interface for cryptographic hashing
type HashingService interface {
	HashScope(scope interface{}) (string, error)
	HashAttestations(attestations interface{}) (string, error)
	HashMetadata(metadata interface{}) (string, error)
	VerifyHash(data interface{}, hash string) (bool, error)
}

// IPFSService defines the interface for IPFS operations
type IPFSService interface {
	StoreMetadata(ctx context.Context, metadata interface{}) (cid string, err error)
	RetrieveMetadata(ctx context.Context, cid string) (interface{}, error)
	PinMetadata(ctx context.Context, cid string) error
	UnpinMetadata(ctx context.Context, cid string) error
}

// EventListener defines the interface for blockchain event listening
type EventListener interface {
	OnPoARegistered(ctx context.Context, event *PoARegisteredEvent) error
	OnPoARevoked(ctx context.Context, event *PoARevokedEvent) error
	OnPoAStatusUpdated(ctx context.Context, event *PoAStatusUpdatedEvent) error
	OnAIAgentRegistered(ctx context.Context, event *AIAgentRegisteredEvent) error
}

// Blockchain events
type PoARegisteredEvent struct {
	PoAID       string
	IssuerID    string
	GranteeID   string
	TxHash      string
	BlockNumber int64
	Timestamp   time.Time
}

type PoARevokedEvent struct {
	PoAID       string
	RevokedBy   string
	Reason      string
	TxHash      string
	BlockNumber int64
	Timestamp   time.Time
}

type PoAStatusUpdatedEvent struct {
	PoAID       string
	NewStatus   string
	TxHash      string
	BlockNumber int64
	Timestamp   time.Time
}

type AIAgentRegisteredEvent struct {
	AgentID     string
	OwnerID     string
	AgentType   string
	TxHash      string
	BlockNumber int64
	Timestamp   time.Time
}

// MockBlockchainRegistry provides a mock implementation for testing
type MockBlockchainRegistry struct {
	records map[string]*BlockchainPoARecord
}

func NewMockBlockchainRegistry() *MockBlockchainRegistry {
	return &MockBlockchainRegistry{
		records: make(map[string]*BlockchainPoARecord),
	}
}

func (m *MockBlockchainRegistry) RegisterPoA(ctx context.Context, record *PoARecord) (string, error) {
	txHash := fmt.Sprintf("0x%s", record.ID)
	m.records[record.ID] = &BlockchainPoARecord{
		ID:            record.ID,
		IssuerIDHash:  record.IssuerID,
		GranteeIDHash: record.GranteeID,
		ScopeHash:     record.ScopeHash,
		ValidFrom:     record.ValidFrom,
		ValidUntil:    record.ValidUntil,
		Status:        record.Status,
		RegisteredAt:  time.Now(),
		TxHash:        txHash,
	}
	return txHash, nil
}

func (m *MockBlockchainRegistry) RevokePoA(ctx context.Context, poaID string, revokedBy string, reason string) (string, error) {
	record, exists := m.records[poaID]
	if !exists {
		return "", fmt.Errorf("PoA not found")
	}
	record.Revoked = true
	record.RevokedAt = time.Now()
	record.RevocationReason = reason
	return fmt.Sprintf("0x%s_revoked", poaID), nil
}

func (m *MockBlockchainRegistry) UpdatePoAStatus(ctx context.Context, poaID string, status string) (string, error) {
	record, exists := m.records[poaID]
	if !exists {
		return "", fmt.Errorf("PoA not found")
	}
	record.Status = status
	return fmt.Sprintf("0x%s_updated", poaID), nil
}

func (m *MockBlockchainRegistry) VerifyPoAOnChain(ctx context.Context, poaID string) (*BlockchainPoARecord, error) {
	record, exists := m.records[poaID]
	if !exists {
		return nil, fmt.Errorf("PoA not found")
	}
	return record, nil
}

func (m *MockBlockchainRegistry) GetPublicVerificationURL(poaID string) string {
	return fmt.Sprintf("https://mock-blockchain.com/verify/%s", poaID)
}

func (m *MockBlockchainRegistry) ListPoAsByIssuer(ctx context.Context, issuerID string) ([]*BlockchainPoARecord, error) {
	var results []*BlockchainPoARecord
	for _, record := range m.records {
		if record.IssuerIDHash == issuerID {
			results = append(results, record)
		}
	}
	return results, nil
}

func (m *MockBlockchainRegistry) ListPoAsByGrantee(ctx context.Context, granteeID string) ([]*BlockchainPoARecord, error) {
	var results []*BlockchainPoARecord
	for _, record := range m.records {
		if record.GranteeIDHash == granteeID {
			results = append(results, record)
		}
	}
	return results, nil
}

func (m *MockBlockchainRegistry) RegisterAIAgent(ctx context.Context, registration *AIAgentRegistration) (string, string, error) {
	return registration.AgentID, fmt.Sprintf("0x%s_agent", registration.AgentID), nil
}

func (m *MockBlockchainRegistry) GetBlockchainHeight(ctx context.Context) (int64, error) {
	return 1000000, nil
}

func (m *MockBlockchainRegistry) GetTransactionStatus(ctx context.Context, txHash string) (*TransactionStatus, error) {
	return &TransactionStatus{
		TxHash:        txHash,
		Status:        "confirmed",
		Confirmations: 12,
		BlockNumber:   1000000,
	}, nil
}

func (m *MockBlockchainRegistry) HealthCheck(ctx context.Context) error {
	return nil
}
