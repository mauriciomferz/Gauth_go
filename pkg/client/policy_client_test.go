package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
	"github.com/mauriciomferz/AgentAuth/pkg/gauth/external"
)

func TestGetProvenance(t *testing.T) {
	// Mock Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/policy/provenance" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := ProvenanceResponse{
			Success:  true,
			HeadHash: "dummy-hash",
			Chain:    []string{"h1", "h2"},
			Verified: true,
			Length:   2,
			RevocationSnapshot: &delegation.SignedTreeHead{
				Version:       1,
				MerkleRoot:    "root-hash",
				ChainLength:   10,
				AggregateHash: "agg-hash",
				Timestamp:     time.Now(),
			},
		}
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Logf("server encode fail: %v", err)
		}
	}))
	defer ts.Close()

	// Init Client
	cli := NewPolicyClient(&PolicyClientConfig{
		BaseURL:        ts.URL,
		Timeout:        1 * time.Second,
		CircuitBreaker: external.NewCircuitBreaker(5, 1*time.Second, 1*time.Second),
	})

	// Test Call
	ctx := context.Background()
	res, err := cli.GetProvenance(ctx, "")
	if err != nil {
		t.Fatalf("GetProvenance failed: %v", err)
	}

	// Verify Data
	if !res.Success {
		t.Error("Expected success=true")
	}
	if res.HeadHash != "dummy-hash" {
		t.Errorf("Expected head=dummy-hash, got %s", res.HeadHash)
	}
	if res.RevocationSnapshot == nil {
		t.Error("Expected RevocationSnapshot to be populated")
	} else {
		if res.RevocationSnapshot.ChainLength != 10 {
			t.Errorf("Expected ChainLength=10, got %d", res.RevocationSnapshot.ChainLength)
		}
	}
}
