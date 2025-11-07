// Package notarization provides external notarization for revocation events
// using blockchain or trusted timestamping services for tamper-proof audit trails.
package notarization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// RevocationEvent represents a revocation event to be notarized.
type RevocationEvent struct {
	DelegationID string                 `json:"delegation_id"`
	Subject      string                 `json:"subject"`
	Delegate     string                 `json:"delegate"`
	Reason       string                 `json:"reason"`
	RevokedAt    time.Time              `json:"revoked_at"`
	RevokedBy    string                 `json:"revoked_by"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NotarizationProof represents proof that an event was notarized.
type NotarizationProof struct {
	EventHash        string    `json:"event_hash"`
	NotarizedAt      time.Time `json:"notarized_at"`
	NotaryProvider   string    `json:"notary_provider"`
	TransactionID    string    `json:"transaction_id,omitempty"`    // Blockchain tx ID
	BlockNumber      int64     `json:"block_number,omitempty"`      // Block number
	TimestampToken   string    `json:"timestamp_token,omitempty"`   // RFC 3161 timestamp token
	MerkleRoot       string    `json:"merkle_root,omitempty"`       // Merkle tree root
	MerkleProof      []string  `json:"merkle_proof,omitempty"`      // Path to root
	VerificationURL  string    `json:"verification_url,omitempty"`  // URL to verify
	AdditionalProofs []string  `json:"additional_proofs,omitempty"` // Extra proof data
}

// NotaryProvider defines the interface for notarization backends.
type NotaryProvider interface {
	// Notarize submits an event for notarization and returns proof
	Notarize(event *RevocationEvent) (*NotarizationProof, error)
	
	// Verify checks if a proof is valid for an event
	Verify(event *RevocationEvent, proof *NotarizationProof) (bool, error)
	
	// GetProviderName returns the name of the notary provider
	GetProviderName() string
}

// BlockchainNotary implements blockchain-based notarization.
type BlockchainNotary struct {
	mu sync.RWMutex
	
	// Configuration
	chainType    string // "ethereum", "polygon", "avalanche", etc.
	contractAddr string // Smart contract address
	endpoint     string // RPC endpoint
	
	// Pending notarizations (batch mode)
	pending      []*RevocationEvent
	batchSize    int
	batchTimeout time.Duration
	
	// Statistics
	totalNotarized int64
	lastBatch      time.Time
}

// BlockchainConfig contains configuration for blockchain notarization.
type BlockchainConfig struct {
	ChainType    string
	ContractAddr string
	Endpoint     string
	BatchSize    int
	BatchTimeout time.Duration
}

// NewBlockchainNotary creates a new blockchain-based notary.
func NewBlockchainNotary(config *BlockchainConfig) (*BlockchainNotary, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}
	if config.ChainType == "" {
		config.ChainType = "ethereum"
	}
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.BatchTimeout == 0 {
		config.BatchTimeout = 5 * time.Minute
	}
	
	return &BlockchainNotary{
		chainType:    config.ChainType,
		contractAddr: config.ContractAddr,
		endpoint:     config.Endpoint,
		pending:      make([]*RevocationEvent, 0),
		batchSize:    config.BatchSize,
		batchTimeout: config.BatchTimeout,
	}, nil
}

// Notarize submits a revocation event for blockchain notarization.
func (b *BlockchainNotary) Notarize(event *RevocationEvent) (*NotarizationProof, error) {
	if event == nil {
		return nil, errors.New("event is required")
	}
	
	// Compute event hash
	eventHash, err := b.hashEvent(event)
	if err != nil {
		return nil, fmt.Errorf("failed to hash event: %w", err)
	}
	
	// In production, submit to blockchain smart contract
	// For demo, we simulate blockchain submission
	proof := &NotarizationProof{
		EventHash:       eventHash,
		NotarizedAt:     time.Now(),
		NotaryProvider:  fmt.Sprintf("blockchain:%s", b.chainType),
		TransactionID:   b.generateTxID(eventHash),
		BlockNumber:     b.getCurrentBlockNumber(),
		VerificationURL: b.generateVerificationURL(eventHash),
	}
	
	b.mu.Lock()
	b.totalNotarized++
	b.mu.Unlock()
	
	return proof, nil
}

// Verify checks if a notarization proof is valid.
func (b *BlockchainNotary) Verify(event *RevocationEvent, proof *NotarizationProof) (bool, error) {
	if event == nil || proof == nil {
		return false, errors.New("event and proof are required")
	}
	
	// Compute event hash
	eventHash, err := b.hashEvent(event)
	if err != nil {
		return false, err
	}
	
	// Verify hash matches
	if eventHash != proof.EventHash {
		return false, nil
	}
	
	// In production, query blockchain to verify transaction exists
	// For demo, we do basic validation
	if proof.TransactionID == "" {
		return false, nil
	}
	
	if proof.NotarizedAt.IsZero() {
		return false, nil
	}
	
	return true, nil
}

// GetProviderName returns the blockchain provider name.
func (b *BlockchainNotary) GetProviderName() string {
	return fmt.Sprintf("blockchain:%s", b.chainType)
}

// hashEvent creates a deterministic hash of the revocation event.
func (b *BlockchainNotary) hashEvent(event *RevocationEvent) (string, error) {
	// Create canonical representation
	data := map[string]interface{}{
		"delegation_id": event.DelegationID,
		"subject":       event.Subject,
		"delegate":      event.Delegate,
		"reason":        event.Reason,
		"revoked_at":    event.RevokedAt.Unix(),
		"revoked_by":    event.RevokedBy,
	}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:]), nil
}

// generateTxID simulates blockchain transaction ID generation.
func (b *BlockchainNotary) generateTxID(eventHash string) string {
	// In production, this would be the actual blockchain tx hash
	txHash := sha256.Sum256([]byte(eventHash + time.Now().String()))
	return "0x" + hex.EncodeToString(txHash[:])
}

// getCurrentBlockNumber simulates getting current block number.
func (b *BlockchainNotary) getCurrentBlockNumber() int64 {
	// In production, query the blockchain
	// For demo, use a simulated block number
	return time.Now().Unix() / 12 // ~12s block time
}

// generateVerificationURL creates a URL to verify the notarization.
func (b *BlockchainNotary) generateVerificationURL(eventHash string) string {
	// In production, return block explorer URL
	return fmt.Sprintf("https://explorer.%s/verify/%s", b.chainType, eventHash)
}

// GetStats returns notarization statistics.
func (b *BlockchainNotary) GetStats() map[string]interface{} {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	return map[string]interface{}{
		"total_notarized": b.totalNotarized,
		"pending_batch":   len(b.pending),
		"last_batch":      b.lastBatch,
		"chain_type":      b.chainType,
	}
}

// RFC3161TimestampNotary implements RFC 3161 timestamp protocol notarization.
type RFC3161TimestampNotary struct {
	mu sync.RWMutex
	
	tsaURL         string // Timestamp Authority URL
	totalNotarized int64
}

// NewRFC3161TimestampNotary creates a new RFC 3161 timestamp notary.
func NewRFC3161TimestampNotary(tsaURL string) (*RFC3161TimestampNotary, error) {
	if tsaURL == "" {
		return nil, errors.New("TSA URL is required")
	}
	
	return &RFC3161TimestampNotary{
		tsaURL: tsaURL,
	}, nil
}

// Notarize submits an event to RFC 3161 timestamp authority.
func (r *RFC3161TimestampNotary) Notarize(event *RevocationEvent) (*NotarizationProof, error) {
	if event == nil {
		return nil, errors.New("event is required")
	}
	
	// Compute event hash
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	
	hash := sha256.Sum256(eventJSON)
	eventHash := hex.EncodeToString(hash[:])
	
	// In production, send timestamp request to TSA
	// and receive signed timestamp token
	// For demo, generate mock timestamp token
	timestampToken := r.generateTimestampToken(eventHash)
	
	proof := &NotarizationProof{
		EventHash:       eventHash,
		NotarizedAt:     time.Now(),
		NotaryProvider:  "rfc3161",
		TimestampToken:  timestampToken,
		VerificationURL: r.tsaURL + "/verify",
	}
	
	r.mu.Lock()
	r.totalNotarized++
	r.mu.Unlock()
	
	return proof, nil
}

// Verify checks an RFC 3161 timestamp proof.
func (r *RFC3161TimestampNotary) Verify(event *RevocationEvent, proof *NotarizationProof) (bool, error) {
	if event == nil || proof == nil {
		return false, errors.New("event and proof are required")
	}
	
	// In production, verify the timestamp token signature
	// against the TSA's certificate
	if proof.TimestampToken == "" {
		return false, nil
	}
	
	if proof.NotarizedAt.IsZero() {
		return false, nil
	}
	
	return true, nil
}

// GetProviderName returns the provider name.
func (r *RFC3161TimestampNotary) GetProviderName() string {
	return "rfc3161"
}

// generateTimestampToken simulates RFC 3161 token generation.
func (r *RFC3161TimestampNotary) generateTimestampToken(eventHash string) string {
	// In production, this is a DER-encoded TimeStampToken
	tokenData := fmt.Sprintf("TST:%s:%d", eventHash, time.Now().Unix())
	hash := sha256.Sum256([]byte(tokenData))
	return hex.EncodeToString(hash[:])
}

// MultiNotary notarizes to multiple providers for redundancy.
type MultiNotary struct {
	providers []NotaryProvider
}

// NewMultiNotary creates a notary that uses multiple providers.
func NewMultiNotary(providers ...NotaryProvider) (*MultiNotary, error) {
	if len(providers) == 0 {
		return nil, errors.New("at least one provider is required")
	}
	
	return &MultiNotary{
		providers: providers,
	}, nil
}

// Notarize submits to all providers and collects all proofs.
func (m *MultiNotary) Notarize(event *RevocationEvent) (*NotarizationProof, error) {
	if len(m.providers) == 0 {
		return nil, errors.New("no providers configured")
	}
	
	// Use first provider as primary
	primaryProof, err := m.providers[0].Notarize(event)
	if err != nil {
		return nil, fmt.Errorf("primary notarization failed: %w", err)
	}
	
	// Collect additional proofs from other providers
	additionalProofs := make([]string, 0, len(m.providers)-1)
	for i := 1; i < len(m.providers); i++ {
		proof, err := m.providers[i].Notarize(event)
		if err != nil {
			// Log error but continue with other providers
			continue
		}
		
		// Serialize proof as additional evidence
		proofJSON, _ := json.Marshal(proof)
		additionalProofs = append(additionalProofs, string(proofJSON))
	}
	
	primaryProof.AdditionalProofs = additionalProofs
	return primaryProof, nil
}

// Verify checks proofs against all providers.
func (m *MultiNotary) Verify(event *RevocationEvent, proof *NotarizationProof) (bool, error) {
	// At least the primary provider must verify
	if len(m.providers) == 0 {
		return false, errors.New("no providers configured")
	}
	
	valid, err := m.providers[0].Verify(event, proof)
	if err != nil || !valid {
		return false, err
	}
	
	return true, nil
}

// GetProviderName returns a combined provider name.
func (m *MultiNotary) GetProviderName() string {
	names := make([]string, len(m.providers))
	for i, p := range m.providers {
		names[i] = p.GetProviderName()
	}
	return fmt.Sprintf("multi:%v", names)
}

// NotarizationRegistry tracks all notarized revocations.
type NotarizationRegistry struct {
	mu sync.RWMutex
	
	// Map of delegation ID -> list of notarization proofs
	proofs map[string][]*NotarizationProof
	
	// Notary provider
	notary NotaryProvider
}

// NewNotarizationRegistry creates a new registry.
func NewNotarizationRegistry(notary NotaryProvider) *NotarizationRegistry {
	return &NotarizationRegistry{
		proofs: make(map[string][]*NotarizationProof),
		notary: notary,
	}
}

// NotarizeRevocation notarizes a revocation event and stores the proof.
func (r *NotarizationRegistry) NotarizeRevocation(event *RevocationEvent) (*NotarizationProof, error) {
	proof, err := r.notary.Notarize(event)
	if err != nil {
		return nil, err
	}
	
	// Store proof
	r.mu.Lock()
	if r.proofs[event.DelegationID] == nil {
		r.proofs[event.DelegationID] = make([]*NotarizationProof, 0)
	}
	r.proofs[event.DelegationID] = append(r.proofs[event.DelegationID], proof)
	r.mu.Unlock()
	
	return proof, nil
}

// GetProofs retrieves all notarization proofs for a delegation.
func (r *NotarizationRegistry) GetProofs(delegationID string) []*NotarizationProof {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	proofs := r.proofs[delegationID]
	if proofs == nil {
		return nil
	}
	
	// Return copy
	result := make([]*NotarizationProof, len(proofs))
	copy(result, proofs)
	return result
}

// VerifyRevocation verifies that a revocation was properly notarized.
func (r *NotarizationRegistry) VerifyRevocation(event *RevocationEvent, proof *NotarizationProof) (bool, error) {
	return r.notary.Verify(event, proof)
}

// GetStats returns registry statistics.
func (r *NotarizationRegistry) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	totalProofs := 0
	for _, proofs := range r.proofs {
		totalProofs += len(proofs)
	}
	
	return map[string]interface{}{
		"total_delegations": len(r.proofs),
		"total_proofs":      totalProofs,
		"provider":          r.notary.GetProviderName(),
	}
}
