// Package device provides HTTP handlers for OAuth 2.0 Device Authorization Grant (RFC 8628).
package device

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mauriciomferz/Gauth_go/pkg/device"
)

// Handler provides HTTP handlers for device authorization.
type Handler struct {
	store           device.DeviceCodeStore
	verificationURI string
	expiresIn       int // Default expiration in seconds
	interval        int // Default polling interval in seconds
}

// NewHandler creates a new device authorization handler.
func NewHandler(store device.DeviceCodeStore) *Handler {
	baseURL := os.Getenv("GAUTH_DEVICE_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	return &Handler{
		store:           store,
		verificationURI: baseURL + "/device/verify",
		expiresIn:       600, // 10 minutes
		interval:        5,   // 5 seconds
	}
}

// RegisterRoutes registers device authorization routes.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/device/authorize", h.DeviceAuthorize)
	r.GET("/device/verify", h.VerifyPage)
	r.POST("/device/verify", h.VerifyCode)
	r.POST("/device/token", h.TokenPoll)
}

// DeviceAuthorize handles the device authorization request (RFC 8628 §3.1-3.2).
func (h *Handler) DeviceAuthorize(c *gin.Context) {
	var req device.DeviceAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            "invalid_request",
			ErrorDescription: "Invalid request body",
		})
		return
	}

	if req.ClientID == "" {
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            "invalid_request",
			ErrorDescription: "client_id is required",
		})
		return
	}

	dc, err := h.store.Create(&req, h.expiresIn, h.interval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, device.ErrorResponse{
			Error:            "server_error",
			ErrorDescription: err.Error(),
		})
		return
	}

	resp := device.DeviceAuthResponse{
		DeviceCode:              dc.DeviceCode,
		UserCode:                dc.UserCode,
		VerificationURI:         h.verificationURI,
		VerificationURIComplete: h.verificationURI + "?code=" + dc.UserCode,
		ExpiresIn:               h.expiresIn,
		Interval:                h.interval,
	}

	c.JSON(http.StatusOK, resp)
}

// VerifyPage serves the user code verification page (GET).
func (h *Handler) VerifyPage(c *gin.Context) {
	code := c.Query("code")
	c.HTML(http.StatusOK, "device_verify.html", gin.H{
		"code": code,
	})
}

// VerifyCodeRequest is the user code verification request.
type VerifyCodeRequest struct {
	UserCode string `json:"user_code" form:"user_code"`
	Action   string `json:"action" form:"action"` // "authorize" or "deny"
	UserID   string `json:"user_id" form:"user_id"`
}

// VerifyCode handles user code verification (POST).
func (h *Handler) VerifyCode(c *gin.Context) {
	var req VerifyCodeRequest
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            "invalid_request",
			ErrorDescription: "Invalid request",
		})
		return
	}

	if req.UserCode == "" {
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            "invalid_request",
			ErrorDescription: "user_code is required",
		})
		return
	}

	dc, err := h.store.GetByUserCode(req.UserCode)
	if err != nil {
		c.JSON(http.StatusNotFound, device.ErrorResponse{
			Error:            "invalid_grant",
			ErrorDescription: "User code not found",
		})
		return
	}

	if dc.IsExpired() {
		c.JSON(http.StatusGone, device.ErrorResponse{
			Error:            device.ErrorExpiredToken,
			ErrorDescription: "Device code has expired",
		})
		return
	}

	if req.Action == "deny" {
		_ = h.store.Deny(req.UserCode)
		c.JSON(http.StatusOK, gin.H{
			"status":  "denied",
			"message": "Authorization denied",
		})
		return
	}

	// Default: authorize
	userID := req.UserID
	if userID == "" {
		userID = "anonymous"
	}
	if err := h.store.Authorize(req.UserCode, userID); err != nil {
		c.JSON(http.StatusInternalServerError, device.ErrorResponse{
			Error:            "server_error",
			ErrorDescription: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "authorized",
		"message": "Device authorized successfully",
	})
}

// TokenPoll handles the token polling request (RFC 8628 §3.4-3.5).
func (h *Handler) TokenPoll(c *gin.Context) {
	var req device.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            "invalid_request",
			ErrorDescription: "Invalid request body",
		})
		return
	}

	if req.GrantType != device.DeviceFlowGrantType {
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            "unsupported_grant_type",
			ErrorDescription: "grant_type must be " + device.DeviceFlowGrantType,
		})
		return
	}

	dc, err := h.store.GetByDeviceCode(req.DeviceCode)
	if err != nil {
		c.JSON(http.StatusNotFound, device.ErrorResponse{
			Error:            "invalid_grant",
			ErrorDescription: "Device code not found",
		})
		return
	}

	// Check expiration
	if dc.IsExpired() {
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            device.ErrorExpiredToken,
			ErrorDescription: "Device code has expired",
		})
		return
	}

	// Check polling rate
	if !dc.CanPoll() {
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            device.ErrorSlowDown,
			ErrorDescription: "Polling too frequently",
		})
		return
	}
	_ = h.store.UpdateLastPoll(req.DeviceCode)

	// Check status
	switch dc.Status {
	case device.StatusPending:
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            device.ErrorAuthorizationPending,
			ErrorDescription: "User has not yet authorized",
		})
		return

	case device.StatusDenied:
		c.JSON(http.StatusBadRequest, device.ErrorResponse{
			Error:            device.ErrorAccessDenied,
			ErrorDescription: "User denied the request",
		})
		return

	case device.StatusAuthorized:
		// Issue token (simplified - in production integrate with token service)
		token := device.TokenResponse{
			AccessToken: "device_" + dc.DeviceCode[:16],
			TokenType:   "Bearer",
			ExpiresIn:   3600,
			Scope:       dc.Scope,
		}
		c.JSON(http.StatusOK, token)
		return

	default:
		c.JSON(http.StatusInternalServerError, device.ErrorResponse{
			Error:            "server_error",
			ErrorDescription: "Unknown status",
		})
	}
}
