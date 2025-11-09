package main

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// helper to build HS256 JWT
func buildHS256(secret string, claims map[string]any) string {
	header := map[string]string{"alg": "HS256", "typ": "JWT"}
	hBytes, _ := json.Marshal(header)
	pBytes, _ := json.Marshal(claims)
	hEnc := base64.RawURLEncoding.EncodeToString(hBytes)
	pEnc := base64.RawURLEncoding.EncodeToString(pBytes)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(hEnc + "." + pEnc))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return hEnc + "." + pEnc + "." + sig
}

// helper to build RS256 JWT using provided private key + kid
func buildRS256(priv *rsa.PrivateKey, kid string, claims map[string]any) string {
	header := map[string]string{"alg": "RS256", "typ": "JWT", "kid": kid}
	hBytes, _ := json.Marshal(header)
	pBytes, _ := json.Marshal(claims)
	hEnc := base64.RawURLEncoding.EncodeToString(hBytes)
	pEnc := base64.RawURLEncoding.EncodeToString(pBytes)
	toSign := []byte(hEnc + "." + pEnc)
	hashed := sha256.Sum256(toSign)
	sig, _ := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hashed[:])
	sEnc := base64.RawURLEncoding.EncodeToString(sig)
	return hEnc + "." + pEnc + "." + sEnc
}

// simple indirection for hash identifier (avoids extra import conflicts)
// removed helper (direct usage of crypto.SHA256)

// minimal JWKS server
func startJWKS(keys map[string]*rsa.PrivateKey) *httptest.Server {
	type jwk struct {
		Kid string `json:"kid"`
		Kty string `json:"kty"`
		Alg string `json:"alg"`
		Use string `json:"use"`
		N   string `json:"n"`
		E   string `json:"e"`
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var out struct {
			Keys []jwk `json:"keys"`
		}
		for kid, priv := range keys {
			n := base64.RawURLEncoding.EncodeToString(priv.PublicKey.N.Bytes())
			eBytes := big.NewInt(int64(priv.PublicKey.E)).Bytes()
			e := base64.RawURLEncoding.EncodeToString(eBytes)
			out.Keys = append(out.Keys, jwk{Kid: kid, Kty: "RSA", Alg: "RS256", Use: "sig", N: n, E: e})
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	return httptest.NewServer(handler)
}

func TestAuthMiddleware_HS256_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	os.Setenv("GAUTH_AI_DEMO_JWT_SECRET", "testsecret")
	defer os.Unsetenv("GAUTH_AI_DEMO_JWT_SECRET")
	r.Use(authMiddleware())
	r.GET("/protected", func(c *gin.Context) { c.String(200, "ok") })
	claims := map[string]any{"sub": "user1", "exp": float64(time.Now().Add(1 * time.Hour).Unix())}
	token := buildHS256("testsecret", claims)
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_HS256_Expired(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	os.Setenv("GAUTH_AI_DEMO_JWT_SECRET", "testsecret")
	defer os.Unsetenv("GAUTH_AI_DEMO_JWT_SECRET")
	r.Use(authMiddleware())
	r.GET("/protected", func(c *gin.Context) { c.String(200, "ok") })
	claims := map[string]any{"sub": "user1", "exp": float64(time.Now().Add(-2 * time.Hour).Unix())}
	token := buildHS256("testsecret", claims)
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestAuthMiddleware_RS256_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	priv, _ := rsa.GenerateKey(rand.Reader, 1024) // test key
	keys := map[string]*rsa.PrivateKey{"kid1": priv}
	srv := startJWKS(keys)
	defer srv.Close()
	os.Setenv("GAUTH_AI_DEMO_JWKS_URL", srv.URL)
	os.Setenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG", "RS256")
	defer func() { os.Unsetenv("GAUTH_AI_DEMO_JWKS_URL"); os.Unsetenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG") }()
	r := gin.New()
	r.Use(authMiddleware())
	r.GET("/protected", func(c *gin.Context) { c.String(200, "ok") })
	claims := map[string]any{"sub": "user2", "exp": float64(time.Now().Add(1 * time.Hour).Unix())}
	token := buildRS256(priv, "kid1", claims)
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d body=%s", w.Code, w.Body.String())
	}
}

func TestAuthMiddleware_RS256_KidNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	priv, _ := rsa.GenerateKey(rand.Reader, 1024)
	// JWKS with different kid
	keys := map[string]*rsa.PrivateKey{"otherkid": priv}
	srv := startJWKS(keys)
	defer srv.Close()
	os.Setenv("GAUTH_AI_DEMO_JWKS_URL", srv.URL)
	os.Setenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG", "RS256")
	defer func() { os.Unsetenv("GAUTH_AI_DEMO_JWKS_URL"); os.Unsetenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG") }()
	r := gin.New()
	r.Use(authMiddleware())
	r.GET("/protected", func(c *gin.Context) { c.String(200, "ok") })
	claims := map[string]any{"sub": "user3", "exp": float64(time.Now().Add(1 * time.Hour).Unix())}
	// token signed with kid1 which is not in JWKS
	token := buildRS256(priv, "kid1", claims)
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401 got %d", w.Code)
	}
}

func TestAuthMiddleware_RS256_JWKSFetchError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Use unreachable URL to force network error on fetch
	os.Setenv("GAUTH_AI_DEMO_JWKS_URL", "http://127.0.0.1:9/.well-known/jwks.json")
	os.Setenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG", "RS256")
	// Force cache refresh by clearing existing cache and expiry
	jwksCache = nil
	jwksExpiry = time.Now().Add(-1 * time.Second)
	defer func() { os.Unsetenv("GAUTH_AI_DEMO_JWKS_URL"); os.Unsetenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG") }()
	// Build a token that will attempt RS256 path; the public key won't matter because fetch fails first.
	// Clear any negative cache entries from prior tests
	jwksNegative = nil
	priv, _ := rsa.GenerateKey(rand.Reader, 1024)
	claims := map[string]any{"sub": "user-fetch-error", "exp": float64(time.Now().Add(1 * time.Hour).Unix())}
	uniqueKid := fmt.Sprintf("kid-fetch-error-%d", time.Now().UnixNano())
	token := buildRS256(priv, uniqueKid, claims)
	r := gin.New()
	r.Use(authMiddleware())
	r.GET("/protected", func(c *gin.Context) { c.String(200, "ok") })
	req, _ := http.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401 got %d", w.Code)
	}
	// Assert reason field == jwks_fetch_error
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if body["reason"] != "jwks_fetch_error" {
		t.Fatalf("expected reason jwks_fetch_error got %v body=%s", body["reason"], w.Body.String())
	}
}

