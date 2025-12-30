package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/internal/ai"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth_aap_001"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// helper to set up router with existing main.go logic (subset) for tests
func setupTestRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(authMiddleware())
	integration := ai.NewServerIntegration()
	if os.Getenv("AGENTAUTH_AI_DEMO_TEST_DISABLE_ENFORCEMENT") == "1" {
		integration.EnableEnforcement(false)
	} else {
		integration.EnableEnforcement(true)
	}
	apiHandler := ai.NewAPIHandler(integration)
	apiHandler.RegisterRoutes(r)
	// Add minimal enforce route copy (avoid starting full server initialization)
	r.POST("/demo/enforce", func(c *gin.Context) {
		var req struct {
			Action string         `json:"action"`
			Claims map[string]any `json:"claims"`
			POAID  string         `json:"poa_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		allowed, missing, meta := integration.EnforceAICapabilities(req.Action, req.Claims)
		if req.POAID != "" { // replicate PoA validation logic
			result := "not_found"
			poaRepoMu.RLock()
			poa, ok := poaRepo[req.POAID]
			poaRepoMu.RUnlock()
			if ok {
				now := time.Now().UTC()
				if poa.Status == agentauth_aap_001.POAStatusRevoked {
					result = "revoked"
					allowed = false
				} else if now.After(poa.ValidUntil) || now.Before(poa.ValidFrom) {
					result = "expired"
					allowed = false
				} else {
					match := false
					for _, s := range poa.Scope {
						if s == req.Action {
							match = true
							break
						}
					}
					if !match {
						result = "scope_mismatch"
						allowed = false
					} else {
						result = "success"
					}
				}
				meta["poa_id"] = poa.ID
			} else {
				allowed = false
			}
			_ = result // result currently unused in test response
		}
		c.JSON(200, gin.H{"result": map[string]any{"allowed": allowed, "missing": missing, "metadata": meta}})
	})
	// Multisig endpoints (subset for tests)
	r.POST("/demo/poa/:id/sign", func(c *gin.Context) {
		id := c.Param("id")
		var req struct {
			Signer string `json:"signer"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		poaRepoMu.Lock()
		poa, ok := poaRepo[id]
		if !ok {
			poaRepoMu.Unlock()
			c.JSON(404, gin.H{"error": "poa_not_found"})
			return
		}
		if poa.Status != agentauth_aap_001.POAStatusDraft {
			poaRepoMu.Unlock()
			c.JSON(400, gin.H{"error": "poa_not_draft", "status": poa.Status})
			return
		}
		// signer validation
		known := false
		for _, s := range poa.Signers {
			if s == req.Signer {
				known = true
				break
			}
		}
		if !known {
			poaRepoMu.Unlock()
			c.JSON(400, gin.H{"error": "unknown_signer"})
			return
		}
		if poa.MultiSignatures == nil {
			poa.MultiSignatures = map[string]*agentauth_aap_001.POASignature{}
		}
		if _, exists := poa.MultiSignatures[req.Signer]; exists {
			poaRepoMu.Unlock()
			c.JSON(409, gin.H{"error": "duplicate_signature"})
			return
		}
		digest, canon, _ := agentauth_aap_001.CanonicalPOADigest(poa)
		dummySig := base64.StdEncoding.EncodeToString([]byte("test-" + req.Signer + "-" + digest))
		poa.MultiSignatures[req.Signer] = &agentauth_aap_001.POASignature{Algorithm: "ed25519", KeyID: req.Signer + "-kid", DigestHex: digest, SigBase64: dummySig, Canonical: canon}
		poa.UpdatedAt = time.Now().UTC()
		poaRepo[id] = poa
		poaRepoMu.Unlock()
		c.JSON(200, gin.H{"signed": true, "poa_id": id, "signatures": len(poa.MultiSignatures)})
	})
	r.POST("/demo/poa/:id/finalize", func(c *gin.Context) {
		id := c.Param("id")
		poaRepoMu.Lock()
		poa, ok := poaRepo[id]
		if !ok {
			poaRepoMu.Unlock()
			c.JSON(404, gin.H{"error": "poa_not_found"})
			return
		}
		if poa.Status != agentauth_aap_001.POAStatusDraft {
			poaRepoMu.Unlock()
			c.JSON(400, gin.H{"error": "not_draft", "status": poa.Status})
			return
		}
		if len(poa.MultiSignatures) < poa.Threshold {
			poaRepoMu.Unlock()
			c.JSON(400, gin.H{"error": "threshold_not_met", "current": len(poa.MultiSignatures), "need": poa.Threshold})
			return
		}
		poa.Status = agentauth_aap_001.POAStatusActive
		poa.UpdatedAt = time.Now().UTC()
		poaRepo[id] = poa
		poaRepoMu.Unlock()
		c.JSON(200, gin.H{"finalized": true, "poa_id": id, "status": poa.Status})
	})
	r.GET("/demo/poa/:id/status", func(c *gin.Context) {
		id := c.Param("id")
		poaRepoMu.RLock()
		poa, ok := poaRepo[id]
		poaRepoMu.RUnlock()
		if !ok {
			c.JSON(404, gin.H{"error": "poa_not_found"})
			return
		}
		c.JSON(200, gin.H{"poa_id": id, "status": poa.Status, "signatures": len(poa.MultiSignatures), "threshold": poa.Threshold})
	})
	return r
}

