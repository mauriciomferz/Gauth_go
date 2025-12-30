// Copyright (c) 2025 AgentAuth. All rights reserved.

package metrics

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// mockCollector is a test collector that tracks method calls.
type mockCollector struct {
	noop
	id               string
	callCount        atomic.Int64
	flushCount       atomic.Int64
	closeCount       atomic.Int64
	healthCheckCount atomic.Int64
	shouldFailFlush  bool
	shouldFailClose  bool
	shouldFailHealth bool
}

func newMockCollector(id string) *mockCollector {
	return &mockCollector{id: id}
}

func (m *mockCollector) Metadata() CollectorMetadata {
	return CollectorMetadata{
		ID:           m.id,
		Type:         CollectorTypeCustom,
		Description:  "Mock collector for testing",
		RegisteredAt: time.Now(),
		Version:      "test-1.0",
	}
}

func (m *mockCollector) Flush() error {
	m.flushCount.Add(1)
	if m.shouldFailFlush {
		return errors.New("mock flush failure")
	}
	return nil
}

func (m *mockCollector) Close() error {
	m.closeCount.Add(1)
	if m.shouldFailClose {
		return errors.New("mock close failure")
	}
	return nil
}

func (m *mockCollector) Health() error {
	m.healthCheckCount.Add(1)
	if m.shouldFailHealth {
		return errors.New("mock health failure")
	}
	return nil
}

// Implement minimal Metrics interface methods for testing
func (m *mockCollector) IncDelegationsCreated() {
	m.callCount.Add(1)
}

// All other Metrics methods are handled by the embedded noop struct

// TestCollectorRegistry_Registration tests basic registration and deregistration.
func TestCollectorRegistry_Registration(t *testing.T) {
	registry := NewCollectorRegistry(false)

	// Register collector
	c1 := newMockCollector("collector-1")
	if err := registry.Register(c1); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	// Verify registration
	if got := registry.Get("collector-1"); got != c1 {
		t.Errorf("Get returned wrong collector: got %v, want %v", got, c1)
	}

	// List collectors
	list := registry.List()
	if len(list) != 1 {
		t.Errorf("List returned %d collectors, want 1", len(list))
	}
	if list[0].ID != "collector-1" {
		t.Errorf("List returned collector %q, want %q", list[0].ID, "collector-1")
	}

	// Duplicate registration should fail
	if err := registry.Register(c1); err == nil {
		t.Error("Duplicate registration should fail")
	}

	// Deregister
	if err := registry.Deregister("collector-1"); err != nil {
		t.Fatalf("Deregister failed: %v", err)
	}

	// Verify deregistration
	if got := registry.Get("collector-1"); got != nil {
		t.Errorf("Get after deregister returned %v, want nil", got)
	}

	// Verify flush and close were called
	if c1.flushCount.Load() != 1 {
		t.Errorf("Flush called %d times, want 1", c1.flushCount.Load())
	}
	if c1.closeCount.Load() != 1 {
		t.Errorf("Close called %d times, want 1", c1.closeCount.Load())
	}

	// Deregister non-existent collector should fail
	if err := registry.Deregister("nonexistent"); err == nil {
		t.Error("Deregister of nonexistent collector should fail")
	}
}

