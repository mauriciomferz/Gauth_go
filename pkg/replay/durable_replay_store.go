package replay

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// EvictionPolicy defines pluggable eviction strategies for replay store cleanup (sec6.item1 P1).
type EvictionPolicy interface {
	// ShouldEvict returns true if the entry should be evicted.
	ShouldEvict(jti string, ts time.Time, now time.Time, accessCount int) bool
	// Name returns the policy name for metrics/logging.
	Name() string
}

// TTLEvictionPolicy evicts entries older than TTL (time-based).
type TTLEvictionPolicy struct {
	TTL time.Duration
}

const (
	evictionPolicyNameTTL = "ttl"
)

func (p *TTLEvictionPolicy) ShouldEvict(jti string, ts time.Time, now time.Time, accessCount int) bool {
	return now.Sub(ts) > p.TTL
}

func (p *TTLEvictionPolicy) Name() string {
	return evictionPolicyNameTTL
}

// LRUEvictionPolicy evicts least recently used entries when size exceeds max.
// Requires accessTimes map to track last access.
type LRUEvictionPolicy struct {
	MaxSize     int
	currentSize int
}

func (p *LRUEvictionPolicy) ShouldEvict(jti string, ts time.Time, now time.Time, accessCount int) bool {
	// LRU eviction triggered externally after sorting by accessTime
	// This method is called during scan; actual eviction decision in applyEviction()
	return p.currentSize > p.MaxSize
}

func (p *LRUEvictionPolicy) Name() string {
	return "lru"
}

// SizeBasedEvictionPolicy evicts oldest entries when size exceeds max.
type SizeBasedEvictionPolicy struct {
	MaxSize int
}

func (p *SizeBasedEvictionPolicy) ShouldEvict(jti string, ts time.Time, now time.Time, accessCount int) bool {
	// Size eviction triggered by threshold check (see applyEviction)
	return false // Handled externally
}

func (p *SizeBasedEvictionPolicy) Name() string {
	return "size"
}

// CompositeEvictionPolicy combines multiple policies (ANY match triggers eviction).
type CompositeEvictionPolicy struct {
	Policies []EvictionPolicy
}

func (p *CompositeEvictionPolicy) ShouldEvict(jti string, ts time.Time, now time.Time, accessCount int) bool {
	for _, policy := range p.Policies {
		if policy.ShouldEvict(jti, ts, now, accessCount) {
			return true
		}
	}
	return false
}

func (p *CompositeEvictionPolicy) Name() string {
	names := make([]string, len(p.Policies))
	for i, policy := range p.Policies {
		names[i] = policy.Name()
	}
	return "composite(" + strings.Join(names, ",") + ")"
}

// ParseEvictionPolicy parses env var string into EvictionPolicy.
// Supported: "ttl", "lru", "size", "ttl+size", "none" (default: ttl).
func ParseEvictionPolicy(policy string, ttl time.Duration, maxSize int) EvictionPolicy {
	policy = strings.ToLower(strings.TrimSpace(policy))

	if policy == "none" || policy == "" {
		return &TTLEvictionPolicy{TTL: ttl} // Default to TTL
	}

	if policy == evictionPolicyNameTTL {
		return &TTLEvictionPolicy{TTL: ttl}
	}

	if policy == "lru" {
		if maxSize <= 0 {
			maxSize = 10000 // Default max size for LRU
		}
		return &LRUEvictionPolicy{MaxSize: maxSize}
	}

	if policy == "size" {
		if maxSize <= 0 {
			maxSize = 10000
		}
		return &SizeBasedEvictionPolicy{MaxSize: maxSize}
	}

	// Composite: "ttl+size", "ttl+lru"
	if strings.Contains(policy, "+") {
		parts := strings.Split(policy, "+")
		policies := make([]EvictionPolicy, 0, len(parts))
		for _, part := range parts {
			policies = append(policies, ParseEvictionPolicy(part, ttl, maxSize))
		}
		return &CompositeEvictionPolicy{Policies: policies}
	}

	// Unknown policy, default to TTL
	return &TTLEvictionPolicy{TTL: ttl}
}

