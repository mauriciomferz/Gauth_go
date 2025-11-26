package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Prometheus metrics for blockchain sync
	syncOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "blockchain_sync_operations_total",
			Help: "Total number of blockchain sync operations",
		},
		[]string{"operation", "status"},
	)

	syncDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "blockchain_sync_duration_seconds",
			Help:    "Duration of blockchain sync operations",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)

	consistencyCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "blockchain_consistency_checks_total",
			Help: "Total number of consistency checks performed",
		},
		[]string{"result"},
	)

	blockchainLag = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "blockchain_sync_lag_blocks",
			Help: "Number of blocks behind the blockchain head",
		},
	)
)

// BlockchainSyncService implements dual-write pattern for PostgreSQL + Blockchain
type BlockchainSyncService struct {
	registry          BlockchainRegistry
	poaStore          PoAStore
	hashingService    HashingService
	ipfsService       IPFSService
	eventListeners    []EventListener
	
	// Sync state
	mu                sync.RWMutex
	lastBlockSynced   int64
	pendingSyncs      map[string]*PendingSync
	failedSyncs       map[string]*FailedSync
	
	// Configuration
	config            *SyncConfig
	
	// Channels for async operations
	syncQueue         chan *SyncJob
	resultQueue       chan *SyncResult
	stopCh            chan struct{}
	wg                sync.WaitGroup
}

// SyncConfig holds configuration for the sync service
type SyncConfig struct {
	Enabled              bool          `json:"enabled"`
	SyncMode             string        `json:"sync_mode"` // "immediate", "batch", "async"
	BatchSize            int           `json:"batch_size"`
	BatchInterval        time.Duration `json:"batch_interval"`
	WorkerCount          int           `json:"worker_count"`
	RetryAttempts        int           `json:"retry_attempts"`
	RetryDelay           time.Duration `json:"retry_delay"`
	ConsistencyCheckInt  time.Duration `json:"consistency_check_interval"`
	MaxPendingSyncs      int           `json:"max_pending_syncs"`
	ConfirmationBlocks   int           `json:"confirmation_blocks"`
}

// PendingSync represents a sync operation waiting for confirmation
type PendingSync struct {
	PoAID           string
	Operation       string // "register", "revoke", "update"
	TxHash          string
	SubmittedAt     time.Time
	Confirmations   int
	RequiredConfirms int
	Retries         int
}

// FailedSync represents a failed sync operation
type FailedSync struct {
	PoAID       string
	Operation   string
	Error       string
	FailedAt    time.Time
	Attempts    int
	LastAttempt time.Time
}

// SyncJob represents a sync job to be processed
type SyncJob struct {
	PoAID     string
	Operation string
	PoA       *EnhancedPoA
	Metadata  map[string]interface{}
}

// SyncResult represents the result of a sync operation
type SyncResult struct {
	PoAID     string
	Operation string
	TxHash    string
	Success   bool
	Error     error
}

// NewBlockchainSyncService creates a new sync service
func NewBlockchainSyncService(
	registry BlockchainRegistry,
	poaStore PoAStore,
	hashingService HashingService,
	ipfsService IPFSService,
	config *SyncConfig,
) *BlockchainSyncService {
	if config == nil {
		config = DefaultSyncConfig()
	}

	service := &BlockchainSyncService{
		registry:        registry,
		poaStore:        poaStore,
		hashingService:  hashingService,
		ipfsService:     ipfsService,
		eventListeners:  []EventListener{},
		lastBlockSynced: 0,
		pendingSyncs:    make(map[string]*PendingSync),
		failedSyncs:     make(map[string]*FailedSync),
		config:          config,
		syncQueue:       make(chan *SyncJob, config.MaxPendingSyncs),
		resultQueue:     make(chan *SyncResult, config.MaxPendingSyncs),
		stopCh:          make(chan struct{}),
	}

	return service
}

