package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/gnap"
)

// This example demonstrates how to set up the GNAP CleanupManager
// in a production environment with proper lifecycle management.

func main() {
	log.Println("Starting GNAP cleanup manager example...")

	// Initialize GNAP stores
	grantStore := gnap.NewMemoryGrantStore()
	tokenStore := gnap.NewMemoryTokenStore()

	// Configure cleanup interval based on environment
	cleanupInterval := getCleanupInterval()
	log.Printf("Cleanup interval: %v", cleanupInterval)

	// Create cleanup manager
	cleanup := gnap.NewCleanupManager(grantStore, tokenStore, cleanupInterval)

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleanup manager
	if err := cleanup.Start(ctx); err != nil {
		log.Fatalf("Failed to start cleanup manager: %v", err)
	}
	defer cleanup.Stop()

	log.Println("Cleanup manager started successfully")

	// Simulate some grant and token creation
	simulateActivity(grantStore, tokenStore)

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// Start metrics reporter
	go reportMetrics(cleanup)

	// Wait for shutdown signal
	<-sigCh
	log.Println("Shutdown signal received, stopping...")

	// Cleanup manager will be stopped by defer
	// Print final stats
	stats := cleanup.Stats()
	log.Printf("Final cleanup stats: %d grants, %d tokens cleaned",
		stats.TotalGrantsCleaned, stats.TotalTokensCleaned)

	log.Println("Cleanup manager stopped gracefully")
}

// getCleanupInterval returns the cleanup interval based on environment
func getCleanupInterval() time.Duration {
	env := os.Getenv("ENVIRONMENT")
	switch env {
	case "production":
		return 15 * time.Minute
	case "staging":
		return 10 * time.Minute
	case "development":
		return 5 * time.Minute
	default:
		// High-volume default
		return 5 * time.Minute
	}
}

// simulateActivity creates some test grants and tokens
func simulateActivity(grantStore *gnap.MemoryGrantStore, tokenStore *gnap.MemoryTokenStore) {
	log.Println("Simulating grant and token activity...")

	now := time.Now()

	// Create a valid grant
	grant1, _ := grantStore.Create(&gnap.GrantRequest{
		Client: &gnap.ClientInstance{InstanceID: "client-123"},
	})
	log.Printf("Created grant: %s", grant1.ID)

	// Create an expired grant (will be cleaned up)
	expiredGrant, _ := grantStore.Create(&gnap.GrantRequest{
		Client: &gnap.ClientInstance{InstanceID: "client-456"},
	})
	expiredGrant.ExpiresAt = now.Add(-1 * time.Hour)
	_ = grantStore.Update(expiredGrant)
	log.Printf("Created expired grant: %s (will be cleaned)", expiredGrant.ID)

	// Create valid token
	token1 := &gnap.IssuedToken{
		Value:     "agentauth_gnap_valid",
		GrantID:   grant1.ID,
		IssuedAt:  now,
		ExpiresAt: now.Add(1 * time.Hour),
	}
	_ = tokenStore.Store(token1)
	log.Printf("Created token: %s", token1.Value)

	// Create expired token (will be cleaned up after grace period)
	expiredToken := &gnap.IssuedToken{
		Value:     "agentauth_gnap_expired",
		GrantID:   grant1.ID,
		IssuedAt:  now.Add(-3 * time.Hour),
		ExpiresAt: now.Add(-2 * time.Hour), // Expired 2 hours ago
	}
	_ = tokenStore.Store(expiredToken)
	log.Printf("Created expired token: %s (will be cleaned)", expiredToken.Value)

	log.Println("Activity simulation complete")
}

// reportMetrics periodically reports cleanup statistics
func reportMetrics(cleanup *gnap.CleanupManager) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		stats := cleanup.Stats()
		if !stats.Running {
			return
		}

		log.Printf("[Metrics] Cleanup stats: Grants=%d, Tokens=%d, LastRun=%v, Running=%v",
			stats.TotalGrantsCleaned,
			stats.TotalTokensCleaned,
			stats.LastCleanup.Format(time.RFC3339),
			stats.Running)
	}
}