func TestAuthMiddleware_RS256_NegativeKidCaching(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// JWKS server with single key 'validkid'
	privValid, _ := rsa.GenerateKey(rand.Reader, 1024)
	reqCount := 0
	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount++
		type jwk struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		}
		n := base64.RawURLEncoding.EncodeToString(privValid.PublicKey.N.Bytes())
		eBytes := big.NewInt(int64(privValid.PublicKey.E)).Bytes()
		e := base64.RawURLEncoding.EncodeToString(eBytes)
		out := struct {
			Keys []jwk `json:"keys"`
		}{Keys: []jwk{{Kid: "validkid", Kty: "RSA", Alg: "RS256", Use: "sig", N: n, E: e}}}
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer jwksSrv.Close()
	os.Setenv("GAUTH_AI_DEMO_JWKS_URL", jwksSrv.URL)
	os.Setenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG", "RS256")
	os.Setenv("GAUTH_AI_DEMO_JWKS_NEGATIVE_TTL_SECONDS", "30")
	defer func() {
		os.Unsetenv("GAUTH_AI_DEMO_JWKS_URL")
		os.Unsetenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG")
		os.Unsetenv("GAUTH_AI_DEMO_JWKS_NEGATIVE_TTL_SECONDS")
	}()
	// Reset caches
	jwksCache = nil
	jwksNegative = nil
	jwksExpiry = time.Now().Add(-1 * time.Second)
	// Build tokens with missing kid 'missingkid'
	privMissing, _ := rsa.GenerateKey(rand.Reader, 1024)
	claims := map[string]any{"sub": "user-missing", "exp": float64(time.Now().Add(1 * time.Hour).Unix())}
	token1 := buildRS256(privMissing, "missingkid", claims)
	token2 := buildRS256(privMissing, "missingkid", claims)
	r := gin.New()
	r.Use(authMiddleware())
	r.GET("/p", func(c *gin.Context) { c.String(200, "ok") })
	// First request (should trigger JWKS fetch + kid_not_found)
	req, _ := http.NewRequest("GET", "/p", nil)
	req.Header.Set("Authorization", "Bearer "+token1)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 401 {
		t.Fatalf("expected 401 first miss got %d", w.Code)
	}
	// Second request (negative cache hit; should NOT trigger second JWKS fetch)
	req2, _ := http.NewRequest("GET", "/p", nil)
	req2.Header.Set("Authorization", "Bearer "+token2)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != 401 {
		t.Fatalf("expected 401 second miss got %d", w2.Code)
	}
	if reqCount != 1 {
		t.Fatalf("expected single JWKS HTTP fetch, got %d", reqCount)
	}
}

