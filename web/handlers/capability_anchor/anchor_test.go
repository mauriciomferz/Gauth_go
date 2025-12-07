package capability_anchor

import (
	"context"
	"testing"
	"time"

	anchorint "github.com/mauriciomferz/Gauth_go/internal/anchor"
)

type MockProvider struct {
	LastHash string
}

func (m *MockProvider) Anchor(hash string) (anchorint.Receipt, error) {
	m.LastHash = hash
	return anchorint.Receipt{
		Hash:      hash,
		Timestamp: time.Now(),
		Provider:  "mock",
		Version:   1,
	}, nil
}

func (m *MockProvider) Verify(r anchorint.Receipt) error {
	return nil
}

func (m *MockProvider) Latest() anchorint.Receipt {
	return anchorint.Receipt{
		Hash:      m.LastHash,
		Timestamp: time.Now(),
		Provider:  "mock",
		Version:   1,
	}
}

type MockStore struct{}

func (s *MockStore) Append(r anchorint.ExternalAnchorReceipt) (anchorint.StoredExternalAnchorReceipt, error) {
	return anchorint.StoredExternalAnchorReceipt{}, nil
}
func (s *MockStore) Latest() anchorint.StoredExternalAnchorReceipt {
	return anchorint.StoredExternalAnchorReceipt{}
}
func (s *MockStore) Entries() []anchorint.StoredExternalAnchorReceipt { return nil }
func (s *MockStore) Load() error                                      { return nil }
func (s *MockStore) VerifyIncremental() (string, int, string)         { return "ok", -1, "" }

func TestAnchorHandlerFlow(t *testing.T) {
	mock := &MockProvider{LastHash: "initial"}
	h := NewHandler(mock, &MockStore{})
	h.SetRegistryHash("test-hash-123")

	ctx := context.Background()
	receipt, err := h.Anchor(ctx)
	if err != nil {
		t.Fatalf("Anchor failed: %v", err)
	}

	if receipt.Hash != "test-hash-123" {
		t.Errorf("Expected hash 'test-hash-123', got '%s'", receipt.Hash)
	}

	if h.GetLastReceipt().Hash != "test-hash-123" {
		t.Errorf("Handler cache not updated")
	}

	// Verify update receipt manually
	h.UpdateReceipt(anchorint.Receipt{Hash: "manual-update", Timestamp: time.Now()})
	if h.GetLastReceipt().Hash != "manual-update" {
		t.Errorf("UpdateReceipt failed")
	}
}

func TestNoProvider(t *testing.T) {
	h := NewHandler(nil, nil)
	ctx := context.Background()
	_, err := h.Anchor(ctx)
	if err == nil {
		t.Error("Expected error when no provider is set")
	}
}
