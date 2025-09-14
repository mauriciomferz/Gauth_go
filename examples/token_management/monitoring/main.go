package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log"
	"sync"
	"time"

	token "github.com/Gimel-Foundation/gauth/pkg/token"
)

// TokenMonitor provides token lifecycle monitoring
type TokenMonitor struct {
	store     token.Store
	querier   token.TokenQuerier
	blacklist *token.Blacklist
	metrics   *TokenMetrics
}

type TokenMetrics struct {
	mu              sync.RWMutex
	activeTokens    map[string]int
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
	store := token.NewMemoryStore()
	blacklist := token.NewBlacklist()
	monitor := NewTokenMonitor(store, blacklist)

	// Generate an in-memory RSA key for demo purposes
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate RSA key: %v", err)
	}

	config := token.Config{
		SigningMethod:    token.RS256,
		SigningKey:       privateKey,
		ValidityPeriod:   time.Hour,
		RefreshPeriod:    24 * time.Hour,
		DefaultScopes:    []string{"read"},
		ValidateAudience: false,
		ValidateIssuer:   false,
	}

	tokenService := token.NewService(config, store)

	t := &token.Token{
		ID:        token.GenerateID(),
		Type:      token.Access,
		Subject:   "user-monitor",
		Issuer:    "monitoring-service",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    []string{"read"},
	}

	issuedToken, err := tokenService.Issue(ctx, t)
	if err != nil {
		log.Fatalf("Failed to issue token: %v", err)
	}
	fmt.Printf("Issued token: %s\n", issuedToken.Value)

	monitor.TrackToken(ctx, issuedToken)
	monitor.PrintStats(ctx)
}
