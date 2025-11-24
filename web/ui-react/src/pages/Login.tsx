import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Card } from '@/components/Card'
import { Button } from '@/components/Button'
import { Input } from '@/components/Form'
import { apiClient, LoginRequest, MFAVerifyRequest, LoginInitResponse } from '@/lib/api'
import { toast } from 'sonner'
import { Shield, Key, Lock } from 'lucide-react'

type Step = 'credentials' | 'mfa' | 'success'

export default function Login() {
  const navigate = useNavigate()
  const [step, setStep] = useState<Step>('credentials')
  const [loading, setLoading] = useState(false)
  const [form, setForm] = useState<LoginRequest>({ username: '', password: '' })
  const [mfaCode, setMfaCode] = useState('')
  const [loginInit, setLoginInit] = useState<LoginInitResponse | null>(null)
  const [mfaMethod, setMfaMethod] = useState('totp')

  const handleCredentials = async () => {
    if (!form.username || !form.password) {
      toast.error('Enter username and password')
      return
    }
    setLoading(true)
    try {
      const init = await apiClient.initiateLogin(form)
      setLoginInit(init)
      if (!init.success) {
        toast.error(init.error || 'Login failed')
        return
      }
      if (init.requiresMFA) {
        setStep('mfa')
        toast.info('MFA required. Enter your code.')
      } else {
        setStep('success')
        toast.success('Login successful (no MFA required)')
      }
    } catch (e: any) {
      toast.error(e.message || 'Login error')
    } finally {
      setLoading(false)
    }
  }

  const handleMFA = async () => {
    if (!loginInit?.sessionChallenge) {
      toast.error('No active challenge')
      return
    }
    if (!mfaCode) {
      toast.error('Enter MFA code')
      return
    }
    setLoading(true)
    try {
      const payload: MFAVerifyRequest = {
        challengeId: loginInit.sessionChallenge,
        code: mfaCode,
        method: mfaMethod,
      }
      const res = await apiClient.verifyMFA(payload)
      if (res.success) {
        setStep('success')
        toast.success('MFA verified, login complete')
      } else {
        toast.error(res.error || 'Invalid code')
      }
    } catch (e: any) {
      toast.error(e.message || 'MFA verification failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-md mx-auto space-y-6">
      <div className="flex items-center gap-3 justify-center">
        <Shield className="h-10 w-10 text-primary-500" />
        <h1 className="text-2xl font-bold">Secure Login</h1>
      </div>
      {step === 'credentials' && (
        <Card title="Credentials" icon={<Key className="h-6 w-6" />}>        
          <div className="space-y-4">
            <Input
              label="Username"
              value={form.username}
              onChange={(e) => setForm({ ...form, username: e.target.value })}
              placeholder="you@example.com"
            />
            <Input
              label="Password"
              type="password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              placeholder="••••••••"
            />
            <Button onClick={handleCredentials} loading={loading} className="w-full">
              Sign In
            </Button>
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-gray-300 dark:border-gray-600"></div>
              </div>
              <div className="relative flex justify-center text-sm">
                <span className="px-2 bg-white dark:bg-gray-800 text-gray-500">Or</span>
              </div>
            </div>
            <Button
              variant="secondary"
              className="w-full"
              onClick={() => navigate('/oidc-login')}
            >
              Sign in with SSO Provider
            </Button>
          </div>
        </Card>
      )}
      {step === 'mfa' && (
        <Card title="Multi-Factor Authentication" icon={<Lock className="h-6 w-6" />}>          
          <div className="space-y-4">
            {loginInit?.mfaMethods && loginInit.mfaMethods.length > 1 && (
              <div className="flex gap-2 flex-wrap">
                {loginInit.mfaMethods.map((m) => (
                  <button
                    key={m}
                    onClick={() => setMfaMethod(m)}
                    className={`px-3 py-1 rounded-md text-sm border transition ${
                      mfaMethod === m
                        ? 'bg-primary-500 text-white border-primary-600'
                        : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 border-gray-300 dark:border-gray-600'
                    }`}
                  >
                    {m.toUpperCase()}
                  </button>
                ))}
              </div>
            )}
            <Input
              label={`Enter ${mfaMethod.toUpperCase()} Code`}
              value={mfaCode}
              onChange={(e) => setMfaCode(e.target.value)}
              placeholder="123456"
            />
            <Button onClick={handleMFA} loading={loading} className="w-full">
              Verify MFA
            </Button>
            <Button
              variant="secondary"
              disabled={loading}
              onClick={() => setStep('credentials')}
              className="w-full"
            >
              Back
            </Button>
          </div>
        </Card>
      )}
      {step === 'success' && (
        <Card title="Login Successful" icon={<Shield className="h-6 w-6" />}>          
          <div className="space-y-2">
            <p className="text-sm text-gray-700 dark:text-gray-300">
              You are now authenticated. A session token has been established.
            </p>
            <Button variant="secondary" onClick={() => setStep('credentials')} className="w-full">
              Log out
            </Button>
          </div>
        </Card>
      )}
    </div>
  )
}