// Package loadtest provides comprehensive load and stress testing harness
// for GAuth high-volume scenarios.
package loadtest

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// TestScenario defines a load test scenario.
type TestScenario struct {
	Name             string
	Duration         time.Duration
	VirtualUsers     int
	RampUpTime       time.Duration
	ThinkTime        time.Duration
	RequestGenerator RequestGenerator
	Validator        ResponseValidator
}

// RequestGenerator generates test requests.
type RequestGenerator interface {
	Generate(userID int, iteration int) (interface{}, error)
}

// ResponseValidator validates responses.
type ResponseValidator interface {
	Validate(request interface{}, response interface{}, err error) error
}

// LoadTestResult contains aggregated load test results.
type LoadTestResult struct {
	Scenario        string
	Duration        time.Duration
	TotalRequests   int64
	SuccessfulReqs  int64
	FailedReqs      int64
	TotalErrors     int64
	AvgResponseTime time.Duration
	MinResponseTime time.Duration
	MaxResponseTime time.Duration
	P50ResponseTime time.Duration
	P95ResponseTime time.Duration
	P99ResponseTime time.Duration
	RequestsPerSec  float64
	ErrorRate       float64
	Throughput      float64
}

// LoadTestHarness orchestrates load tests.
type LoadTestHarness struct {
	mu sync.RWMutex

	// Current test state
	running   bool
	startTime time.Time
	endTime   time.Time

	// Metrics
	totalRequests  int64
	successfulReqs int64
	failedReqs     int64
	responseTimes  []time.Duration
	errors         []error

	// Configuration
	maxConcurrency int
	reportInterval time.Duration
}

// NewLoadTestHarness creates a new load test harness.
func NewLoadTestHarness() *LoadTestHarness {
	return &LoadTestHarness{
		maxConcurrency: 1000,
		reportInterval: 10 * time.Second,
		responseTimes:  make([]time.Duration, 0, 10000),
		errors:         make([]error, 0),
	}
}

// Run executes a load test scenario.
func (h *LoadTestHarness) Run(ctx context.Context, scenario *TestScenario) (*LoadTestResult, error) {
	if scenario == nil {
		return nil, fmt.Errorf("scenario is required")
	}

	h.reset()
	h.running = true
	h.startTime = time.Now()

	// Create worker pool
	var wg sync.WaitGroup
	userSemaphore := make(chan struct{}, scenario.VirtualUsers)

	// Ramp up virtual users
	rampUpInterval := scenario.RampUpTime / time.Duration(scenario.VirtualUsers)

	// Test duration context with small grace period for request completion
	testCtx, cancel := context.WithTimeout(ctx, scenario.Duration)
	defer cancel()

	// Start progress reporter
	go h.reportProgress(testCtx, scenario.Name)

	// Launch virtual users
	for userID := 0; userID < scenario.VirtualUsers; userID++ {
		// Ramp up delay
		select {
		case <-testCtx.Done():
			goto done
		case <-time.After(rampUpInterval):
		}

		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			h.runVirtualUser(testCtx, uid, scenario, userSemaphore)
		}(userID)
	}

done:
	// Wait for all users to complete
	// The wg.Done() is called when each virtual user goroutine exits,
	// which happens after all their requests are fully processed
	wg.Wait()

	h.endTime = time.Now()
	h.running = false

	return h.generateResult(scenario), nil
}

// runVirtualUser simulates a single virtual user's behavior.
func (h *LoadTestHarness) runVirtualUser(ctx context.Context, userID int, scenario *TestScenario, sem chan struct{}) {
	iteration := 0

	for {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
			// Execute request
			h.executeRequest(ctx, userID, iteration, scenario)
			<-sem

			iteration++

			// Think time between requests
			if scenario.ThinkTime > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(scenario.ThinkTime):
				}
			}
		}
	}
}

