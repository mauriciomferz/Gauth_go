package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/mauriciomferz/AgentAuth/pkg/crypto"
)

// Example integration of the multi-tenant key rotation system
func main() {
	// Create a file-based key store for development
	fileStore, err := crypto.NewFileKeyStore("/tmp/gauth-keys", 24*time.Hour)
	if err != nil {
		log.Fatalf("Failed to create file key store: %v", err)
	}

	// Create default rotation policy
	defaultPolicy := &crypto.RotationPolicy{
		Enabled:     true,
		Interval:    24 * time.Hour,     // Rotate daily
		Jitter:      time.Hour,          // Random 1-hour jitter
		MaxKeyAge:   7 * 24 * time.Hour, // Keys expire after 1 week
		GracePeriod: 24 * time.Hour,     // 1-day grace period for old keys
		Backend:     "file",
	}

	// Create multi-tenant key manager
	manager := crypto.NewMultiTenantKeyManager(fileStore, defaultPolicy)

	// Register some example tenants
	tenants := []string{"tenant-a", "tenant-b", "tenant-c"}
	for _, tenant := range tenants {
		// Each tenant gets the default policy initially
		if err := manager.RegisterTenant(tenant, fileStore, defaultPolicy); err != nil {
			log.Printf("Failed to register tenant %s: %v", tenant, err)
		}

		// Generate initial key for each tenant
		ctx := context.Background()
		keyID, err := fileStore.Generate(ctx, tenant)
		if err != nil {
			log.Printf("Failed to generate initial key for tenant %s: %v", tenant, err)
			continue
		}

		// Activate the initial key
		if err := fileStore.Activate(ctx, tenant, keyID); err != nil {
			log.Printf("Failed to activate initial key for tenant %s: %v", tenant, err)
		} else {
			log.Printf("Generated and activated initial key %s for tenant %s", keyID, tenant)
		}
	}

	// Create the key rotation API
	rotationAPI := crypto.NewKeyRotationAPI(manager)

	// Set up Gin router
	router := gin.Default()

	// Add middleware for logging
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Register API routes
	v1 := router.Group("/api/v1")
	rotationAPI.RegisterRoutes(v1)

	// Add a simple welcome endpoint
	router.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "AgentAuth Multi-Tenant Key Rotation System",
			"version": "1.0.0",
			"endpoints": gin.H{
				"rotation_status":      "GET /api/v1/keys/rotation/status",
				"tenant_status":        "GET /api/v1/keys/rotation/status/:tenant",
				"rotation_policies":    "GET /api/v1/keys/rotation/policy",
				"tenant_policy":        "GET /api/v1/keys/rotation/policy/:tenant",
				"update_tenant_policy": "PUT /api/v1/keys/rotation/policy/:tenant",
				"trigger_rotation":     "POST /api/v1/keys/rotation/trigger/:tenant",
				"list_tenant_keys":     "GET /api/v1/keys/list/:tenant",
				"activate_key":         "POST /api/v1/keys/activate/:tenant/:keyId",
				"archive_key":          "POST /api/v1/keys/archive/:tenant/:keyId",
				"delete_key":           "DELETE /api/v1/keys/:tenant/:keyId",
				"health":               "GET /api/v1/keys/health",
			},
			"registered_tenants": manager.GetRegisteredTenants(),
		})
	})

	// Add demonstration endpoint to show rotation in action
	router.POST("/demo/trigger-all-rotations", func(c *gin.Context) {
		results := make(map[string]interface{})

		for _, tenant := range manager.GetRegisteredTenants() {
			if err := manager.TriggerRotation(tenant, false, "demo trigger"); err != nil {
				results[tenant] = gin.H{"status": "error", "message": err.Error()}
			} else {
				results[tenant] = gin.H{"status": "success", "message": "rotation triggered"}
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Triggered rotation for all tenants",
			"results": results,
		})
	})

	// Add endpoint to demonstrate different key store backends
	router.GET("/demo/backends", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Available key store backends",
			"backends": gin.H{
				"file": gin.H{
					"description": "File-based key storage for development",
					"location":    "/tmp/gauth-keys",
					"security":    "File system permissions (600)",
				},
				"vault": gin.H{
					"description": "HashiCorp Vault integration",
					"features":    []string{"HSM support", "audit logging", "high availability"},
					"status":      "implemented",
				},
				"kms": gin.H{
					"description": "AWS KMS integration",
					"features":    []string{"cloud-native", "envelope encryption", "compliance"},
					"status":      "implemented",
				},
			},
			"current_backend": "file",
		})
	})

	// Start the server
	port := ":8080"
	log.Printf("Starting AgentAuth Key Rotation API server on %s", port)
	log.Printf("Access the API at: http://localhost%s", port)
	log.Printf("View status: http://localhost%s/api/v1/keys/rotation/status", port)
	log.Printf("Health check: http://localhost%s/api/v1/keys/health", port)

	if err := router.Run(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