func issueTestPOA(id string, status agentauth_aap_001.POAStatus, scope []string, durSeconds int) *agentauth_aap_001.PowerOfAttorney {
	now := time.Now().UTC()
	poa := &agentauth_aap_001.PowerOfAttorney{
		ID:           id,
		Version:      1,
		Grantor:      "grantor",
		Grantee:      "grantee",
		Scope:        scope,
		Restrictions: map[string]string{},
		ValidFrom:    now.Add(-time.Minute),
		ValidUntil:   now.Add(time.Duration(durSeconds) * time.Second),
		Status:       status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	poaRepoMu.Lock()
	poaRepo[poa.ID] = poa
	poaRepoMu.Unlock()
	return poa
}

// helper to sign HS256 JWT used in integrity tests
func signHS256(claims map[string]any, secret string) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadBytes, _ := json.Marshal(claims)
	payload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(header + "." + payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return header + "." + payload + "." + sig
}

func TestPOAValidationSuccess(t *testing.T) {
	os.Setenv("AGENTAUTH_AI_DEMO_JWT_SECRET", "secret123")
	// use bypass so we can focus on PoA logic without JWT signature
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "1")
	// disable enforcement checks (focus on PoA presence)
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_DISABLE_ENFORCEMENT", "1")
	router := setupTestRouter(t)
	poa := issueTestPOA("poa-success", agentauth_aap_001.POAStatusActive, []string{"transaction:read"}, 300)
	body := `{"action":"transaction:read","claims":{"ai_entity_type":"agent","ai_entity_verified":true,"ai_agent_registered":true,"algorithmic_accountability":true,"risk_level":"medium","jurisdiction":"US"},"poa_id":"` + poa.ID + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/demo/enforce", http.NoBody)
	req.Body = httptestBody(body)
	router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	res := resp["result"].(map[string]any)
	if allowed, _ := res["allowed"].(bool); !allowed {
		t.Fatalf("expected allowed true")
	}
	os.Unsetenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH")
	os.Unsetenv("AGENTAUTH_AI_DEMO_TEST_DISABLE_ENFORCEMENT")
}

func TestPOAValidationRevoked(t *testing.T) {
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "1")
	router := setupTestRouter(t)
	poa := issueTestPOA("poa-revoked", agentauth_aap_001.POAStatusRevoked, []string{"transaction:read"}, 300)
	body := `{"action":"transaction:read","claims":{},"poa_id":"` + poa.ID + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/demo/enforce", http.NoBody)
	req.Body = httptestBody(body)
	router.ServeHTTP(w, req)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	res := resp["result"].(map[string]any)
	if allowed, _ := res["allowed"].(bool); allowed {
		t.Fatalf("expected revoked PoA to disallow")
	}
}

