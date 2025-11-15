import { useState } from 'react'
import { Card } from '../Card'
import { apiClient } from '../../lib/api'

interface SubscriptionWizardProps {
  onComplete: (token: string) => void
  onCancel?: () => void
}

interface SubscriptionState {
  subscriptionId: string | null
  currentStep: number
  token: string | null
  error: string | null
}

/**
 * SubscriptionWizard implements the RFC-0111 8-step subscription flow
 * This is a simplified implementation showing the pattern.
 * For production, each step would have its own component with full validation.
 */
export function SubscriptionWizard({ onComplete, onCancel }: SubscriptionWizardProps) {
  const [state, setState] = useState<SubscriptionState>({
    subscriptionId: null,
    currentStep: 1,
    token: null,
    error: null
  })
  const [loading, setLoading] = useState(false)

  // Form data for simplified demo
  const [formData, setFormData] = useState({
    clientId: 'demo-client-' + Date.now(),
    authorizerSubject: 'authorizer-123',
    clientOwner: 'owner-456',
    resourceOwner: 'resource-789',
    resourceServer: 'server-001',
    scope: ['read', 'write']
  })

  const steps = [
    { number: 1, title: 'Initiate', description: 'Create subscription' },
    { number: 2, title: 'Authorizer Auth', description: 'Verify authorizer' },
    { number: 3, title: 'Client Owner ID', description: 'Verify client owner' },
    { number: 4, title: 'Client Owner Auth', description: 'Authorize client owner' },
    { number: 5, title: 'Client Auth', description: 'Authorize client' },
    { number: 6, title: 'Resource Owner ID', description: 'Verify resource owner' },
    { number: 7, title: 'Resource Owner Auth', description: 'Authorize resource owner' },
    { number: 8, title: 'Resource Server', description: 'Complete & get token' }
  ]

  const executeStep = async () => {
    setLoading(true)
    setState(prev => ({ ...prev, error: null }))

    try {
      switch (state.currentStep) {
        case 1:
          await executeStepI()
          break
        case 2:
          await executeStepII()
          break
        case 3:
          await executeStepIII()
          break
        case 4:
          await executeStepIV()
          break
        case 5:
          await executeStepV()
          break
        case 6:
          await executeStepVI()
          break
        case 7:
          await executeStepVII()
          break
        case 8:
          await executeStepVIII()
          break
      }
    } catch (error: any) {
      setState(prev => ({
        ...prev,
        error: error.response?.data?.message || error.message || 'Step failed'
      }))
    } finally {
      setLoading(false)
    }
  }

  const executeStepI = async () => {
    const response = await apiClient.createSubscription({
      client_id: formData.clientId,
      requested_scope: formData.scope
    })
    
    setState(prev => ({
      ...prev,
      subscriptionId: response.subscription_id,
      currentStep: 2
    }))
  }

  const executeStepII = async () => {
    await apiClient.subscriptionStepII(state.subscriptionId!, {
      subject_id: formData.authorizerSubject,
      proof_method: 'eIDAS',
      proof_data: { verified: true }
    })
    
    setState(prev => ({ ...prev, currentStep: 3 }))
  }

  const executeStepIII = async () => {
    await apiClient.subscriptionStepIII(state.subscriptionId!, {
      subject_id: formData.clientOwner,
      proof_method: 'eIDAS',
      proof_data: { verified: true }
    })
    
    setState(prev => ({ ...prev, currentStep: 4 }))
  }

  const executeStepIV = async () => {
    await apiClient.subscriptionStepIV(state.subscriptionId!, {
      entity_id: formData.clientOwner,
      proof_method: 'commercial_register',
      proof_data: { verified: true }
    })
    
    setState(prev => ({ ...prev, currentStep: 5 }))
  }

  const executeStepV = async () => {
    await apiClient.subscriptionStepV(state.subscriptionId!, {
      client_id: formData.clientId,
      poa_proof: { valid: true }
    })
    
    setState(prev => ({ ...prev, currentStep: 6 }))
  }

  const executeStepVI = async () => {
    await apiClient.subscriptionStepVI(state.subscriptionId!, {
      subject_id: formData.resourceOwner,
      proof_method: 'eIDAS',
      proof_data: { verified: true }
    })
    
    setState(prev => ({ ...prev, currentStep: 7 }))
  }

  const executeStepVII = async () => {
    await apiClient.subscriptionStepVII(state.subscriptionId!, {
      entity_id: formData.resourceOwner,
      proof_method: 'commercial_register',
      proof_data: { verified: true }
    })
    
    setState(prev => ({ ...prev, currentStep: 8 }))
  }

  const executeStepVIII = async () => {
    const response = await apiClient.subscriptionStepVIII(state.subscriptionId!, {
      server_id: formData.resourceServer,
      authorization_proof: { valid: true }
    })
    
    const token = response.token || response.access_token
    setState(prev => ({ ...prev, token }))
    
    if (token) {
      onComplete(token)
    }
  }

  return (
    <div className="space-y-6">
      <Card title="RFC-0111 Subscription Flow" className="max-w-4xl mx-auto">
        {/* Progress Indicator */}
        <div className="mb-8">
          <div className="flex items-center justify-between mb-4">
            {steps.map((step, index) => (
              <div key={step.number} className="flex items-center">
                <div
                  className={`w-10 h-10 rounded-full flex items-center justify-center font-semibold ${
                    step.number < state.currentStep
                      ? 'bg-green-500 text-white'
                      : step.number === state.currentStep
                      ? 'bg-blue-500 text-white'
                      : 'bg-gray-300 text-gray-600'
                  }`}
                >
                  {step.number < state.currentStep ? '✓' : step.number}
                </div>
                {index < steps.length - 1 && (
                  <div
                    className={`w-12 h-1 ${
                      step.number < state.currentStep ? 'bg-green-500' : 'bg-gray-300'
                    }`}
                  />
                )}
              </div>
            ))}
          </div>
          <div className="text-center">
            <h3 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
              Step {state.currentStep}: {steps[state.currentStep - 1].title}
            </h3>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              {steps[state.currentStep - 1].description}
            </p>
          </div>
        </div>

        {/* Form Fields (simplified for demo) */}
        {state.currentStep === 1 && (
          <div className="space-y-4 mb-6">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Client ID
              </label>
              <input
                type="text"
                value={formData.clientId}
                onChange={(e) => setFormData(prev => ({ ...prev, clientId: e.target.value }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Requested Scope
              </label>
              <input
                type="text"
                value={formData.scope.join(', ')}
                onChange={(e) => setFormData(prev => ({ ...prev, scope: e.target.value.split(',').map(s => s.trim()) }))}
                className="w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:ring-blue-500 focus:border-blue-500"
                placeholder="read, write, delete"
              />
            </div>
          </div>
        )}

        {/* Subscription ID Display */}
        {state.subscriptionId && state.currentStep > 1 && (
          <div className="mb-6 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
            <p className="text-sm text-gray-600 dark:text-gray-400">
              <span className="font-semibold">Subscription ID:</span>{' '}
              <code className="text-xs bg-gray-200 dark:bg-gray-600 px-2 py-1 rounded">
                {state.subscriptionId}
              </code>
            </p>
          </div>
        )}

        {/* Error Display */}
        {state.error && (
          <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <p className="text-sm text-red-600 dark:text-red-400">
              <span className="font-semibold">Error:</span> {state.error}
            </p>
          </div>
        )}

        {/* Token Display */}
        {state.token && (
          <div className="mb-6 p-4 bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 rounded-lg">
            <p className="text-sm text-green-600 dark:text-green-400 mb-2 font-semibold">
              ✓ Token Created Successfully!
            </p>
            <div className="bg-white dark:bg-gray-800 p-3 rounded border border-gray-200 dark:border-gray-700">
              <code className="text-xs break-all">{state.token}</code>
            </div>
          </div>
        )}

        {/* Action Buttons */}
        <div className="flex gap-4 justify-end">
          {onCancel && !state.token && (
            <button
              onClick={onCancel}
              className="px-6 py-2 bg-gray-300 text-gray-700 rounded-lg hover:bg-gray-400 transition-colors"
              disabled={loading}
            >
              Cancel
            </button>
          )}
          {!state.token && (
            <button
              onClick={executeStep}
              disabled={loading}
              className="px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:bg-gray-400 disabled:cursor-not-allowed transition-colors"
            >
              {loading ? 'Processing...' : state.currentStep === 8 ? 'Complete & Get Token' : 'Next Step'}
            </button>
          )}
          {state.token && (
            <button
              onClick={() => onComplete(state.token!)}
              className="px-6 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 transition-colors"
            >
              Done
            </button>
          )}
        </div>

        {/* Step Information */}
        <div className="mt-6 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
          <p className="text-xs text-blue-600 dark:text-blue-400">
            <span className="font-semibold">Note:</span> This is a simplified wizard for demonstration.
            Each step calls the real RFC-0111 backend endpoint. In production, each step would have
            detailed form fields and validation.
          </p>
        </div>
      </Card>
    </div>
  )
}
