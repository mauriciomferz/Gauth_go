package metrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockMetrics embeds Noop to satisfy interface, overrides specific methods for tracking.
type mockMetrics struct {
	noop
	signaturesIssued int
	latencies        []time.Duration
}

func (m *mockMetrics) IncSignaturesIssued() {
	m.signaturesIssued++
}

func (m *mockMetrics) ObserveValidationLatency(d time.Duration) {
	m.latencies = append(m.latencies, d)
}

func TestFailoverMetricsCollector(t *testing.T) {
	primary := &mockMetrics{}
	secondary := &mockMetrics{}

	failover := NewFailoverMetricsCollector(primary, secondary)

	// 1. Default: Primary active
	assert.True(t, failover.IsPrimaryHealthy())
	failover.IncSignaturesIssued()
	assert.Equal(t, 1, primary.signaturesIssued)
	assert.Equal(t, 0, secondary.signaturesIssued)

	// 2. Mark Unhealthy -> Switch to Secondary
	failover.MarkPrimaryUnhealthy()
	assert.False(t, failover.IsPrimaryHealthy())

	failover.IncSignaturesIssued()
	assert.Equal(t, 1, primary.signaturesIssued)   // Unchanged
	assert.Equal(t, 1, secondary.signaturesIssued) // Incremented

	// 3. Delegation of other methods
	failover.ObserveValidationLatency(100 * time.Millisecond)
	assert.Empty(t, primary.latencies)
	assert.Len(t, secondary.latencies, 1)

	// 4. Mark Healthy -> Switch back to Primary
	failover.MarkPrimaryHealthy()
	assert.True(t, failover.IsPrimaryHealthy())

	failover.IncSignaturesIssued()
	assert.Equal(t, 2, primary.signaturesIssued)   // Incremented again
	assert.Equal(t, 1, secondary.signaturesIssued) // Unchanged
}