func TestPOAValidationExpired(t *testing.T) {
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "1")
	router := setupTestRouter(t)
	// expires immediately
	now := time.Now().UTC()
	poa := &agentauth_aap_001.PowerOfAttorney{ID: "poa-expired", Version: 1, Grantor: "g", Grantee: "gr", Scope: []string{"transaction:read"}, Restrictions: map[string]string{}, ValidFrom: now.Add(-time.Hour), ValidUntil: now.Add(-time.Minute), Status: agentauth_aap_001.POAStatusActive, CreatedAt: now, UpdatedAt: now}
	poaRepoMu.Lock()
	poaRepo[poa.ID] = poa
	poaRepoMu.Unlock()
	body := `{"action":"transaction:read","claims":{},"poa_id":"poa-expired"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/demo/enforce", http.NoBody)
	req.Body = httptestBody(body)
	router.ServeHTTP(w, req)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	res := resp["result"].(map[string]any)
	if allowed, _ := res["allowed"].(bool); allowed {
		t.Fatalf("expected expired PoA to disallow")
	}
}

func TestPOAValidationScopeMismatch(t *testing.T) {
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "1")
	router := setupTestRouter(t)
	poa := issueTestPOA("poa-scope", agentauth_aap_001.POAStatusActive, []string{"transaction:read"}, 300)
	body := `{"action":"transaction:execute","claims":{},"poa_id":"` + poa.ID + `"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/demo/enforce", http.NoBody)
	req.Body = httptestBody(body)
	router.ServeHTTP(w, req)
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	res := resp["result"].(map[string]any)
	if allowed, _ := res["allowed"].(bool); allowed {
		t.Fatalf("expected scope mismatch to disallow")
	}
}