// EvictionStats tracks eviction activity for observability.
type EvictionStats struct {
	TotalEvictions uint64
	TTLEvictions   uint64
	LRUEvictions   uint64
	SizeEvictions  uint64
	LastEviction   time.Time
}

// DurableReplayStore wraps WALStore with automatic snapshot scheduling and recovery.
// This completes the replay persistence recovery gap (sec6.item3) by adding:
// - Automatic periodic snapshots (configurable interval)
// - WAL compaction after snapshots
// - Graceful shutdown with final snapshot
// - Concurrent-safe access with metrics integration
// Enhanced for sec6.item1 P1 with:
// - Pluggable eviction policies (TTL, LRU, size-based, composite)
// - Access time tracking for LRU
// - Eviction statistics and metrics
type DurableReplayStore struct {
	mu               sync.RWMutex
	entries          map[string]time.Time // JTI -> creation timestamp
	accessTimes      map[string]time.Time // JTI -> last access timestamp (for LRU)
	ttl              time.Duration
	wal              *WALStore
	snapshotInterval time.Duration
	stopCh           chan struct{}
	stoppedCh        chan struct{}
	metrics          DurableReplayMetrics
	evictionPolicy   EvictionPolicy
	evictionStats    EvictionStats
}

// DurableReplayMetrics defines metrics collection interface for replay persistence.
type DurableReplayMetrics interface {
	IncReplayStoreErrors()
	ObserveReplayStoreLatency(d time.Duration)
	ObserveReplayWALSnapshotDuration(d time.Duration)
	ObserveReplayWALFlushLatency(d time.Duration)
	SetReplayWALPending(n int)
	// Eviction metrics (sec6.item1 P1)
	IncReplayEvictions(reason string)
	SetReplayStoreSize(n int)
	IncReplayCacheHit()
	IncReplayCacheMiss()
}

// NoopReplayMetrics provides a no-op metrics implementation.
type NoopReplayMetrics struct{}

func (n NoopReplayMetrics) IncReplayStoreErrors()                            {}
func (n NoopReplayMetrics) ObserveReplayStoreLatency(d time.Duration)        {}
func (n NoopReplayMetrics) ObserveReplayWALSnapshotDuration(d time.Duration) {}
func (n NoopReplayMetrics) ObserveReplayWALFlushLatency(d time.Duration)     {}
func (n NoopReplayMetrics) SetReplayWALPending(count int)                    {}
func (n NoopReplayMetrics) IncReplayEvictions(reason string)                 {}
func (n NoopReplayMetrics) SetReplayStoreSize(size int)                      {}
func (n NoopReplayMetrics) IncReplayCacheHit()                               {}
func (n NoopReplayMetrics) IncReplayCacheMiss()                              {}

// DurableReplayStoreConfig configures DurableReplayStore creation.
type DurableReplayStoreConfig struct {
	WALPath          string               // Path to WAL file
	TTL              time.Duration        // TTL for replay entries
	SnapshotInterval time.Duration        // How often to create snapshots (default: 5m)
	Metrics          DurableReplayMetrics // Optional metrics (uses NoopReplayMetrics if nil)
	EvictionPolicy   EvictionPolicy       // Optional eviction policy (default: TTL-based)
	MaxSize          int                  // Max entries for size-based policies (default: 10000)
}

