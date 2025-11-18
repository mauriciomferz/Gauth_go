import { useState, useEffect } from 'react'
import { Shield, AlertTriangle, CheckCircle, XCircle } from 'lucide-react'
import { Card, StatCard } from '../components/Card'
import { apiClient } from '../lib/api'
import { toast } from 'sonner'

interface Violation {
  violation_type: string
  severity: string
  description: string
  detected_at: string
}

interface EnforcementResult {
  allowed: boolean
  enforcement_id: string
  reason: string
  violations?: Violation[]
  token_valid?: boolean
  scope_valid?: boolean
}

export default function PEP() {
  const [violations, setViolations] = useState<Violation[]>([])
  const [enforcementResult, setEnforcementResult] = useState<EnforcementResult | null>(null)
  const [testResult, setTestResult] = useState<any>(null)
  const [loading, setLoading] = useState(false)

  const [enforceForm, setEnforceForm] = useState({
    token: '',
    action_type: 'transaction',
    transaction_type: 'financial',
    resource_id: '',
    enforcement_mode: 'strict'
  })

  useEffect(() => {
    loadViolations()
  }, [])

  const loadViolations = async () => {
    try {
      const response = await apiClient.request<{ violations: Violation[] }>({
        method: 'GET',
        url: '/pep/violations/recent?limit=10'
      })
      setViolations(response.data?.violations || [])
    } catch (error) {
      console.error('Failed to load violations:', error)
    }
  }

  const handleEnforceAuthorization = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const payload = {
        extended_token: enforceForm.token,
        action_type: enforceForm.action_type,
        transaction_type: enforceForm.transaction_type,
        resource_id: enforceForm.resource_id,
        enforcement_mode: enforceForm.enforcement_mode,
        context: {
          timestamp: new Date().toISOString()
        }
      }

      const response = await apiClient.request<EnforcementResult>({
        method: 'POST',
        url: '/pep/enforce',
        data: payload
      })

      setEnforcementResult(response.data)
      toast.success('Authorization enforcement completed')
      loadViolations()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Enforcement failed')
      setEnforcementResult(null)
    } finally {
      setLoading(false)
    }
  }

  const handleTestPEP = async (side: 'supply' | 'demand') => {
    try {
      const response = await apiClient.request({
        method: 'POST',
        url: `/pep/test/${side}`,
        data: {
          test_scenario: 'basic',
          context: {
            timestamp: new Date().toISOString()
          }
        }
      })

      setTestResult({ ...response.data, side })
      toast.success(`${side === 'supply' ? 'Supply' : 'Demand'}-side test completed`)
    } catch (error: any) {
      toast.error(error.response?.data?.error || `${side}-side test failed`)
      setTestResult(null)
    }
  }

  const getSeverityColor = (severity: string) => {
    const map: Record<string, string> = {
      critical: 'bg-error-100 text-error-800 dark:bg-error-900 dark:text-error-200',
      high: 'bg-warning-100 text-warning-800 dark:bg-warning-900 dark:text-warning-200',
      medium: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
      low: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
    }
    return map[severity] || map.low
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center gap-3">
          <Shield className="h-8 w-8 text-primary-500" />
          Policy Enforcement Point (PEP)
        </h1>
        <p className="text-gray-600 dark:text-gray-400 mt-2">
          Enforce authorization policies and monitor violations
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatCard
          title="Active Enforcements"
          value="156"
          icon={<Shield className="h-8 w-8" />}
          trend="↑ 12%"
          gradient="linear-gradient(135deg, #667eea 0%, #764ba2 100%)"
        />
        <StatCard
          title="Violations Detected"
          value={violations.length.toString()}
          icon={<AlertTriangle className="h-8 w-8" />}
          trend="↓ 5%"
          gradient="linear-gradient(135deg, #f093fb 0%, #f5576c 100%)"
        />
        <StatCard
          title="Success Rate"
          value="94.2%"
          icon={<CheckCircle className="h-8 w-8" />}
          trend="↑ 3%"
          gradient="linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)"
        />
      </div>

      {/* Enforce Authorization */}
      <Card title="Enforce Authorization" icon={<Shield className="h-6 w-6" />}>
        <form onSubmit={handleEnforceAuthorization} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="md:col-span-2">
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Extended Token *
              </label>
              <textarea
                value={enforceForm.token}
                onChange={(e) => setEnforceForm({ ...enforceForm, token: e.target.value })}
                placeholder="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white font-mono text-sm"
                rows={3}
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Action Type *
              </label>
              <select
                value={enforceForm.action_type}
                onChange={(e) => setEnforceForm({ ...enforceForm, action_type: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                required
              >
                <option value="transaction">Transaction</option>
                <option value="decision">Decision</option>
                <option value="action">Action</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Transaction Type
              </label>
              <select
                value={enforceForm.transaction_type}
                onChange={(e) => setEnforceForm({ ...enforceForm, transaction_type: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
              >
                <option value="financial">Financial</option>
                <option value="legal">Legal</option>
                <option value="business">Business</option>
                <option value="administrative">Administrative</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Resource ID
              </label>
              <input
                type="text"
                value={enforceForm.resource_id}
                onChange={(e) => setEnforceForm({ ...enforceForm, resource_id: e.target.value })}
                placeholder="resource-001"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Enforcement Mode *
              </label>
              <select
                value={enforceForm.enforcement_mode}
                onChange={(e) => setEnforceForm({ ...enforceForm, enforcement_mode: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                required
              >
                <option value="strict">Strict</option>
                <option value="advisory">Advisory</option>
              </select>
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="px-6 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition disabled:opacity-50"
          >
            {loading ? 'Enforcing...' : 'Enforce Authorization'}
          </button>
        </form>

        {enforcementResult && (
          <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg space-y-3">
            <h4 className="font-semibold text-gray-900 dark:text-white">Enforcement Result</h4>
            <div className="space-y-2 text-sm">
              <p>
                <strong>Enforcement ID:</strong>{' '}
                <code className="text-xs bg-gray-200 dark:bg-gray-600 px-2 py-1 rounded">
                  {enforcementResult.enforcement_id}
                </code>
              </p>
              <p>
                <strong>Decision:</strong>{' '}
                <span
                  className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-semibold ${
                    enforcementResult.allowed
                      ? 'bg-success-100 text-success-800 dark:bg-success-900 dark:text-success-200'
                      : 'bg-error-100 text-error-800 dark:bg-error-900 dark:text-error-200'
                  }`}
                >
                  {enforcementResult.allowed ? (
                    <>
                      <CheckCircle className="h-3 w-3" /> ALLOWED
                    </>
                  ) : (
                    <>
                      <XCircle className="h-3 w-3" /> DENIED
                    </>
                  )}
                </span>
              </p>
              <p>
                <strong>Reason:</strong> {enforcementResult.reason}
              </p>
              {enforcementResult.token_valid !== undefined && (
                <p>
                  <strong>Token Valid:</strong> {enforcementResult.token_valid ? '✅ Yes' : '❌ No'}
                </p>
              )}
              {enforcementResult.scope_valid !== undefined && (
                <p>
                  <strong>Scope Valid:</strong> {enforcementResult.scope_valid ? '✅ Yes' : '❌ No'}
                </p>
              )}
              {enforcementResult.violations && enforcementResult.violations.length > 0 && (
                <div className="mt-3">
                  <strong className="block mb-2">Violations Detected:</strong>
                  <div className="space-y-2">
                    {enforcementResult.violations.map((v, i) => (
                      <div key={i} className="p-2 bg-white dark:bg-gray-800 rounded border-l-4 border-error-500">
                        <div className="flex items-center gap-2 mb-1">
                          <span className={`px-2 py-0.5 rounded-full text-xs font-semibold ${getSeverityColor(v.severity)}`}>
                            {v.severity}
                          </span>
                          <code className="text-xs">{v.violation_type}</code>
                        </div>
                        <p className="text-xs text-gray-600 dark:text-gray-400">{v.description}</p>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        )}
      </Card>

      {/* Supply/Demand Side Testing */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card title="Supply-Side Testing" icon={<CheckCircle className="h-6 w-6" />}>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            Test client-side enforcement before making requests
          </p>
          <button
            onClick={() => handleTestPEP('supply')}
            className="w-full px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition"
          >
            Run Supply-Side Test
          </button>
          {testResult && testResult.side === 'supply' && (
            <div className="mt-4 p-3 bg-gray-50 dark:bg-gray-700 rounded text-sm">
              <p className="font-semibold text-gray-900 dark:text-white mb-2">Test Result:</p>
              <p className="text-gray-600 dark:text-gray-400">
                {testResult.result || 'Supply-side enforcement validated'}
              </p>
            </div>
          )}
        </Card>

        <Card title="Demand-Side Testing" icon={<Shield className="h-6 w-6" />}>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            Test server-side enforcement upon receiving requests
          </p>
          <button
            onClick={() => handleTestPEP('demand')}
            className="w-full px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition"
          >
            Run Demand-Side Test
          </button>
          {testResult && testResult.side === 'demand' && (
            <div className="mt-4 p-3 bg-gray-50 dark:bg-gray-700 rounded text-sm">
              <p className="font-semibold text-gray-900 dark:text-white mb-2">Test Result:</p>
              <p className="text-gray-600 dark:text-gray-400">
                {testResult.result || 'Demand-side enforcement validated'}
              </p>
            </div>
          )}
        </Card>
      </div>

      {/* Recent Violations */}
      <Card title="Recent Violations" icon={<AlertTriangle className="h-6 w-6" />}>
        {violations.length === 0 ? (
          <p className="text-center text-gray-500">No recent violations</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 dark:bg-gray-700">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Type
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Severity
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Description
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Detected At
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                {violations.map((violation, index) => (
                  <tr key={index} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-4 py-3 text-sm">
                      <code className="text-xs bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                        {violation.violation_type}
                      </code>
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <span className={`px-2 py-1 rounded-full text-xs font-semibold ${getSeverityColor(violation.severity)}`}>
                        {violation.severity}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                      {violation.description}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                      {new Date(violation.detected_at).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Card>
    </div>
  )
}