func TestJWKSBackgroundRefresh(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// Short cache TTL and small refresh factor to force early refresh
	ttl := 4 // seconds
	factor := 0.25
	var fetches atomic.Int32
	priv, _ := rsa.GenerateKey(rand.Reader, 1024)
	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fetches.Add(1)
		type jwk struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		}
		n := base64.RawURLEncoding.EncodeToString(priv.PublicKey.N.Bytes())
		eBytes := big.NewInt(int64(priv.PublicKey.E)).Bytes()
		e := base64.RawURLEncoding.EncodeToString(eBytes)
		out := struct {
			Keys []jwk `json:"keys"`
		}{Keys: []jwk{{Kid: "bkid", Kty: "RSA", Alg: "RS256", Use: "sig", N: n, E: e}}}
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer jwksSrv.Close()
	os.Setenv("GAUTH_AI_DEMO_JWKS_URL", jwksSrv.URL)
	os.Setenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG", "RS256")
	os.Setenv("GAUTH_AI_DEMO_JWKS_CACHE_SECONDS", fmt.Sprintf("%d", ttl))
	os.Setenv("GAUTH_AI_DEMO_JWKS_BG_REFRESH_FACTOR", fmt.Sprintf("%f", factor))
	defer func() {
		os.Unsetenv("GAUTH_AI_DEMO_JWKS_URL")
		os.Unsetenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG")
		os.Unsetenv("GAUTH_AI_DEMO_JWKS_CACHE_SECONDS")
		os.Unsetenv("GAUTH_AI_DEMO_JWKS_BG_REFRESH_FACTOR")
	}()
	// Clear caches
	jwksCache = nil
	jwksExpiry = time.Now().Add(-1 * time.Second)
	// Start background refresh loop manually
	startJWKSBackgroundRefresh()
	// Wait a moment for initial fetch
	time.Sleep(200 * time.Millisecond)
	if fetches.Load() < 1 {
		t.Fatalf("expected initial JWKS fetch, got %d", fetches.Load())
	}
	// Sleep long enough for refresh to trigger: factor * ttl + small buffer (< ttl)
	sleepFor := time.Duration(float64(ttl)*factor)*time.Second + 500*time.Millisecond
	time.Sleep(sleepFor)
	if fetches.Load() < 2 {
		t.Fatalf("expected background refresh second fetch, still %d", fetches.Load())
	}
}

