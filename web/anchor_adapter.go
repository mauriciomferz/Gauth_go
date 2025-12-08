package web

import (
	"fmt"

	anchorint "github.com/mauriciomferz/Gauth_go/internal/anchor"
	"github.com/mauriciomferz/Gauth_go/pkg/anchor"
)

// anchorClientAdapter adapts pkg/anchor.MemoryAnchor to internal/anchor.Provider.
type anchorClientAdapter struct {
	client *anchor.MemoryAnchor
}

func (a *anchorClientAdapter) Anchor(hash string) (anchorint.Receipt, error) {
	rec, err := a.client.Anchor(hash)
	if err != nil {
		return anchorint.Receipt{}, err
	}
	return anchorint.Receipt{
		Hash:      rec.Hash,
		Timestamp: rec.AnchoredAt,
		Provider:  "memory",
		Version:   1,
	}, nil
}

func (a *anchorClientAdapter) Latest() anchorint.Receipt {
	rec, _ := a.client.LatestAnchor()
	return anchorint.Receipt{
		Hash:      rec.Hash,
		Timestamp: rec.AnchoredAt,
		Provider:  "memory",
		Version:   1,
	}
}

func (a *anchorClientAdapter) Verify(r anchorint.Receipt) error {
	// Basic consistency check (prototype)
	if r.Hash == "" {
		return fmt.Errorf("invalid receipt hash")
	}
	return nil
}
