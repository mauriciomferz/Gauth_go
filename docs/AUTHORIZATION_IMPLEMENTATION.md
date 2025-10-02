# Real Authorization Implementation

## 🛡️ **AUTHORIZATION ENGINE OVERHAUL**

### **Current State: COMPLETELY FAKE**
- Returns hardcoded "granted" responses
- No permission checking
- No role-based access control
- No policy enforcement

### **Required Implementation:**

#### **A. Role-Based Access Control (RBAC)**
```go
type RBACEngine struct {
    roleStore       RoleStore
    permissionStore PermissionStore
    userRoleStore   UserRoleStore
    policyEvaluator *PolicyEvaluator
    auditLogger     AuditLogger
}

type Role struct {
    ID          string
    Name        string
    Description string
    Permissions []Permission
    Inherits    []string // Parent roles
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type Permission struct {
    ID       string
    Resource string
    Action   string
    Effect   PermissionEffect // Allow/Deny
    Conditions []Condition
}

type AccessRequest struct {
    UserID    string
    Resource  string
    Action    string
    Context   map[string]interface{}
    Timestamp time.Time
}

func (rbac *RBACEngine) Authorize(req *AccessRequest) (*AuthorizationDecision, error) {
    // Get user roles
    userRoles, err := rbac.userRoleStore.GetUserRoles(req.UserID)
    if err != nil {
        return nil, fmt.Errorf("failed to get user roles: %w", err)
    }
    
    // Collect all permissions (including inherited)
    allPermissions := make([]Permission, 0)
    for _, roleID := range userRoles {
        permissions, err := rbac.getEffectivePermissions(roleID)
        if err != nil {
            return nil, fmt.Errorf("failed to get permissions for role %s: %w", roleID, err)
        }
        allPermissions = append(allPermissions, permissions...)
    }
    
    // Evaluate permissions
    decision := rbac.evaluatePermissions(allPermissions, req)
    
    // Log the decision
    rbac.auditLogger.LogAuthorizationDecision(req, decision)
    
    return decision, nil
}

func (rbac *RBACEngine) evaluatePermissions(permissions []Permission, req *AccessRequest) *AuthorizationDecision {
    var allowDecisions []Permission
    var denyDecisions []Permission
    
    for _, perm := range permissions {
        if rbac.matchesResourceAction(perm, req.Resource, req.Action) {
            // Evaluate conditions
            if rbac.evaluateConditions(perm.Conditions, req.Context) {
                if perm.Effect == PermissionAllow {
                    allowDecisions = append(allowDecisions, perm)
                } else {
                    denyDecisions = append(denyDecisions, perm)
                }
            }
        }
    }
    
    // Deny takes precedence
    if len(denyDecisions) > 0 {
        return &AuthorizationDecision{
            Decision: DecisionDeny,
            Reason:   "Explicit deny permission found",
            MatchingPermissions: denyDecisions,
        }
    }
    
    if len(allowDecisions) > 0 {
        return &AuthorizationDecision{
            Decision: DecisionAllow,
            Reason:   "Allow permission granted",
            MatchingPermissions: allowDecisions,
        }
    }
    
    return &AuthorizationDecision{
        Decision: DecisionDeny,
        Reason:   "No matching allow permissions found",
    }
}
```

#### **B. Attribute-Based Access Control (ABAC)**
```go
type ABACEngine struct {
    policyStore    PolicyStore
    attributeStore AttributeStore
    evaluator      *PolicyEvaluator
}

type Policy struct {
    ID          string
    Name        string
    Description string
    Rules       []PolicyRule
    Effect      PolicyEffect
    Version     string
    Active      bool
}

type PolicyRule struct {
    Condition   string // XACML-like expression
    Target      Target
    Effect      PolicyEffect
    Obligations []Obligation
}

type Target struct {
    Subjects  []AttributeMatch
    Resources []AttributeMatch
    Actions   []AttributeMatch
    Environment []AttributeMatch
}

type AttributeMatch struct {
    AttributeID string
    Match       MatchType
    Value       interface{}
}

func (abac *ABACEngine) Evaluate(request *AuthorizationRequest) (*PolicyDecision, error) {
    // Get applicable policies
    policies, err := abac.policyStore.GetApplicablePolicies(request)
    if err != nil {
        return nil, fmt.Errorf("failed to get applicable policies: %w", err)
    }
    
    var decisions []*PolicyDecision
    
    for _, policy := range policies {
        decision, err := abac.evaluatePolicy(policy, request)
        if err != nil {
            continue // Log error but continue evaluation
        }
        decisions = append(decisions, decision)
    }
    
    // Combine decisions using policy combining algorithm
    return abac.combineDecisions(decisions), nil
}

func (abac *ABACEngine) evaluatePolicy(policy *Policy, request *AuthorizationRequest) (*PolicyDecision, error) {
    for _, rule := range policy.Rules {
        // Check if rule target matches request
        if !abac.matchesTarget(rule.Target, request) {
            continue
        }
        
        // Evaluate rule condition
        result, err := abac.evaluator.EvaluateCondition(rule.Condition, request.Attributes)
        if err != nil {
            return nil, fmt.Errorf("condition evaluation failed: %w", err)
        }
        
        if result {
            return &PolicyDecision{
                Decision:    rule.Effect,
                PolicyID:    policy.ID,
                RuleID:      rule.Condition,
                Obligations: rule.Obligations,
            }, nil
        }
    }
    
    return &PolicyDecision{
        Decision: PolicyNotApplicable,
        PolicyID: policy.ID,
    }, nil
}
```

