package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/audit"
)

func main() {
	// Create a file-based audit logger
	storage, err := audit.NewFileStorage(audit.FileConfig{
		Directory: "audit-logs",
	})
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	logger := audit.NewLogger(audit.Config{
		Storage:      storage,
		MaxRetention: 90 * 24 * time.Hour,
		BatchSize:    100,
		Async:       true,
		BufferSize:   1000,
		ErrorHandler: func(err error) {
			fmt.Printf("Audit error: %v\n", err)
		},
	})
	defer logger.Close()

	ctx := context.Background()

	fmt.Println("Audit Logging Example")
	fmt.Println("====================")

	// 1. Log authentication events
	fmt.Println("\n1. Authentication Events")
	fmt.Println("----------------------")

	// Successful login
	loginEntry := audit.NewEntry(audit.TypeAuth).
		WithActor("user123", audit.ActorUser).
		WithAction(audit.ActionLogin).
		WithTarget("webapp", "application").
		WithResult(audit.ResultSuccess).
		WithMetadata("ip_address", "192.168.1.100").
		WithMetadata("user_agent", "Mozilla/5.0")

	if err := logger.Log(ctx, loginEntry); err != nil {
		log.Printf("Failed to log login: %v", err)
	}
	fmt.Printf("Logged successful login for user123\n")

	// Failed login attempt
	failedLogin := audit.NewEntry(audit.TypeAuth).
		WithActor("unknown", audit.ActorUser).
		WithAction(audit.ActionLogin).
		WithTarget("webapp", "application").
		WithResult(audit.ResultFailure).
		WithError(fmt.Errorf("invalid credentials")).
		WithMetadata("ip_address", "10.0.0.50").
		WithMetadata("attempt", "3")

	if err := logger.Log(ctx, failedLogin); err != nil {
		log.Printf("Failed to log failed login: %v", err)
	}
	fmt.Printf("Logged failed login attempt\n")

	// 2. Log token management
	fmt.Println("\n2. Token Management")
	fmt.Println("-----------------")

	// Token creation
	tokenEntry := audit.NewEntry(audit.TypeToken).
		WithActor("user123", audit.ActorUser).
		WithAction(audit.ActionTokenCreate).
		WithTarget("token456", "access_token").
		WithResult(audit.ResultSuccess).
		WithMetadata("expires_in", "3600").
		WithMetadata("scope", "read write")

	if err := logger.Log(ctx, tokenEntry); err != nil {
		log.Printf("Failed to log token creation: %v", err)
	}
	fmt.Printf("Logged token creation\n")

	// Token revocation
	revokeEntry := audit.NewEntry(audit.TypeToken).
		WithActor("admin", audit.ActorUser).
		WithAction(audit.ActionTokenRevoke).
		WithTarget("token456", "access_token").
		WithResult(audit.ResultSuccess).
		WithMetadata("reason", "user logout")

	if err := logger.Log(ctx, revokeEntry); err != nil {
		log.Printf("Failed to log token revocation: %v", err)
	}
	fmt.Printf("Logged token revocation\n")

	// 3. Search audit logs
	fmt.Println("\n3. Search Audit Logs")
	fmt.Println("------------------")

	// Search by type
	authEntries, err := logger.Search(ctx, &audit.Filter{
		Types: []audit.Type{audit.TypeAuth},
		Limit: 10,
	})
	if err != nil {
		log.Printf("Failed to search auth entries: %v", err)
	}
	fmt.Printf("Found %d auth entries\n", len(authEntries))

	// Search by actor
	userEntries, err := logger.Search(ctx, &audit.Filter{
		ActorIDs: []string{"user123"},
		TimeRange: &audit.TimeRange{
			Start: time.Now().Add(-time.Hour),
			End:   time.Now(),
		},
	})
	if err != nil {
		log.Printf("Failed to search user entries: %v", err)
	}
	fmt.Printf("Found %d entries for user123\n", len(userEntries))

	// 4. Chain related events
	fmt.Println("\n4. Event Chains")
	fmt.Println("--------------")

	chainID := "session-789"

	// Login event in chain
	loginChain := audit.NewEntry(audit.TypeAuth).
		WithActor("user123", audit.ActorUser).
		WithAction(audit.ActionLogin).
		WithResult(audit.ResultSuccess)
	loginChain.ChainID = chainID

	if err := logger.Log(ctx, loginChain); err != nil {
		log.Printf("Failed to log chain login: %v", err)
	}

	// Resource access in same chain
	accessChain := audit.NewEntry(audit.TypeResource).
		WithActor("user123", audit.ActorUser).
		WithAction(audit.ActionResourceAccess).
		WithTarget("document123", "document").
		WithResult(audit.ResultSuccess)
	accessChain.ChainID = chainID

	if err := logger.Log(ctx, accessChain); err != nil {
		log.Printf("Failed to log chain access: %v", err)
	}

	// Get all events in chain
	chainEvents, err := logger.GetChain(ctx, chainID)
	if err != nil {
		log.Printf("Failed to get chain events: %v", err)
	}
	fmt.Printf("Found %d events in chain\n", len(chainEvents))

	// Cleanup old logs
	if err := storage.Cleanup(ctx, time.Now().Add(-30*24*time.Hour)); err != nil {
		log.Printf("Failed to cleanup old logs: %v", err)
	}
}

func init() {
	// Create audit logs directory
	if err := os.MkdirAll("audit-logs", 0750); err != nil {
		log.Fatalf("Failed to create audit logs directory: %v", err)
	}
}