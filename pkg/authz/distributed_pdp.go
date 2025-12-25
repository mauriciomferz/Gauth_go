package authz

import (
	"sort"
	"strings"

	"github.com/mauriciomferz/Gauth_go/pkg/policy"
)

// DecisionCache provides a cache for PDP decisions with invalidation hooks.
type DecisionCache interface {
	Get(key string) (Decision, bool)
	Set(key string, decision Decision)
	Invalidate(key string)
}

// DistributedPDP supports distributed policy decision points.
type DistributedPDP interface {
	Decide(request map[string]interface{}) (Decision, error)
	RegisterCache(cache DecisionCache)
}

// DefaultDecisionCache is a simple in-memory implementation.
type DefaultDecisionCache struct {
	store map[string]Decision
}

func NewDefaultDecisionCache() *DefaultDecisionCache {
	return &DefaultDecisionCache{store: make(map[string]Decision)}
}

func (c *DefaultDecisionCache) Get(key string) (Decision, bool) {
	d, ok := c.store[key]
	return d, ok
}

func (c *DefaultDecisionCache) Set(key string, decision Decision) {
	c.store[key] = decision
}

func (c *DefaultDecisionCache) Invalidate(key string) {
	delete(c.store, key)
}

// DefaultDistributedPDP implements distributed policy decision logic with local caching.
type DefaultDistributedPDP struct {
	cache  DecisionCache
	engine policy.Engine
}

// NewDefaultDistributedPDP creates a new distributed PDP backed by a policy engine.
func NewDefaultDistributedPDP(engine policy.Engine) *DefaultDistributedPDP {
	return &DefaultDistributedPDP{
		cache:  NewDefaultDecisionCache(),
		engine: engine,
	}
}

func (p *DefaultDistributedPDP) Decide(request map[string]interface{}) (Decision, error) {
	// 1. Construct Cache Key
	key := p.generateCacheKey(request)

	// 2. Check Cache
	if p.cache != nil {
		if dec, found := p.cache.Get(key); found {
			return dec, nil
		}
	}

	// 3. Evaluate via Policy Engine
	input := p.mapRequestToInput(request)
	policyDec, err := p.engine.EvaluateAuthorization(input)
	if err != nil {
		return Decision{}, err
	}

	// 4. Map Result
	dec := Decision{
		Allow:    policyDec.Allow,
		Allowed:  policyDec.Allow,
		Reason:   policyDec.ReasonCode,
		Metadata: policyDec.Metadata,
	}

	// 5. Update Cache
	if p.cache != nil {
		p.cache.Set(key, dec)
	}

	return dec, nil
}

func (p *DefaultDistributedPDP) RegisterCache(cache DecisionCache) {
	p.cache = cache
}

func (p *DefaultDistributedPDP) generateCacheKey(req map[string]interface{}) string {
	subject, _ := req["subject"].(string)
	action, _ := req["action"].(string)
	resource, _ := req["resource"].(string)

	var sb strings.Builder
	sb.WriteString(subject)
	sb.WriteByte('|')
	sb.WriteString(action)
	sb.WriteByte('|')
	sb.WriteString(resource)

	if scopes, ok := req["scopes"].([]string); ok {
		sb.WriteByte('|')
		// Sort scopes for determinism
		configScopes := make([]string, len(scopes))
		copy(configScopes, scopes)
		sort.Strings(configScopes)
		sb.WriteString(strings.Join(configScopes, ","))
	}

	if ctx, ok := req["context"].(map[string]interface{}); ok {
		keys := make([]string, 0, len(ctx))
		for k := range ctx {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if s, ok := ctx[k].(string); ok {
				sb.WriteByte('|')
				sb.WriteString(k)
				sb.WriteByte('=')
				sb.WriteString(s)
			}
		}
	}

	return sb.String()
}

func (p *DefaultDistributedPDP) mapRequestToInput(req map[string]interface{}) policy.AuthzInput {
	subject, _ := req["subject"].(string)
	action, _ := req["action"].(string)
	resource, _ := req["resource"].(string)

	var scopes []string
	if s, ok := req["scopes"].([]string); ok {
		scopes = s
	}

	attributes := make(map[string]string)
	// Deterministic context iteration
	if ctx, ok := req["context"].(map[string]interface{}); ok {
		keys := make([]string, 0, len(ctx))
		for k := range ctx {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if s, ok := ctx[k].(string); ok {
				attributes[k] = s
				// Add to cache key to differentiate requests with different context
			}
		}
	}

	return policy.AuthzInput{
		Subject:    subject,
		Action:     action,
		Resource:   resource,
		Scopes:     scopes,
		Attributes: attributes,
	}
}
