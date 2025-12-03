import { useState, useEffect } from 'react';
import { Card, StatCard } from '@/components/Card';
import { Button } from '@/components/Button';
import { Input, Textarea } from '@/components/Form';
import { apiClient } from '@/lib/api';
import { toast } from 'sonner';
import { Shield, CheckCircle, XCircle, Database, Clock, Activity } from 'lucide-react';

interface ValidationResult {
  allowed: boolean;
  policies: string[];
  reason: string;
  evaluationTime: number;
}

export default function PIP() {
  const [tokenId, setTokenId] = useState('');
  const [resource, setResource] = useState('');
  const [action, setAction] = useState('');
  const [context, setContext] = useState('{}');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<ValidationResult | null>(null);
  const [cacheStats, setCacheStats] = useState({ hits: 0, misses: 0, hitRate: '0%' });
  const [policies, setPolicies] = useState<any[]>([]);
  const [loadingPolicies, setLoadingPolicies] = useState(true);

  // Load cache stats and policies on component mount
  useEffect(() => {
    loadInitialData();
  }, []);

  const loadInitialData = async () => {
    try {
      // Fetch real cache statistics
      const stats = await apiClient.getAuthzCacheMetrics();
      setCacheStats({
        hits: stats.hits,
        misses: stats.misses,
        hitRate: `${(stats.hitRate * 100).toFixed(1)}%`
      });

      // Fetch active policies
      setLoadingPolicies(true);
      const activePolicies = await apiClient.getActivePolicies();
      setPolicies(activePolicies);
    } catch (error) {
      console.error('Failed to load initial data:', error);
    } finally {
      setLoadingPolicies(false);
    }
  };

  const handleValidate = async () => {
    if (!tokenId || !resource || !action) {
      toast.error('Please fill in all required fields');
      return;
    }

    setLoading(true);
    try {
      let parsedContext = {};
      if (context.trim()) {
        parsedContext = JSON.parse(context);
      }

      const response = await apiClient.validateAuthorization({
        clientId: tokenId,
        action,
        geographic: resource,
        sector: JSON.stringify(parsedContext),
      });

      // Map AuthorizationResponse to ValidationResult
      setResult({
        allowed: response.allowed || response.authorized,
        policies: response.policies || [],
        reason: response.allowed ? 'Authorization granted' : 'Authorization denied',
        evaluationTime: response.evaluationTime || 0
      });
      
      if (response.allowed) {
        toast.success('Authorization granted');
      } else {
        toast.error('Authorization denied');
      }

      // Update cache stats from real backend
      const stats = await apiClient.getAuthzCacheMetrics();
      setCacheStats({
        hits: stats.hits,
        misses: stats.misses,
        hitRate: `${(stats.hitRate * 100).toFixed(1)}%`
      });
    } catch (error: any) {
      toast.error(error.message || 'Authorization validation failed');
      setResult(null);
    } finally {
      setLoading(false);
    }
  };

  const handleClear = () => {
    setTokenId('');
    setResource('');
    setAction('');
    setContext('{}');
    setResult(null);
  };

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <div className="flex items-center gap-3 mb-2">
          <Shield className="w-8 h-8 text-primary" />
          <h1 className="text-3xl font-bold">Policy Information Point (PIP)</h1>
        </div>
        <p className="text-muted-foreground">
          Validate authorization policies and check access control decisions
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatCard
          title="Cache Hits"
          value={cacheStats.hits}
          icon={<Database className="h-6 w-6" />}
          gradient="linear-gradient(135deg, #667eea 0%, #764ba2 100%)"
          trend="High efficiency"
        />
        <StatCard
          title="Cache Misses"
          value={cacheStats.misses}
          icon={<Activity className="h-6 w-6" />}
          gradient="linear-gradient(135deg, #f093fb 0%, #f5576c 100%)"
          trend="Low rate"
        />
        <StatCard
          title="Hit Rate"
          value={cacheStats.hitRate}
          icon={<CheckCircle className="h-6 w-6" />}
          gradient="linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)"
          trend="Excellent"
        />
      </div>

      {/* Authorization Validation Form */}
      <Card title="Validate Authorization">
        <div className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Input
              label="Token ID"
              value={tokenId}
              onChange={(e) => setTokenId(e.target.value)}
              placeholder="token_abc123..."
              required
            />
            <Input
              label="Resource"
              value={resource}
              onChange={(e) => setResource(e.target.value)}
              placeholder="/api/users"
              required
            />
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <Input
              label="Action"
              value={action}
              onChange={(e) => setAction(e.target.value)}
              placeholder="read, write, delete"
              required
            />
          </div>

          <Textarea
            label="Context (JSON)"
            value={context}
            onChange={(e) => setContext(e.target.value)}
            placeholder='{"ip": "192.168.1.1", "time": "2025-11-12T10:00:00Z"}'
            rows={4}
          />

          <div className="flex gap-3">
            <Button onClick={handleValidate} loading={loading}>
              Validate Authorization
            </Button>
            <Button variant="secondary" onClick={handleClear}>
              Clear
            </Button>
          </div>
        </div>
      </Card>

      {/* Result Display */}
      {result && (
        <Card
          title="Authorization Result"
          className={`border-2 ${
            result.allowed
              ? 'border-green-500 dark:border-green-400'
              : 'border-red-500 dark:border-red-400'
          }`}
        >
          <div className="space-y-4">
            {/* Decision */}
            <div className="flex items-center gap-3">
              {result.allowed ? (
                <CheckCircle className="w-6 h-6 text-green-600 dark:text-green-400" />
              ) : (
                <XCircle className="w-6 h-6 text-red-600 dark:text-red-400" />
              )}
              <div>
                <p className="font-semibold text-lg">
                  {result.allowed ? 'Access Granted' : 'Access Denied'}
                </p>
                <p className="text-sm text-muted-foreground">{result.reason}</p>
              </div>
            </div>

            {/* Evaluation Time */}
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <Clock className="w-4 h-4" />
              <span>Evaluation time: {result.evaluationTime}ms</span>
            </div>

            {/* Applied Policies */}
            {result.policies && result.policies.length > 0 && (
              <div>
                <p className="font-semibold mb-2">Applied Policies:</p>
                <ul className="list-disc list-inside space-y-1">
                  {result.policies.map((policy, index) => (
                    <li key={index} className="text-sm text-muted-foreground">
                      {policy}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
        </Card>
      )}

      {/* Active Policies Info */}
      <Card title="Active Policy Rules">
        {loadingPolicies ? (
          <div className="text-center py-8 text-muted-foreground">
            Loading active policies...
          </div>
        ) : policies.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            No active policies found. Policies will appear here when loaded from the backend.
          </div>
        ) : (
          <div className="space-y-3">
            {policies.map((policy) => (
              <PolicyRule
                key={policy.id}
                name={policy.name}
                description={policy.description}
                status={policy.status}
              />
            ))}
          </div>
        )}
      </Card>
    </div>
  );
}

interface PolicyRuleProps {
  name: string;
  description: string;
  status: 'active' | 'inactive';
}

function PolicyRule({ name, description, status }: PolicyRuleProps) {
  return (
    <div className="flex items-start gap-3 p-3 rounded-lg bg-muted/50">
      <div
        className={`w-2 h-2 rounded-full mt-2 ${
          status === 'active' ? 'bg-green-500' : 'bg-gray-400'
        }`}
      />
      <div>
        <p className="font-medium">{name}</p>
        <p className="text-sm text-muted-foreground">{description}</p>
      </div>
    </div>
  );
}
