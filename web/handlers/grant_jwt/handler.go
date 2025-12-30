package grant_jwt

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mauriciomferz/AgentAuth/pkg/auth"
)

// #nosec G101: false positive - these are OIDC/OAuth URN strings, not credentials
const (
	GrantTypeJWTBearer         = "urn:ietf:params:oauth:grant-type:jwt-bearer"
	GrantTypeIdentityAssertion = "urn:ietf:params:oauth:grant-type:identity-assertion"
)

type Handler struct {
	authenticator auth.ClientAuthenticator
}

func NewHandler(authenticator auth.ClientAuthenticator) *Handler {
	return &Handler{
		authenticator: authenticator,
	}
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

	// 2. Authenticate / Validate Assertion
	// We reuse the ClientAuthenticator which enforces standard RFC 7523 checks
	// Note: We intentionally pass ClientAssertionTypeJWT as the type since the structure is identical
	if err := h.authenticator.Authenticate(iss, assertion, auth.ClientAssertionTypeJWT); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_grant", "error_description": err.Error()})
		return
	}

	// 3. Issue Token
	// In a real system, we'd generate a real access token bound to 'iss' (Subject)
	// For this MVP, we return a mocked success response
	resp := TokenResponse{
		AccessToken: "agentauth_access_" + iss + "_" + strings.Split(assertion, ".")[2][:10],
		TokenType:   "Bearer",
		ExpiresIn:   3600,
	}

	c.JSON(http.StatusOK, resp)
}
