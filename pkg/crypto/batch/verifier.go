package batch

import (
	"context"
	"crypto/ed25519"
	"errors"
	"runtime"

	"golang.org/x/sync/errgroup"
)

// Item represents a single signature verification task.
type Item struct {
	PublicKey ed25519.PublicKey
	Message   []byte
	Signature []byte
}

// VerifyBatchEd25519 verifies a batch of Ed25519 signatures in parallel.
// It returns nil if all signatures are valid.
// It returns an error if any signature is invalid or malformed.
// Note: This does not implement true cryptographic batch verification (s*B + h*A),
// but rather parallelizes standard verification for throughput.
func VerifyBatchEd25519(ctx context.Context, items []Item) error {
	if len(items) == 0 {
		return nil
	}

	// Optimize worker count based on CPU cores
	numWorkers := runtime.NumCPU()
	if len(items) < numWorkers {
		numWorkers = len(items)
	}

	g, _ := errgroup.WithContext(ctx)

	// Channel to distribute work
	workChan := make(chan Item, len(items))
	for _, item := range items {
		workChan <- item
	}
	close(workChan)

	// Start workers
	for i := 0; i < numWorkers; i++ {
		g.Go(func() error {
			for item := range workChan {
				if len(item.PublicKey) != ed25519.PublicKeySize {
					return errors.New("invalid public key size")
				}
				if !ed25519.Verify(item.PublicKey, item.Message, item.Signature) {
					return errors.New("signature verification failed")
				}
			}
			return nil
		})
	}

	return g.Wait()
}
