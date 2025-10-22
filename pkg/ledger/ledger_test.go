package ledger

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestAppendAndVerifyChain(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		e := &Entry{ID: fmtID(i), TS: time.Now().UTC(), Type: "delegation.create", Subject: "alice", Object: "poa-1"}
		if err := store.Append(ctx, e); err != nil {
			t.Fatalf("append failed: %v", err)
		}
	}
	res, err := store.VerifyChain(ctx)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if res.Count != 3 {
		t.Fatalf("expected 3 entries, got %d", res.Count)
	}
	if res.Mismatches != 0 {
		t.Fatalf("expected 0 mismatches, got %d", res.Mismatches)
	}
}

func TestQuerySubjectAndObject(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	subjCounts := map[string]int{"alice": 2, "bob": 1}
	seq := 0
	for subj, count := range subjCounts {
		for i := 0; i < count; i++ {
			e := &Entry{ID: fmtID(seq), TS: time.Now().UTC(), Type: "delegation.create", Subject: subj, Object: "poa-x"}
			seq++
			if err := store.Append(ctx, e); err != nil {
				t.Fatalf("append failed: %v", err)
			}
		}
	}
	aliceEntries, _ := store.QueryBySubject(ctx, "alice")
	bobEntries, _ := store.QueryBySubject(ctx, "bob")
	if len(aliceEntries) != 2 || len(bobEntries) != 1 {
		t.Fatalf("unexpected subject counts: alice=%d bob=%d", len(aliceEntries), len(bobEntries))
	}
	objEntries, _ := store.QueryByObject(ctx, "poa-x")
	if len(objEntries) != 3 {
		t.Fatalf("expected 3 object matches got %d", len(objEntries))
	}
}

func fmtID(i int) string { return fmt.Sprintf("evt-%d", i) }
