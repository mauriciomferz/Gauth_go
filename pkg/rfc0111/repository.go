package rfc0111

// Milestone 2A: Persistence abstraction scaffold.
// NOTE: Existing code still uses the in-struct map directly; the POARepository
// interface and memoryRepository implementation are introduced now so that
// subsequent changes can migrate logic incrementally without a large diff.

// POARepository defines minimal persistence operations for PowerOfAttorney records.
// A future persistent implementation (e.g. BoltDB / Postgres) will satisfy this.
type POARepository interface {
	Create(p *PowerOfAttorney) error
	Get(id string) (*PowerOfAttorney, bool)
	ListByPrincipal(principal string) []*PowerOfAttorney
	Update(p *PowerOfAttorney) error
}

// memoryRepository is an in-memory implementation (prototype / tests).
type memoryRepository struct{ store map[string]*PowerOfAttorney }

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{store: make(map[string]*PowerOfAttorney)}
}

func (m *memoryRepository) Create(p *PowerOfAttorney) error { m.store[p.ID] = p; return nil }
func (m *memoryRepository) Get(id string) (*PowerOfAttorney, bool) {
	p, ok := m.store[id]
	return p, ok
}
func (m *memoryRepository) ListByPrincipal(principal string) []*PowerOfAttorney {
	out := make([]*PowerOfAttorney, 0)
	for _, p := range m.store {
		if p.Grantor == principal || p.Grantee == principal {
			out = append(out, p)
		}
	}
	return out
}

func (m *memoryRepository) Update(p *PowerOfAttorney) error {
	if _, ok := m.store[p.ID]; !ok {
		return nil
	}
	m.store[p.ID] = p
	return nil
}
