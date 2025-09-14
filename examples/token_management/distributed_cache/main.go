package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Gimel-Foundation/gauth/pkg/token"
	redis "github.com/go-redis/redis/v8"
)

// RedisStore implements token storage using Redis
	type RedisStore struct {
		client    redis.UniversalClient
		expiry    time.Duration
		keyPrefix string
	}

	// NewRedisStore creates a new Redis-backed token store
	func NewRedisStore(opts *redis.UniversalOptions, expiry time.Duration) (*RedisStore, error) {
		client := redis.NewUniversalClient(opts)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Ping(ctx).Err(); err != nil {
			return nil, fmt.Errorf("failed to connect to Redis: %w", err)
		}
		return &RedisStore{
			client:    client,
			expiry:    expiry,
			keyPrefix: "token:",
		}, nil
	}

	func (s *RedisStore) key(id string) string {
		return s.keyPrefix + id
	}

func main() {
	// Configure Redis connection (localhost:6379 by default)
	opts := &redis.UniversalOptions{
		Addrs: []string{"localhost:6379"},
		DB:    0,
	}
	expiry := 1 * time.Hour
	store, err := NewRedisStore(opts, expiry)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	ctx := context.Background()
	tokenID := "demo-token-1"
	demoToken := &token.Token{
		ID:        tokenID,
		Type:      token.Access,
		Value:     "demo-token-value",
		IssuedAt:  time.Now(),
		ExpiresAt: time.Now().Add(expiry),
		NotBefore: time.Now(),
		Issuer:    "demo-issuer",
		Subject:   "demo-user",
		Audience:  []string{"demo-api"},
		Scopes:    []string{"read"},
		Algorithm: token.RS256,
	}

	// Store token in Redis
	data, err := json.Marshal(demoToken)
	if err != nil {
		log.Fatalf("Failed to marshal token: %v", err)
	}
	if err := store.client.Set(ctx, store.key(tokenID), data, expiry).Err(); err != nil {
		log.Fatalf("Failed to store token in Redis: %v", err)
	}
	fmt.Println("Token stored in Redis.")

	// Retrieve token from Redis
	val, err := store.client.Get(ctx, store.key(tokenID)).Result()
	if err != nil {
		log.Fatalf("Failed to retrieve token from Redis: %v", err)
	}
	var retrieved token.Token
	if err := json.Unmarshal([]byte(val), &retrieved); err != nil {
		log.Fatalf("Failed to unmarshal token: %v", err)
	}
		fmt.Printf("Retrieved token: ID=%s, Subject=%s, ExpiresAt=%s\n", retrieved.ID, retrieved.Subject, retrieved.ExpiresAt)
	}


