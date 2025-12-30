// Package pip provides Power Information Point (PIP) implementation
// This consolidates authorization data from scattered sources per AAP001
package pip

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/poa"
	"github.com/mauriciomferz/AgentAuth/pkg/pvp"
	"github.com/mauriciomferz/AgentAuth/pkg/registry"
)

// PowerInformationPoint (PIP) consolidates authorization data from multiple sources
// AAP001 §5: PIP provides centralized access to:
// - Power of Attorney definitions
// - Authorization chains
// - Commercial register data
// - Identity verification results
// - Trust service provider information
type PowerInformationPoint interface {
	// GetPoADefinition retrieves a Power of Attorney definition by ID
	GetPoADefinition(ctx context.Context, poaID string) (*poa.PoADefinition, error)

	// GetAuthorizationChain retrieves the complete authorization chain for a client
	GetAuthorizationChain(ctx context.Context, clientID string) (*agentauth.AuthorizationChain, error)

	// GetClientOwnerInfo retrieves client owner information
	GetClientOwnerInfo(ctx context.Context, ownerID string) (*agentauth.ClientOwnerInfo, error)

	// GetOwnersAuthorizerInfo retrieves owner's authorizer information
	GetOwnersAuthorizerInfo(ctx context.Context, authorizerID string) (*agentauth.OwnersAuthorizerInfo, error)

	// VerifyCommercialRegister verifies commercial register entry
	VerifyCommercialRegister(ctx context.Context, registrationNumber, jurisdiction string) (*registry.RegistrationVerificationResult, error)

	// VerifyIdentityChain verifies complete identity verification chain
	VerifyIdentityChain(ctx context.Context, req *pvp.IdentityChainVerificationRequest) (*pvp.IdentityChainVerificationResult, error)

	// GetTrustServiceProvider retrieves TSP information
	GetTrustServiceProvider(ctx context.Context, tspID string) (*pvp.TSPVerificationResult, error)

	// GetAuthorizedActions retrieves authorized actions for a client
	GetAuthorizedActions(ctx context.Context, clientID string) (*poa.AuthorizedActions, error)

	// GetGeographicScope retrieves authorized geographic regions
	GetGeographicScope(ctx context.Context, clientID string) ([]poa.GeographicScope, error)

	// GetIndustrySectors retrieves authorized industry sectors
	GetIndustrySectors(ctx context.Context, clientID string) ([]poa.IndustrySector, error)

	// GetPowerLimits retrieves power limitations for a client
	GetPowerLimits(ctx context.Context, clientID string) (*poa.PowerLimitSet, error)

	// GetRightsObligations retrieves rights and obligations
	GetRightsObligations(ctx context.Context, clientID string) (*poa.RightsObligationSet, error)

	// ValidateAuthorization validates if a specific action is authorized
	ValidateAuthorization(ctx context.Context, req *AuthorizationValidationRequest) (*AuthorizationValidationResult, error)

	// RefreshCache refreshes cached authorization data
	RefreshCache(ctx context.Context, clientID string) error

	// GetCacheStats returns cache statistics
	GetCacheStats() *CacheStats
}

// AuthorizationValidationRequest represents a request to validate authorization
type AuthorizationValidationRequest struct {
	ClientID       string
	Action         string
	Resource       string
	Jurisdiction   string
	IndustrySector string
	Timestamp      time.Time
}

// AuthorizationValidationResult represents the result of authorization validation
type AuthorizationValidationResult struct {
	Authorized         bool
	AuthorizationChain *agentauth.AuthorizationChain
	ValidatedActions   []string
	Restrictions       []string
	Warnings           []string
	ValidationTime     time.Time
	CacheHit           bool
}

// CacheStats provides cache statistics
type CacheStats struct {
	TotalEntries      int
	HitRate           float64
	MissRate          float64
	EvictionCount     int64
	AverageAccessTime time.Duration
}

// DefaultPIP is the default implementation of PowerInformationPoint
type DefaultPIP struct {
	// Data sources
	poaService         poa.Service
	commercialRegister registry.CommercialRegisterService
	pvp                pvp.PowerVerificationPoint

	// Caching
	cache    *AuthorizationCache
	cacheTTL time.Duration

	// Metrics
	mu            sync.RWMutex
	totalRequests int64
	cacheHits     int64
	cacheMisses   int64
}

