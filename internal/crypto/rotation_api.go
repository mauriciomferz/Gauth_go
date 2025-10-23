package crypto

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// KeyRotationAPI provides HTTP endpoints for key rotation management.
type KeyRotationAPI struct {
	multiTenantManager *MultiTenantKeyManager
}

// NewKeyRotationAPI creates a new key rotation API handler.
func NewKeyRotationAPI(manager *MultiTenantKeyManager) *KeyRotationAPI {
	return &KeyRotationAPI{
		multiTenantManager: manager,
	}
}

// RegisterRoutes registers key rotation API routes with the given Gin router.
func (api *KeyRotationAPI) RegisterRoutes(router *gin.RouterGroup) {
	// Key rotation status and management endpoints
	keys := router.Group("/keys")
	{
		keys.GET("/rotation/status", api.GetRotationStatus)
		keys.GET("/rotation/status/:tenant", api.GetTenantRotationStatus)
		keys.GET("/rotation/policy", api.GetRotationPolicies)
		keys.GET("/rotation/policy/:tenant", api.GetTenantRotationPolicy)
		keys.PUT("/rotation/policy/:tenant", api.UpdateTenantRotationPolicy)
		keys.POST("/rotation/trigger/:tenant", api.TriggerRotation)
		keys.GET("/list/:tenant", api.ListTenantKeys)
		keys.POST("/activate/:tenant/:keyId", api.ActivateKey)
		keys.POST("/archive/:tenant/:keyId", api.ArchiveKey)
		keys.DELETE("/:tenant/:keyId", api.DeleteKey)
		keys.GET("/health", api.GetHealthStatus)
	}
}

// RotationStatusResponse represents the rotation status for all tenants.
type RotationStatusResponse struct {
	Tenants map[string]TenantRotationInfo `json:"tenants"`
	Summary RotationSummary              `json:"summary"`
}

// TenantRotationInfo represents rotation information for a single tenant.
type TenantRotationInfo struct {
	TenantID         string              `json:"tenant_id"`
	Policy           *RotationPolicy     `json:"policy"`
	Status           *RotationStatus     `json:"status"`
	ActiveKeyID      string              `json:"active_key_id,omitempty"`
	ActiveKeyExpires *time.Time          `json:"active_key_expires,omitempty"`
	KeyCount         int                 `json:"key_count"`
	HealthStatus     string              `json:"health_status"`
	LastRotation     *time.Time          `json:"last_rotation,omitempty"`
	NextRotation     *time.Time          `json:"next_rotation,omitempty"`
}

// RotationSummary provides aggregate rotation statistics.
type RotationSummary struct {
	TotalTenants       int `json:"total_tenants"`
	ActiveRotations    int `json:"active_rotations"`
	PendingRotations   int `json:"pending_rotations"`
	FailedRotations    int `json:"failed_rotations"`
	TenantsWithExpired int `json:"tenants_with_expired"`
}

// KeyListResponse represents a list of keys for a tenant.
type KeyListResponse struct {
	TenantID string    `json:"tenant_id"`
	Keys     []KeyInfo `json:"keys"`
}

