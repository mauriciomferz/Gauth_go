import { useState, useEffect } from 'react'
import { Gavel, PlayCircle, History, TrendingUp, CheckCircle } from 'lucide-react'
import { Card, StatCard } from '../components/Card'
import { apiClient } from '../lib/api'
import { toast } from 'sonner'

interface Decision {
  decision_id: string
  subject: string
  resource: string
  action: string
  authorized: boolean
  reason: string
  timestamp: string
}

interface PDPMetrics {
  total_decisions: number
  permit_rate: number
  deny_rate: number
  avg_response_time: string | number
}

export default function PDP() {
  const [decisions, setDecisions] = useState<Decision[]>([])
  const [metrics, setMetrics] = useState<PDPMetrics>({
    total_decisions: 0,
    permit_rate: 0,
    deny_rate: 0,
    avg_response_time: 0
  })
  const [loading, setLoading] = useState(false)
  
  const [decisionForm, setDecisionForm] = useState({
    subject: '',
    resource: '',
    action: 'read',
    role: '',
    department: '',
    location: ''
  })

  const [evalForm, setEvalForm] = useState({
    policy_id: '',
    context: '{}'
  })

  const [decisionResult, setDecisionResult] = useState<any>(null)
  const [evalResult, setEvalResult] = useState<any>(null)

  useEffect(() => {
    loadRecentDecisions()
    loadMetrics()
  }, [])

  const loadRecentDecisions = async () => {
    try {
      const response = await apiClient.request<{ decisions: Decision[] }>({
        method: 'GET',
        url: '/pdp/decisions/recent'
      })
      setDecisions(response.data?.decisions || [])
    } catch (error) {
      console.error('Failed to load decisions:', error)
    }
  }

  const loadMetrics = async () => {
    try {
      const response = await apiClient.request<PDPMetrics>({
        method: 'GET',
        url: '/pdp/metrics'
      })
      if (response.data) {
        setMetrics(response.data)
      }
    } catch (error) {
      console.error('Failed to load metrics:', error)
    }
  }

  const handleMakeDecision = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      const context: any = {
        ip_address: '192.168.1.100',
        timestamp: new Date().toISOString()
      }

      if (decisionForm.role) context.role = decisionForm.role
      if (decisionForm.department) context.department = decisionForm.department
      if (decisionForm.location) context.location = decisionForm.location

      const payload = {
        subject: decisionForm.subject,
        resource: decisionForm.resource,
        action: decisionForm.action,
        context
      }

      const response = await apiClient.request({
        method: 'POST',
        url: '/pdp/decision',
        data: payload
      })

      setDecisionResult(response.data)
      toast.success('Decision evaluated successfully')
      loadRecentDecisions()
      loadMetrics()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Failed to evaluate decision')
      setDecisionResult(null)
    } finally {
      setLoading(false)
    }
  }

  const handleEvaluatePolicy = async (e: React.FormEvent) => {
    e.preventDefault()

    try {
      let context = {}
      try {
        context = JSON.parse(evalForm.context)
      } catch {
        toast.error('Invalid JSON in context field')
        return
      }

      const response = await apiClient.request({
        method: 'POST',
        url: `/pdp/evaluate/${evalForm.policy_id}`,
        data: { context }
      })

      setEvalResult(response.data)
      toast.success('Policy evaluated successfully')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Failed to evaluate policy')
      setEvalResult(null)
    }
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center gap-3">
          <Gavel className="h-8 w-8 text-primary-500" />
          Policy Decision Point (PDP)
        </h1>
        <p className="text-gray-600 dark:text-gray-400 mt-2">
          Make authorization decisions based on policies and context
        </p>
      </div>

      {/* Metrics */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="Total Decisions"
          value={metrics.total_decisions.toString()}
          icon={<Gavel className="h-8 w-8" />}
          gradient="linear-gradient(135deg, #667eea 0%, #764ba2 100%)"
        />
        <StatCard
          title="Permit Rate"
          value={`${metrics.permit_rate.toFixed(1)}%`}
          icon={<TrendingUp className="h-8 w-8" />}
          trend="↑"
          gradient="linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)"
        />
        <StatCard
          title="Deny Rate"
          value={`${metrics.deny_rate.toFixed(1)}%`}
          icon={<TrendingUp className="h-8 w-8" />}
          trend="↓"
          gradient="linear-gradient(135deg, #f093fb 0%, #f5576c 100%)"
        />
        <StatCard
          title="Avg Response"
          value={typeof metrics.avg_response_time === 'number' ? `${(metrics.avg_response_time as number).toFixed(0)}ms` : String(metrics.avg_response_time)}
          icon={<TrendingUp className="h-8 w-8" />}
          gradient="linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)"
        />
      </div>

      {/* Make Authorization Decision */}
      <Card title="Make Authorization Decision" icon={<PlayCircle className="h-6 w-6" />}>
        <form onSubmit={handleMakeDecision} className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Subject *
              </label>
              <input
                type="text"
                value={decisionForm.subject}
                onChange={(e) => setDecisionForm({ ...decisionForm, subject: e.target.value })}
                placeholder="user:alice@example.com"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Resource *
              </label>
              <input
                type="text"
                value={decisionForm.resource}
                onChange={(e) => setDecisionForm({ ...decisionForm, resource: e.target.value })}
                placeholder="/api/patients/12345"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Action *
              </label>
              <select
                value={decisionForm.action}
                onChange={(e) => setDecisionForm({ ...decisionForm, action: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                required
              >
                <option value="read">Read</option>
                <option value="write">Write</option>
                <option value="update">Update</option>
                <option value="delete">Delete</option>
                <option value="execute">Execute</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                User Role
              </label>
              <input
                type="text"
                value={decisionForm.role}
                onChange={(e) => setDecisionForm({ ...decisionForm, role: e.target.value })}
                placeholder="doctor"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Department
              </label>
              <input
                type="text"
                value={decisionForm.department}
                onChange={(e) => setDecisionForm({ ...decisionForm, department: e.target.value })}
                placeholder="cardiology"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                Location
              </label>
              <input
                type="text"
                value={decisionForm.location}
                onChange={(e) => setDecisionForm({ ...decisionForm, location: e.target.value })}
                placeholder="US-CA"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={loading}
            className="px-6 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition disabled:opacity-50"
          >
            {loading ? 'Evaluating...' : 'Evaluate Decision'}
          </button>
        </form>

        {decisionResult && (
          <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg space-y-2">
            <h4 className="font-semibold text-gray-900 dark:text-white">Decision Result</h4>
            <div className="space-y-2 text-sm">
              <p>
                <strong>Decision ID:</strong>{' '}
                <code className="text-xs bg-gray-200 dark:bg-gray-600 px-2 py-1 rounded">
                  {decisionResult.decision_id}
                </code>
              </p>
              <p>
                <strong>Decision:</strong>{' '}
                <span
                  className={`px-2 py-1 rounded-full text-xs font-semibold ${
                    decisionResult.authorized
                      ? 'bg-success-100 text-success-800 dark:bg-success-900 dark:text-success-200'
                      : 'bg-error-100 text-error-800 dark:bg-error-900 dark:text-error-200'
                  }`}
                >
                  {decisionResult.authorized ? 'PERMIT' : 'DENY'}
                </span>
              </p>
              <p>
                <strong>Reason:</strong> {decisionResult.reason}
              </p>
              {decisionResult.valid_until && (
                <p>
                  <strong>Valid Until:</strong> {new Date(decisionResult.valid_until).toLocaleString()}
                </p>
              )}
            </div>
          </div>
        )}
      </Card>

      {/* Policy Evaluation */}
      <Card title="Evaluate Policy" icon={<CheckCircle className="h-6 w-6" />}>
        <form onSubmit={handleEvaluatePolicy} className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Policy ID *
            </label>
            <input
              type="text"
              value={evalForm.policy_id}
              onChange={(e) => setEvalForm({ ...evalForm, policy_id: e.target.value })}
              placeholder="policy-001"
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              Context (JSON)
            </label>
            <textarea
              value={evalForm.context}
              onChange={(e) => setEvalForm({ ...evalForm, context: e.target.value })}
              placeholder='{"user":"alice", "role":"admin"}'
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white font-mono text-sm"
              rows={4}
            />
          </div>

          <button
            type="submit"
            className="px-6 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition"
          >
            Evaluate Policy
          </button>
        </form>

        {evalResult && (
          <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg space-y-2">
            <h4 className="font-semibold text-gray-900 dark:text-white">Evaluation Result</h4>
            <div className="space-y-2 text-sm">
              <p>
                <strong>Policy ID:</strong>{' '}
                <code className="text-xs bg-gray-200 dark:bg-gray-600 px-2 py-1 rounded">
                  {evalResult.policy_id}
                </code>
              </p>
              <p>
                <strong>Result:</strong>{' '}
                <span
                  className={`px-2 py-1 rounded-full text-xs font-semibold ${
                    evalResult.result === 'pass'
                      ? 'bg-success-100 text-success-800 dark:bg-success-900 dark:text-success-200'
                      : 'bg-error-100 text-error-800 dark:bg-error-900 dark:text-error-200'
                  }`}
                >
                  {evalResult.result || 'N/A'}
                </span>
              </p>
              <p>
                <strong>Details:</strong> {evalResult.details || 'Policy evaluated successfully'}
              </p>
            </div>
          </div>
        )}
      </Card>

      {/* Recent Decisions */}
      <Card title="Recent Decisions" icon={<History className="h-6 w-6" />}>
        {decisions.length === 0 ? (
          <p className="text-center text-gray-500">No recent decisions</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 dark:bg-gray-700">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Subject
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Resource
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Action
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Decision
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Reason
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Timestamp
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                {decisions.map((decision) => (
                  <tr key={decision.decision_id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-4 py-3 text-sm text-gray-900 dark:text-white">
                      {decision.subject}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                      {decision.resource}
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <code className="text-xs bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                        {decision.action}
                      </code>
                    </td>
                    <td className="px-4 py-3 text-sm">
                      <span
                        className={`px-2 py-1 rounded-full text-xs font-semibold ${
                          decision.authorized
                            ? 'bg-success-100 text-success-800 dark:bg-success-900 dark:text-success-200'
                            : 'bg-error-100 text-error-800 dark:bg-error-900 dark:text-error-200'
                        }`}
                      >
                        {decision.authorized ? 'PERMIT' : 'DENY'}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                      {decision.reason}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                      {new Date(decision.timestamp).toLocaleString()}
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
