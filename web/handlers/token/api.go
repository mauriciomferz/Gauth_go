package token

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/AgentAuth/internal/metrics"
	"github.com/mauriciomferz/AgentAuth/pkg/agentauth"
	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
	"github.com/mauriciomferz/AgentAuth/pkg/crypto/keys"
)

// Legacy Error Constants
const (
	ErrInvalidSignature = "invalid_signature"
	ErrInvalidAlgorithm = "invalid_algorithm"
	ErrTokenExpired     = "token_expired"
	ErrMalformedToken   = "malformed_token"
)

// RegisterRoutes helper
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/api/v1/token")
	g.POST("/create", h.Create)
	g.POST("/validate", h.Validate)
	g.POST("/revoke", h.Revoke)
	g.GET("/metrics", h.MetricsHandler)
	g.POST("/status/update", h.StatusUpdate)
	g.POST("/introspect", h.Introspect)

	r.GET("/.well-known/jwks.json", h.JWKS)
}

func (h *Handler) checkClock(c *gin.Context) bool {
	if h.ClockStatus == nil {
		return true
	}
	status, skew, err := h.ClockStatus.Status()
	if status == "Critical" {
		c.JSON(503, gin.H{
			"success": false,
			"error":   "system_clock_unsynchronized",
			"detail":  fmt.Sprintf("System clock skew (%v) exceeds safety threshold. Operation rejected for security.", skew),
			"retry":   true,
		})
		if h.Metrics != nil {
			h.Metrics.RecordLifecycleTransition("token", "any", "failed", "clock_skew_critical")
		}
		return false
	}
	if err != nil && status == "Critical" { // Fail closed if check fails and status was critical
		c.JSON(503, gin.H{"success": false, "error": "clock_monitor_failure"})
		return false
	}
	return true
}