// KeyInfo represents key information in API responses.
type KeyInfo struct {
	ID        string    `json:"id"`
	Algorithm string    `json:"algorithm"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Active    bool      `json:"active"`
	Archived  bool      `json:"archived"`
	Use       string    `json:"use"`
}

// UpdatePolicyRequest represents a request to update a tenant's rotation policy.
type UpdatePolicyRequest struct {
	Enabled       bool                   `json:"enabled"`
	Interval      string                 `json:"interval"`
	Jitter        string                 `json:"jitter"`
	MaxKeyAge     string                 `json:"max_key_age"`
	GracePeriod   string                 `json:"grace_period"`
	Backend       string                 `json:"backend"`
	BackendConfig map[string]interface{} `json:"backend_config"`
}

// TriggerRotationRequest represents a request to trigger manual rotation.
type TriggerRotationRequest struct {
	Force  bool   `json:"force"`
	Reason string `json:"reason"`
}

// GetRotationStatus returns the rotation status for all tenants.
func (api *KeyRotationAPI) GetRotationStatus(c *gin.Context) {
	tenants := api.multiTenantManager.GetRegisteredTenants()
	response := RotationStatusResponse{
		Tenants: make(map[string]TenantRotationInfo),
		Summary: RotationSummary{},
	}
	
	var activeRotations, pendingRotations, failedRotations, tenantsWithExpired int
	
	for _, tenantID := range tenants {
		info, err := api.getTenantInfo(tenantID)
		if err != nil {
			continue
		}
		
		response.Tenants[tenantID] = info
		
		// Update summary statistics
		if info.Status != nil {
			switch info.Status.State {
			case RotationStateInProgress, RotationStateGenerating:
				activeRotations++
			case RotationStatePending:
				pendingRotations++
			case RotationStateFailed:
				failedRotations++
			}
		}
		
		if info.ActiveKeyExpires != nil && time.Now().After(*info.ActiveKeyExpires) {
			tenantsWithExpired++
		}
	}
	
	response.Summary = RotationSummary{
		TotalTenants:       len(tenants),
		ActiveRotations:    activeRotations,
		PendingRotations:   pendingRotations,
		FailedRotations:    failedRotations,
		TenantsWithExpired: tenantsWithExpired,
	}
	
	c.JSON(http.StatusOK, response)
}

// GetTenantRotationStatus returns the rotation status for a specific tenant.
func (api *KeyRotationAPI) GetTenantRotationStatus(c *gin.Context) {
	tenantID := c.Param("tenant")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant ID is required"})
		return
	}
	
	info, err := api.getTenantInfo(tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("tenant not found: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, info)
}

// GetRotationPolicies returns rotation policies for all tenants.
func (api *KeyRotationAPI) GetRotationPolicies(c *gin.Context) {
	tenants := api.multiTenantManager.GetRegisteredTenants()
	policies := make(map[string]*RotationPolicy)
	
	for _, tenantID := range tenants {
		if policy := api.multiTenantManager.GetRotationPolicy(tenantID); policy != nil {
			policies[tenantID] = policy
		}
	}
	
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

// GetTenantRotationPolicy returns the rotation policy for a specific tenant.
func (api *KeyRotationAPI) GetTenantRotationPolicy(c *gin.Context) {
	tenantID := c.Param("tenant")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant ID is required"})
		return
	}
	
	policy := api.multiTenantManager.GetRotationPolicy(tenantID)
	if policy == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found or no policy configured"})
		return
	}
	
	c.JSON(http.StatusOK, policy)
}

// UpdateTenantRotationPolicy updates the rotation policy for a specific tenant.
func (api *KeyRotationAPI) UpdateTenantRotationPolicy(c *gin.Context) {
	tenantID := c.Param("tenant")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant ID is required"})
		return
	}
	
	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request body: %v", err)})
		return
	}
	
	// Convert request to RotationPolicy
	policy, err := api.requestToPolicy(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid policy: %v", err)})
		return
	}
	
	// Update policy
	if err := api.multiTenantManager.UpdateRotationPolicy(tenantID, policy); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to update policy: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "policy updated successfully", "policy": policy})
}

// TriggerRotation manually triggers key rotation for a tenant.
func (api *KeyRotationAPI) TriggerRotation(c *gin.Context) {
	tenantID := c.Param("tenant")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant ID is required"})
		return
	}
	
	var req TriggerRotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body for simple trigger
		req = TriggerRotationRequest{Force: false}
	}
	
	// Trigger rotation
	if err := api.multiTenantManager.TriggerRotation(tenantID, req.Force, req.Reason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to trigger rotation: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "rotation triggered successfully",
		"tenant_id": tenantID,
		"force":     req.Force,
		"reason":    req.Reason,
	})
}

// ListTenantKeys returns all keys for a specific tenant.
func (api *KeyRotationAPI) ListTenantKeys(c *gin.Context) {
	tenantID := c.Param("tenant")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant ID is required"})
		return
	}
	
	keys, err := api.multiTenantManager.keyStore.ListKeys(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to list keys: %v", err)})
		return
	}
	
	keyInfos := make([]KeyInfo, len(keys))
	activeKeyID := ""
	
	// Get active key ID
	if activeKey, err := api.multiTenantManager.keyStore.GetActive(c.Request.Context(), tenantID); err == nil && activeKey != nil {
		activeKeyID = activeKey.ID
	}
	
	for i, key := range keys {
		keyInfos[i] = KeyInfo{
			ID:        key.ID,
			Algorithm: key.Alg,
			CreatedAt: key.CreatedAt,
			ExpiresAt: key.ExpiresAt,
			Active:    key.ID == activeKeyID,
			Use:       key.Use,
			// Note: We don't have direct archived status in Key struct
			// This would need to be enhanced based on your KeyStore implementation
			Archived: false,
		}
	}
	
	response := KeyListResponse{
		TenantID: tenantID,
		Keys:     keyInfos,
	}
	
	c.JSON(http.StatusOK, response)
}

// ActivateKey activates a specific key for a tenant.
func (api *KeyRotationAPI) ActivateKey(c *gin.Context) {
	tenantID := c.Param("tenant")
	keyID := c.Param("keyId")
	
	if tenantID == "" || keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant ID and key ID are required"})
		return
	}
	
	if err := api.multiTenantManager.keyStore.Activate(c.Request.Context(), tenantID, keyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to activate key: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "key activated successfully",
		"tenant_id": tenantID,
		"key_id":    keyID,
	})
}

// ArchiveKey archives a specific key for a tenant.
func (api *KeyRotationAPI) ArchiveKey(c *gin.Context) {
	tenantID := c.Param("tenant")
	keyID := c.Param("keyId")
	
	if tenantID == "" || keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant ID and key ID are required"})
		return
	}
	
	if err := api.multiTenantManager.keyStore.Archive(c.Request.Context(), tenantID, keyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to archive key: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "key archived successfully",
		"tenant_id": tenantID,
		"key_id":    keyID,
	})
}

// DeleteKey deletes a specific key for a tenant.
func (api *KeyRotationAPI) DeleteKey(c *gin.Context) {
	tenantID := c.Param("tenant")
	keyID := c.Param("keyId")
	
	if tenantID == "" || keyID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant ID and key ID are required"})
		return
	}
	
	if err := api.multiTenantManager.keyStore.Delete(c.Request.Context(), tenantID, keyID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to delete key: %v", err)})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "key deleted successfully",
		"tenant_id": tenantID,
		"key_id":    keyID,
	})
}

// GetHealthStatus returns the health status of the key rotation system.
func (api *KeyRotationAPI) GetHealthStatus(c *gin.Context) {
	ctx := c.Request.Context()
	
	health := gin.H{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"checks":    gin.H{},
	}
	
	// Check key store health
	if err := api.multiTenantManager.keyStore.Health(ctx); err != nil {
		health["status"] = "unhealthy"
		health["checks"].(gin.H)["keystore"] = gin.H{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	} else {
		health["checks"].(gin.H)["keystore"] = gin.H{
			"status": "healthy",
		}
	}
	
	// Check scheduler health
	schedulerHealth := api.multiTenantManager.IsHealthy()
	health["checks"].(gin.H)["scheduler"] = gin.H{
		"status": map[bool]string{true: "healthy", false: "unhealthy"}[schedulerHealth],
	}
	
	if !schedulerHealth {
		health["status"] = "unhealthy"
	}
	
	// Return appropriate status code
	statusCode := http.StatusOK
	if health["status"] == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, health)
}

// Helper methods

// getTenantInfo retrieves comprehensive information about a tenant's rotation status.
func (api *KeyRotationAPI) getTenantInfo(tenantID string) (TenantRotationInfo, error) {
	info := TenantRotationInfo{
		TenantID:     tenantID,
		HealthStatus: "unknown",
	}
	
	// Get policy
	info.Policy = api.multiTenantManager.GetRotationPolicy(tenantID)
	
	// Get status
	info.Status = api.multiTenantManager.GetRotationStatus(tenantID)
	
	// Get active key information
	if activeKey, err := api.multiTenantManager.keyStore.GetActive(nil, tenantID); err == nil && activeKey != nil {
		info.ActiveKeyID = activeKey.ID
		info.ActiveKeyExpires = &activeKey.ExpiresAt
	}
	
	// Get key count
	if keys, err := api.multiTenantManager.keyStore.ListKeys(nil, tenantID); err == nil {
		info.KeyCount = len(keys)
	}
	
	// Get rotation timing information
	if info.Status != nil {
		info.LastRotation = info.Status.LastRotation
		info.NextRotation = info.Status.NextRotation
	}
	
	// Determine health status
	ctx := context.Background() // Use background context for health checks
	if err := api.multiTenantManager.keyStore.Health(ctx); err == nil {
		if info.ActiveKeyExpires == nil || time.Now().Before(*info.ActiveKeyExpires) {
			info.HealthStatus = "healthy"
		} else {
			info.HealthStatus = "expired"
		}
	} else {
		info.HealthStatus = "unhealthy"
	}
	
	return info, nil
}

// requestToPolicy converts an UpdatePolicyRequest to a RotationPolicy.
func (api *KeyRotationAPI) requestToPolicy(req UpdatePolicyRequest) (*RotationPolicy, error) {
	policy := &RotationPolicy{
		Enabled:       req.Enabled,
		Backend:       req.Backend,
		BackendConfig: req.BackendConfig,
	}
	
	var err error
	
	// Parse durations
	if req.Interval != "" {
		policy.Interval, err = time.ParseDuration(req.Interval)
		if err != nil {
			return nil, fmt.Errorf("invalid interval: %v", err)
		}
	}
	
	if req.Jitter != "" {
		policy.Jitter, err = time.ParseDuration(req.Jitter)
		if err != nil {
			return nil, fmt.Errorf("invalid jitter: %v", err)
		}
	}
	
	if req.MaxKeyAge != "" {
		policy.MaxKeyAge, err = time.ParseDuration(req.MaxKeyAge)
		if err != nil {
			return nil, fmt.Errorf("invalid max_key_age: %v", err)
		}
	}
	
	if req.GracePeriod != "" {
		policy.GracePeriod, err = time.ParseDuration(req.GracePeriod)
		if err != nil {
			return nil, fmt.Errorf("invalid grace_period: %v", err)
		}
	}
	
	// Validate policy
	if policy.Interval <= 0 {
		return nil, fmt.Errorf("interval must be positive")
	}
	
	if policy.Backend == "" {
		policy.Backend = "file" // Default backend
	}
	
	return policy, nil
}