// Package loadtest tests.
package loadtest

import (
	"context"
	"testing"
	"time"
)

func TestLoadTestHarness_BasicScenario(t *testing.T) {
	harness := NewLoadTestHarness()

	reqGen := &SimpleRequestGenerator{
		RequestTemplate: "test-request",
	}

	scenario := &TestScenario{
		Name:             "Basic-Test",
		Duration:         2 * time.Second,
		VirtualUsers:     5,
		RampUpTime:       500 * time.Millisecond,
		ThinkTime:        100 * time.Millisecond,
		RequestGenerator: reqGen,
		Validator:        &AlwaysValidValidator{},
	}

	ctx := context.Background()
	result, err := harness.Run(ctx, scenario)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.TotalRequests == 0 {
		t.Error("Expected some requests, got 0")
	}

	if result.Duration == 0 {
		t.Error("Expected non-zero duration")
	}

	t.Logf("Total requests: %d", result.TotalRequests)
	t.Logf("Successful: %d", result.SuccessfulReqs)
	t.Logf("RPS: %.2f", result.RequestsPerSec)
}

func TestLoadTestHarness_HighConcurrency(t *testing.T) {
	harness := NewLoadTestHarness()

	reqGen := &SimpleRequestGenerator{
		RequestTemplate: "concurrent-test",
	}

	scenario := &TestScenario{
		Name:             "High-Concurrency",
		Duration:         3 * time.Second,
		VirtualUsers:     50,
		RampUpTime:       1 * time.Second,
		ThinkTime:        50 * time.Millisecond,
		RequestGenerator: reqGen,
		Validator:        &AlwaysValidValidator{},
	}

	ctx := context.Background()
	result, err := harness.Run(ctx, scenario)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.TotalRequests < 50 {
		t.Errorf("Expected at least 50 requests, got %d", result.TotalRequests)
	}

	// Under high concurrency, some requests may fail due to context cancellation
	// when the test duration expires. Allow up to 1% failure rate.
	const maxAcceptableFailureRate = 0.01

	failureRate := float64(result.FailedReqs) / float64(result.TotalRequests)
	successRate := float64(result.SuccessfulReqs) / float64(result.TotalRequests)

	if failureRate > maxAcceptableFailureRate {
		// Log details about failures for debugging
		t.Logf("Failed requests: %d/%d (%.2f%%)",
			result.FailedReqs, result.TotalRequests, failureRate*100)
		t.Logf("Successful requests: %d/%d (%.2f%%)",
			result.SuccessfulReqs, result.TotalRequests, successRate*100)

		t.Errorf("High failure rate: %.2f%% exceeds acceptable threshold of %.2f%% (%d/%d requests failed)",
			failureRate*100, maxAcceptableFailureRate*100,
			result.FailedReqs, result.TotalRequests)
	}

	// Ensure we have a reasonable success rate (>99%)
	if successRate < 0.99 {
		t.Errorf("Success rate too low: %.2f%% (expected ≥99%%) - %d/%d succeeded",
			successRate*100, result.SuccessfulReqs, result.TotalRequests)
	}

	t.Logf("Total requests: %d", result.TotalRequests)
	t.Logf("Successful: %d (%.2f%%)", result.SuccessfulReqs, successRate*100)
	t.Logf("Failed: %d (%.2f%%)", result.FailedReqs, failureRate*100)
	t.Logf("RPS: %.2f", result.RequestsPerSec)
	t.Logf("Avg response time: %v", result.AvgResponseTime)
}

