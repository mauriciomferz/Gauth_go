package authz

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusHandler returns an http.HandlerFunc exposing MemoryAuthorizer metrics
// in Prometheus text exposition format without external dependencies.
// Gauge semantics used for counters (monotonically increasing) for simplicity; a real
// implementation may differentiate types.
//
// Example usage:
//
//	ma := NewMemoryAuthorizer()
//	http.HandleFunc("/metrics", authz.PrometheusHandler(ma))
//	http.ListenAndServe(":8080", nil)
func PrometheusHandler(ma *MemoryAuthorizer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap := ma.GetMetricsSnapshot()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		var b strings.Builder
		writeLine := func(format string, args ...interface{}) {
			fmt.Fprintf(&b, format+"\n", args...)
		}
		writeLine("# HELP authz_decisions_total Total authorization decisions made")
		writeLine("# TYPE authz_decisions_total counter")
		writeLine("authz_decisions_total %d", snap.Decisions)
		writeLine("# HELP authz_cache_hits_total Total cache hits")
		writeLine("# TYPE authz_cache_hits_total counter")
		writeLine("authz_cache_hits_total %d", snap.CacheHits)
		writeLine("# HELP authz_cache_misses_total Total cache misses")
		writeLine("# TYPE authz_cache_misses_total counter")
		writeLine("authz_cache_misses_total %d", snap.CacheMisses)
		writeLine("# HELP authz_policy_reload_total Policy reloads detected")
		writeLine("# TYPE authz_policy_reload_total counter")
		writeLine("authz_policy_reload_total %d", snap.Reloads)
		writeLine("# HELP authz_latency_average_nanoseconds Average decision latency (nanoseconds)")
		writeLine("# TYPE authz_latency_average_nanoseconds gauge")
		writeLine("authz_latency_average_nanoseconds %.0f", snap.AvgLatencyNs)
		writeLine("# HELP authz_latency_p99_nanoseconds Approximate P99 decision latency (nanoseconds)")
		writeLine("# TYPE authz_latency_p99_nanoseconds gauge")
		writeLine("authz_latency_p99_nanoseconds %.0f", snap.P99LatencyNs)
		writeLine("# HELP authz_policy_conflicts_total Decisions with simultaneous allow+deny matches")
		writeLine("# TYPE authz_policy_conflicts_total counter")
		writeLine("authz_policy_conflicts_total %d", snap.Conflicts)
		writeLine("# HELP authz_regex_compiles_total Successful regex pattern compilations (cached)")
		writeLine("# TYPE authz_regex_compiles_total counter")
		writeLine("authz_regex_compiles_total %d", snap.RegexCompiles)
		writeLine("# HELP authz_regex_compile_errors_total Failed regex pattern compilations")
		writeLine("# TYPE authz_regex_compile_errors_total counter")
		writeLine("authz_regex_compile_errors_total %d", snap.RegexCompileErrors)
		writeLine("# HELP authz_regex_cache_size Number of cached compiled regex patterns")
		writeLine("# TYPE authz_regex_cache_size gauge")
		writeLine("authz_regex_cache_size %d", snap.RegexCacheSize)
		writeLine("# HELP authz_regex_evictions_total Total regex cache evictions (TTL or capacity)")
		writeLine("# TYPE authz_regex_evictions_total counter")
		writeLine("authz_regex_evictions_total %d", snap.RegexEvictions)
		writeLine("# HELP authz_regex_matches_total Total successful regex matches evaluated")
		writeLine("# TYPE authz_regex_matches_total counter")
		writeLine("authz_regex_matches_total %d", snap.RegexMatches)
		writeLine("# HELP authz_latency_bucket Count of decisions with latency <= bucket upper bound (nanoseconds)")
		writeLine("# TYPE authz_latency_bucket counter")
		// Deterministic ordering of histogram buckets for stable output
		var uppers []int64
		for upper := range snap.LatencyHistogram {
			uppers = append(uppers, upper)
		}
		sort.Slice(uppers, func(i, j int) bool { return uppers[i] < uppers[j] })
		for _, upper := range uppers {
			writeLine("authz_latency_bucket{le=\"%d\"} %d", upper, snap.LatencyHistogram[upper])
		}
		if _, err := w.Write([]byte(b.String())); err != nil {
			fmt.Fprintf(w, "# ERROR write metrics: %v\n", err) // nolint:errcheck
		}
	}
}

// PrometheusMetricsProvider implements the authorizer's metrics interface
// using the official Prometheus client library with labeled vectors.
type PrometheusMetricsProvider struct {
	decisions   *prometheus.CounterVec
	latency     *prometheus.HistogramVec
	obligations *prometheus.CounterVec
	obLatency   prometheus.Histogram
}

// NewPrometheusMetricsProvider creates a new provider registering metrics in the default registry (or specified one).
func NewPrometheusMetricsProvider(namespace string) *PrometheusMetricsProvider {
	if namespace == "" {
		namespace = "authz"
	}
	// Use promauto to auto-register with default registry
	return &PrometheusMetricsProvider{
		decisions: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decisions_total",
			Help:      "Total authorization decisions by action and outcome",
		}, []string{"action", "resource_type", "outcome"}),
		latency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "decision_duration_seconds",
			Help:      "Authorization decision latency in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"action"}),
		obligations: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "obligations_total",
			Help:      "Total obligations executed by status",
		}, []string{"status"}), // executed, failed, mandatory_failed
		obLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "obligation_duration_seconds",
			Help:      "Obligation execution latency in seconds",
			Buckets:   prometheus.DefBuckets,
		}),
	}
}

func (p *PrometheusMetricsProvider) IncObligationsExecuted() {
	p.obligations.WithLabelValues("executed").Inc()
}

func (p *PrometheusMetricsProvider) IncObligationsFailed() {
	p.obligations.WithLabelValues("failed").Inc()
}

func (p *PrometheusMetricsProvider) IncMandatoryObligationFailures() {
	p.obligations.WithLabelValues("mandatory_failed").Inc()
}

func (p *PrometheusMetricsProvider) ObserveObligationLatency(d time.Duration) {
	p.obLatency.Observe(d.Seconds())
}

func (p *PrometheusMetricsProvider) RecordDecision(action, resource, decision string, duration time.Duration) {
	// resource label is tricky if high cardinality (IDs), better use type if available, else literal "resource"
	// To avoid high cardinality explosion, we default resource to a static string or type if available
	// For now we trust the caller passed a safe string or use "resource"
	p.decisions.WithLabelValues(action, resource, decision).Inc()
	p.latency.WithLabelValues(action).Observe(duration.Seconds())
}
