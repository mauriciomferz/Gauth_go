package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
	"github.com/Gimel-Foundation/gauth/pkg/authz"
	"github.com/Gimel-Foundation/gauth/pkg/token/store"
)

func TestAuthAndAuthzIntegration(t *testing.T) {
	ctx := context.Background()

	// Setup token store
	tokenStore, err := store.NewMemoryStore(store.Config{
		EncryptionKey: []byte("test-key-32-bytes-long-required!"),
		TokenTTL:      time.Hour,
	})
	require.NoError(t, err)
	defer tokenStore.Close()

	// Setup auth service
	authService := auth.New(auth.Config{
		TokenType: auth.JWT,
		Store:     tokenStore,
		TTL:       time.Hour,
		EnableMFA: true,
	})

	// Setup authz service
	authzService := authz.New(authz.Config{
		PolicyStore: authz.NewMemoryPolicyStore(),
		EnableAudit: true,
	})

	// Test complete authentication and authorization flow
	t.Run("CompleteFlow", func(t *testing.T) {
		userID := "test-user"
		password := "test-password"
		resource := "test-resource"
		action := "read"

		// 1. Register user
		err := authService.Register(ctx, auth.Credentials{
			Username: userID,
			Password: password,
		})
		require.NoError(t, err)

		// 2. Authenticate
		token, err := authService.Authenticate(ctx, auth.Credentials{
			Username: userID,
			Password: password,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// 3. Validate token
		claims, err := authService.ValidateToken(ctx, token)
		require.NoError(t, err)
		assert.Equal(t, userID, claims.Subject)

		// 4. Create policy
		policy := &authz.Policy{
			Subject:  userID,
			Resource: resource,
			Actions:  []string{action},
			Effect:   authz.Allow,
		}
		err = authzService.AddPolicy(ctx, policy)
		require.NoError(t, err)

		// 5. Check authorization
		allowed, err := authzService.IsAllowed(ctx, authz.Request{
			Subject:  userID,
			Resource: resource,
			Action:   action,
		})
		require.NoError(t, err)
		assert.True(t, allowed)

		// 6. Revoke token
		err = authService.RevokeToken(ctx, token)
		require.NoError(t, err)

		// 7. Verify token is revoked
		_, err = authService.ValidateToken(ctx, token)
		assert.Error(t, err)
	})

	// Test MFA flow
	t.Run("MFAFlow", func(t *testing.T) {
		userID := "mfa-user"
		password := "mfa-password"

		// 1. Register user
		err := authService.Register(ctx, auth.Credentials{
			Username: userID,
			Password: password,
		})
		require.NoError(t, err)

		// 2. Enable MFA
		secret, err := authService.EnableMFA(ctx, userID)
		require.NoError(t, err)
		assert.NotEmpty(t, secret)

		// 3. Generate MFA code (normally done by authenticator app)
		code, err := authService.GenerateMFACode(ctx, secret)
		require.NoError(t, err)

		// 4. Authenticate with MFA
		token, err := authService.AuthenticateWithMFA(ctx, auth.Credentials{
			Username: userID,
			Password: password,
			MFACode:  code,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	// Test policy inheritance
	t.Run("PolicyInheritance", func(t *testing.T) {
		// Create hierarchical policies
		policies := []*authz.Policy{
			{
				Subject:  "admin",
				Resource: "/*",
				Actions:  []string{"*"},
				Effect:   authz.Allow,
			},
			{
				Subject:  "user",
				Resource: "/docs/*",
				Actions:  []string{"read"},
				Effect:   authz.Allow,
			},
			{
				Subject:  "guest",
				Resource: "/docs/public/*",
				Actions:  []string{"read"},
				Effect:   authz.Allow,
			},
		}

		// Add policies
		for _, p := range policies {
			err := authzService.AddPolicy(ctx, p)
			require.NoError(t, err)
		}

		// Test different access levels
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
				allowed, err := authzService.IsAllowed(ctx, authz.Request{
					Subject:  tc.subject,
					Resource: tc.resource,
					Action:   tc.action,
				})
				require.NoError(t, err)
				assert.Equal(t, tc.allowed, allowed)
			})
		}
	})

	// Test concurrent access
	t.Run("ConcurrentAccess", func(t *testing.T) {
		const numGoroutines = 10
		const numRequests = 100

		// Create shared resources
		userID := "concurrent-user"
		resource := "shared-resource"

		err := authService.Register(ctx, auth.Credentials{
			Username: userID,
			Password: "password",
		})
		require.NoError(t, err)

		err = authzService.AddPolicy(ctx, &authz.Policy{
			Subject:  userID,
			Resource: resource,
			Actions:  []string{"read", "write"},
			Effect:   authz.Allow,
		})
		require.NoError(t, err)

		// Run concurrent auth checks
		done := make(chan bool)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				for j := 0; j < numRequests; j++ {
					allowed, err := authzService.IsAllowed(ctx, authz.Request{
						Subject:  userID,
						Resource: resource,
						Action:   "read",
					})
					assert.NoError(t, err)
					assert.True(t, allowed)
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}
