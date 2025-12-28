package poa

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockPoARepository implements PoARepository for testing
type MockPoARepository struct {
	mu    sync.Mutex
	poas  map[string]*PoARecord
	delay time.Duration
}

func NewMockPoARepository() *MockPoARepository {
	return &MockPoARepository{
		poas: make(map[string]*PoARecord),
	}
}

// AddMultiSignature simulates the DB transaction with locking
func (m *MockPoARepository) AddMultiSignature(ctx context.Context, tenantID, poaID string, signerID string, signature map[string]interface{}, threshold int) (*PoARecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate IO delay to encourage race conditions if locking wasn't working (though mutex enforces it here)
	if m.delay > 0 {
		time.Sleep(m.delay)
	}

	key := tenantID + ":" + poaID
	poa, exists := m.poas[key]
	if !exists {
		return nil, assert.AnError
	}

	// Unmarshal existing signatures
	sigs := make(map[string]interface{})
	if poa.MultiSignatures != nil {
		_ = json.Unmarshal(*poa.MultiSignatures, &sigs)
	}

	// Add new signature
	sigs[signerID] = signature

	// Check threshold
	if len(sigs) >= threshold && poa.Status == "pending" {
		poa.Status = "active"
	}

	// Save back
	raw, _ := json.Marshal(sigs)
	jsonRaw := json.RawMessage(raw)
	poa.MultiSignatures = &jsonRaw

	// Create a copy to return
	updatedPoa := *poa
	return &updatedPoa, nil
}

// Stub other interface methods required
func (m *MockPoARepository) CreatePoA(ctx context.Context, poa *PoARecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := poa.TenantID + ":" + poa.ID
	m.poas[key] = poa
	return nil
}

func (m *MockPoARepository) ListPoAs(ctx context.Context, tenantID string, limit, offset int) ([]PoARecord, int, error) {
	return nil, 0, nil
}
func (m *MockPoARepository) GetPoA(ctx context.Context, tenantID, poaID string) (*PoARecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := tenantID + ":" + poaID
	if poa, ok := m.poas[key]; ok {
		return poa, nil
	}
	return nil, assert.AnError
}
func (m *MockPoARepository) RevokePoA(ctx context.Context, tenantID, poaID, revokedBy, reason string) error {
	return nil
}
func (m *MockPoARepository) ApprovePoA(ctx context.Context, tenantID, poaID, approvedBy string) error {
	return nil
}
func (m *MockPoARepository) RejectPoA(ctx context.Context, tenantID, poaID, rejectedBy, reason string) error {
	return nil
}
func (m *MockPoARepository) ValidatePoA(ctx context.Context, tenantID, grantorID, representativeID, action, resource string) (*PoARecord, bool, string) {
	return nil, false, ""
}
func (m *MockPoARepository) GetPoAStats(ctx context.Context, tenantID string) (*PoAStats, error) {
	return nil, nil
}
func (m *MockPoARepository) CreateTemplate(ctx context.Context, template *PoATemplate) error {
	return nil
}
func (m *MockPoARepository) ListTemplates(ctx context.Context, tenantID *string) ([]PoATemplate, error) {
	return nil, nil
}

func TestMultiSigCoordinator_CollectSignature_Concurrency(t *testing.T) {
	repo := NewMockPoARepository()
	// Add a small delay to simulate DB latency
	repo.delay = 10 * time.Millisecond

	coordinator := NewMultiSigCoordinator(repo)

	tenantID := "tenant1"
	poaID := "poa1"

	// Setup initial POA
	repo.CreatePoA(context.Background(), &PoARecord{
		ID:       poaID,
		TenantID: tenantID,
		Status:   "pending",
	})

	threshold := 5
	wg := sync.WaitGroup{}

	// Spawn 5 concurrent signers
	for i := 0; i < threshold; i++ {
		wg.Add(1)
		signerID := fmt.Sprintf("signer-%d", i)
		go func(sid string) {
			defer wg.Done()
			sig := map[string]interface{}{"s": "valid"}
			_, err := coordinator.CollectSignature(context.Background(), tenantID, poaID, sid, sig, threshold)
			assert.NoError(t, err)
		}(signerID)
	}

	wg.Wait()

	// Verify state
	poa, err := repo.GetPoA(context.Background(), tenantID, poaID)
	assert.NoError(t, err)

	// Should be active
	assert.Equal(t, "active", poa.Status)

	// Should have 5 signatures
	var sigs map[string]interface{}
	json.Unmarshal(*poa.MultiSignatures, &sigs)
	assert.Equal(t, threshold, len(sigs))
}

func TestMultiSigCoordinator_CollectSignature_ThresholdNotMet(t *testing.T) {
	repo := NewMockPoARepository()
	coordinator := NewMultiSigCoordinator(repo)

	tenantID := "tenant1"
	poaID := "poa2"

	repo.CreatePoA(context.Background(), &PoARecord{
		ID:       poaID,
		TenantID: tenantID,
		Status:   "pending",
	})

	threshold := 3

	// Add 1 signature
	_, err := coordinator.CollectSignature(context.Background(), tenantID, poaID, "signer1", map[string]interface{}{"v": 1}, threshold)
	assert.NoError(t, err)

	poa, _ := repo.GetPoA(context.Background(), tenantID, poaID)
	assert.Equal(t, "pending", poa.Status)

	// Add 2nd signature
	_, err = coordinator.CollectSignature(context.Background(), tenantID, poaID, "signer2", map[string]interface{}{"v": 1}, threshold)
	assert.NoError(t, err)

	poa, _ = repo.GetPoA(context.Background(), tenantID, poaID)
	assert.Equal(t, "pending", poa.Status)

	// Add 3rd signature -> Active
	_, err = coordinator.CollectSignature(context.Background(), tenantID, poaID, "signer3", map[string]interface{}{"v": 1}, threshold)
	assert.NoError(t, err)

	poa, _ = repo.GetPoA(context.Background(), tenantID, poaID)
	assert.Equal(t, "active", poa.Status)
}
