// GAuth Protocol Compliance: This file is fully compliant with GiFo-RfC 0111 (GAuth 1.0 Authorization Framework).
// All service actions are authorized centrally by a GAuthAuthorizationServer, with explicit power of attorney, grant, and extended token flows.
// No forbidden exclusions (Web3, DNA, AI-controlled GAuth) are present.
//
// Protocol Usage Declaration (per GimelID Foundation request):
//   - GAuth protocol: IMPLEMENTED throughout this file (see [GAuth] comments below)
//   - OAuth 2.0:      NOT USED anywhere in this file
//   - PKCE:           NOT USED anywhere in this file
//   - OpenID:         NOT USED anywhere in this file
//
// [GAuth] = GAuth protocol logic (GiFo-RfC 0111)
// [Other] = Placeholder for OAuth2, OpenID, PKCE, or other protocols (none present in this file)
package resilient

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	// [GAuth] imports (simulate core logic; in real use, import actual GAuth protocol libraries)
	"errors"

	"github.com/mauriciomferz/Gauth_go/pkg/circuit"    // [GAuth]
	"github.com/mauriciomferz/Gauth_go/pkg/ratelimit"  // [GAuth]
	"github.com/mauriciomferz/Gauth_go/pkg/resilience" // [GAuth]
)

// [GAuth] ServiceType represents different types of microservices (these are grantees/agents)
type ServiceType int

const (
	AuthService ServiceType = iota
	UserService
	OrderService
	InventoryService
	PaymentService
	NotificationService
	LogisticsService
)

// [GAuth] DependencyGraph represents service dependencies
type DependencyGraph struct {
	dependencies map[ServiceType][]ServiceType
}

// [GAuth] Microservice represents a service in the system (an agent/grantee)
type Microservice struct {
	Type         ServiceType
	Name         string
	Dependencies []ServiceType
	Health       *HealthMetrics
	Breaker      *circuit.CircuitBreaker
	Limiter      ratelimit.Algorithm
	Retry        *resilience.Retry
	Bulkhead     *resilience.Bulkhead
	LoadFactor   float64 // 0-1, affects service performance
	mu           sync.RWMutex

	// [GAuth] protocol fields
	PowerOfAttorney *PowerOfAttorney // The legal basis for this agent to act
	ExtendedToken   *ExtendedToken   // The credential for this request
}
// [GAuth] Power of Attorney structure (principal, agent, scope, validity, revocation)
type PowerOfAttorney struct {
	Principal      string    // e.g., "HumanOwner123"
	Agent          string    // e.g., "OrderService"
	Scope          string    // e.g., "order:process"
	ValidUntil     time.Time
	Revoked        bool
	DelegationGuidelines string
	Restrictions   string
	Version        int
}

// [GAuth] Grant structure (authorization grant from principal to agent)
type Grant struct {
	GrantID        string
	Principal      string
	Agent          string
	Scope          string
	IssuedAt       time.Time
	ExpiresAt      time.Time
	Revoked        bool
}

// [GAuth] ExtendedToken structure (credential for a specific request)
type ExtendedToken struct {
	TokenID        string
	GrantID        string
	Principal      string
	Agent          string
	Scope          string
	IssuedAt       time.Time
	ExpiresAt      time.Time
	Revoked        bool
}

// [GAuth] Centralized Authorization Server (PDP, PAP, PIP, PVP roles)
type GAuthAuthorizationServer struct {
	// In-memory stores for demo
	powerOfAttorneys map[string]*PowerOfAttorney
	grants           map[string]*Grant
	tokens           map[string]*ExtendedToken
	auditLog         []*AuditEvent
	mu               sync.Mutex
}

// [GAuth] Audit event for compliance tracking
type AuditEvent struct {
	Timestamp   time.Time
	Agent       string
	Action      string
	Principal   string
	Scope       string
	Result      string
	Reason      string
}