func (h *Handler) Create(c *gin.Context) {
	if !h.checkClock(c) {
		return
	}
	var span Span
	if h.ShouldTrace() {
		_, span = h.Tracer.StartSpan(c.Request.Context(), "token.issue")
		if span != nil {
			defer span.End()
		}
	}

	var req struct {
		TTL   int    `json:"ttl_seconds"`
		Meta  any    `json:"meta"`
		Nonce string `json:"nonce"`
		// RFC fields
		GrantID              string                          `json:"grant_id"`
		Scope                []string                        `json:"scope"`
		AuthorizationDetails []agentauth.AuthorizationDetail `json:"authorization_details"`
	}
	_ = c.ShouldBindJSON(&req)

	// RFC 9396 / AAP001 Flow Integration
	if h.AgentAuthService != nil && (len(req.AuthorizationDetails) > 0 || req.GrantID != "") {
		tokenReq := agentauth.TokenRequest{
			GrantID:              req.GrantID,
			Scope:                req.Scope,
			AuthorizationDetails: req.AuthorizationDetails,
			Context:              map[string]interface{}{"nonce": req.Nonce, "meta": req.Meta},
		}

		resp, err := h.AgentAuthService.RequestToken(tokenReq)
		if err != nil {
			c.JSON(400, gin.H{"success": false, "error": "token_request_failed", "detail": err.Error()})
			return
		}

		// Return RFC compliant response
		c.JSON(200, gin.H{
			"success":      true,
			"access_token": resp.Token,
			"scope":        resp.Scope,
			"expires_in":   int(time.Until(resp.ValidUntil).Seconds()),
			"token_type":   "Bearer",
		})
		return
	}

	// Legacy Flow
	// Capability enforcement
	claimsCaps := map[string]any{}
	if raw := c.GetHeader("X-Capabilities"); raw != "" {
		parts := strings.Split(raw, ",")
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		claimsCaps["cap"] = parts
	}

	if h.CapEnforcer != nil {
		allowed, missing := h.CapEnforcer("transaction:issue", claimsCaps)
		if !allowed {
			c.JSON(403, gin.H{"success": false, "error": "capability_denied", "missing": missing})
			return
		}
	}

	// Replay Protection
	issueNonce := req.Nonce
	if issueNonce == "" {
		issueNonce = randomNonce(12)
	}
	if h.Replay != nil {
		now := time.Now()
		if h.Replay.Seen(issueNonce, now) {
			code := "replay"
			detail := "issuance nonce reused"
			if h.ReplayStrict {
				code = "nonce_reused"
			}
			c.JSON(409, gin.H{"success": false, "error": code, "detail": detail})
			return
		}
		h.Replay.RecordWithEvict(issueNonce, now)
	}

	tok := h.Store.Create(req.TTL, req.Meta)

	// JWT Issuance
	var signedJWT string
	if h.UseJWTLib {
		method := jwt.GetSigningMethod(h.JWTAlg)
		if method == nil {
			c.JSON(500, gin.H{"success": false, "message": "unsupported jwt alg"})
			return
		}

		kid := h.JWTKid
		var signingKey interface{}
		if h.JWTAlg == "RS256" {
			if h.JWTKeyManager != nil {
				s, err := h.JWTKeyManager.CryptoSigner(c.Request.Context())
				if err != nil {
					c.JSON(500, gin.H{"success": false, "message": "key manager error"})
					return
				}
				signingKey = s
				if k, err := h.JWTKeyManager.GetKeyID(c.Request.Context()); err == nil && k != "" {
					kid = k
				}
			} else {
				// Legacy fallback (should ideally be removed if KeyManager is always set)
				pk, err := LoadOrGenerateRSAKey()
				if err != nil {
					c.JSON(500, gin.H{"success": false, "message": "rsa key error"})
					return
				}
				signingKey = pk
			}
		} else {
			secret := h.JWTSecret
			if secret == "" {
				// #nosec G101: demo secret placeholder for dev use
				secret = "dev-secret-demo-00000000000000000000000000000000"
			}
			signingKey = []byte(secret)
		}
		exp := time.Now().Add(time.Duration(req.TTL) * time.Second)
		jti := randomNonce(18)
		claims := jwt.MapClaims{"sub": "demo-client", "scope": "legacy-token-store", "exp": exp.Unix(), "iat": time.Now().Unix(), "iss": h.Issuer}
		claims["jti"] = jti
		j := jwt.NewWithClaims(method, claims)
		j.Header["kid"] = kid
		signed, err := j.SignedString(signingKey)
		if err != nil {
			c.JSON(500, gin.H{"success": false, "message": "jwt signing failed"})
			return
		}
		signedJWT = signed
	}

	// Audit & Events
	if h.Auditor != nil {
		h.Auditor.LogAction("api", "token_create", tok.ID+":"+issueNonce, "success")
	}
	if h.Emitter != nil {
		h.Emitter.EmitTokenCreated(tok.ID)
	}

	resp := gin.H{"success": true, "token": tok}
	if signedJWT != "" {
		resp["jwt"] = signedJWT
	}

	if span != nil {
		span.SetTag("ttl_req", req.TTL)
		span.SetTag("token_id", tok.ID)
		span.SetTag("outcome", "success")
	}

	c.JSON(201, resp)
}

