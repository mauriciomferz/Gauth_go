package prometheus

import (
	"net/http"

	"github.com/mauriciomferz/AgentAuth/internal/monitoring"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Exporter wraps monitoring with Prometheus export capabilities
type Exporter struct {
	metrics *monitoring.DefaultMetricsCollector
	handler http.Handler
}

// NewExporter creates a new Prometheus exporter for the given metrics collector
func NewExporter(metrics *monitoring.DefaultMetricsCollector) *Exporter {
	handler := promhttp.Handler()
	return &Exporter{
		metrics: metrics,
		handler: handler,
	}
}

// Handler returns the HTTP handler for Prometheus metrics endpoint
func (e *Exporter) Handler() http.Handler {
	return e.handler
}

// Metrics returns the underlying metrics collector
func (e *Exporter) Metrics() *monitoring.DefaultMetricsCollector {
	return e.metrics
}

// ServeHTTP implements http.Handler interface
func (e *Exporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	e.handler.ServeHTTP(w, r)
}

// RegisterCollector registers a Prometheus collector
func (e *Exporter) RegisterCollector(collector prometheus.Collector) error {
	return prometheus.Register(collector)
}

// UnregisterCollector unregisters a Prometheus collector
func (e *Exporter) UnregisterCollector(collector prometheus.Collector) bool {
	return prometheus.Unregister(collector)
}
