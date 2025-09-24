package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/handlers"
	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/middleware"
	"github.com/Gimel-Foundation/gauth/gauth-demo-app/web/backend/services"
)

// @title GAuth Demo API
// @version 1.0
// @description Comprehensive demonstration of GAuth protocol capabilities
// @termsOfService https://gimelfoundation.com/terms

// @contact.name GAuth Support
// @contact.url https://gimelfoundation.com/support
// @contact.email support@gimelfoundation.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

func main() {
	// Initialize configuration
	config := initConfig()

	// Initialize logger
	logger := initLogger(config)

	// Initialize services
	svc, err := services.NewGAuthService(config, logger)
	if err != nil {
		logger.Fatalf("Failed to initialize GAuth service: %v", err)
	}

	// Initialize HTTP server
	router := setupRouter(svc, logger, config)

	// Start server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.GetInt("server.port")),
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Infof("GAuth Demo Server started on port %d", config.GetInt("server.port"))

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func initConfig() *viper.Viper {
	config := viper.New()

	// Set defaults
	config.SetDefault("server.port", 8080)
	config.SetDefault("server.mode", "debug")
	config.SetDefault("log.level", "info")
	config.SetDefault("redis.addr", "localhost:6379")
	config.SetDefault("redis.password", "")
	config.SetDefault("redis.db", 0)

	// Read configuration from file
	config.SetConfigName("config")
	config.SetConfigType("yaml")
	config.AddConfigPath(".")
	config.AddConfigPath("./config")

	// Read environment variables
	config.AutomaticEnv()

	if err := config.ReadInConfig(); err != nil {
		log.Printf("Warning: Could not read config file: %v", err)
	}

	return config
}

func initLogger(config *viper.Viper) *logrus.Logger {
	logger := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(config.GetString("log.level"))
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	return logger
}

func setupRouter(svc *services.GAuthService, logger *logrus.Logger, config *viper.Viper) *gin.Engine {
	// Set gin mode
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Middleware
	router.Use(gin.Recovery())
	router.Use(middleware.Logger(logger))
	router.Use(middleware.RequestID())

	// CORS configuration
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:3000", "http://127.0.0.1:3000"}
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	router.Use(cors.New(corsConfig))

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Authentication endpoints
		auth := api.Group("/auth")
		authHandler := handlers.NewAuthHandler(svc, logger)
		{
			auth.POST("/authorize", authHandler.Authorize)
			auth.POST("/token", authHandler.Token)
			auth.POST("/revoke", authHandler.Revoke)
			auth.GET("/userinfo", authHandler.UserInfo)
			auth.POST("/validate", authHandler.Validate)
		}

		// Legal Framework endpoints
		legal := api.Group("/legal")
		legalHandler := handlers.NewLegalFrameworkHandler(svc, logger)
		{
			legal.POST("/entities", legalHandler.CreateEntity)
			legal.GET("/entities/:id", legalHandler.GetEntity)
			legal.POST("/entities/:id/verify", legalHandler.VerifyLegalCapacity)

			legal.POST("/power-of-attorney", legalHandler.CreatePowerOfAttorney)
			legal.GET("/power-of-attorney/:id", legalHandler.GetPowerOfAttorney)
			legal.POST("/power-of-attorney/:id/delegate", legalHandler.DelegatePower)

			legal.POST("/requests", legalHandler.CreateRequest)
			legal.GET("/requests/:id", legalHandler.GetRequest)
			legal.POST("/requests/:id/approve", legalHandler.ApproveRequest)

			legal.GET("/jurisdictions", legalHandler.GetJurisdictions)
			legal.GET("/jurisdictions/:id/rules", legalHandler.GetJurisdictionRules)
		}

		// Audit endpoints
		audit := api.Group("/audit")
		auditHandler := handlers.NewAuditHandler(svc, logger)
		{
			audit.GET("/events", auditHandler.GetEvents)
			audit.GET("/events/:id", auditHandler.GetEvent)
			audit.GET("/compliance", auditHandler.GetComplianceReport)
			audit.GET("/trails/:entity", auditHandler.GetAuditTrail)
		}

		// Rate limiting endpoints
		rate := api.Group("/rate")
		rateHandler := handlers.NewRateHandler(svc, logger)
		{
			rate.GET("/limits", rateHandler.GetLimits)
			rate.POST("/limits", rateHandler.SetLimits)
			rate.GET("/status/:client", rateHandler.GetStatus)
		}

		// Demo scenarios endpoints
		demo := api.Group("/demo")
		demoHandler := handlers.NewDemoHandler(svc, logger)
		{
			demo.GET("/scenarios", demoHandler.GetScenarios)
			demo.POST("/scenarios/:id/run", demoHandler.RunScenario)
			demo.GET("/scenarios/:id/status", demoHandler.GetScenarioStatus)
		}

		// RFC111/RFC115 Compliance endpoints - Full RFC implementation
		rfc111Handler, err := handlers.NewRFC111Handler(config, logger)
		if err != nil {
			logger.Fatalf("Failed to initialize RFC111 handler: %v", err)
		}
		rfc111Handler.RegisterRoutes(router)

		// Token Management endpoints
		tokens := api.Group("/tokens")
		tokenHandler := handlers.NewTokenHandler(svc, logger)
		{
			tokens.POST("", tokenHandler.CreateToken)
			tokens.GET("", tokenHandler.GetTokens)
			tokens.DELETE("/:id", tokenHandler.RevokeToken)
			tokens.POST("/validate", tokenHandler.ValidateToken)
			tokens.POST("/refresh", tokenHandler.RefreshToken)
		}

		// Metrics endpoints
		metrics := api.Group("/metrics")
		{
			metrics.GET("/tokens", tokenHandler.GetTokenMetrics)
			metrics.GET("/system", func(c *gin.Context) {
				// Return basic system metrics
				c.JSON(200, gin.H{
					"active_users":          42,
					"total_transactions":    1234,
					"success_rate":          0.98,
					"average_response_time": 120,
					"last_updated":          time.Now(),
				})
			})
		}
	}

	// WebSocket endpoints for real-time updates
	wsHandler := handlers.NewWebSocketHandler(svc, logger)
	router.GET("/ws", wsHandler.HandleEvents)
	router.GET("/ws/events", wsHandler.HandleEvents)

	// Swagger documentation
	// router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Serve static files (React app)
	router.Static("/static", "./static")
	router.StaticFile("/", "./static/index.html")
	router.NoRoute(func(c *gin.Context) {
		c.File("./static/index.html")
	})

	return router
}
