package gauth.authz

# Default deny - fail-safe
default allow = false

# =============================================================================
# Scope Validation Policy
# =============================================================================

# Allow if all child scopes are covered by parent scopes
allow {
    input.action == "validate_scope"
    scope_subset(input.parent_scopes, input.child_scopes)
}

# Check if child is subset of parent
scope_subset(parent, child) {
    every_child_covered(parent, child)
}

# Verify each child scope is covered
every_child_covered(parent, child) {
    count([c | c := child[_]; not scope_matches_any(parent, c)]) == 0
}

# Check if scope matches any parent pattern
scope_matches_any(patterns, scope) {
    pattern := patterns[_]
    scope_matches(pattern, scope)
}

# Exact match
scope_matches(pattern, scope) {
    pattern == scope
}

# Global wildcard (dangerous - use sparingly!)
scope_matches(pattern, scope) {
    pattern == "*"
}

# Prefix wildcard (e.g., "users:*")
scope_matches(pattern, scope) {
    endswith(pattern, "*")
    prefix := trim_suffix(pattern, "*")
    startswith(scope, prefix)
}

# Suffix wildcard (e.g., "*:read")
scope_matches(pattern, scope) {
    startswith(pattern, "*")
    suffix := trim_prefix(pattern, "*")
    endswith(scope, suffix)
}

# =============================================================================
# Attribute-Based Access Control (ABAC) Examples
# =============================================================================

# Allow if user has required department and clearance
allow {
    input.action == "access_resource"
    input.user.department == input.resource.owner_department
    input.user.clearance_level >= input.resource.classification_level
}

# Allow managers to access team resources
allow {
    input.action == "access_resource"
    input.user.role == "manager"
    input.resource.team == input.user.team
}

# =============================================================================
# Time-Based Access Control
# =============================================================================

# Business hours check (9 AM - 6 PM Pacific Time)
is_business_hours {
    now := time.now_ns()
    hour := time.clock([now, "America/Los_Angeles"])[0]
    hour >= 9
    hour < 18
}

# Allow sensitive operations only during business hours
allow {
    input.action == "sensitive_operation"
    is_business_hours
    input.user.requires_mfa == true
    input.user.mfa_verified == true
}

# =============================================================================
# Multi-Tenant Isolation
# =============================================================================

# Ensure tenant isolation for data access
allow {
    input.action == "access_data"
    input.user.tenant_id == input.resource.tenant_id
    has_permission(input.user.permissions, input.resource.required_permission)
}

# Helper: Check if user has required permission
has_permission(user_perms, required) {
    user_perms[_] == required
}

# Helper: Wildcard permission check
has_permission(user_perms, _) {
    user_perms[_] == "*"
}

# =============================================================================
# Delegation Chain Validation
# =============================================================================

# Validate delegation chain integrity
allow {
    input.action == "validate_delegation_chain"
    validate_chain_structure(input.chain)
    validate_chain_scopes(input.chain)
}

# Check chain structure (grantee[n] == grantor[n+1])
validate_chain_structure(chain) {
    count([i | 
        link := chain[i]
        next_link := chain[i+1]
        link.grantee != next_link.grantor
    ]) == 0
}

# Check scope inheritance in chain
validate_chain_scopes(chain) {
    count([i |
        link := chain[i]
        next_link := chain[i+1]
        not scope_subset(link.scope, next_link.scope)
    ]) == 0
}

# =============================================================================
# Geographic Restrictions
# =============================================================================

# Allow access only from specific countries
allow {
    input.action == "access_data"
    input.resource.geo_restrictions[_] == input.request.country_code
    has_permission(input.user.permissions, input.resource.required_permission)
}

# =============================================================================
# Rate Limiting Policy
# =============================================================================

# Check if user exceeded rate limit
allow {
    input.action == "api_request"
    count(input.user.recent_requests) < input.user.rate_limit
}

# =============================================================================
# Audit Decision Details
# =============================================================================

# Provide detailed decision information for audit logs
decision_details = {
    "allow": allow,
    "reason": reason,
    "matched_rules": matched_rules,
    "timestamp": time.now_ns()
}

# Determine primary reason for decision
reason = "scope_validation_passed" {
    allow
    input.action == "validate_scope"
}

reason = "abac_passed" {
    allow
    input.action == "access_resource"
}

reason = "business_hours_check_passed" {
    allow
    is_business_hours
}

reason = "tenant_isolation_enforced" {
    allow
    input.action == "access_data"
    input.user.tenant_id == input.resource.tenant_id
}

reason = "default_deny" {
    not allow
}

# Track which rules matched (for debugging)
matched_rules[rule] {
    allow
    # This would need to be enhanced to track specific rule matches
    rule := "basic_authorization"
}
