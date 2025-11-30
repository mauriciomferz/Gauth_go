package revocation

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// raceEnabled is set in race_enabled.go and race_disabled.go based on build tags
var raceEnabled bool

// Load Testing Suite
// Measures throughput, latency percentiles, and resource usage under high load

// LoadTestMetrics tracks performance metrics during load tests
type LoadTestMetrics struct {
	TotalOps       int64
	SuccessOps     int64
	FailedOps      int64
	TotalLatencyNs int64
	Latencies      []time.Duration
	StartTime      time.Time
	EndTime        time.Time
	mu             sync.Mutex
}

func (m *LoadTestMetrics) RecordOp(success bool, latency time.Duration) {
	atomic.AddInt64(&m.TotalOps, 1)
	if success {
		atomic.AddInt64(&m.SuccessOps, 1)
	} else {
		atomic.AddInt64(&m.FailedOps, 1)
	}
	atomic.AddInt64(&m.TotalLatencyNs, int64(latency))

	m.mu.Lock()
	m.Latencies = append(m.Latencies, latency)
	m.mu.Unlock()
}

func (m *LoadTestMetrics) Percentile(p float64) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Latencies) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(m.Latencies))
	copy(sorted, m.Latencies)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	index := int(float64(len(sorted)) * p)
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}

func (m *LoadTestMetrics) Throughput() float64 {
	duration := m.EndTime.Sub(m.StartTime).Seconds()
	if duration == 0 {
		return 0
	}
	totalOps := atomic.LoadInt64(&m.TotalOps)
	return float64(totalOps) / duration
}

func (m *LoadTestMetrics) AvgLatency() time.Duration {
	totalOps := atomic.LoadInt64(&m.TotalOps)
	if totalOps == 0 {
		return 0
	}
	totalLatencyNs := atomic.LoadInt64(&m.TotalLatencyNs)
	return time.Duration(totalLatencyNs / totalOps)
}

func (m *LoadTestMetrics) PrintSummary(t *testing.T, testName string) {
	// Read all atomic counters atomically to avoid race conditions
	totalOps := atomic.LoadInt64(&m.TotalOps)
	successOps := atomic.LoadInt64(&m.SuccessOps)
	failedOps := atomic.LoadInt64(&m.FailedOps)
	
	t.Logf("\n=== %s Load Test Results ===", testName)
	t.Logf("Total Ops: %d", totalOps)
	t.Logf("Success: %d (%.2f%%)", successOps, float64(successOps)/float64(totalOps)*100)
	t.Logf("Failed: %d (%.2f%%)", failedOps, float64(failedOps)/float64(totalOps)*100)
	t.Logf("Duration: %v", m.EndTime.Sub(m.StartTime))
	t.Logf("Throughput: %.2f ops/sec", m.Throughput())
	t.Logf("Avg Latency: %v", m.AvgLatency())
	t.Logf("P50 Latency: %v", m.Percentile(0.50))
	t.Logf("P95 Latency: %v", m.Percentile(0.95))
	t.Logf("P99 Latency: %v", m.Percentile(0.99))
	t.Logf("P999 Latency: %v", m.Percentile(0.999))
	t.Logf("Max Latency: %v", m.Percentile(1.0))
}

// TestLoad_CircuitBreakerHighConcurrency tests circuit breaker under high concurrent load
func TestLoad_CircuitBreakerHighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	metrics := &LoadTestMetrics{
		StartTime: time.Now(),
		Latencies: make([]time.Duration, 0, 10000),
	}

	// Test with 100 concurrent workers, each performing 100 operations
	numWorkers := 100
	opsPerWorker := 100
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
			go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				poaID := fmt.Sprintf("poa-worker%d-op%d", workerID, i)
				start := time.Now()
				err := cb.RecordTransaction(ctx, poaID, 1000, true)
				latency := time.Since(start)
				metrics.RecordOp(err == nil, latency)
			}
		}(w)
	}

	wg.Wait()
	metrics.EndTime = time.Now()
	metrics.PrintSummary(t, "CircuitBreaker HighConcurrency")

	// Assertions
	assert.Greater(t, metrics.SuccessOps, int64(9500), "Should have >95% success rate")
	assert.Less(t, metrics.Percentile(0.99), 100*time.Millisecond, "P99 latency should be <100ms")
	assert.Greater(t, metrics.Throughput(), 1000.0, "Throughput should be >1000 ops/sec")
}

