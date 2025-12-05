package web

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/mauriciomferz/Gauth_go/internal/capability"
	metrics "github.com/mauriciomferz/Gauth_go/internal/metrics"
	cryptopkg "github.com/mauriciomferz/Gauth_go/pkg/crypto"
)

// prepareEdDSAKey ensures an active EdDSA key exists for signing tests.
// prepareEdDSAKey ensures an active EdDSA key exists for signing tests.
func prepareEdDSAKey(t *testing.T) *cryptopkg.Manager {
	t.Helper()
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "eddsa")
	km, err := cryptopkg.NewManager(24 * time.Hour)
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	return km
}

func TestPolicyManifestDeterministic(t *testing.T) {
	km := prepareEdDSAKey(t)
	// Seed capability registry
	capability.Reset([]capability.Capability{{ID: "cap.transfer", Version: "v1", Stable: true}, {ID: "cap.issue", Version: "v1", Stable: true}})
	srv := NewBetaServer(":0", WithKeyProvider(km))
	t.Cleanup(func() { srv.Shutdown() })
	// First request
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/policy/manifest", nil)
	srv.router.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status %d body=%s", w.Code, w.Body.String())
	}
	var body1 map[string]any
	if json.Unmarshal(w.Body.Bytes(), &body1) != nil {
		t.Fatalf("json1 decode fail")
	}
	hash1, ok := body1["manifest_hash"].(string)
	if !ok || hash1 == "" {
		t.Fatalf("missing manifest_hash")
	}
	sig1, ok := body1["signature"].(string)
	if !ok || sig1 == "" {
		t.Fatalf("missing signature")
	}
	etag := w.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("missing etag header")
	}
	// Second request should 304 with If-None-Match
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/api/v1/policy/manifest", nil)
	r2.Header.Set("If-None-Match", etag)
	srv.router.ServeHTTP(w2, r2)
	if w2.Code != 304 {
		t.Fatalf("expected 304 got %d", w2.Code)
	}
	// Third request (no header) should produce same hash & different generated_at but identical signature over core
	w3 := httptest.NewRecorder()
	r3 := httptest.NewRequest("GET", "/api/v1/policy/manifest", nil)
	srv.router.ServeHTTP(w3, r3)
	var body3 map[string]any
	if json.Unmarshal(w3.Body.Bytes(), &body3) != nil {
		t.Fatalf("json3 decode fail")
	}
	if body3["manifest_hash"] != hash1 {
		t.Fatalf("hash mismatch %s vs %s", body3["manifest_hash"], hash1)
	}
	if body3["signature"] != sig1 {
		t.Fatalf("signature changed unexpectedly")
	}
}

func TestPolicyManifestSignatureVerifyAndTamper(t *testing.T) {
	km := prepareEdDSAKey(t)
	capability.Reset([]capability.Capability{{ID: "cap.transfer", Version: "v1", Stable: true}})
	srv := NewBetaServer(":0", WithKeyProvider(km))
	t.Cleanup(func() { srv.Shutdown() })
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/policy/manifest", nil)
	srv.router.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status %d", w.Code)
	}
	var payload map[string]any
	if json.Unmarshal(w.Body.Bytes(), &payload) != nil {
		t.Fatalf("json decode fail")
	}
	sigB64 := payload["signature"].(string)
	kid := payload["sig_kid"].(string)
	hashStr := payload["manifest_hash"].(string)
	// Reconstruct canonical struct (must match buildPolicyManifest ordering & types)
	// Convert capabilities
	capsRaw := payload["capabilities"].([]any)
	mcaps := make([]manifestCap, 0, len(capsRaw))
	for _, c := range capsRaw {
		cm := c.(map[string]any)
		capEntry := manifestCap{ID: cm["id"].(string), Version: cm["version"].(string), Stable: cm["stable"].(bool)}
		if v, ok := cm["deprecated_after"].(string); ok && v != "" {
			capEntry.DeprecatedAfter = v
		}
		if v, ok := cm["sunset_after"].(string); ok && v != "" {
			capEntry.SunsetAfter = v
		}
		if v, ok := cm["versions"].([]any); ok && len(v) > 0 {
			vers := make([]string, 0, len(v))
			for _, vv := range v {
				vers = append(vers, vv.(string))
			}
			capEntry.Versions = vers
		}
		mcaps = append(mcaps, capEntry)
	}
	// Convert action matrix
	actsRaw := payload["action_matrix"].([]any)
	mactions := make([]manifestAction, 0, len(actsRaw))
	for _, a := range actsRaw {
		am := a.(map[string]any)
		reqArr := []string{}
		if rr, ok := am["required"].([]any); ok {
			for _, r := range rr {
				reqArr = append(reqArr, r.(string))
			}
		}
		mactions = append(mactions, manifestAction{Action: am["action"].(string), Required: reqArr})
	}
	canon := manifestCanonical{
		SchemaVersion:   int(payload["schema_version"].(float64)),
		Capabilities:    mcaps,
		ActionMatrix:    mactions,
		RegistryHash:    payload["registry_hash"].(string),
		CapabilityCount: int(payload["capability_count"].(float64)),
		ActionCount:     int(payload["action_count"].(float64)),
	}
	if v, ok := payload["registry_prev_hash"].(string); ok && v != "" {
		canon.RegistryPrevHash = v
	}
	if v, ok := payload["registry_last_changed_at"].(string); ok && v != "" {
		canon.RegistryLastChangedAt = v
	}
	raw, _ := json.Marshal(canon)
	sum := sha256.Sum256(raw)
	recomputedHash := fmt.Sprintf("sha256:%x", sum[:])
	if recomputedHash != hashStr {
		t.Fatalf("manifest_hash mismatch recomputed=%s stored=%s", recomputedHash, hashStr)
	}
	sigBytes, _ := base64.RawURLEncoding.DecodeString(sigB64)
	key := km.FindByID(kid)
	if key == nil {
		t.Fatalf("key not found for kid=%s", kid)
	}
	msg := append([]byte("GAUTH_POLICY_MANIFEST:"), raw...)
	if !ed25519.Verify(key.Public, msg, sigBytes) {
		t.Fatalf("signature verify failed")
	}
	// Tamper capability version and expect verification failure
	if len(mcaps) == 0 {
		t.Fatalf("no capabilities to tamper")
	}
	mcaps[0].Version = "v2" // tamper
	canonTamper := canon
	canonTamper.Capabilities = mcaps
	rawTamper, _ := json.Marshal(canonTamper)
	msgTamper := append([]byte("GAUTH_POLICY_MANIFEST:"), rawTamper...)
	if ed25519.Verify(key.Public, msgTamper, sigBytes) {
		t.Fatalf("tampered signature unexpectedly valid")
	}
}

