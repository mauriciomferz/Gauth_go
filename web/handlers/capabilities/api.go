package capabilities

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/AgentAuth/internal/capability"
)

// API provides HTTP handlers for capability management.
type API struct {
	Handler *Handler
}

// NewAPI creates a new capabilities API.
func NewAPI(h *Handler) *API {
	return &API{Handler: h}
}

// RegisterRoutes registers endpoints on the router.
func (a *API) RegisterRoutes(r *gin.Engine) {
	beta := r.Group("/api/v1/beta/capabilities")
	beta.GET("", a.List)
	beta.POST("/reload", a.Reload)
	beta.POST("/negotiate", a.Negotiate)
	beta.GET("/audit/verify", a.AuditVerify)
	beta.POST("/audit/anchor", a.AuditAnchor)

	// Key endpoint
	r.GET("/api/v1/beta/keys/eddsa/public", a.EdDSAPublicKey)
}

// List returns all registered capabilities.
func (a *API) List(c *gin.Context) {
	caps := capability.DefaultRegistry().List()
	hash := a.Handler.GetRegistryHash()
	prevHash := a.Handler.GetPrevRegistryHash()
	lastChanged := a.Handler.GetRegistryChangeAt()

	resp := gin.H{
		"success":      true,
		"capabilities": caps,
	}
	if hash != "" {
		resp["capability_registry_hash"] = hash
	}
	if prevHash != "" {
		resp["capability_registry_prev_hash"] = prevHash
	}
	if !lastChanged.IsZero() {
		resp["capability_registry_last_changed_at"] = lastChanged.Format(time.RFC3339)
	}
	c.JSON(200, resp)
}

// Reload reloads capability file (if AGENTAUTH_CAPABILITIES_PATH set) and returns summary.
func (a *API) Reload(c *gin.Context) {
	path := os.Getenv("AGENTAUTH_CAPABILITIES_PATH")
	if path == "" {
		c.JSON(400, gin.H{"success": false, "error": "no_capabilities_path", "detail": "AGENTAUTH_CAPABILITIES_PATH not set"})
		return
	}
	before := capability.DefaultRegistry().List()

	if err := a.Handler.LoadFromFile(path); err != nil {
		c.JSON(500, gin.H{"success": false, "error": "reload_failed", "detail": err.Error()})
		return
	}

	after := capability.DefaultRegistry().List()

	source, _, _, _, actions, loaded, _ := a.Handler.GetState()

	c.JSON(200, gin.H{
		"success":             true,
		"capabilities_before": len(before),
		"capabilities_after":  len(after),
		"action_mappings":     actions,
		"source":              source,
		"last_loaded":         loaded.Format(time.RFC3339),
	})
}