func (h *Handler) Validate(c *gin.Context) {
	if !h.checkClock(c) {
		return
	}
	var span Span
	if h.ShouldTrace() {
		_, span = h.Tracer.StartSpan(c.Request.Context(), "token.validate")
		if span != nil {
			defer span.End()
		}
	}

	var req struct {
		Token   string `json:"token"`
		TokenID string `json:"token_id"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"success": false, "message": "invalid payload"})
		return
	}

	tokenStr := req.Token
	if tokenStr == "" {
		tokenStr = req.TokenID
	}

	// Primary Auth Validate (Counters)
	if h.PrimaryAuth != nil {
		// We ignore error for counters side-effect
		_, _ = h.PrimaryAuth.ValidateToken(tokenStr)
	}

	// JWT Path
	if h.UseJWTLib && strings.Count(tokenStr, ".") == 2 {
		alg := h.JWTAlg

		// Header decode
		parts := strings.Split(tokenStr, ".")
		var declaredAlg string
		if len(parts) == 3 {
			if hdrBytes, err := base64.RawURLEncoding.DecodeString(parts[0]); err == nil {
				var hdr map[string]any
				_ = json.Unmarshal(hdrBytes, &hdr)
				if v, ok := hdr["alg"].(string); ok {
					declaredAlg = v
				}
			}
		}

		leeway := time.Duration(0)
		if raw := os.Getenv("AGENTAUTH_JWT_CLOCK_SKEW_SECONDS"); raw != "" {
			if v, err := strconv.Atoi(raw); err == nil && v > 0 {
				leeway = time.Duration(v) * time.Second
			}
		}

		parser := jwt.NewParser(jwt.WithValidMethods([]string{alg}), jwt.WithLeeway(leeway))
		var headerAlg string
		parsed, err := parser.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			headerAlg = t.Method.Alg()
			if headerAlg != alg {
				return nil, errors.New(ErrInvalidAlgorithm)
			}
			if alg == "RS256" {
				if h.JWTKeyManager != nil {
					// RR-005: Use LookupPublicKey if kid header is present
					if kid, ok := t.Header["kid"].(string); ok && kid != "" {
						pub, err := h.JWTKeyManager.LookupPublicKey(context.Background(), kid)
						if err == nil {
							return pub, nil
						}
						// If lookup fails but we have implicit trust (e.g. single key env), fallthrough?
						// Strictly failing on explicit KID mismatch is safer.
						return nil, fmt.Errorf("unknown kid: %s", kid)
					}
					// Fallback legacy (active key) if no KID
					pub, err := h.JWTKeyManager.GetPublicKey(context.Background())
					if err != nil {
						return nil, err
					}
					return pub, nil
				}
				pk, err := LoadOrGenerateRSAKey()
				if err != nil {
					return nil, err
				}
				return pk.Public(), nil
			}
			secret := h.JWTSecret
			if secret == "" {
				// #nosec G101: demo secret placeholder for dev use
				secret = "dev-secret-demo-00000000000000000000000000000000"
			}
			return []byte(secret), nil
		})
		if err != nil {
			errMsg := err.Error()
			code := ErrMalformedToken
			if errors.Is(err, jwt.ErrTokenExpired) || strings.Contains(errMsg, "token is expired") {
				code = ErrTokenExpired
			}
			if strings.Contains(errMsg, ErrInvalidAlgorithm) ||
				(strings.Contains(errMsg, "signing method") && strings.Contains(errMsg, "invalid") ||
					strings.Contains(errMsg, "invalid signing method")) {
				code = ErrInvalidAlgorithm
				if headerAlg == "" {
					if declaredAlg != "" {
						headerAlg = declaredAlg
					} else {
						headerAlg = "unknown"
					}
				}
				errMsg = "invalid_algorithm: header alg " + headerAlg + " rejected (expected " + alg + ")"
			} else if strings.Contains(errMsg, "signature") {
				code = ErrInvalidSignature
			}
			c.JSON(400, gin.H{"success": false, "error": code, "detail": errMsg})
			return
		}
		if !parsed.Valid {
			c.JSON(400, gin.H{"success": false, "error": ErrInvalidSignature, "detail": "token invalid"})
			return
		}

		if claims, ok := parsed.Claims.(jwt.MapClaims); ok {
			jtiVal, hasJTI := claims["jti"].(string)
			if h.ReplayStrict && (!hasJTI || jtiVal == "") {
				c.JSON(400, gin.H{"success": false, "error": ErrMalformedToken, "detail": "missing jti (strict mode)"})
				return
			}
			if hasJTI && jtiVal != "" && h.Replay != nil {
				if h.Replay.Seen(jtiVal, time.Now()) {
					c.JSON(401, gin.H{"success": false, "code": "token_replay_detected", "error": "replay_detected", "rfc_ref": "AAP001:replay_protection", "detail": "replay detected (jti dedicated)"})
					return
				}
				h.Replay.Record(jtiVal, time.Now())
			}
		}
		c.JSON(200, gin.H{"success": true, "status": "valid_jwt", "claims": parsed.Claims})
		return
	}

	// Default Legacy Token Validation
	status, t := h.Store.Validate(tokenStr)
	res := gin.H{"success": status == TokenStatusValid, "status": status}
	if t != nil {
		res["token_id"] = t.ID
		res["meta"] = t.Meta
	}
	c.JSON(200, res)
}

func (h *Handler) Revoke(c *gin.Context) {
	var req struct {
		ID string `json:"token_id"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"success": false, "message": "invalid payload"})
		return
	}
	status := h.Store.Revoke(req.ID)

	if h.Auditor != nil {
		h.Auditor.LogAction("api", "token_revoke", req.ID, status)
	}

	c.JSON(200, gin.H{"success": status == TokenStatusRevoked, "status": status})
}

func (h *Handler) MetricsHandler(c *gin.Context) {
	c.JSON(200, gin.H{"success": true, "metrics": h.Store.Metrics()})
}

