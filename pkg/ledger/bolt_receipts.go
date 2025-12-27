package ledger

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mauriciomferz/Gauth_go/internal/anchor"
	bolt "go.etcd.io/bbolt"
)

const bucketExternalReceipts = "external_receipts" // sequence -> stored receipt JSON

// BoltReceiptStore implements anchor.ReceiptStore using BoltDB.
type BoltReceiptStore struct {
	db *bolt.DB
}

// NewBoltReceiptStore creates a store using an existing BoltDB instance.
// It initializes the bucket if not present.
func NewBoltReceiptStore(db *bolt.DB) (*BoltReceiptStore, error) {
	if db == nil {
		return nil, errors.New("bolt db is nil")
	}
	err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucketExternalReceipts))
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to init receipt bucket: %w", err)
	}
	return &BoltReceiptStore{db: db}, nil
}

// Append persists a receipt.
func (s *BoltReceiptStore) Append(r anchor.ExternalAnchorReceipt) (anchor.StoredExternalAnchorReceipt, error) {
	var stored anchor.StoredExternalAnchorReceipt
	err := s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketExternalReceipts))
		if b == nil {
			return errors.New("bucket not found")
		}

		// 1. Get Head
		var prevHash string
		c := b.Cursor()
		_, v := c.Last() // Bolt keys are sorted sequences
		if v != nil {
			var prev anchor.StoredExternalAnchorReceipt
			if err := json.Unmarshal(v, &prev); err == nil {
				prevHash = prev.ChainHash
			}
		}

		// 2. Compute Linkage
		sr := anchor.StoredExternalAnchorReceipt{
			ExternalAnchorReceipt: r,
			PrevHash:              prevHash,
		}

		// To compute chain hash, we need the canonical base json
		base := struct {
			Hash           string  `json:"hash"`
			Timestamp      string  `json:"timestamp"`
			Provider       string  `json:"provider"`
			Version        int     `json:"version"`
			LatencySeconds float64 `json:"latency_seconds"`
			PrevHash       string  `json:"prev_hash"`
		}{
			Hash: r.Hash, Timestamp: r.Timestamp, Provider: r.Provider,
			Version: r.Version, LatencySeconds: r.LatencySeconds, PrevHash: sr.PrevHash,
		}
		enc, err := json.Marshal(base)
		if err != nil {
			return err
		}
		h := sha256.Sum256(append([]byte(sr.PrevHash), enc...))
		sr.ChainHash = fmt.Sprintf("%x", h[:])

		// 3. Store
		seq, err := b.NextSequence()
		if err != nil {
			return err
		}

		seqKey := make([]byte, 8)
		binary.BigEndian.PutUint64(seqKey, seq)

		data, err := json.Marshal(sr)
		if err != nil {
			return err
		}

		if err := b.Put(seqKey, data); err != nil {
			return err
		}

		stored = sr
		return nil
	})

	return stored, err
}

// VerifyIncremental performs incremental verification.
// For BoltDB, we can just scan all for now as optimization isn't critical for this prototype.
func (s *BoltReceiptStore) VerifyIncremental() (string, int, string) {
	var status string = anchor.ExternalReceiptStatusOK
	var mismatchIdx int = -1
	var headHash string

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketExternalReceipts))
		if b == nil {
			status = anchor.ExternalReceiptStatusEmpty
			return nil
		}

		c := b.Cursor()
		k, v := c.First()
		if k == nil {
			status = anchor.ExternalReceiptStatusEmpty
			return nil
		}

		var prevHash string
		idx := 0

		for k != nil {
			var e anchor.StoredExternalAnchorReceipt
			if err := json.Unmarshal(v, &e); err != nil {
				status = anchor.ExternalReceiptStatusMismatch
				mismatchIdx = idx
				return nil
			}

			// Verify
			base := struct {
				Hash           string  `json:"hash"`
				Timestamp      string  `json:"timestamp"`
				Provider       string  `json:"provider"`
				Version        int     `json:"version"`
				LatencySeconds float64 `json:"latency_seconds"`
				PrevHash       string  `json:"prev_hash"`
			}{
				Hash: e.Hash, Timestamp: e.Timestamp, Provider: e.Provider,
				Version: e.Version, LatencySeconds: e.LatencySeconds, PrevHash: e.PrevHash,
			}
			enc, _ := json.Marshal(base)
			h := sha256.Sum256(append([]byte(e.PrevHash), enc...))
			expected := fmt.Sprintf("%x", h[:])

			if expected != e.ChainHash || (idx == 0 && e.PrevHash != "") || (idx > 0 && e.PrevHash != prevHash) {
				status = anchor.ExternalReceiptStatusMismatch
				mismatchIdx = idx
				// Keep headHash as whatever we had before failing?
				// Or current? Let's return what we have processed so far or the failure point
				return nil
			}

			prevHash = e.ChainHash
			headHash = e.ChainHash

			k, v = c.Next()
			idx++
		}
		return nil
	})

	if err != nil {
		return anchor.ExternalReceiptStatusMismatch, 0, ""
	}

	return status, mismatchIdx, headHash
}

// Latest returns the most recent receipt.
func (s *BoltReceiptStore) Latest() anchor.StoredExternalAnchorReceipt {
	var r anchor.StoredExternalAnchorReceipt
	_ = s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketExternalReceipts))
		if b == nil {
			return nil
		}
		c := b.Cursor()
		_, v := c.Last()
		if v != nil {
			_ = json.Unmarshal(v, &r)
		}
		return nil
	})
	return r
}