func TestLoadTestHarness_ResponseTimeMetrics(t *testing.T) {
	harness := NewLoadTestHarness()

	scenario := &TestScenario{
		Name:             "Metrics-Test",
		Duration:         1 * time.Second,
		VirtualUsers:     10,
		RampUpTime:       200 * time.Millisecond,
		ThinkTime:        50 * time.Millisecond,
		RequestGenerator: &SimpleRequestGenerator{},
	}

	ctx := context.Background()
	result, err := harness.Run(ctx, scenario)

	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.MinResponseTime == 0 {
		t.Error("Expected non-zero min response time")
	}

	if result.MaxResponseTime == 0 {
		t.Error("Expected non-zero max response time")
	}

	if result.AvgResponseTime == 0 {
		t.Error("Expected non-zero avg response time")
	}

	if result.MinResponseTime > result.AvgResponseTime {
		t.Error("Min should be <= Avg")
	}

	if result.AvgResponseTime > result.MaxResponseTime {
		t.Error("Avg should be <= Max")
	}

	t.Logf("Response times - Min: %v, Avg: %v, Max: %v",
		result.MinResponseTime, result.AvgResponseTime, result.MaxResponseTime)
}

func TestLoadTestHarness_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	harness := NewLoadTestHarness()

	config := &StressTestConfig{
		StartUsers:     10,
		MaxUsers:       50,
		UserIncrement:  10,
		IncrementEvery: 2 * time.Second,
	}

	reqGen := &SimpleRequestGenerator{}

	ctx := context.Background()
	results, err := harness.RunStressTest(ctx, config, reqGen)

	if err != nil {
		t.Fatalf("Stress test failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("Expected stress test results")
	}

	// Verify increasing load
	for i := 1; i < len(results); i++ {
		if results[i].RequestsPerSec < results[i-1].RequestsPerSec*0.8 {
			t.Logf("Warning: RPS decreased from %.2f to %.2f at step %d",
				results[i-1].RequestsPerSec, results[i].RequestsPerSec, i)
		}
	}

	t.Logf("Stress test completed with %d steps", len(results))
}

func TestAuthorizationRequestGenerator(t *testing.T) {
	subjects := []string{"user:1", "user:2", "user:3"}
	resources := []string{"res:a", "res:b"}
	actions := []string{"read", "write"}

	gen := NewAuthorizationRequestGenerator(subjects, resources, actions)

	// Generate multiple requests
	requests := make([]*AuthorizationRequest, 0)
	for i := 0; i < 100; i++ {
		req, err := gen.Generate(i%10, i)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		authReq, ok := req.(*AuthorizationRequest)
		if !ok {
			t.Fatal("Expected AuthorizationRequest")
		}

		requests = append(requests, authReq)
	}

	// Verify variety
	subjectSet := make(map[string]bool)
	resourceSet := make(map[string]bool)
	actionSet := make(map[string]bool)

	for _, req := range requests {
		subjectSet[req.Subject] = true
		resourceSet[req.Resource] = true
		actionSet[req.Action] = true
	}

	if len(subjectSet) < 2 {
		t.Error("Expected variety in subjects")
	}

	if len(resourceSet) < 2 {
		t.Error("Expected variety in resources")
	}

	if len(actionSet) < 2 {
		t.Error("Expected variety in actions")
	}

	t.Logf("Generated %d requests with %d subjects, %d resources, %d actions",
		len(requests), len(subjectSet), len(resourceSet), len(actionSet))
}

func TestDelegationRequestGenerator(t *testing.T) {
	subjects := []string{"user:1", "user:2"}
	delegates := []string{"delegate:a", "delegate:b"}
	resources := []string{"res:1"}
	actions := []string{"read"}

	gen := NewDelegationRequestGenerator(subjects, delegates, resources, actions)

	operations := make(map[string]int)

	// Generate requests and count operations
	for i := 0; i < 100; i++ {
		req, err := gen.Generate(i, i)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		reqMap, ok := req.(map[string]interface{})
		if !ok {
			t.Fatal("Expected map response")
		}

		op, ok := reqMap["operation"].(string)
		if !ok {
			t.Fatal("Expected operation string")
		}

		operations[op]++
	}

	// Verify distribution (verify should be most common)
	if operations["verify"] < operations["create"] {
		t.Error("Expected more verify operations than create")
	}

	if operations["verify"] < operations["revoke"] {
		t.Error("Expected more verify operations than revoke")
	}

	t.Logf("Operation distribution: %v", operations)
}

