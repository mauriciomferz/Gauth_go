import { useState } from 'react'
import { Card, StatCard } from '../components/Card'
import { Button } from '../components/Button'
import { Input, Select } from '../components/Form'
import { Building2, CheckCircle, XCircle, Users, Shield } from 'lucide-react'
import { apiClient, EntityVerificationResponse, SignatoryVerificationResponse } from '../lib/api'
import { toast } from 'sonner'

interface EntityForm {
  jurisdiction: string
  registrationNumber: string
  entityName: string
}

interface SignatoryForm {
  entityId: string
  signatoryId: string
  documentType: string
}

const mockEntities = [
  { id: 'HRB12345-DE', name: 'Siemens AG', jurisdiction: 'DE', status: 'Active' },
  { id: '12345678-GB', name: 'British Airways PLC', jurisdiction: 'GB', status: 'Active' },
  { id: 'B123456-FR', name: 'Airbus SE', jurisdiction: 'FR', status: 'Active' },
  { id: 'CHE123456-CH', name: 'Nestlé SA', jurisdiction: 'CH', status: 'Active' },
]

export default function Registry() {
  const [entityForm, setEntityForm] = useState<EntityForm>({
    jurisdiction: 'DE',
    registrationNumber: '',
    entityName: '',
  })
  const [signatoryForm, setSignatoryForm] = useState<SignatoryForm>({
    entityId: '',
    signatoryId: '',
    documentType: 'certificate',
  })
  const [entityResult, setEntityResult] = useState<EntityVerificationResponse | null>(null)
  const [signatoryResult, setSignatoryResult] = useState<SignatoryVerificationResponse | null>(null)
  const [loading, setLoading] = useState(false)

  const handleVerifyEntity = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const result = await apiClient.verifyEntity({
        jurisdiction: entityForm.jurisdiction,
        registrationNumber: entityForm.registrationNumber,
      })
      setEntityResult(result)
      if (result.verified) {
        toast.success('Entity verified successfully!')
      } else {
        toast.error('Entity verification failed')
      }
    } catch (error: any) {
      toast.error(error.response?.data?.message || 'Failed to verify entity')
      console.error('Entity verification error:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleVerifySignatory = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const result = await apiClient.verifySignatory({
        entity: signatoryForm.entityId,
        signatoryName: signatoryForm.signatoryId,
        authorityType: signatoryForm.documentType,
      })
      setSignatoryResult(result)
      if (result.authorized) {
        toast.success('Signatory authorized!')
      } else {
        toast.error('Signatory not authorized')
      }
    } catch (error: any) {
      toast.error(error.response?.data?.message || 'Failed to verify signatory')
      console.error('Signatory verification error:', error)
    } finally {
      setLoading(false)
    }
  }

  const stats = [
    { 
      title: 'Verified Entities', 
      value: '2,347', 
      icon: <Building2 className="h-6 w-6" />, 
      trend: '+18.2%',
      gradient: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)'
    },
    { 
      title: 'Active Jurisdictions', 
      value: '47', 
      icon: <Shield className="h-6 w-6" />, 
      trend: '+3',
      gradient: 'linear-gradient(135deg, #f093fb 0%, #f5576c 100%)'
    },
    { 
      title: 'Authorized Signatories', 
      value: '8,912', 
      icon: <Users className="h-6 w-6" />, 
      trend: '+24.5%',
      gradient: 'linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)'
    },
    { 
      title: 'Success Rate', 
      value: '99.1%', 
      icon: <CheckCircle className="h-6 w-6" />, 
      trend: '+0.3%',
      gradient: 'linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)'
    },
  ]

  return (
    <div className="space-y-6 animate-fade-in">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
          Commercial Registry
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Verify legal entities and authorized signatories across international jurisdictions for RFC-0111 compliance.
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {stats.map((stat, idx) => (
          <StatCard key={idx} {...stat} />
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Entity Verification */}
        <Card title="Verify Legal Entity" icon={<Building2 className="h-6 w-6" />}>
          <form onSubmit={handleVerifyEntity} className="space-y-4">
            <Select
              label="Jurisdiction"
              value={entityForm.jurisdiction}
              onChange={(e) => setEntityForm({ ...entityForm, jurisdiction: e.target.value })}
              required
            >
              <option value="DE">Germany (DE)</option>
              <option value="GB">United Kingdom (GB)</option>
              <option value="FR">France (FR)</option>
              <option value="CH">Switzerland (CH)</option>
              <option value="US">United States (US)</option>
              <option value="NL">Netherlands (NL)</option>
              <option value="BE">Belgium (BE)</option>
              <option value="AT">Austria (AT)</option>
            </Select>

            <Input
              label="Registration Number"
              value={entityForm.registrationNumber}
              onChange={(e) => setEntityForm({ ...entityForm, registrationNumber: e.target.value })}
              placeholder="e.g., HRB12345 (DE), 12345678 (GB)"
              required
            />

            <Input
              label="Entity Name"
              value={entityForm.entityName}
              onChange={(e) => setEntityForm({ ...entityForm, entityName: e.target.value })}
              placeholder="Company legal name"
              required
            />

            <Button type="submit" loading={loading} icon={<Building2 className="h-4 w-4" />}>
              Verify Entity
            </Button>
          </form>

          {entityResult && (
            <div
              className={`mt-6 p-4 rounded-lg border ${
                entityResult.verified
                  ? 'bg-success-50 dark:bg-success-900/20 border-success-200 dark:border-success-800'
                  : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800'
              }`}
            >
              <div className="flex items-center gap-2 mb-3">
                {entityResult.verified ? (
                  <CheckCircle className="h-5 w-5 text-success-600 dark:text-success-400" />
                ) : (
                  <XCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                )}
                <h4
                  className={`font-semibold ${
                    entityResult.verified
                      ? 'text-success-900 dark:text-success-100'
                      : 'text-red-900 dark:text-red-100'
                  }`}
                >
                  {entityResult.verified ? 'Entity Verified' : 'Verification Failed'}
                </h4>
              </div>
              <div className="space-y-2 text-sm">
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Registration Number:</span>
                  <span className="ml-2 font-mono text-gray-900 dark:text-white">
                    {entityResult.registrationNumber}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Jurisdiction:</span>
                  <span className="ml-2 font-semibold text-gray-900 dark:text-white">
                    {entityResult.jurisdiction}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Legal Name:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">
                    {entityResult.legalName}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Status:</span>
                  <span className="ml-2 font-semibold text-gray-900 dark:text-white">
                    {entityResult.status}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Legal Form:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">
                    {entityResult.legalForm}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Registration Date:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">
                    {new Date(entityResult.registrationDate).toLocaleDateString()}
                  </span>
                </div>
              </div>
            </div>
          )}
        </Card>

        {/* Signatory Verification */}
        <Card title="Verify Authorized Signatory" icon={<Users className="h-6 w-6" />}>
          <form onSubmit={handleVerifySignatory} className="space-y-4">
            <Input
              label="Entity ID"
              value={signatoryForm.entityId}
              onChange={(e) => setSignatoryForm({ ...signatoryForm, entityId: e.target.value })}
              placeholder="e.g., HRB12345-DE"
              required
            />

            <Input
              label="Signatory ID / Passport"
              value={signatoryForm.signatoryId}
              onChange={(e) => setSignatoryForm({ ...signatoryForm, signatoryId: e.target.value })}
              placeholder="e.g., DE123456789"
              required
            />

            <Select
              label="Document Type"
              value={signatoryForm.documentType}
              onChange={(e) => setSignatoryForm({ ...signatoryForm, documentType: e.target.value })}
              required
            >
              <option value="certificate">Appointment Certificate</option>
              <option value="power_of_attorney">Proof of Authorization</option>
              <option value="board_resolution">Board Resolution</option>
              <option value="extract">Registry Extract</option>
            </Select>

            <Button
              type="submit"
              loading={loading}
              variant="secondary"
              icon={<Users className="h-4 w-4" />}
            >
              Verify Signatory
            </Button>
          </form>

          {signatoryResult && (
            <div
              className={`mt-6 p-4 rounded-lg border ${
                signatoryResult.authorized
                  ? 'bg-success-50 dark:bg-success-900/20 border-success-200 dark:border-success-800'
                  : 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800'
              }`}
            >
              <div className="flex items-center gap-2 mb-3">
                {signatoryResult.authorized ? (
                  <CheckCircle className="h-5 w-5 text-success-600 dark:text-success-400" />
                ) : (
                  <XCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
                )}
                <h4
                  className={`font-semibold ${
                    signatoryResult.authorized
                      ? 'text-success-900 dark:text-success-100'
                      : 'text-red-900 dark:text-red-100'
                  }`}
                >
                  {signatoryResult.authorized ? 'Signatory Authorized' : 'Not Authorized'}
                </h4>
              </div>
              <div className="space-y-2 text-sm">
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Signatory Name:</span>
                  <span className="ml-2 font-mono text-gray-900 dark:text-white">
                    {signatoryResult.signatoryName}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Entity:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">
                    {signatoryResult.entity}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Authority Type:</span>
                  <span className="ml-2 font-semibold text-gray-900 dark:text-white">
                    {signatoryResult.authorityType}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Status:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">
                    {signatoryResult.status}
                  </span>
                </div>
                <div>
                  <span className="text-gray-600 dark:text-gray-400">Appointment Date:</span>
                  <span className="ml-2 text-gray-900 dark:text-white">
                    {new Date(signatoryResult.appointmentDate).toLocaleDateString()}
                  </span>
                </div>
                {signatoryResult.restrictions && (
                  <div>
                    <span className="text-gray-600 dark:text-gray-400">Restrictions:</span>
                    <span className="ml-2 text-gray-900 dark:text-white">
                      {signatoryResult.restrictions}
                    </span>
                  </div>
                )}
              </div>
            </div>
          )}
        </Card>
      </div>

      {/* Mock Entities Table */}
      <Card title="Registered Entities (Sample)" icon={<Building2 className="h-6 w-6" />}>
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="border-b border-gray-200 dark:border-gray-700">
                <th className="text-left p-3 text-gray-600 dark:text-gray-400 font-semibold">
                  Entity ID
                </th>
                <th className="text-left p-3 text-gray-600 dark:text-gray-400 font-semibold">
                  Legal Name
                </th>
                <th className="text-left p-3 text-gray-600 dark:text-gray-400 font-semibold">
                  Jurisdiction
                </th>
                <th className="text-left p-3 text-gray-600 dark:text-gray-400 font-semibold">
                  Status
                </th>
              </tr>
            </thead>
            <tbody>
              {mockEntities.map((entity, idx) => (
                <tr key={idx} className="border-b border-gray-100 dark:border-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700">
                  <td className="p-3 font-mono text-gray-900 dark:text-white">{entity.id}</td>
                  <td className="p-3 text-gray-900 dark:text-white">{entity.name}</td>
                  <td className="p-3">
                    <span className="px-2 py-1 bg-primary-100 dark:bg-primary-900 text-primary-800 dark:text-primary-200 text-xs font-semibold rounded">
                      {entity.jurisdiction}
                    </span>
                  </td>
                  <td className="p-3">
                    <span className="px-2 py-1 bg-success-100 dark:bg-success-900 text-success-800 dark:text-success-200 text-xs font-semibold rounded-full">
                      {entity.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  )
}
