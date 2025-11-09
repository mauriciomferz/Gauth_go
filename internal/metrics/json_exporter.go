package metrics

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// JSONMetricsExporter exports metrics in JSON format for interoperability
type JSONMetricsExporter struct {
	mu              sync.RWMutex
	counters        map[string]int64
	gauges          map[string]float64
	histograms      map[string]*HistogramData
	labels          map[string]map[string]string
	metadata        MetricsMetadata
	reasonTaxonomy  map[string]ReasonCategory
	includeReasons  bool
	includeMetadata bool
}

// MetricsMetadata provides context about the metrics collection
type MetricsMetadata struct {
	ServiceName    string    `json:"service_name"`
	ServiceVersion string    `json:"service_version"`
	Hostname       string    `json:"hostname"`
	Timestamp      time.Time `json:"timestamp"`
	CollectionTime time.Time `json:"collection_time"`
	Uptime         int64     `json:"uptime_seconds"`
}

// ReasonCategory categorizes authorization decision reasons
type ReasonCategory struct {
	Category    string   `json:"category"`
	Description string   `json:"description"`
	Examples    []string `json:"examples,omitempty"`
}

// HistogramData represents histogram metric data
type HistogramData struct {
	Count   int64              `json:"count"`
	Sum     float64            `json:"sum"`
	Buckets map[string]int64   `json:"buckets,omitempty"`
	P50     float64            `json:"p50,omitempty"`
	P95     float64            `json:"p95,omitempty"`
	P99     float64            `json:"p99,omitempty"`
	Min     float64            `json:"min,omitempty"`
	Max     float64            `json:"max,omitempty"`
}

// MetricEntry represents a single metric in JSON format
type MetricEntry struct {
	Name        string             `json:"name"`
	Type        string             `json:"type"` // counter, gauge, histogram
	Value       interface{}        `json:"value"`
	Labels      map[string]string  `json:"labels,omitempty"`
	Description string             `json:"description,omitempty"`
	Unit        string             `json:"unit,omitempty"`
	Timestamp   time.Time          `json:"timestamp"`
}

// JSONMetricsResponse is the complete JSON response structure
type JSONMetricsResponse struct {
	Metadata MetricsMetadata         `json:"metadata"`
	Metrics  []MetricEntry           `json:"metrics"`
	Reasons  map[string]ReasonCategory `json:"reason_taxonomy,omitempty"`
}

// NewJSONMetricsExporter creates a new JSON metrics exporter
func NewJSONMetricsExporter(serviceName, serviceVersion, hostname string) *JSONMetricsExporter {
	return &JSONMetricsExporter{
		counters:        make(map[string]int64),
		gauges:          make(map[string]float64),
		histograms:      make(map[string]*HistogramData),
		labels:          make(map[string]map[string]string),
		reasonTaxonomy:  initializeReasonTaxonomy(),
		includeReasons:  true,
		includeMetadata: true,
		metadata: MetricsMetadata{
			ServiceName:    serviceName,
			ServiceVersion: serviceVersion,
			Hostname:       hostname,
		},
	}
}

// RecordCounter increments a counter metric
func (e *JSONMetricsExporter) RecordCounter(name string, value int64, labels map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	key := e.metricKey(name, labels)
	e.counters[key] += value
	if labels != nil && len(labels) > 0 {
		e.labels[key] = labels
	}
}

// RecordGauge sets a gauge metric value
func (e *JSONMetricsExporter) RecordGauge(name string, value float64, labels map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	key := e.metricKey(name, labels)
	e.gauges[key] = value
	if labels != nil && len(labels) > 0 {
		e.labels[key] = labels
	}
}

