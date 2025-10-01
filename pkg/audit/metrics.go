package audit

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds audit metrics collectors
type Metrics struct {
	entriesTotal   *prometheus.CounterVec
	entryLatency   *prometheus.HistogramVec
	storageErrors  *prometheus.CounterVec
	batchSize      prometheus.Gauge
	storageLatency *prometheus.HistogramVec
	chainLength    *prometheus.HistogramVec
}

// NewMetrics creates new audit metrics collectors
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		entriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "audit_entries_total",
				Help:      "Total number of audit entries by type and result",
			},
			[]string{"type", "action", "result"},
		),
		entryLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "audit_entry_duration_seconds",
				Help:      "Duration of audit entry operations",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
		storageErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "audit_storage_errors_total",
				Help:      "Total number of storage errors by operation",
			},
			[]string{"operation", "error"},
		),
		batchSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "audit_batch_size",
				Help:      "Current size of the audit entry batch",
			},
		),
		storageLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "audit_storage_duration_seconds",
				Help:      "Duration of storage operations",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"operation", "storage"},
		),
		chainLength: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "audit_chain_length",
				Help:      "Length of audit chains",
				Buckets:   []float64{1, 2, 5, 10, 20, 50, 100},
			},
			[]string{"chain_type"},
		),
	}

	prometheus.MustRegister(
		m.entriesTotal,
		m.entryLatency,
		m.storageErrors,
		m.batchSize,
		m.storageLatency,
		m.chainLength,
	)

	return m
}

// ObserveEntry records metrics for an audit entry
func (m *Metrics) ObserveEntry(entry *Entry, duration time.Duration) {
	m.entriesTotal.WithLabelValues(entry.Type, entry.Action, entry.Result).Inc()
	m.entryLatency.WithLabelValues("process").Observe(duration.Seconds())
}

// ObserveStorageOperation records metrics for a storage operation
func (m *Metrics) ObserveStorageOperation(operation, storage string, duration time.Duration) {
	m.storageLatency.WithLabelValues(operation, storage).Observe(duration.Seconds())
}

// ObserveStorageError records a storage error
func (m *Metrics) ObserveStorageError(operation, errType string) {
	m.storageErrors.WithLabelValues(operation, errType).Inc()
}

// SetBatchSize updates the current batch size
func (m *Metrics) SetBatchSize(size int) {
	m.batchSize.Set(float64(size))
}

// ObserveChainLength records the length of an audit chain
func (m *Metrics) ObserveChainLength(chainType string, length int) {
	m.chainLength.WithLabelValues(chainType).Observe(float64(length))
}
