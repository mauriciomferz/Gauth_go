package web

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestRB3DiscoveryBasic verifies required fields and caching semantics.
func TestRB3DiscoveryBasic(t *testing.T) {
    srv := NewBetaServer(":0")
    w := httptest.NewRecorder()
    req := httptest.NewRequest("GET", "/api/v1/discovery", nil)
    srv.router.ServeHTTP(w, req)
    if w.Code != 200 { t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String()) }
    var payload map[string]any
    if err := json.Unmarshal(w.Body.Bytes(), &payload); err != nil { t.Fatalf("json decode: %v", err) }
    // Required keys
    required := []string{"schema_version","digest_domains","active_digest_domain","token_algorithms","replay_strict_mode","poa_version_current","capabilities_hash","rotation_tip_hash","taxonomy_supported","generated_at","etag"}
    for _, k := range required {
        if _, ok := payload[k]; !ok { t.Fatalf("missing key %s", k) }
    }
    if v, ok := payload["schema_version"].(float64); !ok || int(v) != 1 { t.Fatalf("schema_version unexpected %#v", payload["schema_version"]) }
    if dom, ok := payload["active_digest_domain"].(string); !ok || dom == "" { t.Fatalf("active_digest_domain invalid %#v", payload["active_digest_domain"]) }
    // Caching headers
    cc := w.Header().Get("Cache-Control")
    if cc != "max-age=30" { t.Fatalf("Cache-Control mismatch %s", cc) }
    etag := w.Header().Get("ETag")
    if etag == "" { t.Fatalf("missing ETag header") }
    // Second request with If-None-Match should 304
    w2 := httptest.NewRecorder()
    req2 := httptest.NewRequest("GET", "/api/v1/discovery", nil)
    req2.Header.Set("If-None-Match", etag)
    srv.router.ServeHTTP(w2, req2)
    if w2.Code != 304 { t.Fatalf("expected 304 got %d", w2.Code) }
    if w2.Body.Len() != 0 { t.Fatalf("expected empty body on 304 got %s", w2.Body.String()) }
}
