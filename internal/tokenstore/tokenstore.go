package tokenstore

import (
	"context"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/token"
)

// Re-export types from token package
type (
	Token       = token.Token
	MemoryStore = token.MemoryStore
	Store       = token.Store
	Filter      = token.Filter
	TokenType   = token.TokenType
)

// Re-export constants
const (
	Access  = token.Access
	Refresh = token.Refresh
	ID      = token.ID
)

// Re-export functions
var (
	NewMemoryStore = token.NewMemoryStore
)

// InternalStore provides additional internal functionality
type InternalStore struct {
	*token.MemoryStore
	metrics map[string]int64
}

// NewInternalStore creates a new internal store with metrics
func NewInternalStore() *InternalStore {
	return &InternalStore{
		MemoryStore: token.NewMemoryStore(),
		metrics:     make(map[string]int64),
	}
}

// SaveWithMetrics saves a token and updates metrics
func (s *InternalStore) SaveWithMetrics(ctx context.Context, key string, tok *token.Token) error {
	err := s.MemoryStore.Save(ctx, key, tok)
	if err == nil {
		s.metrics["saves"]++
	}
	return err
}

// GetWithMetrics gets a token and updates metrics
func (s *InternalStore) GetWithMetrics(ctx context.Context, tokenID string) (*token.Token, error) {
	tok, err := s.MemoryStore.Get(ctx, tokenID)
	if err == nil && tok != nil {
		s.metrics["gets"]++
	}
	return tok, err
}

// GetMetrics returns internal metrics
func (s *InternalStore) GetMetrics() map[string]int64 {
	return s.metrics
}

// Demo demonstrates internal tokenstore functionality
func Demo() error {
	store := NewInternalStore()
	ctx := context.Background()

	// Create test token
	testToken := &token.Token{
		ID:        "internal-test-token",
		Value:     "test-value",
		Type:      Access,
		Subject:   "test-user",
		Scopes:    []string{"read", "write"},
		ExpiresAt: time.Now().Add(time.Hour),
	}

	// Save with metrics
	if err := store.SaveWithMetrics(ctx, testToken.ID, testToken); err != nil {
		return err
	}

	// Get with metrics
	_, err := store.GetWithMetrics(ctx, testToken.ID)
	if err != nil {
		return err
	}

	// Check metrics
	metrics := store.GetMetrics()
	if saves := metrics["saves"]; saves != 1 {
		return nil // Simple validation
	}

	return nil
}
