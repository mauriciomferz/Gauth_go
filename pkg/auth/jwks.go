package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"
)

// JWKSClient fetches and caches JSON Web Key Sets.
type JWKSClient struct {
	uri        string
	httpClient *http.Client
	cacheTTL   time.Duration

	mu          sync.RWMutex
	keys        map[string]interface{}
	lastRefresh time.Time
}

// NewJWKSClient creates a new client for a specific JWKS URI.
func NewJWKSClient(uri string, ttl time.Duration) *JWKSClient {
	if ttl == 0 {
		ttl = 1 * time.Hour
	}
	return &JWKSClient{
		uri:        uri,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cacheTTL:   ttl,
		keys:       make(map[string]interface{}),
	}
}

// GetKey retrieves a public key by Key ID (kid).
// It refreshes the cache if the key is missing or the cache is stale.
func (c *JWKSClient) GetKey(ctx context.Context, kid string) (interface{}, error) {
	// 1. Try generic read lock
	c.mu.RLock()
	cachedKey, ok := c.keys[kid]
	lastRef := c.lastRefresh
	c.mu.RUnlock()

	// If found and cache is fresh, return
	if ok && time.Since(lastRef) < c.cacheTTL {
		return cachedKey, nil
	}

	// 2. Refresh needed (missing or stale)
	// If missing, we force refresh. If present but stale, we refresh.
	// We use double-checked locking to avoid stampede.
	c.mu.Lock()
	defer c.mu.Unlock()

	// Re-check after lock
	if refreshedKey, keyFound := c.keys[kid]; keyFound && time.Since(c.lastRefresh) < c.cacheTTL {
		return refreshedKey, nil
	}

	if err := c.refresh(ctx); err != nil {
		// If refresh failed, but we have a stale key, should we return it?
		// Fallback to stale if available
		if ok {
			return cachedKey, nil
		}
		return nil, fmt.Errorf("failed to fetch JWKS and key not cached: %w", err)
	}

	// 3. Retry lookup
	if refreshedKey, keyFound := c.keys[kid]; keyFound {
		return refreshedKey, nil
	}

	return nil, fmt.Errorf("key %s not found in JWKS", kid)
}

func (c *JWKSClient) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.uri, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	var jwks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode jwks: %w", err)
	}

	// Update cache
	newKeys := make(map[string]interface{})
	for _, k := range jwks.Keys {
		newKeys[k.KeyID] = k.Key
	}

	c.keys = newKeys
	c.lastRefresh = time.Now()
	return nil
}

// UnsafeSetKeys replaces the in-memory keys (useful for testing/mocking)
func (c *JWKSClient) UnsafeSetKeys(keys map[string]interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = keys
	c.lastRefresh = time.Now()
}
