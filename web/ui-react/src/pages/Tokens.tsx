import { useState } from 'react'
import { Card } from '../components/Card'
import { Button } from '../components/Button'
import { Input, Textarea } from '../components/Form'
import { Key, CheckCircle, Copy, Clock } from 'lucide-react'
import { apiClient, TokenResponse, TokenValidationResponse } from '../lib/api'
import { toast } from 'sonner'
import { formatDate } from '../lib/utils'
import { SubscriptionWizard } from '../components/subscription/SubscriptionWizard'

interface CreateTokenForm {
  clientId: string
  ownersAuthorizer: string
  clientOwner: string
  scope: string
  expirationHours: number
}

export default function Tokens() {
  const [createForm, setCreateForm] = useState<CreateTokenForm>({
    clientId: 'demo-client-001',
    ownersAuthorizer: 'HRB12345-DE',
    clientOwner: '12345678-GB',
    scope: 'read,write',
    expirationHours: 24,
  })
  const [validateToken, setValidateToken] = useState('')
  const [createdToken, setCreatedToken] = useState<TokenResponse | null>(null)
  const [validationResult, setValidationResult] = useState<TokenValidationResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [recentTokens, setRecentTokens] = useState<TokenResponse[]>([])
  const [showWizard, setShowWizard] = useState(false)

  const handleCreateToken = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const token = await apiClient.createToken({
        ...createForm,
        scope: createForm.scope.split(',').map((s) => s.trim()),
      })
      setCreatedToken(token)
      setRecentTokens([token, ...recentTokens.slice(0, 4)])
      toast.success('Token created successfully!')
    } catch (error: any) {
      toast.error(error.response?.data?.message || 'Failed to create token')
      console.error('Token creation error:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleValidateToken = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const result = await apiClient.validateToken(validateToken)
      setValidationResult(result)
      if (result.valid) {
        toast.success('Token is valid!')
      } else {
        toast.error('Token validation failed')
      }
    } catch (error: any) {
      toast.error(error.response?.data?.message || 'Failed to validate token')
      console.error('Token validation error:', error)
    } finally {
      setLoading(false)
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    toast.success('Copied to clipboard!')
  }

  return (
    <div className="space-y-6 animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
          Extended Token Management
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Create, validate, and manage RFC-0111 compliant extended tokens with 3-level
          authorization chains.
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Create Token - RFC-0111 Wizard */}
        <Card title="Create Extended Token (RFC-0111)" icon={<Key className="h-6 w-6" />}>
          {!showWizard ? (
            <div className="space-y-4">
              <p className="text-sm text-gray-600 dark:text-gray-400">
                Create a new RFC-0111 compliant extended token using the 8-step subscription flow with
                multi-level authorization.
              </p>
              <Button
                onClick={() => setShowWizard(true)}
                icon={<Key className="h-4 w-4" />}
              >
                Start Subscription Wizard
              </Button>
            </div>
          ) : (
            <SubscriptionWizard
              onComplete={(token) => {
                const newToken: TokenResponse = {
                  token,
                  clientId: 'RFC-0111 Subscription',
                  scope: ['read', 'write'],
                  expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
                  authorizationChain: {
                    ownersAuthorizer: 'RFC-0111 Flow',
                    clientOwner: 'RFC-0111 Flow',
                    client: 'RFC-0111 Flow',
                  },
                }
                setCreatedToken(newToken)
                setRecentTokens([newToken, ...recentTokens.slice(0, 4)])
                setShowWizard(false)
                toast.success('Token created via RFC-0111 subscription flow!')
              }}
              onCancel={() => setShowWizard(false)}
            />
          )}

          {/* Legacy Form - Keep for reference */}
          {!showWizard && (
            <>
              <div className="mt-6 pt-6 border-t border-gray-200 dark:border-gray-700">
                <h4 className="text-sm font-semibold text-gray-700 dark:text-gray-300 mb-4">
                  Legacy Form (For Testing)
                </h4>
                <form onSubmit={handleCreateToken} className="space-y-4">
            <Input
              label="Client ID"
              value={createForm.clientId}
              onChange={(e) => setCreateForm({ ...createForm, clientId: e.target.value })}
              placeholder="client-123"
              required
            />
            <Input
              label="Owner's Authorizer Entity"
              value={createForm.ownersAuthorizer}
              onChange={(e) => setCreateForm({ ...createForm, ownersAuthorizer: e.target.value })}
              placeholder="HRB12345-DE"
              required
            />
            <Input
              label="Client Owner Entity"
              value={createForm.clientOwner}
              onChange={(e) => setCreateForm({ ...createForm, clientOwner: e.target.value })}
              placeholder="12345678-GB"
              required
            />
            <Input
              label="Scope (comma-separated)"
              value={createForm.scope}
              onChange={(e) => setCreateForm({ ...createForm, scope: e.target.value })}
              placeholder="read, write, admin"
              required
            />
            <Input
              label="Expiration (hours)"
              type="number"
              value={createForm.expirationHours}
              onChange={(e) =>
                setCreateForm({ ...createForm, expirationHours: parseInt(e.target.value) })
              }
              min={1}
              max={720}
              required
            />
            <Button type="submit" loading={loading} icon={<Key className="h-4 w-4" />}>
              Create Token (Legacy)
            </Button>
          </form>
              </div>
            </>
          )}

          {createdToken && !showWizard && (
            <div className="mt-6 p-4 bg-success-50 dark:bg-success-900/20 border border-success-200 dark:border-success-800 rounded-lg">
              <div className="flex items-center gap-2 mb-3">
                <CheckCircle className="h-5 w-5 text-success-600 dark:text-success-400" />
                <h4 className="font-semibold text-success-900 dark:text-success-100">
                  Token Created Successfully
                </h4>
              </div>
              <div className="space-y-2 text-sm">
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Client ID:</span>
                  <span className="ml-2 font-mono text-gray-900 dark:text-white">
                    {String(createdToken.clientId || 'N/A')}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Expires:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">
                    {createdToken.expiresAt ? formatDate(createdToken.expiresAt) : 'N/A'}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Scope:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">
                    {createdToken.scope && Array.isArray(createdToken.scope) 
                      ? createdToken.scope.join(', ')
                      : 'N/A'}
                  </span>
                </div>
                <div className="pt-2">
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-gray-600 dark:text-gray-400">Token:</span>
                    <button
                      onClick={() => copyToClipboard(
                        typeof createdToken.token === 'object' 
                          ? JSON.stringify(createdToken.token)
                          : String(createdToken.token)
                      )}
                      className="text-primary-600 hover:text-primary-700 dark:text-primary-400"
                    >
                      <Copy className="h-4 w-4" />
                    </button>
                  </div>
                  <pre className="p-2 bg-gray-100 dark:bg-gray-800 rounded text-xs overflow-x-auto">
                    {typeof createdToken.token === 'object' 
                      ? JSON.stringify(createdToken.token, null, 2)
                      : String(createdToken.token)}
                  </pre>
                </div>
              </div>
            </div>
          )}
        </Card>

        {/* Validate Token */}
        <Card title="Validate Token" icon={<CheckCircle className="h-6 w-6" />}>
          <form onSubmit={handleValidateToken} className="space-y-4">
            <Textarea
              label="Token (JWT)"
              value={validateToken}
              onChange={(e) => setValidateToken(e.target.value)}
              placeholder="Paste JWT token here..."
              rows={6}
              required
            />
            <Button
              type="submit"
              loading={loading}
              variant="secondary"
              icon={<CheckCircle className="h-4 w-4" />}
            >
              Validate Token
            </Button>
          </form>

          {validationResult && (
            <div
              className={`mt-6 p-4 rounded-lg border ${
                validationResult.valid
                  ? 'bg-success-50 dark:bg-success-900/20 border-success-200 dark:border-success-800'
                  : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800'
              }`}
            >
              <div className="flex items-center gap-2 mb-3">
                {validationResult.valid ? (
                  <CheckCircle className="h-5 w-5 text-success-600 dark:text-success-400" />
                ) : (
                  <div className="h-5 w-5 text-red-600 dark:text-red-400">✕</div>
                )}
                <h4
                  className={`font-semibold ${
                    validationResult.valid
                      ? 'text-success-900 dark:text-success-100'
                      : 'text-red-900 dark:text-red-100'
                  }`}
                >
                  {validationResult.valid ? 'Token is Valid' : 'Token is Invalid'}
                </h4>
              </div>
              <div className="space-y-2 text-sm">
                {validationResult.checks && Object.entries(validationResult.checks).map(([key, value]) => (
                  <div key={key}>
                    <span className="text-gray-600 dark:text-gray-400">{key}:</span>
                    <span className="ml-2 font-semibold text-gray-900 dark:text-white">
                      {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                    </span>
                  </div>
                ))}
                {validationResult.decoded && (
                  <div className="pt-2">
                    <span className="text-gray-600 dark:text-gray-400">Decoded:</span>
                    <pre className="mt-1 p-2 bg-gray-100 dark:bg-gray-800 rounded text-xs overflow-x-auto">
                      {typeof validationResult.decoded === 'object' 
                        ? JSON.stringify(validationResult.decoded, null, 2)
                        : String(validationResult.decoded)}
                    </pre>
                  </div>
                )}
              </div>
            </div>
          )}
        </Card>
      </div>

      {/* Recent Tokens */}
      {recentTokens.length > 0 && (
        <Card title="Recent Tokens" icon={<Clock className="h-6 w-6" />}>
          <div className="space-y-3">
            {recentTokens.map((token, idx) => (
              <div
                key={idx}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg"
              >
                <div className="flex-1">
                  <div className="font-semibold text-gray-900 dark:text-white">
                    {String(token.clientId || 'N/A')}
                  </div>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    Expires: {token.expiresAt ? formatDate(token.expiresAt) : 'N/A'}
                  </div>
                  <div className="text-xs text-gray-500 dark:text-gray-500">
                    Scope: {token.scope && Array.isArray(token.scope) 
                      ? token.scope.join(', ')
                      : 'N/A'}
                  </div>
                </div>
                <span className="px-2 py-1 bg-success-100 dark:bg-success-900 text-success-800 dark:text-success-200 text-xs font-semibold rounded-full">
                  Active
                </span>
              </div>
            ))}
          </div>
        </Card>
      )}
    </div>
  )
}