// TestCollectorRegistry_MultiCollectorDispatch tests that metrics are dispatched to all collectors.
func TestCollectorRegistry_MultiCollectorDispatch(t *testing.T) {
	registry := NewCollectorRegistry(false)

	// Register multiple collectors
	c1 := newMockCollector("collector-1")
	c2 := newMockCollector("collector-2")
	c3 := newMockCollector("collector-3")

	if err := registry.Register(c1); err != nil {
		t.Fatalf("Register c1 failed: %v", err)
	}
	if err := registry.Register(c2); err != nil {
		t.Fatalf("Register c2 failed: %v", err)
	}
	if err := registry.Register(c3); err != nil {
		t.Fatalf("Register c3 failed: %v", err)
	}

	// Call metric method
	registry.IncDelegationsCreated()

	// Verify all collectors received the call
	if c1.callCount.Load() != 1 {
		t.Errorf("c1.callCount = %d, want 1", c1.callCount.Load())
	}
	if c2.callCount.Load() != 1 {
		t.Errorf("c2.callCount = %d, want 1", c2.callCount.Load())
	}
	if c3.callCount.Load() != 1 {
		t.Errorf("c3.callCount = %d, want 1", c3.callCount.Load())
	}

	// Call multiple times
	for i := 0; i < 10; i++ {
		registry.IncDelegationsCreated()
	}

	if c1.callCount.Load() != 11 {
		t.Errorf("c1.callCount = %d, want 11", c1.callCount.Load())
	}
	if c2.callCount.Load() != 11 {
		t.Errorf("c2.callCount = %d, want 11", c2.callCount.Load())
	}
	if c3.callCount.Load() != 11 {
		t.Errorf("c3.callCount = %d, want 11", c3.callCount.Load())
	}
}

// TestCollectorRegistry_ConcurrentDispatch tests concurrent dispatch mode.
func TestCollectorRegistry_ConcurrentDispatch(t *testing.T) {
	registry := NewCollectorRegistry(true) // Enable concurrent dispatch

	// Register collectors
	c1 := newMockCollector("collector-1")
	c2 := newMockCollector("collector-2")

	if err := registry.Register(c1); err != nil {
		t.Fatalf("Register c1 failed: %v", err)
	}
	if err := registry.Register(c2); err != nil {
		t.Fatalf("Register c2 failed: %v", err)
	}

	// Call metric methods concurrently
	const numCalls = 100
	var wg sync.WaitGroup
	for i := 0; i < numCalls; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			registry.IncDelegationsCreated()
		}()
	}
	wg.Wait()

	// Verify counts (may have slight variation due to concurrency)
	if c1.callCount.Load() != numCalls {
		t.Errorf("c1.callCount = %d, want %d", c1.callCount.Load(), numCalls)
	}
	if c2.callCount.Load() != numCalls {
		t.Errorf("c2.callCount = %d, want %d", c2.callCount.Load(), numCalls)
	}
}

// TestCollectorRegistry_FlushAll tests flushing all collectors.
func TestCollectorRegistry_FlushAll(t *testing.T) {
	registry := NewCollectorRegistry(false)

	c1 := newMockCollector("collector-1")
	c2 := newMockCollector("collector-2")
	c3 := newMockCollector("collector-3")
	c3.shouldFailFlush = true // Make c3 fail

	if err := registry.Register(c1); err != nil {
		t.Fatalf("failed to register c1: %v", err)
	}
	if err := registry.Register(c2); err != nil {
		t.Fatalf("failed to register c2: %v", err)
	}
	if err := registry.Register(c3); err != nil {
		t.Fatalf("failed to register c3: %v", err)
	}

	errors := registry.FlushAll()

	// c1 and c2 should succeed, c3 should fail
	if len(errors) != 1 {
		t.Errorf("FlushAll returned %d errors, want 1", len(errors))
	}
	if _, ok := errors["collector-3"]; !ok {
		t.Error("FlushAll should report error for collector-3")
	}

	if c1.flushCount.Load() != 1 {
		t.Errorf("c1.flushCount = %d, want 1", c1.flushCount.Load())
	}
	if c2.flushCount.Load() != 1 {
		t.Errorf("c2.flushCount = %d, want 1", c2.flushCount.Load())
	}
	if c3.flushCount.Load() != 1 {
		t.Errorf("c3.flushCount = %d, want 1", c3.flushCount.Load())
	}
}

