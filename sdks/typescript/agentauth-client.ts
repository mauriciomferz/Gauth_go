/**
 * AgentAuth TypeScript/JavaScript SDK
 * 
 * Official client library for the AgentAuth OAuth 2.0 Authorization Server
 * Supports RFC-0111 subscription flow, Power of Attorney, and more.
 * 
 * @version 1.0.0-beta
 * @author AgentAuth Team
 * @license MIT
 */

// ============================================================================
// Types and Interfaces
// ============================================================================

export interface AgentAuthConfig {
  baseURL: string;
  apiKey?: string;
  accessToken?: string;
  timeout?: number;
}

export interface Subscription {
  id: string;
  client_id: string;
  scope: string;
  status: 'initiated' | 'in_progress' | 'completed' | 'failed';
  current_step: 'I' | 'II' | 'III' | 'IV' | 'V' | 'VI' | 'VII' | 'VIII';
  created_at: string;
  updated_at: string;
  completed_at?: string;
}

export interface SubscriptionCreate {
  client_id: string;
  scope: string;
  redirect_uri?: string;
  state?: string;
}

export interface StepResult {
  subscription_id: string;
  step: string;
  status: 'success' | 'failed';
  message: string;
  completed_at: string;
}

export interface PoA {
  id: string;
  grantor: string;
  grantee: string;
  scope: string[];
  valid_from: string;
  valid_until: string;
  resource_pattern?: string;
  max_uses?: number;
  status: 'active' | 'expired' | 'revoked';
  created_at: string;
  updated_at: string;
  revoked_at?: string;
  use_count: number;
}

export interface PoACreate {
  grantor: string;
  grantee: string;
  scope: string[];
  valid_from: string;
  valid_until: string;
  resource_pattern?: string;
  max_uses?: number;
  constraints?: {
    ip_whitelist?: string[];
    time_windows?: string[];
  };
}

export interface PoAValidateRequest {
  action: string;
  resource?: string;
  context?: {
    ip_address?: string;
    user_agent?: string;
    timestamp?: string;
  };
}

export interface PoAValidateResponse {
  valid: boolean;
  poa_id: string;
  action: string;
  reason?: string;
  validated_at: string;
}

export interface PVPVerifyRequest {
  subject: string;
  proof_type: 'document' | 'biometric' | 'challenge';
  proof_data: string;
}

export interface PVPVerifyResponse {
  valid: boolean;
  subject: string;
  verified_at: string;
  proof_type: string;
  confidence_score: number;
}

export interface AuthzEvaluateRequest {
  subject: string;
  action: string;
  resource: string;
  context?: Record<string, any>;
}

export interface AuthzEvaluateResponse {
  decision: 'permit' | 'deny';
  policy_id: string;
  reason: string;
  evaluated_at: string;
  cache_hit: boolean;
}

export interface Token {
  access_token: string;
  token_type: string;
  expires_in: number;
  scope: string;
  issued_at: string;
}

export interface APIError {
  error: string;
  error_description: string;
  error_uri?: string;
  timestamp: string;
}

// ============================================================================
// Main Client Class
// ============================================================================

export class AgentAuthClient {
  private config: Required<AgentAuthConfig>;

  constructor(config: AgentAuthConfig) {
    this.config = {
      baseURL: config.baseURL,
      apiKey: config.apiKey || '',
      accessToken: config.accessToken || '',
      timeout: config.timeout || 30000,
    };
  }

  /**
   * Set the access token for authenticated requests
   */
  setAccessToken(token: string): void {
    this.config.accessToken = token;
  }

  /**
   * Set the API key for authenticated requests
   */
  setAPIKey(apiKey: string): void {
    this.config.apiKey = apiKey;
  }

  // ==========================================================================
  // Private Helper Methods
  // ==========================================================================

