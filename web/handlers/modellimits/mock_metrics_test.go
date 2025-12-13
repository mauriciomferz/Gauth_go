package modellimits

import (
	"sync/atomic"
	"time"
)

type mockMetrics struct {
	unknown                 uint64
	limitExceeded           uint64
	outputLimitExceeded     uint64
	rateLimitExceeded       uint64
	userInputLimitExceeded  uint64
	userOutputLimitExceeded uint64
	userRateLimitExceeded   uint64
	surge                   uint64
	decisions               []string
}

func (m *mockMetrics) IncModelUnknown() { atomic.AddUint64(&m.unknown, 1) }

func (m *mockMetrics) IncModelLimitExceeded() { atomic.AddUint64(&m.limitExceeded, 1) }

func (m *mockMetrics) IncModelOutputLimitExceeded() { atomic.AddUint64(&m.outputLimitExceeded, 1) }

func (m *mockMetrics) IncModelRateLimitExceeded() { atomic.AddUint64(&m.rateLimitExceeded, 1) }

func (m *mockMetrics) IncModelUserInputLimitExceeded() {
	atomic.AddUint64(&m.userInputLimitExceeded, 1)
}

func (m *mockMetrics) IncModelUserOutputLimitExceeded() {
	atomic.AddUint64(&m.userOutputLimitExceeded, 1)
}

func (m *mockMetrics) IncModelUserRateLimitExceeded() {
	atomic.AddUint64(&m.userRateLimitExceeded, 1)
}

func (m *mockMetrics) IncModelLimitSurge() { atomic.AddUint64(&m.surge, 1) }

func (m *mockMetrics) RecordDecision(kind, id, result string, d time.Duration) {
	m.decisions = append(m.decisions, kind+"|"+id+"|"+result)
}
