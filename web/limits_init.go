package web

// limits_init.go wires the limits persistent counters manager into BetaServer startup.
// We keep this isolated to avoid bloating server_clean.go further.

import (
	"fmt"
	"time"

	"github.com/mauriciomferz/AgentAuth/internal/limits"
)

// initLimitsManager initializes the limits manager (idempotent). Logs to stdout on failure.
func initLimitsManager() {
	mgr, err := limits.InitFromEnv()
	if err != nil {
		fmt.Printf("[limits] initialization error: %v\n", err)
		return
	}
	// Register simple stdout callback for now. In future we can append to audit ledger if server provides accessor.
	if mgr != nil {
		mgr.SetSnapshotCallback(func(entry map[string]any) {
			// structured minimal log
			fmt.Printf("[limits] snapshot persisted at %s (counters=%d)\n", time.Now().UTC().Format(time.RFC3339), len(entry)-2)
		})
	}
}

// init auto-invokes limits manager initialization on package load.
func init() { initLimitsManager() }
