package oidc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// CachedDiscovery represents a cached OIDC discovery document.
type CachedDiscovery struct {
	// Document is the cached discovery document
	Document *OIDCConfiguration

	// FetchedAt is when the document was retrieved
	FetchedAt time.Time

	// ExpiresAt is when the cache entry expires
	ExpiresAt time.Time

	// ETag is the HTTP ETag for conditional requests
	ETag string
}

// IsExpired checks if the cache entry has expired.
func (c *CachedDiscovery) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// ShouldRefresh checks if the cache entry should be refreshed soon.
// Returns true if less than 10% of TTL remains.
func (c *CachedDiscovery) ShouldRefresh() bool {
	now := time.Now()
	ttl := c.ExpiresAt.Sub(c.FetchedAt)
	remainingTTL := c.ExpiresAt.Sub(now)
	return remainingTTL < ttl/10
}

// DiscoveryCache manages cached OIDC discovery documents.
type DiscoveryCache interface {
	// Get retrieves a discovery document from cache or fetches it
	Get(ctx context.Context, issuerURL string) (*OIDCConfiguration, error)

	// Set adds or updates a discovery document in the cache
	Set(issuerURL string, doc *OIDCConfiguration, ttl time.Duration) error

	// Invalidate removes a discovery document from the cache
	Invalidate(issuerURL string) error

	// Clear removes all entries from the cache
	Clear() error
}

// InMemoryDiscoveryCache implements DiscoveryCache using in-memory storage.
type InMemoryDiscoveryCache struct {
	mu         sync.RWMutex
	cache      map[string]*CachedDiscovery
	httpClient *http.Client
	defaultTTL time.Duration
	maxEntries int
}

// DiscoveryCacheOption configures the discovery cache.
type DiscoveryCacheOption func(*InMemoryDiscoveryCache)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) DiscoveryCacheOption {
	return func(c *InMemoryDiscoveryCache) {
		c.httpClient = client
	}
}

// WithDefaultTTL sets the default cache TTL.
func WithDefaultTTL(ttl time.Duration) DiscoveryCacheOption {
	return func(c *InMemoryDiscoveryCache) {
		c.defaultTTL = ttl
	}
}

// WithMaxEntries sets the maximum number of cache entries.
func WithMaxEntries(max int) DiscoveryCacheOption {
	return func(c *InMemoryDiscoveryCache) {
		c.maxEntries = max
	}
}

// NewInMemoryDiscoveryCache creates a new in-memory discovery cache.
func NewInMemoryDiscoveryCache(opts ...DiscoveryCacheOption) *InMemoryDiscoveryCache {
	cache := &InMemoryDiscoveryCache{
		cache:      make(map[string]*CachedDiscovery),
		httpClient: &http.Client{Timeout: 10 * time.Second},
		defaultTTL: 24 * time.Hour,
		maxEntries: 100,
	}

	for _, opt := range opts {
		opt(cache)
	}

	return cache
}

// Get retrieves a discovery document from cache or fetches it.
func (c *InMemoryDiscoveryCache) Get(ctx context.Context, issuerURL string) (*OIDCConfiguration, error) {
	// Check cache first
	c.mu.RLock()
	cached, exists := c.cache[issuerURL]
	c.mu.RUnlock()

	// Cache hit - return if not expired
	if exists && !cached.IsExpired() {
		return cached.Document, nil
	}

	// Cache miss or expired - fetch from provider
	discoveryURL := getDiscoveryURL(issuerURL)
	doc, etag, err := c.fetchDiscoveryDocument(ctx, discoveryURL)
	if err != nil {
		// If cache exists but expired, return stale data on error
		if exists {
			return cached.Document, nil
		}
		return nil, fmt.Errorf("failed to fetch discovery document: %w", err)
	}

	// Update cache
	if err := c.Set(issuerURL, doc, c.defaultTTL); err != nil {
		// Log error but return the document anyway
		return doc, nil
	}

	// Update ETag
	c.mu.Lock()
	if entry, ok := c.cache[issuerURL]; ok {
		entry.ETag = etag
	}
	c.mu.Unlock()

	return doc, nil
}

// Set adds or updates a discovery document in the cache.
func (c *InMemoryDiscoveryCache) Set(issuerURL string, doc *OIDCConfiguration, ttl time.Duration) error {
	if doc == nil {
		return fmt.Errorf("discovery document cannot be nil")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Enforce max entries limit
	if len(c.cache) >= c.maxEntries {
		// Remove oldest entry
		c.evictOldest()
	}

	now := time.Now()
	c.cache[issuerURL] = &CachedDiscovery{
		Document:  doc,
		FetchedAt: now,
		ExpiresAt: now.Add(ttl),
	}

	return nil
}

// Invalidate removes a discovery document from the cache.
func (c *InMemoryDiscoveryCache) Invalidate(issuerURL string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, issuerURL)
	return nil
}

// Clear removes all entries from the cache.
func (c *InMemoryDiscoveryCache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache = make(map[string]*CachedDiscovery)
	return nil
}

// evictOldest removes the oldest cache entry (must be called with lock held).
func (c *InMemoryDiscoveryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestKey == "" || entry.FetchedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.FetchedAt
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

// fetchDiscoveryDocument fetches a discovery document from the provider.
func (c *InMemoryDiscoveryCache) fetchDiscoveryDocument(ctx context.Context, discoveryURL string) (*OIDCConfiguration, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	var doc OIDCConfiguration
	if err := json.Unmarshal(body, &doc); err != nil {
		return nil, "", fmt.Errorf("failed to parse discovery document: %w", err)
	}

	// Get ETag from response headers
	etag := resp.Header.Get("ETag")

	return &doc, etag, nil
}

// getDiscoveryURL constructs the OIDC discovery URL from an issuer URL.
func getDiscoveryURL(issuerURL string) string {
	// Remove trailing slash if present
	if len(issuerURL) > 0 && issuerURL[len(issuerURL)-1] == '/' {
		issuerURL = issuerURL[:len(issuerURL)-1]
	}
	return issuerURL + "/.well-known/openid-configuration"
}
