package policy

// Store provides an abstraction over policy bundle persistence / retrieval.
// Initial implementation is in-memory and backed by the existing Registry.
// Future durable stores (file, DB) can implement this without changing higher layers.
type Store interface {
	// AppendBundle adds a new bundle (hash computed internally) and returns the stored bundle.
	AppendBundle(Bundle) (Bundle, error)
	// Head returns the most recent bundle (nil if empty).
	Head() *Bundle
	// GetByHash retrieves a bundle by its hash (nil if not found).
	GetByHash(hash string) *Bundle
	// List returns a slice of bundles with pagination (offset, limit) and total length.
	List(offset, limit int) ([]Bundle, int)
	// ChainHashes returns ordered list of bundle hashes.
	ChainHashes() []string
	// VerifyChain re-hashes and verifies linkage.
	VerifyChain() error
	// Registry exposes underlying registry for legacy engine integration (temporary; will be removed once engine shifts to Store).
	Registry() *Registry
}

// InMemoryStore is a Store backed by the existing in-memory Registry.
type InMemoryStore struct {
	reg *Registry
}

// NewInMemoryStore constructs a new empty in-memory store.
func NewInMemoryStore() *InMemoryStore { return &InMemoryStore{reg: NewRegistry()} }

func (s *InMemoryStore) AppendBundle(b Bundle) (Bundle, error) { return s.reg.AddBundle(b) }
func (s *InMemoryStore) Head() *Bundle                         { return s.reg.Head() }
func (s *InMemoryStore) GetByHash(hash string) *Bundle         { return s.reg.FindByHash(hash) }
func (s *InMemoryStore) ChainHashes() []string                 { return s.reg.ChainHashes() }
func (s *InMemoryStore) VerifyChain() error                    { return s.reg.VerifyChain() }
func (s *InMemoryStore) Registry() *Registry                   { return s.reg }

// List implements naive pagination (no concurrency guarantees beyond underlying append-only semantics).
func (s *InMemoryStore) List(offset, limit int) ([]Bundle, int) {
	if offset < 0 {
		offset = 0
	}
	total := len(s.reg.bundles)
	if offset >= total {
		return []Bundle{}, total
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	// Return shallow copies to prevent external mutation of internal slice entries.
	out := make([]Bundle, end-offset)
	copy(out, s.reg.bundles[offset:end])
	return out, total
}
