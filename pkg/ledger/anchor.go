package ledger

import (
	"context"
	"fmt"
)

// AnchorClient defines an external timestamp anchoring interface.
// Implementations should record (hash, timestamp) in an external immutable medium (e.g. blockchain, TSA, notarization API).
type AnchorClient interface {
	Anchor(hash string) error
}

// NoopAnchorClient is a do-nothing implementation.
type NoopAnchorClient struct{}

func (NoopAnchorClient) Anchor(hash string) error { return nil }

// AnchoringStore is a decorator that wraps a Store and invokes AnchorClient after successful append.
type AnchoringStore struct {
	inner  Store
	anchor AnchorClient
}

// NewAnchoringStore wraps an existing Store with anchoring behavior.
func NewAnchoringStore(inner Store, ac AnchorClient) *AnchoringStore {
	if inner == nil {
		return nil
	}
	if ac == nil {
		ac = NoopAnchorClient{}
	}
	return &AnchoringStore{inner: inner, anchor: ac}
}

func (a *AnchoringStore) Append(ctx context.Context, e *Entry) error {
	if err := a.inner.Append(ctx, e); err != nil {
		return err
	}
	tip := ChainTip(a.inner)
	if tip != "" {
		if err := a.anchor.Anchor(tip); err != nil {
			return fmt.Errorf("anchoring failed: %w", err)
		}
	}
	return nil
}
func (a *AnchoringStore) Get(ctx context.Context, id string) (*Entry, error) {
	return a.inner.Get(ctx, id)
}
func (a *AnchoringStore) QueryBySubject(ctx context.Context, s string) ([]*Entry, error) {
	return a.inner.QueryBySubject(ctx, s)
}
func (a *AnchoringStore) QueryByObject(ctx context.Context, o string) ([]*Entry, error) {
	return a.inner.QueryByObject(ctx, o)
}
func (a *AnchoringStore) VerifyChain(ctx context.Context) (*VerificationResult, error) {
	return a.inner.VerifyChain(ctx)
}