// Integrity mismatch: digest
func TestPOAIntegrityDigestMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "intsecret"
	os.Setenv("AGENTAUTH_AI_DEMO_JWT_SECRET", secret)
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "0")
	r := setupTestRouter(t)
	poa := issueTestPOA("poa-digest", agentauth_aap_001.POAStatusActive, []string{"transaction:read"}, 300)
	now := time.Now().UTC()
	digest, _, _ := agentauth_aap_001.CanonicalPOADigest(poa)
	claims := map[string]any{"sub": poa.Grantee, "iss": "gauth-demo", "aud": "gauth-demo-api", "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(time.Hour).Unix(), "poa_id": poa.ID, "poa_digest": digest + "tamper", "poa_version": poa.Version, "token_version": "et_v1"}
	token := signHS256(claims, secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/demo/enforce", httptestBody(`{"action":"transaction:read","claims":{},"poa_id":"`+poa.ID+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["reason"] != "poa_digest_mismatch" {
		t.Fatalf("expected poa_digest_mismatch got %v", resp["reason"])
	}
	os.Unsetenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH")
}

// Integrity mismatch: version
func TestPOAIntegrityVersionMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "intsecret2"
	os.Setenv("AGENTAUTH_AI_DEMO_JWT_SECRET", secret)
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "0")
	r := setupTestRouter(t)
	poa := issueTestPOA("poa-version", agentauth_aap_001.POAStatusActive, []string{"transaction:read"}, 300)
	now := time.Now().UTC()
	digest, _, _ := agentauth_aap_001.CanonicalPOADigest(poa)
	claims := map[string]any{"sub": poa.Grantee, "iss": "gauth-demo", "aud": "gauth-demo-api", "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(time.Hour).Unix(), "poa_id": poa.ID, "poa_digest": digest, "poa_version": poa.Version + 1, "token_version": "et_v1"}
	token := signHS256(claims, secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/demo/enforce", httptestBody(`{"action":"transaction:read","claims":{},"poa_id":"`+poa.ID+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
	var resp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["reason"] != "poa_version_mismatch" {
		t.Fatalf("expected poa_version_mismatch got %v", resp["reason"])
	}
	os.Unsetenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH")
}

// Integrity success path with correct digest + version claims
func TestPOAIntegritySuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "intsecretsuccess"
	os.Setenv("AGENTAUTH_AI_DEMO_JWT_SECRET", secret)
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "0")
	r := setupTestRouter(t)
	poa := issueTestPOA("poa-success-int", agentauth_aap_001.POAStatusActive, []string{"transaction:read"}, 300)
	now := time.Now().UTC()
	digest, _, _ := agentauth_aap_001.CanonicalPOADigest(poa)
	claims := map[string]any{"sub": poa.Grantee, "iss": "gauth-demo", "aud": "gauth-demo-api", "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(30 * time.Minute).Unix(), "poa_id": poa.ID, "poa_digest": digest, "poa_version": poa.Version, "token_version": "et_v1"}
	token := signHS256(claims, secret)
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/demo/enforce", httptestBody(`{"action":"transaction:read","claims":{},"poa_id":"`+poa.ID+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 got %d (body=%s)", w.Code, w.Body.String())
	}
	os.Unsetenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH")
}

// Metrics increment test: ensure integrity failure increments counter when metrics enabled
func TestPOAIntegrityMetricsIncrement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "intsecretmetrics"
	os.Setenv("AGENTAUTH_AI_DEMO_JWT_SECRET", secret)
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "0")
	// Initialize metrics counter manually if not already (simulate AGENTAUTH_AI_DEMO_METRICS=1 block)
	if poaIntegrityFailuresCounter == nil {
		poaIntegrityFailuresCounter = prometheus.NewCounterVec(prometheus.CounterOpts{Name: "ai_demo_poa_integrity_failures_total", Help: "test counter"}, []string{"reason"})
	}
	r := setupTestRouter(t)
	poa := issueTestPOA("poa-metrics", agentauth_aap_001.POAStatusActive, []string{"transaction:read"}, 300)
	now := time.Now().UTC()
	digest, _, _ := agentauth_aap_001.CanonicalPOADigest(poa)
	claims := map[string]any{"sub": poa.Grantee, "iss": "gauth-demo", "aud": "gauth-demo-api", "iat": now.Unix(), "nbf": now.Unix(), "exp": now.Add(time.Hour).Unix(), "poa_id": poa.ID, "poa_digest": digest + "x", "poa_version": poa.Version, "token_version": "et_v1"}
	token := signHS256(claims, secret)
	before := testutil.ToFloat64(poaIntegrityFailuresCounter.WithLabelValues("digest_mismatch"))
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/demo/enforce", httptestBody(`{"action":"transaction:read","claims":{},"poa_id":"`+poa.ID+`"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 got %d", w.Code)
	}
	after := testutil.ToFloat64(poaIntegrityFailuresCounter.WithLabelValues("digest_mismatch"))
	if after != before+1 {
		t.Fatalf("expected digest_mismatch counter to increment by 1 (before=%f after=%f)", before, after)
	}
	os.Unsetenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH")
}

// --- Multisig Tests ---

// TestMultisigQuorumActivation verifies a draft PoA becomes active after threshold signatures.
func TestMultisigQuorumActivation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Setenv("AGENTAUTH_AI_DEMO_JWT_SECRET", "multisigsecret")
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "1")
	router := setupTestRouter(t)
	// Prepare draft via direct in-memory creation (simpler than hitting prepare endpoint for unit scope)
	now := time.Now().UTC()
	poa := &agentauth_aap_001.PowerOfAttorney{ID: "poa-multi", Version: 1, Grantor: "g1", Grantee: "gr1", Scope: []string{"transaction:read"}, ValidFrom: now, ValidUntil: now.Add(time.Hour), Status: agentauth_aap_001.POAStatusDraft, CreatedAt: now, UpdatedAt: now, Signers: []string{"alice", "bob"}, Threshold: 2, MultiSignatures: map[string]*agentauth_aap_001.POASignature{}}
	poaRepoMu.Lock()
	poaRepo[poa.ID] = poa
	poaRepoMu.Unlock()
	// Simulate signatures hitting the sign endpoint for realism
	for _, signer := range []string{"alice", "bob"} {
		body := `{"signer":"` + signer + `"}`
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/demo/poa/"+poa.ID+"/sign", http.NoBody)
		req.Body = httptestBody(body)
		router.ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatalf("signature submission failed %d body=%s", w.Code, w.Body.String())
		}
	}
	// Finalize
	wFin := httptest.NewRecorder()
	reqFin := httptest.NewRequest("POST", "/demo/poa/"+poa.ID+"/finalize", http.NoBody)
	router.ServeHTTP(wFin, reqFin)
	if wFin.Code != 200 {
		t.Fatalf("finalize failed: %d body=%s", wFin.Code, wFin.Body.String())
	}
	// Validate status changed
	poaRepoMu.RLock()
	updated := poaRepo[poa.ID]
	poaRepoMu.RUnlock()
	if updated.Status != agentauth_aap_001.POAStatusActive {
		t.Fatalf("expected status active got %s", updated.Status)
	}
}

