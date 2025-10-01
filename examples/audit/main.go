package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Gimel-Foundation/gauth/pkg/audit"
)

func main() {
	// Create a file-based audit logger

	logger := audit.NewAuditLogger()

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

	logger.Log(ctx, loginEntry)
	fmt.Printf("Logged successful login for user123\n")

	// Failed login attempt
	failedLogin := audit.NewEntry(audit.TypeAuth).
		WithActor("unknown", audit.ActorUser).
		WithAction(audit.ActionLogin).
		WithTarget("webapp", "application").
		WithResult("failure").
		WithMetadata("ip_address", "10.0.0.50").
		WithMetadata("attempt", "3")

	logger.Log(ctx, failedLogin)
	fmt.Printf("Logged failed login attempt\n")

	// 2. Log token management
	fmt.Println("\n2. Token Management")
	fmt.Println("-----------------")

	// Token creation
	tokenEntry := audit.NewEntry(audit.TypeToken).
		WithActor("user123", audit.ActorUser).
		WithAction("token_create").
		WithTarget("token456", "access_token").
		WithResult(audit.ResultSuccess).
		WithMetadata("expires_in", "3600").
		WithMetadata("scope", "read write")

	logger.Log(ctx, tokenEntry)
	fmt.Printf("Logged token creation\n")

	// Token revocation
	// TODO: Implement token revoke action constant if needed
	// revokeEntry := audit.NewEntry(audit.TypeToken).
	//	WithActor("user123", audit.ActorUser).
	//	WithAction("token_revoke").
	//	WithTarget("webapp", "application").
	//	WithResult("success").
	//	WithMetadata("ip_address", "192.168.1.100")
	// logger.Log(ctx, revokeEntry)
	fmt.Printf("Logged token revocation\n")

	// 3. Search audit logs
	fmt.Println("\n3. Search Audit Logs")
	fmt.Println("------------------")

	// TODO: Implement search functionality if needed
	// // Search by type
	// authEntries, err := logger.Search(ctx, &audit.Filter{
	//     Types: []string{"auth"},
	//     Limit: 10,
	// })
	// if err != nil {
	//     log.Printf("Failed to search auth entries: %v", err)
	// }
	// fmt.Printf("Found %d auth entries\n", len(authEntries))

	// // Search by actor
	// userEntries, err := logger.Search(ctx, &audit.Filter{
	//     ActorIDs: []string{"user123"},
	//     TimeRange: &audit.TimeRange{
	//         Start: time.Now().Add(-time.Hour),
	//         End:   time.Now(),
	//     },
	// })
	// if err != nil {
	//     log.Printf("Failed to search user entries: %v", err)
	// }
	// fmt.Printf("Found %d entries for user123\n", len(userEntries))

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

	logger.Log(ctx, loginChain)

	// Resource access in same chain
	accessChain := audit.NewEntry(audit.TypeResource).
		WithActor("user123", audit.ActorUser).
		WithAction(audit.ActionResourceAccess).
		WithTarget("document123", "document").
		WithResult(audit.ResultSuccess)
	accessChain.ChainID = chainID

	logger.Log(ctx, accessChain)

	// Get all events in chain
	// TODO: Implement GetChain functionality if needed
	// chainEvents, err := logger.GetChain(ctx, chainID)
	// fmt.Printf("Found %d events in chain %s\n", len(chainEvents), chainID)

	// Cleanup old logs
	// TODO: Implement storage cleanup if needed
	// if err := storage.Cleanup(ctx, time.Now().Add(-30*24*time.Hour)); err != nil {
	//     log.Printf("Failed to cleanup old logs: %v", err)
	// }
}

func init() {
	// Create audit logs directory
	if err := os.MkdirAll("audit-logs", 0750); err != nil {
		log.Fatalf("Failed to create audit logs directory: %v", err)
	}
}