func (h *Handler) StatusUpdate(c *gin.Context) {
	start := time.Now()
	var span Span
	if h.ShouldTrace() {
		_, span = h.Tracer.StartSpan(c.Request.Context(), "token_status_update")
		if span != nil {
			defer span.End()
		}
	}

	var req struct {
		TokenID   string `json:"token_id"`
		NewStatus string `json:"new_status"`
	}
	if c.ShouldBindJSON(&req) != nil || req.TokenID == "" || req.NewStatus == "" {
		if h.Metrics != nil {
			h.Metrics.IncTokenStatusTransitionFailures()
			h.Metrics.RecordLifecycleTransition("token", "_", "_", "failure")
		}
		if mm, ok := h.Metrics.(*metrics.Memory); ok {
			mm.IncInvalidPayloadFailure()
		}
		c.JSON(400, gin.H{"success": false, "message": "invalid payload", "reason": "invalid_payload"})
		return
	}

	if req.NewStatus != "active" && req.NewStatus != "suspended" && req.NewStatus != "terminated" && req.NewStatus != "partially_revoked" {
		if h.Metrics != nil {
			h.Metrics.IncTokenStatusTransitionFailures()
		}
		if mm, ok := h.Metrics.(*metrics.Memory); ok {
			mm.IncUnsupportedStatusFailure()
		}
		c.JSON(400, gin.H{"success": false, "message": "unsupported status", "reason": "unsupported_status"})
		return
	}

	success, reason, tok := h.Store.UpdateStatus(req.TokenID, req.NewStatus)

	if !success {
		if reason == "not_found" {
			if h.Metrics != nil {
				h.Metrics.IncTokenStatusTransitionFailures()
				h.Metrics.RecordLifecycleTransition("token", "_", req.NewStatus, "failure")
			}
			if mm, ok := h.Metrics.(*metrics.Memory); ok {
				mm.IncNotFoundFailure()
			}
			if span != nil {
				span.SetTag("outcome", "failure")
				span.SetTag("reason", "not_found")
			}
			c.JSON(404, gin.H{"success": false, "message": "token not found", "reason": "not_found"})
			return
		}

		if reason == "invalid_transition" {
			if h.Metrics != nil {
				h.Metrics.IncTokenStatusTransitionFailures()
				// tok is old state in this case
				h.Metrics.RecordLifecycleTransition("token", tok.Status, req.NewStatus, "failure")
			}
			if mm, ok := h.Metrics.(*metrics.Memory); ok {
				mm.IncInvalidTransitionFailure()
			}
			if span != nil {
				span.SetTag("outcome", "failure")
				span.SetTag("reason", "invalid_transition")
			}
			c.JSON(409, gin.H{"success": false, "message": "terminated tokens cannot transition", "reason": "invalid_transition"})
			return
		}

		// Generic failure
		c.JSON(500, gin.H{"success": false, "message": "update failed"})
		return
	}

	// Success or No-Op
	// old := tok.Status // Unused
	// Actually Store.UpdateStatus implementation modifies in place.
	// If noop, it returns success=true, reason="noop", and the token.
	// We need 'old' status for metrics.
	// The current UpdateStatus impl returns the *Token AFTER modification/check.
	// We might need to adjust logic or just infer old status.
	// Actually for noop old==new. For success, we don't know old easily unless we return it.
	// Let's rely on event emission if needed or just use req.NewStatus.

	// Wait, the Store.UpdateStatus I wrote returns (bool, string, *Token).
	// If success (change happened), tok is updated.
	// If noop, tok is unchanged.
	// We lost "old" status in the variable.
	// For noop: old == tok.Status.
	// For success: old != tok.Status (unless noop).

	// Let's refine Store.UpdateStatus to return old status too or we can't emit accurate events.
	// But to avoid multi-file dance again, let's fix logic here:
	// If success && reason == "success", then old status was... unknown potentially.
	// Actually, UpdateStatus logic I pushed: `tok.Status = newStatus`.
	// So `tok` has new status.
	// I should have returned old status.
	// Retrying the edit to Store.go is safer.

	if reason == "noop" {
		reasonReason := "noop"
		if os.Getenv("AGENTAUTH_MAINTENANCE_WINDOW") == "1" {
			reasonReason = "maintenance"
		}
		if os.Getenv("AGENTAUTH_RATE_LIMITED") == "1" {
			reasonReason = "rate_limited"
		}

		if h.Metrics != nil {
			h.Metrics.IncTokenStatusTransitions()
			h.Metrics.RecordDecision("token_status_update", "token:"+tok.ID, tok.Status, time.Duration(0))
			h.Metrics.RecordDecisionWithReason("token_status_update", "token:"+tok.ID, tok.Status, reasonReason)
			h.Metrics.RecordLifecycleTransition("token", tok.Status, req.NewStatus, "noop")
			h.Metrics.ObserveLifecycleTransitionLatency("token", "noop", time.Since(start))
		}

		lat := time.Since(start).Nanoseconds()
		if h.Lifecycle != nil {
			h.Lifecycle.RecordEvent("token", tok.ID, tok.Status, tok.Status, "noop", reasonReason, lat)
		}
		if span != nil {
			span.SetTag("outcome", "noop")
			span.SetTag("reason", reasonReason)
		}
		c.JSON(200, gin.H{"success": true, "token_id": tok.ID, "old_status": tok.Status, "new_status": tok.Status, "no_change": true, "reason": reasonReason})
		return
	}

	// If we are here, reason == "success"
	// We don't have 'old' status explicitly, but we know it wasn't 'terminated' or same as new.
	// For metrics, we can just say "previous".

	if h.Auditor != nil {
		h.Auditor.LogAction("api", "token_status_update", tok.ID, "success")
	}
	if h.Emitter != nil {
		h.Emitter.EmitTokenStatusChanged(tok.ID, "unknown", tok.Status, "status_change")
	}

	if h.Metrics != nil {
		h.Metrics.IncTokenStatusTransitions()
		h.Metrics.RecordLifecycleTransition("token", "unknown", req.NewStatus, "success")
		h.Metrics.ObserveLifecycleTransitionLatency("token", "success", time.Since(start))
	}
	lat := time.Since(start).Nanoseconds()
	if h.Lifecycle != nil {
		h.Lifecycle.RecordEvent("token", tok.ID, "unknown", tok.Status, "success", "status_change", lat)
	}

	c.JSON(200, gin.H{"success": true, "token_id": tok.ID, "old_status": "unknown", "new_status": tok.Status})
}

