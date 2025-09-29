package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/internal/circuit"
	"github.com/Gimel-Foundation/gauth/internal/ratelimit"
	"github.com/Gimel-Foundation/gauth/internal/resilience"
)

// APIGateway demonstrates resilience patterns in an API Gateway
type APIGateway struct {
	services    map[string]*BackendService
	globalLimit ratelimit.Algorithm
	routeRules  map[string]*RouteConfig
}

// BackendService represents a downstream service
type BackendService struct {
	name      string
	endpoint  string
	breaker   *circuit.Breaker
	limiter   ratelimit.Algorithm
	retry     *resilience.Retry
	bulkhead  *resilience.Bulkhead
	latency   time.Duration
	errorRate float64
	mu        sync.RWMutex
}

// RouteConfig defines routing and resilience configuration for a path
type RouteConfig struct {
	Path           string
	Service        string
	Timeout        time.Duration
	RateLimit      *ratelimit.Config
	CircuitBreaker circuit.Options
	RetryStrategy  resilience.RetryStrategy
	BulkheadLimit  int
}

// RequestContext holds request-specific information
type RequestContext struct {
	Path        string
	ClientID    string
	StartTime   time.Time
	TraceID     string
	RequestType string
}

func NewAPIGateway() *APIGateway {
	gateway := &APIGateway{
		services:   make(map[string]*BackendService),
		routeRules: make(map[string]*RouteConfig),
	}

	// Configure global rate limiter
	gateway.globalLimit = ratelimit.WrapTokenBucket(&ratelimit.Config{
		RequestsPerSecond: 1000,
		WindowSize:        1,
		BurstSize:         100,
	})

	// Add example backend services
	gateway.AddService("auth", "http://auth-service", 50*time.Millisecond, 0.05)
	gateway.AddService("users", "http://user-service", 100*time.Millisecond, 0.10)
	gateway.AddService("orders", "http://order-service", 150*time.Millisecond, 0.15)

	// Configure routes
	gateway.AddRoute(&RouteConfig{
		Path:    "/api/auth",
		Service: "auth",
		Timeout: 1 * time.Second,
		RateLimit: &ratelimit.Config{
			RequestsPerSecond: 100,
			WindowSize:        1,
			BurstSize:         20,
		},
		CircuitBreaker: circuit.Options{
			Name:             "auth-route",
			FailureThreshold: 5,
			ResetTimeout:     10 * time.Second,
			HalfOpenLimit:    2,
		},
		RetryStrategy: resilience.RetryStrategy{
			MaxAttempts:     3,
			InitialInterval: 50 * time.Millisecond,
			MaxInterval:     500 * time.Millisecond,
			Multiplier:      2.0,
		},
		BulkheadLimit: 20,
	})

	gateway.AddRoute(&RouteConfig{
		Path:    "/api/users",
		Service: "users",
		Timeout: 2 * time.Second,
		RateLimit: &ratelimit.Config{
			RequestsPerSecond: 50,
			WindowSize:        1,
			BurstSize:         10,
		},
		CircuitBreaker: circuit.Options{
			Name:             "users-route",
			FailureThreshold: 3,
			ResetTimeout:     5 * time.Second,
			HalfOpenLimit:    1,
		},
		RetryStrategy: resilience.RetryStrategy{
			MaxAttempts:     2,
			InitialInterval: 100 * time.Millisecond,
			MaxInterval:     1 * time.Second,
			Multiplier:      2.0,
		},
		BulkheadLimit: 10,
	})

	return gateway
}

func (g *APIGateway) AddService(name, endpoint string, latency time.Duration, errorRate float64) {
	svc := &BackendService{
		name:      name,
		endpoint:  endpoint,
		latency:   latency,
		errorRate: errorRate,
	}
	g.services[name] = svc
}

