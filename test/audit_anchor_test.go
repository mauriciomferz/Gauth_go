package test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	a "github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/common"
	"github.com/stretchr/testify/require"
)

func TestAuditAnchorCallbackInvoked(t *testing.T) {
	ml := a.NewMemoryLogger(&common.SimpleLogger{})
	var called atomic.Int64
	ml.SetAnchor(func(idx int, hash string) {
		if idx == 0 && hash != "" {
			called.Add(1)
		}
	})
	ev := a.NewEvent(a.EventTypeAuthorization, "create", a.ResultSuccess)
	require.NoError(t, ml.Log(context.Background(), ev))
	// allow goroutine to run
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, int64(1), called.Load())
}
