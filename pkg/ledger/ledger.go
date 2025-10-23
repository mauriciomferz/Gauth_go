package ledger

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// Entry represents an immutable audit record.
type Entry struct {
	ID        string                 `json:"id"`
	TS        time.Time              `json:"ts"`
	Type      string                 `json:"type"`
	Subject   string                 `json:"subject"`
	Object    string                 `json:"object"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	PrevHash  string                 `json:"prev_hash"`
	Hash      string                 `json:"hash"`
	Signature *EntrySignature        `json:"signature,omitempty"`
}

// EntrySignature provides authenticity metadata for a ledger entry.
// Signature is generated over the same canonical payload used for hash computation: prevHash || canonicalWithoutHash(entry).
type EntrySignature struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
	SigBase64 string `json:"sig"`
}

// VerificationResult summarizes chain verification outcome.
type VerificationResult struct {
	Count      int    `json:"count"`
	FirstHash  string `json:"first_hash"`
	LastHash   string `json:"last_hash"`
	Mismatches int    `json:"mismatches"`
}

// Store defines ledger operations.
// Phase 1: in-memory prototype; later replace with Bolt.
type Store interface {
	Append(ctx context.Context, e *Entry) error
	Get(ctx context.Context, id string) (*Entry, error)
	QueryBySubject(ctx context.Context, subject string) ([]*Entry, error)
	QueryByObject(ctx context.Context, object string) ([]*Entry, error)
	VerifyChain(ctx context.Context) (*VerificationResult, error)
}

// ChainTip returns the current last hash of the ledger if retrievable, otherwise empty string.
// It performs a lightweight lookup without full verification when possible.
func ChainTip(s Store) string {
	switch st := s.(type) {
	case *memoryStore:
		st.mu.RLock()
		defer st.mu.RUnlock()
		if len(st.entries) == 0 {
			return ""
		}
		return st.entries[len(st.entries)-1].Hash
	case interface{ lastHash() (string, error) }:
		if h, err := st.lastHash(); err == nil {
			return h
		}
	default:
		// Fallback: attempt verify (may be expensive for large ledgers)
		if vr, err := s.VerifyChain(context.Background()); err == nil {
			return vr.LastHash
		}
	}
	return ""
}

// memoryStore is an in-memory append-only chain for prototype usage.
type memoryStore struct {
	mu           sync.RWMutex
	entries      []*Entry
	indexByID    map[string]*Entry
	indexSubject map[string][]*Entry
	indexObject  map[string][]*Entry
	signer       ed25519.PrivateKey // optional signer (Ed25519)
	keyID        string             // key identifier when signer present
	pubKey       ed25519.PublicKey  // retained for potential future verification
}

// NewMemoryStore creates a new in-memory ledger store.
func NewMemoryStore() Store {
	return &memoryStore{
		indexByID:      make(map[string]*Entry),
		indexSubject:   make(map[string][]*Entry),
		indexObject:    make(map[string][]*Entry),
	}
}

// ConfigureEd25519Signer installs an Ed25519 signer for automatic entry signatures.
// keyID must be a stable identifier for pub; if empty a short hex hash is derived.
func (m *memoryStore) ConfigureEd25519Signer(priv ed25519.PrivateKey, pub ed25519.PublicKey, keyID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(priv) != ed25519.PrivateKeySize || len(pub) != ed25519.PublicKeySize {
		return
	}
	if keyID == "" {
		// derive first 12 hex of sha256(pub)
		h := sha256.Sum256(pub)
		keyID = fmt.Sprintf("%x", h[:6])
	}
	m.signer = priv
	m.keyID = keyID
	m.pubKey = pub
}

// Append adds a new entry computing its hash and linking to previous.
func (m *memoryStore) Append(ctx context.Context, e *Entry) error {
	if e == nil {
		return fmt.Errorf("nil entry")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	prevHash := ""
	if len(m.entries) > 0 {
		prevHash = m.entries[len(m.entries)-1].Hash
	}
	e.PrevHash = prevHash
	canon, err := canonicalWithoutHash(e)
	if err != nil {
		return err
	}
	h := sha256.Sum256(append([]byte(prevHash), canon...))
	e.Hash = fmt.Sprintf("%x", h[:])
	// Optional signature
	if m.signer != nil {
		payload := append([]byte(prevHash), canon...)
		sig := ed25519.Sign(m.signer, payload)
		e.Signature = &EntrySignature{Algorithm: "ed25519", KeyID: m.keyID, SigBase64: base64.StdEncoding.EncodeToString(sig)}
	}
	m.entries = append(m.entries, e)
	m.indexByID[e.ID] = e
	m.indexSubject[e.Subject] = append(m.indexSubject[e.Subject], e)
	m.indexObject[e.Object] = append(m.indexObject[e.Object], e)
	return nil
}

// Get retrieves an entry by ID.
func (m *memoryStore) Get(ctx context.Context, id string) (*Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if e, ok := m.indexByID[id]; ok {
		return e, nil
	}
	return nil, fmt.Errorf("not found")
}

// QueryBySubject returns entries for a subject.
func (m *memoryStore) QueryBySubject(ctx context.Context, subject string) ([]*Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*Entry{}, m.indexSubject[subject]...), nil
}

// QueryByObject returns entries for an object.
func (m *memoryStore) QueryByObject(ctx context.Context, object string) ([]*Entry, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]*Entry{}, m.indexObject[object]...), nil
}

// VerifyChain recomputes hash linkage for all entries.
func (m *memoryStore) VerifyChain(ctx context.Context) (*VerificationResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := &VerificationResult{Count: len(m.entries)}
	var prev string
	for i, e := range m.entries {
		canon, err := canonicalWithoutHash(e)
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(append([]byte(prev), canon...))
		expected := fmt.Sprintf("%x", h[:])
		if i == 0 {
			res.FirstHash = expected
		}
		if expected != e.Hash {
			res.Mismatches++
		}
		// Verify signature if present and signer configured (we have public key stored)
		if e.Signature != nil && len(m.pubKey) == ed25519.PublicKeySize {
			payload := append([]byte(prev), canon...)
			sigBytes, err := base64.StdEncoding.DecodeString(e.Signature.SigBase64)
			if err != nil || len(sigBytes) != ed25519.SignatureSize || !ed25519.Verify(m.pubKey, payload, sigBytes) {
				res.Mismatches++ // treat signature failure as mismatch
			}
		}
		prev = e.Hash
	}
	res.LastHash = prev
	return res, nil
}

// canonicalWithoutHash produces stable JSON excluding Hash field.
func canonicalWithoutHash(e *Entry) ([]byte, error) {
	// Build a map with deterministic key order by constructing a struct and marshalling.
	// Simpler approach: create interim struct omitting Hash.
	aux := struct {
		ID       string                 `json:"id"`
		TS       time.Time              `json:"ts"`
		Type     string                 `json:"type"`
		Subject  string                 `json:"subject"`
		Object   string                 `json:"object"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
		PrevHash string                 `json:"prev_hash"`
	}{
		ID: e.ID, TS: e.TS, Type: e.Type, Subject: e.Subject, Object: e.Object, Metadata: e.Metadata, PrevHash: e.PrevHash,
	}
	return json.Marshal(aux)
}
