// Package loadtest provides authorization-specific load testing.
package loadtest

import (
	"context"
	"fmt"
	"math/rand"
	"time"
)

// AuthorizationRequest represents an authorization check request.
type AuthorizationRequest struct {
	Subject  string
	Resource string
	Action   string
	Context  map[string]interface{}
}

// AuthorizationResponse represents an authorization check response.
type AuthorizationResponse struct {
	Decision string
	Reason   string
	Duration time.Duration
}

// AuthorizationRequestGenerator generates realistic authorization requests.
type AuthorizationRequestGenerator struct {
	Subjects  []string
	Resources []string
	Actions   []string
	rng       *rand.Rand
}

// NewAuthorizationRequestGenerator creates a new authorization request generator.
func NewAuthorizationRequestGenerator(subjects, resources, actions []string) *AuthorizationRequestGenerator {
	return &AuthorizationRequestGenerator{
		Subjects:  subjects,
		Resources: resources,
		Actions:   actions,
		rng:       rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate creates an authorization request.
func (g *AuthorizationRequestGenerator) Generate(userID, iteration int) (interface{}, error) {
	// Mix of patterns: some predictable, some random
	var subject, resource, action string

	if iteration%3 == 0 {
		// Predictable pattern for cache hit testing
		subject = g.Subjects[userID%len(g.Subjects)]
		resource = g.Resources[0]
		action = g.Actions[0]
	} else {
		// Random for realistic workload
		subject = g.Subjects[g.rng.Intn(len(g.Subjects))]
		resource = g.Resources[g.rng.Intn(len(g.Resources))]
		action = g.Actions[g.rng.Intn(len(g.Actions))]
	}

	return &AuthorizationRequest{
		Subject:  subject,
		Resource: resource,
		Action:   action,
		Context: map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"user_id":   userID,
			"iteration": iteration,
		},
	}, nil
}

// AuthorizationValidator validates authorization responses.
type AuthorizationValidator struct {
	ExpectedSuccessRate float64
}

// Validate checks if the authorization response is valid.
func (v *AuthorizationValidator) Validate(request, response interface{}, err error) error {
	if err != nil {
		return fmt.Errorf("authorization error: %w", err)
	}

	authResp, ok := response.(*AuthorizationResponse)
	if !ok {
		return fmt.Errorf("invalid response type")
	}

	if authResp.Decision != "permit" && authResp.Decision != "deny" {
		return fmt.Errorf("invalid decision: %s", authResp.Decision)
	}

	return nil
}

// DelegationRequestGenerator generates delegation operation requests.
type DelegationRequestGenerator struct {
	Subjects   []string
	Delegates  []string
	Resources  []string
	Actions    []string
	Operations []string // "create", "revoke", "verify"
	rng        *rand.Rand
}

// NewDelegationRequestGenerator creates a delegation request generator.
func NewDelegationRequestGenerator(subjects, delegates, resources, actions []string) *DelegationRequestGenerator {
	return &DelegationRequestGenerator{
		Subjects:   subjects,
		Delegates:  delegates,
		Resources:  resources,
		Actions:    actions,
		Operations: []string{"create", "revoke", "verify"},
		rng:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate creates a delegation request.
func (g *DelegationRequestGenerator) Generate(userID, iteration int) (interface{}, error) {
	operation := g.Operations[g.rng.Intn(len(g.Operations))]

	// Weight operations: 60% verify, 30% create, 10% revoke
	roll := g.rng.Float64()
	if roll < 0.6 {
		operation = "verify"
	} else if roll < 0.9 {
		operation = "create"
	} else {
		operation = "revoke"
	}

	return map[string]interface{}{
		"operation": operation,
		"subject":   g.Subjects[g.rng.Intn(len(g.Subjects))],
		"delegate":  g.Delegates[g.rng.Intn(len(g.Delegates))],
		"resource":  g.Resources[g.rng.Intn(len(g.Resources))],
		"action":    g.Actions[g.rng.Intn(len(g.Actions))],
		"user_id":   userID,
		"iteration": iteration,
	}, nil
}

// CachePressureGenerator generates requests to test cache efficiency.
type CachePressureGenerator struct {
	HotSetSize  int     // Number of frequently accessed items
	ColdSetSize int     // Number of rarely accessed items
	HotSetRatio float64 // Probability of hitting hot set (e.g., 0.8)
	Subjects    []string
	Resources   []string
	Actions     []string
	rng         *rand.Rand
}

// NewCachePressureGenerator creates a cache pressure generator.
func NewCachePressureGenerator(subjects, resources, actions []string, hotSetSize, coldSetSize int, hotSetRatio float64) *CachePressureGenerator {
	return &CachePressureGenerator{
		HotSetSize:  hotSetSize,
		ColdSetSize: coldSetSize,
		HotSetRatio: hotSetRatio,
		Subjects:    subjects,
		Resources:   resources,
		Actions:     actions,
		rng:         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate creates a request targeting hot or cold cache entries.
func (g *CachePressureGenerator) Generate(userID, iteration int) (interface{}, error) {
	var resource string

	if g.rng.Float64() < g.HotSetRatio {
		// Hot set - frequently accessed
		resource = g.Resources[g.rng.Intn(min(g.HotSetSize, len(g.Resources)))]
	} else {
		// Cold set - rarely accessed
		offset := min(g.HotSetSize, len(g.Resources))
		if offset < len(g.Resources) {
			resource = g.Resources[offset+g.rng.Intn(len(g.Resources)-offset)]
		} else {
			resource = g.Resources[g.rng.Intn(len(g.Resources))]
		}
	}

	return &AuthorizationRequest{
		Subject:  g.Subjects[g.rng.Intn(len(g.Subjects))],
		Resource: resource,
		Action:   g.Actions[g.rng.Intn(len(g.Actions))],
		Context: map[string]interface{}{
			"cache_test": true,
			"timestamp":  time.Now().Unix(),
		},
	}, nil
}

// BurstLoadGenerator generates bursty traffic patterns.
type BurstLoadGenerator struct {
	BurstProbability float64 // Probability of burst (e.g., 0.1 = 10% chance)
	BurstMultiplier  int     // Request multiplier during burst
	BaseGenerator    RequestGenerator
	rng              *rand.Rand
}

// NewBurstLoadGenerator creates a burst load generator.
func NewBurstLoadGenerator(baseGen RequestGenerator, burstProb float64, burstMult int) *BurstLoadGenerator {
	return &BurstLoadGenerator{
		BurstProbability: burstProb,
		BurstMultiplier:  burstMult,
		BaseGenerator:    baseGen,
		rng:              rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Generate creates requests with burst patterns.
func (g *BurstLoadGenerator) Generate(userID, iteration int) (interface{}, error) {
	// Check if this is a burst moment
	if g.rng.Float64() < g.BurstProbability {
		// Return batch of requests for burst
		requests := make([]interface{}, g.BurstMultiplier)
		for i := 0; i < g.BurstMultiplier; i++ {
			req, err := g.BaseGenerator.Generate(userID, iteration*g.BurstMultiplier+i)
			if err != nil {
				return nil, err
			}
			requests[i] = req
		}
		return map[string]interface{}{
			"burst":    true,
			"requests": requests,
		}, nil
	}

	// Normal single request
	return g.BaseGenerator.Generate(userID, iteration)
}

// LoadTestSuite defines a comprehensive load test suite.
type LoadTestSuite struct {
	harness *LoadTestHarness
}

// NewLoadTestSuite creates a new load test suite.
func NewLoadTestSuite() *LoadTestSuite {
	return &LoadTestSuite{
		harness: NewLoadTestHarness(),
	}
}

// RunAuthorizationLoadTest runs a standard authorization load test.
func (s *LoadTestSuite) RunAuthorizationLoadTest(ctx context.Context) (*LoadTestResult, error) {
	subjects := generateTestSubjects(100)
	resources := generateTestResources(50)
	actions := []string{"read", "write", "delete", "execute"}

	reqGen := NewAuthorizationRequestGenerator(subjects, resources, actions)
	validator := &AuthorizationValidator{ExpectedSuccessRate: 0.95}

	scenario := &TestScenario{
		Name:             "Authorization-Standard",
		Duration:         1 * time.Minute,
		VirtualUsers:     50,
		RampUpTime:       10 * time.Second,
		ThinkTime:        100 * time.Millisecond,
		RequestGenerator: reqGen,
		Validator:        validator,
	}

	return s.harness.Run(ctx, scenario)
}

// RunHighVolumeAuthorizationTest runs a high-volume authorization test.
func (s *LoadTestSuite) RunHighVolumeAuthorizationTest(ctx context.Context) (*LoadTestResult, error) {
	subjects := generateTestSubjects(1000)
	resources := generateTestResources(500)
	actions := []string{"read", "write", "delete", "execute", "admin"}

	reqGen := NewAuthorizationRequestGenerator(subjects, resources, actions)
	validator := &AuthorizationValidator{ExpectedSuccessRate: 0.95}

	scenario := &TestScenario{
		Name:             "Authorization-HighVolume",
		Duration:         2 * time.Minute,
		VirtualUsers:     200,
		RampUpTime:       20 * time.Second,
		ThinkTime:        50 * time.Millisecond,
		RequestGenerator: reqGen,
		Validator:        validator,
	}

	return s.harness.Run(ctx, scenario)
}

// RunCacheEfficiencyTest tests cache hit rates.
func (s *LoadTestSuite) RunCacheEfficiencyTest(ctx context.Context) (*LoadTestResult, error) {
	subjects := generateTestSubjects(100)
	resources := generateTestResources(1000)
	actions := []string{"read", "write"}

	// 80% hot set (first 100 resources), 20% cold set (remaining 900)
	reqGen := NewCachePressureGenerator(subjects, resources, actions, 100, 900, 0.8)

	scenario := &TestScenario{
		Name:             "Cache-Efficiency",
		Duration:         1 * time.Minute,
		VirtualUsers:     100,
		RampUpTime:       10 * time.Second,
		ThinkTime:        10 * time.Millisecond,
		RequestGenerator: reqGen,
	}

	return s.harness.Run(ctx, scenario)
}

// RunDelegationLoadTest tests delegation operations.
func (s *LoadTestSuite) RunDelegationLoadTest(ctx context.Context) (*LoadTestResult, error) {
	subjects := generateTestSubjects(50)
	delegates := generateTestSubjects(200)
	resources := generateTestResources(100)
	actions := []string{"read", "write"}

	reqGen := NewDelegationRequestGenerator(subjects, delegates, resources, actions)

	scenario := &TestScenario{
		Name:             "Delegation-Operations",
		Duration:         1 * time.Minute,
		VirtualUsers:     30,
		RampUpTime:       10 * time.Second,
		ThinkTime:        200 * time.Millisecond,
		RequestGenerator: reqGen,
	}

	return s.harness.Run(ctx, scenario)
}

// RunBurstTrafficTest tests handling of bursty traffic.
func (s *LoadTestSuite) RunBurstTrafficTest(ctx context.Context) (*LoadTestResult, error) {
	subjects := generateTestSubjects(100)
	resources := generateTestResources(50)
	actions := []string{"read", "write"}

	baseGen := NewAuthorizationRequestGenerator(subjects, resources, actions)
	burstGen := NewBurstLoadGenerator(baseGen, 0.1, 5)

	scenario := &TestScenario{
		Name:             "Burst-Traffic",
		Duration:         1 * time.Minute,
		VirtualUsers:     50,
		RampUpTime:       10 * time.Second,
		ThinkTime:        100 * time.Millisecond,
		RequestGenerator: burstGen,
	}

	return s.harness.Run(ctx, scenario)
}

// RunFullSuite runs all load tests in the suite.
func (s *LoadTestSuite) RunFullSuite(ctx context.Context) (map[string]*LoadTestResult, error) {
	results := make(map[string]*LoadTestResult)

	tests := []struct {
		name string
		fn   func(context.Context) (*LoadTestResult, error)
	}{
		{"Authorization-Standard", s.RunAuthorizationLoadTest},
		{"Authorization-HighVolume", s.RunHighVolumeAuthorizationTest},
		{"Cache-Efficiency", s.RunCacheEfficiencyTest},
		{"Delegation-Operations", s.RunDelegationLoadTest},
		{"Burst-Traffic", s.RunBurstTrafficTest},
	}

	for _, test := range tests {
		fmt.Printf("\n🚀 Starting test: %s\n", test.name)
		result, err := test.fn(ctx)
		if err != nil {
			return results, fmt.Errorf("test %s failed: %w", test.name, err)
		}
		results[test.name] = result
		s.harness.PrintResult(result)

		// Cool-down period between tests
		time.Sleep(5 * time.Second)
	}

	return results, nil
}

// Helper functions

func generateTestSubjects(count int) []string {
	subjects := make([]string, count)
	for i := 0; i < count; i++ {
		subjects[i] = fmt.Sprintf("user:%d", i)
	}
	return subjects
}

func generateTestResources(count int) []string {
	resources := make([]string, count)
	for i := 0; i < count; i++ {
		resources[i] = fmt.Sprintf("resource:%d", i)
	}
	return resources
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
