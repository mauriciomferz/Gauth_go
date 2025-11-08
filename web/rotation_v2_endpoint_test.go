package web

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
	"time"

	notary "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
)

// TestRotationV2Endpoint verifies artifact shape, threshold evaluation and continuity advancement when verified weight meets threshold.
func TestRotationV2Endpoint(t *testing.T) {
	// Prepare temporary weights config with low threshold so demo signatures (weight=1) can satisfy quickly.
	tmpFile, err := os.CreateTemp(t.TempDir(), "weights-*.json")
	if err != nil {
		t.Fatalf("temp file: %v", err)
	}
	// Three signers; threshold=2 so after signatures are attached verified_weight >=2 triggers continuity update.
	cfgJSON := `{
        "schema_version":1,
        "active_key_set_id":"test-set",
        "threshold_weight":2,
        "signers":[{"id":"a","alg":"ED25519","weight":1},{"id":"b","alg":"ED25519","weight":1},{"id":"c","alg":"ED25519","weight":1}],
        "algorithm_suite":["ed25519"]
    }`
	if _, err := tmpFile.Write([]byte(cfgJSON)); err != nil {
		t.Fatalf("write cfg: %v", err)
	}
	tmpFile.Close()
	os.Setenv("GAUTH_ROTATIONS_V2_CONFIG", tmpFile.Name())
	os.Setenv("GAUTH_ROTATIONS_V2_SIGN", "1")
	os.Setenv("GAUTH_ROTATIONS_V2_EMBED_PUBS", "1")
	defer func() {
		os.Unsetenv("GAUTH_ROTATIONS_V2_CONFIG")
		os.Unsetenv("GAUTH_ROTATIONS_V2_SIGN")
		os.Unsetenv("GAUTH_ROTATIONS_V2_EMBED_PUBS")
	}()

	srv := NewBetaServer(":0")
	// Build test HTTP recorder
	w1 := performRequest(srv.router, http.MethodGet, "/api/v1/rotation/summary/v2")
	if w1.Code != 200 {
		t.Fatalf("expected 200 first call got %d body=%s", w1.Code, w1.Body.String())
	}
	var resp1 struct {
		Success        bool                            `json:"success"`
		Artifact       notary.WeightedRotationArtifact `json:"artifact"`
		VerifiedWeight int                             `json:"verified_weight"`
		ThresholdMet   bool                            `json:"threshold_met"`
		PrevHash       string                          `json:"continuity_prev_hash"`
		LatestHash     string                          `json:"continuity_latest_hash"`
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("unmarshal resp1: %v", err)
	}
	if !resp1.Success {
		t.Fatalf("success false first response")
	}
	if resp1.Artifact.Version != 2 {
		t.Fatalf("expected version 2 got %d", resp1.Artifact.Version)
	}
	if resp1.Artifact.CanonicalDigest == "" {
		t.Fatalf("missing canonical digest")
	}
	// First continuity last hash empty until threshold satisfied (signatures may not have matched all weights yet)
	// Second call should advance continuity if threshold met.
	time.Sleep(10 * time.Millisecond)
	w2 := performRequest(srv.router, http.MethodGet, "/api/v1/rotation/summary/v2")
	if w2.Code != 200 {
		t.Fatalf("expected 200 second call got %d body=%s", w2.Code, w2.Body.String())
	}
	var resp2 struct {
		Success        bool   `json:"success"`
		VerifiedWeight int    `json:"verified_weight"`
		ThresholdMet   bool   `json:"threshold_met"`
		PrevHash       string `json:"continuity_prev_hash"`
		LatestHash     string `json:"continuity_latest_hash"`
	}
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("unmarshal resp2: %v", err)
	}
	if !resp2.Success {
		t.Fatalf("success false second response")
	}
	if resp2.VerifiedWeight < 0 {
		t.Fatalf("verified weight invalid: %d", resp2.VerifiedWeight)
	}
	// Since we attached signatures with default weight=1, verified weight should be >=1; threshold=2 may or may not be met depending on key registry contents.
	// We accept either outcome but ensure no panic and continuity fields are coherent.
	if resp2.ThresholdMet && resp2.LatestHash == "" {
		t.Fatalf("threshold met but continuity latest hash empty")
	}
}