  private async request<T>(
    method: string,
    path: string,
    data?: any,
    options?: RequestInit
  ): Promise<T> {
    const url = `${this.config.baseURL}${path}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...((options?.headers as Record<string, string>) || {}),
    };

    if (this.config.accessToken) {
      headers['Authorization'] = `Bearer ${this.config.accessToken}`;
    }

    if (this.config.apiKey) {
      headers['X-API-Key'] = this.config.apiKey;
    }

    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

    try {
      const response = await fetch(url, {
        method,
        headers,
        body: data ? JSON.stringify(data) : undefined,
        signal: controller.signal,
        ...options,
      });

      clearTimeout(timeoutId);

      if (!response.ok) {
        const error: APIError = await response.json().catch(() => ({
          error: 'unknown_error',
          error_description: `HTTP ${response.status}: ${response.statusText}`,
          timestamp: new Date().toISOString(),
        }));
        throw new AgentAuthError(error);
      }

      return await response.json();
    } catch (error) {
      clearTimeout(timeoutId);
      if (error instanceof AgentAuthError) {
        throw error;
      }
      throw new Error(`Request failed: ${error instanceof Error ? error.message : 'Unknown error'}`);
    }
  }

  // ==========================================================================
  // RFC-0111 Subscription Flow Methods
  // ==========================================================================

  /**
   * Create a new subscription (Step I)
   */
  async createSubscription(data: SubscriptionCreate): Promise<Subscription> {
    return this.request<Subscription>('POST', '/api/v1/rfc0111/subscriptions', data);
  }

  /**
   * Get subscription details
   */
  async getSubscription(id: string): Promise<Subscription> {
    return this.request<Subscription>('GET', `/api/v1/rfc0111/subscriptions/${id}`);
  }

  /**
   * List subscriptions with optional filters
   */
  async listSubscriptions(filters?: {
    client_id?: string;
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ subscriptions: Subscription[]; total: number }> {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    const path = `/api/v1/rfc0111/subscriptions${params.toString() ? `?${params}` : ''}`;
    return this.request<{ subscriptions: Subscription[]; total: number }>('GET', path);
  }

  /**
   * Execute Step II - Authorizer Authentication (PVP)
   */
  async executeStepII(id: string, proof: Omit<PVPVerifyRequest, 'subject'>): Promise<StepResult> {
    return this.request<StepResult>('POST', `/api/v1/rfc0111/subscriptions/${id}/step-ii`, proof);
  }

  /**
   * Execute Step III - Client Owner Identification
   */
  async executeStepIII(id: string): Promise<StepResult> {
    return this.request<StepResult>('POST', `/api/v1/rfc0111/subscriptions/${id}/step-iii`);
  }

  /**
   * Execute Step IV - Client Owner Authorization
   */
  async executeStepIV(id: string): Promise<StepResult> {
    return this.request<StepResult>('POST', `/api/v1/rfc0111/subscriptions/${id}/step-iv`);
  }

  /**
   * Execute Step V - Client Authorization
   */
  async executeStepV(id: string): Promise<StepResult> {
    return this.request<StepResult>('POST', `/api/v1/rfc0111/subscriptions/${id}/step-v`);
  }

  /**
   * Execute Step VI - Resource Owner Identification
   */
  async executeStepVI(id: string): Promise<StepResult> {
    return this.request<StepResult>('POST', `/api/v1/rfc0111/subscriptions/${id}/step-vi`);
  }

  /**
   * Execute Step VII - Resource Owner Authorization
   */
  async executeStepVII(id: string): Promise<StepResult> {
    return this.request<StepResult>('POST', `/api/v1/rfc0111/subscriptions/${id}/step-vii`);
  }

  /**
   * Execute Step VIII - Resource Server Verification (Final Step)
   * Returns the access token upon successful completion
   */
  async executeStepVIII(id: string): Promise<{ status: string; token: string; token_type: string; expires_in: number }> {
    return this.request<{ status: string; token: string; token_type: string; expires_in: number }>(
      'POST',
      `/api/v1/rfc0111/subscriptions/${id}/step-viii`
    );
  }

  /**
   * Complete entire RFC-0111 subscription flow automatically
   * Executes all 8 steps in sequence
   */
  async completeSubscriptionFlow(data: SubscriptionCreate): Promise<{ subscription: Subscription; token: string }> {
    // Step I: Create subscription
    const subscription = await this.createSubscription(data);
    
    // Steps II-VIII: Execute automatically
    await this.executeStepII(subscription.id, {
      proof_type: 'document',
      proof_data: 'auto_verified',
    });
    await this.executeStepIII(subscription.id);
    await this.executeStepIV(subscription.id);
    await this.executeStepV(subscription.id);
    await this.executeStepVI(subscription.id);
    await this.executeStepVII(subscription.id);
    
    // Step VIII: Get token
    const result = await this.executeStepVIII(subscription.id);
    
    // Update access token
    this.setAccessToken(result.token);
    
    return {
      subscription: await this.getSubscription(subscription.id),
      token: result.token,
    };
  }

  // ==========================================================================
  // Power of Attorney (PoA) Methods
  // ==========================================================================

  /**
   * Create a new Power of Attorney
   */
  async createPoA(data: PoACreate): Promise<PoA> {
    return this.request<PoA>('POST', '/api/v1/beta/poa', data);
  }

  /**
   * Get PoA details
   */
  async getPoA(id: string): Promise<PoA> {
    return this.request<PoA>('GET', `/api/v1/beta/poa/${id}`);
  }

  /**
   * List Powers of Attorney with optional filters
   */
  async listPoAs(filters?: {
    grantor?: string;
    grantee?: string;
    status?: string;
    limit?: number;
    offset?: number;
  }): Promise<{ poas: PoA[]; total: number }> {
    const params = new URLSearchParams();
    if (filters) {
      Object.entries(filters).forEach(([key, value]) => {
        if (value !== undefined) params.append(key, String(value));
      });
    }
    const path = `/api/v1/beta/poa${params.toString() ? `?${params}` : ''}`;
    return this.request<{ poas: PoA[]; total: number }>('GET', path);
  }

  /**
   * Update a Power of Attorney
   */
  async updatePoA(id: string, data: Partial<Pick<PoA, 'scope' | 'valid_until'>>): Promise<PoA> {
    return this.request<PoA>('PUT', `/api/v1/beta/poa/${id}`, data);
  }

  /**
   * Revoke a Power of Attorney
   */
  async revokePoA(id: string): Promise<{ id: string; status: string; revoked_at: string }> {
    return this.request<{ id: string; status: string; revoked_at: string }>('DELETE', `/api/v1/beta/poa/${id}`);
  }

  /**
   * Validate a Power of Attorney for a specific action
   */
  async validatePoA(id: string, data: PoAValidateRequest): Promise<PoAValidateResponse> {
    return this.request<PoAValidateResponse>('POST', `/api/v1/beta/poa/${id}/validate`, data);
  }

  // ==========================================================================
  // PVP (Person Verification Protocol) Methods
  // ==========================================================================

  /**
   * Verify identity via PVP
   */
  async verifyPVP(data: PVPVerifyRequest): Promise<PVPVerifyResponse> {
    return this.request<PVPVerifyResponse>('POST', '/api/v1/beta/pvp/verify', data);
  }

  // ==========================================================================
  // Commercial Registry Methods
  // ==========================================================================

  /**
   * Verify a commercial entity
   */
  async verifyEntity(data: {
    entity_id: string;
    registry: string;
  }): Promise<{
    valid: boolean;
    entity_id: string;
    registry: string;
    entity_name: string;
    status: string;
    verified_at: string;
  }> {
    return this.request('POST', '/api/v1/beta/registry/verify-entity', data);
  }

  /**
   * Verify signatory authority
   */
  async verifySignatory(data: {
    signatory_id: string;
    entity_id: string;
    role: string;
  }): Promise<{
    valid: boolean;
    signatory_id: string;
    entity_id: string;
    role: string;
    authority_level: string;
    verified_at: string;
  }> {
    return this.request('POST', '/api/v1/beta/registry/verify-signatory', data);
  }

  // ==========================================================================
  // Authorization Methods
  // ==========================================================================

  /**
   * Evaluate an authorization request
   */
  async evaluateAuthorization(data: AuthzEvaluateRequest): Promise<AuthzEvaluateResponse> {
    return this.request<AuthzEvaluateResponse>('POST', '/api/v1/beta/authz/evaluate', data);
  }

  /**
   * Get authorization metrics
   */
  async getAuthzMetrics(): Promise<any> {
    return this.request('GET', '/api/v1/beta/authz/metrics');
  }

  // ==========================================================================
  // Token Methods
  // ==========================================================================

  /**
   * Create a new token
   */
  async createToken(data: {
    client_id: string;
    scope: string;
    expires_in?: number;
  }): Promise<Token> {
    return this.request<Token>('POST', '/api/v1/token/create', data);
  }

  /**
   * Validate a token
   */
  async validateToken(token: string): Promise<{
    valid: boolean;
    claims: Record<string, any>;
    expires_at: string;
  }> {
    return this.request('POST', '/api/v1/token/validate', { token });
  }

  /**
   * Revoke a token
   */
  async revokeToken(token: string): Promise<{
    revoked: boolean;
    revoked_at: string;
  }> {
    return this.request('POST', '/api/v1/token/revoke', { token });
  }

  // ==========================================================================
  // System Methods
  // ==========================================================================

  /**
   * Health check
   */
  async health(): Promise<{ status: string; timestamp: string; version: string }> {
    return this.request('GET', '/api/v1/beta/health');
  }

  /**
   * Get server information
   */
  async info(): Promise<any> {
    return this.request('GET', '/api/v1/beta/info');
  }

  /**
   * Ping endpoint
   */
  async ping(): Promise<{ message: string }> {
    return this.request('GET', '/api/v1/beta/ping');
  }
}

// ============================================================================
// Error Handling
// ============================================================================

export class AgentAuthError extends Error {
  public readonly error: string;
  public readonly error_description: string;
  public readonly error_uri?: string;
  public readonly timestamp: string;

  constructor(apiError: APIError) {
    super(apiError.error_description);
    this.name = 'AgentAuthError';
    this.error = apiError.error;
    this.error_description = apiError.error_description;
    this.error_uri = apiError.error_uri;
    this.timestamp = apiError.timestamp;
    
    // Maintain proper prototype chain
    Object.setPrototypeOf(this, AgentAuthError.prototype);
  }
}

// ============================================================================
// Exports
// ============================================================================

export default AgentAuthClient;
