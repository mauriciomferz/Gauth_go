import { useState } from 'react'
import { Card, StatCard } from '../components/Card'
import { Button } from '../components/Button'
import { Input, Select } from '../components/Form'
import { UserCheck, ShieldCheck, CheckCircle, XCircle } from 'lucide-react'
import { apiClient, IdentityVerificationResponse } from '../lib/api'
import { toast } from 'sonner'

interface VerificationForm {
  type: 'individual' | 'legal_entity'
  trustLevel: 'substantial' | 'high'
  entityId: string
  tspId: string
}

const availableTSPs = [
  { id: 'tsp-001', name: 'eIDAS Trust Service Provider', country: 'EU' },
  { id: 'tsp-002', name: 'National ID Service', country: 'DE' },
  { id: 'tsp-003', name: 'UK Gov Verify', country: 'GB' },
  { id: 'tsp-004', name: 'US Fed Identity Service', country: 'US' },
]

export default function PVP() {
  const [form, setForm] = useState<VerificationForm>({
    type: 'individual',
    trustLevel: 'substantial',
    entityId: '',
    tspId: 'tsp-001',
  })
  const [result, setResult] = useState<IdentityVerificationResponse | null>(null)
  const [loading, setLoading] = useState(false)
  const [verificationHistory, setVerificationHistory] = useState<
    Array<{ entityId: string; verified: boolean; timestamp: string }>
  >([])

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const response = await apiClient.verifyIdentity({
        type: form.type,
        trustLevel: form.trustLevel,
        entityId: form.entityId,
        tsp: form.tspId,
      })
      setResult(response)
      setVerificationHistory([
        { entityId: form.entityId, verified: response.verified, timestamp: new Date().toISOString() },
        ...verificationHistory.slice(0, 4),
      ])
      if (response.verified) {
        toast.success('Identity verified successfully!')
      } else {
        toast.error('Identity verification failed')
      }
    } catch (error: any) {
      toast.error(error.response?.data?.message || 'Failed to verify identity')
      console.error('Verification error:', error)
    } finally {
      setLoading(false)
    }
  }

  const stats = [
    { 
      title: 'Total Verifications', 
      value: '1,247', 
      icon: <UserCheck className="h-6 w-6" />, 
      trend: '+12.5%',
      gradient: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
    },
    { 
      title: 'Success Rate', 
      value: '98.3%', 
      icon: <CheckCircle className="h-6 w-6" />, 
      trend: '+2.1%',
      gradient: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)'
    },
    { 
      title: 'Active TSPs', 
      value: availableTSPs.length.toString(), 
      icon: <ShieldCheck className="h-6 w-6" />,
      gradient: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)'
    },
    { 
      title: 'Avg Response Time', 
      value: '1.2s', 
      icon: <CheckCircle className="h-6 w-6" />, 
      trend: '-0.3s',
      gradient: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)'
    },
  ]

  return (
    <div className="space-y-6 animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
          PVP Identity Verification
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Person Verification Provider integration for RFC-0111 compliant identity verification with
          eIDAS trust levels.
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {stats.map((stat, idx) => (
          <StatCard key={idx} {...stat} />
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Verification Form */}
        <Card title="Verify Identity" icon={<UserCheck className="h-6 w-6" />}>
          <form onSubmit={handleVerify} className="space-y-4">
            <Select
              label="Identity Type"
              value={form.type}
              onChange={(e) => setForm({ ...form, type: e.target.value as any })}
              required
            >
              <option value="individual">Individual Person</option>
              <option value="legal_entity">Legal Entity</option>
            </Select>

            <Select
              label="Trust Level (eIDAS)"
              value={form.trustLevel}
              onChange={(e) => setForm({ ...form, trustLevel: e.target.value as any })}
              required
            >
              <option value="substantial">Substantial (Level 2)</option>
              <option value="high">High (Level 3)</option>
            </Select>

            <Input
              label="Entity ID / Passport / National ID"
              value={form.entityId}
              onChange={(e) => setForm({ ...form, entityId: e.target.value })}
              placeholder="DE123456789 or P12345678"
              required
            />

            <Select
              label="Trust Service Provider"
              value={form.tspId}
              onChange={(e) => setForm({ ...form, tspId: e.target.value })}
              required
            >
              {availableTSPs.map((tsp) => (
                <option key={tsp.id} value={tsp.id}>
                  {tsp.name} ({tsp.country})
                </option>
              ))}
            </Select>

            <Button type="submit" loading={loading} icon={<UserCheck className="h-4 w-4" />}>
              Verify Identity
            </Button>
          </form>

          {result && (
            <div
              className={`mt-6 p-4 rounded-lg border ${
                result.verified
                  ? 'bg-success-50 dark:bg-success-900/20 border-success-200 dark:border-success-800'
                  : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800'
              }`}
            >
              <div className="flex items-center gap-2 mb-3">
                {result.verified ? (
                  <CheckCircle className="h-5 w-5 text-success-600 dark:text-success-400" />
                ) : (
                  <XCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                )}
                <h4
                  className={`font-semibold ${
                    result.verified
                      ? 'text-success-900 dark:text-success-100'
                      : 'text-red-900 dark:text-red-100'
                  }`}
                >
                  {result.verified ? 'Identity Verified' : 'Verification Failed'}
                </h4>
              </div>
              <div className="space-y-2 text-sm">
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Entity ID:</span>
                  <span className="ml-2 font-mono text-gray-900 dark:text-white">
                    {result.entityId}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Trust Level:</span>
                  <span className="ml-2 font-semibold text-gray-900 dark:text-white">
                    {result.trustLevel}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">TSP:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">{result.tsp}</span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">TSP Status:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">{result.tspStatus}</span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Verification Time:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">{result.verificationTime}</span>
                </div>
              </div>
            </div>
          )}
        </Card>

        {/* Trust Service Providers */}
        <Card title="Available Trust Service Providers" icon={<ShieldCheck className="h-6 w-6" />}>
          <div className="space-y-3">
            {availableTSPs.map((tsp) => (
              <div
                key={tsp.id}
                className="flex items-center justify-between p-3 bg-gray-50 dark:bg-gray-700 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-600 transition-colors"
              >
                <div>
                  <div className="font-semibold text-gray-900 dark:text-white">{tsp.name}</div>
                  <div className="text-sm text-gray-600 dark:text-gray-400">ID: {tsp.id}</div>
                </div>
                <span className="px-2 py-1 bg-primary-100 dark:bg-primary-900 text-primary-800 dark:text-primary-200 text-xs font-semibold rounded">
                  {tsp.country}
                </span>
              </div>
            ))}
          </div>

          <div className="mt-6 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
            <h4 className="font-semibold text-blue-900 dark:text-blue-100 mb-2">
              eIDAS Trust Levels
            </h4>
            <ul className="space-y-1 text-sm text-blue-800 dark:text-blue-200">
              <li>• <strong>Substantial:</strong> Medium confidence (e.g., government ID)</li>
              <li>• <strong>High:</strong> Highest confidence (e.g., biometric verification)</li>
            </ul>
          </div>
        </Card>
      </div>

      {/* Verification History */}
      {verificationHistory.length > 0 && (
        <Card title="Recent Verifications" icon={<CheckCircle className="h-6 w-6" />}>
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-gray-200 dark:border-gray-700">
                  <th className="text-left p-3 text-gray-600 dark:text-gray-400 font-semibold">
                    Entity ID
                  </th>
                  <th className="text-left p-3 text-gray-600 dark:text-gray-400 font-semibold">
                    Status
                  </th>
                  <th className="text-left p-3 text-gray-600 dark:text-gray-400 font-semibold">
                    Timestamp
                  </th>
                </tr>
              </thead>
              <tbody>
                {verificationHistory.map((entry, idx) => (
                  <tr key={idx} className="border-b border-gray-100 dark:border-gray-800">
                    <td className="p-3 font-mono text-gray-900 dark:text-white">
                      {entry.entityId}
                    </td>
                    <td className="p-3">
                      <span
                        className={`px-2 py-1 rounded-full text-xs font-semibold ${
                          entry.verified
                            ? 'bg-success-100 dark:bg-success-900 text-success-800 dark:text-success-200'
                            : 'bg-red-100 dark:bg-red-900 text-red-800 dark:text-red-200'
                        }`}
                      >
                        {entry.verified ? 'Verified' : 'Failed'}
                      </span>
                    </td>
                    <td className="p-3 text-gray-600 dark:text-gray-400">
                      {new Date(entry.timestamp).toLocaleString()}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  )
}
