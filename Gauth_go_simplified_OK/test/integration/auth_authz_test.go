package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
	"github.com/Gimel-Foundation/gauth/pkg/authz"
)

func TestAuthAndAuthzIntegration(t *testing.T) {
	ctx := context.Background()

	authService, err := auth.NewAuthenticator(auth.Config{
		Type:              auth.TypeBasic,
		AccessTokenExpiry: time.Hour,
	})
	require.NoError(t, err)

	authzService := authz.NewMemoryAuthorizer()

	t.Run("CompleteFlow", func(t *testing.T) {
		userID := "test-user"
		password := "test-password"
		resource := "test-resource"
		action := "read"

		if ba, ok := authService.(interface{ AddClient(string, string) }); ok {
			ba.AddClient(userID, password)
		}
		err := authService.ValidateCredentials(ctx, struct {
			Username string
			Password string
		}{Username: userID, Password: password})
		require.NoError(t, err)

		policy := &authz.Policy{
			ID:        "policy-1",
			Effect:    authz.Allow,
			Subjects:  []authz.Subject{{ID: userID}},
			Resources: []authz.Resource{{ID: resource}},
			Actions:   []authz.Action{{Name: action}},
		}
		err = authzService.AddPolicy(ctx, policy)
		require.NoError(t, err)

		decision, err := authzService.Authorize(ctx,
			authz.Subject{ID: userID},
			authz.Action{Name: action},
			authz.Resource{ID: resource},
		)
		require.NoError(t, err)
		assert.True(t, decision.Allowed)
	})

	t.Run("PolicyInheritance", func(t *testing.T) {
		policies := []*authz.Policy{
			{
				ID:        "admin-policy",
				Effect:    authz.Allow,
				Subjects:  []authz.Subject{{ID: "admin"}},
				Resources: []authz.Resource{{ID: "/*"}},
				Actions:   []authz.Action{{Name: "*"}},
			},
			{
				ID:        "user-policy",
				Effect:    authz.Allow,
				Subjects:  []authz.Subject{{ID: "user"}},
				Resources: []authz.Resource{{ID: "/docs/*"}},
				Actions:   []authz.Action{{Name: "read"}},
			},
			{
				ID:        "guest-policy",
				Effect:    authz.Allow,
				Subjects:  []authz.Subject{{ID: "guest"}},
				Resources: []authz.Resource{{ID: "/docs/public/*"}},
				Actions:   []authz.Action{{Name: "read"}},
			},
		}
		for _, p := range policies {
			err := authzService.AddPolicy(ctx, p)
			require.NoError(t, err)
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
				decision, err := authzService.Authorize(ctx,
					authz.Subject{ID: tc.subject},
					authz.Action{Name: tc.action},
					authz.Resource{ID: tc.resource},
				)
				require.NoError(t, err)
				assert.Equal(t, tc.allowed, decision.Allowed)
			})
		}
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		const numGoroutines = 10
		const numRequests = 100

		userID := "concurrent-user"
		resource := "shared-resource"

		if ba, ok := authService.(interface{ AddClient(string, string) }); ok {
			ba.AddClient(userID, "password")
		}

		err := authzService.AddPolicy(ctx, &authz.Policy{
			ID:        "concurrent-policy",
			Effect:    authz.Allow,
			Subjects:  []authz.Subject{{ID: userID}},
			Resources: []authz.Resource{{ID: resource}},
			Actions:   []authz.Action{{Name: "read"}, {Name: "write"}},
		})
		require.NoError(t, err)

		done := make(chan bool)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < numRequests; j++ {
					decision, err := authzService.Authorize(ctx,
						authz.Subject{ID: userID},
						authz.Action{Name: "read"},
						authz.Resource{ID: resource},
					)
					assert.NoError(t, err)
					assert.True(t, decision.Allowed)
				}
				done <- true
			}()
		}
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}