#### **C. Policy Decision Point (PDP)**
```go
type PolicyDecisionPoint struct {
    rbacEngine    *RBACEngine
    abacEngine    *ABACEngine
    combiningAlg  CombiningAlgorithm
    obligations   ObligationService
    advices       AdviceService
}

func (pdp *PolicyDecisionPoint) MakeDecision(request *AuthorizationRequest) (*AuthorizationResponse, error) {
    // Validate request
    if err := pdp.validateRequest(request); err != nil {
        return nil, fmt.Errorf("invalid request: %w", err)
    }
    
    // Evaluate RBAC
    rbacDecision, err := pdp.rbacEngine.Authorize(&AccessRequest{
        UserID:   request.Subject.UserID,
        Resource: request.Resource.ID,
        Action:   request.Action.ID,
        Context:  request.Environment,
    })
    if err != nil {
        return nil, fmt.Errorf("RBAC evaluation failed: %w", err)
    }
    
    // Evaluate ABAC
    abacDecision, err := pdp.abacEngine.Evaluate(request)
    if err != nil {
        return nil, fmt.Errorf("ABAC evaluation failed: %w", err)
    }
    
    // Combine decisions
    finalDecision := pdp.combineDecisions(rbacDecision, abacDecision)
    
    // Process obligations
    obligations, err := pdp.obligations.ProcessObligations(finalDecision.Obligations, request)
    if err != nil {
        return nil, fmt.Errorf("obligation processing failed: %w", err)
    }
    
    return &AuthorizationResponse{
        Decision:    finalDecision.Decision,
        Reason:      finalDecision.Reason,
        Obligations: obligations,
        Advice:      pdp.advices.GenerateAdvice(finalDecision, request),
        Timestamp:   time.Now(),
    }, nil
}
```

#### **D. Dynamic Policy Evaluation**
```go
type PolicyEvaluator struct {
    expressionEngine ExpressionEngine
    functions        map[string]Function
    cache           *sync.Map
}

// Evaluate complex policy expressions
func (pe *PolicyEvaluator) EvaluateCondition(condition string, attributes map[string]interface{}) (bool, error) {
    // Check cache first
    cacheKey := fmt.Sprintf("%s:%x", condition, hash(attributes))
    if cached, ok := pe.cache.Load(cacheKey); ok {
        return cached.(bool), nil
    }
    
    // Parse expression
    expr, err := pe.expressionEngine.Parse(condition)
    if err != nil {
        return false, fmt.Errorf("expression parsing failed: %w", err)
    }
    
    // Create evaluation context
    ctx := &EvaluationContext{
        Attributes: attributes,
        Functions:  pe.functions,
        Timestamp:  time.Now(),
    }
    
    // Evaluate
    result, err := expr.Evaluate(ctx)
    if err != nil {
        return false, fmt.Errorf("expression evaluation failed: %w", err)
    }
    
    boolResult, ok := result.(bool)
    if !ok {
        return false, fmt.Errorf("expression must evaluate to boolean, got %T", result)
    }
    
    // Cache result
    pe.cache.Store(cacheKey, boolResult)
    
    return boolResult, nil
}

// Built-in policy functions
func (pe *PolicyEvaluator) initializeFunctions() {
    pe.functions = map[string]Function{
        "hasRole": func(args []interface{}) (interface{}, error) {
            if len(args) != 2 {
                return false, fmt.Errorf("hasRole requires 2 arguments")
            }
            userRoles, ok := args[0].([]string)
            if !ok {
                return false, fmt.Errorf("first argument must be []string")
            }
            requiredRole, ok := args[1].(string)
            if !ok {
                return false, fmt.Errorf("second argument must be string")
            }
            
            for _, role := range userRoles {
                if role == requiredRole {
                    return true, nil
                }
            }
            return false, nil
        },
        
        "inTimeRange": func(args []interface{}) (interface{}, error) {
            // Implementation for time-based access control
            now := time.Now()
            startTime, _ := args[0].(time.Time)
            endTime, _ := args[1].(time.Time)
            return now.After(startTime) && now.Before(endTime), nil
        },
        
        "ipInRange": func(args []interface{}) (interface{}, error) {
            // Implementation for IP-based access control
            clientIP := args[0].(string)
            allowedRange := args[1].(string)
            _, ipNet, err := net.ParseCIDR(allowedRange)
            if err != nil {
                return false, err
            }
            ip := net.ParseIP(clientIP)
            return ipNet.Contains(ip), nil
        },
    }
}
```

### **Implementation Complexity: EXTREMELY HIGH**
- **Time Estimate**: 10-16 weeks
- **Required Skills**: Authorization systems, policy languages, distributed systems
- **Performance Requirements**: Sub-millisecond decision times
- **Scalability**: Handle millions of authorization requests/second
- **Testing**: Extensive policy testing framework required

### **Critical Features Required:**
1. **Policy Versioning and Rollback**
2. **Real-time Policy Updates**
3. **Decision Caching and Invalidation**
4. **Performance Monitoring**
5. **Policy Conflict Detection**
6. **Distributed Decision Points**
7. **Integration with External Attribute Stores**