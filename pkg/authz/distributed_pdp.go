package authz

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

// DefaultDistributedPDP is a stub implementation for demo/testing.
type DefaultDistributedPDP struct {
	cache DecisionCache
}

func (p *DefaultDistributedPDP) Decide(request map[string]interface{}) (Decision, error) {
	// TODO: Implement distributed decision logic
	return Decision{Allow: false, Allowed: false, Policies: nil, Reason: "stub", Metadata: nil}, nil
}

func (p *DefaultDistributedPDP) RegisterCache(cache DecisionCache) {
	p.cache = cache
}
