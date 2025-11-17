package web

import (
	"encoding/json"
	"testing"
)

// TestCryptoAlgorithmsEndpoint verifies the introspection endpoint lists expected algorithms.
func TestCryptoAlgorithmsEndpoint(t *testing.T) {
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := PerformRequest(srv, "GET", "/api/v1/crypto/algorithms")
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var resp struct {
		Success    bool `json:"success"`
		Algorithms []struct {
			Name       string `json:"name"`
			Aggregated bool   `json:"aggregated_supported"`
		} `json:"algorithms"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success false")
	}
	// Minimal presence checks
	want := map[string]bool{"ed25519": false, "ecdsa-p256": false, "bls12-381": false, "bls12-381-agg": true}
	for _, a := range resp.Algorithms {
		delete(want, a.Name)
	}
	if len(want) != 0 {
		t.Fatalf("missing algorithms: %v", want)
	}
}
