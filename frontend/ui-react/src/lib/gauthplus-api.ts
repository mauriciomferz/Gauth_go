/**
 * AgentAuth+ API Client
 * Provides type-safe access to AgentAuth+ management endpoints
 */

import axios, { AxiosInstance } from 'axios'

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const RAW_BASE: string | undefined = (import.meta as any)?.env?.VITE_API_BASE_URL
const API_BASE_URL = RAW_BASE && RAW_BASE.trim() !== '' ? RAW_BASE : '/api/v1'

export interface SuccessorActivation {
  id: string
  poa_id: string
  primary_agent_id: string
  successor_agent_id: string
  activation_reason: string
  activated_at: string
  activated_by: string
  deactivated_at?: string
  deactivated_by?: string
  status: 'active' | 'deactivated' | 'superseded'
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface AIDelegation {
  id: string
  source_poa_id: string
  source_agent_id: string
  target_agent_id: string
  delegated_scope: string[]
  delegation_depth: number
  max_allowed_depth: number
  valid_from: string
  valid_until: string
  status: 'active' | 'revoked' | 'expired'
  delegation_policy?: DelegationPolicy
  revoked_at?: string
  revoked_by?: string
  revocation_reason?: string
  created_at: string
  updated_at: string
}

export interface DelegationPolicy {
  can_delegate: boolean
  max_depth: number
  allowed_delegates: string[]
  requires_approval: boolean
  scope_restriction: 'maintain' | 'reduce_only' | 'none'
  time_restriction: 'maintain' | 'reduce_only' | 'extend_allowed'
}

export interface DualControlApproval {
  id: string
  poa_id: string
  action_type: string
  action_description: string
  requested_by: string
  requested_at: string
  required_approvers: number
  approval_threshold: 'all' | 'majority' | 'quorum' | 'weighted'
  status: 'pending' | 'approved' | 'rejected' | 'expired'
  approved_by: ApprovalRecord[]
  rejected_by: ApprovalRecord[]
  decision_finalized_at?: string
  expires_at?: string
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface ApprovalRecord {
  approver_id: string
  approved_at: string
  comments?: string
  weight?: number
}

export interface AICapabilityAssessment {
  id: string
  agent_id: string
  assessment_date: string
  overall_level: 'L0' | 'L1' | 'L2' | 'L3' | 'L4' | 'L5'
  domain_scores: Record<string, number>
  risk_profile: Record<string, unknown>
  certification_status: 'uncertified' | 'pending' | 'certified' | 'expired'
  certifications: string[]
  limitations?: string[]
  recommended_restrictions?: string[]
  assessed_by: string
  valid_until: string
  notes?: string
  superseded_by?: string
  created_at: string
  updated_at: string
}

export interface FiduciaryDutyViolation {
  id: string
  poa_id: string
  agent_id: string
  duty_type: 'care' | 'loyalty' | 'good_faith' | 'disclosure' | 'confidentiality'
  violation_description: string
  severity: 'minor' | 'moderate' | 'major' | 'critical'
  detected_at: string
  detected_by: string
  reviewed_by?: string
  reviewed_at?: string
  resolution_status: 'open' | 'investigating' | 'resolved' | 'dismissed'
  resolution_notes?: string
  consequences?: Record<string, unknown>
  evidence?: Record<string, unknown>
  created_at: string
  updated_at: string
}

class AgentAuthPlusClient {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: `${API_BASE_URL}/gauthplus`,
      headers: {
        'Content-Type': 'application/json',
      },
    })
  }

  // ====== Successor Management ======

  async activateSuccessor(data: {
    poa_id: string
    primary_agent_id: string
    successor_agent_id: string
    reason: string
    activated_by: string
  }): Promise<{ success: boolean; activation: SuccessorActivation }> {
    const response = await this.client.post('/successors/activate', data)
    return response.data
  }

  async deactivateSuccessor(data: {
    activation_id: string
    deactivated_by: string
  }): Promise<{ success: boolean; message: string }> {
    const response = await this.client.post('/successors/deactivate', data)
    return response.data
  }

  async getActiveSuccessor(poaId: string): Promise<{
    success: boolean
    active_successor: SuccessorActivation | null
  }> {
    const response = await this.client.get(`/successors/active/${poaId}`)
    return response.data
  }

  async listSuccessorHistory(poaId: string): Promise<{
    success: boolean
    history: SuccessorActivation[]
    count: number
  }> {
    const response = await this.client.get(`/successors/history/${poaId}`)
    return response.data
  }

  // ====== Delegation Management ======

  async createDelegation(delegation: Partial<AIDelegation>): Promise<{
    success: boolean
    delegation: AIDelegation
  }> {
    const response = await this.client.post('/delegations', { delegation })
    return response.data
  }

  async revokeDelegation(
    delegationId: string,
    revokedBy: string,
    reason: string
  ): Promise<{ success: boolean; message: string }> {
    const response = await this.client.post(`/delegations/${delegationId}/revoke`, {
      revoked_by: revokedBy,
      reason,
    })
    return response.data
  }

  async validateDelegation(data: {
    source_agent_id: string
    target_agent_id: string
    scope: string[]
    depth: number
  }): Promise<{ success: boolean; valid: boolean; error?: string }> {
    const response = await this.client.post('/delegations/validate', data)
    return response.data
  }

  async getDelegationChain(agentId: string): Promise<{
    success: boolean
    chain: AIDelegation[]
    depth: number
  }> {
    const response = await this.client.get(`/delegations/chain/${agentId}`)
    return response.data
  }

  async checkMaxDepth(
    sourceAgentId: string,
    currentDepth: number
  ): Promise<{ success: boolean; depth_exceeded: boolean; current_depth: number }> {
    const response = await this.client.post('/delegations/check-depth', {
      source_agent_id: sourceAgentId,
      current_depth: currentDepth,
    })
    return response.data
  }

  // ====== Dual Control Approvals ======

  async requestApproval(approval: Partial<DualControlApproval>): Promise<{
    success: boolean
    approval_id: string
    approval: DualControlApproval
  }> {
    const response = await this.client.post('/dual-control/approvals', { approval })
    return response.data
  }

  async approveAction(
    approvalId: string,
    approverId: string,
    comments?: string
  ): Promise<{ success: boolean; message: string }> {
    const response = await this.client.post(`/dual-control/approvals/${approvalId}/approve`, {
      approver_id: approverId,
      comments,
    })
    return response.data
  }

  async rejectAction(
    approvalId: string,
    approverId: string,
    comments: string
  ): Promise<{ success: boolean; message: string }> {
    const response = await this.client.post(`/dual-control/approvals/${approvalId}/reject`, {
      approver_id: approverId,
      comments,
    })
    return response.data
  }

  async getApprovalStatus(approvalId: string): Promise<{
    success: boolean
    approval_id: string
    status: string
  }> {
    const response = await this.client.get(`/dual-control/approvals/${approvalId}/status`)
    return response.data
  }

  async getPendingApprovals(poaId?: string): Promise<{
    success: boolean
    approvals: DualControlApproval[]
    count: number
  }> {
    const params = poaId ? { poa_id: poaId } : {}
    const response = await this.client.get('/dual-control/approvals/pending', { params })
    return response.data
  }

  async findApprovalsByPoAAndAction(
    poaId: string,
    actionType: string
  ): Promise<{
    success: boolean
    poa_id: string
    action_type: string
    approvals: DualControlApproval[]
    count: number
  }> {
    const response = await this.client.get('/dual-control/approvals/query', {
      params: { poa_id: poaId, action_type: actionType },
    })
    return response.data
  }

  // ====== Capability Assessment ======

  async createAssessment(assessment: Partial<AICapabilityAssessment>): Promise<{
    success: boolean
    assessment: AICapabilityAssessment
  }> {
    const response = await this.client.post('/capabilities/assess', { assessment })
    return response.data
  }

  async getLatestAssessment(agentId: string): Promise<{
    success: boolean
    agent_id: string
    assessment: AICapabilityAssessment
  }> {
    const response = await this.client.get(`/capabilities/assessments/${agentId}`)
    return response.data
  }

  async listCertifications(agentId: string): Promise<{
    success: boolean
    agent_id: string
    certifications: string[]
    count: number
  }> {
    const response = await this.client.get(`/capabilities/certifications/${agentId}`)
    return response.data
  }

  // ====== Fiduciary Duty ======

  async recordViolation(violation: Partial<FiduciaryDutyViolation>): Promise<{
    success: boolean
    violation: FiduciaryDutyViolation
  }> {
    const response = await this.client.post('/fiduciary/violations', { violation })
    return response.data
  }

  async getViolations(poaId?: string, agentId?: string): Promise<{
    success: boolean
    violations: FiduciaryDutyViolation[]
    count: number
  }> {
    const params: Record<string, string> = {}
    if (poaId) params.poa_id = poaId
    if (agentId) params.agent_id = agentId
    const response = await this.client.get('/fiduciary/violations', { params })
    return response.data
  }

  async getViolationsBySeverity(minSeverity: string): Promise<{
    success: boolean
    min_severity: string
    violations: FiduciaryDutyViolation[]
    count: number
  }> {
    const response = await this.client.get('/fiduciary/violations/by-severity', {
      params: { min_severity: minSeverity },
    })
    return response.data
  }

  async resolveViolation(
    violationId: string,
    reviewedBy: string,
    notes: string
  ): Promise<{ success: boolean; message: string }> {
    const response = await this.client.post(`/fiduciary/violations/${violationId}/resolve`, {
      reviewed_by: reviewedBy,
      notes,
    })
    return response.data
  }
}

export const gauthPlusAPI = new AgentAuthPlusClient()
