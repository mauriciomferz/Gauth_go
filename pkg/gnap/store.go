package gnap

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
)

// GrantStore manages grant lifecycle and persistence.
type GrantStore interface {
	// Create stores a new grant request
	Create(req *GrantRequest) (*Grant, error)

	// Get retrieves a grant by ID
	Get(id string) (*Grant, error)

	// Update modifies an existing grant
	Update(g *Grant) error

	// Delete removes a grant
	Delete(id string) error

	// ListByClient returns grants for a client instance
	ListByClient(instanceID string) ([]*Grant, error)
}

// ResourceServerStore manages RS lifecycles (RFC 9767).
type ResourceServerStore interface {
	// Register stores a new RS registration
	Register(rs *ResourceServerRequest) (*ResourceServerResponse, error)

	// Get retrieves an RS by instance ID
	Get(instanceID string) (*ResourceServerRequest, error)
}

// Grant represents a grant in progress or completed.
type Grant struct {
	// ID is the unique grant identifier
	ID string `json:"grant_id"`

	// State is the current grant state
	State GrantState `json:"state"`

	// Request is the original grant request
	Request *GrantRequest `json:"request"`

	// Response is the current response (updated as grant progresses)
	Response *GrantResponse `json:"response,omitempty"`

	// ContinueToken for continuation requests
	ContinueToken string `json:"continue_token,omitempty"`

	// ContinueURI for continuation endpoint
	ContinueURI string `json:"continue_uri,omitempty"`

	// InteractRef from interaction callback
	InteractRef string `json:"interact_ref,omitempty"`

	// InteractNonce for hash verification
	InteractNonce string `json:"interact_nonce,omitempty"`

	// ClientInstanceID assigned to client
	ClientInstanceID string `json:"client_instance_id,omitempty"`

	// CreatedAt timestamp
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt timestamp
	UpdatedAt time.Time `json:"updated_at"`

	// ExpiresAt for grant timeout
	ExpiresAt time.Time `json:"expires_at,omitempty"`

	// --- AgentAuth Extensions ---

	// PoAID if linked to Power of Attorney
	PoAID string `json:"poa_id,omitempty"`

	// SubscriptionID if linked to AgentAuth subscription
	SubscriptionID string `json:"subscription_id,omitempty"`
}

// MemoryGrantStore implements GrantStore in-memory.
type MemoryGrantStore struct {
	mu       sync.RWMutex
	grants   map[string]*Grant
	byClient map[string][]string // instance_id -> grant_ids
}

// NewMemoryGrantStore creates an in-memory grant store.
func NewMemoryGrantStore() *MemoryGrantStore {
	return &MemoryGrantStore{
		grants:   make(map[string]*Grant),
		byClient: make(map[string][]string),
	}
}

// Create stores a new grant request.
func (s *MemoryGrantStore) Create(req *GrantRequest) (*Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := generateID("gnt_")
	now := time.Now().UTC()

	g := &Grant{
		ID:        id,
		State:     GrantStateProcessing,
		Request:   req,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(10 * time.Minute), // Default TTL
	}

	// Generate continuation token
	g.ContinueToken = generateID("cnt_")

	// Link to client if available
	if req.Client != nil && req.Client.InstanceID != "" {
		g.ClientInstanceID = req.Client.InstanceID
		s.byClient[g.ClientInstanceID] = append(s.byClient[g.ClientInstanceID], g.ID)
	}

	// Link to AgentAuth subscription if present
	if req.SubscriptionID != "" {
		g.SubscriptionID = req.SubscriptionID
	}

	s.grants[id] = g
	return g, nil
}

// Get retrieves a grant by ID.
func (s *MemoryGrantStore) Get(id string) (*Grant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	g, ok := s.grants[id]
	if !ok {
		return nil, errors.New("grant not found")
	}

	// Check expiration
	if !g.ExpiresAt.IsZero() && time.Now().After(g.ExpiresAt) {
		return nil, errors.New("grant expired")
	}

	return g, nil
}

// Update modifies an existing grant.
func (s *MemoryGrantStore) Update(g *Grant) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.grants[g.ID]; !ok {
		return errors.New("grant not found")
	}

	g.UpdatedAt = time.Now().UTC()
	s.grants[g.ID] = g
	return nil
}

// Delete removes a grant.
func (s *MemoryGrantStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	g, ok := s.grants[id]
	if !ok {
		return errors.New("grant not found")
	}

	// Remove from client index
	if g.ClientInstanceID != "" {
		ids := s.byClient[g.ClientInstanceID]
		for i, gid := range ids {
			if gid == id {
				s.byClient[g.ClientInstanceID] = append(ids[:i], ids[i+1:]...)
				break
			}
		}
	}

	delete(s.grants, id)
	return nil
}