// NewDefaultPIP creates a new default PIP implementation
func NewDefaultPIP(
	poaService poa.Service,
	commercialRegister registry.CommercialRegisterService,
	pvp pvp.PowerVerificationPoint,
	cacheTTL time.Duration,
) *DefaultPIP {
	return &DefaultPIP{
		poaService:         poaService,
		commercialRegister: commercialRegister,
		pvp:                pvp,
		cache:              NewAuthorizationCache(1000, cacheTTL), // 1000 entries default
		cacheTTL:           cacheTTL,
	}
}

// GetPoADefinition retrieves a Power of Attorney definition
func (pip *DefaultPIP) GetPoADefinition(ctx context.Context, poaID string) (*poa.PoADefinition, error) {
	// Check cache first
	if cached := pip.cache.GetPoA(poaID); cached != nil {
		pip.recordCacheHit()
		return cached, nil
	}
	pip.recordCacheMiss()

	// Retrieve from PoA service
	// Note: This assumes poa.Service has a method to get PoA by ID
	// In practice, you'd need to implement this in the poa.Service interface

	// For now, return a placeholder error
	return nil, fmt.Errorf("PoA retrieval not yet implemented in poa.Service")
}

// GetAuthorizationChain retrieves the complete authorization chain
func (pip *DefaultPIP) GetAuthorizationChain(ctx context.Context, clientID string) (*agentauth.AuthorizationChain, error) {
	// Check cache
	if cached := pip.cache.GetAuthorizationChain(clientID); cached != nil {
		pip.recordCacheHit()
		return cached, nil
	}
	pip.recordCacheMiss()

	// Build authorization chain from multiple sources
	// This is a complex operation that requires:
	// 1. Get client information
	// 2. Get client owner information
	// 3. Get owner's authorizer information
	// 4. Verify each link in the chain

	// Placeholder implementation
	chain := &agentauth.AuthorizationChain{
		// Would populate OwnersAuthorizer, ClientOwner, Client
	}

	// Cache the result
	pip.cache.SetAuthorizationChain(clientID, chain)

	return chain, nil
}

// GetClientOwnerInfo retrieves client owner information
func (pip *DefaultPIP) GetClientOwnerInfo(ctx context.Context, ownerID string) (*agentauth.ClientOwnerInfo, error) {
	// Check cache
	if cached := pip.cache.GetClientOwner(ownerID); cached != nil {
		pip.recordCacheHit()
		return cached, nil
	}
	pip.recordCacheMiss()

	// In practice, this would query a database or external service
	// For now, return placeholder
	return nil, fmt.Errorf("client owner retrieval not yet implemented")
}

// GetOwnersAuthorizerInfo retrieves owner's authorizer information
func (pip *DefaultPIP) GetOwnersAuthorizerInfo(ctx context.Context, authorizerID string) (*agentauth.OwnersAuthorizerInfo, error) {
	// Check cache
	if cached := pip.cache.GetOwnersAuthorizer(authorizerID); cached != nil {
		pip.recordCacheHit()
		return cached, nil
	}
	pip.recordCacheMiss()

	// Retrieve from commercial register and other sources
	// Placeholder implementation
	return nil, fmt.Errorf("owners authorizer retrieval not yet implemented")
}

// VerifyCommercialRegister verifies commercial register entry
func (pip *DefaultPIP) VerifyCommercialRegister(ctx context.Context, registrationNumber, jurisdiction string) (*registry.RegistrationVerificationResult, error) {
	// Check cache
	cacheKey := fmt.Sprintf("%s:%s", jurisdiction, registrationNumber)
	if cached := pip.cache.GetCommercialRegisterVerification(cacheKey); cached != nil {
		pip.recordCacheHit()
		return cached, nil
	}
	pip.recordCacheMiss()

	// Verify with commercial register service
	req := &registry.RegistrationVerificationRequest{
		RegistrationNumber: registrationNumber,
		Jurisdiction:       jurisdiction,
	}
	result, err := pip.commercialRegister.VerifyRegistration(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("commercial register verification: %w", err)
	}

	// Cache the result
	pip.cache.SetCommercialRegisterVerification(cacheKey, result)

	return result, nil
}

// VerifyIdentityChain verifies complete identity verification chain
func (pip *DefaultPIP) VerifyIdentityChain(ctx context.Context, req *pvp.IdentityChainVerificationRequest) (*pvp.IdentityChainVerificationResult, error) {
	// Delegate to PVP
	result, err := pip.pvp.VerifyIdentityChain(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("identity chain verification: %w", err)
	}

	return result, nil
}

