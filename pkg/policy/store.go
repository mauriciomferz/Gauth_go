package policy

import (
	"context"
)

// Store provides an abstraction over policy bundle persistence / retrieval.
// Initial implementation is in-memory and backed by the existing Registry.
// Future durable stores (file, DB) can implement this without changing higher layers.
type Store interface {
	// AppendBundle adds a new bundle (hash computed internally) and returns the stored bundle.
	AppendBundle(ctx context.Context, b Bundle) (Bundle, error)
	// Head returns the most recent bundle (nil if empty).
	Head(ctx context.Context) (*Bundle, error)
	// GetByHash retrieves a bundle by its hash (nil if not found).
	GetByHash(ctx context.Context, hash string) (*Bundle, error)
	// GetByVersion retrieves a bundle by its version (nil if not found).
	GetByVersion(ctx context.Context, version int) (*Bundle, error)
	// List returns a slice of bundles with pagination (offset, limit) and total length.
	// List returns a slice of bundles with pagination (offset, limit) and total length.
	List(ctx context.Context, offset, limit int) ([]Bundle, int, error)
	// ChainHashes returns ordered list of bundle hashes.
	ChainHashes(ctx context.Context) ([]string, error)
	// VerifyChain re-hashes and verifies linkage.
	VerifyChain(ctx context.Context) error
	// ActiveVersion returns the currently active version.
	ActiveVersion(ctx context.Context) (int, error)
	// Rollback sets the active version to the specified version.
	Rollback(ctx context.Context, version int) error
	// Registry exposes underlying registry for legacy engine integration.
	Registry() *Registry
}

// InMemoryStore is a Store backed by the existing in-memory Registry.
type InMemoryStore struct {
	reg *Registry
}

// NewInMemoryStore constructs a new empty in-memory store.
func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{reg: NewRegistry()} }

func (s *InMemoryStore) AppendBundle(ctx context.Context, b Bundle) (Bundle, error) {
	return s.reg.AddBundle(b)
}
func (s *InMemoryStore) Head(ctx context.Context) (*Bundle, error) {
	return s.reg.Head(), nil
}
func (s *InMemoryStore) GetByHash(ctx context.Context, hash string) (*Bundle, error) {
	return s.reg.FindByHash(hash), nil
}
func (s *InMemoryStore) GetByVersion(ctx context.Context, version int) (*Bundle, error) {
	return s.reg.findByVersion(version), nil
}
func (s *InMemoryStore) ChainHashes(ctx context.Context) ([]string, error) {
	return s.reg.ChainHashes(), nil
}
func (s *InMemoryStore) VerifyChain(ctx context.Context) error {
	return s.reg.VerifyChain()
}
func (s *InMemoryStore) ActiveVersion(ctx context.Context) (int, error) {
	return s.reg.ActiveVersion(), nil
}
func (s *InMemoryStore) Rollback(ctx context.Context, version int) error {
	return s.reg.Rollback(version)
}
func (s *InMemoryStore) Registry() *Registry { return s.reg }

// List implements naive pagination.
func (s *InMemoryStore) List(ctx context.Context, offset, limit int) ([]Bundle, int, error) {
	if offset < 0 {
		offset = 0
	}
	total := len(s.reg.bundles)
	if offset >= total {
		return []Bundle{}, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	// Return shallow copies to prevent external mutation of internal slice entries.
	out := make([]Bundle, end-offset)
	copy(out, s.reg.bundles[offset:end])
	return out, total, nil
}