// NewDurableReplayStore creates a durable replay store with automatic snapshots.
func NewDurableReplayStore(config DurableReplayStoreConfig) (*DurableReplayStore, error) {
	if config.TTL <= 0 {
		config.TTL = 15 * time.Minute // Default TTL
	}
	if config.SnapshotInterval <= 0 {
		config.SnapshotInterval = 5 * time.Minute // Default snapshot interval
	}
	if config.Metrics == nil {
		config.Metrics = NoopReplayMetrics{}
	}
	if config.EvictionPolicy == nil {
		// Default to TTL-based eviction
		config.EvictionPolicy = &TTLEvictionPolicy{TTL: config.TTL}
	}
	if config.MaxSize <= 0 {
		config.MaxSize = 10000 // Default max size
	}

	wal, err := NewWALStore(config.WALPath)
	if err != nil {
		config.Metrics.IncReplayStoreErrors()
		return nil, fmt.Errorf("create WAL store: %w", err)
	}

	store := &DurableReplayStore{
		entries:          make(map[string]time.Time),
		accessTimes:      make(map[string]time.Time),
		ttl:              config.TTL,
		wal:              wal,
		snapshotInterval: config.SnapshotInterval,
		stopCh:           make(chan struct{}),
		stoppedCh:        make(chan struct{}),
		metrics:          config.Metrics,
		evictionPolicy:   config.EvictionPolicy,
	}

	// Recover from existing WAL + snapshot
	if err := store.recover(); err != nil {
		config.Metrics.IncReplayStoreErrors()
		// Non-fatal: continue with empty store
	}

	// Start snapshot scheduler
	go store.snapshotScheduler()

	return store, nil
}

// recover replays WAL and loads snapshot to rebuild in-memory state.
func (d *DurableReplayStore) recover() error {
	start := time.Now()

	// 1. Load snapshot if it exists
	snapshotPath := d.wal.Path() + ".snapshot"
	if err := d.loadSnapshot(snapshotPath); err != nil {
		// Snapshot may not exist (first run), non-fatal
		if !isNotExist(err) {
			return fmt.Errorf("load snapshot: %w", err)
		}
	}

	// 2. Replay WAL on top of snapshot
	applied, skipped, err := d.wal.RecoverWithStats(func(rec WALRecord) error {
		if rec.Op == "Record" {
			ts := time.Unix(rec.TS, 0)
			// Clamp future timestamps to now
			if ts.After(time.Now()) {
				ts = time.Now()
			}
			d.entries[string(rec.Key)] = ts
		}
		return nil
	})

	d.metrics.ObserveReplayStoreLatency(time.Since(start))

	if err != nil {
		return fmt.Errorf("WAL recovery: %w (applied: %d, skipped: %d)", err, applied, skipped)
	}

	// Update metrics
	for i := 0; i < skipped; i++ {
		d.metrics.IncReplayStoreErrors()
	}

	return nil
}

// loadSnapshot loads snapshot file into memory.
func (d *DurableReplayStore) loadSnapshot(path string) error {
	f, err := os.Open(path)
	if isNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()

	type snapEntry struct {
		Key string `json:"key"`
		TS  int64  `json:"ts"`
	}

	var entries []snapEntry
	if err := json.NewDecoder(f).Decode(&entries); err != nil {
		if err == io.EOF {
			return nil
		}
		return fmt.Errorf("decode snapshot: %w", err)
	}

	for _, entry := range entries {
		ts := time.Unix(entry.TS, 0)
		d.entries[entry.Key] = ts
		d.accessTimes[entry.Key] = ts // Initialize access time to preserved timestamp
	}
	return nil
}

// isNotExist checks if error is "file not found".
func isNotExist(err error) bool {
	return os.IsNotExist(err)
}

// Snapshot creates a point-in-time snapshot and compacts WAL.
func (d *DurableReplayStore) Snapshot() error {
	d.mu.RLock()
	// Build active entries (exclude expired)
	now := time.Now()
	active := make(map[string]time.Time, len(d.entries))
	for k, ts := range d.entries {
		if now.Sub(ts) <= d.ttl {
			active[k] = ts
		}
	}
	d.mu.RUnlock()

	// 1. Write snapshot file (outside lock)
	snapStart := time.Now()
	if err := d.wal.Snapshot(active); err != nil {
		d.metrics.IncReplayStoreErrors()
		return fmt.Errorf("create snapshot: %w", err)
	}
	d.metrics.ObserveReplayWALSnapshotDuration(time.Since(snapStart))

	// 2. Clear WAL (truncate) to start fresh delta log
	// Note: We intentionally do NOT re-append entries to WAL here.
	// We rely on loadSnapshot + new WAL updates for recovery.
	if err := d.wal.Rotate(); err != nil {
		d.metrics.IncReplayStoreErrors()
		return fmt.Errorf("rotate WAL: %w", err)
	}
	d.metrics.SetReplayWALPending(0)

	return nil
}

