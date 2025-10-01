package authz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisAuthorizerConfig contains configuration for Redis-based authorizer
type RedisAuthorizerConfig struct {
	Client    *redis.Client
	KeyPrefix string
	PolicyTTL time.Duration
	CacheTTL  time.Duration
}

// RedisAuthorizer implements Authorizer with Redis storage
type RedisAuthorizer struct {
	config RedisAuthorizerConfig
	memory *memoryAuthorizer // Local cache
}

// NewRedisAuthorizer creates a new Redis-based authorizer
func NewRedisAuthorizer(config RedisAuthorizerConfig) (Authorizer, error) {
	if config.Client == nil {
		return nil, fmt.Errorf("redis client is required")
	}

	if config.KeyPrefix == "" {
		config.KeyPrefix = "authz:"
	}

	if config.PolicyTTL == 0 {
		config.PolicyTTL = 24 * time.Hour
	}

	if config.CacheTTL == 0 {
		config.CacheTTL = 5 * time.Minute
	}

	return &RedisAuthorizer{
		config: config,
		memory: NewMemoryAuthorizer().(*memoryAuthorizer),
	}, nil
}

// Authorize implements the Authorizer interface for RedisAuthorizer
func (a *RedisAuthorizer) Authorize(ctx context.Context, subject Subject, action Action, resource Resource) (*Decision, error) {
	req := &AccessRequest{
		Subject:  subject,
		Action:   action,
		Resource: resource,
	}
	resp, err := a.IsAllowed(ctx, req)
	if err != nil {
		return nil, err
	}
	return &Decision{
		Allowed:   resp.Allowed,
		Reason:    resp.Reason,
		Policy:    resp.PolicyID,
		Timestamp: time.Now(),
	}, nil
}

func (a *RedisAuthorizer) policyKey(id string) string {
	return a.config.KeyPrefix + "policy:" + id
}

func (a *RedisAuthorizer) roleKey(role Role) string {
	return a.config.KeyPrefix + "role:" + string(role)
}

func (a *RedisAuthorizer) assignmentKey(subject Subject) string {
	return a.config.KeyPrefix + "assignment:" + subject.ID
}

func (a *RedisAuthorizer) AddPolicy(ctx context.Context, policy *Policy) error {
	// Store in Redis
	data, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	key := a.policyKey(policy.ID)
	if err := a.config.Client.Set(ctx, key, data, a.config.PolicyTTL).Err(); err != nil {
		return fmt.Errorf("failed to store policy in Redis: %w", err)
	}

	// Update local cache
	return a.memory.AddPolicy(ctx, policy)
}

func (a *RedisAuthorizer) RemovePolicy(ctx context.Context, policyID string) error {
	key := a.policyKey(policyID)
	if err := a.config.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove policy from Redis: %w", err)
	}

	return a.memory.RemovePolicy(ctx, policyID)
}

func (a *RedisAuthorizer) UpdatePolicy(ctx context.Context, policy *Policy) error {
	return a.AddPolicy(ctx, policy) // Same operation for Redis
}

func (a *RedisAuthorizer) GetPolicy(ctx context.Context, policyID string) (*Policy, error) {
	// Check local cache first
	if policy, err := a.memory.GetPolicy(ctx, policyID); err == nil {
		return policy, nil
	}

	// Get from Redis
	key := a.policyKey(policyID)
	data, err := a.config.Client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("policy not found: %s", policyID)
		}
		return nil, fmt.Errorf("failed to get policy from Redis: %w", err)
	}

	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy: %w", err)
	}

	// Update local cache
	if err := a.memory.AddPolicy(ctx, &policy); err != nil {
		// Log the error but don't fail the main operation
		// since the policy is already stored in Redis
		fmt.Printf("Warning: failed to update local cache: %v\n", err)
	}

	return &policy, nil
}

func (a *RedisAuthorizer) ListPolicies(ctx context.Context) ([]*Policy, error) {
	// Get all policy keys
	pattern := a.policyKey("*")
	keys, err := a.config.Client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to list policies from Redis: %w", err)
	}

	var policies []*Policy
	for _, key := range keys {
		data, err := a.config.Client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var policy Policy
		if err := json.Unmarshal(data, &policy); err != nil {
			continue
		}

		policies = append(policies, &policy)
	}

	return policies, nil
}

