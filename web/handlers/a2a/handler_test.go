package a2a_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/a2a"
	a2aHandler "github.com/mauriciomferz/Gauth_go/web/handlers/a2a"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	h := a2aHandler.NewHandler()
	h.RegisterRoutes(r)

	return r
}

func TestA2AIssuanceAndVerification(t *testing.T) {
	r := setupRouter()

	// 1. Issue initial token (Start Chain)
	agent1 := a2a.AgentIdentity{ID: "agent-1", Type: "orch"}
	req1 := a2aHandler.IssueTokenRequest{
		Audience: "agent-2",
		Subject:  agent1,
	}
	body, _ := json.Marshal(req1)
	req, _ := http.NewRequest("POST", "/a2a/token", bytes.NewBuffer(body))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var token1 a2a.A2ATransactionToken
	if err := json.Unmarshal(w.Body.Bytes(), &token1); err != nil {
		t.Fatalf("Failed to parse token: %v", err)
	}

	if token1.Context.ChainID == "" {
		t.Fatal("Expected ChainID")
	}
	if len(token1.Context.Hops) != 0 {
		t.Fatal("Expected 0 hops for initial token (or 1 if we count start? implementation creates empty hops for start)")
		// My implementation: Start new chain -> builder -> empty hops.
	}

	// 2. Extend Chain (Agent 2 calls Agent 3)
	// Agent 2 receives token1, validates it, and requests new token for Agent 3
	agent2 := a2a.AgentIdentity{ID: "agent-2", Type: "worker"}
	req2 := a2aHandler.IssueTokenRequest{
		Audience: "agent-3",
		Subject:  agent2,
		Context:  &token1.Context,
	}
	body, _ = json.Marshal(req2)
	req, _ = http.NewRequest("POST", "/a2a/token", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", w.Code)
	}

	var token2 a2a.A2ATransactionToken
	if err := json.Unmarshal(w.Body.Bytes(), &token2); err != nil {
		t.Fatalf("Failed to parse token 2: %v", err)
	}

	if len(token2.Context.Hops) != 1 {
		t.Fatalf("Expected 1 hop, got %d", len(token2.Context.Hops))
	}
	if token2.Context.Hops[0].Caller.ID != "agent-2" {
		t.Errorf("Expected caller agent-2, got %s", token2.Context.Hops[0].Caller.ID)
	}

	// 3. Verify Chain
	verifyReq := a2aHandler.VerifyChainRequest{
		Context: token2.Context,
	}
	body, _ = json.Marshal(verifyReq)
	req, _ = http.NewRequest("POST", "/a2a/verify", bytes.NewBuffer(body))
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK verify, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestListAgents(t *testing.T) {
	r := setupRouter()

	req, _ := http.NewRequest("GET", "/a2a/agents", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", w.Code)
	}
}
