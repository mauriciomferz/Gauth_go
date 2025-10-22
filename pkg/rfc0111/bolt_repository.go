package rfc0111

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Bolt buckets
const (
	boltBucketPOA       = "poa"       // primary storage by ID
	boltBucketPrincipal = "principal" // secondary index: principal -> JSON array of POA IDs (grantor+grantee)
)

// BoltRepository is a BoltDB-backed implementation of POARepository.
// Layout:
//
//	Bucket "poa": key = POA.ID, value = JSON serialized PowerOfAttorney
//	Bucket "principal": key = principal (grantor or grantee), value = JSON array of POA IDs
//
// Rationale: Simple index supporting ListByPrincipal without scanning all POAs.
// Concurrency: Bolt supports multiple readers and single writer; we keep writes small.
// Thread-safety: DB handle is safe for concurrent use; we only guard close semantics.
type BoltRepository struct {
	db   *bolt.DB
	path string
	mu   sync.RWMutex // protects Close operations / future mutable state
}

// NewBoltRepository opens (or creates) a BoltDB file at path.
// It initializes required buckets. Caller should call Close() when done.
func NewBoltRepository(path string) (*BoltRepository, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("abs path: %w", err)
	}
	db, err := bolt.Open(abs, 0o600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("open bolt: %w", err)
	}
	br := &BoltRepository{db: db, path: abs}
	if err := db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists([]byte(boltBucketPOA)); e != nil {
			return e
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(boltBucketPrincipal)); e != nil {
			return e
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, err
	}
	return br, nil
}

// Close closes the underlying DB.
func (b *BoltRepository) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.db == nil {
		return nil
	}
	err := b.db.Close()
	b.db = nil
	return err
}

// ensureOpen returns error if DB has been closed.
func (b *BoltRepository) ensureOpen() error {
	if b.db == nil {
		return errors.New("bolt repository closed")
	}
	return nil
}

// Create persists a new PowerOfAttorney. Overwrites existing same ID (idempotent issuance path).
func (b *BoltRepository) Create(p *PowerOfAttorney) error {
	if p == nil {
		return errors.New("nil poa")
	}
	if p.ID == "" {
		return errors.New("empty id")
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return b.db.Update(func(tx *bolt.Tx) error {
		poaB := tx.Bucket([]byte(boltBucketPOA))
		princB := tx.Bucket([]byte(boltBucketPrincipal))
		if poaB == nil || princB == nil {
			return errors.New("missing buckets")
		}
		if err := poaB.Put([]byte(p.ID), data); err != nil {
			return err
		}
		// Index grantor & grantee
		for _, principal := range []string{p.Grantor, p.Grantee} {
			if principal == "" {
				continue
			}
			prev := princB.Get([]byte(principal))
			var ids []string
			if prev != nil {
				_ = json.Unmarshal(prev, &ids)
			}
			// append if not present
			found := false
			for _, id := range ids {
				if id == p.ID {
					found = true
					break
				}
			}
			if !found {
				ids = append(ids, p.ID)
			}
			enc, mErr := json.Marshal(ids)
			if mErr != nil {
				return mErr
			}
			if err := princB.Put([]byte(principal), enc); err != nil {
				return err
			}
		}
		return nil
	})
}

// Get retrieves a POA by ID.
func (b *BoltRepository) Get(id string) (*PowerOfAttorney, bool) {
	if id == "" {
		return nil, false
	}
	if err := b.ensureOpen(); err != nil {
		return nil, false
	}
	var out *PowerOfAttorney
	_ = b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(boltBucketPOA))
		if bkt == nil {
			return errors.New("missing bucket")
		}
		v := bkt.Get([]byte(id))
		if v == nil {
			return nil
		}
		var p PowerOfAttorney
		if err := json.Unmarshal(v, &p); err != nil {
			return err
		}
		out = &p
		return nil
	})
	if out == nil {
		return nil, false
	}
	return out, true
}

// ListByPrincipal lists POAs where principal is grantor or grantee.
func (b *BoltRepository) ListByPrincipal(principal string) []*PowerOfAttorney {
	if principal == "" {
		return []*PowerOfAttorney{}
	}
	if err := b.ensureOpen(); err != nil {
		return []*PowerOfAttorney{}
	}
	ids := make([]string, 0)
	_ = b.db.View(func(tx *bolt.Tx) error {
		bkt := tx.Bucket([]byte(boltBucketPrincipal))
		if bkt == nil {
			return errors.New("missing bucket")
		}
		raw := bkt.Get([]byte(principal))
		if raw != nil {
			_ = json.Unmarshal(raw, &ids)
		}
		return nil
	})
	if len(ids) == 0 {
		return []*PowerOfAttorney{}
	}
	out := make([]*PowerOfAttorney, 0, len(ids))
	_ = b.db.View(func(tx *bolt.Tx) error {
		poaB := tx.Bucket([]byte(boltBucketPOA))
		if poaB == nil {
			return errors.New("missing bucket")
		}
		for _, id := range ids {
			v := poaB.Get([]byte(id))
			if v == nil {
				continue
			}
			var p PowerOfAttorney
			if err := json.Unmarshal(v, &p); err == nil {
				out = append(out, &p)
			}
		}
		return nil
	})
	return out
}

// Update replaces existing POA content (status, timestamps, etc.). Returns error if not found.
func (b *BoltRepository) Update(p *PowerOfAttorney) error {
	if p == nil {
		return errors.New("nil poa")
	}
	if p.ID == "" {
		return errors.New("empty id")
	}
	if err := b.ensureOpen(); err != nil {
		return err
	}
	data, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return b.db.Update(func(tx *bolt.Tx) error {
		poaB := tx.Bucket([]byte(boltBucketPOA))
		if poaB == nil {
			return errors.New("missing bucket")
		}
		existing := poaB.Get([]byte(p.ID))
		if existing == nil {
			return errors.New("not found")
		}
		return poaB.Put([]byte(p.ID), data)
	})
}

// WithPOARepository injects a custom repository (e.g., Bolt).
func WithPOARepository(repo POARepository) Option {
	return func(s *Service) {
		if repo != nil {
			s.repo = repo
		}
	}
}
