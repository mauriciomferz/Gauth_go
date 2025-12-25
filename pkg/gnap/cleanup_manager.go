package gnap

import (
	"context"
	"log"
	"sync"
	"time"
)

// CleanupManager orchestrates periodic cleanup of GNAP stores.
type CleanupManager struct {
	grantStore *MemoryGrantStore
	tokenStore *MemoryTokenStore
	interval   time.Duration
	stopCh     chan struct{}
	wg         sync.WaitGroup
	mu         sync.Mutex
	running    bool

	// Metrics
	lastCleanup        time.Time
	totalGrantsCleaned int64
	totalTokensCleaned int64
}

// CleanupStats contains cleanup operation statistics.
type CleanupStats struct {
	LastCleanup        time.Time
	TotalGrantsCleaned int64
	TotalTokensCleaned int64
	Running            bool
}

// NewCleanupManager creates a new cleanup manager.
// Recommended interval: 5-15 minutes.
func NewCleanupManager(grantStore *MemoryGrantStore, tokenStore *MemoryTokenStore, interval time.Duration) *CleanupManager {
	return &CleanupManager{
		grantStore: grantStore,
		tokenStore: tokenStore,
		interval:   interval,
		stopCh:     make(chan struct{}),
	}
}

// Start begins periodic cleanup operations.
func (m *CleanupManager) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil // Already running
	}
	m.running = true
	m.mu.Unlock()

	m.wg.Add(1)
	go m.run(ctx)

	log.Printf("[gnap-cleanup] Started with interval: %v", m.interval)
	return nil
}

// Stop gracefully stops the cleanup manager.
func (m *CleanupManager) Stop() {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}
	m.running = false
	m.mu.Unlock()

	close(m.stopCh)
	m.wg.Wait()

	log.Printf("[gnap-cleanup] Stopped. Total cleaned: %d grants, %d tokens",
		m.totalGrantsCleaned, m.totalTokensCleaned)
}

// run executes the cleanup loop.
func (m *CleanupManager) run(ctx context.Context) {
	defer m.wg.Done()
	defer func() {
		m.mu.Lock()
		m.running = false
		m.mu.Unlock()
	}()

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Run immediately on start
	m.doCleanup()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.doCleanup()
		}
	}
}

// doCleanup performs the actual cleanup operation.
func (m *CleanupManager) doCleanup() {
	start := time.Now()

	grantsRemoved := 0
	tokensRemoved := 0

	if m.grantStore != nil {
		grantsRemoved = m.grantStore.Cleanup()
	}

	if m.tokenStore != nil {
		tokensRemoved = m.tokenStore.Cleanup()
	}

	m.mu.Lock()
	m.lastCleanup = time.Now()
	m.totalGrantsCleaned += int64(grantsRemoved)
	m.totalTokensCleaned += int64(tokensRemoved)
	m.mu.Unlock()

	duration := time.Since(start)

	if grantsRemoved > 0 || tokensRemoved > 0 {
		log.Printf("[gnap-cleanup] Removed %d grants, %d tokens in %v",
			grantsRemoved, tokensRemoved, duration)
	}
}

// Stats returns current cleanup statistics.
func (m *CleanupManager) Stats() CleanupStats {
	m.mu.Lock()
	defer m.mu.Unlock()

	return CleanupStats{
		LastCleanup:        m.lastCleanup,
		TotalGrantsCleaned: m.totalGrantsCleaned,
		TotalTokensCleaned: m.totalTokensCleaned,
		Running:            m.running,
	}
}

// RunOnce performs a single cleanup operation immediately.
// Useful for testing or manual triggering.
func (m *CleanupManager) RunOnce() (grantsRemoved, tokensRemoved int) {
	if m.grantStore != nil {
		grantsRemoved = m.grantStore.Cleanup()
	}

	if m.tokenStore != nil {
		tokensRemoved = m.tokenStore.Cleanup()
	}

	m.mu.Lock()
	m.lastCleanup = time.Now()
	m.totalGrantsCleaned += int64(grantsRemoved)
	m.totalTokensCleaned += int64(tokensRemoved)
	m.mu.Unlock()

	return grantsRemoved, tokensRemoved
}
