// API Integration Test Utility
// This file provides functions to test backend API connectivity

import { apiClient } from '../lib/api'

export interface ApiTestResult {
  endpoint: string
  method: string
  status: 'success' | 'error'
  responseTime: number
  data?: any
  error?: string
}

export async function testBackendHealth(): Promise<ApiTestResult> {
  const start = Date.now()
  try {
    const data = await apiClient.health()
    return {
      endpoint: '/api/v1/beta/health',
      method: 'GET',
      status: 'success',
      responseTime: Date.now() - start,
      data
    }
  } catch (error: any) {
    return {
      endpoint: '/api/v1/beta/health',
      method: 'GET',
      status: 'error',
      responseTime: Date.now() - start,
      error: error.message
    }
  }
}

export async function testTokenCreation(): Promise<ApiTestResult> {
  const start = Date.now()
  try {
    const data = await apiClient.createToken({
      clientId: 'test-client',
      ownersAuthorizer: 'test-authorizer',
      clientOwner: 'test-owner',
      scope: ['read', 'write'],
      expirationHours: 24
    })
    return {
      endpoint: '/api/v1/token/create',
      method: 'POST',
      status: 'success',
      responseTime: Date.now() - start,
      data
    }
  } catch (error: any) {
    return {
      endpoint: '/api/v1/token/create',
      method: 'POST',
      status: 'error',
      responseTime: Date.now() - start,
      error: error.message
    }
  }
}

export async function testAuthorizationCheck(): Promise<ApiTestResult> {
  const start = Date.now()
  try {
    const data = await apiClient.checkAuthorization({
      clientId: 'test-client',
      action: 'read',
      geographic: 'EU',
      sector: 'finance'
    })
    return {
      endpoint: '/api/v1/beta/authz/check',
      method: 'POST',
      status: 'success',
      responseTime: Date.now() - start,
      data
    }
  } catch (error: any) {
    return {
      endpoint: '/api/v1/beta/authz/check',
      method: 'POST',
      status: 'error',
      responseTime: Date.now() - start,
      error: error.message
    }
  }
}

export async function testCapabilities(): Promise<ApiTestResult> {
  const start = Date.now()
  try {
    const data = await apiClient.getCapabilities()
    return {
      endpoint: '/api/v1/beta/capabilities',
      method: 'GET',
      status: 'success',
      responseTime: Date.now() - start,
      data
    }
  } catch (error: any) {
    return {
      endpoint: '/api/v1/beta/capabilities',
      method: 'GET',
      status: 'error',
      responseTime: Date.now() - start,
      error: error.message
    }
  }
}

export async function testAuditLogs(): Promise<ApiTestResult> {
  const start = Date.now()
  try {
    const data = await apiClient.getAuditLogs()
    return {
      endpoint: '/api/v1/audit/logs',
      method: 'GET',
      status: 'success',
      responseTime: Date.now() - start,
      data
    }
  } catch (error: any) {
    return {
      endpoint: '/api/v1/audit/logs',
      method: 'GET',
      status: 'error',
      responseTime: Date.now() - start,
      error: error.message
    }
  }
}

export async function testMetrics(): Promise<ApiTestResult> {
  const start = Date.now()
  try {
    const data = await apiClient.getAuthzMetrics()
    return {
      endpoint: '/api/v1/beta/authz/metrics',
      method: 'GET',
      status: 'success',
      responseTime: Date.now() - start,
      data
    }
  } catch (error: any) {
    return {
      endpoint: '/api/v1/beta/authz/metrics',
      method: 'GET',
      status: 'error',
      responseTime: Date.now() - start,
      error: error.message
    }
  }
}

export async function runAllTests(): Promise<ApiTestResult[]> {
  const results = await Promise.all([
    testBackendHealth(),
    testTokenCreation(),
    testAuthorizationCheck(),
    testCapabilities(),
    testAuditLogs(),
    testMetrics()
  ])
  
  return results
}
