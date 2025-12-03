import { ReactElement } from 'react'
import { render, RenderOptions } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'

/**
 * Custom render function that wraps components with common providers
 */
export function renderWithRouter(
  ui: ReactElement,
  options?: Omit<RenderOptions, 'wrapper'>
) {
  return render(ui, {
    wrapper: ({ children }) => <BrowserRouter>{children}</BrowserRouter>,
    ...options,
  })
}

/**
 * Mock API responses for testing
 */
export const mockTokenResponse = {
  token: 'eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.test.signature',
  clientId: 'test-client',
  expiresAt: new Date(Date.now() + 3600000).toISOString(),
  scope: ['read', 'write'],
  authorizationChain: {
    ownersAuthorizer: 'HRB12345-DE',
    clientOwner: '12345678-GB',
    client: 'test-client',
  },
}

export const mockTokenValidationResponse = {
  valid: true,
  clientId: 'test-client',
  scope: ['read', 'write'],
  expiresAt: new Date(Date.now() + 3600000).toISOString(),
  authorizationChain: {
    ownersAuthorizer: 'HRB12345-DE',
    clientOwner: '12345678-GB',
    client: 'test-client',
  },
}

export const mockPVPVerificationResponse = {
  verified: true,
  confidence: 0.95,
  verificationId: 'pvp-123',
  timestamp: new Date().toISOString(),
}

export const mockPoAResponse = {
  id: 'poa-123',
  grantor: 'entity-001',
  grantee: 'entity-002',
  scope: ['read', 'write'],
  expiresAt: new Date(Date.now() + 86400000).toISOString(),
  createdAt: new Date().toISOString(),
  status: 'active',
}

// Re-export everything from @testing-library/react
export * from '@testing-library/react'
export { default as userEvent } from '@testing-library/user-event'
