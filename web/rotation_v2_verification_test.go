package web

import (
    "crypto/ed25519"
    "crypto/rand"
    "encoding/hex"
    "encoding/json"
    "os"
    "testing"
    "net/http/httptest"
    "github.com/gin-gonic/gin"
    notary "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
)

// helper to install keys into global registry (simplified for tests)
func installTestKey(t *testing.T, id string) (ed25519.PrivateKey, ed25519.PublicKey) {
    pub, priv, _ := ed25519.GenerateKey(rand.Reader)
    // Populate fallback env variable used by server handler for signing (tests avoid full key manager mutation)
    hexPriv := hex.EncodeToString(priv)
    existing := os.Getenv("GAUTH_ROTATIONS_V2_ED25519_KEYS")
    entry := id + ":" + hexPriv
    if existing == "" { os.Setenv("GAUTH_ROTATIONS_V2_ED25519_KEYS", entry) } else { os.Setenv("GAUTH_ROTATIONS_V2_ED25519_KEYS", existing+","+entry) }
    return priv, pub
}

func TestRotationV2VerifiedWeightSuccess(t *testing.T) {
    gin.SetMode(gin.TestMode)
    // Prepare config file
    cfgData := `{"schema_version":1,"active_key_set_id":"set","threshold_weight":2,"signers":[{"id":"k1","alg":"ED25519","weight":1},{"id":"k2","alg":"ED25519","weight":1}],"algorithm_suite":["ed25519"]}`
    tmpCfg := writeTempFile(t, cfgData)
    os.Setenv("GAUTH_ROTATIONS_V2_CONFIG", tmpCfg)
    os.Setenv("GAUTH_ROTATIONS_V2_SIGN", "1")
    os.Setenv("GAUTH_ROTATIONS_V2_FORCE_SIGN", "1")
    // Install keys
    installTestKey(t, "k1")
    installTestKey(t, "k2")
    // Use full server initialization so route wiring matches production: /api/v1/beta/rotations/summary/v2
    srv := NewBetaServerWithMetrics("", nil)
    // Provide a rotation ledger with dummy entries to satisfy endpoint preconditions
    srv.rotationLedger = notary.NewRotationLedger("")
    srv.rotationLedger.AppendDescriptor(&notary.KeyRotationDescriptor{OldKeyID: "a", NewKeyID: "b"})
    srv.rotationLedger.AppendDescriptor(&notary.KeyRotationDescriptor{OldKeyID: "b", NewKeyID: "c"})
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/v1/beta/rotations/summary/v2", nil)
    srv.router.ServeHTTP(w, req)
    if w.Code != 200 { t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String()) }
    var body map[string]any
    _ = json.Unmarshal(w.Body.Bytes(), &body)
    if body["success"] != true { t.Fatalf("success false") }
}

func TestRotationV2ThresholdUnsatisfied(t *testing.T) {
    gin.SetMode(gin.TestMode)
    // Config requires total weight 3 (2+1) threshold 3; we will only install key k1 so verified weight=2 < threshold.
    cfgData := `{"schema_version":1,"active_key_set_id":"set","threshold_weight":3,"signers":[{"id":"k1","alg":"ED25519","weight":2},{"id":"k2","alg":"ED25519","weight":1}],"algorithm_suite":["ed25519"]}`
    tmpCfg := writeTempFile(t, cfgData)
    os.Setenv("GAUTH_ROTATIONS_V2_CONFIG", tmpCfg)
    os.Setenv("GAUTH_ROTATIONS_V2_SIGN", "1")
    os.Setenv("GAUTH_ROTATIONS_V2_FORCE_SIGN", "1")
    // Ensure clean slate then install only k1
    os.Unsetenv("GAUTH_ROTATIONS_V2_ED25519_KEYS")
    installTestKey(t, "k1") // single key -> verified weight 2 < threshold 3
    srv := NewBetaServerWithMetrics("", nil)
    srv.rotationLedger = notary.NewRotationLedger("")
    srv.rotationLedger.AppendDescriptor(&notary.KeyRotationDescriptor{OldKeyID: "a", NewKeyID: "b"})
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/v1/beta/rotations/summary/v2", nil)
    srv.router.ServeHTTP(w, req)
    if w.Code != 400 { t.Fatalf("expected 400 got %d", w.Code) }
    if !contains(w.Body.String(), "threshold_unsatisfied") { t.Fatalf("missing threshold_unsatisfied in body: %s", w.Body.String()) }
}

// writeTempFile helper
func writeTempFile(t *testing.T, content string) string {
    f, err := os.CreateTemp(t.TempDir(), "cfg-*.json")
    if err != nil { t.Fatalf("tmp: %v", err) }
    if _, err := f.WriteString(content); err != nil { t.Fatalf("write: %v", err) }
    f.Close()
    return f.Name()
}