// DefaultSyncConfig returns default sync configuration
func DefaultSyncConfig() *SyncConfig {
	return &SyncConfig{
		Enabled:             true,
		SyncMode:            "async",
		BatchSize:           10,
		BatchInterval:       5 * time.Minute,
		WorkerCount:         3,
		RetryAttempts:       3,
		RetryDelay:          30 * time.Second,
		ConsistencyCheckInt: 15 * time.Minute,
		MaxPendingSyncs:     100,
		ConfirmationBlocks:  12,
	}
}

// Start starts the sync service
func (s *BlockchainSyncService) Start(ctx context.Context) error {
	if !s.config.Enabled {
		return fmt.Errorf("sync service is disabled")
	}

	// Start workers
	for i := 0; i < s.config.WorkerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}

	// Start result processor
	s.wg.Add(1)
	go s.resultProcessor(ctx)

	// Start consistency checker
	s.wg.Add(1)
	go s.consistencyChecker(ctx)

	// Start pending confirmation tracker
	s.wg.Add(1)
	go s.confirmationTracker(ctx)

	return nil
}

// Stop stops the sync service
func (s *BlockchainSyncService) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// SyncPoARegistration syncs a PoA registration to blockchain
func (s *BlockchainSyncService) SyncPoARegistration(ctx context.Context, poa *EnhancedPoA) error {
	if !s.config.Enabled {
		return nil // Silently skip if disabled
	}

	startTime := time.Now()
	defer func() {
		syncDuration.WithLabelValues("register").Observe(time.Since(startTime).Seconds())
	}()

	// Create sync job
	job := &SyncJob{
		PoAID:     poa.ID,
		Operation: "register",
		PoA:       poa,
	}

	// Submit job based on sync mode
	switch s.config.SyncMode {
	case "immediate":
		return s.syncImmediate(ctx, job)
	case "async":
		return s.syncAsync(job)
	case "batch":
		return s.syncBatch(job)
	default:
		return fmt.Errorf("unknown sync mode: %s", s.config.SyncMode)
	}
}

// SyncPoARevocation syncs a PoA revocation to blockchain
func (s *BlockchainSyncService) SyncPoARevocation(ctx context.Context, poaID string, revokedBy string, reason string) error {
	if !s.config.Enabled {
		return nil
	}

	startTime := time.Now()
	defer func() {
		syncDuration.WithLabelValues("revoke").Observe(time.Since(startTime).Seconds())
	}()

	job := &SyncJob{
		PoAID:     poaID,
		Operation: "revoke",
		Metadata: map[string]interface{}{
			"revoked_by": revokedBy,
			"reason":     reason,
		},
	}

	switch s.config.SyncMode {
	case "immediate":
		return s.syncImmediate(ctx, job)
	case "async":
		return s.syncAsync(job)
	case "batch":
		return s.syncBatch(job)
	default:
		return fmt.Errorf("unknown sync mode: %s", s.config.SyncMode)
	}
}

// syncImmediate performs immediate synchronous sync
func (s *BlockchainSyncService) syncImmediate(ctx context.Context, job *SyncJob) error {
	result := s.processJob(ctx, job)
	
	if result.Success {
		syncOperationsTotal.WithLabelValues(job.Operation, "success").Inc()
		return nil
	}
	
	syncOperationsTotal.WithLabelValues(job.Operation, "failure").Inc()
	return result.Error
}

// syncAsync performs asynchronous sync
func (s *BlockchainSyncService) syncAsync(job *SyncJob) error {
	select {
	case s.syncQueue <- job:
		return nil
	default:
		syncOperationsTotal.WithLabelValues(job.Operation, "queue_full").Inc()
		return fmt.Errorf("sync queue is full")
	}
}

// syncBatch adds job to batch queue
func (s *BlockchainSyncService) syncBatch(job *SyncJob) error {
	// For batch mode, we'd accumulate jobs and process them in batches
	// Simplified implementation sends to async queue
	return s.syncAsync(job)
}

// worker processes sync jobs from the queue
func (s *BlockchainSyncService) worker(ctx context.Context, workerID int) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case job := <-s.syncQueue:
			result := s.processJob(ctx, job)
			s.resultQueue <- result
		}
	}
}

