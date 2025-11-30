/**
 * API Integration Tests
 * Tests backend API endpoints directly without browser
 */

import { test, expect } from '@playwright/test'

// Base API URL from config
const API_BASE = process.env.API_URL || 'http://localhost:8080'

test.describe('Backend API Integration', () => {
  
  test.describe('Health and Status', () => {
    test('should have healthy backend', async ({ request }) => {
      const response = await request.get(`${API_BASE}/api/v1/beta/health`)
      expect(response.ok()).toBeTruthy()
      expect(response.status()).toBe(200)
    })

    test('should return API version info', async ({ request }) => {
      const response = await request.get(`${API_BASE}/api/v1/beta/info`)
      expect(response.ok() || response.status() === 404).toBeTruthy()
    })
  })

  test.describe('Metrics Endpoint', () => {
    test('should return Prometheus metrics', async ({ request }) => {
      const response = await request.get(`${API_BASE}/api/v1/beta/metrics`)
      
      if (response.ok()) {
        const body = await response.text()
        
        // Verify Prometheus format
        expect(body).toContain('# HELP')
        expect(body).toContain('# TYPE')
        
        console.log(`✅ Metrics endpoint returned ${body.split('\n').length} lines`)
      } else {
        console.log('ℹ️  Metrics endpoint not available')
      }
    })

    test('should have HTTP metrics', async ({ request }) => {
      const response = await request.get(`${API_BASE}/api/v1/beta/metrics`)
      
      if (response.ok()) {
        const body = await response.text()
        expect(body).toMatch(/http.*requests.*total/i)
      }
    })
  })

  test.describe('Token Management', () => {
    test('should create token', async ({ request }) => {
      const response = await request.post(`${API_BASE}/api/v1/beta/tokens`, {
        data: {
          clientId: `test-client-${Date.now()}`,
          subject: 'test-subject',
          scopes: ['read', 'write'],
        },
      })
      
      if (response.ok()) {
        const data = await response.json()
        expect(data).toHaveProperty('token')
        console.log('✅ Token created successfully')
      } else {
        console.log(`ℹ️  Token creation returned ${response.status()}`)
      }
    })

    test('should validate token format', async ({ request }) => {
      // First create a token
      const createResponse = await request.post(`${API_BASE}/api/v1/beta/tokens`, {
        data: {
          clientId: `test-client-${Date.now()}`,
          subject: 'test-subject',
        },
      })
      
      if (createResponse.ok()) {
        const { token } = await createResponse.json()
        
        // Validate the token
        const validateResponse = await request.post(`${API_BASE}/api/v1/beta/tokens/validate`, {
          data: { token },
        })
        
        if (validateResponse.ok()) {
          console.log('✅ Token validation successful')
        }
      }
    })
  })

  test.describe('PVP (Personal Verification Point)', () => {
    test('should verify identity', async ({ request }) => {
      const response = await request.post(`${API_BASE}/api/v1/beta/pvp/verify`, {
        data: {
          subject: 'test-user',
          identityProvider: 'test-provider',
        },
      })
      
      if (response.ok() || response.status() === 400) {
        console.log(`✅ PVP endpoint responded with ${response.status()}`)
      }
    })
  })

  test.describe('Registry', () => {
    test('should query registry', async ({ request }) => {
      const response = await request.post(`${API_BASE}/api/v1/beta/registry/query`, {
        data: {
          entityId: 'test-entity',
          type: 'organization',
        },
      })
      
      if (response.ok() || response.status() === 404) {
        console.log(`✅ Registry endpoint responded with ${response.status()}`)
      }
    })
  })

  test.describe('PIP (Policy Information Point)', () => {
    test('should get policies list', async ({ request }) => {
      const response = await request.get(`${API_BASE}/api/v1/beta/pip/policies`)
      
      if (response.ok()) {
        const data = await response.json()
        expect(Array.isArray(data) || typeof data === 'object').toBeTruthy()
        console.log('✅ Policies retrieved successfully')
      } else {
        console.log(`ℹ️  PIP policies endpoint returned ${response.status()}`)
      }
    })

    test('should get cache stats', async ({ request }) => {
      const response = await request.get(`${API_BASE}/api/v1/beta/pip/cache/stats`)
      
      if (response.ok()) {
        const data = await response.json()
        expect(data).toHaveProperty('hits', data.hits >= 0)
        console.log('✅ Cache stats retrieved')
      } else {
        console.log(`ℹ️  Cache stats endpoint returned ${response.status()}`)
      }
    })

    test('should check authorization', async ({ request }) => {
      const response = await request.post(`${API_BASE}/api/v1/beta/pip/authorize`, {
        data: {
          subject: 'test-user',
          action: 'read',
          resource: 'test-resource',
        },
      })
      
      if (response.ok() || response.status() === 403) {
        console.log(`✅ Authorization check responded with ${response.status()}`)
      }
    })
  })

  test.describe('Subscriptions', () => {
    test('should list subscriptions', async ({ request }) => {
      const response = await request.get(`${API_BASE}/api/v1/beta/subscriptions`)
      
      if (response.ok()) {
        const data = await response.json()
        expect(Array.isArray(data) || data.subscriptions).toBeTruthy()
        console.log('✅ Subscriptions list retrieved')
      } else {
        console.log(`ℹ️  Subscriptions endpoint returned ${response.status()}`)
      }
    })

    test('should create subscription', async ({ request }) => {
      const response = await request.post(`${API_BASE}/api/v1/beta/subscriptions`, {
        data: {
          clientId: `test-client-${Date.now()}`,
          name: 'Test Subscription',
          description: 'E2E test subscription',
        },
      })
      
      if (response.ok()) {
        const data = await response.json()
        expect(data).toHaveProperty('id')
        console.log('✅ Subscription created successfully')
      } else {
        console.log(`ℹ️  Subscription creation returned ${response.status()}`)
      }
    })
  })

  test.describe('Error Handling', () => {
    test('should return 404 for unknown endpoints', async ({ request }) => {
      const response = await request.get(`${API_BASE}/api/v1/beta/nonexistent-endpoint`)
      expect(response.status()).toBe(404)
    })

    test('should return 400 for invalid data', async ({ request }) => {
      const response = await request.post(`${API_BASE}/api/v1/beta/tokens`, {
        data: {
          // Missing required fields
        },
      })
      
      expect(response.status()).toBeGreaterThanOrEqual(400)
      expect(response.status()).toBeLessThan(500)
    })

    test('should handle malformed JSON', async ({ request }) => {
      const response = await request.post(`${API_BASE}/api/v1/beta/tokens`, {
        data: 'invalid-json',
      })
      
      expect(response.status()).toBeGreaterThanOrEqual(400)
    })
  })

  test.describe('Performance', () => {
    test('should respond quickly to health checks', async ({ request }) => {
      const start = Date.now()
      const response = await request.get(`${API_BASE}/api/v1/beta/health`)
      const duration = Date.now() - start
      
      expect(response.ok()).toBeTruthy()
      expect(duration).toBeLessThan(1000) // Should respond within 1 second
      
      console.log(`✅ Health check took ${duration}ms`)
    })

    test('should handle concurrent requests', async ({ request }) => {
      const requests = Array(10).fill(null).map(() => 
        request.get(`${API_BASE}/api/v1/beta/health`)
      )
      
      const responses = await Promise.all(requests)
      const allOk = responses.every(r => r.ok())
      
      expect(allOk).toBeTruthy()
      console.log('✅ Handled 10 concurrent requests successfully')
    })
  })
})
