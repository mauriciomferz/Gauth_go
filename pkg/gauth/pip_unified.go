// Package gauth provides a unified Power Information Point (PIP) interface
// RFC-0111 Section 4 - Power Information Point implementation
package gauth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mauriciomferz/Gauth_go/pkg/poa"
)

// PIP defines the unified Power Information Point interface
// Centralizes attribute and information retrieval across the P*P architecture
type PIP interface {
	// Attribute Management
	GetAttribute(ctx context.Context, attrName string, subject string) (interface{}, error)
	GetAttributes(ctx context.Context, attrNames []string, subject string) (map[string]interface{}, error)
	SetAttribute(ctx context.Context, attrName string, subject string, value interface{}) error
	DeleteAttribute(ctx context.Context, attrName string, subject string) error

	// Entity Information Retrieval
	GetClientInfo(ctx context.Context, clientID string) (*ClientInfo, error)
	GetClientOwnerInfo(ctx context.Context, ownerID string) (*ClientOwnerInfo, error)
	GetOwnersAuthorizerInfo(ctx context.Context, authorizerID string) (*OwnersAuthorizerInfo, error)
	GetAuthorizationServerInfo(ctx context.Context, serverID string) (*AuthorizationServerInfo, error)
	GetResourceOwnerInfo(ctx context.Context, ownerID string) (*ResourceOwnerInfo, error)

	// Power of Attorney Queries
	GetPoAByID(ctx context.Context, poaID string) (*poa.PoADefinition, error)
	GetPoAsByClient(ctx context.Context, clientID string) ([]*poa.PoADefinition, error)
	GetPoAsByOwner(ctx context.Context, ownerID string) ([]*poa.PoADefinition, error)
	GetActivePoAs(ctx context.Context, clientID string) ([]*poa.PoADefinition, error)

	// Commercial Register Queries
	GetCommercialRegisterEntry(ctx context.Context, entityID string, jurisdiction string) (*RegisterEntry, error)
	GetManagingDirectorInfo(ctx context.Context, companyID string, directorID string) (*DirectorInfo, error)

	// Authorization Chain Queries
	GetAuthorizationChain(ctx context.Context, clientID string) (*AuthorizationChain, error)
	ValidateChainMembership(ctx context.Context, entityID string, chainID string) (bool, error)

	// Attribute Store Status
	GetStatus(ctx context.Context) (*PIPStatus, error)
	ClearCache(ctx context.Context) error
}

// UnifiedPIP provides a unified implementation of the PIP interface
// Aggregates data from multiple sources with caching
type UnifiedPIP struct {
	mu sync.RWMutex

	// Attribute storage
	attributes map[string]map[string]interface{} // subject -> attrName -> value

	// Entity stores
	clients              map[string]*ClientInfo
	clientOwners         map[string]*ClientOwnerInfo
	ownersAuthorizers    map[string]*OwnersAuthorizerInfo
	authorizationServers map[string]*AuthorizationServerInfo
	resourceOwners       map[string]*ResourceOwnerInfo

	// PoA store
	poas         map[string]*poa.PoADefinition
	poasByClient map[string][]string // clientID -> []poaID
	poasByOwner  map[string][]string // ownerID -> []poaID

	// Register store
	registerEntries map[string]*RegisterEntry
	directorInfo    map[string]*DirectorInfo

	// Authorization chains
	authChains map[string]*AuthorizationChain

	// External data sources
	commercialRegister CommercialRegisterClient
	trustProvider      TrustServiceProvider

	// Cache configuration
	cacheEnabled bool
	cacheTTL     time.Duration
	cacheExpiry  map[string]time.Time

	// Statistics
	stats PIPStats
}

// PIPStats contains PIP usage statistics
type PIPStats struct {
	TotalQueries     int64
	CacheHits        int64
	CacheMisses      int64
	ExternalCalls    int64
	AverageLatencyMs float64
	LastQueryTime    time.Time
}

// PIPStatus contains PIP status information
type PIPStatus struct {
	Operational    bool
	AttributeCount int
	EntityCount    int
	PoACount       int
	CacheEnabled   bool
	CacheHitRatio  float64
	Stats          PIPStats
	LastUpdated    time.Time
}

// Note: Using types from external_integrations.go:
// - DirectorInfo (already defined)
// - CompanyInfo (for register entries)
// Define only what's unique to PIP

// RegisterEntry is an alias for CompanyInfo from commercial register
type RegisterEntry = CompanyInfo

