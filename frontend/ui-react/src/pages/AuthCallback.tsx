import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { CheckCircle, XCircle } from 'lucide-react'
import { toast } from 'sonner'

interface AuthCallbackResponse {
  success: boolean
  sessionId: string
  userId: string
  user: {
    email: string
    name: string
    sub: string
  }
}

export default function AuthCallback() {
  const [loading, setLoading] = useState(true)
  const [success, setSuccess] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [userInfo, setUserInfo] = useState<any>(null)
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()

  const code = searchParams.get('code')
  const state = searchParams.get('state')
  const errorParam = searchParams.get('error')
  const errorDescription = searchParams.get('error_description')

  useEffect(() => {
    handleCallback()
  }, [])

  const handleCallback = async () => {
    // Check for errors from provider
    if (errorParam) {
      setLoading(false)
      setError(errorDescription || errorParam || 'Authentication failed')
      return
    }

    // Validate required parameters
    if (!code || !state) {
      setLoading(false)
      setError('Missing authorization code or state parameter')
      return
    }

    try {
      // The backend /auth/callback endpoint handles the OAuth flow
      const response = await fetch(`/auth/callback?code=${code}&state=${state}`)
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || `HTTP ${response.status}`)
      }

      const data: AuthCallbackResponse = await response.json()

      if (data.success) {
        // Store session information
        const sessionData = {
          sessionId: data.sessionId,
          userId: data.userId,
          user: data.user,
          timestamp: new Date().toISOString(),
        }

        // Store in localStorage for persistence
        localStorage.setItem('auth_session', JSON.stringify(sessionData))
        localStorage.setItem('user_id', data.userId)
        localStorage.setItem('user_email', data.user.email)

        setUserInfo(data.user)
        setSuccess(true)
        toast.success(`Welcome back, ${data.user.name || data.user.email}!`)

        // Redirect to return URL after a short delay
        const returnUrl = sessionStorage.getItem('auth_return_url') || '/admin/dashboard'
        sessionStorage.removeItem('auth_return_url')

        setTimeout(() => {
          navigate(returnUrl, { replace: true })
        }, 2000)
      } else {
        setError('Authentication succeeded but session creation failed')
      }
    } catch (err: any) {
      console.error('Authentication callback error:', err)
      const errorMessage = err.message || 'Authentication failed. Please try again.'
      setError(errorMessage)
      toast.error(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  const handleRetry = () => {
    navigate('/oidc-login', { replace: true })
  }

  const handleGoToDashboard = () => {
    navigate('/admin/dashboard', { replace: true })
  }

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100">
        <Card className="w-full max-w-md p-8 text-center">
          <div className="animate-spin rounded-full h-16 w-16 border-b-2 border-blue-600 mx-auto mb-4"></div>
          <h2 className="text-2xl font-bold text-gray-900 mb-2">
            Completing authentication...
          </h2>
          <p className="text-gray-600">
            Please wait while we verify your credentials
          </p>
        </Card>
      </div>
    )
  }

  if (success && userInfo) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100">
        <Card className="w-full max-w-md p-8 text-center">
          <CheckCircle className="w-20 h-20 text-green-500 mx-auto mb-4" />
          <h2 className="text-3xl font-bold text-gray-900 mb-2">
            Authentication Successful!
          </h2>
          <p className="text-gray-600 mb-6">
            Welcome back, {userInfo.name || userInfo.email}
          </p>

          <div className="bg-gray-50 rounded-lg p-4 mb-6 text-left">
            <p className="text-sm text-gray-700 mb-1">
              <strong>Email:</strong> {userInfo.email}
            </p>
            {userInfo.name && (
              <p className="text-sm text-gray-700">
                <strong>Name:</strong> {userInfo.name}
              </p>
            )}
          </div>

          <p className="text-sm text-gray-500 mb-4">
            Redirecting you to the dashboard...
          </p>

          <Button
            variant="primary"
            className="w-full"
            onClick={handleGoToDashboard}
          >
            Go to Dashboard Now
          </Button>
        </Card>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100">
        <Card className="w-full max-w-md p-8 text-center">
          <XCircle className="w-20 h-20 text-red-500 mx-auto mb-4" />
          <h2 className="text-3xl font-bold text-gray-900 mb-4">
            Authentication Failed
          </h2>

          <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6 text-left">
            <p className="text-red-800 text-sm">{error}</p>
          </div>

          <div className="flex gap-3">
            <Button variant="primary" className="flex-1" onClick={handleRetry}>
              Try Again
            </Button>
            <Button variant="secondary" className="flex-1" onClick={handleGoToDashboard}>
              Dashboard
            </Button>
          </div>

          <p className="text-sm text-gray-500 mt-6">
            If this problem persists, please contact your administrator.
          </p>
        </Card>
      </div>
    )
  }

  return null
}
