// Package oidc provides Prometheus metrics instrumentation for OIDC operations.
package oidc

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	statusSuccess = "success"
)

// MetricsCollector provides Prometheus metrics for OIDC operations.
type MetricsCollector struct {
	// Request metrics
	requestsTotal     *prometheus.CounterVec
	requestDuration   *prometheus.HistogramVec
	activeConnections prometheus.Gauge

	// Token metrics
	tokensIssuedTotal       *prometheus.CounterVec
	tokensRefreshedTotal    prometheus.Counter
	tokensRevokedTotal      prometheus.Counter
	tokensIntrospectedTotal prometheus.Counter

	// Device flow metrics
	deviceCodesCreatedTotal  prometheus.Counter
	deviceCodesApprovedTotal prometheus.Counter
	deviceCodesDeniedTotal   prometheus.Counter
	deviceCodesExpiredTotal  prometheus.Counter
	devicePollsTotal         *prometheus.CounterVec

	// PAR metrics
	parRequestsCreatedTotal prometheus.Counter
	parRequestsUsedTotal    prometheus.Counter
	parRequestsExpiredTotal prometheus.Counter

	// Rate limit metrics
	rateLimitRequestsTotal *prometheus.CounterVec

	// Storage metrics
	storageOperationsTotal *prometheus.CounterVec
	storageDuration        *prometheus.HistogramVec

	// Error metrics
	errorsTotal *prometheus.CounterVec

	// Cache metrics
	cacheHitsTotal   *prometheus.CounterVec
	cacheMissesTotal *prometheus.CounterVec
}

// NewMetricsCollector creates a new Prometheus metrics collector with all OIDC metrics.
func NewMetricsCollector(namespace string) *MetricsCollector {
	if namespace == "" {
		namespace = "oidc"
	}

	return &MetricsCollector{
		// Request metrics
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "requests_total",
				Help:      "Total number of HTTP requests by endpoint, method, and status code.",
			},
			[]string{"endpoint", "method", "status"},
		),
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "request_duration_seconds",
				Help:      "HTTP request latencies in seconds.",
				Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"endpoint", "method"},
		),
		activeConnections: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Name:      "active_connections",
				Help:      "Number of active HTTP connections.",
			},
		),

		// Token metrics
		tokensIssuedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tokens_issued_total",
				Help:      "Total number of tokens issued by grant type.",
			},
			[]string{"grant_type"},
		),
		tokensRefreshedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tokens_refreshed_total",
				Help:      "Total number of tokens refreshed.",
			},
		),
		tokensRevokedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tokens_revoked_total",
				Help:      "Total number of tokens revoked.",
			},
		),
		tokensIntrospectedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "tokens_introspected_total",
				Help:      "Total number of token introspection requests.",
			},
		),

		// Device flow metrics
		deviceCodesCreatedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "device_codes_created_total",
				Help:      "Total number of device authorization codes created.",
			},
		),
		deviceCodesApprovedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "device_codes_approved_total",
				Help:      "Total number of device authorization codes approved by users.",
			},
		),
		deviceCodesDeniedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "device_codes_denied_total",
				Help:      "Total number of device authorization codes denied by users.",
			},
		),
		deviceCodesExpiredTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "device_codes_expired_total",
				Help:      "Total number of device authorization codes expired.",
			},
		),
		devicePollsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "device_polls_total",
				Help:      "Total number of device code polling attempts by status.",
			},
			[]string{"status"}, // pending, approved, denied, expired, slow_down
		),

		// PAR metrics
		parRequestsCreatedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "par_requests_created_total",
				Help:      "Total number of pushed authorization requests created.",
			},
		),
		parRequestsUsedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "par_requests_used_total",
				Help:      "Total number of pushed authorization requests used in authorization flow.",
			},
		),
		parRequestsExpiredTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "par_requests_expired_total",
				Help:      "Total number of pushed authorization requests expired.",
			},
		),

		// Rate limit metrics
		rateLimitRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "rate_limit_requests_total",
				Help:      "Total number of requests checked by rate limiter, by endpoint and result.",
			},
			[]string{"endpoint", "result"}, // allowed, denied
		),

		// Storage metrics
		storageOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "storage_operations_total",
				Help:      "Total number of storage operations by operation type, backend, and status.",
			},
			[]string{"operation", "backend", "status"}, // success, error
		),
		storageDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "storage_duration_seconds",
				Help:      "Storage operation latencies in seconds.",
				Buckets:   []float64{.0001, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"operation", "backend"},
		),

		// Error metrics
		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "errors_total",
				Help:      "Total number of errors by error type and endpoint.",
			},
			[]string{"error_type", "endpoint"},
		),

		// Cache metrics
		cacheHitsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_hits_total",
				Help:      "Total number of cache hits by cache type.",
			},
			[]string{"cache_type"},
		),
		cacheMissesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "cache_misses_total",
				Help:      "Total number of cache misses by cache type.",
			},
			[]string{"cache_type"},
		),
	}
}

