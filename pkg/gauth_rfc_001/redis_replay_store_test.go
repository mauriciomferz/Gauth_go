package gauth_rfc_001

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// TestRedisReplayStoreBasic exercises first-seen vs replay using a real Redis if available.
func TestRedisReplayStoreBasic(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	rs, err := NewRedisReplayStore(client, "gauthtest", time.Minute)
	if err != nil {
		t.Fatalf("redis store create error: %v", err)
	}
	jti := "test-jti-" + time.Now().Format("150405.000000")
	seen, err := rs.Seen(jti)
	if err != nil {
		t.Fatalf("seen error: %v", err)
	}
	if seen {
		t.Fatalf("expected not seen first time")
	}
	if err2 := rs.Record(jti, time.Now()); err2 != nil {
		t.Fatalf("record error: %v", err2)
	}
	seen2, err := rs.Seen(jti)
	if err != nil {
		t.Fatalf("seen2 error: %v", err)
	}
	if !seen2 {
		t.Fatalf("expected seen after record")
	}
}