// NewUnifiedPIP creates a new unified PIP instance
func NewUnifiedPIP(
	commercialRegister CommercialRegisterClient,
	trustProvider TrustServiceProvider,
	cacheEnabled bool,
	cacheTTL time.Duration,
) *UnifiedPIP {
	return &UnifiedPIP{
		attributes:           make(map[string]map[string]interface{}),
		clients:              make(map[string]*ClientInfo),
		clientOwners:         make(map[string]*ClientOwnerInfo),
		ownersAuthorizers:    make(map[string]*OwnersAuthorizerInfo),
		authorizationServers: make(map[string]*AuthorizationServerInfo),
		resourceOwners:       make(map[string]*ResourceOwnerInfo),
		poas:                 make(map[string]*poa.PoADefinition),
		poasByClient:         make(map[string][]string),
		poasByOwner:          make(map[string][]string),
		registerEntries:      make(map[string]*RegisterEntry),
		directorInfo:         make(map[string]*DirectorInfo),
		authChains:           make(map[string]*AuthorizationChain),
		commercialRegister:   commercialRegister,
		trustProvider:        trustProvider,
		cacheEnabled:         cacheEnabled,
		cacheTTL:             cacheTTL,
		cacheExpiry:          make(map[string]time.Time),
		stats:                PIPStats{},
	}
}

// GetAttribute retrieves a single attribute for a subject
func (p *UnifiedPIP) GetAttribute(ctx context.Context, attrName string, subject string) (interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++
	p.stats.LastQueryTime = time.Now()

	if subjectAttrs, ok := p.attributes[subject]; ok {
		if value, ok := subjectAttrs[attrName]; ok {
			// Check cache expiry
			if p.cacheEnabled {
				key := fmt.Sprintf("%s:%s", subject, attrName)
				if expiry, ok := p.cacheExpiry[key]; ok && time.Now().After(expiry) {
					// Cache expired
					p.stats.CacheMisses++
					return nil, fmt.Errorf("attribute cache expired for %s:%s", subject, attrName)
				}
			}
			p.stats.CacheHits++
			return value, nil
		}
	}

	p.stats.CacheMisses++
	return nil, fmt.Errorf("attribute %s not found for subject %s", attrName, subject)
}

