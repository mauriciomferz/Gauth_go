package grant_jwt

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/AgentAuth/pkg/auth"
)

// #nosec G101: false positive - these are OIDC/OAuth URN strings, not credentials
const (
	GrantTypeJWTBearer         = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	GrantTypeIdentityAssertion = "urn:ietf:params:oauth:grant-type:identity-assertion"
)

type TokenSigner interface {
	Sign(claims jwt.MapClaims) (string, error)
}

type Handler struct {
	authenticator  auth.ClientAuthenticator
	signer         TokenSigner
	issuerRegistry *auth.IssuerRegistry
}

func NewHandler(authenticator auth.ClientAuthenticator) *Handler {
	return &Handler{
		authenticator:  authenticator,
		issuerRegistry: auth.GlobalRegistry,
	}
}

func (h *Handler) SetSigner(signer TokenSigner) {
	h.signer = signer
}

// SetIssuerRegistry allows injecting a custom registry (useful for tests)
func (h *Handler) SetIssuerRegistry(registry *auth.IssuerRegistry) {
	h.issuerRegistry = registry
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/oauth/token", h.HandleTokenRequest)
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

func (h *Handler) HandleTokenRequest(c *gin.Context) {
	// Parse Form Data (application/x-www-form-urlencoded)
	grantType := c.PostForm("grant_type")
	assertion := c.PostForm("assertion")

	if grantType != GrantTypeJWTBearer && grantType != GrantTypeIdentityAssertion {
		// Pass through to next handler or return error if this is the only one
		// For now, we assume this endpoint might handle multiple, but here we error
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported_grant_type"})
		return
	}

	if assertion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "error_description": "assertion is required"})
		return
	}

	// 1. Peek at the JWT to get 'iss' (to know which key to use)
	// We can cheat and use the auth package if we refactor, but here we do a quick parse
	token, _, err := new(jwt.Parser).ParseUnverified(assertion, jwt.MapClaims{})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "malformed assertion"})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "invalid claims"})
		return
	}

	iss, _ := claims.GetIssuer()
	if iss == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_grant", "error_description": "issuer missing"})
		return
	}

	// 2. Validate trusting the issuer (Phase 22)
	// If the issuer is in our trusted registry, we use its configuration
	var trustedIssuer *auth.TrustedIssuer
	if h.issuerRegistry != nil {
		if ti, err := h.issuerRegistry.Get(iss); err == nil {
			trustedIssuer = ti
		}
	}

	// 3. Authenticate / Validate Assertion
	if trustedIssuer != nil {
		// Validating against a Trusted Issuer (e.g. Entra ID) using Dynamic JWKS
		parser := jwt.NewParser(jwt.WithValidMethods([]string{"RS256", "ES256"})) // Enforce algorithms?
		// We re-parse to verify signature
		parsedToken, err := parser.Parse(assertion, func(t *jwt.Token) (interface{}, error) {
			// Extract kid
			kid, ok := t.Header["kid"].(string)
			if !ok {
				return nil, jwt.ErrTokenMalformed
			}
			// Fetch key dynamically
			return trustedIssuer.GetKey(c.Request.Context(), kid)
		})

		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":             "invalid_grant",
				"error_description": "assertion validation failed: " + err.Error(),
			})
			return
		}

		if !parsedToken.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant", "error_description": "invalid assertion"})
			return
		}

		// Audience Check
		if trustedIssuer.Audience != "" {
			aud, _ := parsedToken.Claims.GetAudience()
			found := false
			for _, a := range aud {
				if a == trustedIssuer.Audience {
					found = true
					break
				}
			}
			if !found {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant", "error_description": "audience mismatch for trusted issuer"})
				return
			}
		}
	} else {
		// Fallback to Client Authentication (Static Keys)
		// Note: We intentionally pass ClientAssertionTypeJWT as the type since the structure is identical
		if err := h.authenticator.Authenticate(iss, assertion, auth.ClientAssertionTypeJWT); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant", "error_description": err.Error()})
			return
		}
	}

	// 4. Issue Token
	// If a signer is configured, we generate a real signed JWT.
	// Otherwise we fall back to the mock for backward compatibility during migration.
	var accessToken string
	if h.signer != nil {
		now := time.Now()

		// Map Subject if Trusted Issuer has mapping
		sub, _ := claims.GetSubject()
		if trustedIssuer != nil && trustedIssuer.ClaimsMapping != nil {
			// Example: Map "oid" to "sub" if tailored
			if target, ok := trustedIssuer.ClaimsMapping["oid"]; ok && target == "sub" {
				if oid, ok := claims["oid"].(string); ok {
					sub = oid
				}
			}
		}

		oboClaims := jwt.MapClaims{
			"iss": "agentauth-issuer",
			"sub": sub, // The agent (subject of the assertion) becomes the subject of the new token
			"aud": "https://resource.example.com",
			"iat": now.Unix(),
			"exp": now.Add(1 * time.Hour).Unix(),
			"act": map[string]interface{}{
				"sub": "agentauth-gateway", // Acting on behalf of
			},
			"scope": "User.Read", // Mock scope from OBO logic
		}

		// Copy over mapped claims if any
		if trustedIssuer != nil {
			for extKey, intKey := range trustedIssuer.ClaimsMapping {
				if val, ok := claims[extKey]; ok {
					oboClaims[intKey] = val
				}
			}
		}

		var err error
		accessToken, err = h.signer.Sign(oboClaims)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server_error", "error_description": "failed to sign token"})
			return
		}
	} else {
		accessToken = "agentauth_access_" + iss + "_" + strings.Split(assertion, ".")[2][:10]
	}

	resp := TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}

	c.JSON(http.StatusOK, resp)
}