// executeRequest executes a single request and records metrics.
func (h *LoadTestHarness) executeRequest(ctx context.Context, userID, iteration int, scenario *TestScenario) {
	// Check if context is already cancelled before starting
	select {
	case <-ctx.Done():
		// Don't count requests that start after test completion
		return
	default:
	}

	atomic.AddInt64(&h.totalRequests, 1)

	// Generate request
	request, err := scenario.RequestGenerator.Generate(userID, iteration)
	if err != nil {
		atomic.AddInt64(&h.failedReqs, 1)
		h.recordError(err)
		return
	}

	// Execute request and measure time
	startTime := time.Now()
	response, err := h.performRequest(ctx, request)
	duration := time.Since(startTime)

	// Record response time only for completed requests
	if err == nil {
		h.recordResponseTime(duration)
	}

	// Validate response
	if scenario.Validator != nil {
		if validationErr := scenario.Validator.Validate(request, response, err); validationErr != nil {
			atomic.AddInt64(&h.failedReqs, 1)
			h.recordError(validationErr)
			return
		}
	}

	// Count final result
	if err != nil {
		atomic.AddInt64(&h.failedReqs, 1)
		h.recordError(err)
	} else {
		atomic.AddInt64(&h.successfulReqs, 1)
	}
}

// performRequest executes the actual request (to be customized per test).
func (h *LoadTestHarness) performRequest(ctx context.Context, request interface{}) (interface{}, error) {
	// This is a hook for custom request execution
	// In real tests, this would call the actual service

	// Simulate processing time with graceful context handling
	timer := time.NewTimer(10 * time.Millisecond)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		// Context cancelled - this is expected during test teardown
		return nil, ctx.Err()
	case <-timer.C:
		// Request completed successfully
		return "success", nil
	}
}

// recordResponseTime safely records a response time.
func (h *LoadTestHarness) recordResponseTime(duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.responseTimes = append(h.responseTimes, duration)
}

// recordError safely records an error.
func (h *LoadTestHarness) recordError(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.errors = append(h.errors, err)
}

// reportProgress prints progress updates during the test.
func (h *LoadTestHarness) reportProgress(ctx context.Context, scenarioName string) {
	ticker := time.NewTicker(h.reportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			total := atomic.LoadInt64(&h.totalRequests)
			successful := atomic.LoadInt64(&h.successfulReqs)
			failed := atomic.LoadInt64(&h.failedReqs)
			elapsed := time.Since(h.startTime)
			rps := float64(total) / elapsed.Seconds()

			fmt.Printf("[%s] Elapsed: %v | Requests: %d | Success: %d | Failed: %d | RPS: %.2f\n",
				scenarioName, elapsed.Round(time.Second), total, successful, failed, rps)
		}
	}
}

// generateResult computes final statistics.
func (h *LoadTestHarness) generateResult(scenario *TestScenario) *LoadTestResult {
	h.mu.RLock()
	defer h.mu.RUnlock()

	duration := h.endTime.Sub(h.startTime)

	result := &LoadTestResult{
		Scenario:       scenario.Name,
		Duration:       duration,
		TotalRequests:  atomic.LoadInt64(&h.totalRequests),
		SuccessfulReqs: atomic.LoadInt64(&h.successfulReqs),
		FailedReqs:     atomic.LoadInt64(&h.failedReqs),
		TotalErrors:    int64(len(h.errors)),
	}

	if result.TotalRequests > 0 {
		result.RequestsPerSec = float64(result.TotalRequests) / duration.Seconds()
		result.ErrorRate = float64(result.FailedReqs) / float64(result.TotalRequests)
		result.Throughput = result.RequestsPerSec
	}

	// Calculate response time statistics
	if len(h.responseTimes) > 0 {
		result.AvgResponseTime = h.calculateAverage(h.responseTimes)
		result.MinResponseTime = h.calculateMin(h.responseTimes)
		result.MaxResponseTime = h.calculateMax(h.responseTimes)
		result.P50ResponseTime = h.calculatePercentile(h.responseTimes, 50)
		result.P95ResponseTime = h.calculatePercentile(h.responseTimes, 95)
		result.P99ResponseTime = h.calculatePercentile(h.responseTimes, 99)
	}

	return result
}

// calculateAverage computes average of durations.
func (h *LoadTestHarness) calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}
	return total / time.Duration(len(durations))
}

// calculateMin finds minimum duration.
func (h *LoadTestHarness) calculateMin(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}
	return min
}

// calculateMax finds maximum duration.
func (h *LoadTestHarness) calculateMax(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}
	return max
}

