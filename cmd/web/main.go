package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/Gimel-Foundation/gauth/pkg/gauth"
	"github.com/Gimel-Foundation/gauth/pkg/common"
)

func main() {
	fmt.Println("GAuth Web Server")
	fmt.Println("================")

	// Initialize GAuth service
	service, err := gauth.NewService(gauth.Config{
		AuthServerURL:     "https://gauth.gimelfoundation.com",
		ClientID:          "demo-web-client",
		ClientSecret:      "demo-web-secret",
		Scopes:            []string{"read", "write"},
		AccessTokenExpiry: time.Hour,
		RateLimit: common.RateLimitConfig{
			RequestsPerSecond: 100,
			BurstSize:         10,
			WindowSize:        60,
		},
	})
	if err != nil {
		log.Fatalf("Failed to initialize GAuth service: %v", err)
	}

	// Create Gin router
	router := gin.Default()
	
	// Add basic CORS
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	})

	// Add routes
	setupRoutes(router, service)

	// Start server
	port := 8080
	fmt.Printf("Starting GAuth Web Server on port %d...\n", port)
	
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Fatal(server.ListenAndServe())
}

func setupRoutes(router *gin.Engine, service *gauth.Service) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now(),
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Token endpoints
		api.POST("/tokens", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Token creation endpoint",
				"service": "gauth",
			})
		})
		
		api.GET("/tokens", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"tokens": []gin.H{},
				"total":  0,
			})
		})

		// RFC111 endpoints
		rfc111 := api.Group("/rfc111")
		{
			rfc111.POST("/authorize", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{
					"code":        "rfc111_auth_" + fmt.Sprintf("%d", time.Now().UnixNano()),
					"status":      "success",
					"compliance":  "rfc111_compliant",
				})
			})
		}

		// RFC115 endpoints  
		rfc115 := api.Group("/rfc115")
		{
			rfc115.POST("/delegation", func(c *gin.Context) {
				c.JSON(http.StatusCreated, gin.H{
					"delegation_id": "delegation_" + fmt.Sprintf("%d", time.Now().UnixNano()),
					"status":        "active",
					"created_at":    time.Now(),
				})
			})
		}
	}

	// Serve static files (if needed)
	router.Static("/static", "./gauth-demo-app/web/frontend/build/static")
	router.StaticFile("/", "./gauth-demo-app/web/frontend/build/index.html")
}