// Seen checks if JTI has been seen before (thread-safe).
// Enhanced for sec6.item1 P1 with cache hit/miss metrics and access time tracking.
func (d *DurableReplayStore) Seen(jti string) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()

	// Apply eviction policy (lazy cleanup during read)
	d.applyEviction(now)

	ts, exists := d.entries[jti]
	if !exists {
		d.metrics.IncReplayCacheMiss()
		d.metrics.SetReplayStoreSize(len(d.entries))
		return false, nil
	}

	// Update access time for LRU tracking
	d.accessTimes[jti] = now

	// Check if eviction policy says this should be evicted
	if d.evictionPolicy.ShouldEvict(jti, ts, now, 0) {
		delete(d.entries, jti)
		delete(d.accessTimes, jti)
		d.metrics.IncReplayCacheMiss()
		d.metrics.SetReplayStoreSize(len(d.entries))
		return false, nil
	}

	d.metrics.IncReplayCacheHit()
	d.metrics.SetReplayStoreSize(len(d.entries))
	return true, nil
}

// applyEviction applies the configured eviction policy (must be called with lock held).
func (d *DurableReplayStore) applyEviction(now time.Time) {
	evicted := 0

	// For TTL-based eviction
	if _, ok := d.evictionPolicy.(*TTLEvictionPolicy); ok {
		for jti, ts := range d.entries {
			if d.evictionPolicy.ShouldEvict(jti, ts, now, 0) {
				delete(d.entries, jti)
				delete(d.accessTimes, jti)
				evicted++
			}
		}
		if evicted > 0 {
			d.evictionStats.TotalEvictions += uint64(evicted)
			d.evictionStats.TTLEvictions += uint64(evicted)
			d.evictionStats.LastEviction = now
			d.metrics.IncReplayEvictions("ttl")
		}
		return
	}

	// For size-based eviction (oldest first)
	if sizePolicy, ok := d.evictionPolicy.(*SizeBasedEvictionPolicy); ok {
		if len(d.entries) <= sizePolicy.MaxSize {
			return
		}

		// Sort by timestamp, evict oldest
		type entry struct {
			jti string
			ts  time.Time
		}
		entries := make([]entry, 0, len(d.entries))
		for jti, ts := range d.entries {
			entries = append(entries, entry{jti, ts})
		}

		// Simple bubble sort (OK for small eviction batches)
		for i := 0; i < len(entries)-1; i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[i].ts.After(entries[j].ts) {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}

		// Evict oldest until under max size
		toEvict := len(d.entries) - sizePolicy.MaxSize
		for i := 0; i < toEvict && i < len(entries); i++ {
			delete(d.entries, entries[i].jti)
			delete(d.accessTimes, entries[i].jti)
			evicted++
		}

		if evicted > 0 {
			d.evictionStats.TotalEvictions += uint64(evicted)
			d.evictionStats.SizeEvictions += uint64(evicted)
			d.evictionStats.LastEviction = now
			d.metrics.IncReplayEvictions("size")
		}
		return
	}

	// For LRU eviction
	if lruPolicy, ok := d.evictionPolicy.(*LRUEvictionPolicy); ok {
		lruPolicy.currentSize = len(d.entries)
		if len(d.entries) <= lruPolicy.MaxSize {
			return
		}

		// Sort by access time, evict least recently used
		type entry struct {
			jti        string
			accessTime time.Time
		}
		entries := make([]entry, 0, len(d.accessTimes))
		for jti, accessTime := range d.accessTimes {
			entries = append(entries, entry{jti, accessTime})
		}

		// Sort by access time (oldest first)
		for i := 0; i < len(entries)-1; i++ {
			for j := i + 1; j < len(entries); j++ {
				if entries[i].accessTime.After(entries[j].accessTime) {
					entries[i], entries[j] = entries[j], entries[i]
				}
			}
		}

		// Evict LRU entries until under max size
		toEvict := len(d.entries) - lruPolicy.MaxSize
		for i := 0; i < toEvict && i < len(entries); i++ {
			delete(d.entries, entries[i].jti)
			delete(d.accessTimes, entries[i].jti)
			evicted++
		}

		if evicted > 0 {
			d.evictionStats.TotalEvictions += uint64(evicted)
			d.evictionStats.LRUEvictions += uint64(evicted)
			d.evictionStats.LastEviction = now
			d.metrics.IncReplayEvictions("lru")
		}
		return
	}

	// For composite policy, check all sub-policies
	if composite, ok := d.evictionPolicy.(*CompositeEvictionPolicy); ok {
		for jti, ts := range d.entries {
			if composite.ShouldEvict(jti, ts, now, 0) {
				delete(d.entries, jti)
				delete(d.accessTimes, jti)
				evicted++
			}
		}
		if evicted > 0 {
			d.evictionStats.TotalEvictions += uint64(evicted)
			d.evictionStats.LastEviction = now
			d.metrics.IncReplayEvictions("composite")
		}
	}
}

