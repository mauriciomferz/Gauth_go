package ledger

// BoltDB-backed persistent ledger prototype (Phase 2).
// Provides basic append-only hash chained integrity identical to memoryStore.
// Not optimized for large query workloads; QueryBySubject/Object perform full scans.

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Bucket names.
const (
	bucketEntries = "entries"  // sequence -> entry JSON
	bucketIndexID = "index_id" // entry.ID -> sequence (8 bytes)
)

// boltStore implements Store using BoltDB for persistence.
type boltStore struct {
	db     *bolt.DB
	mu     sync.RWMutex // coarse lock for lifecycle ops; transactions give isolation inside.
	path   string
	signer ed25519.PrivateKey // optional signer
	keyID  string
	pubKey ed25519.PublicKey
	// External anchoring prototype: when enableAnchorFile != "", each successful append emits/update an anchor material file.
	anchorFilePath      string
	lastAnchorWrite     time.Time
	anchorWriteInterval time.Duration
	anchorWrites        uint64 // count of writes (for observability)
}

// NewBoltStore opens (or creates) a BoltDB database at the given path.
// It initializes required buckets. Caller should Close() when done (not exposed via Store yet).
func NewBoltStore(path string) (Store, error) {
	if path == "" {
		return nil, errors.New("bolt ledger: path required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("bolt ledger: resolve path: %w", err)
	}
	db, err := bolt.Open(abs, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("bolt ledger: open: %w", err)
	}
	// Create buckets if not exist.
	if err := db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists([]byte(bucketEntries)); e != nil {
			return e
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(bucketIndexID)); e != nil {
			return e
		}
		return nil
	}); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("bolt ledger: init buckets: %w", err)
	}
	return &boltStore{db: db, path: abs, anchorWriteInterval: 5 * time.Second}, nil
}

// ConfigureEd25519Signer installs an Ed25519 signer for automatic persistent entry signatures.
// keyID may be empty to derive from public key (first 12 hex of sha256(pub)).
func (b *boltStore) ConfigureEd25519Signer(priv ed25519.PrivateKey, pub ed25519.PublicKey, keyID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(priv) != ed25519.PrivateKeySize || len(pub) != ed25519.PublicKeySize {
		return
	}
	if keyID == "" {
		h := sha256.Sum256(pub)
		keyID = fmt.Sprintf("%x", h[:6])
	}
	b.signer = priv
	b.keyID = keyID
	b.pubKey = pub
}

