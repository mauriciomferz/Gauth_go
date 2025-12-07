package modellimits

import (
	"time"

	"github.com/gin-gonic/gin"
)

// API wraps the Handler to provide HTTP endpoints
type API struct {
	Handler *Handler
}

// NewAPI returns a new API instance
func NewAPI(h *Handler) *API {
	return &API{Handler: h}
}

// RegisterRoutes mounts the endings
func (a *API) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/api/v1/model")
	{
		g.POST("/validate", a.HandleValidate)
		g.GET("/limits/snapshot", a.HandleSnapshot)
		g.GET("/limits/attestation", a.HandleAttestation)
		// g.GET("/limits/attestation/keys", a.HandleKeys) // If needed
		g.POST("/limits/attestation/verify", a.HandleVerify)
		g.GET("/limits/attestation/stream", a.HandleStream)
		g.GET("/limits/audit/verify", a.HandleAuditVerify)
		g.GET("/limits/audit/anchor/verify", a.HandleAnchorVerify)
	}
}

// HandleValidate enforces limits
func (a *API) HandleValidate(c *gin.Context) {
	var in struct {
		ModelID      string `json:"model_id"`
		UserID       string `json:"user_id"`
		InputTokens  int    `json:"input_tokens"`
		OutputTokens int    `json:"output_tokens"`
	}
	if err := c.ShouldBindJSON(&in); err != nil || in.ModelID == "" || in.InputTokens < 0 {
		c.JSON(400, gin.H{"success": false, "error": "invalid_payload"})
		return
	}

	res := a.Handler.CheckLimit(in.ModelID, in.UserID, in.InputTokens, in.OutputTokens)

	if !res.Allowed {
		resp := gin.H{
			"success":  false,
			"error":    res.Error,
			"model_id": in.ModelID,
		}
		if in.UserID != "" {
			resp["user_id"] = in.UserID
		}
		if res.Limit > 0 {
			resp["limit"] = res.Limit
		}
		if res.WindowSeconds > 0 {
			resp["window_seconds"] = res.WindowSeconds
		}
		if res.Period != "" {
			resp["period"] = res.Period
		}

		// Add specific fields based on error type for backward compatibility
		switch res.Error {
		case "model_user_input_limit_exceeded", "model_limit_exceeded":
			resp["input_tokens"] = in.InputTokens
		case "model_user_output_limit_exceeded", "model_output_limit_exceeded":
			resp["output_tokens"] = in.OutputTokens
		}

		code := 400
		if res.Error == "model_rate_limit_exceeded" || res.Error == "model_user_rate_limit_exceeded" {
			code = 429
		}
		c.JSON(code, resp)
		return
	}

	c.JSON(200, gin.H{
		"success":        true,
		"model_id":       in.ModelID,
		"input_tokens":   in.InputTokens,
		"output_tokens":  in.OutputTokens,
		"limit_enforced": res.LimitEnforced,
		"input_limit":    res.Limit,
		"rate_limit":     res.RateLimit,
	})
}

// HandleSnapshot returns the current limits snapshot
func (a *API) HandleSnapshot(c *gin.Context) {
	hash, at, models, users := a.Handler.ComputeSnapshot()
	c.JSON(200, gin.H{
		"success":      true,
		"hash":         hash,
		"generated_at": at.Format(time.RFC3339Nano),
		"model_limits": models,
		"user_limits":  users,
	})
}

// HandleAttestation generates an attestation
func (a *API) HandleAttestation(c *gin.Context) {
	reason := c.Query("reason")
	if reason == "" {
		reason = "on_demand"
	}

	att, err := a.Handler.BuildUnsignedAttestation()
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}
	att = a.Handler.MaybeAugmentAndSign(att)
	if att.Reason == "" {
		att.Reason = reason
	}

	c.JSON(200, att)
}

// HandleVerify validates attestation
func (a *API) HandleVerify(c *gin.Context) {
	var att ModelLimitsAttestation
	if err := c.ShouldBindJSON(&att); err != nil {
		c.JSON(400, gin.H{"code": "attestation_invalid_json", "message": "attestation verify invalid JSON", "details": gin.H{"http_path": c.FullPath()}})
		return
	}

	res := a.Handler.VerifyAttestation(att)

	out := gin.H{
		"success": true,
		"valid":   res.Valid,
	}
	if res.Error != "" {
		out["error"] = res.Error
	}
	if res.Kid != "" {
		out["kid"] = res.Kid
	}
	if res.SigMode != "" {
		out["sig_mode"] = res.SigMode
	}
	if res.CombinedHash != "" {
		out["combined_hash"] = res.CombinedHash
	}

	c.JSON(200, out)
}

// HandleAuditVerify checks audit log
func (a *API) HandleAuditVerify(c *gin.Context) {
	entries, lastHash, valid, err := a.Handler.VerifyAudit()
	if err != nil {
		if err.Error() == "audit_disabled" {
			c.JSON(200, gin.H{"success": false, "error": "audit_disabled"})
		} else {
			c.JSON(500, gin.H{"success": false, "error": err.Error()})
		}
		return
	}
	c.JSON(200, gin.H{"success": true, "entries": entries, "last_hash": lastHash, "valid": valid})
}

// HandleAnchorVerify checks anchor log
func (a *API) HandleAnchorVerify(c *gin.Context) {
	entries, lastHash, valid, err := a.Handler.VerifyAnchor()
	if err != nil {
		if err.Error() == "anchor_disabled" {
			c.JSON(200, gin.H{"success": false, "error": "anchor_disabled"})
		} else {
			c.JSON(500, gin.H{"success": false, "error": err.Error()})
		}
		return
	}
	c.JSON(200, gin.H{"success": true, "entries": entries, "last_hash": lastHash, "valid": valid})
}

// HandleStream provides SSE of attestations
func (a *API) HandleStream(c *gin.Context) {
	ch := a.Handler.SubscribeAttestation()
	defer a.Handler.UnsubscribeAttestation(ch)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	c.SSEvent("connected", gin.H{"ts": time.Now().UTC().Format(time.RFC3339Nano)})
	c.Writer.Flush()

	// Emit current state immediately so client doesn't wait for next event
	if att, err := a.Handler.BuildUnsignedAttestation(); err == nil {
		att = a.Handler.MaybeAugmentAndSign(att)
		att.Reason = "initial_state"
		c.SSEvent("attestation", att)
		c.Writer.Flush()
	}

	notify := c.Writer.CloseNotify()
	for {
		select {
		case <-notify:
			return
		case att := <-ch:
			c.SSEvent("attestation", att)
			c.Writer.Flush()
		}
	}
}