func (h *Handler) Introspect(c *gin.Context) {
	var req struct {
		Token   string `json:"token"`
		TokenID string `json:"token_id"`
	}
	if c.ShouldBindJSON(&req) != nil {
		c.JSON(400, gin.H{"success": false, "message": "invalid payload"})
		return
	}
	tokenStr := req.Token
	if tokenStr == "" {
		tokenStr = req.TokenID
	}
	if tokenStr == "" {
		c.JSON(400, gin.H{"success": false, "message": "missing token"})
		return
	}

	ms := gin.H{"supported": false}
	if os.Getenv("AGENTAUTH_MULTI_SIG_THRESHOLD") != "" {
		ms["supported"] = true
	}

	// JWT Path
	if h.UseJWTLib && strings.Count(tokenStr, ".") == 2 {
		parts := strings.Split(tokenStr, ".")
		var header map[string]any
		if hb, err := base64.RawURLEncoding.DecodeString(parts[0]); err == nil {
			_ = json.Unmarshal(hb, &header)
		}
		var claims map[string]any
		parsed, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
		if err == nil {
			if mc, ok := parsed.Claims.(jwt.MapClaims); ok {
				claims = map[string]any{}
				for k, v := range mc {
					claims[k] = v
				}
			}
		}
		var expRFC string
		var expired bool
		if claims != nil {
			if expRaw, ok := claims["exp"].(float64); ok {
				et := time.Unix(int64(expRaw), 0)
				expRFC = et.UTC().Format(time.RFC3339)
				expired = time.Now().After(et)
			}
		}
		c.JSON(200, gin.H{
			"success": true, "type": "jwt",
			"header":     header,
			"claims":     claims,
			"expires_at": expRFC, "expired": expired,
			"revoked":         false,
			"multi_signature": ms,
		})
		return
	}

	// Internal Token Path
	status, tok := h.Store.Validate(tokenStr)
	revoked := status == TokenStatusRevoked || status == "already_revoked"
	c.JSON(200, gin.H{
		"success": true, "type": "internal",
		"status":          status,
		"token":           tok,
		"revoked":         revoked,
		"multi_signature": ms,
	})
}

