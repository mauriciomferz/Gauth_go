// Package pdp provides distributed Policy Decision Point (PDP) clustering
// for high-availability AgentAuth deployments with cache invalidation.
package pdp

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ClusterNode represents a node in the distributed PDP cluster.
type ClusterNode struct {
	ID       string // Unique node identifier
	Address  string // Network address (host:port)
	Status   NodeStatus
	LastSeen time.Time
	Load     float64 // Current load factor (0.0 - 1.0)
}

// NodeStatus represents the health status of a cluster node.
type NodeStatus string

const (
	NodeStatusHealthy   NodeStatus = "healthy"
	NodeStatusUnhealthy NodeStatus = "unhealthy"
	NodeStatusDraining  NodeStatus = "draining" // Gracefully shutting down
	NodeStatusOffline   NodeStatus = "offline"
)

// DecisionRequest represents a policy decision request.
type DecisionRequest struct {
	RequestID string                 // Unique request ID
	Subject   string                 // Who is making the request
	Action    string                 // What action is requested
	Resource  string                 // What resource is being accessed
	Context   map[string]interface{} // Additional context
	Timestamp time.Time
	CacheTTL  time.Duration // How long to cache this decision
}

// DecisionResponse represents a policy decision response.
type DecisionResponse struct {
	RequestID   string
	Decision    Decision
	Reason      string
	CachedAt    time.Time
	ExpiresAt   time.Time
	NodeID      string // Which node made the decision
	Obligations []string
}

// Decision represents the policy decision result.
type Decision string

const (
	DecisionPermit        Decision = "permit"
	DecisionDeny          Decision = "deny"
	DecisionNotApplicable Decision = "not_applicable"
	DecisionIndeterminate Decision = "indeterminate"
)

// CacheEntry represents a cached decision with expiration.
type CacheEntry struct {
	Response  *DecisionResponse
	ExpiresAt time.Time
	Version   int64 // Version number for optimistic locking
}

// CacheInvalidation represents a cache invalidation message.
type CacheInvalidation struct {
	Pattern   string // Pattern to match for invalidation (e.g., "subject:user123:*")
	Reason    string // Why invalidation occurred
	Timestamp time.Time
	NodeID    string // Which node initiated invalidation
}

// DistributedPDP implements a distributed Policy Decision Point with clustering.
type DistributedPDP struct {
	mu sync.RWMutex

	// Node identity
	nodeID  string
	address string

	// Cluster membership
	nodes map[string]*ClusterNode

	// Decision cache
	cache        map[string]*CacheEntry
	cacheVersion int64

	// Configuration
	config *PDPConfig

	// Channels
	invalidationCh chan *CacheInvalidation
	healthCheckCh  chan struct{}
	shutdownCh     chan struct{}

	// Health
	isHealthy bool
}

// PDPConfig contains configuration for the distributed PDP.
type PDPConfig struct {
	NodeID              string
	Address             string
	CacheTTL            time.Duration
	CacheMaxSize        int
	HealthCheckInterval time.Duration
	ClusterSyncInterval time.Duration
	MaxRetries          int
	RequestTimeout      time.Duration
}

// NewDistributedPDP creates a new distributed PDP instance.
func NewDistributedPDP(config *PDPConfig) (*DistributedPDP, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}
	if config.NodeID == "" {
		return nil, errors.New("node ID is required")
	}
	if config.Address == "" {
		return nil, errors.New("address is required")
	}

	// Set defaults
	if config.HealthCheckInterval == 0 {
		config.HealthCheckInterval = 10 * time.Second
	}
	if config.CacheMaxSize == 0 {
		config.CacheMaxSize = 10000
	}

	pdp := &DistributedPDP{
		nodeID:         config.NodeID,
		address:        config.Address,
		nodes:          make(map[string]*ClusterNode),
		cache:          make(map[string]*CacheEntry),
		cacheVersion:   0,
		config:         config,
		invalidationCh: make(chan *CacheInvalidation, 1000),
		healthCheckCh:  make(chan struct{}, 1),
		shutdownCh:     make(chan struct{}),
		isHealthy:      true,
	}

	// Register self as a node
	pdp.nodes[config.NodeID] = &ClusterNode{
		ID:       config.NodeID,
		Address:  config.Address,
		Status:   NodeStatusHealthy,
		LastSeen: time.Now(),
		Load:     0.0,
	}

	return pdp, nil
}

// Start begins the distributed PDP operations.
func (pdp *DistributedPDP) Start(ctx context.Context) error {
	// Start background workers
	go pdp.healthCheckWorker(ctx)
	go pdp.invalidationWorker(ctx)
	go pdp.cacheCleanupWorker(ctx)

	return nil
}