// RecordRequest records an HTTP request metric.
func (m *MetricsCollector) RecordRequest(endpoint, method, status string, duration time.Duration) {
	m.requestsTotal.WithLabelValues(endpoint, method, status).Inc()
	m.requestDuration.WithLabelValues(endpoint, method).Observe(duration.Seconds())
}

// IncActiveConnections increments the active connections gauge.
func (m *MetricsCollector) IncActiveConnections() {
	m.activeConnections.Inc()
}

// DecActiveConnections decrements the active connections gauge.
func (m *MetricsCollector) DecActiveConnections() {
	m.activeConnections.Dec()
}

// RecordTokenIssued records a token issuance event.
func (m *MetricsCollector) RecordTokenIssued(grantType string) {
	m.tokensIssuedTotal.WithLabelValues(grantType).Inc()
}

// RecordTokenRefreshed records a token refresh event.
func (m *MetricsCollector) RecordTokenRefreshed() {
	m.tokensRefreshedTotal.Inc()
}

// RecordTokenRevoked records a token revocation event.
func (m *MetricsCollector) RecordTokenRevoked() {
	m.tokensRevokedTotal.Inc()
}

// RecordTokenIntrospected records a token introspection event.
func (m *MetricsCollector) RecordTokenIntrospected() {
	m.tokensIntrospectedTotal.Inc()
}

// RecordDeviceCodeCreated records a device code creation event.
func (m *MetricsCollector) RecordDeviceCodeCreated() {
	m.deviceCodesCreatedTotal.Inc()
}

// RecordDeviceCodeApproved records a device code approval event.
func (m *MetricsCollector) RecordDeviceCodeApproved() {
	m.deviceCodesApprovedTotal.Inc()
}

// RecordDeviceCodeDenied records a device code denial event.
func (m *MetricsCollector) RecordDeviceCodeDenied() {
	m.deviceCodesDeniedTotal.Inc()
}

// RecordDeviceCodeExpired records a device code expiration event.
func (m *MetricsCollector) RecordDeviceCodeExpired() {
	m.deviceCodesExpiredTotal.Inc()
}

// RecordDevicePoll records a device code polling attempt.
func (m *MetricsCollector) RecordDevicePoll(status string) {
	m.devicePollsTotal.WithLabelValues(status).Inc()
}

// RecordPARRequestCreated records a PAR request creation event.
func (m *MetricsCollector) RecordPARRequestCreated() {
	m.parRequestsCreatedTotal.Inc()
}

// RecordPARRequestUsed records a PAR request usage event.
func (m *MetricsCollector) RecordPARRequestUsed() {
	m.parRequestsUsedTotal.Inc()
}

// RecordPARRequestExpired records a PAR request expiration event.
func (m *MetricsCollector) RecordPARRequestExpired() {
	m.parRequestsExpiredTotal.Inc()
}

// RecordRateLimitCheck records a rate limit check result.
func (m *MetricsCollector) RecordRateLimitCheck(endpoint, result string) {
	m.rateLimitRequestsTotal.WithLabelValues(endpoint, result).Inc()
}

// RecordStorageOperation records a storage operation with timing.
func (m *MetricsCollector) RecordStorageOperation(operation, backend, status string, duration time.Duration) {
	m.storageOperationsTotal.WithLabelValues(operation, backend, status).Inc()
	m.storageDuration.WithLabelValues(operation, backend).Observe(duration.Seconds())
}

// RecordError records an error event.
func (m *MetricsCollector) RecordError(errorType, endpoint string) {
	m.errorsTotal.WithLabelValues(errorType, endpoint).Inc()
}

