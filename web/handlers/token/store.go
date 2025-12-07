package token

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const (
	TokenStatusNotFound       = "not_found"
	TokenStatusValid          = "valid"
	TokenStatusAlreadyRevoked = "already_revoked"
	TokenStatusRevoked        = "revoked"
	// Constants referenced in server_clean.go loops/logic
	StatusSuspended  = "suspended"
	StatusTerminated = "terminated"
)

type Token struct {
	ID        string     `json:"id"`
	Value     string     `json:"token"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	Meta      any        `json:"meta,omitempty"`
	Status    string     `json:"status,omitempty"` // active, suspended, terminated
}

type Store struct {
	mu        sync.RWMutex
	cap       int
	tokens    map[string]*Token
	valueIdx  map[string]string // token value -> id
	created   int
	validated int
	revoked   int
}

func NewStore(capacity int) *Store {
	if capacity <= 0 {
		capacity = 200
	}
	return &Store{cap: capacity, tokens: make(map[string]*Token), valueIdx: make(map[string]string)}
}

func (ts *Store) Create(ttlSeconds int, meta any) *Token {
	if ttlSeconds <= 0 {
		ttlSeconds = 300
	}
	t := &Token{ID: randomNonce(10), Value: randomNonce(24), CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Duration(ttlSeconds) * time.Second), Meta: meta, Status: "active"}
	ts.mu.Lock()
	if len(ts.tokens) >= ts.cap { // simple eviction: drop oldest arbitrary (first range)
		for k, v := range ts.tokens {
			delete(ts.valueIdx, v.Value)
			delete(ts.tokens, k)
			break
		}
	}
	ts.tokens[t.ID] = t
	ts.valueIdx[t.Value] = t.ID
	ts.created++
	ts.mu.Unlock()
	return t
}

func (ts *Store) lookupNoLock(idOrVal string) (*Token, bool) {
	if idOrVal == "" {
		return nil, false
	}
	if t, ok := ts.tokens[idOrVal]; ok {
		return t, true
	}
	if id, ok := ts.valueIdx[idOrVal]; ok {
		if t, ok2 := ts.tokens[id]; ok2 {
			return t, true
		}
	}
	return nil, false
}

func (ts *Store) Validate(idOrVal string) (string, *Token) {
	ts.mu.RLock()
	t, ok := ts.lookupNoLock(idOrVal)
	if !ok {
		ts.mu.RUnlock()
		return TokenStatusNotFound, nil
	}
	now := time.Now()
	if t.RevokedAt != nil || t.Status == StatusTerminated {
		ts.mu.RUnlock()
		return TokenStatusRevoked, t
	}
	if t.Status == StatusSuspended {
		ts.mu.RUnlock()
		return StatusSuspended, t
	}
	if now.After(t.ExpiresAt) {
		ts.mu.RUnlock()
		return "expired", t
	}
	ts.mu.RUnlock()
	ts.mu.Lock()
	ts.validated++
	ts.mu.Unlock()
	return TokenStatusValid, t
}

func (ts *Store) Revoke(id string) string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	t, ok := ts.tokens[id]
	if !ok {
		return TokenStatusNotFound
	}
	if t.RevokedAt != nil {
		return TokenStatusAlreadyRevoked
	}
	now := time.Now()
	t.RevokedAt = &now
	ts.revoked++
	return TokenStatusRevoked
}

func (ts *Store) Metrics() map[string]any {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return map[string]any{"created": ts.created, "validated": ts.validated, "revoked": ts.revoked, "total": len(ts.tokens)}
}

func randomNonce(length int) string {
	b := make([]byte, length)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
