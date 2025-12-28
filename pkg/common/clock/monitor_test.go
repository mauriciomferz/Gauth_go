package clock

import (
	"errors"
	"testing"
	"time"

	"github.com/beevik/ntp"
	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/stretchr/testify/assert"
)

// MockMetrics for testing
type MockMetrics struct {
	metrics.Memory // Embed to satisfy interface cheaply, override what we need
	lastSkew       float64
	setCalled      bool
}

func (m *MockMetrics) SetSystemClockSkew(seconds float64) {
	m.lastSkew = seconds
	m.setCalled = true
}

func TestNewSystemClockMonitor_Defaults(t *testing.T) {
	m := NewSystemClockMonitor("", 0, 0, nil)
	assert.Equal(t, "pool.ntp.org", m.ntpServer)
	assert.Equal(t, 5*time.Minute, m.maxSkew)
	assert.Equal(t, 1*time.Hour, m.ntpInterval)
	assert.Equal(t, string(StatusHealthy), string(m.status))
}

func TestSystemClockMonitor_Check_Healthy(t *testing.T) {
	mockM := &MockMetrics{}
	m := NewSystemClockMonitor("test.pool", 5*time.Minute, 1*time.Hour, mockM)

	// Mock healthy response
	m.queryFn = func(server string) (*ntp.Response, error) {
		assert.Equal(t, "test.pool", server)
		return &ntp.Response{
			ClockOffset: 50 * time.Millisecond,
		}, nil
	}

	m.check()

	status, skew, err := m.Status()
	assert.Equal(t, string(StatusHealthy), status)
	assert.Equal(t, 50*time.Millisecond, skew)
	assert.NoError(t, err)

	assert.True(t, mockM.setCalled)
	assert.InDelta(t, 0.050, mockM.lastSkew, 0.0001)
}

func TestSystemClockMonitor_Check_Critical(t *testing.T) {
	mockM := &MockMetrics{}
	m := NewSystemClockMonitor("test.pool", 1*time.Second, 1*time.Hour, mockM)

	// Mock critical response (offset > maxSkew)
	m.queryFn = func(server string) (*ntp.Response, error) {
		return &ntp.Response{
			ClockOffset: 2 * time.Second,
		}, nil
	}

	m.check()

	status, skew, err := m.Status()
	assert.Equal(t, string(StatusCritical), status)
	assert.Equal(t, 2*time.Second, skew)
	assert.NoError(t, err)
}

func TestSystemClockMonitor_Check_Warning(t *testing.T) {
	mockM := &MockMetrics{}
	m := NewSystemClockMonitor("test.pool", 10*time.Second, 1*time.Hour, mockM)

	// Mock warning response (offset > maxSkew/2 but < maxSkew)
	m.queryFn = func(server string) (*ntp.Response, error) {
		return &ntp.Response{
			ClockOffset: 6 * time.Second,
		}, nil
	}

	m.check()

	status, _, _ := m.Status()
	assert.Equal(t, string(StatusWarning), status)
}

func TestSystemClockMonitor_Check_Error(t *testing.T) {
	m := NewSystemClockMonitor("test.pool", 5*time.Minute, 1*time.Hour, nil)

	// Mock error response
	m.queryFn = func(server string) (*ntp.Response, error) {
		return nil, errors.New("network timeout")
	}

	m.check()

	status, _, err := m.Status()
	// Status should remain what it was (default Healthy) or could be degraded if we implemented logic for it
	// Current implementation: if err, return. Status field not updated.
	assert.Equal(t, string(StatusHealthy), status)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network timeout")
}