// RecordHistogram records a histogram observation
func (e *JSONMetricsExporter) RecordHistogram(name string, value float64, labels map[string]string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	key := e.metricKey(name, labels)
	hist, exists := e.histograms[key]
	if !exists {
		hist = &HistogramData{
			Buckets: make(map[string]int64),
			Min:     value,
			Max:     value,
		}
		e.histograms[key] = hist
	}
	
	hist.Count++
	hist.Sum += value
	
	// Update min/max
	if value < hist.Min {
		hist.Min = value
	}
	if value > hist.Max {
		hist.Max = value
	}
	
	// Update buckets
	e.updateHistogramBuckets(hist, value)
	
	if labels != nil && len(labels) > 0 {
		e.labels[key] = labels
	}
}

// ExportJSON exports all metrics in JSON format
func (e *JSONMetricsExporter) ExportJSON() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	now := time.Now()
	e.metadata.Timestamp = now
	e.metadata.CollectionTime = now
	
	metrics := []MetricEntry{}
	
	// Export counters
	for key, value := range e.counters {
		name, labels := e.parseMetricKey(key)
		metrics = append(metrics, MetricEntry{
			Name:      name,
			Type:      "counter",
			Value:     value,
			Labels:    labels,
			Timestamp: now,
		})
	}
	
	// Export gauges
	for key, value := range e.gauges {
		name, labels := e.parseMetricKey(key)
		metrics = append(metrics, MetricEntry{
			Name:      name,
			Type:      "gauge",
			Value:     value,
			Labels:    labels,
			Timestamp: now,
		})
	}
	
	// Export histograms
	for key, hist := range e.histograms {
		name, labels := e.parseMetricKey(key)
		metrics = append(metrics, MetricEntry{
			Name:      name,
			Type:      "histogram",
			Value:     hist,
			Labels:    labels,
			Timestamp: now,
		})
	}
	
	response := JSONMetricsResponse{
		Metadata: e.metadata,
		Metrics:  metrics,
	}
	
	if e.includeReasons {
		response.Reasons = e.reasonTaxonomy
	}
	
	return json.MarshalIndent(response, "", "  ")
}

// ExportJSONCompact exports metrics in compact JSON (no indentation)
func (e *JSONMetricsExporter) ExportJSONCompact() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	now := time.Now()
	e.metadata.Timestamp = now
	e.metadata.CollectionTime = now
	
	metrics := []MetricEntry{}
	
	// Export counters
	for key, value := range e.counters {
		name, labels := e.parseMetricKey(key)
		metrics = append(metrics, MetricEntry{
			Name:      name,
			Type:      "counter",
			Value:     value,
			Labels:    labels,
			Timestamp: now,
		})
	}
	
	// Export gauges
	for key, value := range e.gauges {
		name, labels := e.parseMetricKey(key)
		metrics = append(metrics, MetricEntry{
			Name:      name,
			Type:      "gauge",
			Value:     value,
			Labels:    labels,
			Timestamp: now,
		})
	}
	
	// Export histograms
	for key, hist := range e.histograms {
		name, labels := e.parseMetricKey(key)
		metrics = append(metrics, MetricEntry{
			Name:      name,
			Type:      "histogram",
			Value:     hist,
			Labels:    labels,
			Timestamp: now,
		})
	}
	
	response := JSONMetricsResponse{
		Metadata: e.metadata,
		Metrics:  metrics,
	}
	
	if e.includeReasons {
		response.Reasons = e.reasonTaxonomy
	}
	
	return json.Marshal(response)
}

// SetIncludeReasons controls whether reason taxonomy is included in export
func (e *JSONMetricsExporter) SetIncludeReasons(include bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.includeReasons = include
}

// SetIncludeMetadata controls whether metadata is included in export
func (e *JSONMetricsExporter) SetIncludeMetadata(include bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.includeMetadata = include
}

// Reset clears all metric data
func (e *JSONMetricsExporter) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	e.counters = make(map[string]int64)
	e.gauges = make(map[string]float64)
	e.histograms = make(map[string]*HistogramData)
	e.labels = make(map[string]map[string]string)
}

// GetMetricCount returns the total number of metrics
func (e *JSONMetricsExporter) GetMetricCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.counters) + len(e.gauges) + len(e.histograms)
}