// TestPolicyManifestSigningUnavailable ensures endpoint returns structured error when EdDSA signing prerequisites are not met.
func TestPolicyManifestSigningUnavailable(t *testing.T) {
	// Force non-EdDSA mode
	orig := os.Getenv("GAUTH_TOKEN_SIG_MODE")
	os.Setenv("GAUTH_TOKEN_SIG_MODE", "hs256")
	defer os.Setenv("GAUTH_TOKEN_SIG_MODE", orig)
	// No key provider passed
	srv := NewBetaServer(":0")
	t.Cleanup(func() { srv.Shutdown() })
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/v1/policy/manifest", nil)
	srv.router.ServeHTTP(w, r)
	if w.Code != 500 {
		t.Fatalf("expected 500 got %d body=%s", w.Code, w.Body.String())
	}
	var errBody map[string]any
	if json.Unmarshal(w.Body.Bytes(), &errBody) != nil {
		t.Fatalf("error body json decode failed")
	}
	if v := errBody["code"]; v != "signing_unavailable" {
		t.Fatalf("unexpected code %v", v)
	}
	if v := errBody["error"]; v != "signing_unavailable" {
		t.Fatalf("unexpected error legacy field %v", v)
	}
	if ok, _ := errBody["success"].(bool); ok {
		t.Fatalf("success field should be false")
	}
}

// TestPolicyManifestETagMismatch ensures a different If-None-Match value does not suppress body.
func TestPolicyManifestETagMismatch(t *testing.T) {
	km := prepareEdDSAKey(t)
	capability.Reset([]capability.Capability{{ID: "cap.transfer", Version: "v1", Stable: true}})
	srv := NewBetaServer(":0", WithKeyProvider(km))
	t.Cleanup(func() { srv.Shutdown() })
	// First request to get real ETag
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("GET", "/api/v1/policy/manifest", nil)
	srv.router.ServeHTTP(w1, r1)
	if w1.Code != 200 {
		t.Fatalf("status1 %d", w1.Code)
	}
	realETag := w1.Header().Get("ETag")
	if realETag == "" {
		t.Fatalf("missing real etag")
	}
	// Second request with mismatching If-None-Match
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("GET", "/api/v1/policy/manifest", nil)
	r2.Header.Set("If-None-Match", "W/\"deadbeef\"")
	srv.router.ServeHTTP(w2, r2)
	if w2.Code != 200 {
		t.Fatalf("expected 200 for mismatching etag got %d", w2.Code)
	}
	if w2.Header().Get("ETag") != realETag {
		t.Fatalf("etag changed unexpectedly mismatch scenario")
	}
	if len(w2.Body.Bytes()) == 0 {
		t.Fatalf("expected body on mismatching etag")
	}

	// Metrics counter should increment on each successful 200 emission
	mem, ok := srv.metrics.(*metrics.Memory)
	if !ok {
		t.Fatalf("metrics type assertion failed")
	}
	if mem.PolicyManifestEmitted() < 2 {
		t.Fatalf("expected at least 2 emissions counter=%d", mem.PolicyManifestEmitted())
	}
}

// TestPolicyManifestSignerInterface ensures the rotating signer produces identical
// signature bytes to direct ed25519 signing over the domain-separated canonical payload.
func TestPolicyManifestSignerInterface(t *testing.T) {
	km := prepareEdDSAKey(t)
	capability.Reset([]capability.Capability{{ID: "cap.transfer", Version: "v1", Stable: true}})
	srv := NewBetaServer(":0", WithKeyProvider(km))
	t.Cleanup(func() { srv.Shutdown() })
	canon, raw, _, err := srv.buildPolicyManifest()
	if err != nil {
		t.Fatalf("buildPolicyManifest error: %v", err)
	}
	if len(canon.Capabilities) == 0 {
		t.Fatalf("expected capabilities in canonical")
	}
	active := km.Active()
	if active == nil || len(active.Private) != ed25519.PrivateKeySize {
		t.Fatalf("active key missing")
	}
	msg := append([]byte("GAUTH_POLICY_MANIFEST:"), raw...)
	sigDirect := ed25519.Sign(active.Private, msg)
	signer, err := km.ActiveSigner()
	if err != nil {
		t.Fatalf("ActiveSigner error: %v", err)
	}
	sigIface, err := signer.Sign(msg)
	if err != nil {
		t.Fatalf("signer.Sign error: %v", err)
	}
	if base64.RawURLEncoding.EncodeToString(sigIface) != base64.RawURLEncoding.EncodeToString(sigDirect) {
		t.Fatalf("rotating signer signature mismatch with direct signing")
	}
}
