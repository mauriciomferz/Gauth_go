import { CheckCircle, Zap, Shield, TrendingUp, Activity } from 'lucide-react'
import { StatCard } from '../components/Card'
import { Card } from '../components/Card'
import { useState, useEffect } from 'react'
import { apiClient } from '../lib/api'

export default function Overview() {
  const [backendStatus, setBackendStatus] = useState<'checking' | 'healthy' | 'error'>('checking')
  const [backendUptime, setBackendUptime] = useState<string>('')

  useEffect(() => {
    const checkBackend = async () => {
      try {
        const health = await apiClient.health()
        if (health.data?.status === 'healthy') {
          setBackendStatus('healthy')
          setBackendUptime(health.data.uptime || 'Unknown')
        } else {
          setBackendStatus('error')
        }
      } catch (error) {
        setBackendStatus('error')
        console.error('Backend health check failed:', error)
      }
    }

    checkBackend()
    const interval = setInterval(checkBackend, 30000) // Check every 30s
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="space-y-8 animate-fade-in">
      {/* Hero */}
      <div className="text-center py-12">
        <div className="inline-flex items-center gap-3 mb-4">
          <Shield className="h-16 w-16 text-primary-500" />
          <h1 className="text-5xl font-bold bg-gradient-to-r from-primary-500 to-purple-600 bg-clip-text text-transparent">
            Welcome to GAuth 1.0
          </h1>
        </div>
        <p className="text-xl text-gray-600 dark:text-gray-400 mt-4">
          <strong>RFC-0111 & RFC-0115</strong> Compliant Authorization Framework
        </p>
        <div className="flex items-center justify-center gap-3 mt-3">
          <p className="text-sm text-gray-500 dark:text-gray-500">
            91 passing tests • 72.6% coverage
          </p>
          <span className="text-gray-300 dark:text-gray-600">•</span>
          <div className="flex items-center gap-2">
            <Activity className={`h-4 w-4 ${
              backendStatus === 'healthy' ? 'text-success-500 animate-pulse' : 
              backendStatus === 'error' ? 'text-error-500' : 
              'text-gray-400 animate-spin'
            }`} />
            <span className={`text-sm font-medium ${
              backendStatus === 'healthy' ? 'text-success-600 dark:text-success-400' : 
              backendStatus === 'error' ? 'text-error-600 dark:text-error-400' : 
              'text-gray-500'
            }`}>
              Backend {backendStatus === 'checking' ? 'Checking...' : backendStatus}
              {backendStatus === 'healthy' && backendUptime && ` (${backendUptime})`}
            </span>
          </div>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <StatCard
          title="Tests Passing"
          value="91"
          icon={<CheckCircle className="h-8 w-8" />}
          trend="↑ 100%"
          gradient="linear-gradient(135deg, #667eea 0%, #764ba2 100%)"
        />
        <StatCard
          title="Benchmarks"
          value="19"
          icon={<Zap className="h-8 w-8" />}
          trend="All Passing"
          gradient="linear-gradient(135deg, #f093fb 0%, #f5576c 100%)"
        />
        <StatCard
          title="Test Coverage"
          value="72.6%"
          icon={<Shield className="h-8 w-8" />}
          trend="Excellent"
          gradient="linear-gradient(135deg, #4facfe 0%, #00f2fe 100%)"
        />
        <StatCard
          title="E2E Token Flow"
          value="1.3µs"
          icon={<TrendingUp className="h-8 w-8" />}
          trend="Ultra-Fast"
          gradient="linear-gradient(135deg, #43e97b 0%, #38f9d7 100%)"
        />
      </div>

      {/* Features Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <Card title="RFC Compliance" icon={<CheckCircle className="h-6 w-6" />}>
          <div className="space-y-3">
            <div className="flex items-start gap-3">
              <CheckCircle className="h-5 w-5 text-success-500 mt-0.5" />
              <div>
                <p className="font-semibold text-gray-900 dark:text-white">RFC-0111</p>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  GAuth 1.0 Extended Tokens
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <CheckCircle className="h-5 w-5 text-success-500 mt-0.5" />
              <div>
                <p className="font-semibold text-gray-900 dark:text-white">RFC-0115</p>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  Power of Attorney Framework
                </p>
              </div>
            </div>
            <div className="flex items-start gap-3">
              <CheckCircle className="h-5 w-5 text-success-500 mt-0.5" />
              <div>
                <p className="font-semibold text-gray-900 dark:text-white">eIDAS</p>
                <p className="text-sm text-gray-600 dark:text-gray-400">
                  Trust Service Provider Integration
                </p>
              </div>
            </div>
          </div>
        </Card>

        <Card title="System Components" icon={<Shield className="h-6 w-6" />}>
          <div className="space-y-2">
            {[
              'Extended Token',
              'PVP (Identity Verification)',
              'Commercial Register',
              'PIP (Policy Info Point)',
              'PoA (Power of Attorney)',
            ].map((component) => (
              <div
                key={component}
                className="flex items-center justify-between p-2 bg-gray-50 dark:bg-gray-700 rounded"
              >
                <span className="text-sm font-medium text-gray-900 dark:text-white">
                  {component}
                </span>
                <span className="px-2 py-0.5 bg-success-100 dark:bg-success-900 text-success-800 dark:text-success-200 text-xs font-semibold rounded-full">
                  Active
                </span>
              </div>
            ))}
          </div>
        </Card>

        <Card title="Quick Start" icon={<TrendingUp className="h-6 w-6" />}>
          <ol className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
            <li className="flex gap-2">
              <span className="font-semibold text-primary-500">1.</span>
              Navigate to <strong>Extended Tokens</strong> to create and validate tokens
            </li>
            <li className="flex gap-2">
              <span className="font-semibold text-primary-500">2.</span>
              Use <strong>PVP</strong> to verify identity chains with eIDAS
            </li>
            <li className="flex gap-2">
              <span className="font-semibold text-primary-500">3.</span>
              Check <strong>Commercial Register</strong> for entity validation
            </li>
            <li className="flex gap-2">
              <span className="font-semibold text-primary-500">4.</span>
              Test <strong>PIP</strong> for authorization decisions
            </li>
            <li className="flex gap-2">
              <span className="font-semibold text-primary-500">5.</span>
              Manage <strong>PoA</strong> delegations and permissions
            </li>
          </ol>
        </Card>
      </div>
    </div>
  )
}
