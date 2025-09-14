// Moved from multi_region.go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	token "github.com/Gimel-Foundation/gauth/pkg/token"
)

type Service struct {
	name      string
	store     token.Store
	blacklist *token.Blacklist
	validator *token.ValidationChain
	endpoints map[string]http.HandlerFunc
}

type Region struct {
	name     string
	services map[string]*Service
	tokens   *RegionTokenCache
}

type RegionTokenCache struct {
	mu     sync.RWMutex
	cache  map[string]*token.Token
	maxAge time.Duration
}

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

func NewService(name string, keyID string, signingKey []byte) *Service {
	store := token.NewMemoryStore(24 * time.Hour)
	blacklist := token.NewBlacklist()

	svc := &Service{
		name:      name,
		store:     store,
		blacklist: blacklist,
		endpoints: make(map[string]http.HandlerFunc),
	}

	svc.validator = token.NewValidationChain(token.ValidationConfig{
		AllowedIssuers: []string{name},
		ClockSkew:      2 * time.Minute,
	}, blacklist)
	svc.setupEndpoints()
	return svc
}

func (s *Service) setupEndpoints() {
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

		// Example token creation (not used)
	}
}

func main() {
	fmt.Println("Multi-region token management example loaded.")
}
