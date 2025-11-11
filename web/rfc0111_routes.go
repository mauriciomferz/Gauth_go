package web

import (
	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/gauth"
	rfc0111handlers "github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/web/handlers/rfc0111"
)

// RegisterRFC0111Endpoints registers all RFC-0111 subscription and authorization endpoints.
// These endpoints provide the full RFC-0111 compliant subscription flow (Steps I-VIII)
// and authorization flow (Steps a-i).
//
// NOTE: This is a basic registration that demonstrates the API structure.
// Full implementation requires mock external services and proper error handling.
func (s *BetaServer) RegisterRFC0111Endpoints(
	subscriptionManager *gauth.SubscriptionFlowManager,
	subscriptionStore gauth.SubscriptionStore,
	gauthService *gauth.Service,
) {
	// Create handlers
	subscriptionHandlers := rfc0111handlers.NewSubscriptionHandlers(subscriptionManager, subscriptionStore)
	authorizationHandlers := rfc0111handlers.NewAuthorizationHandlers(gauthService)

	// Subscription Flow endpoints (Steps I-VIII)
	s.router.POST("/api/v1/rfc0111/subscriptions", subscriptionHandlers.CreateSubscription)
	s.router.GET("/api/v1/rfc0111/subscriptions/:id", subscriptionHandlers.GetSubscription)
	s.router.GET("/api/v1/rfc0111/subscriptions", subscriptionHandlers.ListSubscriptions)

	// Individual step execution endpoints
	s.router.POST("/api/v1/rfc0111/subscriptions/:id/step-ii", subscriptionHandlers.ExecuteStepII)
	s.router.POST("/api/v1/rfc0111/subscriptions/:id/step-iii", subscriptionHandlers.ExecuteStepIII)
	s.router.POST("/api/v1/rfc0111/subscriptions/:id/step-iv", subscriptionHandlers.ExecuteStepIV)
	s.router.POST("/api/v1/rfc0111/subscriptions/:id/step-v", subscriptionHandlers.ExecuteStepV)
	s.router.POST("/api/v1/rfc0111/subscriptions/:id/step-vi", subscriptionHandlers.ExecuteStepVI)
	s.router.POST("/api/v1/rfc0111/subscriptions/:id/step-vii", subscriptionHandlers.ExecuteStepVII)
	s.router.POST("/api/v1/rfc0111/subscriptions/:id/step-viii", subscriptionHandlers.ExecuteStepVIII)

	// Authorization Flow endpoints (Steps a-i)
	s.router.POST("/api/v1/rfc0111/authorize", authorizationHandlers.RequestToken)
	s.router.POST("/api/v1/rfc0111/token/validate", authorizationHandlers.ValidateToken)
	s.router.POST("/api/v1/rfc0111/token/introspect", authorizationHandlers.IntrospectToken)
	s.router.POST("/api/v1/rfc0111/token/revoke", authorizationHandlers.RevokeToken)
}