// processJob processes a single sync job
func (s *BlockchainSyncService) processJob(ctx context.Context, job *SyncJob) *SyncResult {
	result := &SyncResult{
		PoAID:     job.PoAID,
		Operation: job.Operation,
	}

	switch job.Operation {
	case "register":
		txHash, err := s.registerOnBlockchain(ctx, job.PoA)
		result.TxHash = txHash
		result.Success = err == nil
		result.Error = err
		
	case "revoke":
		revokedBy := job.Metadata["revoked_by"].(string)
		reason := job.Metadata["reason"].(string)
		txHash, err := s.registry.RevokePoA(ctx, job.PoAID, revokedBy, reason)
		result.TxHash = txHash
		result.Success = err == nil
		result.Error = err
		
	case "update":
		status := job.Metadata["status"].(string)
		txHash, err := s.registry.UpdatePoAStatus(ctx, job.PoAID, status)
		result.TxHash = txHash
		result.Success = err == nil
		result.Error = err
		
	default:
		result.Success = false
		result.Error = fmt.Errorf("unknown operation: %s", job.Operation)
	}

	return result
}

// registerOnBlockchain handles the full blockchain registration process
func (s *BlockchainSyncService) registerOnBlockchain(ctx context.Context, poa *EnhancedPoA) (string, error) {
	// Step 1: Compute hashes
	scopeHash, err := s.hashingService.HashScope(poa.StructuredScope)
	if err != nil {
		return "", fmt.Errorf("failed to hash scope: %w", err)
	}

	attestationHash, err := s.hashingService.HashAttestations(poa.Attestations)
	if err != nil {
		return "", fmt.Errorf("failed to hash attestations: %w", err)
	}

	metadataHash, err := s.hashingService.HashMetadata(poa)
	if err != nil {
		return "", fmt.Errorf("failed to hash metadata: %w", err)
	}

	// Step 2: Store full metadata on IPFS
	var metadataURI string
	if s.ipfsService != nil {
		cid, err := s.ipfsService.StoreMetadata(ctx, poa)
		if err != nil {
			// Log warning but continue - IPFS is optional
			metadataURI = ""
		} else {
			metadataURI = fmt.Sprintf("ipfs://%s", cid)
		}
	}

	// Step 3: Create blockchain record
	record := &PoARecord{
		ID:              poa.ID,
		IssuerID:        poa.IssuerID,
		GranteeID:       poa.GranteeID,
		ScopeHash:       scopeHash,
		ValidFrom:       poa.ValidFrom,
		ValidUntil:      poa.ValidUntil,
		Status:          poa.Status,
		AttestationHash: attestationHash,
		MetadataHash:    metadataHash,
		MetadataURI:     metadataURI,
	}

	// Step 4: Register on blockchain
	txHash, err := s.registry.RegisterPoA(ctx, record)
	if err != nil {
		return "", fmt.Errorf("blockchain registration failed: %w", err)
	}

	return txHash, nil
}

// resultProcessor processes sync results
func (s *BlockchainSyncService) resultProcessor(ctx context.Context) {
	defer s.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case result := <-s.resultQueue:
			s.handleSyncResult(result)
		}
	}
}

// handleSyncResult handles a sync result
func (s *BlockchainSyncService) handleSyncResult(result *SyncResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if result.Success {
		// Add to pending confirmations
		s.pendingSyncs[result.PoAID] = &PendingSync{
			PoAID:            result.PoAID,
			Operation:        result.Operation,
			TxHash:           result.TxHash,
			SubmittedAt:      time.Now(),
			Confirmations:    0,
			RequiredConfirms: s.config.ConfirmationBlocks,
			Retries:          0,
		}
		
		// Remove from failed if it was there
		delete(s.failedSyncs, result.PoAID)
		
		syncOperationsTotal.WithLabelValues(result.Operation, "success").Inc()
	} else {
		// Add to failed syncs
		failed, exists := s.failedSyncs[result.PoAID]
		if !exists {
			failed = &FailedSync{
				PoAID:     result.PoAID,
				Operation: result.Operation,
				FailedAt:  time.Now(),
				Attempts:  0,
			}
			s.failedSyncs[result.PoAID] = failed
		}
		
		failed.Error = result.Error.Error()
		failed.Attempts++
		failed.LastAttempt = time.Now()
		
		syncOperationsTotal.WithLabelValues(result.Operation, "failure").Inc()
	}
}