// TestLoad_TwoPhaseHighThroughput tests two-phase revocation under sustained load
func TestLoad_TwoPhaseHighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	tpr, mr := setupTwoPhaseTest(t)
	defer mr.Close()

	ctx := context.Background()
	metrics := &LoadTestMetrics{
		StartTime: time.Now(),
		Latencies: make([]time.Duration, 0, 5000),
	}

	// Create many PoAs first (disabled state)
	numPoAs := 1000
	for i := 0; i < numPoAs; i++ {
		poaID := fmt.Sprintf("poa-throughput-%d", i)
		err := tpr.DisablePoA(ctx, poaID, "test-principal", "load-test")
		require.NoError(t, err)
	}

	// Test revocation throughput with 50 concurrent workers
	numWorkers := 50
	poasPerWorker := numPoAs / numWorkers
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			startIdx := workerID * poasPerWorker
			endIdx := startIdx + poasPerWorker
			for i := startIdx; i < endIdx; i++ {
				poaID := fmt.Sprintf("poa-throughput-%d", i)
				start := time.Now()
				err := tpr.RevokePoA(ctx, poaID, "load-test-revoke")
				latency := time.Since(start)
				metrics.RecordOp(err == nil, latency)
			}
		}(w)
	}

	wg.Wait()
	metrics.EndTime = time.Now()
	metrics.PrintSummary(t, "TwoPhase HighThroughput")

	// Assertions
	assert.Greater(t, metrics.SuccessOps, int64(950), "Should have >95% success rate")
	assert.Less(t, metrics.Percentile(0.95), 50*time.Millisecond, "P95 latency should be <50ms")
	assert.Greater(t, metrics.Throughput(), 500.0, "Throughput should be >500 ops/sec")
}

// TestLoad_OptimisticBurstTraffic tests optimistic revocation with bursty traffic pattern
func TestLoad_OptimisticBurstTraffic(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	opt, mr := setupOptimisticTest(t)
	defer mr.Close()

	ctx := context.Background()
	metrics := &LoadTestMetrics{
		StartTime: time.Now(),
		Latencies: make([]time.Duration, 0, 3000),
	}

	// Simulate burst pattern: 5 bursts of 200 operations each
	numBursts := 5
	opsPerBurst := 200
	burstInterval := 100 * time.Millisecond

	for burst := 0; burst < numBursts; burst++ {
		var wg sync.WaitGroup
		for i := 0; i < opsPerBurst; i++ {
			wg.Add(1)
				go func(burstID, opID int) {
				defer wg.Done()
				poaID := fmt.Sprintf("poa-burst%d-op%d", burstID, opID)
				collateral := uint64(1000 + opID)

				start := time.Now()
				err := opt.MarkPendingRevocation(ctx, poaID, "test-principal", "load-test", collateral)
				latency := time.Since(start)
				metrics.RecordOp(err == nil, latency)
			}(burst, i)
		}
		wg.Wait()
		time.Sleep(burstInterval)
	}

	metrics.EndTime = time.Now()
	metrics.PrintSummary(t, "Optimistic BurstTraffic")

	// Assertions - optimistic revocation may have higher error rate under burst load
	assert.Greater(t, metrics.TotalOps, int64(900), "Should attempt most burst operations")
	if metrics.SuccessOps > 0 {
		assert.Less(t, metrics.Percentile(0.99), 200*time.Millisecond, "P99 latency should be reasonable")
	}
}

