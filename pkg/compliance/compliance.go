package compliance

import "sync"

// DataClass enumerates categories for data classification.
type DataClass string

const (
	DataClassPersonal    DataClass = "personal"
	DataClassOperational DataClass = "operational"
	DataClassCrypto      DataClass = "cryptographic"
)

// Flow models a data flow between a source and destination component.
type Flow struct {
	ID          string
	Source      string
	Destination string
	DataTypes   []DataClass
	Purpose     string
	Retention   string // human-readable policy reference
}

// Registry stores declared data flows (in-memory stub).
type Registry struct {
	mu    sync.RWMutex
	flows map[string]Flow
}

func NewRegistry() *Registry { return &Registry{flows: map[string]Flow{}} }

func (r *Registry) Register(f Flow) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flows[f.ID] = f
}

func (r *Registry) List() []Flow {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Flow, 0, len(r.flows))
	for _, f := range r.flows {
		out = append(out, f)
	}
	return out
}

// RetentionPolicy placeholder for future structured enforcement.
type RetentionPolicy struct {
	Name        string
	Description string
	TTLDays     int
}
