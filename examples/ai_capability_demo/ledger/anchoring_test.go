package ledger_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/examples/ai_capability_demo/ledger"
	"github.com/mauriciomferz/Gauth_go/internal/ai"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth_aap_001"
)

// Minimal in-memory copies of global objects for isolated test
func setupRouterAndLedger() (*gin.Engine, *ledger.Ledger) {
	gin.SetMode(gin.TestMode)
	l := ledger.New()
	integration := ai.NewServerIntegration()
	integration.EnableEnforcement(true)
	router := gin.New()
	router.POST("/demo/enforce", func(c *gin.Context) {
		var req struct {
			Action string `json:"action"`
		}
		if c.ShouldBindJSON(&req) != nil {
			c.JSON(400, gin.H{"error": "bad"})
			return
		}
		claims := map[string]any{"ai_entity_type": "assistant", "ai_entity_verified": true, "algorithmic_accountability": true}
		allowed, missing, meta := integration.EnforceAICapabilities(req.Action, claims)
		payload := map[string]any{"action": req.Action, "allowed": allowed, "missing": missing, "timestamp": time.Now().UTC().Format(time.RFC3339Nano)}
		b, _ := json.Marshal(payload)
		l.Append("decision", string(b), "")
		c.JSON(200, gin.H{"allowed": allowed, "metadata": meta})
	})
	router.POST("/demo/poa/issue", func(c *gin.Context) {
		poa := &gauth_aap_001.PowerOfAttorney{ID: "t1", Version: 1, Grantor: "g1", Grantee: "g2", Scope: []string{"transaction:read"}, Restrictions: map[string]string{}, ValidFrom: time.Now().UTC(), ValidUntil: time.Now().UTC().Add(time.Hour), Status: gauth_aap_001.POAStatusActive, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
		b, _ := json.Marshal(map[string]any{"event": "poa_issue", "poa_id": poa.ID})
		l.Append("poa_issue", string(b), "")
		c.JSON(200, gin.H{"poa_id": poa.ID})
	})
	return router, l
}

func TestAnchoringDecisionAndPOA(t *testing.T) {
	r, l := setupRouterAndLedger()
	// enforce decision
	w := httptest.NewRecorder()
	req := newJSONRequest(t, "POST", "/demo/enforce", map[string]any{"action": "transaction:read"})
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("enforce status %d", w.Code)
	}
	// issue poa
	w2 := httptest.NewRecorder()
	req2 := newJSONRequest(t, "POST", "/demo/poa/issue", map[string]any{})
	r.ServeHTTP(w2, req2)
	if w2.Code != 200 {
		t.Fatalf("poa issue status %d", w2.Code)
	}
	if l.Size() != 2 {
		t.Fatalf("expected 2 ledger entries, got %d", l.Size())
	}
	if l.LatestRoot() == "" {
		t.Fatalf("expected non-empty root after anchoring events")
	}
}

// helper to build JSON body
func newJSONRequest(t *testing.T, method, path string, body any) *http.Request {
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	req, err := http.NewRequest(method, path, bytes.NewReader(b))
	if err != nil {
		t.Fatalf("request error: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}
