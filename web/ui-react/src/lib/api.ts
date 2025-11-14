import axios, { AxiosInstance, AxiosError } from 'axios'

// Frontend API base resolves from VITE_API_BASE_URL. In local dev we prefer a relative
// path ("/api/v1") so the Vite proxy forwards requests to the backend avoiding CORS.
// If a fully qualified URL is provided (e.g. tunnel) we will use that directly.
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const RAW_BASE: string | undefined = (import.meta as any)?.env?.VITE_API_BASE_URL
// Default to relative prefix enabling proxy usage.
const API_BASE_URL = RAW_BASE && RAW_BASE.trim() !== '' ? RAW_BASE : '/api/v1'

class ApiClient {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      // When API_BASE_URL is relative the browser will issue same-origin requests to Vite dev server
      // which proxies /api -> backend. In production you may build with an absolute URL.
      baseURL: API_BASE_URL,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    this.client.interceptors.response.use(
      (response) => response,
      (error: AxiosError) => {
        console.error('API Error:', error)
        console.error('Request URL:', error.config?.url)
        console.error('Request Method:', error.config?.method)
        console.error('Request Data:', error.config?.data)
        console.error('Response Status:', error.response?.status)
        console.error('Response Data:', error.response?.data)
        throw error
      }
    )
  }

  // Authentication APIs (beta / placeholder)
  async initiateLogin(data: LoginRequest): Promise<LoginInitResponse> {
    // Placeholder endpoint; adapt to real auth service
    const response = await this.client.post('/beta/auth/login/init', data)
    return response.data
  }

  async verifyMFA(data: MFAVerifyRequest): Promise<MFAResult> {
    const response = await this.client.post('/beta/auth/login/mfa', data)
    return response.data
  }

  // Token APIs
  async createToken(data: CreateTokenRequest): Promise<TokenResponse> {
    const response = await this.client.post('/token/create', data)
    // Backend returns { success: true, token: {...}, jwt: "..." }
    // Extract the token object and transform to our interface
    const backendToken = response.data.token || response.data
    return {
      token: response.data.jwt || backendToken.token || backendToken.id,
      clientId: data.clientId,
      expiresAt: backendToken.expires_at || backendToken.expiresAt,
      scope: data.scope,
      authorizationChain: {
        ownersAuthorizer: data.ownersAuthorizer,
        clientOwner: data.clientOwner,
        client: data.clientId
      }
    }
  }

  async validateToken(token: string): Promise<TokenValidationResponse> {
    const response = await this.client.post('/token/validate', { token })
    const data = response.data
    
    // Transform backend response to frontend expected format
    return {
      valid: data.success || data.status === 'valid' || data.status === 'valid_jwt',
      decoded: data.token || undefined,
      checks: {
        status: data.status || 'unknown',
        success: String(data.success || false)
      },
      error: data.message || undefined
    }
  }
  
  async revokeToken(token: string): Promise<{ success: boolean }> {
    const response = await this.client.post('/token/revoke', { token })
    return response.data
  }
  
  async introspectToken(token: string): Promise<any> {
    const response = await this.client.post('/token/introspect', { token })
    return response.data
  }
  
  async getTokenMetrics(): Promise<any> {
    const response = await this.client.get('/token/metrics')
    return response.data
  }

  // Revocation API
  async getRevocationHead(): Promise<RevocationHead> {
    const response = await this.client.get('/token/revocation/head')
    return response.data
  }

  // Rotation APIs
  async getRotationSummary(): Promise<RotationSummary> {
    const response = await this.client.get('/rotation/summary')
    return response.data
  }

  // Capability APIs
  async getCapabilityAnchor(): Promise<CapabilityAnchor> {
    const response = await this.client.get('/capability/anchor/latest')
    return response.data
  }

  // Error Catalog
  async getErrorCatalog(): Promise<ErrorCatalog> {
    const response = await this.client.get('/errors/catalog')
    return response.data
  }

  // Algorithms
  async getAlgorithms(): Promise<AlgorithmsResponse> {
    const response = await this.client.get('/crypto/algorithms')
    return response.data
  }

  // Authorization & Policy APIs (using beta endpoints)
  async checkAuthorization(data: AuthorizationRequest): Promise<AuthorizationResponse> {
    // Mock authorization check since PoA endpoint requires different payload structure
    // In production, this would call a proper authorization service
    const allowed = Math.random() > 0.3 // 70% approval rate for demo
    const evaluationTime = Math.floor(Math.random() * 50 + 10)
    
    return Promise.resolve({
      authorized: allowed,
      allowed: allowed,
      clientId: data.clientId,
      action: data.action,
      geographicScope: data.geographic,
      industrySector: data.sector || 'general',
      cacheHit: Math.random() > 0.5,
      processingTime: `${evaluationTime}ms`,
      evaluationTime,
      policies: [
        'Resource Access Policy v1.2',
        'Geographic Restriction Policy',
        'Rate Limiting Policy',
        'Data Protection Policy (GDPR)'
      ],
      policyChecks: [
        {
          policy: 'resource_access_policy',
          result: allowed ? 'allow' : 'deny'
        },
        {
          policy: 'geographic_restriction_policy',
          result: 'allow'
        }
      ]
    })
  }

  async getAuthzMetrics(): Promise<any> {
    const response = await this.client.get('/poa/metrics')
    return response.data
  }
  
  async getDecisionMetrics(): Promise<any> {
    const response = await this.client.get('/token/metrics')
    return response.data
  }

  // Policy APIs
  async evaluatePolicy(data: any): Promise<any> {
    // Use PoA authorization which includes policy evaluation
    const response = await this.client.post('/poa/authorize', data)
    return response.data
  }

  async getPolicyMetrics(): Promise<any> {
    const response = await this.client.get('/poa/metrics')
    return response.data
  }

  // Capability APIs
  async getCapabilities(): Promise<any> {
    const response = await this.client.get('/capabilities/diff')
    return response.data
  }

  // Audit APIs
  async getAuditLogs(params?: any): Promise<any> {
    const response = await this.client.get('/audit/logs', { params })
    return response.data
  }

  async getAuditEntries(): Promise<any> {
    const response = await this.client.get('/audit/logs')
    return response.data
  }

  // Delegation APIs
  async createDelegation(data: any): Promise<any> {
    // Mock implementation for PoA creation
    // Generate unique delegation ID
    const delegationId = `del_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
    
    // Store the delegation for validation later
    const delegation = {
      delegation_id: delegationId,
      grantor: data.grantor,
      representative: data.representative,
      representativeType: data.representativeType,
      actions: data.actions || [],
      geoRestrictions: data.geoRestrictions || [],
      validityPeriod: data.validityPeriod,
      status: 'active',
      createdAt: new Date().toISOString()
    }
    
    // Store in session storage for validation
    const stored = sessionStorage.getItem('delegations') || '[]'
    const delegations = JSON.parse(stored)
    delegations.push(delegation)
    sessionStorage.setItem('delegations', JSON.stringify(delegations))
    
    console.log('Created delegation:', delegation)
    
    // Return mock success response
    return {
      success: true,
      delegation_id: delegationId,
      poaId: delegationId,
      status: 'active',
      grantor: data.grantor,
      representative: data.representative,
      actions: data.actions,
      validUntil: new Date(Date.now() + (data.validityPeriod || 365) * 24 * 60 * 60 * 1000).toISOString()
    }
  }

  async revokeDelegation(data: any): Promise<any> {
    const response = await this.client.post('/delegation/revoke', data)
    return response.data
  }

  // Event APIs
  async emitEvent(data: any): Promise<any> {
    const response = await this.client.post('/events/emit', data)
    return response.data
  }

  async getEventStream(): Promise<any> {
    const response = await this.client.get('/events/stream')
    return response.data
  }

  // PVP APIs (placeholder - need to check if these exist)
  async verifyIdentity(data: VerifyIdentityRequest): Promise<IdentityVerificationResponse> {
    // Mock identity verification since there's no dedicated endpoint
    // In production, this would integrate with actual PVP/eIDAS systems
    
    // Simulate verification logic based on entity ID format
    const isLegalEntity = /^(HRB|GmbH|AG|Ltd|Inc|LLC)/i.test(data.entityId)
    const hasValidFormat = /^[A-Z0-9-]+$/i.test(data.entityId)
    const minLength = data.entityId.length >= 5
    
    // Verification succeeds if:
    // - Legal Entity: Starts with registry prefix (HRB, etc.) and has valid format
    // - Individual: Has valid format and minimum length
    const verified = hasValidFormat && minLength && (
      (data.type === 'legal_entity' && isLegalEntity) ||
      (data.type === 'individual' && !isLegalEntity && data.entityId.length >= 8)
    )
    
    return Promise.resolve({
      verified,
      identityType: data.type,
      trustLevel: verified ? data.trustLevel : 'none',
      entityId: data.entityId,
      tsp: data.tsp,
      tspStatus: verified ? 'qualified' : 'unverified',
      verificationTime: new Date().toISOString(),
      cryptographicBinding: verified ? 'strong' : 'none'
    })
  }

  // Commercial Registry APIs (placeholder)
  async verifyEntity(data: VerifyEntityRequest): Promise<EntityVerificationResponse> {
    // Mock entity verification since there's no dedicated endpoint
    // In production, this would call a real commercial registry API
    return Promise.resolve({
      verified: true,
      registrationNumber: data.registrationNumber,
      legalName: 'Demo Corporation GmbH',
      jurisdiction: data.jurisdiction,
      status: 'active',
      registrationDate: '2020-01-15',
      legalForm: 'GmbH',
      managingDirectors: [
        {
          name: 'Dr. Jane Smith',
          position: 'Managing Director',
          authority: 'full'
        }
      ]
    })
  }

  async verifySignatory(data: VerifySignatoryRequest): Promise<SignatoryVerificationResponse> {
    // Mock signatory verification since there's no dedicated endpoint
    // In production, this would call a real commercial registry API
    return Promise.resolve({
      authorized: true,
      signatoryName: data.signatoryName,
      entity: data.entity,
      authorityType: data.authorityType,
      appointmentDate: '2020-01-15',
      restrictions: 'none',
      status: 'active'
    })
  }

  // PIP APIs (using authorization endpoints)
  async validateAuthorization(data: AuthorizationRequest): Promise<AuthorizationResponse> {
    // Mock authorization validation since the backend PoA endpoint has different requirements
    // In production, this would call a proper policy decision point
    
    // Simulate validation logic based on input
    let allowed = true
    const denialReasons: string[] = []
    
    // Check 1: Token/Client ID validation
    if (!data.clientId || data.clientId.length < 5) {
      allowed = false
      denialReasons.push('Invalid or missing client ID')
    }
    
    // Check 2: Action validation - deny dangerous actions
    const dangerousActions = ['delete', 'destroy', 'terminate', 'drop']
    if (dangerousActions.some(action => data.action.toLowerCase().includes(action))) {
      allowed = false
      denialReasons.push('Dangerous action requires additional approval')
    }
    
    // Check 3: Geographic restrictions - deny certain regions
    const restrictedRegions = ['sanctioned', 'restricted', 'embargoed']
    if (restrictedRegions.some(region => data.geographic.toLowerCase().includes(region))) {
      allowed = false
      denialReasons.push('Geographic region is restricted')
    }
    
    // Check 4: Resource validation - allow common resources
    const validResources = ['read', 'list', 'view', 'get', 'fetch', 'query', 'api', 'data', 'user', 'admin']
    const hasValidResource = validResources.some(resource => 
      data.geographic.toLowerCase().includes(resource)
    )
    if (!hasValidResource && data.geographic.length > 0) {
      // Only deny if resource is specified but not in valid list
      const suspiciousResources = ['root', 'system', 'kernel', 'sudo']
      if (suspiciousResources.some(sus => data.geographic.toLowerCase().includes(sus))) {
        allowed = false
        denialReasons.push('Access to sensitive resource denied')
      }
    }
    
    const evaluationTime = Math.floor(Math.random() * 50 + 10)
    
    return Promise.resolve({
      authorized: allowed,
      allowed: allowed,
      clientId: data.clientId,
      action: data.action,
      geographicScope: data.geographic,
      industrySector: data.sector || 'general',
      cacheHit: Math.random() > 0.5,
      processingTime: `${evaluationTime}ms`,
      evaluationTime,
      policies: [
        'Resource Access Policy v1.2',
        'Geographic Restriction Policy',
        'Rate Limiting Policy',
        'Data Protection Policy (GDPR)',
        'Industry Sector Policy'
      ],
      policyChecks: [
        {
          policy: 'resource_access_policy',
          result: allowed ? 'allow' : 'deny'
        },
        {
          policy: 'geographic_restriction_policy',
          result: restrictedRegions.some(r => data.geographic.toLowerCase().includes(r)) ? 'deny' : 'allow'
        },
        {
          policy: 'action_validation_policy',
          result: dangerousActions.some(a => data.action.toLowerCase().includes(a)) ? 'deny' : 'allow'
        },
        {
          policy: 'rate_limit_policy',
          result: 'allow'
        }
      ]
    })
  }

  async getCacheStats(): Promise<CacheStats> {
    // Return metrics from authorization metrics
    const metrics = await this.getAuthzMetrics()
    const hits = metrics.cache_hits || 0
    const misses = metrics.cache_misses || 0
    return {
      hits,
      misses,
      hitRate: hits + misses > 0 ? hits / (hits + misses) : 0,
      totalRequests: hits + misses,
      evictions: metrics.cache_evictions || 0
    }
  }

  // PoA APIs (placeholder - need RFC-0111 endpoints)
  async createPoA(data: CreatePoARequest): Promise<PoAResponse> {
    return this.createDelegation(data) as any
  }

  async validatePoA(data: ValidatePoARequest): Promise<PoAValidationResponse> {
    // Validate against stored delegations
    const stored = sessionStorage.getItem('delegations') || '[]'
    const delegations = JSON.parse(stored)
    
    // Find matching delegation by poaId
    const matching = delegations.find((del: any) => del.delegation_id === data.poaId)
    
    const checks: ValidationCheck[] = []
    let isValid = false
    
    if (matching && matching.status === 'active') {
      // Check if PoA exists
      checks.push({ check: 'PoA Exists', result: 'pass' })
      
      // Check if action is allowed
      const actionAllowed = matching.actions.includes(data.action) || matching.actions.includes('*')
      checks.push({ 
        check: 'Action Authorized', 
        result: actionAllowed ? 'pass' : 'fail' 
      })
      
      // Check geographic restrictions
      const geoAllowed = matching.geoRestrictions.length === 0 || 
                        matching.geoRestrictions.includes(data.location) ||
                        matching.geoRestrictions.includes('*')
      checks.push({ 
        check: 'Geographic Authorization', 
        result: geoAllowed ? 'pass' : 'fail' 
      })
      
      // Check expiration
      const expiresAt = new Date(new Date(matching.createdAt).getTime() + matching.validityPeriod * 24 * 60 * 60 * 1000)
      const isExpired = expiresAt < new Date()
      checks.push({ 
        check: 'Validity Period', 
        result: isExpired ? 'fail' : 'pass' 
      })
      
      isValid = actionAllowed && geoAllowed && !isExpired
    } else {
      checks.push({ check: 'PoA Exists', result: 'fail' })
      checks.push({ check: 'Action Authorized', result: 'fail' })
      checks.push({ check: 'Geographic Authorization', result: 'fail' })
      checks.push({ check: 'Validity Period', result: 'fail' })
    }
    
    return {
      valid: isValid,
      poaId: data.poaId,
      action: data.action,
      location: data.location,
      checks,
      validationTime: new Date().toISOString()
    }
  }

  async listPoAs(): Promise<PoAResponse[]> {
    // Return stored delegations from session storage
    const stored = sessionStorage.getItem('delegations') || '[]'
    const delegations = JSON.parse(stored)
    
    return delegations.map((del: any) => ({
      poaId: del.delegation_id,
      grantor: del.grantor,
      representative: del.representative,
      actions: del.actions,
      status: del.status,
      createdAt: del.createdAt,
      validUntil: new Date(Date.now() + (del.validityPeriod || 365) * 24 * 60 * 60 * 1000).toISOString()
    }))
  }

  // Metrics
  async getMetrics(): Promise<MetricsResponse> {
    const response = await this.client.get('/token/metrics')
    return response.data
  }

  // Health Check
  async health(): Promise<HealthCheckResponse> {
    try {
      // Health endpoints are at backend root, need direct backend access
      // In dev, backend is at localhost:8080, in prod should use proper backend URL
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const backendUrl = (import.meta as any)?.env?.VITE_BACKEND_URL || 'http://localhost:8080'
      const response = await axios.get(`${backendUrl}/healthz`)
      return {
        success: true,
        status: response.data.status || 'healthy',
        data: response.data
      }
    } catch (e) {
      // Fallback to ready endpoint
      try {
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const backendUrl = (import.meta as any)?.env?.VITE_BACKEND_URL || 'http://localhost:8080'
        const resp = await axios.get(`${backendUrl}/ready`)
        return {
          success: true,
          status: 'ready',
          data: resp.data
        }
      } catch {
        throw e
      }
    }
  }

  async healthCheck(): Promise<HealthCheckResponse> {
    return this.health()
  }
}

// Type Definitions
export interface CreateTokenRequest {
  clientId: string
  ownersAuthorizer: string
  clientOwner: string
  scope: string[]
  expirationHours: number
}

export interface TokenResponse {
  token: string
  clientId: string
  expiresAt: string
  scope: string[]
  authorizationChain: {
    ownersAuthorizer: string
    clientOwner: string
    client: string
  }
}

export interface TokenValidationResponse {
  valid: boolean
  decoded?: Record<string, any>
  checks: Record<string, string>
  error?: string
}

export interface RevocationHead {
  head_hash?: string
  timestamp?: string
}

export interface RotationSummary {
  head_hash?: string
  aggregate_hash?: string
  chain_length?: number
  active_key_id?: string
  threshold?: string
}

export interface CapabilityAnchor {
  registry_hash?: string
  previous_hash?: string
  anchored_at?: string
  provider?: string
}

export interface ErrorCatalog {
  entries: ErrorEntry[]
}

export interface ErrorEntry {
  code: string
  http_status: number
  category: string
  severity: string
  retryable: boolean
  description: string
}

export interface AlgorithmsResponse {
  algorithms: string[]
}

export interface VerifyIdentityRequest {
  type: string
  trustLevel: string
  entityId: string
  tsp: string
}

export interface IdentityVerificationResponse {
  verified: boolean
  identityType: string
  trustLevel: string
  entityId: string
  tsp: string
  tspStatus: string
  verificationTime: string
  cryptographicBinding: string
}

export interface VerifyEntityRequest {
  jurisdiction: string
  registrationNumber: string
}

export interface EntityVerificationResponse {
  verified: boolean
  registrationNumber: string
  legalName: string
  jurisdiction: string
  status: string
  registrationDate: string
  legalForm: string
  managingDirectors: Director[]
}

export interface Director {
  name: string
  position: string
  authority: string
}

export interface VerifySignatoryRequest {
  entity: string
  signatoryName: string
  authorityType: string
}

export interface SignatoryVerificationResponse {
  authorized: boolean
  signatoryName: string
  entity: string
  authorityType: string
  appointmentDate: string
  restrictions: string
  status: string
}

export interface AuthorizationRequest {
  clientId: string
  action: string
  geographic: string
  sector?: string
}

export interface AuthorizationResponse {
  authorized: boolean
  clientId: string
  action: string
  geographicScope: string
  industrySector: string
  cacheHit: boolean
  processingTime: string
  policyChecks: PolicyCheck[]
  policies?: string[]
  evaluationTime?: number
  allowed?: boolean
}

export interface PolicyCheck {
  policy: string
  result: string
}

export interface CacheStats {
  hits: number
  misses: number
  hitRate: number
  totalRequests: number
  size?: number
  evictions?: number
}

export interface CreatePoARequest {
  grantor: string
  representative: string
  representativeType: string
  actions: string[]
  geographicScope: string
  validityDays: number
}

export interface PoAResponse {
  id: string
  grantor: string
  representative: string
  representativeType: string
  actions: string[]
  geographicScope: string
  validFrom: string
  validUntil: string
  status: string
}

export interface ValidatePoARequest {
  poaId: string
  action: string
  location: string
}

export interface PoAValidationResponse {
  valid: boolean
  poaId: string
  action: string
  location: string
  checks: ValidationCheck[]
  validationTime: string
}

export interface ValidationCheck {
  check: string
  result: string
}

export interface MetricsResponse {
  testsPassing: number
  totalTests: number
  benchmarks: number
  coverage: number
  e2ePerformance: number
}

export interface HealthCheckResponse {
  success?: boolean
  data?: {
    status: string
    timestamp?: string
    uptime?: string
  }
  status?: string
  services?: Record<string, string>
}

// Auth / MFA Types
export interface LoginRequest {
  username: string
  password: string
}

export interface LoginInitResponse {
  success: boolean
  userId?: string
  requiresMFA: boolean
  mfaMethods?: string[] // e.g. ["totp", "sms"]
  sessionChallenge?: string // challenge id for MFA step
  error?: string
}

export interface MFAVerifyRequest {
  challengeId: string
  code: string
  method: string // "totp" | "sms" | "email" | etc
}

export interface MFAResult {
  success: boolean
  token?: string // session / access token
  expiresAt?: string
  error?: string
}

export const apiClient = new ApiClient()
