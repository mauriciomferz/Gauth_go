package gauthplus

import (
	"context"
	"sync"
	"time"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/metrics"
)

// CapabilityCache provides thread-safe caching for AI capability assessments
// Reduces database queries for frequently accessed assessments
type CapabilityCache struct {
	cache map[string]*cacheEntry[*AICapabilityAssessment]
	ttl   time.Duration
	mu    sync.RWMutex
}

// DelegationChainCache provides thread-safe caching for delegation chains
// Reduces database queries for delegation chain lookups
type DelegationChainCache struct {
	cache map[string]*cacheEntry[[]*AIDelegation]
	ttl   time.Duration
	mu    sync.RWMutex
}

// cacheEntry wraps cached data with expiration time
type cacheEntry[T any] struct {
	data      T
	expiresAt time.Time
}

// NewCapabilityCache creates a new capability assessment cache
func NewCapabilityCache(ttl time.Duration) *CapabilityCache {
	return &CapabilityCache{
		cache: make(map[string]*cacheEntry[*AICapabilityAssessment]),
		ttl:   ttl,
	}
}

// NewDelegationChainCache creates a new delegation chain cache
func NewDelegationChainCache(ttl time.Duration) *DelegationChainCache {
	return &DelegationChainCache{
		cache: make(map[string]*cacheEntry[[]*AIDelegation]),
		ttl:   ttl,
	}
}

// Get retrieves a capability assessment from cache
func (c *CapabilityCache) Get(agentID string) (*AICapabilityAssessment, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[agentID]
	if !exists {
		metrics.RecordGAuthPlusCacheOperation("capability", false)
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		metrics.RecordGAuthPlusCacheOperation("capability", false)
		return nil, false
	}

	metrics.RecordGAuthPlusCacheOperation("capability", true)
	return entry.data, true
}

// Set stores a capability assessment in cache
func (c *CapabilityCache) Set(agentID string, assessment *AICapabilityAssessment) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[agentID] = &cacheEntry[*AICapabilityAssessment]{
		data:      assessment,
		expiresAt: time.Now().Add(c.ttl),
	}
	metrics.UpdateGAuthPlusCacheSize("capability", len(c.cache))
}

// Invalidate removes a specific entry from cache
func (c *CapabilityCache) Invalidate(agentID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, agentID)
}

// Clear removes all entries from cache
func (c *CapabilityCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*cacheEntry[*AICapabilityAssessment])
}

// CleanExpired removes expired entries from cache
func (c *CapabilityCache) CleanExpired() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range c.cache {
		if now.After(entry.expiresAt) {
			delete(c.cache, key)
			removed++
		}
	}

	return removed
}

// Size returns the number of entries in cache
func (c *CapabilityCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.cache)
}

// Get retrieves a delegation chain from cache
func (d *DelegationChainCache) Get(agentID string) ([]*AIDelegation, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	entry, exists := d.cache[agentID]
	if !exists {
		metrics.RecordGAuthPlusCacheOperation("delegation", false)
		return nil, false
	}

	// Check if expired
	if time.Now().After(entry.expiresAt) {
		metrics.RecordGAuthPlusCacheOperation("delegation", false)
		return nil, false
	}

	metrics.RecordGAuthPlusCacheOperation("delegation", true)
	return entry.data, true
}

// Set stores a delegation chain in cache
func (d *DelegationChainCache) Set(agentID string, chain []*AIDelegation) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache[agentID] = &cacheEntry[[]*AIDelegation]{
		data:      chain,
		expiresAt: time.Now().Add(d.ttl),
	}
	metrics.UpdateGAuthPlusCacheSize("delegation", len(d.cache))
}

// Invalidate removes a specific entry from cache
func (d *DelegationChainCache) Invalidate(agentID string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.cache, agentID)
}

// InvalidateAll removes all delegation chains involving an agent
// This is useful when a delegation is created or revoked
func (d *DelegationChainCache) InvalidateAll() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache = make(map[string]*cacheEntry[[]*AIDelegation])
}

// Clear removes all entries from cache
func (d *DelegationChainCache) Clear() {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.cache = make(map[string]*cacheEntry[[]*AIDelegation])
}

// CleanExpired removes expired entries from cache
func (d *DelegationChainCache) CleanExpired() int {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range d.cache {
		if now.After(entry.expiresAt) {
			delete(d.cache, key)
			removed++
		}
	}

	return removed
}

// Size returns the number of entries in cache
func (d *DelegationChainCache) Size() int {
	d.mu.RLock()
	defer d.mu.RUnlock()

	return len(d.cache)
}

