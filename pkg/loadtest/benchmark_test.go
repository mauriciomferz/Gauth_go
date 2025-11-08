// Package loadtest benchmarks.
package loadtest

import (
	"context"
	"testing"
	"time"
)

// BenchmarkAuthorizationThroughput measures authorization throughput.
func BenchmarkAuthorizationThroughput(b *testing.B) {
	subjects := generateTestSubjects(100)
	resources := generateTestResources(50)
	actions := []string{"read", "write"}

	gen := NewAuthorizationRequestGenerator(subjects, resources, actions)
	harness := NewLoadTestHarness()
	ctx := context.Background()

	scenario := &TestScenario{
		RequestGenerator: gen,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		harness.executeRequest(ctx, i%10, i, scenario)
	}
}

// BenchmarkConcurrentAuthorization measures concurrent authorization performance.
func BenchmarkConcurrentAuthorization(b *testing.B) {
	subjects := generateTestSubjects(100)
	resources := generateTestResources(50)
	actions := []string{"read", "write"}

	gen := NewAuthorizationRequestGenerator(subjects, resources, actions)
	harness := NewLoadTestHarness()
	ctx := context.Background()

	scenario := &TestScenario{
		RequestGenerator: gen,
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			harness.executeRequest(ctx, i%10, i, scenario)
			i++
		}
	})
}

// BenchmarkDelegationOperations measures delegation operation performance.
func BenchmarkDelegationOperations(b *testing.B) {
	subjects := generateTestSubjects(50)
	delegates := generateTestSubjects(200)
	resources := generateTestResources(100)
	actions := []string{"read", "write"}

	gen := NewDelegationRequestGenerator(subjects, delegates, resources, actions)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate(i%10, i)
	}
}

// BenchmarkCacheHotSet measures cache hot set performance.
func BenchmarkCacheHotSet(b *testing.B) {
	subjects := generateTestSubjects(100)
	resources := generateTestResources(1000)
	actions := []string{"read"}

	gen := NewCachePressureGenerator(subjects, resources, actions, 100, 900, 0.9)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate(i%10, i)
	}
}

// BenchmarkCacheColdSet measures cache cold set performance.
func BenchmarkCacheColdSet(b *testing.B) {
	subjects := generateTestSubjects(100)
	resources := generateTestResources(1000)
	actions := []string{"read"}

	gen := NewCachePressureGenerator(subjects, resources, actions, 100, 900, 0.1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate(i%10, i)
	}
}

// BenchmarkBurstLoad measures burst load performance.
func BenchmarkBurstLoad(b *testing.B) {
	subjects := generateTestSubjects(100)
	resources := generateTestResources(50)
	actions := []string{"read", "write"}

	baseGen := NewAuthorizationRequestGenerator(subjects, resources, actions)
	burstGen := NewBurstLoadGenerator(baseGen, 0.2, 5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = burstGen.Generate(i%10, i)
	}
}

// BenchmarkResponseTimeCalculation measures statistics calculation performance.
func BenchmarkResponseTimeCalculation(b *testing.B) {
	harness := NewLoadTestHarness()

	// Populate with sample data
	for i := 0; i < 10000; i++ {
		harness.recordResponseTime(time.Duration(i) * time.Millisecond)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		harness.mu.RLock()
		_ = harness.calculateAverage(harness.responseTimes)
		_ = harness.calculateMin(harness.responseTimes)
		_ = harness.calculateMax(harness.responseTimes)
		_ = harness.calculatePercentile(harness.responseTimes, 95)
		harness.mu.RUnlock()
	}
}