// TestCollectorRegistry_CloseAll tests closing all collectors.
func TestCollectorRegistry_CloseAll(t *testing.T) {
	registry := NewCollectorRegistry(false)

	c1 := newMockCollector("collector-1")
	c2 := newMockCollector("collector-2")
	c3 := newMockCollector("collector-3")
	c3.shouldFailClose = true // Make c3 fail

	_ = registry.Register(c1)
	_ = registry.Register(c2)
	_ = registry.Register(c3)

	errors := registry.CloseAll()

	// c1 and c2 should succeed, c3 should fail
	if len(errors) != 1 {
		t.Errorf("CloseAll returned %d errors, want 1", len(errors))
	}
	if _, ok := errors["collector-3"]; !ok {
		t.Error("CloseAll should report error for collector-3")
	}

	// Registry should be empty after CloseAll
	if len(registry.List()) != 0 {
		t.Errorf("Registry has %d collectors after CloseAll, want 0", len(registry.List()))
	}
}

// TestCollectorRegistry_HealthCheck tests health checking.
func TestCollectorRegistry_HealthCheck(t *testing.T) {
	registry := NewCollectorRegistry(false)

	c1 := newMockCollector("collector-1")
	c2 := newMockCollector("collector-2")
	c3 := newMockCollector("collector-3")
	c3.shouldFailHealth = true // Make c3 unhealthy

	registry.Register(c1) //nolint:errcheck
	registry.Register(c2) //nolint:errcheck
	registry.Register(c3) //nolint:errcheck

	errors := registry.HealthCheck()

	// c1 and c2 should be healthy, c3 should fail
	if len(errors) != 1 {
		t.Errorf("HealthCheck returned %d errors, want 1", len(errors))
	}
	if _, ok := errors["collector-3"]; !ok {
		t.Error("HealthCheck should report error for collector-3")
	}

	if c1.healthCheckCount.Load() != 1 {
		t.Errorf("c1.healthCheckCount = %d, want 1", c1.healthCheckCount.Load())
	}
	if c2.healthCheckCount.Load() != 1 {
		t.Errorf("c2.healthCheckCount = %d, want 1", c2.healthCheckCount.Load())
	}
	if c3.healthCheckCount.Load() != 1 {
		t.Errorf("c3.healthCheckCount = %d, want 1", c3.healthCheckCount.Load())
	}
}

// TestCollectorRegistry_ConcurrentRegistration tests concurrent registration/deregistration.
func TestCollectorRegistry_ConcurrentRegistration(t *testing.T) {
	registry := NewCollectorRegistry(true)

	const numCollectors = 50
	var wg sync.WaitGroup

	// Concurrent registration
	for i := 0; i < numCollectors; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c := newMockCollector(time.Now().Format("collector-" + time.Now().String()))
			_ = registry.Register(c)
		}(i)
	}
	wg.Wait()

	// Verify some collectors were registered
	list := registry.List()
	if len(list) == 0 {
		t.Error("No collectors registered")
	}

	// Concurrent metric calls during registration/deregistration
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				registry.IncDelegationsCreated()
			}
		}
	}()

	// Concurrent deregistration
	for _, meta := range list {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			_ = registry.Deregister(id)
		}(meta.ID)
	}
	wg.Wait()
	close(done)

	// Registry should be empty
	if len(registry.List()) != 0 {
		t.Errorf("Registry has %d collectors after deregistration, want 0", len(registry.List()))
	}
}

// TestCollectorRegistry_ZeroCollectors tests that registry works with no collectors.
func TestCollectorRegistry_ZeroCollectors(t *testing.T) {
	registry := NewCollectorRegistry(false)

	// Should not panic with no collectors
	registry.IncDelegationsCreated()
	registry.ObserveValidationLatency(time.Millisecond)

	errors := registry.FlushAll()
	if len(errors) != 0 {
		t.Errorf("FlushAll with no collectors returned %d errors, want 0", len(errors))
	}

	errors = registry.CloseAll()
	if len(errors) != 0 {
		t.Errorf("CloseAll with no collectors returned %d errors, want 0", len(errors))
	}

	errors = registry.HealthCheck()
	if len(errors) != 0 {
		t.Errorf("HealthCheck with no collectors returned %d errors, want 0", len(errors))
	}
}