// GetTrustServiceProvider retrieves TSP information
func (pip *DefaultPIP) GetTrustServiceProvider(ctx context.Context, tspID string) (*pvp.TSPVerificationResult, error) {
	// Check cache
	if cached := pip.cache.GetTSP(tspID); cached != nil {
		pip.recordCacheHit()
		return cached, nil
	}
	pip.recordCacheMiss()

	// Retrieve from PVP
	// Note: PVP interface doesn't currently have GetTSP method
	// This would need to be added
	return nil, fmt.Errorf("TSP retrieval not yet implemented in PVP")
}

// GetAuthorizedActions retrieves authorized actions for a client
func (pip *DefaultPIP) GetAuthorizedActions(ctx context.Context, clientID string) (*poa.AuthorizedActions, error) {
	// This would be extracted from the PoA definition
	// Check cache first
	if cached := pip.cache.GetAuthorizedActions(clientID); cached != nil {
		pip.recordCacheHit()
		return cached, nil
	}
	pip.recordCacheMiss()

	// Retrieve from PoA service and extract actions
	return nil, fmt.Errorf("authorized actions retrieval not yet implemented")
}

// GetGeographicScope retrieves authorized geographic regions
func (pip *DefaultPIP) GetGeographicScope(ctx context.Context, clientID string) ([]poa.GeographicScope, error) {
	// Similar to GetAuthorizedActions
	return nil, fmt.Errorf("geographic scope retrieval not yet implemented")
}

// GetIndustrySectors retrieves authorized industry sectors
func (pip *DefaultPIP) GetIndustrySectors(ctx context.Context, clientID string) ([]poa.IndustrySector, error) {
	return nil, fmt.Errorf("industry sectors retrieval not yet implemented")
}

// GetPowerLimits retrieves power limitations for a client
func (pip *DefaultPIP) GetPowerLimits(ctx context.Context, clientID string) (*poa.PowerLimitSet, error) {
	return nil, fmt.Errorf("power limits retrieval not yet implemented")
}

// GetRightsObligations retrieves rights and obligations
func (pip *DefaultPIP) GetRightsObligations(ctx context.Context, clientID string) (*poa.RightsObligationSet, error) {
	return nil, fmt.Errorf("rights obligations retrieval not yet implemented")
}

// ValidateAuthorization validates if a specific action is authorized
func (pip *DefaultPIP) ValidateAuthorization(ctx context.Context, req *AuthorizationValidationRequest) (*AuthorizationValidationResult, error) {
	startTime := time.Now()

	// 1. Get authorization chain
	chain, err := pip.GetAuthorizationChain(ctx, req.ClientID)
	if err != nil {
		return &AuthorizationValidationResult{
			Authorized:     false,
			ValidationTime: time.Now(),
		}, fmt.Errorf("get authorization chain: %w", err)
	}

	// 2. Get authorized actions
	actions, err := pip.GetAuthorizedActions(ctx, req.ClientID)
	if err != nil {
		return &AuthorizationValidationResult{
			Authorized:     false,
			ValidationTime: time.Now(),
		}, fmt.Errorf("get authorized actions: %w", err)
	}

	// 3. Validate action is in authorized set
	authorized := pip.isActionAuthorized(req.Action, actions)

	// 4. Check geographic restrictions
	if req.Jurisdiction != "" {
		geoScope, err := pip.GetGeographicScope(ctx, req.ClientID)
		if err == nil && !pip.isJurisdictionAuthorized(req.Jurisdiction, geoScope) {
			authorized = false
		}
	}

	// 5. Check industry sector restrictions
	if req.IndustrySector != "" {
		sectors, err := pip.GetIndustrySectors(ctx, req.ClientID)
		if err == nil && !pip.isSectorAuthorized(req.IndustrySector, sectors) {
			authorized = false
		}
	}

	// 6. Get power limits and check restrictions
	limits, err := pip.GetPowerLimits(ctx, req.ClientID)
	var restrictions []string
	if err == nil {
		restrictions = pip.getApplicableRestrictions(req.Action, limits)
	}

	return &AuthorizationValidationResult{
		Authorized:         authorized,
		AuthorizationChain: chain,
		ValidatedActions:   pip.extractValidatedActions(actions),
		Restrictions:       restrictions,
		ValidationTime:     time.Now(),
		CacheHit:           time.Since(startTime) < 10*time.Millisecond, // Simple heuristic
	}, nil
}

// Helper methods