// TestAuthMiddleware_RS256_NegativeKidEviction validates eviction metric when max entries reached.
func TestAuthMiddleware_RS256_NegativeKidEviction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// JWKS server with single valid key
	privValid, _ := rsa.GenerateKey(rand.Reader, 1024)
	jwksSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type jwk struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Alg string `json:"alg"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		}
		n := base64.RawURLEncoding.EncodeToString(privValid.PublicKey.N.Bytes())
		eBytes := big.NewInt(int64(privValid.PublicKey.E)).Bytes()
		e := base64.RawURLEncoding.EncodeToString(eBytes)
		out := struct {
			Keys []jwk `json:"keys"`
		}{Keys: []jwk{{Kid: "validkid", Kty: "RSA", Alg: "RS256", Use: "sig", N: n, E: e}}}
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer jwksSrv.Close()
	os.Setenv("GAUTH_AI_DEMO_JWKS_URL", jwksSrv.URL)
	os.Setenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG", "RS256")
	// Provide API key so /metrics scrape succeeds (we will set header)
	os.Setenv("GAUTH_AI_DEMO_API_KEY", "testkey")
	os.Setenv("GAUTH_AI_DEMO_JWKS_NEGATIVE_TTL_SECONDS", "30")
	os.Setenv("GAUTH_AI_DEMO_JWKS_NEGATIVE_MAX_ENTRIES", "1")
	os.Setenv("GAUTH_AI_DEMO_METRICS", "1") // enable metrics endpoint
	defer func() {
		os.Unsetenv("GAUTH_AI_DEMO_JWKS_URL")
		os.Unsetenv("GAUTH_AI_DEMO_JWT_EXPECT_ALG")
		os.Unsetenv("GAUTH_AI_DEMO_JWKS_NEGATIVE_TTL_SECONDS")
		os.Unsetenv("GAUTH_AI_DEMO_JWKS_NEGATIVE_MAX_ENTRIES")
		os.Unsetenv("GAUTH_AI_DEMO_METRICS")
		os.Unsetenv("GAUTH_AI_DEMO_API_KEY")
	}()
	// Reset caches and metrics (metrics registered lazily when server starts)
	jwksCache = nil
	jwksNegative = nil
	jwksExpiry = time.Now().Add(-1 * time.Second)
	// Build two tokens with distinct missing kids (kidA, kidB) to trigger eviction of first
	privMissingA, _ := rsa.GenerateKey(rand.Reader, 1024)
	privMissingB, _ := rsa.GenerateKey(rand.Reader, 1024)
	claims := map[string]any{"sub": "user-missing-evict", "exp": float64(time.Now().Add(1 * time.Hour).Unix())}
	tokenA := buildRS256(privMissingA, "kidA", claims)
	tokenB := buildRS256(privMissingB, "kidB", claims)
	// Initialize minimal metrics required (replicating subset of main.go logic)
	jwksFetchCounter = promauto.NewCounterVec(prometheus.CounterOpts{Name: "ai_demo_jwks_fetch_total", Help: "Total JWKS fetch attempts labeled by result (success|error)."}, []string{"result"})
	jwksKeysGauge = promauto.NewGauge(prometheus.GaugeOpts{Name: "ai_demo_jwks_keys_loaded", Help: "Current number of JWKS public keys cached for RS256 verification."})
	jwksNegHitsCounter = promauto.NewCounter(prometheus.CounterOpts{Name: "ai_demo_jwks_negative_hits_total", Help: "Total number of negative JWKS kid cache hits (prevented redundant fetch attempts)."})
	jwksNegEntriesGauge = promauto.NewGauge(prometheus.GaugeOpts{Name: "ai_demo_jwks_negative_entries", Help: "Current number of entries in the negative JWKS kid cache."})
	jwksNegEvictionsCounter = promauto.NewCounter(prometheus.CounterOpts{Name: "ai_demo_jwks_negative_evictions_total", Help: "Total number of evictions performed on the negative JWKS kid cache due to max entries limit."})
	r := gin.New()
	r.Use(authMiddleware())
	// add /metrics endpoint to scrape counters
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/p", func(c *gin.Context) { c.String(200, "ok") })
	// First request (kidA) -> adds kidA negative entry
	reqA, _ := http.NewRequest("GET", "/p", nil)
	reqA.Header.Set("Authorization", "Bearer "+tokenA)
	wA := httptest.NewRecorder()
	r.ServeHTTP(wA, reqA)
	if wA.Code != 401 {
		t.Fatalf("expected 401 for first missing kid, got %d", wA.Code)
	}
	// Second request (kidB) -> triggers eviction (kidA removed, kidB inserted)
	reqB, _ := http.NewRequest("GET", "/p", nil)
	reqB.Header.Set("Authorization", "Bearer "+tokenB)
	wB := httptest.NewRecorder()
	r.ServeHTTP(wB, reqB)
	if wB.Code != 401 {
		t.Fatalf("expected 401 for second missing kid, got %d", wB.Code)
	}
	// Scrape /metrics to assert eviction counter and entries gauge
	mReq, _ := http.NewRequest("GET", "/metrics", nil)
	mReq.Header.Set("X-API-Key", "testkey")
	mW := httptest.NewRecorder()
	r.ServeHTTP(mW, mReq)
	metricsBody := mW.Body.String()
	if !strings.Contains(metricsBody, "ai_demo_jwks_negative_evictions_total") {
		t.Fatalf("metrics output missing eviction counter: %s", metricsBody)
	}
	// find eviction counter value (should be 1)
	lines := strings.Split(metricsBody, "\n")
	var evictionVal int
	for _, ln := range lines {
		if strings.HasPrefix(ln, "ai_demo_jwks_negative_evictions_total") {
			parts := strings.Fields(ln)
			if len(parts) == 2 {
				if v, err := strconv.Atoi(parts[1]); err == nil {
					evictionVal = v
				}
			}
		}
	}
	if evictionVal != 1 {
		t.Fatalf("expected eviction counter=1 got %d full metrics=%s", evictionVal, metricsBody)
	}
	// gauge should show 1 negative entry (only kidB)
	if !strings.Contains(metricsBody, "ai_demo_jwks_negative_entries 1") {
		t.Fatalf("expected negative entries gauge=1; metrics=%s", metricsBody)
	}
}