// Record stores JTI with timestamp (thread-safe, durable via WAL).
// Enhanced for sec6.item1 P1 with access time tracking for LRU.
func (d *DurableReplayStore) Record(jti string, at time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.entries[jti] = at
	d.accessTimes[jti] = at // Initialize access time

	// Append to WAL for durability
	start := time.Now()
	err := d.wal.AppendRecord(WALRecord{
		Op:    "Record",
		Key:   []byte(jti),
		Value: nil,
		TS:    at.Unix(),
	})

	if err != nil {
		d.metrics.IncReplayStoreErrors()
		return fmt.Errorf("WAL append: %w", err)
	}

	d.metrics.ObserveReplayStoreLatency(time.Since(start))
	d.metrics.SetReplayWALPending(len(d.entries))
	d.metrics.SetReplayStoreSize(len(d.entries))

	return nil
}

// snapshotScheduler runs periodic snapshots in background.
func (d *DurableReplayStore) snapshotScheduler() {
	ticker := time.NewTicker(d.snapshotInterval)
	defer ticker.Stop()
	defer close(d.stoppedCh)

	for {
		select {
		case <-ticker.C:
			// Perform snapshot
			if err := d.Snapshot(); err != nil {
				// Log error (in production, use structured logging)
				// For now, metrics track errors
				d.metrics.IncReplayStoreErrors()
			}
		case <-d.stopCh:
			// Final snapshot before shutdown
			_ = d.Snapshot()
			return
		}
	}
}

// Close gracefully shuts down the store with a final snapshot.
func (d *DurableReplayStore) Close() error {
	// Signal shutdown
	close(d.stopCh)

	// Wait for scheduler to finish
	<-d.stoppedCh

	// Close WAL
	return d.wal.Close()
}

// Size returns current number of active (non-expired) entries.
func (d *DurableReplayStore) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now()
	count := 0
	for _, ts := range d.entries {
		if now.Sub(ts) <= d.ttl {
			count++
		}
	}
	return count
}

// Stats returns store statistics for monitoring.
type DurableReplayStoreStats struct {
	TotalEntries     int           `json:"total_entries"`
	ActiveEntries    int           `json:"active_entries"`
	WALPath          string        `json:"wal_path"`
	SnapshotInterval time.Duration `json:"snapshot_interval"`
	TTL              time.Duration `json:"ttl"`
}

// Stats returns current store statistics.
func (d *DurableReplayStore) Stats() DurableReplayStoreStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	now := time.Now()
	active := 0
	for _, ts := range d.entries {
		if now.Sub(ts) <= d.ttl {
			active++
		}
	}

	return DurableReplayStoreStats{
		TotalEntries:     len(d.entries),
		ActiveEntries:    active,
		WALPath:          d.wal.Path(),
		SnapshotInterval: d.snapshotInterval,
		TTL:              d.ttl,
	}
}