// Stop gracefully shuts down the distributed PDP.
func (pdp *DistributedPDP) Stop() error {
	close(pdp.shutdownCh)

	pdp.mu.Lock()
	defer pdp.mu.Unlock()

	// Mark node as draining
	if node, exists := pdp.nodes[pdp.nodeID]; exists {
		node.Status = NodeStatusDraining
	}

	return nil
}

// MakeDecision evaluates a policy decision request.
// It first checks the local cache, then evaluates the policy if not cached.
func (pdp *DistributedPDP) MakeDecision(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error) {
	if req == nil {
		return nil, errors.New("request is required")
	}

	// Generate cache key
	cacheKey := pdp.generateCacheKey(req)

	// Check cache first
	pdp.mu.RLock()
	if entry, found := pdp.cache[cacheKey]; found {
		if time.Now().Before(entry.ExpiresAt) {
			// Cache hit
			pdp.mu.RUnlock()
			response := entry.Response
			response.NodeID = pdp.nodeID
			return response, nil
		}
	}
	pdp.mu.RUnlock()

	// Cache miss or expired - evaluate policy
	response, err := pdp.evaluatePolicy(ctx, req)
	if err != nil {
		return nil, err
	}

	// Cache the decision
	if req.CacheTTL > 0 {
		pdp.cacheDecision(cacheKey, response, req.CacheTTL)
	}

	return response, nil
}

// evaluatePolicy performs the actual policy evaluation.
// This is where the ABAC/RBAC logic would be implemented.
func (pdp *DistributedPDP) evaluatePolicy(ctx context.Context, req *DecisionRequest) (*DecisionResponse, error) {
	// Simplified policy evaluation for demonstration
	// In production, integrate with your policy engine

	response := &DecisionResponse{
		RequestID:   req.RequestID,
		Decision:    DecisionPermit, // Default for demo
		Reason:      "Policy evaluation complete",
		CachedAt:    time.Now(),
		ExpiresAt:   time.Now().Add(req.CacheTTL),
		NodeID:      pdp.nodeID,
		Obligations: []string{},
	}

	// Example: Deny if action is "delete" and resource starts with "critical:"
	if req.Action == "delete" && len(req.Resource) > 9 && req.Resource[:9] == "critical:" {
		response.Decision = DecisionDeny
		response.Reason = "Deletion of critical resources is prohibited"
	}

	return response, nil
}

// generateCacheKey creates a deterministic cache key from the request.
func (pdp *DistributedPDP) generateCacheKey(req *DecisionRequest) string {
	// Create deterministic key from request fields
	data := fmt.Sprintf("%s|%s|%s", req.Subject, req.Action, req.Resource)

	// Include context in key if present
	if len(req.Context) > 0 {
		contextJSON, _ := json.Marshal(req.Context)
		data += "|" + string(contextJSON)
	}

	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)
}

// cacheDecision stores a decision in the cache.
func (pdp *DistributedPDP) cacheDecision(key string, response *DecisionResponse, ttl time.Duration) {
	pdp.mu.Lock()
	defer pdp.mu.Unlock()

	// Check cache size limit
	if len(pdp.cache) >= pdp.config.CacheMaxSize {
		// Evict oldest entry (simple LRU)
		pdp.evictOldest()
	}

	pdp.cacheVersion++
	pdp.cache[key] = &CacheEntry{
		Response:  response,
		ExpiresAt: time.Now().Add(ttl),
		Version:   pdp.cacheVersion,
	}
}

// evictOldest removes the oldest cache entry.
func (pdp *DistributedPDP) evictOldest() {
	var oldestKey string
	var oldestTime time.Time
	first := true

	for key, entry := range pdp.cache {
		if first || entry.ExpiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.ExpiresAt
			first = false
		}
	}

	if oldestKey != "" {
		delete(pdp.cache, oldestKey)
	}
}

// InvalidateCache invalidates cache entries matching the pattern.
func (pdp *DistributedPDP) InvalidateCache(pattern string, reason string) error {
	invalidation := &CacheInvalidation{
		Pattern:   pattern,
		Reason:    reason,
		Timestamp: time.Now(),
		NodeID:    pdp.nodeID,
	}

	// Send to local invalidation channel
	select {
	case pdp.invalidationCh <- invalidation:
		// Also broadcast to other nodes in the cluster
		return pdp.broadcastInvalidation(invalidation)
	default:
		return errors.New("invalidation channel full")
	}
}

// broadcastInvalidation sends invalidation to all cluster nodes.
func (pdp *DistributedPDP) broadcastInvalidation(inv *CacheInvalidation) error {
	pdp.mu.RLock()
	nodes := make([]*ClusterNode, 0, len(pdp.nodes))
	for _, node := range pdp.nodes {
		if node.ID != pdp.nodeID && node.Status == NodeStatusHealthy {
			nodes = append(nodes, node)
		}
	}
	pdp.mu.RUnlock()

	// In production, send HTTP/gRPC messages to other nodes
	// For now, just log the broadcast
	for _, node := range nodes {
		_ = node // Would send invalidation to node.Address
	}

	return nil
}

