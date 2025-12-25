package revocation

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func BenchmarkTwoPhaseRevocation_DisablePoA(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	logger := NewSimpleLogger("BENCH")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	if err != nil {
		b.Fatalf("Failed to create oracle: %v", err)
	}
	defer oracle.Close()

	tpr, err := NewTwoPhaseRevocation(oracle, []string{mr.Addr()}, logger)
	if err != nil {
		b.Fatalf("Failed to create two-phase revocation: %v", err)
	}
	defer tpr.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		poaID := fmt.Sprintf("poa_bench_%d", i)
		principal := "0xBenchPrincipal"
		reason := "Benchmark test"

		err := tpr.DisablePoA(ctx, poaID, principal, reason)
		if err != nil {
			b.Fatalf("DisablePoA failed: %v", err)
		}
	}
}

func BenchmarkTwoPhaseRevocation_RevokePoA(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	logger := NewSimpleLogger("BENCH")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	if err != nil {
		b.Fatalf("Failed to create oracle: %v", err)
	}
	defer oracle.Close()

	tpr, err := NewTwoPhaseRevocation(oracle, []string{mr.Addr()}, logger)
	if err != nil {
		b.Fatalf("Failed to create two-phase revocation: %v", err)
	}
	defer tpr.Close()

	ctx := context.Background()

	// Pre-populate with disabled PoAs
	poaIDs := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		poaID := fmt.Sprintf("poa_revoke_%d", i)
		poaIDs[i] = poaID

		err := tpr.DisablePoA(ctx, poaID, "0xBenchPrincipal", "Setup for benchmark")
		if err != nil {
			b.Fatalf("Setup DisablePoA failed: %v", err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		err := tpr.RevokePoA(ctx, poaIDs[i], "Benchmark revocation")
		if err != nil {
			b.Fatalf("RevokePoA failed: %v", err)
		}
	}
}

func BenchmarkEmergencyOracle_EmergencyRevoke(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	logger := NewSimpleLogger("BENCH")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	if err != nil {
		b.Fatalf("Failed to create oracle: %v", err)
	}
	defer oracle.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		poaID := fmt.Sprintf("poa_emergency_%d", i)
		event := &RevocationEvent{
			PoAID:     poaID,
			Principal: "0xBenchPrincipal",
			EventID:   fmt.Sprintf("event_%d", i),
			Reason:    "Benchmark emergency revocation",
			Timestamp: time.Now(),
			TTL:       3600, // 1 hour
		}

		err := oracle.EmergencyRevoke(ctx, event)
		if err != nil {
			b.Fatalf("EmergencyRevoke failed: %v", err)
		}
	}
}

func BenchmarkCircuitBreaker_RecordTransaction(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	logger := NewSimpleLogger("BENCH")

	// Create circuit breaker with default config
	config := &RateLimitConfig{
		MaxTxPerMinute:    1000,
		MaxTxPerHour:      10000,
		MaxValuePerMinute: 1000000,
		MaxValuePerHour:   10000000,
		MaxFailureRate:    0.1,
		FailureWindowSecs: 300,
	}

	cb, err := NewCircuitBreaker([]string{mr.Addr()}, config, logger)
	if err != nil {
		b.Fatalf("Failed to create circuit breaker: %v", err)
	}
	defer cb.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		poaID := fmt.Sprintf("poa_cb_%d", i%100) // Use 100 different PoAs
		value := uint64(100)
		success := true

		err := cb.RecordTransaction(ctx, poaID, value, success)
		if err != nil {
			b.Fatalf("RecordTransaction failed: %v", err)
		}
	}
}

func BenchmarkCircuitBreaker_IsPoAAllowed(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	logger := NewSimpleLogger("BENCH")

	config := &RateLimitConfig{
		MaxTxPerMinute:    1000,
		MaxTxPerHour:      10000,
		MaxValuePerMinute: 1000000,
		MaxValuePerHour:   10000000,
		MaxFailureRate:    0.1,
		FailureWindowSecs: 300,
	}

	cb, err := NewCircuitBreaker([]string{mr.Addr()}, config, logger)
	if err != nil {
		b.Fatalf("Failed to create circuit breaker: %v", err)
	}
	defer cb.Close()

	ctx := context.Background()

	// Pre-populate some PoAs
	for i := 0; i < 10; i++ {
		poaID := fmt.Sprintf("poa_check_%d", i)
		_ = cb.RecordTransaction(ctx, poaID, 100, true)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		poaID := fmt.Sprintf("poa_check_%d", i%10)

		allowed, _, err := cb.IsPoAAllowed(ctx, poaID)
		if err != nil {
			b.Fatalf("IsPoAAllowed failed: %v", err)
		}
		_ = allowed // Use the result to prevent optimization
	}
}

func BenchmarkConcurrentRevocation(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	logger := NewSimpleLogger("BENCH")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	if err != nil {
		b.Fatalf("Failed to create oracle: %v", err)
	}
	defer oracle.Close()

	tpr, err := NewTwoPhaseRevocation(oracle, []string{mr.Addr()}, logger)
	if err != nil {
		b.Fatalf("Failed to create two-phase revocation: %v", err)
	}
	defer tpr.Close()

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			poaID := fmt.Sprintf("poa_concurrent_%d_%d", b.N, i)
			principal := "0xConcurrentPrincipal"
			reason := fmt.Sprintf("Concurrent revocation test %d", i)

			err := tpr.DisablePoA(ctx, poaID, principal, reason)
			if err != nil {
				b.Fatalf("Concurrent DisablePoA failed: %v", err)
			}
			i++
		}
	})
}

func BenchmarkHighThroughputRevocation(b *testing.B) {
	mr, err := miniredis.Run()
	if err != nil {
		b.Fatalf("Failed to start miniredis: %v", err)
	}
	defer mr.Close()

	logger := NewSimpleLogger("BENCH")
	oracle, err := NewEmergencyOracle([]string{mr.Addr()}, logger)
	if err != nil {
		b.Fatalf("Failed to create oracle: %v", err)
	}
	defer oracle.Close()

	tpr, err := NewTwoPhaseRevocation(oracle, []string{mr.Addr()}, logger)
	if err != nil {
		b.Fatalf("Failed to create two-phase revocation: %v", err)
	}
	defer tpr.Close()

	ctx := context.Background()

	// Benchmark measures ops/sec for high-throughput scenarios
	b.ResetTimer()
	b.ReportAllocs()

	start := time.Now()

	for i := 0; i < b.N; i++ {
		poaID := fmt.Sprintf("poa_throughput_%d", i)
		principal := "0xThroughputPrincipal"
		reason := "High throughput benchmark"

		err := tpr.DisablePoA(ctx, poaID, principal, reason)
		if err != nil {
			b.Fatalf("High throughput DisablePoA failed: %v", err)
		}
	}

	duration := time.Since(start)
	opsPerSec := float64(b.N) / duration.Seconds()

	// Report custom metrics
	b.ReportMetric(opsPerSec, "ops/sec")
	b.ReportMetric(duration.Seconds()/float64(b.N)*1000, "ms/op")
}