func (pip *DefaultPIP) isActionAuthorized(action string, actions *poa.AuthorizedActions) bool {
	if actions == nil {
		return false
	}

	// Check transactions
	for _, t := range actions.Transactions {
		if string(t) == action {
			return true
		}
	}

	// Check decisions
	for _, d := range actions.Decisions {
		if string(d) == action {
			return true
		}
	}

	// Check non-physical actions
	for _, npa := range actions.NonPhysicalActions {
		if string(npa) == action {
			return true
		}
	}

	// Check physical actions
	for _, pa := range actions.PhysicalActions {
		if string(pa) == action {
			return true
		}
	}

	return false
}

func (pip *DefaultPIP) isJurisdictionAuthorized(jurisdiction string, scopes []poa.GeographicScope) bool {
	return poa.IsAuthorizedInRegion(scopes, jurisdiction)
}

func (pip *DefaultPIP) isSectorAuthorized(sector string, sectors []poa.IndustrySector) bool {
	for _, s := range sectors {
		if s.Authorized && (string(s.Code) == sector || s.Description == sector) {
			return true
		}
	}
	return false
}

func (pip *DefaultPIP) getApplicableRestrictions(action string, limits *poa.PowerLimitSet) []string {
	if limits == nil {
		return nil
	}

	var restrictions []string
	// In practice, this would analyze the power limits and extract applicable restrictions
	// For now, placeholder
	return restrictions
}

func (pip *DefaultPIP) extractValidatedActions(actions *poa.AuthorizedActions) []string {
	if actions == nil {
		return nil
	}

	var result []string
	for _, t := range actions.Transactions {
		result = append(result, string(t))
	}
	for _, d := range actions.Decisions {
		result = append(result, string(d))
	}
	for _, npa := range actions.NonPhysicalActions {
		result = append(result, string(npa))
	}
	for _, pa := range actions.PhysicalActions {
		result = append(result, string(pa))
	}

	return result
}

// RefreshCache refreshes cached authorization data
func (pip *DefaultPIP) RefreshCache(ctx context.Context, clientID string) error {
	// Remove from cache to force refresh on next access
	pip.cache.Invalidate(clientID)
	return nil
}

// GetCacheStats returns cache statistics
func (pip *DefaultPIP) GetCacheStats() *CacheStats {
	pip.mu.RLock()
	defer pip.mu.RUnlock()

	totalRequests := pip.totalRequests
	if totalRequests == 0 {
		return &CacheStats{
			TotalEntries: pip.cache.Size(),
			HitRate:      0,
			MissRate:     0,
		}
	}

	hitRate := float64(pip.cacheHits) / float64(totalRequests)
	missRate := float64(pip.cacheMisses) / float64(totalRequests)

	return &CacheStats{
		TotalEntries: pip.cache.Size(),
		HitRate:      hitRate,
		MissRate:     missRate,
	}
}

// Metrics helpers

func (pip *DefaultPIP) recordCacheHit() {
	pip.mu.Lock()
	defer pip.mu.Unlock()
	pip.totalRequests++
	pip.cacheHits++
}

func (pip *DefaultPIP) recordCacheMiss() {
	pip.mu.Lock()
	defer pip.mu.Unlock()
	pip.totalRequests++
	pip.cacheMisses++
}

// AuthorizationCache provides caching for authorization data
type AuthorizationCache struct {
	mu                  sync.RWMutex
	poaDefinitions      map[string]*cachedPoA
	authorizationChains map[string]*cachedAuthChain
	clientOwners        map[string]*cachedClientOwner
	ownersAuthorizers   map[string]*cachedOwnersAuthorizer
	commercialRegister  map[string]*cachedCommercialReg
	tsps                map[string]*cachedTSP
	authorizedActions   map[string]*cachedActions
	maxEntries          int
	ttl                 time.Duration
}

// Cached data structures

type cachedPoA struct {
	data      *poa.PoADefinition
	timestamp time.Time
}

type cachedAuthChain struct {
	data      *agentauth.AuthorizationChain
	timestamp time.Time
}

type cachedClientOwner struct {
	data      *agentauth.ClientOwnerInfo
	timestamp time.Time
}

type cachedOwnersAuthorizer struct {
	data      *agentauth.OwnersAuthorizerInfo
	timestamp time.Time
}

type cachedCommercialReg struct {
	data      *registry.RegistrationVerificationResult
	timestamp time.Time
}

type cachedTSP struct {
	data      *pvp.TSPVerificationResult
	timestamp time.Time
}

type cachedActions struct {
	data      *poa.AuthorizedActions
	timestamp time.Time
}