// Negotiate performs multi-version capability negotiation.
func (a *API) Negotiate(c *gin.Context) {
	var req struct {
		ClientVersions map[string][]string `json:"client_versions"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.ClientVersions) == 0 {
		c.JSON(400, gin.H{"success": false, "error": "invalid_payload", "code": "capabilities_negotiate_invalid_payload"})
		return
	}
	caps := capability.DefaultRegistry().List()
	regMap := make(map[string]capability.Capability, len(caps))
	for _, cap := range caps {
		regMap[cap.ID] = cap
	}
	agreed := make(map[string]string)
	unsupported := make(map[string][]string)

	a.Handler.mu.RLock()
	strict := a.Handler.LifecycleStrict
	a.Handler.mu.RUnlock()

	for cid, clientVers := range req.ClientVersions {
		regCap, ok := regMap[cid]
		if !ok {
			unsupported[cid] = clientVers
			continue
		}
		serverVers := make(map[string]struct{})
		if regCap.Version != "" {
			serverVers[regCap.Version] = struct{}{}
		}
		for _, v := range regCap.Versions {
			serverVers[v] = struct{}{}
		}
		// Lifecycle strict
		if strict && regCap.DeprecatedAfter != "" {
			if t, err := time.Parse(time.RFC3339, regCap.DeprecatedAfter); err == nil {
				if time.Now().After(t) {
					serverVers = map[string]struct{}{}
				}
			}
		}
		negotiated := ""
		for _, cv := range clientVers {
			if _, ok := serverVers[cv]; ok {
				negotiated = cv
				break
			}
		}
		if negotiated == "" {
			unsupported[cid] = clientVers
		} else {
			agreed[cid] = negotiated
		}
	}
	c.JSON(200, gin.H{"success": true, "agreed": agreed, "unsupported": unsupported, "lifecycle_strict": strict})
}

// AuditVerify returns status of latest capability audit chain tip persistence file.
func (a *API) AuditVerify(c *gin.Context) {
	a.Handler.mu.RLock()
	path := a.Handler.AuditPersistPath
	tip := a.Handler.AuditPrevHash
	a.Handler.mu.RUnlock()

	if path == "" {
		c.JSON(200, gin.H{"success": true, "configured": false})
		return
	}
	// #nosec G304
	b, err := os.ReadFile(path)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": "read_failed"})
		return
	}
	var wrapper struct {
		Payload   json.RawMessage `json:"payload"`
		PrevHash  string          `json:"prev_hash"`
		Hash      string          `json:"hash"`
		Timestamp string          `json:"timestamp"`
	}
	if jerr := json.Unmarshal(b, &wrapper); jerr != nil {
		c.JSON(500, gin.H{"success": false, "error": "invalid_json"})
		return
	}
	// Recompute hash of payload for integrity
	h := sha256.New()
	h.Write([]byte(wrapper.PrevHash))
	h.Write(wrapper.Payload)
	recomputed := fmt.Sprintf("sha256:%x", h.Sum(nil))
	integrity := recomputed == wrapper.Hash

	c.JSON(200, gin.H{
		"success":      true,
		"configured":   true,
		"latest":       gin.H{"hash": wrapper.Hash, "prev_hash": wrapper.PrevHash, "timestamp": wrapper.Timestamp},
		"chain_tip":    tip,
		"integrity_ok": integrity,
	})
}

// AuditAnchor anchors the current capability audit chain tip (prev hash state after latest event) if anchoring enabled.
func (a *API) AuditAnchor(c *gin.Context) {
	if os.Getenv("AGENTAUTH_CAPABILITY_ANCHOR_ENABLE") != "1" {
		c.JSON(403, gin.H{"success": false, "error": "anchoring_disabled"})
		return
	}

	a.Handler.mu.RLock()
	client := a.Handler.AuditClient
	tip := a.Handler.AuditPrevHash
	a.Handler.mu.RUnlock()

	if client == nil {
		c.JSON(500, gin.H{"success": false, "error": "anchor_client_unavailable"})
		return
	}
	if tip == "" {
		c.JSON(400, gin.H{"success": false, "error": "chain_tip_empty"})
		return
	}
	rec, err := client.Anchor(tip)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": "anchor_failure", "detail": err.Error()})
		return
	}

	payload := gin.H{
		"success":     true,
		"hash":        rec.Hash,
		"anchored_at": rec.Timestamp,
		"total":       client.TotalAnchors(),
		"chain_tip":   tip,
		"type":        "capability_audit_chain_tip",
	}
	c.JSON(200, payload)
}

// EdDSAPublicKey exposes the active Ed25519 public key.
func (a *API) EdDSAPublicKey(c *gin.Context) {
	a.Handler.mu.RLock()
	km := a.Handler.KeyManager
	a.Handler.mu.RUnlock()

	// Hardcoded check for signature mode to match server_clean.go
	if os.Getenv("AGENTAUTH_TOKEN_SIG_MODE") != "eddsa" || km == nil {
		c.JSON(200, gin.H{"success": true, "configured": false})
		return
	}
	active := km.Active()
	if active == nil || len(active.Public) != ed25519.PublicKeySize {
		c.JSON(200, gin.H{"success": true, "configured": false})
		return
	}
	c.JSON(200, gin.H{"success": true, "configured": true, "kid": active.ID, "public_key": base64.RawStdEncoding.EncodeToString(active.Public)})
}
