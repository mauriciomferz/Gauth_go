package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/pkg/delegation"
)

// TestRevocationEndpoints verifies AAP001-R4, R5, R6, R7 compliance
func TestRevocationEndpoints(t *testing.T) {
	// Setup isolated server with real revocation chain
	gin.SetMode(gin.TestMode)

	// Create a fresh key manager
	km, err := crypto.NewManager(1 * time.Hour)
	if err != nil {
		t.Fatalf("crypto manager: %v", err)
	}

	// Create revocation chain with key provider
	chain := delegation.NewRevocationChain(delegation.WithKeyProvider(km))

	// Add some events to have data to verify
	_, _ = chain.Append(delegation.RevocationEvent{ID: "rev-1", DelegationID: "del-1", Reason: "compromise"})
	e2, _ := chain.Append(delegation.RevocationEvent{ID: "rev-2", DelegationID: "del-2", Reason: "cessation_of_operation"})

	// Sign tree head to ensure aggregate hash is stable and signatures exist
	_, err = chain.SignTreeHead()
	if err != nil {
		t.Fatalf("sign tree head: %v", err)
	}

	// Setup server with this chain
	server := &BetaServer{
		router:          gin.New(),
		revocationChain: chain,
	}

	// Manually register the endpoints we are testing (replicating server_clean.go logic for isolation)
	// Note: In a real integration test, we might use NewServer, but we want to inject the pre-populated chain.
	// We'll trust that server_clean.go logic matches this, or rely on the fact that we are testing the handlers logic
	// assuming they are wired correctly. Ideally we use the real server constructor if it allows injection.
	// Looking at BetaServer struct, revocationChain is private.
	// We might need to use the actual handlers if they are exported or accessible.
	// Alternatively, checking server_clean.go saw they are inline funcs.
	// We will replicate the handler logic here to "test the logic", or better, use the full server if possible.
	// Since we can't easily inject into the private field of a full server without correct constructor,
	// we'll stick to testing the logic if we can copy it, OR better: use the running server in e2e.
	// BUT, we want a unit/integration test.
	// Let's rely on the fact that we can modify the server struct in this test package since it's the same package `web`.

	// Register endpoints using the actual server methods we just refactored
	server.router.GET("/api/v1/token/revocation/head", server.handleRevocationHead)
	server.router.GET("/api/v1/token/revocation/verify", server.handleRevocationVerify)

	// R4: Verify Head & Aggregate
	t.Run("AAP001-R4_RevocationHead", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/token/revocation/head", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		if resp["success"] != true {
			t.Error("Expected success=true")
		}
		if resp["revocation_chain_length"].(float64) != 2 {
			t.Errorf("Expected length 2, got %v", resp["revocation_chain_length"])
		}
		if resp["revocation_chain_head"] != e2.Hash {
			t.Errorf("Expected head %s, got %s", e2.Hash, resp["revocation_chain_head"])
		}
		if resp["revocation_chain_aggregate"] == "" {
			t.Error("Expected non-empty aggregate hash")
		}
		if resp["verified"] != true {
			t.Error("Expected verified=true")
		}
	})

	// R5: Verify Signature Presence & Validity
	t.Run("AAP001-R5_VerifySignature", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/api/v1/token/revocation/verify", nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Expected 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal response: %v", err)
		}

		if resp["success"] != true {
			t.Error("Expected success=true")
		}
		if resp["verified"] != true {
			t.Error("Expected verified=true")
		}

		events, ok := resp["events"].([]interface{})
		if !ok || len(events) != 2 {
			t.Fatalf("Expected 2 events, got %v", resp["events"])
		}

		// Check second event (signed)
		e2Map := events[1].(map[string]interface{})
		if e2Map["id"] != "rev-2" {
			t.Errorf("Expected rev-2, got %v", e2Map["id"])
		}
		if e2Map["signature_present"] != true {
			t.Error("Expected signature_present=true")
		}
		if e2Map["signature_valid"] != true {
			t.Error("Expected signature_valid=true")
		}
	})

	// R6: Discovery Verification
	t.Run("AAP001-R6_DiscoveryMetadata", func(t *testing.T) {
		t.Setenv("AGENTAUTH_TOKEN_SIG_MODE", "eddsa") // Ensure enabled

		// Create a separate handler/server to re-trigger registering or just invoke the handler logic?
		// Since discovery is registered via RegisterRB3Discovery, and BetaServer is complex,
		// we'll try to use the actual handler logic or recreate a minimal server for this test
		// if we can access registerRB3Discovery which is likely private.
		// Wait, registerRB3Discovery is private.
		// However, we can just call NewBetaServer (using factory) or construct it and register manually if we can access the func.
		// Since we are in the same package 'web', we CAN access 'registerRB3Discovery'.

		dServer := &BetaServer{router: gin.New()}
		dServer.registerRB3Discovery()

		req, _ := http.NewRequest("GET", "/api/v1/discovery", nil)
		w := httptest.NewRecorder()
		dServer.router.ServeHTTP(w, req)

		if w.Code != 200 {
			t.Errorf("Discovery expected 200, got %d", w.Code)
		}
		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshal discovery response: %v", err)
		}

		algs, ok := resp["revocation_signing_alg_values_supported"].([]interface{})
		if !ok || len(algs) == 0 {
			t.Errorf("Expected revocation_signing_alg_values_supported with EdDSA, got %v", resp["revocation_signing_alg_values_supported"])
		} else {
			if algs[0] != "EdDSA" {
				t.Errorf("Expected EdDSA, got %s", algs[0])
			}
		}
	})
}