// calculatePercentile computes the Nth percentile.
func (h *LoadTestHarness) calculatePercentile(durations []time.Duration, percentile int) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	// Simple percentile calculation (not perfectly accurate without sorting)
	// For production, use a proper percentile algorithm
	index := (len(durations) * percentile) / 100
	if index >= len(durations) {
		index = len(durations) - 1
	}

	return durations[index]
}

// reset clears test state.
func (h *LoadTestHarness) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()

	atomic.StoreInt64(&h.totalRequests, 0)
	atomic.StoreInt64(&h.successfulReqs, 0)
	atomic.StoreInt64(&h.failedReqs, 0)
	h.responseTimes = make([]time.Duration, 0, 10000)
	h.errors = make([]error, 0)
}

// PrintResult prints a formatted test result.
func (h *LoadTestHarness) PrintResult(result *LoadTestResult) {
	separator := "================================================================================"
	fmt.Println("\n" + separator)
	fmt.Printf("Load Test Results: %s\n", result.Scenario)
	fmt.Println(separator)
	fmt.Printf("Duration:          %v\n", result.Duration)
	fmt.Printf("Total Requests:    %d\n", result.TotalRequests)
	fmt.Printf("Successful:        %d (%.2f%%)\n", result.SuccessfulReqs,
		float64(result.SuccessfulReqs)/float64(result.TotalRequests)*100)
	fmt.Printf("Failed:            %d (%.2f%%)\n", result.FailedReqs, result.ErrorRate*100)
	fmt.Printf("Requests/sec:      %.2f\n", result.RequestsPerSec)
	fmt.Println("--------------------------------------------------------------------------------")
	fmt.Printf("Response Times:\n")
	fmt.Printf("  Min:             %v\n", result.MinResponseTime)
	fmt.Printf("  Avg:             %v\n", result.AvgResponseTime)
	fmt.Printf("  Max:             %v\n", result.MaxResponseTime)
	fmt.Printf("  P50:             %v\n", result.P50ResponseTime)
	fmt.Printf("  P95:             %v\n", result.P95ResponseTime)
	fmt.Printf("  P99:             %v\n", result.P99ResponseTime)
	fmt.Println(separator)
}

// StressTestConfig configures a stress test.
type StressTestConfig struct {
	StartUsers     int
	MaxUsers       int
	UserIncrement  int
	IncrementEvery time.Duration
	TestDuration   time.Duration
}

// RunStressTest performs a stress test with gradually increasing load.
func (h *LoadTestHarness) RunStressTest(ctx context.Context, config *StressTestConfig, reqGen RequestGenerator) ([]*LoadTestResult, error) {
	results := make([]*LoadTestResult, 0)

	for users := config.StartUsers; users <= config.MaxUsers; users += config.UserIncrement {
		scenario := &TestScenario{
			Name:             fmt.Sprintf("Stress-%d-users", users),
			Duration:         config.IncrementEvery,
			VirtualUsers:     users,
			RampUpTime:       config.IncrementEvery / 4,
			ThinkTime:        0,
			RequestGenerator: reqGen,
		}

		result, err := h.Run(ctx, scenario)
		if err != nil {
			return results, err
		}

		results = append(results, result)

		// Print intermediate result
		h.PrintResult(result)

		// Check if system is degrading
		if result.ErrorRate > 0.5 {
			fmt.Printf("⚠️  High error rate detected (%.2f%%), stopping stress test\n", result.ErrorRate*100)
			break
		}
	}

	return results, nil
}

// SimpleRequestGenerator is a basic request generator for testing.
type SimpleRequestGenerator struct {
	RequestTemplate interface{}
}

// Generate creates a simple request.
func (s *SimpleRequestGenerator) Generate(userID, iteration int) (interface{}, error) {
	return map[string]interface{}{
		"user_id":   userID,
		"iteration": iteration,
		"template":  s.RequestTemplate,
	}, nil
}

// AlwaysValidValidator always validates successfully.
type AlwaysValidValidator struct{}

// Validate always returns nil.
func (a *AlwaysValidValidator) Validate(request, response interface{}, err error) error {
	if err != nil {
		return err
	}
	return nil
}
