package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	token "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/token"
)

// TokenMonitor provides token lifecycle monitoring
type TokenMonitor struct {
	store     token.Store
	querier   token.Querier
	blacklist token.Blacklist // interface, not *token.Blacklist
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

func NewTokenMonitor(store token.Store, blacklist token.Blacklist) *TokenMonitor {
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

	// Use a dummy []byte for signing key (not RSA key)
	// config := &token.Config{ ... } // unused
	// tokenService := token.NewService(store, blacklist, config)

	t := &token.Token{
		ID:        token.GenerateID(),
		Type:      token.Access,
		Subject:   "user-monitor",
		Issuer:    "monitoring-service",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    []string{"read"},
	}

	// Skipped: Issue method does not exist on tokenService, so just use t directly
	monitor.TrackToken(ctx, t)
	monitor.PrintStats(ctx)
}