// ListByClient returns grants for a client instance.
func (s *MemoryGrantStore) ListByClient(instanceID string) ([]*Grant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.byClient[instanceID]
	result := make([]*Grant, 0, len(ids))
	for _, id := range ids {
		if g, ok := s.grants[id]; ok {
			result = append(result, g)
		}
	}
	return result, nil
}

// Cleanup removes expired grants and returns the count of removed grants.
// This should be called periodically to prevent memory leaks.
func (s *MemoryGrantStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0
	toDelete := []string{}

	// Find expired grants
	for id, g := range s.grants {
		if !g.ExpiresAt.IsZero() && now.After(g.ExpiresAt) {
			toDelete = append(toDelete, id)
		}
	}

	// Delete expired grants and update client index
	for _, id := range toDelete {
		g := s.grants[id]

		// Remove from client index
		if g.ClientInstanceID != "" {
			ids := s.byClient[g.ClientInstanceID]
			for i, gid := range ids {
				if gid == id {
					s.byClient[g.ClientInstanceID] = append(ids[:i], ids[i+1:]...)
					break
				}
			}
			// Clean up empty client entries
			if len(s.byClient[g.ClientInstanceID]) == 0 {
				delete(s.byClient, g.ClientInstanceID)
			}
		}

		delete(s.grants, id)
		removed++
	}

	return removed
}

// GetByContinueToken finds grant by continuation token.
func (s *MemoryGrantStore) GetByContinueToken(token string) (*Grant, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, g := range s.grants {
		if g.ContinueToken == token {
			if !g.ExpiresAt.IsZero() && time.Now().After(g.ExpiresAt) {
				return nil, errors.New("grant expired")
			}
			return g, nil
		}
	}
	return nil, errors.New("grant not found")
}

// GenerateID creates a random prefixed ID.
func GenerateID(prefix string) string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

// generateID is an alias for internal use.
var generateID = GenerateID

// --- State Transitions ---

// Transition validates and applies a state transition.
func (g *Grant) Transition(newState GrantState) error {
	valid := false
	switch g.State {
	case GrantStateProcessing:
		valid = newState == GrantStatePending || newState == GrantStateApproved || newState == GrantStateDenied
	case GrantStatePending:
		valid = newState == GrantStateApproved || newState == GrantStateDenied
	case GrantStateApproved:
		valid = newState == GrantStateFinalized
	case GrantStateFinalized, GrantStateDenied:
		// Terminal states
		valid = false
	}

	if !valid {
		return errors.New("invalid state transition")
	}

	g.State = newState
	g.UpdatedAt = time.Now().UTC()
	return nil
}

// IsTerminal returns true if grant is in a terminal state.
func (g *Grant) IsTerminal() bool {
	return g.State == GrantStateFinalized || g.State == GrantStateDenied
}

// CanContinue returns true if grant accepts continuation requests.
func (g *Grant) CanContinue() bool {
	return g.State == GrantStateProcessing || g.State == GrantStatePending || g.State == GrantStateApproved
}

// MemoryResourceServerStore implements ResourceServerStore in-memory.
type MemoryResourceServerStore struct {
	mu sync.RWMutex
	rs map[string]*ResourceServerRequest
}

// NewMemoryResourceServerStore creates a new in-memory RS store.
func NewMemoryResourceServerStore() *MemoryResourceServerStore {
	return &MemoryResourceServerStore{
		rs: make(map[string]*ResourceServerRequest),
	}
}

// Register stores a new RS registration.
func (s *MemoryResourceServerStore) Register(req *ResourceServerRequest) (*ResourceServerResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	instanceID := generateID("rs_")
	// Copy request to store
	s.rs[instanceID] = req

	// In a real implementation we would sign/issue a key
	// For now, return what was requested or generate a new one
	key := req.Client.Key
	if key == nil {
		key = &ClientKey{Proof: ProofHTTPSig}
	}

	return &ResourceServerResponse{
		InstanceID: instanceID,
		Key:        key,
	}, nil
}

// Get retrieves an RS by instance ID.
func (s *MemoryResourceServerStore) Get(instanceID string) (*ResourceServerRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rs, ok := s.rs[instanceID]
	if !ok {
		return nil, errors.New("resource server not found")
	}
	return rs, nil
}
