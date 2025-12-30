package web

import (
	"github.com/mauriciomferz/Gauth_go/pkg/gauth"
	aap001handlers "github.com/mauriciomferz/Gauth_go/web/handlers/aap001"
)

// RegisterAAP001Endpoints registers all RFC-0111 subscription and authorization endpoints.
// These endpoints provide the full RFC-0111 compliant subscription flow (Steps I-VIII)
// and authorization flow (Steps a-i).
//
// NOTE: This is a basic registration that demonstrates the API structure.
// Full implementation requires mock external services and proper error handling.
func (s *BetaServer) RegisterAAP001Endpoints(
	subscriptionManager *gauth.SubscriptionFlowManager,
	subscriptionStore gauth.SubscriptionStore,
	gauthService *gauth.Service,
	tokenStore gauth.ExtendedTokenStore,
) {
	// Create handlers
	subscriptionHandlers := aap001handlers.NewSubscriptionHandlers(subscriptionManager, subscriptionStore)
	authorizationHandlers := aap001handlers.NewAuthorizationHandlers(gauthService, tokenStore)

	// Subscription Flow endpoints (Steps I-VIII)
	s.router.POST("/api/v1/aap001/subscriptions", subscriptionHandlers.CreateSubscription)
	s.router.GET("/api/v1/aap001/subscriptions/:id", subscriptionHandlers.GetSubscription)
	s.router.GET("/api/v1/aap001/subscriptions", subscriptionHandlers.ListSubscriptions)

	// Individual step execution endpoints
	s.router.POST("/api/v1/aap001/subscriptions/:id/step-ii", subscriptionHandlers.ExecuteStepII)
	s.router.POST("/api/v1/aap001/subscriptions/:id/step-iii", subscriptionHandlers.ExecuteStepIII)
	s.router.POST("/api/v1/aap001/subscriptions/:id/step-iv", subscriptionHandlers.ExecuteStepIV)
	s.router.POST("/api/v1/aap001/subscriptions/:id/step-v", subscriptionHandlers.ExecuteStepV)
	s.router.POST("/api/v1/aap001/subscriptions/:id/step-vi", subscriptionHandlers.ExecuteStepVI)
	s.router.POST("/api/v1/aap001/subscriptions/:id/step-vii", subscriptionHandlers.ExecuteStepVII)
	s.router.POST("/api/v1/aap001/subscriptions/:id/step-viii", subscriptionHandlers.ExecuteStepVIII)

	// Authorization Flow endpoints (Steps a-i)
	s.router.POST("/api/v1/aap001/authorize", authorizationHandlers.RequestToken)
	s.router.POST("/api/v1/aap001/token/validate", authorizationHandlers.ValidateToken)
	s.router.POST("/api/v1/aap001/token/introspect", authorizationHandlers.IntrospectToken)
	s.router.POST("/api/v1/aap001/token/revoke", authorizationHandlers.RevokeToken)
}