func TestCachePressureGenerator(t *testing.T) {
	subjects := []string{"user:1", "user:2"}
	resources := generateTestResources(100)
	actions := []string{"read"}

	gen := NewCachePressureGenerator(subjects, resources, actions, 10, 90, 0.8)

	hotSetHits := 0

	// Generate requests and track hot set hits
	for i := 0; i < 100; i++ {
		req, err := gen.Generate(i, i)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		authReq, ok := req.(*AuthorizationRequest)
		if !ok {
			t.Fatal("Expected AuthorizationRequest")
		}

		// Check if resource is in hot set (first 10)
		for j := 0; j < 10; j++ {
			if authReq.Resource == resources[j] {
				hotSetHits++
				break
			}
		}
	}

	hotSetRate := float64(hotSetHits) / 100.0

	// Should be around 0.8 (80%)
	if hotSetRate < 0.7 || hotSetRate > 0.9 {
		t.Errorf("Expected hot set rate around 0.8, got %.2f", hotSetRate)
	}

	t.Logf("Hot set hit rate: %.2f%%", hotSetRate*100)
}

func TestBurstLoadGenerator(t *testing.T) {
	baseGen := &SimpleRequestGenerator{}
	burstGen := NewBurstLoadGenerator(baseGen, 0.3, 5)

	burstCount := 0
	normalCount := 0

	// Generate requests
	for i := 0; i < 100; i++ {
		req, err := burstGen.Generate(i, i)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		reqMap, ok := req.(map[string]interface{})
		if ok && reqMap["burst"] == true {
			burstCount++
		} else {
			normalCount++
		}
	}

	// Should have some bursts (around 30%)
	if burstCount < 20 || burstCount > 40 {
		t.Logf("Warning: burst count %d not near expected 30%%", burstCount)
	}

	t.Logf("Burst requests: %d, Normal requests: %d", burstCount, normalCount)
}

func TestAuthorizationValidator(t *testing.T) {
	validator := &AuthorizationValidator{ExpectedSuccessRate: 0.95}

	// Test valid response
	validResp := &AuthorizationResponse{
		Decision: "permit",
		Reason:   "allowed",
		Duration: 10 * time.Millisecond,
	}

	err := validator.Validate(nil, validResp, nil)
	if err != nil {
		t.Errorf("Expected valid response to pass, got: %v", err)
	}

	// Test invalid decision
	invalidResp := &AuthorizationResponse{
		Decision: "invalid",
		Reason:   "bad",
		Duration: 10 * time.Millisecond,
	}

	err = validator.Validate(nil, invalidResp, nil)
	if err == nil {
		t.Error("Expected invalid decision to fail validation")
	}

	// Test error case
	err = validator.Validate(nil, nil, context.DeadlineExceeded)
	if err == nil {
		t.Error("Expected error to fail validation")
	}
}

func TestLoadTestSuite_AuthorizationTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load test suite in short mode")
	}

	t.Skip("Load test requires running server - use for integration testing")

	suite := NewLoadTestSuite()
	ctx := context.Background()

	// Run a quick test (reduced duration for unit testing)
	result, err := suite.RunAuthorizationLoadTest(ctx)
	if err != nil {
		t.Fatalf("Authorization load test failed: %v", err)
	}

	if result.TotalRequests == 0 {
		t.Error("Expected some requests")
	}

	t.Logf("Authorization test: %d requests, %.2f RPS",
		result.TotalRequests, result.RequestsPerSec)
}

func BenchmarkLoadTestHarness_RequestExecution(b *testing.B) {
	harness := NewLoadTestHarness()
	ctx := context.Background()
	scenario := &TestScenario{
		RequestGenerator: &SimpleRequestGenerator{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		harness.executeRequest(ctx, 0, i, scenario)
	}
}

func BenchmarkAuthorizationRequestGenerator(b *testing.B) {
	subjects := generateTestSubjects(100)
	resources := generateTestResources(100)
	actions := []string{"read", "write", "delete"}

	gen := NewAuthorizationRequestGenerator(subjects, resources, actions)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate(i%10, i)
	}
}