// CachedCapabilityService wraps a CapabilityAssessmentService with caching
type CachedCapabilityService struct {
	service CapabilityAssessmentService
	cache   *CapabilityCache
}

// NewCachedCapabilityService creates a new cached capability service
func NewCachedCapabilityService(service CapabilityAssessmentService, ttl time.Duration) *CachedCapabilityService {
	return &CachedCapabilityService{
		service: service,
		cache:   NewCapabilityCache(ttl),
	}
}

// GetLatestAssessment retrieves the latest assessment with caching
func (s *CachedCapabilityService) GetLatestAssessment(ctx context.Context, agentID string) (*AICapabilityAssessment, error) {
	// Try cache first
	if assessment, found := s.cache.Get(agentID); found {
		return assessment, nil
	}

	// Cache miss - fetch from database
	assessment, err := s.service.GetLatestAssessment(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	if assessment != nil {
		s.cache.Set(agentID, assessment)
	}

	return assessment, nil
}

// CreateAssessment creates a new assessment and invalidates cache
func (s *CachedCapabilityService) CreateAssessment(ctx context.Context, assessment *AICapabilityAssessment) error {
	err := s.service.CreateAssessment(ctx, assessment)
	if err != nil {
		return err
	}

	// Invalidate cache for this agent
	s.cache.Invalidate(assessment.AgentID)

	return nil
}

// CheckCapabilityMatch delegates to underlying service (no caching)
func (s *CachedCapabilityService) CheckCapabilityMatch(ctx context.Context, agentID string, requirements *CapabilityRequirements) (bool, []string, error) {
	return s.service.CheckCapabilityMatch(ctx, agentID, requirements)
}

// GetExpiringAssessments delegates to underlying service (no caching)
func (s *CachedCapabilityService) GetExpiringAssessments(ctx context.Context, daysUntilExpiry int) ([]*AICapabilityAssessment, error) {
	return s.service.GetExpiringAssessments(ctx, daysUntilExpiry)
}

// GetCache returns the underlying cache for direct access (e.g., cleanup)
func (s *CachedCapabilityService) GetCache() *CapabilityCache {
	return s.cache
}

// CachedDelegationService wraps a DelegationService with caching
type CachedDelegationService struct {
	service DelegationService
	cache   *DelegationChainCache
}

// NewCachedDelegationService creates a new cached delegation service
func NewCachedDelegationService(service DelegationService, ttl time.Duration) *CachedDelegationService {
	return &CachedDelegationService{
		service: service,
		cache:   NewDelegationChainCache(ttl),
	}
}

// GetDelegationChain retrieves delegation chain with caching
func (s *CachedDelegationService) GetDelegationChain(ctx context.Context, agentID string) ([]*AIDelegation, error) {
	// Try cache first
	if chain, found := s.cache.Get(agentID); found {
		return chain, nil
	}

	// Cache miss - fetch from database
	chain, err := s.service.GetDelegationChain(ctx, agentID)
	if err != nil {
		return nil, err
	}

	// Store in cache
	s.cache.Set(agentID, chain)

	return chain, nil
}

// CreateDelegation creates a new delegation and invalidates all caches
func (s *CachedDelegationService) CreateDelegation(ctx context.Context, delegation *AIDelegation) error {
	err := s.service.CreateDelegation(ctx, delegation)
	if err != nil {
		return err
	}

	// Invalidate all caches since delegation chains may have changed
	s.cache.InvalidateAll()

	return nil
}

// RevokeDelegation revokes a delegation and invalidates all caches
func (s *CachedDelegationService) RevokeDelegation(ctx context.Context, delegationID, revokedBy, reason string) error {
	err := s.service.RevokeDelegation(ctx, delegationID, revokedBy, reason)
	if err != nil {
		return err
	}

	// Invalidate all caches since delegation chains may have changed
	s.cache.InvalidateAll()

	return nil
}

// ValidateDelegation delegates to underlying service (no caching)
func (s *CachedDelegationService) ValidateDelegation(ctx context.Context, sourceAgent, targetAgent string, scope []string, depth int) error {
	return s.service.ValidateDelegation(ctx, sourceAgent, targetAgent, scope, depth)
}

// CheckMaxDepthExceeded delegates to underlying service (no caching)
func (s *CachedDelegationService) CheckMaxDepthExceeded(ctx context.Context, sourceAgentID string, currentDepth int) (bool, error) {
	return s.service.CheckMaxDepthExceeded(ctx, sourceAgentID, currentDepth)
}

// GetCache returns the underlying cache for direct access (e.g., cleanup)
func (s *CachedDelegationService) GetCache() *DelegationChainCache {
	return s.cache
}
