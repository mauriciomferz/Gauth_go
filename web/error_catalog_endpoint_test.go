package web

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	notary "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/internal/notary"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestErrorCatalogEndpoint verifies /api/v1/errors/catalog returns success and non-empty entries.
func TestErrorCatalogEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := &BetaServer{router: gin.New()}
	srv.initUIRevamp() // mounts error catalog
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/errors/catalog", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	body := w.Body.String()
	if len(body) == 0 || !containsStr(body, "entries") {
		t.Fatalf("unexpected body: %s", body)
	}
}

// TestUIIndex ensures /ui serves the dashboard index.
func TestUIIndex(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := &BetaServer{router: gin.New()}
	srv.initUIRevamp()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ui", nil)
	srv.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	if !containsStr(w.Body.String(), "GAuth Beta Dashboard") {
		t.Fatalf("index missing content")
	}
}

// TestAttestationVerifyErrorEnvelope ensures attestation verify returns ErrorEnvelope on malformed JSON.
func TestAttestationVerifyErrorEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := &BetaServer{router: gin.New()}
	srv.initUIRevamp()
	// Minimal route wiring for attestation verify handler (depends on BetaServer fields kept nil for this malformed case).
	srv.router.POST("/api/v1/model/limits/verify", srv.apiModelLimitsAttestationVerify)
	w := httptest.NewRecorder()
	// Malformed JSON triggers invalid_json path
	req := httptest.NewRequest("POST", "/api/v1/model/limits/verify", strings.NewReader("{bad"))
	srv.router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 got %d", w.Code)
	}
	// Parse envelope
	var env struct {
		Code    string         `json:"code"`
		Message string         `json:"message"`
		Details map[string]any `json:"details"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal error: %v body=%s", err, w.Body.String())
	}
	if env.Code != "attestation_invalid_json" {
		t.Fatalf("unexpected code %s", env.Code)
	}
	if env.Message == "" {
		t.Fatalf("missing message")
	}
	if env.Details == nil || env.Details["http_path"] != "/api/v1/model/limits/verify" {
		t.Fatalf("details missing http_path: %#v", env.Details)
	}
}

// TestErrorCatalogCaching validates ETag + 304 conditional fetch behavior.
func TestErrorCatalogCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	srv := &BetaServer{router: gin.New()}
	srv.initUIRevamp()
	// First request to obtain ETag
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("GET", "/api/v1/errors/catalog", nil)
	srv.router.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("expected 200 got %d", w1.Code)
	}
	etag := w1.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("missing ETag header")
	}
	// Second conditional request with If-None-Match
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/api/v1/errors/catalog", nil)
	req2.Header.Set("If-None-Match", etag)
	srv.router.ServeHTTP(w2, req2)
	if w2.Code != 304 {
		t.Fatalf("expected 304 got %d", w2.Code)
	}
	if w2.Body.Len() != 0 {
		t.Fatalf("expected empty body for 304, got %s", w2.Body.String())
	}
}

// TestRotationSummaryV2Deterministic verifies canonical_digest stability for identical signer sets.
func TestRotationSummaryV2Deterministic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Generate two keypairs and build artifacts to ensure canonical digest stable.
	_, priv1, _ := ed25519.GenerateKey(rand.Reader)
	_, priv2, _ := ed25519.GenerateKey(rand.Reader)
	signers := []notary.WeightedRotationSigner{{ID: "b-signer", Alg: "ED25519", Weight: 1}, {ID: "a-signer", Alg: "ED25519", Weight: 1}}
	art1, err := notary.BuildWeightedRotationArtifact("set", "prevhash", 2, signers, []string{"ed25519"}, time.Now())
	if err != nil {
		t.Fatalf("build1: %v", err)
	}
	_ = notary.AttachEd25519Signature(&art1, priv1, "a-signer", "ED25519", 1)
	_ = notary.AttachEd25519Signature(&art1, priv2, "b-signer", "ED25519", 1)
	art2, err := notary.BuildWeightedRotationArtifact("set", "prevhash", 2, signers, []string{"ed25519"}, time.Now())
	if err != nil {
		t.Fatalf("build2: %v", err)
	}
	if art1.CanonicalDigest != art2.CanonicalDigest {
		t.Fatalf("expected identical canonical digest, got %s vs %s", art1.CanonicalDigest, art2.CanonicalDigest)
	}
}

// contains helper (avoid importing strings repeatedly in tests for minimal footprint)
func containsStr(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (func() bool { return ecIndex(haystack, needle) >= 0 })()
}

// ecIndex naive substring search (KISS, small test scope); returns index or -1.
func ecIndex(s, sub string) int {
	if sub == "" {
		return 0
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