func (a *RedisAuthorizer) IsAllowed(ctx context.Context, request *AccessRequest) (*AccessResponse, error) {
	// Try local cache first
	response, err := a.memory.IsAllowed(ctx, request)
	if err == nil {
		return response, nil
	}

	// Get all policies from Redis and evaluate
	policies, err := a.ListPolicies(ctx)
	if err != nil {
		return nil, err
	}

	// Update local cache
	for _, policy := range policies {
		if err := a.memory.AddPolicy(ctx, policy); err != nil {
			// Log the error but continue with other policies
			fmt.Printf("Warning: failed to add policy to local cache: %v\n", err)
		}
	}

	return a.memory.IsAllowed(ctx, request)
}

func (a *RedisAuthorizer) AddRole(ctx context.Context, role Role, permissions []Permission) error {
	data, err := json.Marshal(permissions)
	if err != nil {
		return fmt.Errorf("failed to marshal permissions: %w", err)
	}

	key := a.roleKey(role)
	if err := a.config.Client.Set(ctx, key, data, a.config.PolicyTTL).Err(); err != nil {
		return fmt.Errorf("failed to store role in Redis: %w", err)
	}

	return a.memory.AddRole(ctx, role, permissions)
}

func (a *RedisAuthorizer) RemoveRole(ctx context.Context, role Role) error {
	key := a.roleKey(role)
	if err := a.config.Client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to remove role from Redis: %w", err)
	}

	return a.memory.RemoveRole(ctx, role)
}

func (a *RedisAuthorizer) AssignRole(ctx context.Context, subject Subject, role Role) error {
	// Check if role exists
	key := a.roleKey(role)
	exists, err := a.config.Client.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("failed to check role existence: %w", err)
	}
	if exists == 0 {
		return fmt.Errorf("role not found: %s", role)
	}

	// Get current assignments
	assignKey := a.assignmentKey(subject)
	var roles []Role
	data, err := a.config.Client.Get(ctx, assignKey).Bytes()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get role assignments: %w", err)
	}
	if err != redis.Nil {
		if unmarshalErr := json.Unmarshal(data, &roles); unmarshalErr != nil {
			return fmt.Errorf("failed to unmarshal role assignments: %w", unmarshalErr)
		}
	}

	// Check if role is already assigned
	for _, r := range roles {
		if r == role {
			return nil
		}
	}

	// Add role
	roles = append(roles, role)
	data, err = json.Marshal(roles)
	if err != nil {
		return fmt.Errorf("failed to marshal role assignments: %w", err)
	}

	if err := a.config.Client.Set(ctx, assignKey, data, a.config.PolicyTTL).Err(); err != nil {
		return fmt.Errorf("failed to store role assignment: %w", err)
	}

	return a.memory.AssignRole(ctx, subject, role)
}

func (a *RedisAuthorizer) UnassignRole(ctx context.Context, subject Subject, role Role) error {
	assignKey := a.assignmentKey(subject)
	var roles []Role
	data, err := a.config.Client.Get(ctx, assignKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("no roles assigned to subject: %s", subject)
		}
		return fmt.Errorf("failed to get role assignments: %w", err)
	}

	if unmarshalErr := json.Unmarshal(data, &roles); unmarshalErr != nil {
		return fmt.Errorf("failed to unmarshal role assignments: %w", unmarshalErr)
	}

	found := false
	for i, r := range roles {
		if r == role {
			roles = append(roles[:i], roles[i+1:]...)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("role not assigned to subject: %s", role)
	}

	if len(roles) == 0 {
		if delErr := a.config.Client.Del(ctx, assignKey).Err(); delErr != nil {
			return fmt.Errorf("failed to remove role assignments: %w", delErr)
		}
	} else {
		data, err = json.Marshal(roles)
		if err != nil {
			return fmt.Errorf("failed to marshal role assignments: %w", err)
		}

		if err := a.config.Client.Set(ctx, assignKey, data, a.config.PolicyTTL).Err(); err != nil {
			return fmt.Errorf("failed to update role assignments: %w", err)
		}
	}

	return a.memory.UnassignRole(ctx, subject, role)
}

func (a *RedisAuthorizer) GetRoles(ctx context.Context, subject Subject) ([]Role, error) {
	// Check local cache first
	roles, err := a.memory.GetRoles(ctx, subject)
	if err == nil {
		return roles, nil
	}

	// Get from Redis
	assignKey := a.assignmentKey(subject)
	data, err := a.config.Client.Get(ctx, assignKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return []Role{}, nil
		}
		return nil, fmt.Errorf("failed to get role assignments: %w", err)
	}

	if err := json.Unmarshal(data, &roles); err != nil {
		return nil, fmt.Errorf("failed to unmarshal role assignments: %w", err)
	}

	return roles, nil
}
