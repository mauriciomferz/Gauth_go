package token

import (
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
	"github.com/mauriciomferz/Gauth_go/internal/metrics"
	"github.com/mauriciomferz/Gauth_go/pkg/crypto"
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
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

func (h *Handler) Create(c *gin.Context) {
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
		GrantID              string                      `json:"grant_id"`
		Scope                []string                    `json:"scope"`
		AuthorizationDetails []gauth.AuthorizationDetail `json:"authorization_details"`
	}
	_ = c.ShouldBindJSON(&req)

	// RFC 9396 / RFC-0111 Flow Integration
	if h.GAuthService != nil && (len(req.AuthorizationDetails) > 0 || req.GrantID != "") {
		tokenReq := gauth.TokenRequest{
			GrantID:              req.GrantID,
			Scope:                req.Scope,
			AuthorizationDetails: req.AuthorizationDetails,
			Context:              map[string]interface{}{"nonce": req.Nonce, "meta": req.Meta},
		}

		resp, err := h.GAuthService.RequestToken(tokenReq)
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
			pk, err := LoadOrGenerateRSAKey()
			if err != nil {
				c.JSON(500, gin.H{"success": false, "message": "rsa key error"})
				return
			}
			signingKey = pk
		} else {
			secret := h.JWTSecret
			if secret == "" {
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
		if raw := os.Getenv("GAUTH_JWT_CLOCK_SKEW_SECONDS"); raw != "" {
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
				pk, err := LoadOrGenerateRSAKey()
				if err != nil {
					return nil, err
				}
				return pk.Public(), nil
			}
			secret := h.JWTSecret
			if secret == "" {
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
				(strings.Contains(errMsg, "signing method") && strings.Contains(errMsg, "invalid")) ||
				strings.Contains(errMsg, "invalid signing method") {
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
					c.JSON(401, gin.H{"success": false, "code": "token_replay_detected", "error": "replay_detected", "rfc_ref": "rfc111:replay_protection", "detail": "replay detected (jti dedicated)"})
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

	h.Store.mu.Lock()
	tok, ok := h.Store.tokens[req.TokenID]
	if !ok {
		h.Store.mu.Unlock()
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

	old := tok.Status
	if old == "terminated" && req.NewStatus != "terminated" {
		if h.Metrics != nil {
			h.Metrics.IncTokenStatusTransitionFailures()
			h.Metrics.RecordLifecycleTransition("token", old, req.NewStatus, "failure")
		}
		if mm, ok := h.Metrics.(*metrics.Memory); ok {
			mm.IncInvalidTransitionFailure()
		}
		h.Store.mu.Unlock()
		if span != nil {
			span.SetTag("outcome", "failure")
			span.SetTag("reason", "invalid_transition")
		}
		c.JSON(409, gin.H{"success": false, "message": "terminated tokens cannot transition", "reason": "invalid_transition"})
		return
	}

	if old == req.NewStatus {
		reason := "noop"
		if os.Getenv("GAUTH_MAINTENANCE_WINDOW") == "1" {
			reason = "maintenance"
		}
		if os.Getenv("GAUTH_RATE_LIMITED") == "1" {
			reason = "rate_limited"
		}

		if h.Metrics != nil {
			h.Metrics.IncTokenStatusTransitions()
			h.Metrics.RecordDecision("token_status_update", "token:"+tok.ID, tok.Status, time.Duration(0))
			h.Metrics.RecordDecisionWithReason("token_status_update", "token:"+tok.ID, tok.Status, reason)
			h.Metrics.RecordLifecycleTransition("token", old, req.NewStatus, "noop")
			h.Metrics.ObserveLifecycleTransitionLatency("token", "noop", time.Since(start))
		}
		h.Store.mu.Unlock()

		lat := time.Since(start).Nanoseconds()
		if h.Lifecycle != nil {
			h.Lifecycle.RecordEvent("token", tok.ID, old, tok.Status, "noop", reason, lat)
		}
		if span != nil {
			span.SetTag("outcome", "noop")
			span.SetTag("reason", reason)
		}
		c.JSON(200, gin.H{"success": true, "token_id": tok.ID, "old_status": old, "new_status": tok.Status, "no_change": true, "reason": reason})
		return
	}

	tok.Status = req.NewStatus
	h.Store.mu.Unlock()

	if h.Auditor != nil {
		h.Auditor.LogAction("api", "token_status_update", tok.ID, "success")
	}
	if h.Emitter != nil {
		h.Emitter.EmitTokenStatusChanged(tok.ID, old, tok.Status, "status_change")
	}

	if h.Metrics != nil {
		h.Metrics.IncTokenStatusTransitions()
		h.Metrics.RecordLifecycleTransition("token", old, req.NewStatus, "success")
		h.Metrics.ObserveLifecycleTransitionLatency("token", "success", time.Since(start))
	}
	lat := time.Since(start).Nanoseconds()
	if h.Lifecycle != nil {
		h.Lifecycle.RecordEvent("token", tok.ID, old, tok.Status, "success", "status_change", lat)
	}

	c.JSON(200, gin.H{"success": true, "token_id": tok.ID, "old_status": old, "new_status": tok.Status})
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
	if os.Getenv("GAUTH_MULTI_SIG_THRESHOLD") != "" {
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
	mode := os.Getenv("GAUTH_TOKEN_SIG_MODE")
	alg := h.JWTAlg
	useLib := h.UseJWTLib
	c.Header("Cache-Control", "public, max-age=60")
	if rot := os.Getenv("GAUTH_JWT_ROTATION_DAYS"); rot != "" {
		c.Header("X-Key-Rotation-Interval-Days", rot)
	}

	keys := []any{}
	if useLib {
		if alg == "RS256" {
			jwk, err := rsaPublicJWK() // from keys.go (internal)
			if err != nil {
				c.JSON(500, gin.H{"success": false, "message": "jwks generation error"})
				return
			}
			keys = append(keys, jwk)
		} else {
			kid := h.JWTKid
			keys = append(keys, gin.H{"kty": "oct", "alg": alg, "kid": kid, "use": "sig"})
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
					keys = append(keys, jwk)
				}
			}
		}
	}

	// Emit Warning header if any keys are deprecated
	if len(deprecatedKids) > 0 {
		warning := fmt.Sprintf(`299 - "Keys deprecated: %s"`, strings.Join(deprecatedKids, ", "))
		c.Header("Warning", warning)
	}

	if len(keys) == 0 {
		kid := h.JWTKid
		if kid == "" {
			kid = "demo-rsa"
		}
		keys = append(keys, gin.H{"kty": "oct", "alg": "HS256", "kid": kid, "use": "sig", "metadata_only": true})
	}

	// Generate ETag based on keys content
	keysJSON, _ := json.Marshal(keys)
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
	if os.Getenv("GAUTH_JWKS_SIGNING_KEY_ENABLED") == "1" {
		sigKey := os.Getenv("GAUTH_JWKS_SIGNING_KEY")
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

	c.JSON(200, gin.H{"keys": keys})
}
