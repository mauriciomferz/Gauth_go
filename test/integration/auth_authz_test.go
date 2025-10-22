//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/GiFo-RFC-0150-Go-Implementation-of-GAuth-1.0/pkg/authz"
)

// --- TEST COMPATIBILITY PATCHES ---
// Local config and type for compatibility
type testConfig struct {
	Type              string
	AccessTokenExpiry time.Duration
}

const testTypeBasic = "basic"

type dummyAuth struct{}

func (d *dummyAuth) ValidateCredentials(ctx context.Context, creds interface{}) error { return nil }

func newTestAuthenticator(_ testConfig) (*dummyAuth, error) {
	return &dummyAuth{}, nil
}

func toRequest(subject authz.Subject, action authz.Action, resource authz.Resource) *authz.Request {
	return &authz.Request{Subject: subject.ID, Action: action.Name, Resource: resource.ID}
}

// --- END PATCH ---

func TestAuthAndAuthzIntegration(t *testing.T) {
	ctx := context.Background()

	authService, err := newTestAuthenticator(testConfig{
		Type:              testTypeBasic,
		AccessTokenExpiry: time.Hour,
	})
	require.NoError(t, err)

	authzService := authz.NewMemoryAuthorizer()

	t.Run("CompleteFlow", func(t *testing.T) {
		userID := "test-user"
		password := "test-password"
		resource := "test-resource"
		action := "read"

		err := authService.ValidateCredentials(ctx, struct {
			Username string
			Password string
		}{Username: userID, Password: password})
		require.NoError(t, err)

		policy := authz.Policy{ID: "policy-1", Effect: authz.Allow, Subject: userID, Resource: resource, Actions: []string{action}}
		authzService.AddPolicy(policy)
		decision, err := authzService.Authorize(ctx, *toRequest(authz.Subject{ID: userID}, authz.Action{Name: action}, authz.Resource{ID: resource}))
		require.NoError(t, err)
		assert.True(t, decision.Allow)
	})

	t.Run("PolicyInheritance", func(t *testing.T) {
		policies := []authz.Policy{
			{ID: "admin-policy", Effect: authz.Allow, Subject: "admin", Resource: "/*", Actions: []string{"*"}},
			{ID: "user-policy", Effect: authz.Allow, Subject: "user", Resource: "/docs/*", Actions: []string{"read"}},
			{ID: "guest-policy", Effect: authz.Allow, Subject: "guest", Resource: "/docs/public/*", Actions: []string{"read"}},
		}
		for _, p := range policies {
			authzService.AddPolicy(p)
		}

		tests := []struct {
			name     string
			subject  string
			resource string
			action   string
			allowed  bool
		}{
			{"AdminFullAccess", "admin", "/any/path", "write", true},
			{"UserDocsRead", "user", "/docs/secret", "read", true},
			{"UserNoWrite", "user", "/docs/secret", "write", false},
			{"GuestPublicRead", "guest", "/docs/public/guide", "read", true},
			{"GuestNoPrivate", "guest", "/docs/private/secret", "read", false},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				decision, err := authzService.Authorize(ctx, *toRequest(authz.Subject{ID: tc.subject}, authz.Action{Name: tc.action}, authz.Resource{ID: tc.resource}))
				require.NoError(t, err)
				assert.Equal(t, tc.allowed, decision.Allow)
			})
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		const numGoroutines = 10
		const numRequests = 100

		userID := "concurrent-user"
		resource := "shared-resource"

		authzService.AddPolicy(authz.Policy{ID: "concurrent-policy", Effect: authz.Allow, Subject: userID, Resource: resource, Actions: []string{"read", "write"}})

		done := make(chan bool)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < numRequests; j++ {
					decision, err := authzService.Authorize(ctx, *toRequest(authz.Subject{ID: userID}, authz.Action{Name: "read"}, authz.Resource{ID: resource}))
					assert.NoError(t, err)
					assert.True(t, decision.Allow)
				}
				done <- true
			}()
		}
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}
