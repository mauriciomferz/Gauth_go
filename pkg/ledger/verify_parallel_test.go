package ledger

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVerifyChainParallel(t *testing.T) {
	// 1. Setup Ledger with Signer
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "parallel_test.db")

	store, err := NewBoltStore(dbPath)
	require.NoError(t, err)
	defer store.(*boltStore).Close()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	store.(*boltStore).ConfigureEd25519Signer(priv, pub, "key-1")

	// 2. Populate Entries
	ctx := context.Background()
	count := 2500
	for i := 0; i < count; i++ {
		entry := &Entry{
			ID:   "e" + string(rune(i)),
			TS:   time.Now().UTC(),
			Type: "test",
		}
		err := store.Append(ctx, entry)
		require.NoError(t, err)
	}

	// 3. Verify Sequential (Baseline)
	startSeq := time.Now()
	resSeq, err := store.VerifyChain(ctx)
	durSeq := time.Since(startSeq)
	require.NoError(t, err)
	assert.Equal(t, 0, resSeq.Mismatches)
	assert.Equal(t, count, resSeq.Count)

	// 4. Verify Parallel
	bs := store.(*boltStore)
	startPar := time.Now()
	resPar, err := bs.VerifyChainParallel(ctx)
	durPar := time.Since(startPar)
	require.NoError(t, err)
	assert.Equal(t, 0, resPar.Mismatches)
	assert.Equal(t, count, resPar.Count)
	assert.Equal(t, resSeq.LastHash, resPar.LastHash)

	t.Logf("Sequential: %v, Parallel: %v", durSeq, durPar)
	// Note: Parallel might be slower for small batches or small item counts due to overhead.
	// 2500 items might be borderline.
}

func BenchmarkVerifyChain(b *testing.B) {
	// Setup once
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_seq.db")
	store, _ := NewBoltStore(dbPath)
	defer store.(*boltStore).Close()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	store.(*boltStore).ConfigureEd25519Signer(priv, pub, "key-1")

	ctx := context.Background()
	for i := 0; i < 5000; i++ {
		if err := store.Append(ctx, &Entry{ID: "id", TS: time.Now(), Type: "bench"}); err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.VerifyChain(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkVerifyChainParallel(b *testing.B) {
	// Setup once
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench_par.db")
	store, _ := NewBoltStore(dbPath)
	defer store.(*boltStore).Close()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	store.(*boltStore).ConfigureEd25519Signer(priv, pub, "key-1")

	ctx := context.Background()
	for i := 0; i < 5000; i++ {
		if err := store.Append(ctx, &Entry{ID: "id", TS: time.Now(), Type: "bench"}); err != nil {
			b.Fatal(err)
		}
	}
	bs := store.(*boltStore)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := bs.VerifyChainParallel(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