// metricKey generates a unique key for a metric with labels
func (e *JSONMetricsExporter) metricKey(name string, labels map[string]string) string {
	if labels == nil || len(labels) == 0 {
		return name
	}
	
	// Create deterministic key from labels by sorting keys
	// This ensures consistent ordering for map iteration
	var labelPairs []string
	for k, v := range labels {
		labelPairs = append(labelPairs, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Sort for consistency
	for i := 0; i < len(labelPairs)-1; i++ {
		for j := i + 1; j < len(labelPairs); j++ {
			if labelPairs[i] > labelPairs[j] {
				labelPairs[i], labelPairs[j] = labelPairs[j], labelPairs[i]
			}
		}
	}
	
	key := name
	for _, pair := range labelPairs {
		key += ":" + pair
	}
	return key
}

// parseMetricKey extracts name and labels from a metric key
func (e *JSONMetricsExporter) parseMetricKey(key string) (string, map[string]string) {
	// Simple implementation - just return stored labels
	if storedLabels, exists := e.labels[key]; exists {
		// Extract name (everything before first colon if present)
		name := key
		for i, c := range key {
			if c == ':' {
				name = key[:i]
				break
			}
		}
		return name, storedLabels
	}
	return key, nil
}

// updateHistogramBuckets updates histogram bucket counts
func (e *JSONMetricsExporter) updateHistogramBuckets(hist *HistogramData, value float64) {
	// Standard buckets: 0.001, 0.01, 0.1, 1, 10, 100, 1000, +Inf
	buckets := []float64{0.001, 0.01, 0.1, 1, 10, 100, 1000}
	
	for _, bucket := range buckets {
		if value <= bucket {
			key := fmt.Sprintf("%.3f", bucket)
			hist.Buckets[key]++
		}
	}
	// Always increment +Inf bucket
	hist.Buckets["+Inf"]++
}

// initializeReasonTaxonomy creates the standard reason taxonomy
func initializeReasonTaxonomy() map[string]ReasonCategory {
	return map[string]ReasonCategory{
		"policy_allow": {
			Category:    "allow",
			Description: "Authorization granted by policy evaluation",
			Examples:    []string{"policy_match", "explicit_grant", "role_authorized"},
		},
		"policy_deny": {
			Category:    "deny",
			Description: "Authorization denied by policy evaluation",
			Examples:    []string{"policy_mismatch", "explicit_deny", "role_unauthorized"},
		},
		"scope_violation": {
			Category:    "deny",
			Description: "Request exceeds authorized scope",
			Examples:    []string{"resource_out_of_scope", "action_not_permitted", "sector_mismatch"},
		},
		"temporal_violation": {
			Category:    "deny",
			Description: "Time-based constraint violation",
			Examples:    []string{"token_expired", "not_yet_valid", "outside_hours"},
		},
		"delegation_exceeded": {
			Category:    "deny",
			Description: "Delegation chain or depth limit exceeded",
			Examples:    []string{"max_depth_exceeded", "delegation_revoked", "chain_broken"},
		},
		"signature_invalid": {
			Category:    "deny",
			Description: "Cryptographic signature validation failed",
			Examples:    []string{"signature_mismatch", "key_not_found", "algorithm_unsupported"},
		},
		"replay_detected": {
			Category:    "deny",
			Description: "Token replay attack detected",
			Examples:    []string{"jti_reused", "nonce_duplicate", "clock_skew_excessive"},
		},
		"revocation": {
			Category:    "deny",
			Description: "Token or delegation has been revoked",
			Examples:    []string{"token_revoked", "delegation_revoked", "key_rotated"},
		},
		"capability_insufficient": {
			Category:    "deny",
			Description: "AI agent lacks required capability",
			Examples:    []string{"model_limit_exceeded", "capability_not_granted", "approval_required"},
		},
		"jurisdiction_violation": {
			Category:    "deny",
			Description: "Jurisdiction or compliance constraint violated",
			Examples:    []string{"region_restricted", "gdpr_violation", "cross_border_denied"},
		},
	}
}
