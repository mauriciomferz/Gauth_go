import { useState, useEffect } from 'react';
import { Card, StatCard } from '@/components/Card';
import { Button } from '@/components/Button';
import { Input, Select, Textarea } from '@/components/Form';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import { FileText, CheckCircle, Users, Calendar, Globe } from 'lucide-react';

interface PoARecord {
  id: string;
  grantor: string;
  representative: string;
  actions: string[];
  validUntil: string;
  status: 'active' | 'revoked' | 'expired';
}

export default function PoA() {
  const [grantor, setGrantor] = useState('');
  const [representative, setRepresentative] = useState('');
  const [representativeType, setRepresentativeType] = useState('legal');
  const [actions, setActions] = useState('');
  const [geoRestrictions, setGeoRestrictions] = useState('');
  const [validityPeriod, setValidityPeriod] = useState('365');
  const [loading, setLoading] = useState(false);
  const [validateToken, setValidateToken] = useState('');
  const [validateAction, setValidateAction] = useState('');
  const [validateLocation, setValidateLocation] = useState('');
  const [validationResult, setValidationResult] = useState<any>(null);
  const [activePoAs, setActivePoAs] = useState<PoARecord[]>([]);

  // Load active PoAs
  const loadActivePoAs = async () => {
    try {
      const poas = await apiClient.listPoAs();
      setActivePoAs(poas.map((poa: any) => ({
        id: poa.poaId,
        grantor: poa.grantor,
        representative: poa.representative,
        actions: poa.actions,
        validUntil: poa.validUntil ? new Date(poa.validUntil).toISOString().split('T')[0] : '',
        status: poa.status,
      })));
    } catch (error) {
      console.error('Failed to load PoAs:', error);
    }
  };

  // Load PoAs on mount
  useEffect(() => {
    loadActivePoAs();
  }, []);

  const handleCreate = async () => {
    if (!grantor || !representative || !actions) {
      toast.error('Please fill in all required fields');
      return;
    }

    setLoading(true);
    try {
      const actionList = actions.split(',').map((a) => a.trim());
      const geoList = geoRestrictions
        ? geoRestrictions.split(',').map((g) => g.trim())
        : [];

      const response: any = await apiClient.createPoA({
        grantor,
        representative,
        representativeType,
        actions: actionList,
        geographicScope: geoList.join(',') || '*',
        validityDays: parseInt(validityPeriod),
      });

      toast.success(`PoA created successfully! ID: ${response.poaId || response.delegation_id || response.id}`);
      
      // Refresh the active PoAs list
      await loadActivePoAs();
      
      // Clear form
      setGrantor('');
      setRepresentative('');
      setActions('');
      setGeoRestrictions('');
    } catch (error: any) {
      toast.error(error.message || 'PoA creation failed');
    } finally {
      setLoading(false);
    }
  };

  const handleValidate = async () => {
    if (!validateToken || !validateAction || !validateLocation) {
      toast.error('Please fill in all validation fields');
      return;
    }

    try {
      const response = await apiClient.validatePoA({
        poaId: validateToken,
        action: validateAction,
        location: validateLocation
      });
      setValidationResult(response);
      
      if (response.valid) {
        toast.success('PoA is valid');
      } else {
        toast.error('PoA is invalid or expired');
      }
    } catch (error: any) {
      toast.error(error.message || 'Validation failed');
      setValidationResult(null);
    }
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <div className="flex items-center gap-3 mb-2">
          <FileText className="w-8 h-8 text-primary" />
          <h1 className="text-3xl font-bold">Proof of Authorization (RFC-0115)</h1>
        </div>
        <p className="text-muted-foreground">
          Manage delegations with representative types, action scopes, and geographic restrictions
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
        <StatCard title="Active PoAs" value={activePoAs.length} icon={<FileText className="h-6 w-6" />} gradient="linear-gradient(135deg, #667eea 0%, #764ba2 100%)" trend="Current" />
        <StatCard title="Representatives" value={12} icon={<Users className="h-6 w-6" />} gradient="linear-gradient(135deg, #f093fb 0%, #f5576c 100%)" trend="Authorized" />
        <StatCard title="Avg Validity" value="247d" icon={<Calendar className="h-6 w-6" />} gradient="linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)" trend="Duration" />
        <StatCard title="Jurisdictions" value={8} icon={<Globe className="h-6 w-6" />} gradient="linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)" trend="Supported" />
      </div>

      {/* Create PoA Form */}
      <Card title="Create Proof of Authorization">
        <div className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Input
              label="Grantor (Owner)"
              value={grantor}
              onChange={(e) => setGrantor(e.target.value)}
              placeholder="alice@example.com"
              required
            />
            <Input
              label="Representative"
              value={representative}
              onChange={(e) => setRepresentative(e.target.value)}
              placeholder="bob@example.com"
              required
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Select
              label="Representative Type"
              value={representativeType}
              onChange={(e) => setRepresentativeType(e.target.value)}
            >
              <option value="legal">Legal Representative</option>
              <option value="commercial">Commercial Agent</option>
              <option value="technical">Technical Delegate</option>
              <option value="temporary">Temporary Proxy</option>
            </Select>
            <Input
              label="Validity Period (days)"
              type="number"
              value={validityPeriod}
              onChange={(e) => setValidityPeriod(e.target.value)}
              placeholder="365"
            />
          </div>

          <Textarea
            label="Authorized Actions (comma-separated)"
            value={actions}
            onChange={(e) => setActions(e.target.value)}
            placeholder="sign, approve, view, edit, delete"
            rows={2}
            required
          />

          <Textarea
            label="Geographic Restrictions (comma-separated, optional)"
            value={geoRestrictions}
            onChange={(e) => setGeoRestrictions(e.target.value)}
            placeholder="US, CA, GB, DE"
            rows={2}
          />

          <Button onClick={handleCreate} loading={loading}>
            Create Proof of Authorization
          </Button>
        </div>
      </Card>

      {/* Validate PoA */}
      <Card title="Validate Proof of Authorization">
        <div className="space-y-4">
          <Input
            label="PoA ID"
            value={validateToken}
            onChange={(e) => setValidateToken(e.target.value)}
            placeholder="del_123456..."
          />
          <Input
            label="Action"
            value={validateAction}
            onChange={(e) => setValidateAction(e.target.value)}
            placeholder="sign, approve, view, etc."
          />
          <Input
            label="Location (Country Code)"
            value={validateLocation}
            onChange={(e) => setValidateLocation(e.target.value)}
            placeholder="US, DE, FR, etc."
          />
          <Button onClick={handleValidate}>Validate PoA</Button>

          {validationResult && (
            <div
              className={`mt-4 p-4 rounded-lg border-2 ${
                validationResult.valid
                  ? 'border-green-500 dark:border-green-400 bg-green-50 dark:bg-green-900/20'
                  : 'border-red-500 dark:border-red-400 bg-red-50 dark:bg-red-900/20'
              }`}
            >
              <div className="flex items-center gap-2">
                <CheckCircle
                  className={`w-5 h-5 ${
                    validationResult.valid ? 'text-green-600' : 'text-red-600'
                  }`}
                />
                <p className="font-semibold">
                  {validationResult.valid ? 'Valid PoA' : 'Invalid PoA'}
                </p>
              </div>
              
              {validationResult.checks && validationResult.checks.length > 0 && (
                <div className="mt-4 space-y-2">
                  <p className="text-sm font-medium">Validation Checks:</p>
                  {validationResult.checks.map((check: any, idx: number) => (
                    <div key={idx} className="flex items-center gap-2 text-sm">
                      <span className={`inline-block w-2 h-2 rounded-full ${
                        check.result === 'pass' ? 'bg-green-500' : 'bg-red-500'
                      }`}></span>
                      <span>{check.check}: {check.result}</span>
                    </div>
                  ))}
                </div>
              )}
              
              <div className="mt-3 text-xs text-muted-foreground space-y-1">
                <p>PoA ID: {validationResult.poaId}</p>
                <p>Action: {validationResult.action}</p>
                <p>Location: {validationResult.location}</p>
                <p>Validated: {validationResult.validationTime && new Date(validationResult.validationTime).toLocaleString()}</p>
              </div>
            </div>
          )}
        </div>
      </Card>

      {/* Active PoAs Table */}
      <Card title="Active Powers of Attorney">
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="border-b border-border">
              <tr>
                <th className="text-left py-3 px-4 font-semibold">ID</th>
                <th className="text-left py-3 px-4 font-semibold">Grantor</th>
                <th className="text-left py-3 px-4 font-semibold">Representative</th>
                <th className="text-left py-3 px-4 font-semibold">Actions</th>
                <th className="text-left py-3 px-4 font-semibold">Valid Until</th>
                <th className="text-left py-3 px-4 font-semibold">Status</th>
              </tr>
            </thead>
            <tbody>
              {activePoAs.map((poa) => (
                <tr key={poa.id} className="border-b border-border hover:bg-muted/50">
                  <td className="py-3 px-4 font-mono text-xs">{poa.id}</td>
                  <td className="py-3 px-4">{poa.grantor}</td>
                  <td className="py-3 px-4">{poa.representative}</td>
                  <td className="py-3 px-4">
                    <div className="flex gap-1 flex-wrap">
                      {poa.actions.map((action, idx) => (
                        <span
                          key={idx}
                          className="px-2 py-1 rounded bg-primary/10 text-primary text-xs"
                        >
                          {action}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="py-3 px-4">{poa.validUntil}</td>
                  <td className="py-3 px-4">
                    <span
                      className={`px-2 py-1 rounded text-xs font-medium ${
                        poa.status === 'active'
                          ? 'bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400'
                          : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-400'
                      }`}
                    >
                      {poa.status}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      {/* RFC-0115 Features */}
      <Card title="RFC-0115 Compliance Features">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-green-600 mt-0.5" />
            <div>
              <p className="font-medium">Representative Types</p>
              <p className="text-sm text-muted-foreground">
                Legal, commercial, technical, and temporary proxies
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-green-600 mt-0.5" />
            <div>
              <p className="font-medium">Action Scopes</p>
              <p className="text-sm text-muted-foreground">
                Granular control over delegated actions
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-green-600 mt-0.5" />
            <div>
              <p className="font-medium">Geographic Restrictions</p>
              <p className="text-sm text-muted-foreground">
                Limit PoA validity to specific jurisdictions
              </p>
            </div>
          </div>
          <div className="flex items-start gap-3">
            <CheckCircle className="w-5 h-5 text-green-600 mt-0.5" />
            <div>
              <p className="font-medium">Time-bound Validity</p>
              <p className="text-sm text-muted-foreground">
                Automatic expiration and revocation support
              </p>
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
}