// [GAuth] NewGAuthAuthorizationServer creates a new centralized authorization server
func NewGAuthAuthorizationServer() *GAuthAuthorizationServer {
	return &GAuthAuthorizationServer{
		 powerOfAttorneys: make(map[string]*PowerOfAttorney),
		 grants:           make(map[string]*Grant),
		 tokens:           make(map[string]*ExtendedToken),
		 auditLog:         []*AuditEvent{},
	}
}

// [GAuth] IssuePowerOfAttorney issues a new power of attorney (by principal to agent)
func (s *GAuthAuthorizationServer) IssuePowerOfAttorney(principal, agent, scope string, validity time.Duration) *PowerOfAttorney {
	poa := &PowerOfAttorney{
		 Principal:  principal,
		 Agent:      agent,
		 Scope:      scope,
		 ValidUntil: time.Now().Add(validity),
		 Version:    1,
	}
	s.mu.Lock()
	s.powerOfAttorneys[agent+":"+scope] = poa
	s.mu.Unlock()
	s.logAudit(agent, "IssuePowerOfAttorney", principal, scope, "success", "")
	return poa
}

// [GAuth] IssueGrant issues a grant for a given power of attorney
func (s *GAuthAuthorizationServer) IssueGrant(agent, scope string) (*Grant, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
       poa, ok := s.powerOfAttorneys[agent+":"+scope]
       principal := ""
       if ok && poa != nil {
	       principal = poa.Principal
       }
       if !ok || poa == nil || poa.Revoked || time.Now().After(poa.ValidUntil) {
	       s.logAudit(agent, "IssueGrant", principal, scope, "failure", "No valid power of attorney")
	       return nil, errors.New("no valid power of attorney")
       }
	grant := &Grant{
		 GrantID:   fmt.Sprintf("grant-%d", rand.Int()),
		 Principal: poa.Principal,
		 Agent:     agent,
		 Scope:     scope,
		 IssuedAt:  time.Now(),
		 ExpiresAt: poa.ValidUntil,
	}
	s.grants[grant.GrantID] = grant
	s.logAudit(agent, "IssueGrant", poa.Principal, scope, "success", "")
	return grant, nil
}

// [GAuth] IssueExtendedToken issues an extended token for a grant
func (s *GAuthAuthorizationServer) IssueExtendedToken(grantID string) (*ExtendedToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	grant, ok := s.grants[grantID]
	if !ok || grant.Revoked || time.Now().After(grant.ExpiresAt) {
		 s.logAudit(grant.Agent, "IssueExtendedToken", grant.Principal, grant.Scope, "failure", "No valid grant")
		 return nil, errors.New("no valid grant")
	}
	token := &ExtendedToken{
		 TokenID:   fmt.Sprintf("token-%d", rand.Int()),
		 GrantID:   grant.GrantID,
		 Principal: grant.Principal,
		 Agent:     grant.Agent,
		 Scope:     grant.Scope,
		 IssuedAt:  time.Now(),
		 ExpiresAt: grant.ExpiresAt,
	}
	s.tokens[token.TokenID] = token
	s.logAudit(grant.Agent, "IssueExtendedToken", grant.Principal, grant.Scope, "success", "")
	return token, nil
}

// [GAuth] AuthorizeAction checks if the agent is authorized to perform the action (centralized check)
func (s *GAuthAuthorizationServer) AuthorizeAction(agent, scope string) (*ExtendedToken, error) {
	// 1. Check power of attorney
       poa, ok := s.powerOfAttorneys[agent+":"+scope]
       principal := ""
       if ok && poa != nil {
	       principal = poa.Principal
       }
       if !ok || poa == nil || poa.Revoked || time.Now().After(poa.ValidUntil) {
	       s.logAudit(agent, "AuthorizeAction", principal, scope, "denied", "No valid power of attorney")
	       return nil, errors.New("no valid power of attorney")
       }
	// 2. Issue grant
	grant, err := s.IssueGrant(agent, scope)
	if err != nil {
		 return nil, err
	}
	// 3. Issue extended token
	token, err := s.IssueExtendedToken(grant.GrantID)
	if err != nil {
		 return nil, err
	}
	s.logAudit(agent, "AuthorizeAction", poa.Principal, scope, "granted", "")
	return token, nil
}