// Append adds an entry computing chain linkage & hash.
func (b *boltStore) Append(ctx context.Context, e *Entry) error {
	if e == nil {
		return fmt.Errorf("nil entry")
	}
	// Use write transaction.
	return b.db.Update(func(tx *bolt.Tx) error {
		entriesB := tx.Bucket([]byte(bucketEntries))
		idB := tx.Bucket([]byte(bucketIndexID))
		if entriesB == nil || idB == nil {
			return errors.New("ledger: missing buckets")
		}
		// Determine sequence (monotonic). Use NextSequence for order.
		seq, err := entriesB.NextSequence()
		if err != nil {
			return fmt.Errorf("next sequence: %w", err)
		}
		// Previous hash: fetch last entry by scanning for highest sequence (Bolt keeps keys sorted; NextSequence gives next).
		var prevHash string
		c := entriesB.Cursor()
		k, v := c.Last()
		if k != nil && v != nil {
			var prev Entry
			if unmarshalErr := json.Unmarshal(v, &prev); unmarshalErr == nil {
				prevHash = prev.Hash
			}
		}
		e.PrevHash = prevHash
		canon, err := canonicalWithoutHash(e)
		if err != nil {
			return err
		}
		h := sha256.Sum256(append([]byte(prevHash), canon...))
		e.Hash = fmt.Sprintf("%x", h[:])
		// Optional signature
		if b.signer != nil {
			payload := append([]byte(prevHash), canon...)
			sig := ed25519.Sign(b.signer, payload)
			e.Signature = &EntrySignature{Algorithm: "ed25519", KeyID: b.keyID, SigBase64: base64.StdEncoding.EncodeToString(sig)}
		}
		// Marshal and store.
		data, err := json.Marshal(e)
		if err != nil {
			return fmt.Errorf("marshal entry: %w", err)
		}
		// Sequence key as 8-byte big-endian.
		seqKey := make([]byte, 8)
		binary.BigEndian.PutUint64(seqKey, seq)
		if err := entriesB.Put(seqKey, data); err != nil {
			return fmt.Errorf("put entry: %w", err)
		}
		if err := idB.Put([]byte(e.ID), seqKey); err != nil {
			return fmt.Errorf("put id index: %w", err)
		}
		// After commit: optionally emit anchor file if interval elapsed.
		if b.anchorFilePath != "" {
			// Non-blocking check outside transaction.
			if time.Since(b.lastAnchorWrite) >= b.anchorWriteInterval {
				if tip, err := b.lastHash(); err == nil && tip != "" {
					// Compose anchor material: hash + time + optional signer key id + signature for chain tip (if signer configured).
					var sigB64 string
					if b.signer != nil && len(b.pubKey) == ed25519.PublicKeySize {
						payload := []byte(tip)
						sig := ed25519.Sign(b.signer, payload)
						sigB64 = base64.StdEncoding.EncodeToString(sig)
					}
					anchor := struct {
						Hash       string `json:"hash"`
						AnchoredAt string `json:"anchored_at"`
						KeyID      string `json:"key_id,omitempty"`
						Signature  string `json:"signature,omitempty"`
						Writes     uint64 `json:"writes"`
					}{
					Hash:       tip,
					AnchoredAt: time.Now().UTC().Format(time.RFC3339),
					KeyID:      b.keyID,
					Signature:  sigB64,
					Writes:     atomic.AddUint64(&b.anchorWrites, 1),
				}
					if data, mErr := json.Marshal(anchor); mErr != nil {
						fmt.Fprintf(os.Stderr, "[bolt-ledger] anchor marshal error: %v\n", mErr)
					} else if wErr := os.WriteFile(b.anchorFilePath, data, 0o600); wErr != nil {
						fmt.Fprintf(os.Stderr, "[bolt-ledger] anchor write error path=%s err=%v\n", b.anchorFilePath, wErr)
					} else {
						b.lastAnchorWrite = time.Now()
					}
				}
			}
		}
		return nil
	})
}

// Get finds entry by ID using index.
func (b *boltStore) Get(ctx context.Context, id string) (*Entry, error) {
	var out *Entry
	err := b.db.View(func(tx *bolt.Tx) error {
		idB := tx.Bucket([]byte(bucketIndexID))
		entriesB := tx.Bucket([]byte(bucketEntries))
		if idB == nil || entriesB == nil {
			return errors.New("ledger: missing buckets")
		}
		seqKey := idB.Get([]byte(id))
		if seqKey == nil {
			return fmt.Errorf("not found")
		}
		v := entriesB.Get(seqKey)
		if v == nil {
			return fmt.Errorf("not found")
		}
		var e Entry
		if err := json.Unmarshal(v, &e); err != nil {
			return err
		}
		out = &e
		return nil
	})
	return out, err
}

// QueryBySubject performs a scan filtering by Subject.
func (b *boltStore) QueryBySubject(ctx context.Context, subject string) ([]*Entry, error) {
	res := make([]*Entry, 0)
	err := b.db.View(func(tx *bolt.Tx) error {
		entriesB := tx.Bucket([]byte(bucketEntries))
		if entriesB == nil {
			return errors.New("ledger: missing entries bucket")
		}
		c := entriesB.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var e Entry
			if err := json.Unmarshal(v, &e); err != nil {
				return err
			}
			if e.Subject == subject {
				ee := e
				res = append(res, &ee)
			}
		}
		return nil
	})
	return res, err
}

// QueryByObject performs a scan filtering by Object.
func (b *boltStore) QueryByObject(ctx context.Context, object string) ([]*Entry, error) {
	res := make([]*Entry, 0)
	err := b.db.View(func(tx *bolt.Tx) error {
		entriesB := tx.Bucket([]byte(bucketEntries))
		if entriesB == nil {
			return errors.New("ledger: missing entries bucket")
		}
		c := entriesB.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var e Entry
			if err := json.Unmarshal(v, &e); err != nil {
				return err
			}
			if e.Object == object {
				ee := e
				res = append(res, &ee)
			}
		}
		return nil
	})
	return res, err
}

