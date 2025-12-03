// TypeScript interfaces for Admin Handler API responses
// These match the backend Go structures

// ============================================================================
// Power of Attorney
// ============================================================================

export interface PowerOfAttorney {
  id: string;
  principalId: string;
  principalName: string;
  representativeId: string;
  representativeName: string;
  representativeType: string;
  status: 'active' | 'revoked' | 'expired' | 'pending';
  validFrom: string; // ISO8601
  validUntil: string; // ISO8601
  actions: string[];
  resources: string[];
  geoRestrictions: string[];
  approvalStatus: 'pending' | 'approved' | 'rejected';
  createdAt: string; // ISO8601
  updatedAt?: string; // ISO8601
}

export interface PoAListResponse {
  powerOfAttorneys: PowerOfAttorney[];
  total: number;
}

export interface PoACreateRequest {
  principalId: string;
  principalName: string;
  representativeId: string;
  representativeName: string;
  representativeType: string;
  validFrom: string;
  validUntil: string;
  actions: string[];
  resources?: string[];
  geoRestrictions?: string[];
}

// ============================================================================
// Resilience Patterns
// ============================================================================

export interface CircuitBreaker {
  id: string;
  name: string;
  service: string;
  state: 'open' | 'closed' | 'half-open';
  failureThreshold: number;
  successThreshold: number;
  timeout: number; // milliseconds
  failures: number;
  successes: number;
  lastStateChange: string; // ISO8601
  totalRequests: number;
  failureRate: number;
}

export interface CircuitBreakerListResponse {
  circuitBreakers: CircuitBreaker[];
  total: number;
}

export interface CircuitBreakerCreateRequest {
  name: string;
  service: string;
  failureThreshold: number;
  successThreshold: number;
  timeout: number; // milliseconds, minimum 1000
}

export interface RateLimiter {
  id: string;
  name: string;
  resource: string;
  algorithm: 'token-bucket' | 'leaky-bucket' | 'fixed-window' | 'sliding-window';
  limit: number;
  window: number; // seconds
  burst?: number;
  currentUsage: number;
  resetTime: string; // ISO8601
}

export interface RateLimiterListResponse {
  rateLimiters: RateLimiter[];
  total: number;
}

export interface RateLimiterCreateRequest {
  name: string;
  resource: string;
  algorithm: 'token-bucket' | 'leaky-bucket' | 'fixed-window' | 'sliding-window';
  limit: number;
  window: number;
  burst?: number;
}

export interface RetryPolicy {
  id: string;
  name: string;
  operation: string;
  strategy: 'fixed' | 'exponential' | 'fibonacci' | 'linear';
  maxAttempts: number;
  baseDelay: number; // milliseconds
  maxDelay: number; // milliseconds
  jitter: boolean;
}

export interface RetryPolicyListResponse {
  retryPolicies: RetryPolicy[];
  total: number;
}

export interface RetryPolicyCreateRequest {
  name: string;
  operation: string;
  strategy: 'fixed' | 'exponential' | 'fibonacci' | 'linear';
  maxAttempts: number; // 1-10
  baseDelay: number; // minimum 100ms
  maxDelay: number;
  jitter: boolean;
}

export interface Bulkhead {
  id: string;
  name: string;
  service: string;
  maxConcurrency: number;
  maxQueueSize: number;
  timeout: number; // milliseconds
  currentConcurrency: number;
  queuedRequests: number;
}

export interface BulkheadListResponse {
  bulkheads: Bulkhead[];
  total: number;
}

// ============================================================================
// Event System
// ============================================================================

export interface Event {
  id: string;
  type: string;
  category: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
  source: string;
  timestamp: string; // ISO8601
  data?: Record<string, any>;
  userId?: string;
  ipAddress?: string;
  userAgent?: string;
}

export interface EventListResponse {
  events: Event[];
  total: number;
}

export interface EventType {
  id: string;
  name: string;
  description: string;
  category: string;
  severity: 'info' | 'warning' | 'error' | 'critical';
  schema?: Record<string, any>;
}

export interface EventTypeListResponse {
  eventTypes: EventType[];
  total: number;
}

export interface EventHandler {
  id: string;
  name: string;
  eventType: string;
  endpoint: string;
  method: 'POST' | 'PUT' | 'PATCH';
  enabled: boolean;
  retryPolicy?: string;
}

export interface EventHandlerListResponse {
  eventHandlers: EventHandler[];
  total: number;
}

// ============================================================================
// Authorization Engine
// ============================================================================

export interface AuthorizationPolicy {
  id: string;
  name: string;
  description: string;
  status: 'draft' | 'active' | 'inactive' | 'archived';
  effect: 'allow' | 'deny';
  actions: string[];
  resources: string[];
  conditions?: Record<string, any>;
  priority?: number;
  enabled?: boolean;
  version?: number;
  createdAt: string; // ISO8601
  updatedAt: string; // ISO8601
  createdBy?: string;
  validFrom?: string; // ISO8601
  validUntil?: string; // ISO8601
}

export interface PolicyListResponse {
  policies: AuthorizationPolicy[];
  total: number;
}

export interface PolicyCreateRequest {
  name: string;
  description: string;
  effect: 'allow' | 'deny';
  actions: string[];
  resources: string[];
  conditions?: Record<string, any>;
  priority?: number;
  enabled?: boolean;
}

export interface Role {
  id: string;
  name: string;
  description: string;
  permissions: string[];
  policies: string[];
}

export interface RoleListResponse {
  roles: Role[];
  total: number;
}

// ============================================================================
// Configuration Management
// ============================================================================

export interface ConfigVariable {
  key: string;
  value: string;
  type: 'string' | 'integer' | 'boolean' | 'json';
  sensitive: boolean;
  description: string;
  createdAt?: string; // ISO8601
  updatedAt?: string; // ISO8601
}

export interface VariableListResponse {
  variables: ConfigVariable[];
}

export interface VariableCreateRequest {
  key: string;
  value: string;
  type: 'string' | 'integer' | 'boolean' | 'json';
  sensitive: boolean;
  description: string;
}

export interface FeatureFlag {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  type: 'boolean' | 'percentage' | 'user-list';
  percentage?: number;
  targetTenants?: string[];
  targetUsers?: string[];
  createdAt: string; // ISO8601
  updatedAt: string; // ISO8601
}

export interface FeatureFlagListResponse {
  flags: FeatureFlag[];
}

export interface FeatureFlagCreateRequest {
  name: string;
  description: string;
  enabled: boolean;
  type: 'boolean' | 'percentage' | 'user-list';
  percentage?: number;
  targetTenants?: string[];
}

// ============================================================================
// Common Types
// ============================================================================

export interface PaginationParams {
  page?: number;
  limit?: number;
  offset?: number;
}

export interface FilterParams {
  status?: string;
  type?: string;
  search?: string;
  sortBy?: string;
  sortOrder?: 'asc' | 'desc';
}

export interface ApiError {
  error: string;
  message?: string;
  details?: any;
}

export interface ApiSuccess {
  message: string;
  data?: any;
}
