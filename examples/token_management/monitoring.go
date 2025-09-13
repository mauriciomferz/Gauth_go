package tokenmanagement

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// TokenMonitor provides token lifecycle monitoring
type TokenMonitor struct {
	store     token.Store
	querier   token.TokenQuerier
	blacklist *token.Blacklist
	metrics   *TokenMetrics
}

// TokenMetrics tracks token statistics
type TokenMetrics struct {
	mu              sync.RWMutex
	activeTokens    map[string]int // by subject
	tokensByType    map[token.Type]int
	expirationTimes map[string]time.Time
	revocationCount int
	creationHistory []time.Time
}

func NewTokenMonitor(store token.Store, blacklist *token.Blacklist) *TokenMonitor {
	return &TokenMonitor{
		store:     store,
		querier:   token.NewDefaultQuerier(store),
		blacklist: blacklist,
		metrics: &TokenMetrics{
			activeTokens:    make(map[string]int),
			tokensByType:    make(map[token.Type]int),
			expirationTimes: make(map[string]time.Time),
			creationHistory: make([]time.Time, 0),
		},
	}
}

func (m *TokenMonitor) TrackToken(ctx context.Context, t *token.Token) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.activeTokens[t.Subject]++
	m.metrics.tokensByType[t.Type]++
	m.metrics.expirationTimes[t.ID] = t.ExpiresAt
	m.metrics.creationHistory = append(m.metrics.creationHistory, t.IssuedAt)
}

func (m *TokenMonitor) TrackRevocation(ctx context.Context, tokenID string) {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.revocationCount++
	delete(m.metrics.expirationTimes, tokenID)
}

func (m *TokenMonitor) PrintStats(ctx context.Context) {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	fmt.Println("\nToken Statistics:")
	fmt.Println("================")

	fmt.Println("\nActive Tokens by Subject:")
	for subject, count := range m.metrics.activeTokens {
		fmt.Printf("- Subject %s: %d tokens\n", subject, count)
	}

	fmt.Println("\nTokens by Type:")
	for typ, count := range m.metrics.tokensByType {
		fmt.Printf("- %s: %d tokens\n", typ, count)
	}

	fmt.Printf("\nTotal Revocations: %d\n", m.metrics.revocationCount)

	// Calculate tokens expiring soon
	now := time.Now()
	expiringSoon := 0
	for _, expiry := range m.metrics.expirationTimes {
		if expiry.Sub(now) < time.Hour {
			expiringSoon++
		}
	}
	fmt.Printf("\nTokens Expiring Within 1 Hour: %d\n", expiringSoon)
}

func main() {
	ctx := context.Background()
	store := token.NewMemoryStore(24 * time.Hour)
	blacklist := token.NewBlacklist()
	monitor := NewTokenMonitor(store, blacklist)

	// Create JWT manager for token signing
	jwtManager := token.NewJWTManager(token.JWTConfig{
		SigningMethod: token.HS256,
		SigningKey:    []byte("monitor-test-key"),
		MaxAge:        time.Hour,
	})

	// Generate test tokens
	subjects := []string{"user1", "user2", "user3"}
	types := []token.Type{token.Access, token.Refresh}

	fmt.Println("Creating test tokens...")
	for _, subject := range subjects {
		for _, typ := range types {
			// Create token with varying expiration
			t := &token.Token{
				ID:        token.NewID(),
				Type:      typ,
				Subject:   subject,
				Issuer:    "token-monitor",
				IssuedAt:  time.Now(),
				ExpiresAt: time.Now().Add(time.Duration(30+len(subject)) * time.Minute),
				Scopes:    []string{"read", "write"},
			}

			// Store and track token
			if err := store.Save(ctx, t); err != nil {
				log.Printf("Failed to save token: %v", err)
				continue
			}

			// Sign token
			signed, err := jwtManager.SignToken(ctx, t)
			if err != nil {
				log.Printf("Failed to sign token: %v", err)
				continue
			}

			monitor.TrackToken(ctx, t)
			fmt.Printf("Created %s token for %s: %s\n", t.Type, t.Subject, signed[:30]+"...")
		}
	}

	// Simulate some revocations
	fmt.Println("\nSimulating token revocations...")
	tokensToRevoke := 2
	revoked := 0

	tokens, err := monitor.querier.ListExpiringSoon(ctx, time.Hour)
	if err != nil {
		log.Printf("Failed to list expiring tokens: %v", err)
	} else {
		for _, t := range tokens {
			if revoked >= tokensToRevoke {
				break
			}
			if err := blacklist.Add(ctx, t, "monitoring test"); err != nil {
				log.Printf("Failed to revoke token: %v", err)
				continue
			}
			monitor.TrackRevocation(ctx, t.ID)
			revoked++
			fmt.Printf("Revoked token: %s\n", t.ID)
		}
	}

	// Print statistics
	monitor.PrintStats(ctx)
}
