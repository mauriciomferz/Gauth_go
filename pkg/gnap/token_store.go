package gnap

import (
	"errors"
	"sync"
	"time"
)

// TokenStore manages issued access tokens and their lifecycle.
type TokenStore interface {
	// Store saves a new token
	Store(token *IssuedToken) error

	// Get retrieves a token by value
	Get(value string) (*IssuedToken, error)

	// Rotate replaces a token and returns the new one
	Rotate(value string) (*IssuedToken, error)

	// Revoke invalidates a token
	Revoke(value string) error

	// ListByGrant returns all tokens for a grant
	ListByGrant(grantID string) ([]*IssuedToken, error)
}

// IssuedToken represents an access token in storage.
type IssuedToken struct {
	Value       string        `json:"value"`
	GrantID     string        `json:"grant_id"`
	Access      []AccessRight `json:"access"`
	IssuedAt    time.Time     `json:"issued_at"`
	ExpiresAt   time.Time     `json:"expires_at"`
	RotatedFrom string        `json:"rotated_from,omitempty"` // Previous token value
	Revoked     bool          `json:"revoked"`
	RevokedAt   time.Time     `json:"revoked_at,omitempty"`
	Flags       []TokenFlag   `json:"flags,omitempty"`

	// AgentAuth extensions
	PoAID string `json:"poa_id,omitempty"`
}

// MemoryTokenStore implements TokenStore in-memory.
type MemoryTokenStore struct {
	mu      sync.RWMutex
	tokens  map[string]*IssuedToken
	byGrant map[string][]string // grant_id -> token values
}

// NewMemoryTokenStore creates an in-memory token store.
func NewMemoryTokenStore() *MemoryTokenStore {
	return &MemoryTokenStore{
		tokens:  make(map[string]*IssuedToken),
		byGrant: make(map[string][]string),
	}
}

// Store saves a new token.
func (s *MemoryTokenStore) Store(token *IssuedToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token.Value == "" {
		return errors.New("token value required")
	}

	s.tokens[token.Value] = token

	if token.GrantID != "" {
		s.byGrant[token.GrantID] = append(s.byGrant[token.GrantID], token.Value)
	}

	return nil
}

// Get retrieves a token by value.
func (s *MemoryTokenStore) Get(value string) (*IssuedToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	token, ok := s.tokens[value]
	if !ok {
		return nil, errors.New("token not found")
	}

	if token.Revoked {
		return nil, errors.New("token revoked")
	}

	if !token.ExpiresAt.IsZero() && time.Now().After(token.ExpiresAt) {
		return nil, errors.New("token expired")
	}

	return token, nil
}

// Rotate replaces a token and returns the new one.
func (s *MemoryTokenStore) Rotate(value string) (*IssuedToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldToken, ok := s.tokens[value]
	if !ok {
		return nil, errors.New("token not found")
	}

	if oldToken.Revoked {
		return nil, errors.New("cannot rotate revoked token")
	}

	// Check if durable flag allows rotation
	isDurable := false
	for _, f := range oldToken.Flags {
		if f == TokenFlagDurable {
			isDurable = true
			break
		}
	}

	// Create new token
	newToken := &IssuedToken{
		Value:       GenerateID("gauth_gnap_"),
		GrantID:     oldToken.GrantID,
		Access:      oldToken.Access,
		IssuedAt:    time.Now().UTC(),
		ExpiresAt:   time.Now().Add(time.Hour).UTC(),
		RotatedFrom: oldToken.Value,
		Flags:       oldToken.Flags,
		PoAID:       oldToken.PoAID,
	}

	// Revoke old token unless durable
	if !isDurable {
		oldToken.Revoked = true
		oldToken.RevokedAt = time.Now().UTC()
	}

	// Store new token
	s.tokens[newToken.Value] = newToken
	if newToken.GrantID != "" {
		s.byGrant[newToken.GrantID] = append(s.byGrant[newToken.GrantID], newToken.Value)
	}

	return newToken, nil
}

// Revoke invalidates a token.
func (s *MemoryTokenStore) Revoke(value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	token, ok := s.tokens[value]
	if !ok {
		return errors.New("token not found")
	}

	token.Revoked = true
	token.RevokedAt = time.Now().UTC()
	return nil
}

// ListByGrant returns all tokens for a grant.
func (s *MemoryTokenStore) ListByGrant(grantID string) ([]*IssuedToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	values := s.byGrant[grantID]
	result := make([]*IssuedToken, 0, len(values))
	for _, v := range values {
		if t, ok := s.tokens[v]; ok {
			result = append(result, t)
		}
	}
	return result, nil
}

// Cleanup removes expired and revoked tokens and returns the count of removed tokens.
// This should be called periodically to prevent memory leaks.
func (s *MemoryTokenStore) Cleanup() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	removed := 0
	toDelete := []string{}

	// Find expired or revoked tokens
	for value, token := range s.tokens {
		shouldDelete := false

		// Remove if revoked (and past grace period)
		if token.Revoked && !token.RevokedAt.IsZero() && now.Sub(token.RevokedAt) > 24*time.Hour {
			shouldDelete = true
		}

		// Remove if expired (and past grace period)
		if !token.ExpiresAt.IsZero() && now.Sub(token.ExpiresAt) > 1*time.Hour {
			shouldDelete = true
		}

		if shouldDelete {
			toDelete = append(toDelete, value)
		}
	}

	// Delete tokens and update grant index
	for _, value := range toDelete {
		token := s.tokens[value]

		// Remove from grant index
		if token.GrantID != "" {
			values := s.byGrant[token.GrantID]
			for i, v := range values {
				if v == value {
					s.byGrant[token.GrantID] = append(values[:i], values[i+1:]...)
					break
				}
			}
			// Clean up empty grant entries
			if len(s.byGrant[token.GrantID]) == 0 {
				delete(s.byGrant, token.GrantID)
			}
		}

		delete(s.tokens, value)
		removed++
	}

	return removed
}