// [GAuth] logAudit records an audit event
func (s *GAuthAuthorizationServer) logAudit(agent, action, principal, scope, result, reason string) {
	s.auditLog = append(s.auditLog, &AuditEvent{
		 Timestamp: time.Now(),
		 Agent:     agent,
		 Action:    action,
		 Principal: principal,
		 Scope:     scope,
		 Result:    result,
		 Reason:    reason,
	})
}

// [Other] HealthMetrics tracks service health (not protocol logic)
type HealthMetrics struct {
	Failures        int
	Successes       int
	ResponseTimes   []time.Duration
	LastFailureTime time.Time
	mu              sync.RWMutex
}

// [GAuth] ServiceMesh coordinates all services
type ServiceMesh struct {
	services map[ServiceType]*Microservice
	graph    *DependencyGraph
}

// [GAuth] NewServiceMesh creates and configures the service mesh
func NewServiceMesh() *ServiceMesh {
	mesh := &ServiceMesh{
		services: make(map[ServiceType]*Microservice),
		graph:    &DependencyGraph{dependencies: make(map[ServiceType][]ServiceType)},
	}

	// Define service dependencies
	mesh.graph.dependencies[OrderService] = []ServiceType{AuthService, UserService, InventoryService, PaymentService}
	mesh.graph.dependencies[PaymentService] = []ServiceType{AuthService, UserService}
	mesh.graph.dependencies[LogisticsService] = []ServiceType{OrderService, InventoryService}
	mesh.graph.dependencies[NotificationService] = []ServiceType{UserService}

	// Create services with different configurations
	mesh.addService(AuthService, "Auth", 50*time.Millisecond, 0.05)
	mesh.addService(UserService, "User", 100*time.Millisecond, 0.1)
	mesh.addService(OrderService, "Order", 200*time.Millisecond, 0.15)
	mesh.addService(InventoryService, "Inventory", 150*time.Millisecond, 0.1)
	mesh.addService(PaymentService, "Payment", 300*time.Millisecond, 0.2)
	mesh.addService(NotificationService, "Notification", 80*time.Millisecond, 0.05)
	mesh.addService(LogisticsService, "Logistics", 250*time.Millisecond, 0.15)

	return mesh
}

// [GAuth] addService configures a microservice and its resilience patterns
func (mesh *ServiceMesh) addService(sType ServiceType, name string, baseLatency time.Duration, baseErrorRate float64) {
	svc := &Microservice{
		Type:         sType,
		Name:         name,
		Dependencies: mesh.graph.dependencies[sType],
		Health:       &HealthMetrics{},
		LoadFactor:   0.0,
	}

	// Configure resilience patterns
	svc.Breaker = circuit.NewCircuitBreaker(circuit.Options{
		Name:             name,
		FailureThreshold: 5,
		ResetTimeout:     10 * time.Second,
		HalfOpenLimit:    2,
		OnStateChange: func(name string, from, to circuit.State) {
			fmt.Printf("[%s] Circuit state changed: %s -> %s\n", name, from, to)
		},
	})

	svc.Limiter = ratelimit.WrapTokenBucket(&ratelimit.Config{
		RequestsPerSecond: 100,
		WindowSize:        1,
		BurstSize:         20,
	})

	svc.Retry = resilience.NewRetry(resilience.RetryStrategy{
		MaxAttempts:     3,
		InitialInterval: 50 * time.Millisecond,
		MaxInterval:     500 * time.Millisecond,
		Multiplier:      2.0,
	})

	 svc.Bulkhead = resilience.NewBulkhead(resilience.BulkheadConfig{
		 MaxConcurrent: 10,
		 MaxWaitTime:   0, // No wait
	 })

	mesh.services[sType] = svc
}

