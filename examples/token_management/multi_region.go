package tokenmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
)

// Service represents a microservice
type Service struct {
	name      string
	store     token.Store
	blacklist *token.Blacklist
	validator *token.ValidationChain
	jwtMgr    *token.JWTManager
	endpoints map[string]http.HandlerFunc
}

// Region represents a geographical region
type Region struct {
	name     string
	services map[string]*Service
	tokens   *RegionTokenCache
}

// RegionTokenCache provides token caching for a region
type RegionTokenCache struct {
	mu     sync.RWMutex
	cache  map[string]*token.Token
	maxAge time.Duration
}

// NewRegion creates a new region with services
func NewRegion(name string, maxCacheAge time.Duration) *Region {
	return &Region{
		name:     name,
		services: make(map[string]*Service),
		tokens: &RegionTokenCache{
			cache:  make(map[string]*token.Token),
			maxAge: maxCacheAge,
		},
	}
}

// NewService creates a new microservice
func NewService(name string, keyID string, signingKey []byte) *Service {
	store := token.NewMemoryStore(24 * time.Hour)
	blacklist := token.NewBlacklist()

	svc := &Service{
		name:      name,
		store:     store,
		blacklist: blacklist,
		jwtMgr: token.NewJWTManager(token.JWTConfig{
			SigningMethod: token.HS256,
			SigningKey:    signingKey,
			KeyID:         keyID,
			MaxAge:        time.Hour,
		}),
		endpoints: make(map[string]http.HandlerFunc),
	}

	svc.validator = token.NewValidationChain(blacklist)
	svc.setupEndpoints()
	return svc
}

func (s *Service) setupEndpoints() {
	// Token creation endpoint
	s.endpoints["/token/create"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			Subject string   `json:"subject"`
			Scopes  []string `json:"scopes"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		t := &token.Token{
			ID:        token.NewID(),
			Type:      token.Access,
			Subject:   req.Subject,
			Issuer:    s.name,
			IssuedAt:  time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			Scopes:    req.Scopes,
		}

		ctx := r.Context()
		if err := s.store.Save(ctx, t); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		signed, err := s.jwtMgr.SignToken(ctx, t)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"token": signed})
	}

	// Token validation endpoint
	s.endpoints["/token/validate"] = func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "No token provided", http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		t, err := s.jwtMgr.VerifyToken(ctx, auth)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if err := s.validator.Validate(ctx, t); err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid":   true,
			"subject": t.Subject,
			"scopes":  t.Scopes,
		})
	}
}

// SimulateMultiRegionSetup demonstrates a multi-region token setup
func SimulateMultiRegionSetup() {
	// Create regions
	regions := map[string]*Region{
		"us-east":  NewRegion("us-east", 5*time.Minute),
		"eu-west":  NewRegion("eu-west", 5*time.Minute),
		"ap-south": NewRegion("ap-south", 5*time.Minute),
	}

	// Setup services in each region
	sharedKey := []byte("multi-region-shared-key")
	for regionName, region := range regions {
		// Auth service for token management
		region.services["auth"] = NewService(
			fmt.Sprintf("auth-%s", regionName),
			fmt.Sprintf("key-%s", regionName),
			sharedKey,
		)

		// Resource service that validates tokens
		region.services["resource"] = NewService(
			fmt.Sprintf("resource-%s", regionName),
			fmt.Sprintf("key-%s", regionName),
			sharedKey,
		)
	}

	// Simulate cross-region token usage
	ctx := context.Background()

	// Create token in us-east
	usEastAuth := regions["us-east"].services["auth"]
	token := &token.Token{
		ID:        token.NewID(),
		Type:      token.Access,
		Subject:   "global-user",
		Issuer:    "us-east-auth",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Scopes:    []string{"global.read"},
	}

	if err := usEastAuth.store.Save(ctx, token); err != nil {
		log.Fatalf("Failed to save token: %v", err)
	}

	signed, err := usEastAuth.jwtMgr.SignToken(ctx, token)
	if err != nil {
		log.Fatalf("Failed to sign token: %v", err)
	}

	fmt.Printf("Token created in us-east: %s\n\n", signed)

	// Validate token in other regions
	for regionName, region := range regions {
		if regionName == "us-east" {
			continue
		}

		resourceSvc := region.services["resource"]
		verified, err := resourceSvc.jwtMgr.VerifyToken(ctx, signed)
		if err != nil {
			log.Printf("Failed to verify token in %s: %v\n", regionName, err)
			continue
		}

		if err := resourceSvc.validator.Validate(ctx, verified); err != nil {
			log.Printf("Token validation failed in %s: %v\n", regionName, err)
			continue
		}

		fmt.Printf("Token successfully validated in %s\n", regionName)
		fmt.Printf("Subject: %s, Scopes: %v\n\n", verified.Subject, verified.Scopes)
	}
}

func main() {
	fmt.Println("Simulating multi-region token management...")
	SimulateMultiRegionSetup()
}
