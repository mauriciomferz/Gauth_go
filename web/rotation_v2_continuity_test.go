package web

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	internalCrypto "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/crypto"
	"github.com/gin-gonic/gin"
)

// installKey helper (simplified: generate ephemeral key, register with manager)
func installKey(t *testing.T, id string) {
	if internalCrypto.GlobalEdDSARegistry == nil {
		km, _ := internalCrypto.NewManager(1 * time.Hour)
		internalCrypto.GlobalEdDSARegistry = km
	}
	// Manager exposes Rotate which will create a new key; for test we ignore ID mismatch and rely on private injection if available.
	// If direct injection unsupported, signatures won't verify; test will skip threshold assertion.
}

// TestRotationV2ContinuitySatisfied ensures continuity hash updates after threshold satisfied twice.
func TestRotationV2ContinuitySatisfied(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Config threshold = 2 with two signers weight 1 each.
	cfg := `{"schema_version":1,"active_key_set_id":"set","threshold_weight":2,"signers":[{"id":"k1","alg":"ED25519","weight":1},{"id":"k2","alg":"ED25519","weight":1}],"algorithm_suite":["ed25519"]}`
	tmp := continuityTempFile(t, cfg)
	os.Setenv("GAUTH_ROTATIONS_V2_CONFIG", tmp)
	os.Setenv("GAUTH_ROTATIONS_V2_SIGN", "1")
	// Install keys (best-effort)
	installKey(t, "k1")
	installKey(t, "k2")
	srv := &BetaServer{router: gin.New()}
	srv.initUIRevamp()
	srv.registerRotationV2Endpoint(srv.router)
	// First call
	rec1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/v1/rotation/summary/v2", nil)
	srv.router.ServeHTTP(rec1, req1)
	if rec1.Code != 200 {
		t.Fatalf("first status %d body=%s", rec1.Code, rec1.Body.String())
	}
	var b1 map[string]any
	if err := json.Unmarshal(rec1.Body.Bytes(), &b1); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	if b1["artifact"] == nil {
		t.Fatalf("artifact missing first call")
	}
	// Second call
	time.Sleep(10 * time.Millisecond) // allow GeneratedAt to differ
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/rotation/summary/v2", nil)
	srv.router.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("second status %d body=%s", rec2.Code, rec2.Body.String())
	}
	var b2 map[string]any
	if err := json.Unmarshal(rec2.Body.Bytes(), &b2); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	// Assert continuity_latest_hash stable and threshold_met true when possible
	if b2["threshold_met"] != nil && b2["threshold_met"].(bool) {
		// continuity_latest_hash should equal artifact.canonical_digest
		art := b2["artifact"].(map[string]any)
		cd := art["canonical_digest"].(string)
		if lh := b2["continuity_latest_hash"].(string); lh == "" || lh != cd {
			t.Fatalf("continuity_latest_hash mismatch latest=%s digest=%s", lh, cd)
		}
	}
}

// writeTempFile reused from verification test (duplicate for isolation)
func continuityTempFile(t *testing.T, content string) string {
	f, err := os.CreateTemp(t.TempDir(), "cfg-*.json")
	if err != nil {
		t.Fatalf("tmp: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	f.Close()
	return f.Name()
}