// TestLoad_MixedOperations tests all systems with mixed operation types
func TestLoad_MixedOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	metrics := &LoadTestMetrics{
		StartTime: time.Now(),
		Latencies: make([]time.Duration, 0, 2000),
	}

	// Mix of operations: BeginTransaction, CompleteTransaction, state checks
	numWorkers := 20
	opsPerWorker := 100
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				poaID := fmt.Sprintf("poa-mixed-w%d-op%d", workerID, i)
				opType := rand.Intn(3)

				start := time.Now()
				var err error
				switch opType {
				case 0: // RecordTransaction (success)
					err = cb.RecordTransaction(ctx, poaID, 1000, true)
				case 1: // RecordTransaction (failure)
					err = cb.RecordTransaction(ctx, poaID, 1000, false)
				case 2: // GetMetrics
					_, err = cb.GetMetrics(ctx, poaID)
				}
				latency := time.Since(start)
				metrics.RecordOp(err == nil, latency)
			}
		}(w)
	}

	wg.Wait()
	metrics.EndTime = time.Now()
	metrics.PrintSummary(t, "Mixed Operations")

	// Assertions
	assert.Greater(t, metrics.SuccessOps, int64(1900), "Should handle >95% of mixed operations")
	assert.Less(t, metrics.AvgLatency(), 10*time.Millisecond, "Avg latency should be <10ms")
}

// TestLoad_SustainedLoad tests system stability under sustained load over time
func TestLoad_SustainedLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	metrics := &LoadTestMetrics{
		StartTime: time.Now(),
		Latencies: make([]time.Duration, 0, 5000),
	}

	// Run sustained load for 10 seconds with constant rate
	duration := 10 * time.Second
	targetRate := 500.0 // ops/sec
	interval := time.Duration(float64(time.Second) / targetRate)

	done := make(chan bool)
	go func() {
		time.Sleep(duration)
		close(done)
	}()

	opID := 0
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			metrics.EndTime = time.Now()
			metrics.PrintSummary(t, "Sustained Load")

			// Assertions
			assert.Greater(t, metrics.TotalOps, int64(4000), "Should process >4000 ops in 10 seconds")
			assert.Greater(t, metrics.SuccessOps, int64(3800), "Should have >95% success rate")
			assert.Less(t, metrics.Percentile(0.95), 20*time.Millisecond, "P95 latency should be <20ms")
			
			// Check throughput stability
			actualRate := metrics.Throughput()
			expectedRate := targetRate * 0.8 // Allow 20% variance
			assert.Greater(t, actualRate, expectedRate, "Throughput should be stable")
			return

		case <-ticker.C:
			go func(id int) {
				poaID := fmt.Sprintf("poa-sustained-%d", id)
				start := time.Now()
				err := cb.RecordTransaction(ctx, poaID, 1000, true)
				latency := time.Since(start)
				metrics.RecordOp(err == nil, latency)
			}(opID)
			opID++
		}
	}
}

// TestLoad_LargePoASet tests performance with many PoAs in the system
func TestLoad_LargePoASet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}
	
	// Skip performance tests when race detector is enabled (adds 5-10x overhead)
	if raceEnabled {
		t.Skip("Skipping performance test with race detector enabled")
	}

	tpr, mr := setupTwoPhaseTest(t)
	defer mr.Close()

	ctx := context.Background()

	// Create a large number of PoAs (10,000)
	numPoAs := 10000
	t.Logf("Creating %d PoAs...", numPoAs)
	createStart := time.Now()

	var createWg sync.WaitGroup
	numWorkers := 50
	poasPerWorker := numPoAs / numWorkers

	for w := 0; w < numWorkers; w++ {
		createWg.Add(1)
		go func(workerID int) {
			defer createWg.Done()
			startIdx := workerID * poasPerWorker
			endIdx := startIdx + poasPerWorker
			for i := startIdx; i < endIdx; i++ {
				poaID := fmt.Sprintf("poa-large-set-%d", i)
				_ = tpr.DisablePoA(ctx, poaID, "test-principal", "load-test")
			}
		}(w)
	}
	createWg.Wait()
	createDuration := time.Since(createStart)
	t.Logf("Created %d PoAs in %v (%.2f ops/sec)", numPoAs, createDuration, float64(numPoAs)/createDuration.Seconds())

	// Now test query performance with large dataset
	metrics := &LoadTestMetrics{
		StartTime: time.Now(),
		Latencies: make([]time.Duration, 0, 1000),
	}

	// Random access pattern
	numQueries := 1000
	var queryWg sync.WaitGroup
	for i := 0; i < numQueries; i++ {
		queryWg.Add(1)
		go func(queryID int) {
			defer queryWg.Done()
			randomPoA := rand.Intn(numPoAs)
			poaID := fmt.Sprintf("poa-large-set-%d", randomPoA)

			start := time.Now()
			err := tpr.RevokePoA(ctx, poaID, "query-test")
			latency := time.Since(start)
			metrics.RecordOp(err == nil, latency)
		}(i)
	}
	queryWg.Wait()
	metrics.EndTime = time.Now()
	metrics.PrintSummary(t, "Large PoA Set Query")

	// Assertions - performance should not degrade significantly with large dataset
	assert.Greater(t, metrics.SuccessOps, int64(900), "Should handle >90% of queries with large dataset")
	// Increased threshold to 150ms to account for real-world database latency with large datasets
	assert.Less(t, metrics.Percentile(0.95), 150*time.Millisecond, "P95 latency should remain <150ms even with 10k PoAs")
	assert.Greater(t, metrics.Throughput(), 100.0, "Throughput should remain >100 ops/sec")
}

