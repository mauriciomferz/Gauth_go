// Package oidc provides health check and readiness endpoints for monitoring.
package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// HealthStatus represents the overall health status of the service.
type HealthStatus string

const (
	// HealthStatusHealthy indicates the service is fully operational.
	HealthStatusHealthy HealthStatus = "healthy"

	// HealthStatusDegraded indicates the service is operational but some checks failed.
	HealthStatusDegraded HealthStatus = "degraded"

	// HealthStatusUnhealthy indicates the service is not operational.
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// CheckResult represents the result of a single health check.
type CheckResult struct {
	Status    HealthStatus `json:"status"`
	LatencyMS float64      `json:"latency_ms,omitempty"`
	Message   string       `json:"message,omitempty"`
	Error     string       `json:"error,omitempty"`
}

// HealthResponse represents the complete health check response.
type HealthResponse struct {
	Status    HealthStatus           `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Version   string                 `json:"version,omitempty"`
	Checks    map[string]CheckResult `json:"checks"`
	System    *SystemInfo            `json:"system,omitempty"`
}

// SystemInfo provides system-level information.
type SystemInfo struct {
	Goroutines    int     `json:"goroutines"`
	MemoryUsedMB  float64 `json:"memory_used_mb"`
	MemoryLimitMB float64 `json:"memory_limit_mb,omitempty"`
	CPUCount      int     `json:"cpu_count"`
}

// HealthChecker defines the interface for health check implementations.
type HealthChecker interface {
	// Name returns the name of the health check.
	Name() string

	// Check performs the health check and returns the result.
	Check(ctx context.Context) CheckResult
}

// HealthService manages health checks and endpoints.
type HealthService struct {
	checkers    map[string]HealthChecker
	startTime   time.Time
	version     string
	memoryLimit float64 // in MB, 0 means no limit
	mu          sync.RWMutex
}

// HealthServiceConfig configures the health service.
type HealthServiceConfig struct {
	// Version is the application version to include in health responses.
	Version string

	// MemoryLimitMB is the memory limit in MB (0 means no limit).
	MemoryLimitMB float64

	// IncludeSystemInfo determines whether to include system information in responses.
	IncludeSystemInfo bool
}

// NewHealthService creates a new health service.
func NewHealthService(config HealthServiceConfig) *HealthService {
	return &HealthService{
		checkers:    make(map[string]HealthChecker),
		startTime:   time.Now(),
		version:     config.Version,
		memoryLimit: config.MemoryLimitMB,
	}
}

// RegisterChecker adds a health checker to the service.
func (h *HealthService) RegisterChecker(checker HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers[checker.Name()] = checker
}

// RemoveChecker removes a health checker from the service.
func (h *HealthService) RemoveChecker(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.checkers, name)
}

// Check performs all registered health checks.
func (h *HealthService) Check(ctx context.Context, includeSystem bool) *HealthResponse {
	h.mu.RLock()
	checkers := make(map[string]HealthChecker, len(h.checkers))
	for name, checker := range h.checkers {
		checkers[name] = checker
	}
	h.mu.RUnlock()

	response := &HealthResponse{
		Status:    HealthStatusHealthy,
		Timestamp: time.Now(),
		Uptime:    time.Since(h.startTime).String(),
		Version:   h.version,
		Checks:    make(map[string]CheckResult),
	}

	// Run all health checks in parallel
	resultsChan := make(chan struct {
		name   string
		result CheckResult
	}, len(checkers))

	var wg sync.WaitGroup
	for name, checker := range checkers {
		wg.Add(1)
		go func(n string, c HealthChecker) {
			defer wg.Done()
			result := c.Check(ctx)
			resultsChan <- struct {
				name   string
				result CheckResult
			}{name: n, result: result}
		}(name, checker)
	}

	// Wait for all checks to complete
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	hasUnhealthy := false
	hasDegraded := false

	for result := range resultsChan {
		response.Checks[result.name] = result.result

		switch result.result.Status {
		case HealthStatusUnhealthy:
			hasUnhealthy = true
		case HealthStatusDegraded:
			hasDegraded = true
		}
	}

	// Determine overall status
	if hasUnhealthy {
		response.Status = HealthStatusUnhealthy
	} else if hasDegraded {
		response.Status = HealthStatusDegraded
	}

	// Add system information if requested
	if includeSystem {
		response.System = h.getSystemInfo()
	}

	return response
}

// getSystemInfo returns current system information.
func (h *HealthService) getSystemInfo() *SystemInfo {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	info := &SystemInfo{
		Goroutines:   runtime.NumGoroutine(),
		MemoryUsedMB: float64(m.Alloc) / 1024 / 1024,
		CPUCount:     runtime.NumCPU(),
	}

	if h.memoryLimit > 0 {
		info.MemoryLimitMB = h.memoryLimit
	}

	return info
}

// HealthHandler returns an HTTP handler for the /health endpoint.
// This performs all health checks and returns detailed status information.
func (h *HealthService) HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		response := h.Check(ctx, true)

		w.Header().Set("Content-Type", "application/json")

		// Set HTTP status code based on health status
		switch response.Status {
		case HealthStatusHealthy:
			w.WriteHeader(http.StatusOK)
		case HealthStatusDegraded:
			w.WriteHeader(http.StatusOK) // Still return 200 for degraded
		case HealthStatusUnhealthy:
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

// ReadyHandler returns an HTTP handler for the /ready endpoint.
// This checks if the service is ready to accept traffic (dependencies available).
func (h *HealthService) ReadyHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := h.Check(ctx, false)

		w.Header().Set("Content-Type", "application/json")

		// Return 503 if any critical checks are unhealthy
		if response.Status == HealthStatusUnhealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		_ = json.NewEncoder(w).Encode(response)
	}
}

// LiveHandler returns an HTTP handler for the /live endpoint.
// This is a simple liveness check that always returns 200 OK if the service is running.
func (h *HealthService) LiveHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response := &HealthResponse{
			Status:    HealthStatusHealthy,
			Timestamp: time.Now(),
			Uptime:    time.Since(h.startTime).String(),
			Checks:    make(map[string]CheckResult),
		}

		response.Checks["service"] = CheckResult{
			Status:  HealthStatusHealthy,
			Message: "Service is running",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	}
}

// StorageHealthChecker checks the health of a storage backend.
type StorageHealthChecker struct {
	storage StorageBackend
	name    string
}

// NewStorageHealthChecker creates a new storage health checker.
func NewStorageHealthChecker(storage StorageBackend, name string) *StorageHealthChecker {
	return &StorageHealthChecker{
		storage: storage,
		name:    name,
	}
}

// Name returns the name of the health check.
func (s *StorageHealthChecker) Name() string {
	return s.name
}

// Check performs the storage health check.
func (s *StorageHealthChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	err := s.storage.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:    HealthStatusUnhealthy,
			LatencyMS: latency.Seconds() * 1000,
			Error:     fmt.Sprintf("Storage ping failed: %v", err),
		}
	}

	// Check if latency is too high (>100ms indicates potential issues)
	status := HealthStatusHealthy
	if latency > 100*time.Millisecond {
		status = HealthStatusDegraded
	}

	return CheckResult{
		Status:    status,
		LatencyMS: latency.Seconds() * 1000,
		Message:   "Storage backend is responsive",
	}
}

// MemoryHealthChecker checks memory usage.
type MemoryHealthChecker struct {
	thresholdMB float64 // Alert threshold in MB
}

// NewMemoryHealthChecker creates a new memory health checker.
func NewMemoryHealthChecker(thresholdMB float64) *MemoryHealthChecker {
	return &MemoryHealthChecker{
		thresholdMB: thresholdMB,
	}
}

// Name returns the name of the health check.
func (m *MemoryHealthChecker) Name() string {
	return "memory"
}

// Check performs the memory health check.
func (m *MemoryHealthChecker) Check(ctx context.Context) CheckResult {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	usedMB := float64(memStats.Alloc) / 1024 / 1024

	status := HealthStatusHealthy
	message := fmt.Sprintf("Memory usage: %.2f MB", usedMB)

	if m.thresholdMB > 0 && usedMB > m.thresholdMB {
		status = HealthStatusDegraded
		message = fmt.Sprintf("Memory usage (%.2f MB) exceeds threshold (%.2f MB)", usedMB, m.thresholdMB)
	}

	return CheckResult{
		Status:  status,
		Message: message,
	}
}

// GoroutineHealthChecker checks goroutine count.
type GoroutineHealthChecker struct {
	thresholdCount int // Alert threshold
}

// NewGoroutineHealthChecker creates a new goroutine health checker.
func NewGoroutineHealthChecker(thresholdCount int) *GoroutineHealthChecker {
	return &GoroutineHealthChecker{
		thresholdCount: thresholdCount,
	}
}

// Name returns the name of the health check.
func (g *GoroutineHealthChecker) Name() string {
	return "goroutines"
}

// Check performs the goroutine health check.
func (g *GoroutineHealthChecker) Check(ctx context.Context) CheckResult {
	count := runtime.NumGoroutine()

	status := HealthStatusHealthy
	message := fmt.Sprintf("Goroutine count: %d", count)

	if g.thresholdCount > 0 && count > g.thresholdCount {
		status = HealthStatusDegraded
		message = fmt.Sprintf("Goroutine count (%d) exceeds threshold (%d)", count, g.thresholdCount)
	}

	return CheckResult{
		Status:  status,
		Message: message,
	}
}

// CustomHealthChecker allows custom health check functions.
type CustomHealthChecker struct {
	name    string
	checkFn func(ctx context.Context) CheckResult
}

// NewCustomHealthChecker creates a new custom health checker.
func NewCustomHealthChecker(name string, checkFn func(ctx context.Context) CheckResult) *CustomHealthChecker {
	return &CustomHealthChecker{
		name:    name,
		checkFn: checkFn,
	}
}

// Name returns the name of the health check.
func (c *CustomHealthChecker) Name() string {
	return c.name
}

// Check performs the custom health check.
func (c *CustomHealthChecker) Check(ctx context.Context) CheckResult {
	return c.checkFn(ctx)
}