// NewAuthorizationCache creates a new authorization cache
func NewAuthorizationCache(maxEntries int, ttl time.Duration) *AuthorizationCache {
	return &AuthorizationCache{
		poaDefinitions:      make(map[string]*cachedPoA),
		authorizationChains: make(map[string]*cachedAuthChain),
		clientOwners:        make(map[string]*cachedClientOwner),
		ownersAuthorizers:   make(map[string]*cachedOwnersAuthorizer),
		commercialRegister:  make(map[string]*cachedCommercialReg),
		tsps:                make(map[string]*cachedTSP),
		authorizedActions:   make(map[string]*cachedActions),
		maxEntries:          maxEntries,
		ttl:                 ttl,
	}
}

// Cache accessor methods

func (c *AuthorizationCache) GetPoA(poaID string) *poa.PoADefinition {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.poaDefinitions[poaID]
	if !exists || time.Since(cached.timestamp) > c.ttl {
		return nil
	}
	return cached.data
}

func (c *AuthorizationCache) SetPoA(poaID string, poa *poa.PoADefinition) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.poaDefinitions[poaID] = &cachedPoA{
		data:      poa,
		timestamp: time.Now(),
	}
	c.evictIfNeeded()
}

func (c *AuthorizationCache) GetAuthorizationChain(clientID string) *agentauth.AuthorizationChain {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.authorizationChains[clientID]
	if !exists || time.Since(cached.timestamp) > c.ttl {
		return nil
	}
	return cached.data
}

func (c *AuthorizationCache) SetAuthorizationChain(clientID string, chain *agentauth.AuthorizationChain) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.authorizationChains[clientID] = &cachedAuthChain{
		data:      chain,
		timestamp: time.Now(),
	}
	c.evictIfNeeded()
}

func (c *AuthorizationCache) GetClientOwner(ownerID string) *agentauth.ClientOwnerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.clientOwners[ownerID]
	if !exists || time.Since(cached.timestamp) > c.ttl {
		return nil
	}
	return cached.data
}

func (c *AuthorizationCache) GetOwnersAuthorizer(authorizerID string) *agentauth.OwnersAuthorizerInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.ownersAuthorizers[authorizerID]
	if !exists || time.Since(cached.timestamp) > c.ttl {
		return nil
	}
	return cached.data
}

func (c *AuthorizationCache) GetCommercialRegisterVerification(key string) *registry.RegistrationVerificationResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.commercialRegister[key]
	if !exists || time.Since(cached.timestamp) > c.ttl {
		return nil
	}
	return cached.data
}

func (c *AuthorizationCache) SetCommercialRegisterVerification(key string, result *registry.RegistrationVerificationResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.commercialRegister[key] = &cachedCommercialReg{
		data:      result,
		timestamp: time.Now(),
	}
	c.evictIfNeeded()
}

func (c *AuthorizationCache) GetTSP(tspID string) *pvp.TSPVerificationResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.tsps[tspID]
	if !exists || time.Since(cached.timestamp) > c.ttl {
		return nil
	}
	return cached.data
}

func (c *AuthorizationCache) GetAuthorizedActions(clientID string) *poa.AuthorizedActions {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cached, exists := c.authorizedActions[clientID]
	if !exists || time.Since(cached.timestamp) > c.ttl {
		return nil
	}
	return cached.data
}

func (c *AuthorizationCache) Invalidate(clientID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.authorizationChains, clientID)
	delete(c.clientOwners, clientID)
	delete(c.authorizedActions, clientID)
}

func (c *AuthorizationCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.poaDefinitions) +
		len(c.authorizationChains) +
		len(c.clientOwners) +
		len(c.ownersAuthorizers) +
		len(c.commercialRegister) +
		len(c.tsps) +
		len(c.authorizedActions)
}

func (c *AuthorizationCache) evictIfNeeded() {
	// Simple eviction: if we exceed maxEntries, clear oldest entries
	// In practice, you'd use a more sophisticated LRU algorithm

	// Calculate size inline to avoid deadlock (don't call Size() while holding write lock)
	size := len(c.poaDefinitions) +
		len(c.authorizationChains) +
		len(c.clientOwners) +
		len(c.ownersAuthorizers) +
		len(c.commercialRegister) +
		len(c.tsps) +
		len(c.authorizedActions)

	if size > c.maxEntries {
		// Clear 10% of entries (simple approach)
		// Real implementation would track access times and evict LRU entries
	}
}