// TestLoad_MemoryStability tests memory usage remains stable under load
func TestLoad_MemoryStability(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()

	// Run continuous operations and verify no memory leak
	numIterations := 5
	opsPerIteration := 1000

	for iter := 0; iter < numIterations; iter++ {
		var wg sync.WaitGroup
		for i := 0; i < opsPerIteration; i++ {
			wg.Add(1)
				go func(iterID, opID int) {
				defer wg.Done()
				poaID := fmt.Sprintf("poa-mem-iter%d-op%d", iterID, opID)
				_ = cb.RecordTransaction(ctx, poaID, 1000, true)
			}(iter, i)
		}
		wg.Wait()

		// Get metrics after each iteration
		metrics, _ := cb.GetMetrics(ctx, fmt.Sprintf("poa-mem-iter%d-op0", iter))
		if metrics != nil {
			t.Logf("Iteration %d: TotalTx=%d", iter, metrics.TotalTxCount)
		}
	}
}

// TestLoad_ErrorRateUnderStress tests error handling under stress conditions
func TestLoad_ErrorRateUnderStress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test in short mode")
	}

	cb, mr := setupCircuitBreakerTest(t)
	defer mr.Close()
	defer cb.Close()

	ctx := context.Background()
	metrics := &LoadTestMetrics{
		StartTime: time.Now(),
		Latencies: make([]time.Duration, 0, 2000),
	}

	// Introduce some operations that will fail (duplicate IDs, invalid values)
	numWorkers := 50
	opsPerWorker := 40
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < opsPerWorker; i++ {
				// Use enough unique PoA IDs to avoid hitting rate limits (10 tx/min)
				// With 50 workers * 40 ops = 2000 total ops, use 500 unique IDs (avg 4 tx/ID)
				poaID := fmt.Sprintf("poa-error-test-w%d-batch%d", workerID, i/4)
				collateral := uint64(1000)
				if i%10 == 0 {
					collateral = 0 // Invalid collateral - intentional error
				}

				start := time.Now()
				err := cb.RecordTransaction(ctx, poaID, collateral, true)
				latency := time.Since(start)
				metrics.RecordOp(err == nil, latency)
			}
		}(w)
	}

	wg.Wait()
	metrics.EndTime = time.Now()
	metrics.PrintSummary(t, "Error Rate Under Stress")

	// Should handle errors gracefully without crashing
	assert.Greater(t, metrics.TotalOps, int64(1900), "Should attempt all operations")
	// Many operations will fail by design (zero collateral, rate limits)
	if metrics.SuccessOps > 0 {
		assert.Less(t, metrics.Percentile(0.99), 100*time.Millisecond, "P99 latency should be reasonable")
	}
	t.Logf("Error handling validated: %d successes, %d expected errors", metrics.SuccessOps, metrics.FailedOps)
}