// invalidationWorker processes cache invalidations.
func (pdp *DistributedPDP) invalidationWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-pdp.shutdownCh:
			return
		case inv := <-pdp.invalidationCh:
			pdp.processInvalidation(inv)
		}
	}
}

// processInvalidation applies a cache invalidation.
func (pdp *DistributedPDP) processInvalidation(inv *CacheInvalidation) {
	pdp.mu.Lock()
	defer pdp.mu.Unlock()

	// Simple pattern matching - in production use more sophisticated matching
	keysToRemove := []string{}
	for key := range pdp.cache {
		// Pattern matching logic
		if pdp.matchesPattern(key, inv.Pattern) {
			keysToRemove = append(keysToRemove, key)
		}
	}

	for _, key := range keysToRemove {
		delete(pdp.cache, key)
	}
}

// matchesPattern checks if a cache key matches an invalidation pattern.
func (pdp *DistributedPDP) matchesPattern(key, pattern string) bool {
	// Simplified pattern matching
	// Pattern format: "prefix:*" for wildcard matching
	if len(pattern) == 0 {
		return false
	}

	if pattern == "*" {
		return true // Match all
	}

	// Check for wildcard suffix
	if len(pattern) > 2 && pattern[len(pattern)-2:] == ":*" {
		prefix := pattern[:len(pattern)-2]
		return len(key) >= len(prefix) && key[:len(prefix)] == prefix
	}

	return key == pattern
}

// healthCheckWorker performs periodic health checks on cluster nodes.
func (pdp *DistributedPDP) healthCheckWorker(ctx context.Context) {
	ticker := time.NewTicker(pdp.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pdp.shutdownCh:
			return
		case <-ticker.C:
			pdp.performHealthCheck()
		}
	}
}

// performHealthCheck checks the health of all cluster nodes.
func (pdp *DistributedPDP) performHealthCheck() {
	pdp.mu.Lock()
	defer pdp.mu.Unlock()

	now := time.Now()
	for _, node := range pdp.nodes {
		if node.ID == pdp.nodeID {
			node.LastSeen = now
			continue
		}

		// Check if node is stale
		if now.Sub(node.LastSeen) > pdp.config.HealthCheckInterval*3 {
			node.Status = NodeStatusUnhealthy
		}
	}
}

// cacheCleanupWorker removes expired cache entries.
func (pdp *DistributedPDP) cacheCleanupWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pdp.shutdownCh:
			return
		case <-ticker.C:
			pdp.cleanupExpiredCache()
		}
	}
}

// cleanupExpiredCache removes expired entries from the cache.
func (pdp *DistributedPDP) cleanupExpiredCache() {
	pdp.mu.Lock()
	defer pdp.mu.Unlock()

	now := time.Now()
	for key, entry := range pdp.cache {
		if now.After(entry.ExpiresAt) {
			delete(pdp.cache, key)
		}
	}
}

// AddNode registers a new node in the cluster.
func (pdp *DistributedPDP) AddNode(node *ClusterNode) error {
	if node == nil || node.ID == "" {
		return errors.New("invalid node")
	}

	pdp.mu.Lock()
	defer pdp.mu.Unlock()

	pdp.nodes[node.ID] = node
	return nil
}

// RemoveNode removes a node from the cluster.
func (pdp *DistributedPDP) RemoveNode(nodeID string) error {
	pdp.mu.Lock()
	defer pdp.mu.Unlock()

	if nodeID == pdp.nodeID {
		return errors.New("cannot remove self")
	}

	delete(pdp.nodes, nodeID)
	return nil
}

// GetClusterStatus returns the current status of all cluster nodes.
func (pdp *DistributedPDP) GetClusterStatus() []*ClusterNode {
	pdp.mu.RLock()
	defer pdp.mu.RUnlock()

	nodes := make([]*ClusterNode, 0, len(pdp.nodes))
	for _, node := range pdp.nodes {
		// Create copy to avoid race conditions
		nodeCopy := *node
		nodes = append(nodes, &nodeCopy)
	}

	return nodes
}

// GetCacheStats returns statistics about the decision cache.
func (pdp *DistributedPDP) GetCacheStats() map[string]interface{} {
	pdp.mu.RLock()
	defer pdp.mu.RUnlock()

	return map[string]interface{}{
		"size":     len(pdp.cache),
		"max_size": pdp.config.CacheMaxSize,
		"version":  pdp.cacheVersion,
	}
}
