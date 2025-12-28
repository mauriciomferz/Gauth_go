// Package mcp provides connection pooling for MCP clients
package mcp

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// PoolConfig defines connection pool configuration
type PoolConfig struct {
	MaxConnections       int           // Maximum connections per server
	MaxIdleTime          time.Duration // Max time a connection can be idle
	ConnectionTimeout    time.Duration // Timeout for creating new connections
	HealthCheckPeriod    time.Duration // Period for health checks
	EnableCircuitBreaker bool          // Enable circuit breaker pattern
}

// DefaultPoolConfig returns default pool configuration
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		MaxConnections:       10,
		MaxIdleTime:          5 * time.Minute,
		ConnectionTimeout:    30 * time.Second,
		HealthCheckPeriod:    1 * time.Minute,
		EnableCircuitBreaker: true,
	}
}

// ConnectionPool manages a pool of MCP client connections
type ConnectionPool struct {
	config       *PoolConfig
	servers      map[string]*ServerConfig
	pools        map[string]*serverPool
	rateLimiters map[string]*RateLimiter
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

// serverPool manages connections for a single server
type serverPool struct {
	config         *ServerConfig
	clients        []*pooledClient
	available      chan *pooledClient
	mu             sync.Mutex
	activeCount    int
	totalCreated   int64
	totalClosed    int64
	circuitBreaker *CircuitBreaker
}

// pooledClient wraps an MCP client with metadata
type pooledClient struct {
	client     *MCPClient
	transport  Transport
	createdAt  time.Time
	lastUsedAt time.Time
	useCount   int64
	inUse      bool
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	mu           sync.RWMutex
	failures     int
	lastFailure  time.Time
	state        string // "closed", "open", "half-open"
}

// RateLimiter implements token bucket rate limiting
type RateLimiter struct {
	rate       int     // requests per second
	burst      int     // max burst size
	tokens     float64 // current tokens
	lastRefill time.Time
	mu         sync.Mutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(config *PoolConfig) *ConnectionPool {
	if config == nil {
		config = DefaultPoolConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	pool := &ConnectionPool{
		config:       config,
		servers:      make(map[string]*ServerConfig),
		pools:        make(map[string]*serverPool),
		rateLimiters: make(map[string]*RateLimiter),
		ctx:          ctx,
		cancel:       cancel,
	}

	// Start health check goroutine
	pool.wg.Add(1)
	go pool.healthCheckLoop()

	return pool
}

// RegisterServer registers a server and creates its pool
func (p *ConnectionPool) RegisterServer(config *ServerConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if config.ID == "" {
		return fmt.Errorf("server ID is required")
	}

	if config.Name == "" {
		return fmt.Errorf("server name is required")
	}

	if config.TransportType == "" {
		return fmt.Errorf("transport type is required")
	}

	// Validate transport-specific fields
	switch config.TransportType {
	case "stdio":
		if config.Command == "" {
			return fmt.Errorf("command is required for stdio transport")
		}
	case "websocket", "http-sse":
		if config.URL == "" {
			return fmt.Errorf("URL is required for %s transport", config.TransportType)
		}
	default:
		return fmt.Errorf("unsupported transport type: %s", config.TransportType)
	}

	// Create server pool
	pool := &serverPool{
		config:    config,
		clients:   make([]*pooledClient, 0, p.config.MaxConnections),
		available: make(chan *pooledClient, p.config.MaxConnections),
	}

	// Create circuit breaker if enabled
	if p.config.EnableCircuitBreaker {
		pool.circuitBreaker = NewCircuitBreaker(5, 30*time.Second)
	}

	p.servers[config.ID] = config
	p.pools[config.ID] = pool

	// Create rate limiter (100 req/sec, burst 200)
	p.rateLimiters[config.ID] = NewRateLimiter(100, 200)

	return nil
}

// GetClient acquires a client from the pool
func (p *ConnectionPool) GetClient(ctx context.Context, serverID string) (*MCPClient, func(), error) {
	p.mu.RLock()
	pool, exists := p.pools[serverID]
	rateLimiter := p.rateLimiters[serverID]
	p.mu.RUnlock()

	if !exists {
		return nil, nil, fmt.Errorf("server not registered: %s", serverID)
	}

	// Check rate limit
	if !rateLimiter.Allow() {
		return nil, nil, fmt.Errorf("rate limit exceeded for server: %s", serverID)
	}

	// Check circuit breaker
	if pool.circuitBreaker != nil && !pool.circuitBreaker.Allow() {
		return nil, nil, fmt.Errorf("circuit breaker open for server: %s", serverID)
	}

	// Try to get available client
	select {
	case client := <-pool.available:
		client.lastUsedAt = time.Now()
		client.useCount++
		client.inUse = true

		releaseFunc := func() {
			p.releaseClient(serverID, client)
		}

		return client.client, releaseFunc, nil

	default:
		// No available client, try to create new one
		client, err := p.createClient(ctx, pool)
		if err != nil {
			if pool.circuitBreaker != nil {
				pool.circuitBreaker.RecordFailure()
			}
			return nil, nil, err
		}

		if pool.circuitBreaker != nil {
			pool.circuitBreaker.RecordSuccess()
		}

		releaseFunc := func() {
			p.releaseClient(serverID, client)
		}

		return client.client, releaseFunc, nil
	}
}

// createClient creates a new pooled client
func (p *ConnectionPool) createClient(ctx context.Context, pool *serverPool) (*pooledClient, error) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	// Check if we've hit max connections
	if pool.activeCount >= p.config.MaxConnections {
		return nil, fmt.Errorf("max connections (%d) reached", p.config.MaxConnections)
	}

	config := pool.config

	// Create transport with timeout
	ctx, cancel := context.WithTimeout(ctx, p.config.ConnectionTimeout)
	defer cancel()

	var transport Transport
	var err error

	switch config.TransportType {
	case "stdio":
		transport, err = NewStdioTransport(ctx, config.Command, config.Args...)

	case "websocket":
		wsTransport := NewWebSocketTransport(config.URL, http.Header{})
		err = wsTransport.Connect(ctx)
		if err == nil {
			transport = wsTransport
		}

	case "http-sse":
		sseTransport := NewSSETransport(config.URL, http.Header{})
		err = sseTransport.Connect(ctx)
		if err == nil {
			transport = sseTransport
		}

	default:
		return nil, fmt.Errorf("unsupported transport type: %s", config.TransportType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	// Create client
	client := NewMCPClient(config.ID, config.Name, transport)

	pooled := &pooledClient{
		client:     client,
		transport:  transport,
		createdAt:  time.Now(),
		lastUsedAt: time.Now(),
		useCount:   1,
		inUse:      true,
	}

	pool.clients = append(pool.clients, pooled)
	pool.activeCount++
	pool.totalCreated++

	return pooled, nil
}

// releaseClient returns a client to the pool
func (p *ConnectionPool) releaseClient(serverID string, client *pooledClient) {
	p.mu.RLock()
	pool := p.pools[serverID]
	p.mu.RUnlock()

	if pool == nil {
		return
	}

	client.inUse = false

	// Check if client should be closed (idle timeout)
	if time.Since(client.lastUsedAt) > p.config.MaxIdleTime {
		p.closeClient(pool, client)
		return
	}

	// Return to available pool
	select {
	case pool.available <- client:
	default:
		// Pool is full, close this client
		p.closeClient(pool, client)
	}
}

// closeClient closes a pooled client
func (p *ConnectionPool) closeClient(pool *serverPool, client *pooledClient) {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	_ = client.client.Close()
	pool.activeCount--
	pool.totalClosed++

	// Remove from clients slice
	for i, c := range pool.clients {
		if c == client {
			pool.clients = append(pool.clients[:i], pool.clients[i+1:]...)
			break
		}
	}
}

// healthCheckLoop periodically checks and cleans up idle connections
func (p *ConnectionPool) healthCheckLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.HealthCheckPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.performHealthCheck()
		}
	}
}

// performHealthCheck checks and cleans up idle connections
func (p *ConnectionPool) performHealthCheck() {
	p.mu.RLock()
	pools := make([]*serverPool, 0, len(p.pools))
	for _, pool := range p.pools {
		pools = append(pools, pool)
	}
	p.mu.RUnlock()

	for _, pool := range pools {
		pool.mu.Lock()
		now := time.Now()

		for i := len(pool.clients) - 1; i >= 0; i-- {
			client := pool.clients[i]
			if !client.inUse && now.Sub(client.lastUsedAt) > p.config.MaxIdleTime {
				p.closeClient(pool, client)
			}
		}

		pool.mu.Unlock()
	}
}

// GetPoolStats returns statistics for a server pool
func (p *ConnectionPool) GetPoolStats(serverID string) map[string]interface{} {
	p.mu.RLock()
	pool := p.pools[serverID]
	rateLimiter := p.rateLimiters[serverID]
	p.mu.RUnlock()

	if pool == nil {
		return nil
	}

	pool.mu.Lock()
	stats := map[string]interface{}{
		"active_connections": pool.activeCount,
		"total_created":      pool.totalCreated,
		"total_closed":       pool.totalClosed,
		"max_connections":    p.config.MaxConnections,
		"available_in_pool":  len(pool.available),
	}

	if pool.circuitBreaker != nil {
		stats["circuit_breaker_state"] = pool.circuitBreaker.GetState()
		stats["circuit_breaker_failures"] = pool.circuitBreaker.failures
	}

	if rateLimiter != nil {
		stats["rate_limiter_tokens"] = rateLimiter.tokens
		stats["rate_limiter_rate"] = rateLimiter.rate
	}

	pool.mu.Unlock()

	return stats
}

// Close closes all connections and shuts down the pool
func (p *ConnectionPool) Close() error {
	p.cancel()
	p.wg.Wait()

	p.mu.Lock()
	defer p.mu.Unlock()

	for _, pool := range p.pools {
		pool.mu.Lock()
		for _, client := range pool.clients {
			_ = client.client.Close()
		}
		pool.clients = nil
		close(pool.available)
		pool.mu.Unlock()
	}

	return nil
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        "closed",
	}
}

// Allow checks if request should be allowed
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "closed" {
		return true
	}

	if cb.state == "open" {
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			cb.state = "half-open"
			return true
		}
		return false
	}

	// half-open state
	return true
}

// RecordSuccess records a successful operation
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == "half-open" {
		cb.state = "closed"
		cb.failures = 0
	}
}

// RecordFailure records a failed operation
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = "open"
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// NewRateLimiter creates a new token bucket rate limiter
func NewRateLimiter(rate, burst int) *RateLimiter {
	return &RateLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastRefill: time.Now(),
	}
}

// Allow checks if request should be allowed (token bucket algorithm)
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastRefill).Seconds()

	// Refill tokens based on elapsed time
	rl.tokens += elapsed * float64(rl.rate)
	if rl.tokens > float64(rl.burst) {
		rl.tokens = float64(rl.burst)
	}
	rl.lastRefill = now

	// Check if we have tokens available
	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}

	return false
}
