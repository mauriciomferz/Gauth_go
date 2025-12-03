import { useState, useEffect } from 'react'
import { FileText, Plus, Search, CheckCircle, XCircle, Clock, Ban } from 'lucide-react'
import { Card } from '../components/Card'
import { apiClient } from '../lib/api'
import { toast } from 'sonner'

interface Policy {
  policy_id: string
  policy_name: string
  policy_type: string
  status: string
  created_at: string
  owners_authorizer: string
  description?: string
}

export default function PAP() {
  const [policies, setPolicies] = useState<Policy[]>([])
  const [loading, setLoading] = useState(false)
  const [searchId, setSearchId] = useState('')
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null)
  const [showCreateForm, setShowCreateForm] = useState(false)
  
  const [formData, setFormData] = useState({
    policy_name: '',
    policy_type: 'poa',
    description: '',
    client_owner: '',
    owners_authorizer: '',
    allowed_actions: '',
    countries: '',
    sectors: '',
    tags: ''
  })

  useEffect(() => {
    loadActivePolicies()
  }, [])

  const loadActivePolicies = async () => {
    try {
      setLoading(true)
      const response = await apiClient.request<{ policies: Policy[] }>({
        method: 'GET',
        url: '/pap/policies?status=active'
      })
      setPolicies(response.data?.policies || [])
    } catch (error) {
      console.error('Failed to load policies:', error)
      toast.error('Failed to load active policies')
    } finally {
      setLoading(false)
    }
  }

  const handleCreatePolicy = async (e: React.FormEvent) => {
    e.preventDefault()
    
    try {
      const payload = {
        policy_name: formData.policy_name,
        policy_type: formData.policy_type,
        description: formData.description,
        client_owner: formData.client_owner,
        owners_authorizer: formData.owners_authorizer,
        policy_rules: {
          allowed_actions: formData.allowed_actions.split(',').map(a => a.trim()).filter(a => a)
        },
        scope: {
          countries: formData.countries.split(',').map(c => c.trim()).filter(c => c),
          sectors: formData.sectors.split(',').map(s => s.trim()).filter(s => s)
        },
        tags: formData.tags.split(',').map(t => t.trim()).filter(t => t)
      }

      await apiClient.request({
        method: 'POST',
        url: '/pap/policies',
        data: payload
      })

      toast.success('Policy created successfully')
      setShowCreateForm(false)
      setFormData({
        policy_name: '',
        policy_type: 'poa',
        description: '',
        client_owner: '',
        owners_authorizer: '',
        allowed_actions: '',
        countries: '',
        sectors: '',
        tags: ''
      })
      loadActivePolicies()
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Failed to create policy')
    }
  }

  const handleSearchPolicy = async () => {
    if (!searchId.trim()) {
      toast.error('Please enter a policy ID')
      return
    }

    try {
      const response = await apiClient.request<Policy>({
        method: 'GET',
        url: `/pap/policies/${searchId}`
      })
      setSelectedPolicy(response.data)
      toast.success('Policy found')
    } catch (error: any) {
      toast.error(error.response?.data?.error || 'Policy not found')
      setSelectedPolicy(null)
    }
  }

  const handlePolicyAction = async (policyId: string, action: string) => {
    if (!confirm(`Are you sure you want to ${action} this policy?`)) {
      return
    }

    try {
      await apiClient.request({
        method: 'POST',
        url: `/pap/policies/${policyId}/${action}`,
        data: action === 'delete' ? undefined : { reason: `${action} via UI` }
      })

      toast.success(`Policy ${action}d successfully`)
      loadActivePolicies()
      setSelectedPolicy(null)
    } catch (error: any) {
      toast.error(error.response?.data?.error || `Failed to ${action} policy`)
    }
  }

  const getStatusBadge = (status: string) => {
    const statusMap: Record<string, { color: string; icon: any }> = {
      active: { color: 'bg-success-100 text-success-800 dark:bg-success-900 dark:text-success-200', icon: CheckCircle },
      draft: { color: 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200', icon: Clock },
      suspended: { color: 'bg-warning-100 text-warning-800 dark:bg-warning-900 dark:text-warning-200', icon: Ban },
      revoked: { color: 'bg-error-100 text-error-800 dark:bg-error-900 dark:text-error-200', icon: XCircle }
    }
    
    const config = statusMap[status] || statusMap.draft
    const Icon = config.icon

    return (
      <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-semibold ${config.color}`}>
        <Icon className="h-3 w-3" />
        {status.toUpperCase()}
      </span>
    )
  }

  return (
    <div className="space-y-6 animate-fade-in">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white flex items-center gap-3">
            <FileText className="h-8 w-8 text-primary-500" />
            Policy Administration Point (PAP)
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mt-2">
            Create, manage, and control authorization policies
          </p>
        </div>
        <button
          onClick={() => setShowCreateForm(!showCreateForm)}
          className="flex items-center gap-2 px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition"
        >
          <Plus className="h-4 w-4" />
          Create Policy
        </button>
      </div>

      {/* Create Policy Form */}
      {showCreateForm && (
        <Card title="Create New Policy" icon={<Plus className="h-6 w-6" />}>
          <form onSubmit={handleCreatePolicy} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Policy Name *
                </label>
                <input
                  type="text"
                  value={formData.policy_name}
                  onChange={(e) => setFormData({ ...formData, policy_name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Policy Type *
                </label>
                <select
                  value={formData.policy_type}
                  onChange={(e) => setFormData({ ...formData, policy_type: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                  required
                >
                  <option value="poa">Power of Attorney</option>
                  <option value="authorization_chain">Authorization Chain</option>
                  <option value="scope">Scope Restriction</option>
                  <option value="restriction">Power Restriction</option>
                  <option value="compliance">Compliance</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Client Owner *
                </label>
                <input
                  type="text"
                  value={formData.client_owner}
                  onChange={(e) => setFormData({ ...formData, client_owner: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                  placeholder="client-001"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Owner's Authorizer *
                </label>
                <input
                  type="text"
                  value={formData.owners_authorizer}
                  onChange={(e) => setFormData({ ...formData, owners_authorizer: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                  placeholder="authorizer-001"
                  required
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Allowed Actions (comma-separated)
                </label>
                <input
                  type="text"
                  value={formData.allowed_actions}
                  onChange={(e) => setFormData({ ...formData, allowed_actions: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                  placeholder="read, write, delete"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Countries (comma-separated)
                </label>
                <input
                  type="text"
                  value={formData.countries}
                  onChange={(e) => setFormData({ ...formData, countries: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                  placeholder="US, CA, MX"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Sectors (comma-separated)
                </label>
                <input
                  type="text"
                  value={formData.sectors}
                  onChange={(e) => setFormData({ ...formData, sectors: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                  placeholder="healthcare, finance"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Tags (comma-separated)
                </label>
                <input
                  type="text"
                  value={formData.tags}
                  onChange={(e) => setFormData({ ...formData, tags: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                  placeholder="production, critical"
                />
              </div>
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                  Description
                </label>
                <textarea
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
                  rows={3}
                  placeholder="Policy description..."
                />
              </div>
            </div>
            <div className="flex gap-3">
              <button
                type="submit"
                className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition"
              >
                Create Policy
              </button>
              <button
                type="button"
                onClick={() => setShowCreateForm(false)}
                className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded-lg hover:bg-gray-300 dark:hover:bg-gray-600 transition"
              >
                Cancel
              </button>
            </div>
          </form>
        </Card>
      )}

      {/* Search Policy */}
      <Card title="Search Policy" icon={<Search className="h-6 w-6" />}>
        <div className="flex gap-3">
          <input
            type="text"
            value={searchId}
            onChange={(e) => setSearchId(e.target.value)}
            placeholder="Enter policy ID..."
            className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg focus:ring-2 focus:ring-primary-500 dark:bg-gray-700 dark:text-white"
          />
          <button
            onClick={handleSearchPolicy}
            className="px-4 py-2 bg-primary-500 text-white rounded-lg hover:bg-primary-600 transition"
          >
            <Search className="h-4 w-4" />
          </button>
        </div>

        {selectedPolicy && (
          <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg space-y-2">
            <div className="flex items-center justify-between">
              <h4 className="font-semibold text-gray-900 dark:text-white">{selectedPolicy.policy_name}</h4>
              {getStatusBadge(selectedPolicy.status)}
            </div>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              <strong>ID:</strong> <code className="text-xs bg-gray-200 dark:bg-gray-600 px-2 py-1 rounded">{selectedPolicy.policy_id}</code>
            </p>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              <strong>Type:</strong> {selectedPolicy.policy_type}
            </p>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              <strong>Authorizer:</strong> {selectedPolicy.owners_authorizer}
            </p>
            {selectedPolicy.description && (
              <p className="text-sm text-gray-600 dark:text-gray-400">
                <strong>Description:</strong> {selectedPolicy.description}
              </p>
            )}
            <div className="flex gap-2 pt-2">
              {selectedPolicy.status === 'draft' && (
                <button
                  onClick={() => handlePolicyAction(selectedPolicy.policy_id, 'activate')}
                  className="px-3 py-1 bg-success-500 text-white rounded text-sm hover:bg-success-600"
                >
                  Activate
                </button>
              )}
              {selectedPolicy.status === 'active' && (
                <button
                  onClick={() => handlePolicyAction(selectedPolicy.policy_id, 'suspend')}
                  className="px-3 py-1 bg-warning-500 text-white rounded text-sm hover:bg-warning-600"
                >
                  Suspend
                </button>
              )}
              {(selectedPolicy.status === 'draft' || selectedPolicy.status === 'revoked') && (
                <button
                  onClick={() => handlePolicyAction(selectedPolicy.policy_id, 'delete')}
                  className="px-3 py-1 bg-error-500 text-white rounded text-sm hover:bg-error-600"
                >
                  Delete
                </button>
              )}
              <button
                onClick={() => handlePolicyAction(selectedPolicy.policy_id, 'revoke')}
                className="px-3 py-1 bg-error-500 text-white rounded text-sm hover:bg-error-600"
              >
                Revoke
              </button>
            </div>
          </div>
        )}
      </Card>

      {/* Active Policies */}
      <Card title="Active Policies" icon={<CheckCircle className="h-6 w-6" />}>
        {loading ? (
          <p className="text-center text-gray-500">Loading policies...</p>
        ) : policies.length === 0 ? (
          <p className="text-center text-gray-500">No active policies found</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead className="bg-gray-50 dark:bg-gray-700">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Policy Name
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Type
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Status
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Authorizer
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                    Created
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                {policies.map((policy) => (
                  <tr key={policy.policy_id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                    <td className="px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">
                      {policy.policy_name}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                      <code className="text-xs bg-gray-100 dark:bg-gray-700 px-2 py-1 rounded">
                        {policy.policy_type}
                      </code>
                    </td>
                    <td className="px-4 py-3 text-sm">
                      {getStatusBadge(policy.status)}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                      {policy.owners_authorizer}
                    </td>
                    <td className="px-4 py-3 text-sm text-gray-600 dark:text-gray-400">
                      {new Date(policy.created_at).toLocaleDateString()}
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