// GetAttributes retrieves multiple attributes for a subject
func (p *UnifiedPIP) GetAttributes(ctx context.Context, attrNames []string, subject string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for _, attrName := range attrNames {
		value, err := p.GetAttribute(ctx, attrName, subject)
		if err == nil {
			result[attrName] = value
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no attributes found for subject %s", subject)
	}

	return result, nil
}

// SetAttribute sets an attribute value for a subject
func (p *UnifiedPIP) SetAttribute(ctx context.Context, attrName string, subject string, value interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.attributes[subject] == nil {
		p.attributes[subject] = make(map[string]interface{})
	}

	p.attributes[subject][attrName] = value

	// Set cache expiry
	if p.cacheEnabled {
		key := fmt.Sprintf("%s:%s", subject, attrName)
		p.cacheExpiry[key] = time.Now().Add(p.cacheTTL)
	}

	return nil
}

// DeleteAttribute deletes an attribute for a subject
func (p *UnifiedPIP) DeleteAttribute(ctx context.Context, attrName string, subject string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if subjectAttrs, ok := p.attributes[subject]; ok {
		delete(subjectAttrs, attrName)

		// Remove cache expiry
		key := fmt.Sprintf("%s:%s", subject, attrName)
		delete(p.cacheExpiry, key)
	}

	return nil
}

// GetClientInfo retrieves client information
func (p *UnifiedPIP) GetClientInfo(ctx context.Context, clientID string) (*ClientInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++

	if client, ok := p.clients[clientID]; ok {
		p.stats.CacheHits++
		return client, nil
	}

	p.stats.CacheMisses++
	return nil, fmt.Errorf("client %s not found", clientID)
}

// GetClientOwnerInfo retrieves client owner information
func (p *UnifiedPIP) GetClientOwnerInfo(ctx context.Context, ownerID string) (*ClientOwnerInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++

	if owner, ok := p.clientOwners[ownerID]; ok {
		p.stats.CacheHits++
		return owner, nil
	}

	p.stats.CacheMisses++
	return nil, fmt.Errorf("client owner %s not found", ownerID)
}

// GetOwnersAuthorizerInfo retrieves owner's authorizer information
func (p *UnifiedPIP) GetOwnersAuthorizerInfo(ctx context.Context, authorizerID string) (*OwnersAuthorizerInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++

	if authorizer, ok := p.ownersAuthorizers[authorizerID]; ok {
		p.stats.CacheHits++
		return authorizer, nil
	}

	p.stats.CacheMisses++
	return nil, fmt.Errorf("owner's authorizer %s not found", authorizerID)
}

// GetAuthorizationServerInfo retrieves authorization server information
func (p *UnifiedPIP) GetAuthorizationServerInfo(ctx context.Context, serverID string) (*AuthorizationServerInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++

	if server, ok := p.authorizationServers[serverID]; ok {
		p.stats.CacheHits++
		return server, nil
	}

	p.stats.CacheMisses++
	return nil, fmt.Errorf("authorization server %s not found", serverID)
}

// GetResourceOwnerInfo retrieves resource owner information
func (p *UnifiedPIP) GetResourceOwnerInfo(ctx context.Context, ownerID string) (*ResourceOwnerInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++

	if owner, ok := p.resourceOwners[ownerID]; ok {
		p.stats.CacheHits++
		return owner, nil
	}

	p.stats.CacheMisses++
	return nil, fmt.Errorf("resource owner %s not found", ownerID)
}

// GetPoAByID retrieves a Power of Attorney by ID
func (p *UnifiedPIP) GetPoAByID(ctx context.Context, poaID string) (*poa.PoADefinition, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++

	if poaDef, ok := p.poas[poaID]; ok {
		p.stats.CacheHits++
		return poaDef, nil
	}

	p.stats.CacheMisses++
	return nil, fmt.Errorf("PoA %s not found", poaID)
}

// GetPoAsByClient retrieves all PoAs for a client
func (p *UnifiedPIP) GetPoAsByClient(ctx context.Context, clientID string) ([]*poa.PoADefinition, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++

	poaIDs, ok := p.poasByClient[clientID]
	if !ok {
		return nil, fmt.Errorf("no PoAs found for client %s", clientID)
	}

	result := make([]*poa.PoADefinition, 0, len(poaIDs))
	for _, poaID := range poaIDs {
		if poaDef, ok := p.poas[poaID]; ok {
			result = append(result, poaDef)
		}
	}

	return result, nil
}

// GetPoAsByOwner retrieves all PoAs for an owner
func (p *UnifiedPIP) GetPoAsByOwner(ctx context.Context, ownerID string) ([]*poa.PoADefinition, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++

	poaIDs, ok := p.poasByOwner[ownerID]
	if !ok {
		return nil, fmt.Errorf("no PoAs found for owner %s", ownerID)
	}

	result := make([]*poa.PoADefinition, 0, len(poaIDs))
	for _, poaID := range poaIDs {
		if poaDef, ok := p.poas[poaID]; ok {
			result = append(result, poaDef)
		}
	}

	return result, nil
}

// GetActivePoAs retrieves all active PoAs for a client
func (p *UnifiedPIP) GetActivePoAs(ctx context.Context, clientID string) ([]*poa.PoADefinition, error) {
	allPoAs, err := p.GetPoAsByClient(ctx, clientID)
	if err != nil {
		return nil, err
	}

	result := make([]*poa.PoADefinition, 0)
	now := time.Now()

	for _, poaDef := range allPoAs {
		// Check if PoA is active
		if poaDef.Parties.AuthorizedClient.StatusEnum == poa.OperationalStatusActive {
			// Check validity period
			validityPeriod := poaDef.Requirements.ValidityPeriod
			if !validityPeriod.StartTime.IsZero() && now.Before(validityPeriod.StartTime) {
				continue // Not yet valid
			}
			if !validityPeriod.EndTime.IsZero() && now.After(validityPeriod.EndTime) {
				continue // Expired
			}
			result = append(result, poaDef)
		}
	}

	return result, nil
}

// GetCommercialRegisterEntry retrieves commercial register entry
func (p *UnifiedPIP) GetCommercialRegisterEntry(ctx context.Context, entityID string, jurisdiction string) (*RegisterEntry, error) {
	p.mu.RLock()

	// Check cache first
	key := fmt.Sprintf("%s:%s", jurisdiction, entityID)
	if entry, ok := p.registerEntries[key]; ok {
		p.mu.RUnlock()
		p.stats.CacheHits++
		return entry, nil
	}
	p.mu.RUnlock()

	// Query external commercial register
	if p.commercialRegister != nil {
		p.stats.ExternalCalls++
		companyInfo, err := p.commercialRegister.VerifyCompany(ctx, jurisdiction, entityID)
		if err != nil {
			return nil, fmt.Errorf("failed to query commercial register: %w", err)
		}

		// Cache the result (CompanyInfo = RegisterEntry via type alias)
		p.mu.Lock()
		p.registerEntries[key] = companyInfo
		p.mu.Unlock()

		return companyInfo, nil
	}

	return nil, fmt.Errorf("commercial register entry %s not found and no external client configured", entityID)
}

// GetManagingDirectorInfo retrieves managing director information
func (p *UnifiedPIP) GetManagingDirectorInfo(ctx context.Context, companyID string, directorID string) (*DirectorInfo, error) {
	p.mu.RLock()

	// Check cache first
	key := fmt.Sprintf("%s:%s", companyID, directorID)
	if info, ok := p.directorInfo[key]; ok {
		p.mu.RUnlock()
		p.stats.CacheHits++
		return info, nil
	}
	p.mu.RUnlock()

	// Query external commercial register
	if p.commercialRegister != nil {
		p.stats.ExternalCalls++
		dirInfo, err := p.commercialRegister.VerifyManagingDirector(ctx, companyID, directorID)
		if err != nil {
			return nil, fmt.Errorf("failed to query managing director info: %w", err)
		}

		// Cache the result directly (DirectorInfo from external_integrations.go)
		p.mu.Lock()
		p.directorInfo[key] = dirInfo
		p.mu.Unlock()

		return dirInfo, nil
	}

	return nil, fmt.Errorf("director info not found and no external client configured")
}

// GetAuthorizationChain retrieves authorization chain for a client
func (p *UnifiedPIP) GetAuthorizationChain(ctx context.Context, clientID string) (*AuthorizationChain, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	p.stats.TotalQueries++

	if chain, ok := p.authChains[clientID]; ok {
		p.stats.CacheHits++
		return chain, nil
	}

	p.stats.CacheMisses++
	return nil, fmt.Errorf("authorization chain for client %s not found", clientID)
}

// ValidateChainMembership validates if an entity is part of an authorization chain
func (p *UnifiedPIP) ValidateChainMembership(ctx context.Context, entityID string, chainIntegrity string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, chain := range p.authChains {
		if chain.ChainIntegrity == chainIntegrity {
			// Check if entity is in the chain
			if chain.OwnersAuthorizer != nil && chain.OwnersAuthorizer.EntityID == entityID {
				return true, nil
			}
			if chain.ClientOwner != nil && chain.ClientOwner.EntityID == entityID {
				return true, nil
			}
			if chain.Client != nil && chain.Client.EntityID == entityID {
				return true, nil
			}
		}
	}

	return false, nil
}

// GetStatus returns PIP status information
func (p *UnifiedPIP) GetStatus(ctx context.Context) (*PIPStatus, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	attributeCount := 0
	for _, attrs := range p.attributes {
		attributeCount += len(attrs)
	}

	entityCount := len(p.clients) + len(p.clientOwners) + len(p.ownersAuthorizers) +
		len(p.authorizationServers) + len(p.resourceOwners)

	cacheHitRatio := 0.0
	totalCacheOps := p.stats.CacheHits + p.stats.CacheMisses
	if totalCacheOps > 0 {
		cacheHitRatio = float64(p.stats.CacheHits) / float64(totalCacheOps)
	}

	return &PIPStatus{
		Operational:    true,
		AttributeCount: attributeCount,
		EntityCount:    entityCount,
		PoACount:       len(p.poas),
		CacheEnabled:   p.cacheEnabled,
		CacheHitRatio:  cacheHitRatio,
		Stats:          p.stats,
		LastUpdated:    time.Now(),
	}, nil
}

// ClearCache clears the PIP cache
func (p *UnifiedPIP) ClearCache(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.cacheExpiry = make(map[string]time.Time)

	// Reset stats
	p.stats.CacheHits = 0
	p.stats.CacheMisses = 0

	return nil
}

// RegisterClient registers a client in the PIP
func (p *UnifiedPIP) RegisterClient(clientInfo *ClientInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.clients[clientInfo.ClientID] = clientInfo
	return nil
}

// RegisterClientOwner registers a client owner in the PIP
func (p *UnifiedPIP) RegisterClientOwner(ownerInfo *ClientOwnerInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.clientOwners[ownerInfo.OwnerID] = ownerInfo
	return nil
}

// RegisterOwnersAuthorizer registers an owner's authorizer in the PIP
func (p *UnifiedPIP) RegisterOwnersAuthorizer(authorizerInfo *OwnersAuthorizerInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.ownersAuthorizers[authorizerInfo.AuthorizerID] = authorizerInfo
	return nil
}

// RegisterPoA registers a Power of Attorney in the PIP
func (p *UnifiedPIP) RegisterPoA(poaDef *poa.PoADefinition, poaID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.poas[poaID] = poaDef

	// Index by client
	clientID := poaDef.Parties.AuthorizedClient.Identity
	p.poasByClient[clientID] = append(p.poasByClient[clientID], poaID)

	// Index by owner (principal)
	ownerID := poaDef.Parties.Principal.Identity
	p.poasByOwner[ownerID] = append(p.poasByOwner[ownerID], poaID)

	return nil
}

// RegisterAuthorizationChain registers an authorization chain in the PIP
func (p *UnifiedPIP) RegisterAuthorizationChain(chain *AuthorizationChain) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Use ChainIntegrity as key
	key := chain.ChainIntegrity
	if key == "" {
		return fmt.Errorf("authorization chain must have integrity hash")
	}
	p.authChains[key] = chain

	return nil
}

// RegisterAuthorizationServer registers an authorization server in the PIP
func (p *UnifiedPIP) RegisterAuthorizationServer(serverInfo *AuthorizationServerInfo) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.authorizationServers[serverInfo.ServerID] = serverInfo
	return nil
}
