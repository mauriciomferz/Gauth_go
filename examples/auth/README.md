# Authentication Examples

This directory contains examples demonstrating various authentication patterns.

## Basic Authentication

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	// Initialize auth service
	service := auth.New(auth.Config{
		SigningKey:  []byte("your-secret-key"),
		TokenExpiry: 1 * time.Hour,
	})

	// Create claims
	claims := auth.NewClaims().
		SetString("sub", "user123").
		SetString("name", "John Doe").
		SetStringSlice("scope", []string{"read", "write"})

	// Generate token
	token, err := service.GenerateToken(context.Background(), claims)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	// Validate token
	validClaims, err := service.ValidateToken(context.Background(), token)
	if err != nil {
		log.Fatalf("Failed to validate token: %v", err)
	}

	// Access claims
	subject, _ := validClaims.GetString("sub")
	scopes, _ := validClaims.GetStringSlice("scope")
	log.Printf("Token validated for user %s with scopes %v", subject, scopes)
}
```

## OAuth2 Flow

```go
package main

import (
	"context"
	"log"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	// Create OAuth2 config
	oauth := auth.NewOAuth2(auth.OAuth2Config{
		ClientID:     "your-client-id",
		ClientSecret: "your-client-secret",
		RedirectURL:  "http://localhost:8080/callback",
		Scopes:      []string{"profile", "email"},
	})

	// Get authorization URL
	url := oauth.GetAuthorizationURL("state123")
	log.Printf("Visit URL to authorize: %s", url)

	// Exchange code for token (in callback handler)
	code := "..." // From callback
	token, err := oauth.ExchangeCode(context.Background(), code)
	if err != nil {
		log.Fatalf("Failed to exchange code: %v", err)
	}

	// Validate token
	claims, err := oauth.ValidateToken(context.Background(), token.AccessToken)
	if err != nil {
		log.Fatalf("Failed to validate token: %v", err)
	}

	log.Printf("Token validated for user %s", claims.Subject)
}
```

## PASETO Tokens

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	// Create PASETO config
	paseto := auth.NewPASETO(auth.PASETOConfig{
		Version: auth.V2,
		Purpose: auth.Local,
		KeyRotation: auth.KeyRotationConfig{
			Interval: 24 * time.Hour,
		},
	})

	// Generate token
	claims := auth.NewClaims().
		SetString("sub", "user123").
		SetStringSlice("scope", []string{"read", "write"})

	token, err := paseto.GenerateToken(context.Background(), claims)
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	// Validate token
	validClaims, err := paseto.ValidateToken(context.Background(), token)
	if err != nil {
		log.Fatalf("Failed to validate token: %v", err)
	}

	log.Printf("Token validated for user %s", validClaims.GetString("sub"))
}
```

## Key Rotation

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	// Create service with key rotation
	service := auth.New(auth.Config{
		SigningKey:  []byte("initial-key"),
		TokenExpiry: 1 * time.Hour,
		KeyRotation: auth.KeyRotationConfig{
			Enabled:  true,
			Interval: 24 * time.Hour,
			Storage:  auth.NewFileKeyStore("keys/"),
		},
	})

	// Start key rotation
	ctx := context.Background()
	if err := service.StartKeyRotation(ctx); err != nil {
		log.Fatalf("Failed to start key rotation: %v", err)
	}
	defer service.StopKeyRotation()

	// Generate token
	token, err := service.GenerateToken(ctx, auth.NewClaims())
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	// Validate token (will work even after key rotation)
	_, err = service.ValidateToken(ctx, token)
	if err != nil {
		log.Fatalf("Failed to validate token: %v", err)
	}
}
```

## Encrypted Storage

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	// Create encrypted store
	store, err := auth.NewEncryptedStore(auth.EncryptedStoreConfig{
		EncryptionKey: []byte("32-byte-key-for-AES-256----------"),
		Backend:      auth.NewRedisStore("localhost:6379"),
	})
	if err != nil {
		log.Fatalf("Failed to create store: %v", err)
	}

	// Create service with encrypted store
	service := auth.New(auth.Config{
		SigningKey:  []byte("signing-key"),
		TokenExpiry: 1 * time.Hour,
		Store:      store,
	})

	// Use service
	token, err := service.GenerateToken(context.Background(), auth.NewClaims())
	if err != nil {
		log.Fatalf("Failed to generate token: %v", err)
	}

	log.Printf("Generated token: %s", token)
}
```

## Error Handling

```go
package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/auth"
)

func main() {
	service := auth.New(auth.Config{
		SigningKey:  []byte("secret"),
		TokenExpiry: 1 * time.Hour,
	})

	// Generate expired token
	claims := auth.NewClaims()
	claims.SetInt64("exp", time.Now().Add(-1*time.Hour).Unix())

	token, _ := service.GenerateToken(context.Background(), claims)

	// Validate token
	_, err := service.ValidateToken(context.Background(), token)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrExpiredToken):
			log.Println("Token has expired")
		case errors.Is(err, auth.ErrInvalidToken):
			log.Println("Token is invalid")
		case errors.Is(err, auth.ErrMalformedToken):
			log.Println("Token is malformed")
		default:
			log.Printf("Unknown error: %v", err)
		}
	}
}
```

## Best Practices

1. **Always use context**
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
   defer cancel()
   token, err := service.GenerateToken(ctx, claims)
   ```

2. **Handle all errors**
   ```go
   if err != nil {
       if errors.Is(err, auth.ErrExpiredToken) {
           // Handle expired token
       }
       return fmt.Errorf("token validation failed: %w", err)
   }
   ```

3. **Use strong keys**
   ```go
   key := make([]byte, 32)
   if _, err := rand.Read(key); err != nil {
       log.Fatal(err)
   }
   ```

4. **Clean up resources**
   ```go
   store := auth.NewStore()
   defer store.Close()
   ```

5. **Validate input**
   ```go
   if len(token) == 0 {
       return auth.ErrInvalidToken
   }
   ```