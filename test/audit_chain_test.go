package test

import (
	"context"
	"testing"
	"time"

	a "github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/common"
	"github.com/stretchr/testify/require"
)

// TestAuditChainIntegrity ensures events are hash chained and tamper evident.
func TestAuditChainIntegrity(t *testing.T) {
	logger := a.NewMemoryLogger(&common.SimpleLogger{})
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		ev := a.NewEvent(a.EventTypeAuthorization, "op", a.ResultSuccess)
		ev.Subject = "user"
		ev.Object = "resource"
		require.NoError(t, logger.Log(ctx, ev))
	}

	// Allow async audit processing to complete
	time.Sleep(50 * time.Millisecond)

	// Verify chain passes initially
	require.NoError(t, logger.VerifyChain())

	// Tamper: modify one event's Action field directly
	events, _ := logger.Query(ctx, nil)
	events[2].Action = "tampered"

	// Verification should now fail
	require.Error(t, logger.VerifyChain())
}
