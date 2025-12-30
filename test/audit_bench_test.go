package test

import (
	"context"
	"testing"

	a "github.com/mauriciomferz/AgentAuth/pkg/audit"
	"github.com/mauriciomferz/AgentAuth/pkg/testutil"
)

const benchUser = "user"

// Benchmark comparing direct MemoryLogger vs Ledger projection overhead.
func BenchmarkAuditLoggingVsLedger(b *testing.B) {
	ctx := context.Background()
	b.Run("memory_logger", func(b *testing.B) {
		ml := a.NewMemoryLogger(testutil.NoopLogger{})
		for i := 0; i < b.N; i++ {
			ev := a.NewEvent(a.EventTypeAuthorization, "act", a.ResultSuccess)
			ev.Subject = benchUser
			ev.Object = "res"
			if err := ml.Log(ctx, ev); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("ledger_append", func(b *testing.B) {
		l := a.NewLedgerWithLogger(testutil.NoopLogger{})
		for i := 0; i < b.N; i++ {
			if _, err := l.Append(benchUser, "act", "res", ""); err != nil {
				b.Fatal(err)
			}
		}
	})
}
