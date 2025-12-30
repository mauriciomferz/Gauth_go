package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDiscoveryETagPresent(t *testing.T) {
	bs := NewTestServerNoSeed(t)
	// Wait for async startup anchor to complete
	time.Sleep(100 * time.Millisecond)
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/AGENTAUTH-configuration", nil)
	bs.router.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status=%d", rr.Code)
	}
	etag := rr.Header().Get("ETag")
	if etag == "" {
		t.Fatalf("expected ETag header present")
	}
	// second request should produce same ETag (deterministic)
	rr2 := httptest.NewRecorder()
	bs.router.ServeHTTP(rr2, req)
	if rr2.Header().Get("ETag") != etag {
		t.Fatalf("ETag changed across requests: %s vs %s", etag, rr2.Header().Get("ETag"))
	}
}

func TestDiscoverySignatureOptional(t *testing.T) {
	// ensure env not set for unsigned server
	t.Setenv("AGENTAUTH_DISCOVERY_SIGNING_KEY", "")
	t.Setenv("AGENTAUTH_DISCOVERY_SIGNING_KEY_ENABLED", "0")
	unsigned := NewTestServerNoSeed(t)
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/AGENTAUTH-configuration", nil)
	unsigned.router.ServeHTTP(rr, req)
	var doc map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &doc); err != nil {
		t.Fatalf("json parse error: %v", err)
	}
	if _, ok := doc["signature"]; ok {
		t.Fatalf("unexpected signature without key")
	}

	// set key for signed server
	t.Setenv("AGENTAUTH_DISCOVERY_SIGNING_KEY", "demo-secret")
	t.Setenv("AGENTAUTH_DISCOVERY_SIGNING_KEY_ENABLED", "1")
	signed := NewTestServerNoSeed(t)
	rr2 := httptest.NewRecorder()
	signed.router.ServeHTTP(rr2, req)
	if err := json.Unmarshal(rr2.Body.Bytes(), &doc); err != nil {
		t.Fatalf("json parse error signed: %v", err)
	}
	if _, ok := doc["signature"]; !ok {
		t.Fatalf("expected signature field with key set")
	}
}
