package keyring

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// SymKey represents a symmetric key with metadata (no secret exposure via String()).
type SymKey struct {
	ID        string // stable key identifier (hex of first 8 bytes + timestamp chunk)
	Material  []byte // raw key bytes (length 32 for PASETO local)
	CreatedAt time.Time
}

// KeyRing maintains an active key and a bounded set of previous keys (grace window).
type KeyRing struct {
	active   *SymKey
	previous []*SymKey
	maxPrev  int
}

// NewKeyRing creates a key ring with a freshly generated active key.
func NewKeyRing() *KeyRing {
	k := generateSymKey()
	return &KeyRing{active: k, previous: []*SymKey{}, maxPrev: 3}
}

// Active returns the current active key.
func (kr *KeyRing) Active() *SymKey { return kr.active }

// Previous returns snapshot of previous keys.
func (kr *KeyRing) Previous() []*SymKey { return append([]*SymKey(nil), kr.previous...) }

// Rotate replaces the active key, pushing the old key into the previous list.
func (kr *KeyRing) Rotate() *SymKey {
	if kr.active != nil {
		kr.previous = append([]*SymKey{kr.active}, kr.previous...)
		if len(kr.previous) > kr.maxPrev {
			kr.previous = kr.previous[:kr.maxPrev]
		}
	}
	kr.active = generateSymKey()
	return kr.active
}

func generateSymKey() *SymKey {
	material := make([]byte, 32)
	if _, err := rand.Read(material); err != nil {
		panic("key generation failure: " + err.Error())
	}
	id := keyID(material)
	return &SymKey{ID: id, Material: material, CreatedAt: time.Now().UTC()}
}

func keyID(b []byte) string {
	if len(b) < 8 {
		return fmt.Sprintf("key_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("k_%s_%x", hex.EncodeToString(b[:4]), time.Now().Unix()%0xffff)
}