// confirmationTracker tracks transaction confirmations
func (s *BlockchainSyncService) confirmationTracker(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.checkConfirmations(ctx)
		}
	}
}

// checkConfirmations checks confirmations for pending transactions
func (s *BlockchainSyncService) checkConfirmations(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for poaID, pending := range s.pendingSyncs {
		status, err := s.registry.GetTransactionStatus(ctx, pending.TxHash)
		if err != nil {
			continue
		}

		pending.Confirmations = status.Confirmations

		// If confirmed, remove from pending
		if pending.Confirmations >= pending.RequiredConfirms {
			delete(s.pendingSyncs, poaID)
		}
	}
}

// consistencyChecker periodically checks database-blockchain consistency
func (s *BlockchainSyncService) consistencyChecker(ctx context.Context) {
	defer s.wg.Done()
	
	ticker := time.NewTicker(s.config.ConsistencyCheckInt)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			report, err := s.CheckConsistency(ctx)
			if err != nil {
				consistencyCheckTotal.WithLabelValues("error").Inc()
				continue
			}
			
			if report.InconsistentPoAs > 0 {
				consistencyCheckTotal.WithLabelValues("inconsistent").Inc()
			} else {
				consistencyCheckTotal.WithLabelValues("consistent").Inc()
			}
		}
	}
}

// CheckConsistency verifies database and blockchain are in sync
func (s *BlockchainSyncService) CheckConsistency(ctx context.Context) (*ConsistencyReport, error) {
	report := &ConsistencyReport{
		CheckedAt:        time.Now(),
		Inconsistencies:  []InconsistencyDetail{},
	}

	// This would be implemented with actual database queries
	// Simplified version for now
	report.TotalPoAs = 0
	report.ConsistentPoAs = 0
	report.InconsistentPoAs = 0
	report.MissingOnBlockchain = 0
	report.MissingInDatabase = 0

	return report, nil
}

// GetSyncStatus returns current synchronization status
func (s *BlockchainSyncService) GetSyncStatus(ctx context.Context) (*SyncStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get current blockchain height
	height, err := s.registry.GetBlockchainHeight(ctx)
	if err != nil {
		height = 0
	}

	syncLag := height - s.lastBlockSynced
	blockchainLag.Set(float64(syncLag))

	status := &SyncStatus{
		LastSyncTime:       time.Now(),
		LastBlockProcessed: s.lastBlockSynced,
		CurrentBlockHeight: height,
		SyncProgress:       0,
		PendingSyncs:       len(s.pendingSyncs),
		FailedSyncs:        len(s.failedSyncs),
		IsHealthy:          len(s.failedSyncs) < 10 && syncLag < 100,
	}

	if height > 0 {
		status.SyncProgress = float64(s.lastBlockSynced) / float64(height)
	}

	return status, nil
}

// AddEventListener adds an event listener
func (s *BlockchainSyncService) AddEventListener(listener EventListener) {
	s.eventListeners = append(s.eventListeners, listener)
}

// Simple hashing service implementation
type SimpleHashingService struct{}

func NewSimpleHashingService() *SimpleHashingService {
	return &SimpleHashingService{}
}

func (h *SimpleHashingService) HashScope(scope interface{}) (string, error) {
	data, err := json.Marshal(scope)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (h *SimpleHashingService) HashAttestations(attestations interface{}) (string, error) {
	data, err := json.Marshal(attestations)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (h *SimpleHashingService) HashMetadata(metadata interface{}) (string, error) {
	data, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func (h *SimpleHashingService) VerifyHash(data interface{}, hash string) (bool, error) {
	computed, err := h.HashMetadata(data)
	if err != nil {
		return false, err
	}
	return computed == hash, nil
}