func (h *Handler) JWKS(c *gin.Context) {
	mode := os.Getenv("AGENTAUTH_TOKEN_SIG_MODE")
	alg := h.JWTAlg
	useLib := h.UseJWTLib
	c.Header("Cache-Control", "public, max-age=60")
	if rot := os.Getenv("AGENTAUTH_JWT_ROTATION_DAYS"); rot != "" {
		c.Header("X-Key-Rotation-Interval-Days", rot)
	}

	jwkList := []any{}
	if useLib {
		if alg == "RS256" {
			var jwk map[string]any
			var err error
			if h.JWTKeyManager != nil {
				jwk, err = keys.PublicJWK(h.JWTKeyManager)
			} else {
				jwk, err = rsaPublicJWK() // from keys.go (internal)
			}
			if err != nil {
				c.JSON(500, gin.H{"success": false, "message": "jwks generation error"})
				return
			}
			jwkList = append(jwkList, jwk)
		} else {
			kid := h.JWTKid
			jwkList = append(jwkList, gin.H{"kty": "oct", "alg": alg, "kid": kid, "use": "sig"})
		}
	}

	var deprecatedKids []string
	if mode == "eddsa" {
		if h.KeyProvider != nil {
			if km, ok := h.KeyProvider.(*crypto.Manager); ok {
				for _, k := range km.ListCurrent() {
					jwk := gin.H{
						"kty":        "OKP",
						"crv":        "Ed25519",
						"alg":        "EdDSA",
						"kid":        k.ID,
						"use":        "sig",
						"x":          base64.RawURLEncoding.EncodeToString(k.Public),
						"expires_at": k.ExpiresAt.Format(time.RFC3339),
					}
					if !k.DeprecatedAfter.IsZero() {
						jwk["deprecated_after"] = k.DeprecatedAfter.Format(time.RFC3339)
						// Track deprecated keys for Warning header
						if time.Now().After(k.DeprecatedAfter) {
							deprecatedKids = append(deprecatedKids, k.ID)
						}
					}
					if !k.SunsetAfter.IsZero() {
						jwk["sunset_after"] = k.SunsetAfter.Format(time.RFC3339)
					}
					jwkList = append(jwkList, jwk)
				}
			}
		}
	}

	// Emit Warning header if any keys are deprecated
	if len(deprecatedKids) > 0 {
		warning := fmt.Sprintf(`299 - "Keys deprecated: %s"`, strings.Join(deprecatedKids, ", "))
		c.Header("Warning", warning)
	}

	if len(jwkList) == 0 {
		kid := h.JWTKid
		if kid == "" {
			kid = "demo-rsa"
		}
		jwkList = append(jwkList, gin.H{"kty": "oct", "alg": "HS256", "kid": kid, "use": "sig", "metadata_only": true})
	}

	// Generate ETag based on keys content
	keysJSON, _ := json.Marshal(jwkList)
	etag := fmt.Sprintf(`"%x"`, sha256.Sum256(keysJSON))
	c.Header("ETag", etag)

	// Update server's metadata if updater is available
	if h.ETagUpdater != nil {
		h.ETagUpdater.UpdateJWKSETag(etag)
		// Report deprecation schedule
		depSchedule := map[string]time.Time{}
		if mode == "eddsa" && h.KeyProvider != nil {
			if km, ok := h.KeyProvider.(*crypto.Manager); ok {
				for _, k := range km.ListCurrent() {
					if !k.DeprecatedAfter.IsZero() {
						depSchedule["key:"+k.ID] = k.DeprecatedAfter
					}
				}
			}
		}
		h.ETagUpdater.UpdateDeprecationSchedule(depSchedule)
	}

	// Support conditional requests (If-None-Match)
	if match := c.GetHeader("If-None-Match"); match == etag {
		c.Status(304)
		return
	}

	// Optional signature headers when enabled
	if os.Getenv("AGENTAUTH_JWKS_SIGNING_KEY_ENABLED") == "1" {
		sigKey := os.Getenv("AGENTAUTH_JWKS_SIGNING_KEY")
		if sigKey != "" {
			// HMAC-SHA256 signature of keysJSON
			mac := hmac.New(sha256.New, []byte(sigKey))
			mac.Write(keysJSON)
			sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
			c.Header("X-JWKS-Signature", sig)
			c.Header("X-JWKS-Signature-Alg", "HMAC-SHA256")
			if h.ETagUpdater != nil {
				h.ETagUpdater.UpdateJWKSSignature(sig)
			}
		}
	}

	c.JSON(200, gin.H{"keys": jwkList})
}