// TestMultisigDuplicateSignature ensures duplicate signer submission is rejected.
func TestMultisigDuplicateSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "1")
	router := setupTestRouter(t)
	now := time.Now().UTC()
	poa := &agentauth_aap_001.PowerOfAttorney{ID: "poa-multi-dup", Version: 1, Grantor: "g1", Grantee: "gr1", Scope: []string{"transaction:read"}, ValidFrom: now, ValidUntil: now.Add(time.Hour), Status: agentauth_aap_001.POAStatusDraft, CreatedAt: now, UpdatedAt: now, Signers: []string{"alice", "bob"}, Threshold: 2, MultiSignatures: map[string]*agentauth_aap_001.POASignature{}}
	poaRepoMu.Lock()
	poaRepo[poa.ID] = poa
	poaRepoMu.Unlock()
	// First signature
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest("POST", "/demo/poa/"+poa.ID+"/sign", http.NoBody)
	req1.Body = httptestBody(`{"signer":"alice"}`)
	router.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Fatalf("first signature failed: %d", w1.Code)
	}
	// Duplicate
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("POST", "/demo/poa/"+poa.ID+"/sign", http.NoBody)
	req2.Body = httptestBody(`{"signer":"alice"}`)
	router.ServeHTTP(w2, req2)
	if w2.Code != 409 {
		t.Fatalf("expected 409 duplicate got %d", w2.Code)
	}
}

// TestMultisigUnknownSigner ensures submission from non-listed signer fails.
func TestMultisigUnknownSigner(t *testing.T) {
	gin.SetMode(gin.TestMode)
	os.Setenv("AGENTAUTH_AI_DEMO_TEST_BYPASS_AUTH", "1")
	router := setupTestRouter(t)
	now := time.Now().UTC()
	poa := &agentauth_aap_001.PowerOfAttorney{ID: "poa-multi-unknown", Version: 1, Grantor: "g1", Grantee: "gr1", Scope: []string{"transaction:read"}, ValidFrom: now, ValidUntil: now.Add(time.Hour), Status: agentauth_aap_001.POAStatusDraft, CreatedAt: now, UpdatedAt: now, Signers: []string{"alice", "bob"}, Threshold: 2, MultiSignatures: map[string]*agentauth_aap_001.POASignature{}}
	poaRepoMu.Lock()
	poaRepo[poa.ID] = poa
	poaRepoMu.Unlock()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/demo/poa/"+poa.ID+"/sign", http.NoBody)
	req.Body = httptestBody(`{"signer":"charlie"}`)
	router.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Fatalf("expected 400 unknown signer got %d", w.Code)
	}
}

func httptestBody(s string) *nopCloser { return &nopCloser{Reader: *bytesNewBufferString(s)} }

// minimal no-op closer with bytes buffer
type nopCloser struct{ Reader bytesBuffer }

func (n *nopCloser) Read(p []byte) (int, error) { return n.Reader.Read(p) }
func (n *nopCloser) Close() error               { return nil }

// lightweight local buffer types (avoid importing bytes multiple times inside helper for clarity)
type bytesBuffer struct {
	b []byte
	i int
}

func bytesNewBufferString(s string) *bytesBuffer { return &bytesBuffer{b: []byte(s)} }
func (bb *bytesBuffer) Read(p []byte) (int, error) {
	if bb.i >= len(bb.b) {
		return 0, ioEOF
	}
	n := copy(p, bb.b[bb.i:])
	bb.i += n
	return n, nil
}

var ioEOF = func() error { type eof struct{}; return eofErr }()
var eofErr error = &eofType{}

type eofType struct{}

func (e *eofType) Error() string { return "EOF" }
