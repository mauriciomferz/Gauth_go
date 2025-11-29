package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// BetaAuthHandler handles beta authentication endpoints for the frontend
type BetaAuthHandler struct {
	jwtSecret string
}

// NewBetaAuthHandler creates a new beta auth handler
func NewBetaAuthHandler(jwtSecret string) *BetaAuthHandler {
	return &BetaAuthHandler{
		jwtSecret: jwtSecret,
	}
}

// LoginInitRequest represents login initialization request
type LoginInitRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginInitResponse represents login initialization response
type LoginInitResponse struct {
	Success          bool     `json:"success"`
	UserID           string   `json:"userId,omitempty"`
	RequiresMFA      bool     `json:"requiresMFA"`
	MFAMethods       []string `json:"mfaMethods,omitempty"`
	SessionChallenge string   `json:"sessionChallenge,omitempty"`
	Error            string   `json:"error,omitempty"`
}

// MFAVerifyRequest represents MFA verification request
type MFAVerifyRequest struct {
	ChallengeID string `json:"challengeId" binding:"required"`
	Code        string `json:"code" binding:"required"`
	Method      string `json:"method" binding:"required"`
}

// MFAVerifyResponse represents MFA verification response
type MFAVerifyResponse struct {
	Success      bool   `json:"success"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
	TokenType    string `json:"tokenType,omitempty"`
	Error        string `json:"error,omitempty"`
}

// RefreshTokenRequest represents token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// RefreshTokenResponse represents token refresh response
type RefreshTokenResponse struct {
	Success      bool   `json:"success"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refreshToken,omitempty"`
	ExpiresAt    string `json:"expiresAt,omitempty"`
	Error        string `json:"error,omitempty"`
}

// LogoutRequest represents logout request
type LogoutRequest struct {
	Token string `json:"token" binding:"required"`
}

// LogoutResponse represents logout response
type LogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// LoginInit initiates the login process (step 1 of 2)
func (h *BetaAuthHandler) LoginInit(c *gin.Context) {
	var req LoginInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LoginInitResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// For development, accept test credentials
	// In production, this would validate against a database or identity provider
	if req.Username == "admin" && req.Password == "password" {
		c.JSON(http.StatusOK, LoginInitResponse{
			Success:          true,
			UserID:           "user-" + req.Username,
			RequiresMFA:      true,
			MFAMethods:       []string{"totp", "sms"},
			SessionChallenge: "challenge-" + req.Username + "-" + time.Now().Format("20060102150405"),
		})
		return
	}

	// Also accept admin@example.com for compatibility with admin handler
	if req.Username == "admin@example.com" && req.Password == "password" {
		c.JSON(http.StatusOK, LoginInitResponse{
			Success:          true,
			UserID:           "user-admin",
			RequiresMFA:      true,
			MFAMethods:       []string{"totp", "sms", "email"},
			SessionChallenge: "challenge-admin-" + time.Now().Format("20060102150405"),
		})
		return
	}

	c.JSON(http.StatusUnauthorized, LoginInitResponse{
		Success: false,
		Error:   "Invalid username or password",
	})
}

// MFAVerify verifies MFA code and issues JWT token (step 2 of 2)
func (h *BetaAuthHandler) MFAVerify(c *gin.Context) {
	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, MFAVerifyResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// For development, accept test code "123456"
	// In production, this would validate against TOTP/SMS/Email codes
	if req.Code == "123456" {
		// Generate JWT access token
		token, expiresAt, err := h.generateToken(req.ChallengeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, MFAVerifyResponse{
				Success: false,
				Error:   "Failed to generate token",
			})
			return
		}

		// Generate refresh token (valid for 7 days)
		refreshToken, _, err := h.generateRefreshToken(req.ChallengeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, MFAVerifyResponse{
				Success: false,
				Error:   "Failed to generate refresh token",
			})
			return
		}

		c.JSON(http.StatusOK, MFAVerifyResponse{
			Success:      true,
			Token:        token,
			RefreshToken: refreshToken,
			ExpiresAt:    expiresAt,
			TokenType:    "Bearer",
		})
		return
	}

	c.JSON(http.StatusUnauthorized, MFAVerifyResponse{
		Success: false,
		Error:   "Invalid MFA code",
	})
}

// generateToken creates a JWT access token (valid for 1 hour)
func (h *BetaAuthHandler) generateToken(challengeID string) (string, string, error) {
	expiresAt := time.Now().Add(1 * time.Hour)
	claims := jwt.MapClaims{
		"user_id":      "user-1",
		"username":     "admin",
		"role":         "admin",
		"challenge_id": challengeID,
		"token_type":   "access",
		"exp":          expiresAt.Unix(),
		"iat":          time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return "", "", err
	}

	return tokenString, expiresAt.Format(time.RFC3339), nil
}

// generateRefreshToken creates a JWT refresh token (valid for 7 days)
func (h *BetaAuthHandler) generateRefreshToken(challengeID string) (string, string, error) {
	expiresAt := time.Now().Add(7 * 24 * time.Hour)
	claims := jwt.MapClaims{
		"user_id":      "user-1",
		"username":     "admin",
		"challenge_id": challengeID,
		"token_type":   "refresh",
		"exp":          expiresAt.Unix(),
		"iat":          time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(h.jwtSecret))
	if err != nil {
		return "", "", err
	}

	return tokenString, expiresAt.Format(time.RFC3339), nil
}

// RefreshToken generates a new access token from a refresh token
func (h *BetaAuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, RefreshTokenResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// Parse and validate refresh token
	token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(h.jwtSecret), nil
	})

	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, RefreshTokenResponse{
			Success: false,
			Error:   "Invalid or expired refresh token",
		})
		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, RefreshTokenResponse{
			Success: false,
			Error:   "Invalid token claims",
		})
		return
	}

	// Verify it's a refresh token
	tokenType, ok := claims["token_type"].(string)
	if !ok || tokenType != "refresh" {
		c.JSON(http.StatusUnauthorized, RefreshTokenResponse{
			Success: false,
			Error:   "Invalid token type",
		})
		return
	}

	challengeID, _ := claims["challenge_id"].(string)

	// Generate new access token
	newToken, expiresAt, err := h.generateToken(challengeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, RefreshTokenResponse{
			Success: false,
			Error:   "Failed to generate new token",
		})
		return
	}

	// Generate new refresh token
	newRefreshToken, _, err := h.generateRefreshToken(challengeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, RefreshTokenResponse{
			Success: false,
			Error:   "Failed to generate new refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{
		Success:      true,
		Token:        newToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    expiresAt,
	})
}

// Logout invalidates a token (in production, add to token blacklist)
func (h *BetaAuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, LogoutResponse{
			Success: false,
			Error:   "Invalid request format",
		})
		return
	}

	// In production, add token to blacklist/revocation list
	// For now, just return success
	c.JSON(http.StatusOK, LogoutResponse{
		Success: true,
		Message: "Logout successful",
	})
}

// RegisterRoutes registers beta auth routes
func (h *BetaAuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/login/init", h.LoginInit)
	router.POST("/login/mfa", h.MFAVerify)
	// Alternative path for frontend compatibility
	router.POST("/mfa/verify", h.MFAVerify)
	// Token management
	router.POST("/token/refresh", h.RefreshToken)
	router.POST("/logout", h.Logout)
}
