package rfc0111

import (
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

// TestRedisReplayStoreBasic exercises first-seen vs replay using a real Redis if available.
func TestRedisReplayStoreBasic(t *testing.T) {
	client := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	rs, err := NewRedisReplayStore(client, "gauthtest", time.Minute)
	if err != nil {
		t.Skipf("redis not available: %v", err)
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