// WithContext wraps DurableReplayStore to satisfy AAP001 ReplayStore interface.
type DurableReplayStoreAdapter struct {
	store *DurableReplayStore
}

// NewDurableReplayStoreAdapter creates an adapter for AAP001 integration.
func NewDurableReplayStoreAdapter(store *DurableReplayStore) *DurableReplayStoreAdapter {
	return &DurableReplayStoreAdapter{store: store}
}

// Seen implements AAP001 ReplayStore.Seen.
func (a *DurableReplayStoreAdapter) Seen(jti string) (bool, error) {
	return a.store.Seen(jti)
}

// Record implements AAP001 ReplayStore.Record.
func (a *DurableReplayStoreAdapter) Record(jti string, at time.Time) error {
	return a.store.Record(jti, at)
}

// CheckAndStore implements gauth.ReplayStore interface for fail-closed mode.
// Returns error if JTI already seen (replay detected).
func (a *DurableReplayStoreAdapter) CheckAndStore(jti string) error {
	seen, err := a.store.Seen(jti)
	if err != nil {
		return fmt.Errorf("replay store error: %w", err)
	}
	if seen {
		return fmt.Errorf("replay detected: JTI %s already seen", jti)
	}
	// Record with current timestamp
	if err := a.store.Record(jti, time.Now()); err != nil {
		return fmt.Errorf("record JTI failed: %w", err)
	}
	return nil
}

// Close closes the underlying store.
func (a *DurableReplayStoreAdapter) Close() error {
	return a.store.Close()
}

// SnapshotTrigger allows external snapshot triggering (e.g., from HTTP endpoint, signal handler).
type SnapshotTrigger struct {
	store *DurableReplayStore
}

// NewSnapshotTrigger creates a snapshot trigger for manual snapshots.
func NewSnapshotTrigger(store *DurableReplayStore) *SnapshotTrigger {
	return &SnapshotTrigger{store: store}
}

// Trigger performs an immediate snapshot.
func (st *SnapshotTrigger) Trigger(ctx context.Context) error {
	return st.store.Snapshot()
}

// NewDurableReplayStoreFromEnv creates a DurableReplayStore from environment variables.
// Supported env vars:
//   - GAUTH_REPLAY_WAL_PATH (default: ./data/replay.wal)
//   - GAUTH_REPLAY_TTL_SEC (default: 900 = 15 minutes)
//   - GAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC (default: 300 = 5 minutes)
//   - GAUTH_REPLAY_EVICTION_POLICY (default: ttl, options: ttl|lru|size|ttl+size)
//   - GAUTH_REPLAY_EVICTION_MAX_SIZE (default: 10000)
func NewDurableReplayStoreFromEnv(metrics DurableReplayMetrics) (*DurableReplayStore, error) {
	walPath := os.Getenv("GAUTH_REPLAY_WAL_PATH")
	if walPath == "" {
		walPath = "./data/replay.wal"
	}

	ttl := 15 * time.Minute
	if v := os.Getenv("GAUTH_REPLAY_TTL_SEC"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			ttl = time.Duration(secs) * time.Second
		}
	}

	snapshotInterval := 5 * time.Minute
	if v := os.Getenv("GAUTH_REPLAY_SNAPSHOT_INTERVAL_SEC"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
			snapshotInterval = time.Duration(secs) * time.Second
		}
	}

	maxSize := 10000
	if v := os.Getenv("GAUTH_REPLAY_EVICTION_MAX_SIZE"); v != "" {
		if size, err := strconv.Atoi(v); err == nil && size > 0 {
			maxSize = size
		}
	}

	policyStr := os.Getenv("GAUTH_REPLAY_EVICTION_POLICY")
	if policyStr == "" {
		policyStr = "ttl" // Default
	}
	policy := ParseEvictionPolicy(policyStr, ttl, maxSize)

	config := DurableReplayStoreConfig{
		WALPath:          walPath,
		TTL:              ttl,
		SnapshotInterval: snapshotInterval,
		Metrics:          metrics,
		EvictionPolicy:   policy,
		MaxSize:          maxSize,
	}

	return NewDurableReplayStore(config)
}
