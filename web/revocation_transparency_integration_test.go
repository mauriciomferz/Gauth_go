package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/delegation"
)

// buildMultiSigServer sets up a BetaServer with multi-sig environment, several events, and a signed tree head history.
// buildMultiSigServer sets up a BetaServer with multi-sig environment, several events, and a signed tree head history.
func buildMultiSigServer(t *testing.T) (*BetaServer, crypto.KeyProvider) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	km, err := crypto.NewManager(time.Hour)
	if err != nil {
		t.Fatalf("km init: %v", err)
	}
	// Rotate to create additional keys
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate1: %v", err)
	}
	if _, err := km.Rotate(); err != nil {
		t.Fatalf("rotate2: %v", err)
	}

	s := NewBetaServer("", WithKeyProvider(km))
	t.Cleanup(func() { s.Shutdown() })

	// Configure multi-sig threshold environment (count fallback, threshold 2)
	t.Setenv("GAUTH_MULTI_SIG_THRESHOLD", "2")
	rc := delegation.NewRevocationChain(delegation.WithKeyProvider(km))
	// Append several events
	for i := 0; i < 5; i++ {
		_, _ = rc.Append(delegation.RevocationEvent{ID: "rev-int-" + time.Now().Format("150405") + string(rune('a'+i)), DelegationID: "del-int"})
		time.Sleep(3 * time.Millisecond)
	}
	// Sign two tree heads to create history for consistency proof
	if _, err := rc.SignTreeHead(); err != nil {
		t.Fatalf("sign head 1: %v", err)
	}
	_, _ = rc.Append(delegation.RevocationEvent{ID: "rev-int-final", DelegationID: "del-int"})
	if _, err := rc.SignTreeHead(); err != nil {
		t.Fatalf("sign head 2: %v", err)
	}
	s.revocationChain = rc
	return s, km
}

func TestRevocationTransparencyIntegration(t *testing.T) {
	s, kp := buildMultiSigServer(t)
	// 1. Discovery
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/.well-known/gauth-configuration", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("discovery status: %d", w.Code)
	}
	var discovery map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &discovery); err != nil {
		t.Fatalf("discovery unmarshal: %v", err)
	}
	rs, okSupport := discovery["revocation_support"].(map[string]any)
	if !okSupport {
		t.Fatalf("revocation_support missing")
	}
	sthLatest, okLatest := rs["sth_latest"].(map[string]any)
	if !okLatest {
		t.Fatalf("sth_latest missing")
	}
	if v, okRoot := sthLatest["merkle_root"].(string); !okRoot || v == "" {
		t.Fatalf("merkle_root missing in discovery")
	}
	// 2. Root endpoint
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/token/revocation/root", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("root status: %d", w.Code)
	}
	var rootResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &rootResp)
	rootStr, rootOK := rootResp["merkle_root"].(string)
	if !rootOK || rootStr == "" {
		t.Fatalf("empty merkle_root")
	}
	// 3. Fetch verify for list of events (choose last hash)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/token/revocation/verify", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("verify status: %d", w.Code)
	}
	var verifyResp struct {
		Events []map[string]any `json:"events"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &verifyResp); err != nil {
		t.Fatalf("verify unmarshal: %v", err)
	}
	if len(verifyResp.Events) == 0 {
		t.Fatalf("no events returned")
	}
	target := verifyResp.Events[len(verifyResp.Events)-1]
	hashStr, hashOK := target["hash"].(string)
	if !hashOK || hashStr == "" {
		t.Fatalf("target hash missing")
	}
	// 4. Inclusion proof by hash
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/token/revocation/proof?hash="+hashStr, nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("proof status: %d", w.Code)
	}
	var proofResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &proofResp)
	if pr, okProof := proofResp["proof"].([]any); !okProof || len(pr) == 0 {
		t.Fatalf("proof steps missing")
	}
	// 5. Consistency proof from first tree head index 0
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/token/revocation/consistency?start=0", nil)
	s.router.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("consistency status: %d", w.Code)
	}
	var consResp map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &consResp)
	if _, ok := consResp["proof"].(map[string]any); !ok {
		t.Fatalf("consistency proof missing")
	}
	latest, okLatestTH := consResp["latest_tree_head"].(map[string]any)
	if !okLatestTH {
		t.Fatalf("latest_tree_head missing")
	}
	if v, okThresh := latest["threshold"].(float64); !okThresh || v < 2 {
		t.Fatalf("expected multi-sig threshold >=2")
	}
	// 6. Multi-sig verification via library (convert JSON back to struct)
	// Build SignedTreeHead minimal instance for VerifyTreeHeadMultiSig
	payload, _ := json.Marshal(latest)
	var libSTH delegation.SignedTreeHead
	if err := json.Unmarshal(payload, &libSTH); err != nil {
		t.Fatalf("sth unmarshal: %v", err)
	}
	if err := delegation.VerifyTreeHeadMultiSig(&libSTH, kp); err != nil {
		t.Fatalf("library multi-sig verify failed: %v", err)
	}
}