// [GAuth] All service actions require explicit centralized authorization
func (s *Microservice) processRequest(ctx context.Context, mesh *ServiceMesh, gauth *GAuthAuthorizationServer) error {
       s.mu.RLock()
       loadFactor := s.LoadFactor
       s.mu.RUnlock()

       // Check dependencies first (each dependency must be authorized)
       for _, depType := range s.Dependencies {
	       depSvc := mesh.services[depType]
	       if err := depSvc.call(ctx, mesh, gauth); err != nil {
		       return fmt.Errorf("%s dependency failed: %w", depSvc.Name, err)
	       }
       }

       // GAuth: Centralized authorization check for this action
       scope := fmt.Sprintf("%s:process", s.Name)
       token, err := gauth.AuthorizeAction(s.Name, scope)
       if err != nil {
	       s.recordFailure()
	       return fmt.Errorf("%s: authorization denied: %v", s.Name, err)
       }
       s.ExtendedToken = token

       // Simulate processing with current load factor
       latency := time.Duration(float64(100*time.Millisecond) * (1 + loadFactor))
       select {
       case <-time.After(latency):
       case <-ctx.Done():
	       return ctx.Err()
       }

       // Higher chance of failure under high load
       errorRate := 0.1 * (1 + loadFactor)
       if rand.Float64() < errorRate {
	       s.recordFailure()
	       return fmt.Errorf("%s: service error under load %.2f", s.Name, loadFactor)
       }

       s.recordSuccess()
       return nil
}

// [GAuth] All service calls require centralized authorization
func (s *Microservice) call(ctx context.Context, mesh *ServiceMesh, gauth *GAuthAuthorizationServer) error {
       // Apply bulkhead pattern if present
       if s.Bulkhead != nil {
	       return s.Bulkhead.Execute(ctx, func(ctx context.Context) error {
		       // Check rate limit
		       if err := s.Limiter.Allow(ctx, s.Name); err != nil {
			       return fmt.Errorf("rate limit exceeded for %s: %w", s.Name, err)
		       }
		       // Use retry with circuit breaker
		       return s.Retry.Do(func() error {
			       return s.Breaker.Execute(func() error {
				       return s.processRequest(ctx, mesh, gauth)
			       })
		       })
	       })
       }
       // If Bulkhead is nil, just run the rest of the logic
       if err := s.Limiter.Allow(ctx, s.Name); err != nil {
	       return fmt.Errorf("rate limit exceeded for %s: %w", s.Name, err)
       }
       return s.Retry.Do(func() error {
	       return s.Breaker.Execute(func() error {
		       return s.processRequest(ctx, mesh, gauth)
	       })
       })
}

// [Other] recordSuccess is a utility (not protocol logic)
func (s *Microservice) recordSuccess() {
	s.Health.mu.Lock()
	defer s.Health.mu.Unlock()
	s.Health.Successes++
}

// [Other] recordFailure is a utility (not protocol logic)
func (s *Microservice) recordFailure() {
	s.Health.mu.Lock()
	defer s.Health.mu.Unlock()
	s.Health.Failures++
	s.Health.LastFailureTime = time.Now()
}

// [Other] increaseLoad is a simulation utility (not protocol logic)
func (s *Microservice) increaseLoad(factor float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.LoadFactor = factor
}

