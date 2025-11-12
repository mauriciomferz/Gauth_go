package oidc

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCachedDiscovery_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "not expired",
			expiresAt: now.Add(1 * time.Hour),
			want:      false,
		},
		{
			name:      "expired",
			expiresAt: now.Add(-1 * time.Hour),
			want:      true,
		},
		{
			name:      "just expired",
			expiresAt: now.Add(-1 * time.Second),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := &CachedDiscovery{
				ExpiresAt: tt.expiresAt,
			}
			got := cd.IsExpired()
			if got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCachedDiscovery_ShouldRefresh(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		fetchedAt time.Time
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "should not refresh (plenty of time left)",
			fetchedAt: now.Add(-1 * time.Hour),
			expiresAt: now.Add(9 * time.Hour),
			want:      false,
		},
		{
			name:      "should refresh (less than 10% TTL remaining)",
			fetchedAt: now.Add(-9 * time.Hour),
			expiresAt: now.Add(30 * time.Minute),
			want:      true,
		},
		{
			name:      "should refresh (expiring soon)",
			fetchedAt: now.Add(-9*time.Hour - 30*time.Minute),
			expiresAt: now.Add(30 * time.Minute),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cd := &CachedDiscovery{
				FetchedAt: tt.fetchedAt,
				ExpiresAt: tt.expiresAt,
			}
			got := cd.ShouldRefresh()
			if got != tt.want {
				t.Errorf("ShouldRefresh() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInMemoryDiscoveryCache_SetAndGet(t *testing.T) {
	cache := NewInMemoryDiscoveryCache()
	ctx := context.Background()

	issuer := "https://accounts.google.com"
	doc := &OIDCConfiguration{
		Issuer:                issuer,
		AuthorizationEndpoint: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenEndpoint:         "https://oauth2.googleapis.com/token",
	}

	// Set document
	err := cache.Set(issuer, doc, 1*time.Hour)
	if err != nil {
		t.Fatalf("Set() unexpected error = %v", err)
	}

	// Get document (should be cached)
	got, err := cache.Get(ctx, issuer)
	if err != nil {
		t.Fatalf("Get() unexpected error = %v", err)
	}

	if got.Issuer != doc.Issuer {
		t.Errorf("Get() issuer = %v, want %v", got.Issuer, doc.Issuer)
	}
}

func TestInMemoryDiscoveryCache_SetNilDocument(t *testing.T) {
	cache := NewInMemoryDiscoveryCache()

	err := cache.Set("https://example.com", nil, 1*time.Hour)
	if err == nil {
		t.Error("Set() with nil document expected error, got nil")
	}
}

func TestInMemoryDiscoveryCache_Invalidate(t *testing.T) {
	cache := NewInMemoryDiscoveryCache()

	issuer := "https://accounts.google.com"
	doc := &OIDCConfiguration{
		Issuer: issuer,
	}

	// Set document
	if err := cache.Set(issuer, doc, 1*time.Hour); err != nil {
		t.Fatalf("Set() unexpected error = %v", err)
	}

	// Invalidate
	err := cache.Invalidate(issuer)
	if err != nil {
		t.Fatalf("Invalidate() unexpected error = %v", err)
	}

	// Verify cache is empty (Get will try to fetch, which will fail without mock server)
	cache.mu.RLock()
	_, exists := cache.cache[issuer]
	cache.mu.RUnlock()

	if exists {
		t.Error("Invalidate() did not remove entry from cache")
	}
}

func TestInMemoryDiscoveryCache_Clear(t *testing.T) {
	cache := NewInMemoryDiscoveryCache()

	// Add multiple entries
	issuers := []string{
		"https://accounts.google.com",
		"https://dev.okta.com",
		"https://login.microsoftonline.com/tenant/v2.0",
	}

	for _, issuer := range issuers {
		doc := &OIDCConfiguration{Issuer: issuer}
		if err := cache.Set(issuer, doc, 1*time.Hour); err != nil {
			t.Fatalf("Set() unexpected error = %v", err)
		}
	}

	// Clear cache
	err := cache.Clear()
	if err != nil {
		t.Fatalf("Clear() unexpected error = %v", err)
	}

	// Verify cache is empty
	cache.mu.RLock()
	count := len(cache.cache)
	cache.mu.RUnlock()

	if count != 0 {
		t.Errorf("Clear() cache not empty, got %d entries", count)
	}
}

func TestInMemoryDiscoveryCache_MaxEntries(t *testing.T) {
	cache := NewInMemoryDiscoveryCache(WithMaxEntries(3))

	// Add more entries than max
	for i := 0; i < 5; i++ {
		issuer := "https://example.com/" + string(rune('a'+i))
		doc := &OIDCConfiguration{Issuer: issuer}
		if err := cache.Set(issuer, doc, 1*time.Hour); err != nil {
			t.Fatalf("Set() unexpected error = %v", err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// Verify cache size is at max
	cache.mu.RLock()
	count := len(cache.cache)
	cache.mu.RUnlock()

	if count != 3 {
		t.Errorf("Cache size = %d, want 3 (max entries)", count)
	}
}

func TestInMemoryDiscoveryCache_ExpiredEntry(t *testing.T) {
	cache := NewInMemoryDiscoveryCache()

	issuer := "https://accounts.google.com"
	doc := &OIDCConfiguration{Issuer: issuer}

	// Set document with very short TTL
	if err := cache.Set(issuer, doc, 1*time.Millisecond); err != nil {
		t.Fatalf("Set() unexpected error = %v", err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Verify entry is expired
	cache.mu.RLock()
	cached, exists := cache.cache[issuer]
	cache.mu.RUnlock()

	if !exists {
		t.Fatal("Cache entry does not exist")
	}

	if !cached.IsExpired() {
		t.Error("Cache entry should be expired")
	}
}

func TestInMemoryDiscoveryCache_FetchFromProvider(t *testing.T) {
	// Create mock OIDC provider
	mockDoc := OIDCConfiguration{
		Issuer:                "https://mock.provider.com",
		AuthorizationEndpoint: "https://mock.provider.com/oauth2/authorize",
		TokenEndpoint:         "https://mock.provider.com/oauth2/token",
		JWKSUri:               "https://mock.provider.com/oauth2/jwks",
		ResponseTypesSupported: []string{"code", "token", "id_token"},
		SubjectTypesSupported:  []string{"public"},
		IDTokenSigningAlgValuesSupported: []string{"RS256"},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/.well-known/openid-configuration" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", "\"test-etag\"")
		json.NewEncoder(w).Encode(mockDoc)
	}))
	defer mockServer.Close()

	cache := NewInMemoryDiscoveryCache()
	ctx := context.Background()

	// Fetch discovery document (cache miss)
	got, err := cache.Get(ctx, mockServer.URL)
	if err != nil {
		t.Fatalf("Get() unexpected error = %v", err)
	}

	if got.AuthorizationEndpoint != mockDoc.AuthorizationEndpoint {
		t.Errorf("Get() authorization_endpoint = %v, want %v",
			got.AuthorizationEndpoint, mockDoc.AuthorizationEndpoint)
	}

	// Second fetch should be from cache (no server hit)
	got2, err := cache.Get(ctx, mockServer.URL)
	if err != nil {
		t.Fatalf("Get() unexpected error = %v", err)
	}

	if got2.TokenEndpoint != mockDoc.TokenEndpoint {
		t.Errorf("Get() token_endpoint = %v, want %v",
			got2.TokenEndpoint, mockDoc.TokenEndpoint)
	}
}

func TestInMemoryDiscoveryCache_FetchError(t *testing.T) {
	cache := NewInMemoryDiscoveryCache()
	ctx := context.Background()

	// Try to fetch from non-existent provider
	_, err := cache.Get(ctx, "https://nonexistent.provider.invalid")
	if err == nil {
		t.Error("Get() expected error for non-existent provider, got nil")
	}
}

func TestInMemoryDiscoveryCache_StaleDataOnError(t *testing.T) {
	requestCount := 0

	mockDoc := OIDCConfiguration{
		Issuer:        "https://mock.provider.com",
		TokenEndpoint: "https://mock.provider.com/oauth2/token",
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// First request succeeds
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockDoc)
		} else {
			// Subsequent requests fail
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}))
	defer mockServer.Close()

	cache := NewInMemoryDiscoveryCache()
	ctx := context.Background()

	// First fetch succeeds
	doc1, err := cache.Get(ctx, mockServer.URL)
	if err != nil {
		t.Fatalf("First Get() unexpected error = %v", err)
	}

	// Set entry to expired
	cache.mu.Lock()
	if entry, ok := cache.cache[mockServer.URL]; ok {
		entry.ExpiresAt = time.Now().Add(-1 * time.Hour)
	}
	cache.mu.Unlock()

	// Second fetch fails but returns stale data
	doc2, err := cache.Get(ctx, mockServer.URL)
	if err != nil {
		t.Fatalf("Second Get() unexpected error = %v", err)
	}

	if doc2.Issuer != doc1.Issuer {
		t.Error("Get() should return stale data on fetch error")
	}
}

func TestGetDiscoveryURL(t *testing.T) {
	tests := []struct {
		name      string
		issuerURL string
		want      string
	}{
		{
			name:      "Google",
			issuerURL: "https://accounts.google.com",
			want:      "https://accounts.google.com/.well-known/openid-configuration",
		},
		{
			name:      "trailing slash",
			issuerURL: "https://accounts.google.com/",
			want:      "https://accounts.google.com/.well-known/openid-configuration",
		},
		{
			name:      "Okta",
			issuerURL: "https://dev-12345.okta.com",
			want:      "https://dev-12345.okta.com/.well-known/openid-configuration",
		},
		{
			name:      "Azure AD",
			issuerURL: "https://login.microsoftonline.com/tenant-id/v2.0",
			want:      "https://login.microsoftonline.com/tenant-id/v2.0/.well-known/openid-configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getDiscoveryURL(tt.issuerURL)
			if got != tt.want {
				t.Errorf("getDiscoveryURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInMemoryDiscoveryCache_CustomOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	customTTL := 12 * time.Hour
	customMax := 50

	cache := NewInMemoryDiscoveryCache(
		WithHTTPClient(customClient),
		WithDefaultTTL(customTTL),
		WithMaxEntries(customMax),
	)

	if cache.httpClient != customClient {
		t.Error("Custom HTTP client not set")
	}

	if cache.defaultTTL != customTTL {
		t.Errorf("Default TTL = %v, want %v", cache.defaultTTL, customTTL)
	}

	if cache.maxEntries != customMax {
		t.Errorf("Max entries = %d, want %d", cache.maxEntries, customMax)
	}
}
