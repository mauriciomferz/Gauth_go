import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Cloud, Shield, Key, Lock } from 'lucide-react'
import { toast } from 'sonner'

interface OIDCProvider {
  id: string
  providerName: string
  providerType: string
  displayName: string
  status: string
  issuerUrl: string
}

const providerIcons: { [key: string]: any } = {
  azure_ad: Cloud,
  google: Cloud,
  okta: Shield,
  auth0: Key,
  custom: Lock,
}

const providerColors: { [key: string]: string } = {
  azure_ad: 'bg-blue-500',
  google: 'bg-red-500',
  okta: 'bg-cyan-500',
  auth0: 'bg-orange-500',
  custom: 'bg-gray-500',
}

export default function OIDCLogin() {
  const [providers, setProviders] = useState<OIDCProvider[]>([])
  const [loading, setLoading] = useState(true)
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()

  const tenantId = searchParams.get('tenant_id') || 'test-tenant-1'
  const returnUrl = searchParams.get('return_url') || '/admin/dashboard'
  const errorMessage = searchParams.get('error')

  useEffect(() => {
    fetchProviders()
  }, [])

  const fetchProviders = async () => {
    try {
      setLoading(true)
      const backendUrl = 'http://localhost:8080'
      const response = await fetch(`${backendUrl}/api/admin/oidc-providers`, {
        headers: {
          'X-Tenant-ID': tenantId,
        },
      })
      
      if (!response.ok) {
        throw new Error('Failed to fetch providers')
      }

      const data = await response.json()
      const activeProviders = data.providers?.filter(
        (p: OIDCProvider) => p.status === 'active'
      ) || []
      
      setProviders(activeProviders)
    } catch (err: any) {
      console.error('Failed to fetch OIDC providers:', err)
      toast.error('Failed to load authentication providers')
    } finally {
      setLoading(false)
    }
  }

  const handleProviderLogin = (providerId: string) => {
    // Backend is on port 8080, frontend is on 3001
    const backendUrl = 'http://localhost:8080'
    const callbackUrl = `${backendUrl}/auth/callback`
    const authUrl = new URL('/auth/authorize', backendUrl)
    
    authUrl.searchParams.set('provider_id', providerId)
    authUrl.searchParams.set('tenant_id', tenantId)
    authUrl.searchParams.set('redirect_uri', callbackUrl)
    
    console.log('🔐 OIDC Auth URL:', authUrl.toString())
    
    // Store return URL in session storage
    sessionStorage.setItem('auth_return_url', returnUrl)
    
    // Redirect to authorization endpoint
    window.location.href = authUrl.toString()
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100">
        <Card className="w-full max-w-md p-8">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto mb-4"></div>
            <p className="text-gray-600">Loading authentication providers...</p>
          </div>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <Card className="w-full max-w-md p-8">
        <div className="text-center mb-8">
          <Shield className="w-16 h-16 mx-auto mb-4 text-blue-600" />
          <h1 className="text-3xl font-bold text-gray-900 mb-2">Sign In</h1>
          <p className="text-gray-600">Choose your authentication provider</p>
        </div>

        {errorMessage && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-red-800 text-sm">{decodeURIComponent(errorMessage)}</p>
          </div>
        )}

        {providers.length === 0 ? (
          <div className="mb-6 p-4 bg-yellow-50 border border-yellow-200 rounded-lg">
            <p className="text-yellow-800 text-sm">
              No authentication providers configured. Please contact your administrator.
            </p>
          </div>
        ) : (
          <div className="space-y-3 mb-6">
            {providers.map((provider) => {
              const IconComponent = providerIcons[provider.providerType] || Lock
              const colorClass = providerColors[provider.providerType] || 'bg-gray-500'
              
              return (
                <button
                  key={provider.id}
                  onClick={() => handleProviderLogin(provider.id)}
                  className="w-full flex items-center gap-4 p-4 border-2 border-gray-200 rounded-lg hover:border-blue-500 hover:shadow-md transition-all duration-200 bg-white"
                >
                  <div className={`p-3 rounded-lg ${colorClass} text-white`}>
                    <IconComponent className="w-6 h-6" />
                  </div>
                  <div className="flex-1 text-left">
                    <div className="font-semibold text-gray-900">
                      {provider.displayName || provider.providerName}
                    </div>
                    <div className="text-sm text-gray-500">
                      {provider.providerType.replace('_', ' ').toUpperCase()}
                    </div>
                  </div>
                </button>
              )
            })}
          </div>
        )}

        <div className="relative mb-6">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-300"></div>
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="px-2 bg-white text-gray-500">or</span>
          </div>
        </div>

        <Button
          variant="secondary"
          className="w-full"
          onClick={() => navigate('/admin/login')}
        >
          Use Traditional Login
        </Button>

        <div className="mt-6 text-center">
          <button
            onClick={fetchProviders}
            className="text-sm text-blue-600 hover:text-blue-700"
          >
            Refresh providers
          </button>
        </div>

        <p className="mt-6 text-xs text-center text-gray-500">
          By signing in, you agree to our Terms of Service and Privacy Policy
        </p>
      </Card>
    </div>
  )
}