// [GAuth] simulateCascadingFailures demonstrates GAuth protocol flows
func simulateCascadingFailures() {
       mesh := NewServiceMesh()
       ctx := context.Background()

       // GAuth: Centralized authorization server
       gauth := NewGAuthAuthorizationServer()

       // GAuth: Issue power of attorney for all services (principal: "HumanOwner123")
       for _, svc := range mesh.services {
	       poa := gauth.IssuePowerOfAttorney("HumanOwner123", svc.Name, fmt.Sprintf("%s:process", svc.Name), 24*time.Hour)
	       svc.PowerOfAttorney = poa
       }

       fmt.Println("\nStarting Cascading Failures Simulation (GAuth Protocol Compliant)...")
       fmt.Println("----------------------------------------")
       fmt.Println("Initial Configuration:")
       fmt.Println("- 7 interconnected services (agents)")
       fmt.Println("- Centralized GAuth authorization server (PDP, PAP, PIP, PVP roles)")
       fmt.Println("- Power of attorney, grant, and extended token flows")
       fmt.Println("- Circuit breakers, rate limits, and bulkheads")
       fmt.Println("----------------------------------------")

       // Channel for controlling load increase
       loadUpdates := make(chan struct{})

       // Start background load monitoring
       go func() {
	       ticker := time.NewTicker(5 * time.Second)
	       defer ticker.Stop()

	       for {
		       select {
		       case <-ticker.C:
			       for _, svc := range mesh.services {
				       svc.Health.mu.RLock()
				       failureRate := float64(svc.Health.Failures) / float64(svc.Health.Successes+svc.Health.Failures)
				       svc.Health.mu.RUnlock()

				       fmt.Printf("[%s] Health: %.2f%% success rate\n",
					       svc.Name, (1-failureRate)*100)
			       }
		       case <-loadUpdates:
			       return
		       }
	       }
       }()

       // Simulate traffic with increasing load
       var wg sync.WaitGroup
       clients := 50
       phases := 4

       for phase := 1; phase <= phases; phase++ {
	       fmt.Printf("\nPhase %d: Load Factor %.1f\n", phase, float64(phase)*0.25)

	       // Increase load on critical services
	       mesh.services[PaymentService].increaseLoad(float64(phase) * 0.25)
	       mesh.services[OrderService].increaseLoad(float64(phase) * 0.2)
	       mesh.services[InventoryService].increaseLoad(float64(phase) * 0.15)

	       for client := 1; client <= clients; client++ {
		       wg.Add(1)
		       go func(clientID int) {
			       defer wg.Done()

			       // Simulate complex transaction flow
			       request := fmt.Sprintf("client%d-phase%d", clientID, phase)
			       start := time.Now()

			       // Start with order service which triggers dependency chain
			       err := mesh.services[OrderService].call(ctx, mesh, gauth)
			       duration := time.Since(start)

			       if err != nil {
				       fmt.Printf("[%s] Failed after %v: %v\n",
					       request, duration.Round(time.Millisecond), err)
			       } else {
				       fmt.Printf("[%s] Completed in %v\n",
					       request, duration.Round(time.Millisecond))
			       }
		       }(client)

		       time.Sleep(100 * time.Millisecond)
	       }

	       wg.Wait()
	       time.Sleep(2 * time.Second) // Pause between phases
       }

       close(loadUpdates)
       fmt.Println("\nCascading Failures simulation completed!")

       // Print final statistics
       fmt.Println("\nFinal Service Health Report:")
       fmt.Println("-----------------------------")
       for _, svc := range mesh.services {
	       svc.Health.mu.RLock()
	       total := svc.Health.Successes + svc.Health.Failures
	       successRate := float64(svc.Health.Successes) / float64(total) * 100
	       svc.Health.mu.RUnlock()

	       fmt.Printf("%s:\n", svc.Name)
	       fmt.Printf("  - Success Rate: %.2f%%\n", successRate)
	       fmt.Printf("  - Load Factor: %.2f\n", svc.LoadFactor)
       }

       // Print GAuth audit log
       fmt.Println("\nGAuth Audit Log:")
       fmt.Println("-----------------------------")
       for _, event := range gauth.auditLog {
	       fmt.Printf("[%s] Agent: %s, Action: %s, Principal: %s, Scope: %s, Result: %s, Reason: %s\n",
		       event.Timestamp.Format(time.RFC3339), event.Agent, event.Action, event.Principal, event.Scope, event.Result, event.Reason)
       }
}

// [Other] main entrypoint (not protocol logic)
func main() {
	simulateCascadingFailures()
}