func (g *APIGateway) AddRoute(config *RouteConfig) {
	g.routeRules[config.Path] = config
	svc := g.services[config.Service]
	if svc != nil {
		svc.breaker = circuit.NewBreaker(config.CircuitBreaker)
		svc.limiter = ratelimit.WrapTokenBucket(config.RateLimit)
		svc.retry = resilience.NewRetry(config.RetryStrategy)
		svc.bulkhead = resilience.NewBulkhead(config.BulkheadLimit)
	}
}

func (g *APIGateway) HandleRequest(ctx context.Context, req *RequestContext) error {
	// Apply global rate limiting
	if err := g.globalLimit.Allow(ctx, "global"); err != nil {
		return fmt.Errorf("global rate limit exceeded: %w", err)
	}

	// Get route configuration
	route, exists := g.routeRules[req.Path]
	if !exists {
		return fmt.Errorf("route not found: %s", req.Path)
	}

	// Get service
	svc, exists := g.services[route.Service]
	if !exists {
		return fmt.Errorf("service not found: %s", route.Service)
	}

	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, route.Timeout)
	defer cancel()

	// Execute request through all resilience patterns
	return svc.bulkhead.Execute(ctx, func() error {
		// Service-level rate limiting
		if err := svc.limiter.Allow(ctx, req.ClientID); err != nil {
			return fmt.Errorf("service rate limit exceeded: %w", err)
		}

		return svc.retry.Execute(ctx, func() error {
			return svc.breaker.Execute(func() error {
				return svc.simulateRequest(ctx, req)
			})
		})
	})
}

func (s *BackendService) simulateRequest(ctx context.Context, req *RequestContext) error {
	// Simulate processing time
	select {
	case <-time.After(s.latency):
	case <-ctx.Done():
		return ctx.Err()
	}

	// Simulate random failures using crypto/rand
	var randomBytes [8]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return fmt.Errorf("failed to generate secure random: %w", err)
	}
	randomFloat := float64(randomBytes[0]) / 255.0 // Convert to 0-1 range

	if randomFloat < s.errorRate {
		return fmt.Errorf("%s: service temporarily unavailable", s.name)
	}
	return nil
}

func main() {
	gateway := NewAPIGateway()
	ctx := context.Background()

	// Test scenarios
	var wg sync.WaitGroup
	clients := 20
	requestsPerClient := 5

	fmt.Println("\nStarting API Gateway Demo...")
	fmt.Println("----------------------------------------")
	fmt.Println("Configuration:")
	fmt.Println("- Global Rate Limit: 1000 req/s, burst: 100")
	fmt.Println("- Services: auth, users, orders")
	fmt.Println("- Routes: /api/auth, /api/users")
	fmt.Println("----------------------------------------")

	start := time.Now()

	// Simulate multiple clients making requests
	for client := 1; client <= clients; client++ {
		clientID := fmt.Sprintf("client%d", client)

		for req := 1; req <= requestsPerClient; req++ {
			wg.Add(1)
			go func(clientID string, reqID int) {
				defer wg.Done()

				// Alternate between routes
				path := "/api/auth"
				if reqID%2 == 0 {
					path = "/api/users"
				}

				request := &RequestContext{
					Path:        path,
					ClientID:    clientID,
					StartTime:   time.Now(),
					TraceID:     fmt.Sprintf("trace-%s-%d", clientID, reqID),
					RequestType: "GET",
				}

				err := gateway.HandleRequest(ctx, request)
				duration := time.Since(request.StartTime)

				if err != nil {
					fmt.Printf("[%s] Request %d failed after %v: %v\n",
						clientID, reqID, duration.Round(time.Millisecond), err)
				} else {
					fmt.Printf("[%s] Request %d completed in %v\n",
						clientID, reqID, duration.Round(time.Millisecond))
				}
			}(clientID, req)

			// Add some delay between requests
			time.Sleep(50 * time.Millisecond)
		}
	}

	wg.Wait()
	totalDuration := time.Since(start)
	fmt.Printf("\nAPI Gateway demo completed in %v!\n", totalDuration.Round(time.Millisecond))
}
