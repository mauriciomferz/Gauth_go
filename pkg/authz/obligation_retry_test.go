package authz

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// MockExecutor implements ObligationExecutor for testing.
type MockExecutor struct {
	AttemptCount atomic.Int32
	FailAttempts int32
	AuditCalled  atomic.Int32
}

func (m *MockExecutor) Execute(ob Obligation, ctx map[string]interface{}) error {
	count := m.AttemptCount.Add(1)
	if count <= m.FailAttempts {
		return errors.New("simulated transient error")
	}
	return nil
}

func (m *MockExecutor) PersistAudit(ob Obligation, ctx map[string]interface{}, result error) error {
	m.AuditCalled.Add(1)
	return nil
}

func TestRetryingExecutor_Mandatory_Success(t *testing.T) {
	mock := &MockExecutor{FailAttempts: 2} // Fail twice, succeed on 3rd
	config := DefaultRetryConfig
	config.MaxRetries = 3
	config.BaseDelay = 1 * time.Millisecond

	exec := NewRetryingObligationExecutor(mock, config)
	defer exec.Stop()

	ob := Obligation{ID: "test-1", Mandatory: true}
	err := exec.Execute(ob, nil)

	assert.NoError(t, err)
	assert.Equal(t, int32(3), mock.AttemptCount.Load()) // 1 initial + 2 retries
}

func TestRetryingExecutor_Mandatory_Failure(t *testing.T) {
	mock := &MockExecutor{FailAttempts: 10} // Fail forever
	config := DefaultRetryConfig
	config.MaxRetries = 2
	config.BaseDelay = 1 * time.Millisecond

	exec := NewRetryingObligationExecutor(mock, config)
	defer exec.Stop()

	ob := Obligation{ID: "test-2", Mandatory: true}
	err := exec.Execute(ob, nil)

	assert.Error(t, err)
	assert.Equal(t, int32(3), mock.AttemptCount.Load()) // 1 initial + 2 retries
}

func TestRetryingExecutor_Async_Success(t *testing.T) {
	mock := &MockExecutor{FailAttempts: 2} // Fail twice, succeed on 3rd
	config := DefaultRetryConfig
	config.MaxRetries = 3
	config.BaseDelay = 1 * time.Millisecond

	exec := NewRetryingObligationExecutor(mock, config)
	defer exec.Stop()

	ob := Obligation{ID: "test-3", Mandatory: false}
	err := exec.Execute(ob, nil)

	assert.NoError(t, err) // Should return nil immediately after queuing

	// Wait for async processing
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(3), mock.AttemptCount.Load())
}

func TestRetryingExecutor_Async_Ordering(t *testing.T) {
	// Verify that tasks are picked up. Detailed ordering is hard to test with concurrency.
	mock := &MockExecutor{FailAttempts: 0}
	config := DefaultRetryConfig

	exec := NewRetryingObligationExecutor(mock, config)
	defer exec.Stop()

	assert.NoError(t, exec.Execute(Obligation{ID: "A", Mandatory: false}, nil))
	assert.NoError(t, exec.Execute(Obligation{ID: "B", Mandatory: false}, nil))

	time.Sleep(20 * time.Millisecond)
	assert.GreaterOrEqual(t, mock.AttemptCount.Load(), int32(2))
}
