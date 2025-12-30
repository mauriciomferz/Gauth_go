package web

import (
	"os"
	"strconv"
	"testing"
	"time"

	replaypkg "github.com/mauriciomferz/AgentAuth/pkg/replay"
	"github.com/mauriciomferz/AgentAuth/web/handlers/token"
)

// BenchmarkAttestationReplay compares memory vs redis latency (best-effort, skips if redis unavailable).
func BenchmarkReplayNonceStore(b *testing.B) {
	s := token.NewReplayNonceStore(5 * time.Minute)
	b.Run("memory_record_seen", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			k := "m-" + strconv.Itoa(i)
			s.Record(k, time.Now())
			_ = s.Seen(k, time.Now())
		}
	})
	addr := os.Getenv("GAUTH_ATTEST_REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}
	backend, err := replaypkg.NewRedisReplayBackend(addr, 10*time.Minute)
	if err != nil {
		b.Skipf("redis unavailable: %v", err)
		return
	}
	defer backend.Close()
	b.Run("redis_record_seen", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			k := "r-" + strconv.Itoa(i)
			_ = backend.Record(k)
			_, _ = backend.Seen(k)
		}
	})
}