// VerifyChain recomputes linkage to ensure integrity.
func (b *boltStore) VerifyChain(ctx context.Context) (*VerificationResult, error) {
	res := &VerificationResult{}
	err := b.db.View(func(tx *bolt.Tx) error {
		entriesB := tx.Bucket([]byte(bucketEntries))
		if entriesB == nil {
			return errors.New("ledger: missing entries bucket")
		}
		c := entriesB.Cursor()
		var prev string
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var e Entry
			if err := json.Unmarshal(v, &e); err != nil {
				return err
			}
			canon, err := canonicalWithoutHash(&e)
			if err != nil {
				return err
			}
			h := sha256.Sum256(append([]byte(prev), canon...))
			expected := fmt.Sprintf("%x", h[:])
			if res.Count == 0 {
				res.FirstHash = expected
			}
			if expected != e.Hash {
				res.Mismatches++
			}
			if e.Signature != nil && len(b.pubKey) == ed25519.PublicKeySize {
				payload := append([]byte(prev), canon...)
				sigBytes, err := base64.StdEncoding.DecodeString(e.Signature.SigBase64)
				if err != nil || len(sigBytes) != ed25519.SignatureSize || !ed25519.Verify(b.pubKey, payload, sigBytes) {
					res.Mismatches++
				}
			}
			prev = e.Hash
			res.Count++
		}
		res.LastHash = prev
		return nil
	})
	return res, err
}

// Close closes underlying DB (helper for tests). Not part of Store interface yet.
func (b *boltStore) Close() error { return b.db.Close() }

// lastHash returns the current last hash by reading the last sequence entry.
func (b *boltStore) lastHash() (string, error) {
	var hash string
	err := b.db.View(func(tx *bolt.Tx) error {
		entriesB := tx.Bucket([]byte(bucketEntries))
		if entriesB == nil {
			return errors.New("ledger: missing entries bucket")
		}
		c := entriesB.Cursor()
		k, v := c.Last()
		if k == nil || v == nil {
			return nil
		}
		var e Entry
		if err := json.Unmarshal(v, &e); err != nil {
			return err
		}
		hash = e.Hash
		return nil
	})
	return hash, err
}

// EnableAnchorFile configures external anchor material emission to the given file path.
// The file is updated at most every interval (default 5s) after successful appends.
func (b *boltStore) EnableAnchorFile(path string, interval time.Duration) error {
	if path == "" {
		return fmt.Errorf("anchor file path required")
	}
	if interval <= 0 {
		interval = 5 * time.Second
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.anchorFilePath = abs
	b.anchorWriteInterval = interval
	b.mu.Unlock()
	// Write initial file if chain has entries.
	if tip, err := b.lastHash(); err == nil && tip != "" {
		anchor := struct {
			Hash       string `json:"hash"`
			AnchoredAt string `json:"anchored_at"`
			KeyID      string `json:"key_id,omitempty"`
			Signature  string `json:"signature,omitempty"`
			Writes     uint64 `json:"writes"`
		}{Hash: tip, AnchoredAt: time.Now().UTC().Format(time.RFC3339), KeyID: b.keyID}
		if b.signer != nil && len(b.pubKey) == ed25519.PublicKeySize {
			sig := ed25519.Sign(b.signer, []byte(tip))
			anchor.Signature = base64.StdEncoding.EncodeToString(sig)
		}
		data, mErr := json.Marshal(anchor)
		if mErr != nil {
			fmt.Fprintf(os.Stderr, "[bolt-ledger] initial anchor marshal error: %v\n", mErr)
		} else if wErr := os.WriteFile(abs, data, 0o600); wErr != nil {
			fmt.Fprintf(os.Stderr, "[bolt-ledger] initial anchor write error path=%s err=%v\n", abs, wErr)
		} else {
			b.lastAnchorWrite = time.Now()
		}
	}
	return nil
}