// RecordCacheHit records a cache hit.
func (m *MetricsCollector) RecordCacheHit(cacheType string) {
	m.cacheHitsTotal.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss records a cache miss.
func (m *MetricsCollector) RecordCacheMiss(cacheType string) {
	m.cacheMissesTotal.WithLabelValues(cacheType).Inc()
}

// MetricsMiddleware returns an HTTP middleware that records request metrics.
func MetricsMiddleware(metrics *MetricsCollector, endpointName string) func(next func()) func() {
	return func(next func()) func() {
		return func() {
			start := time.Now()
			metrics.IncActiveConnections()
			defer metrics.DecActiveConnections()

			// Execute the handler
			next()

			// Record metrics (status would be extracted from response)
			duration := time.Since(start)
			// Note: In a real HTTP middleware, you would extract method and status from the request/response
			metrics.RecordRequest(endpointName, "POST", "200", duration)
		}
	}
}

// StorageMetricsWrapper wraps a StorageBackend with metrics instrumentation.
type StorageMetricsWrapper struct {
	backend     StorageBackend
	metrics     *MetricsCollector
	backendName string
}

// NewStorageMetricsWrapper creates a new storage backend wrapper with metrics.
func NewStorageMetricsWrapper(backend StorageBackend, metrics *MetricsCollector, backendName string) *StorageMetricsWrapper {
	return &StorageMetricsWrapper{
		backend:     backend,
		metrics:     metrics,
		backendName: backendName,
	}
}

// StoreRefreshToken stores a refresh token with metrics.
func (s *StorageMetricsWrapper) StoreRefreshToken(ctx context.Context, entry *RefreshTokenEntry) error {
	start := time.Now()
	err := s.backend.StoreRefreshToken(ctx, entry)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("store_refresh_token", s.backendName, status, time.Since(start))
	return err
}

// GetRefreshToken retrieves a refresh token with metrics.
func (s *StorageMetricsWrapper) GetRefreshToken(ctx context.Context, tokenHash string) (*RefreshTokenEntry, error) {
	start := time.Now()
	entry, err := s.backend.GetRefreshToken(ctx, tokenHash)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("get_refresh_token", s.backendName, status, time.Since(start))
	return entry, err
}

// DeleteRefreshToken deletes a refresh token with metrics.
func (s *StorageMetricsWrapper) DeleteRefreshToken(ctx context.Context, tokenHash string) error {
	start := time.Now()
	err := s.backend.DeleteRefreshToken(ctx, tokenHash)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("delete_refresh_token", s.backendName, status, time.Since(start))
	return err
}

// ListRefreshTokensByUser lists refresh tokens by user with metrics.
func (s *StorageMetricsWrapper) ListRefreshTokensByUser(ctx context.Context, userID string) ([]*RefreshTokenEntry, error) {
	start := time.Now()
	entries, err := s.backend.ListRefreshTokensByUser(ctx, userID)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("list_refresh_tokens_by_user", s.backendName, status, time.Since(start))
	return entries, err
}

// ListRefreshTokensByClient lists refresh tokens by client with metrics.
func (s *StorageMetricsWrapper) ListRefreshTokensByClient(ctx context.Context, clientID string) ([]*RefreshTokenEntry, error) {
	start := time.Now()
	entries, err := s.backend.ListRefreshTokensByClient(ctx, clientID)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("list_refresh_tokens_by_client", s.backendName, status, time.Since(start))
	return entries, err
}

// CleanupExpiredRefreshTokens deletes expired refresh tokens with metrics.
func (s *StorageMetricsWrapper) CleanupExpiredRefreshTokens(ctx context.Context) (int, error) {
	start := time.Now()
	count, err := s.backend.CleanupExpiredRefreshTokens(ctx)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("cleanup_expired_refresh_tokens", s.backendName, status, time.Since(start))
	return count, err
}

// StoreRevokedToken stores a revoked token with metrics.
func (s *StorageMetricsWrapper) StoreRevokedToken(ctx context.Context, entry *RevokedTokenEntry) error {
	start := time.Now()
	err := s.backend.StoreRevokedToken(ctx, entry)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("store_revoked_token", s.backendName, status, time.Since(start))
	return err
}

// IsTokenRevoked checks if a token is revoked with metrics.
func (s *StorageMetricsWrapper) IsTokenRevoked(ctx context.Context, tokenHash string) (bool, error) {
	start := time.Now()
	revoked, err := s.backend.IsTokenRevoked(ctx, tokenHash)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("is_token_revoked", s.backendName, status, time.Since(start))
	return revoked, err
}

// CleanupExpiredRevocations deletes expired revoked tokens with metrics.
func (s *StorageMetricsWrapper) CleanupExpiredRevocations(ctx context.Context) (int, error) {
	start := time.Now()
	count, err := s.backend.CleanupExpiredRevocations(ctx)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("cleanup_expired_revocations", s.backendName, status, time.Since(start))
	return count, err
}

// StoreDeviceCode stores a device code with metrics.
func (s *StorageMetricsWrapper) StoreDeviceCode(ctx context.Context, entry *DeviceCodeEntry) error {
	start := time.Now()
	err := s.backend.StoreDeviceCode(ctx, entry)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("store_device_code", s.backendName, status, time.Since(start))
	return err
}

// GetDeviceCodeByDeviceCode retrieves a device code by device code with metrics.
func (s *StorageMetricsWrapper) GetDeviceCodeByDeviceCode(ctx context.Context, deviceCode string) (*DeviceCodeEntry, error) {
	start := time.Now()
	entry, err := s.backend.GetDeviceCodeByDeviceCode(ctx, deviceCode)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("get_device_code_by_device_code", s.backendName, status, time.Since(start))
	return entry, err
}

// GetDeviceCodeByUserCode retrieves a device code by user code with metrics.
func (s *StorageMetricsWrapper) GetDeviceCodeByUserCode(ctx context.Context, userCode string) (*DeviceCodeEntry, error) {
	start := time.Now()
	entry, err := s.backend.GetDeviceCodeByUserCode(ctx, userCode)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("get_device_code_by_user_code", s.backendName, status, time.Since(start))
	return entry, err
}

// UpdateDeviceCodeStatus updates a device code status with metrics.
func (s *StorageMetricsWrapper) UpdateDeviceCodeStatus(ctx context.Context, deviceCode string, entry *DeviceCodeEntry) error {
	start := time.Now()
	err := s.backend.UpdateDeviceCodeStatus(ctx, deviceCode, entry)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("update_device_code_status", s.backendName, status, time.Since(start))
	return err
}

// DeleteDeviceCode deletes a device code with metrics.
func (s *StorageMetricsWrapper) DeleteDeviceCode(ctx context.Context, deviceCode string) error {
	start := time.Now()
	err := s.backend.DeleteDeviceCode(ctx, deviceCode)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("delete_device_code", s.backendName, status, time.Since(start))
	return err
}

// CleanupExpiredDeviceCodes deletes expired device codes with metrics.
func (s *StorageMetricsWrapper) CleanupExpiredDeviceCodes(ctx context.Context) (int, error) {
	start := time.Now()
	count, err := s.backend.CleanupExpiredDeviceCodes(ctx)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("cleanup_expired_device_codes", s.backendName, status, time.Since(start))
	return count, err
}

// StorePARRequest stores a PAR request with metrics.
func (s *StorageMetricsWrapper) StorePARRequest(ctx context.Context, requestURI string, entry *RequestURIEntry) error {
	start := time.Now()
	err := s.backend.StorePARRequest(ctx, requestURI, entry)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("store_par_request", s.backendName, status, time.Since(start))
	return err
}

// GetPARRequest retrieves a PAR request with metrics.
func (s *StorageMetricsWrapper) GetPARRequest(ctx context.Context, requestURI string) (*RequestURIEntry, error) {
	start := time.Now()
	entry, err := s.backend.GetPARRequest(ctx, requestURI)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("get_par_request", s.backendName, status, time.Since(start))
	return entry, err
}

// DeletePARRequest deletes a PAR request with metrics.
func (s *StorageMetricsWrapper) DeletePARRequest(ctx context.Context, requestURI string) error {
	start := time.Now()
	err := s.backend.DeletePARRequest(ctx, requestURI)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("delete_par_request", s.backendName, status, time.Since(start))
	return err
}

// MarkPARRequestUsed marks a PAR request as used with metrics.
func (s *StorageMetricsWrapper) MarkPARRequestUsed(ctx context.Context, requestURI string) error {
	start := time.Now()
	err := s.backend.MarkPARRequestUsed(ctx, requestURI)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("mark_par_request_used", s.backendName, status, time.Since(start))
	return err
}

// CleanupExpiredPARRequests deletes expired PAR requests with metrics.
func (s *StorageMetricsWrapper) CleanupExpiredPARRequests(ctx context.Context) (int, error) {
	start := time.Now()
	count, err := s.backend.CleanupExpiredPARRequests(ctx)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("cleanup_expired_par_requests", s.backendName, status, time.Since(start))
	return count, err
}

// Ping checks storage backend connectivity with metrics.
func (s *StorageMetricsWrapper) Ping(ctx context.Context) error {
	start := time.Now()
	err := s.backend.Ping(ctx)
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("ping", s.backendName, status, time.Since(start))
	return err
}

// Close closes the storage backend with metrics.
func (s *StorageMetricsWrapper) Close() error {
	start := time.Now()
	err := s.backend.Close()
	status := statusSuccess
	if err != nil {
		status = string(LogLevelError)
	}
	s.metrics.RecordStorageOperation("close", s.backendName, status, time.Since(start))
	return err
}
