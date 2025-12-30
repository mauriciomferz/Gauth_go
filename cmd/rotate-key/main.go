// Package main contains a utility to rotate the Ed25519 signing key when the server
// runs in eddsa signature mode. It initializes a transient manager if the global
// registry is unset and prints rotation details for operational auditing.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// rotate-key is a minimal utility to rotate the Ed25519 signing key when running in eddsa mode.
// Usage:
//
//	AGENTAUTH_TOKEN_SIG_MODE=eddsa go run ./cmd/rotate-key
func main() {
	if os.Getenv("AGENTAUTH_TOKEN_SIG_MODE") != "eddsa" {
		fmt.Println("AGENTAUTH_TOKEN_SIG_MODE != eddsa (no rotation performed)")
		return
	}
	// In a standalone tool, the global registry is always nil initially.
	// We must initialize a manager backed by the persistence path to affect shared state.
	fmt.Println("Initializing manager for rotation")
	km, err := crypto.NewManager(24 * time.Hour)
	if err != nil {
		fmt.Println("manager init error:", err)
		os.Exit(1)
	}
	old := km.Active().ID
	k, err := km.Rotate()
	if err != nil {
		fmt.Println("rotate error:", err)
		os.Exit(1)
	}
	fmt.Printf("Rotated key: old=%s new=%s expires=%s\n", old, k.ID, k.ExpiresAt.Format(time.RFC3339))
}
