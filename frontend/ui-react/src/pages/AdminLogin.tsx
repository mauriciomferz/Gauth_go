
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  FluentProvider,
  webLightTheme,
  Card,
  Input,
  Button,
  Text,
  Title1,
  makeStyles,
  tokens,
  Spinner,
  MessageBar,
  MessageBarBody,
  MessageBarTitle,
} from '@fluentui/react-components';
import {
  ShieldCheckmark24Regular,
  LockClosed24Regular,
  Key24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    justifyContent: 'center',
    minHeight: '100vh',
    backgroundColor: tokens.colorNeutralBackground3,
    padding: '20px',
  },
  card: {
    width: '100%',
    maxWidth: '440px',
    padding: '40px',
  },
  header: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: '12px',
    marginBottom: '32px',
  },
  logo: {
    fontSize: '48px',
    color: tokens.colorBrandForeground1,
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '20px',
  },
  inputWrapper: {
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  label: {
    fontSize: '14px',
    fontWeight: 600,
    color: tokens.colorNeutralForeground1,
  },
  button: {
    marginTop: '8px',
  },
  footer: {
    marginTop: '24px',
    textAlign: 'center',
    fontSize: '12px',
    color: tokens.colorNeutralForeground3,
  },
  stepIndicator: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    marginBottom: '24px',
    fontSize: '13px',
    color: tokens.colorNeutralForeground2,
  },
});

type LoginStep = 'credentials' | 'mfa' | 'success';

interface LoginState {
  username: string;
  password: string;
  mfaCode: string;
  role: 'admin' | 'auditor' | null;
}

export default function AdminLogin() {
  const classes = useStyles();
  const navigate = useNavigate();
  const [step, setStep] = useState<LoginStep>('credentials');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [loginState, setLoginState] = useState<LoginState>({
    username: '',
    password: '',
    mfaCode: '',
    role: null,
  });

  const handleCredentialsSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    try {
      // Simulate API call for credentials verification
      const response = await fetch('/api/admin/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          email: loginState.username,
          password: loginState.password,
        }),
      });

      if (!response.ok) {
        throw new Error('Invalid credentials');
      }

      const data = await response.json();
      if (data.success && data.requiresMFA) {
        sessionStorage.setItem('mfa_challenge', data.sessionChallenge);
        setLoginState({ ...loginState, role: data.role });
        setStep('mfa');
      } else {
        throw new Error('Unexpected login response');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Authentication failed');
    } finally {
      setLoading(false);
    }
  };

  const handleMFASubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);
    setLoading(true);

    const challengeId = sessionStorage.getItem('mfa_challenge');
    if (!challengeId) {
      setError('Session expired. Please login again.');
      setStep('credentials');
      setLoading(false);
      return;
    }

    try {
      // Simulate API call for MFA verification
      const response = await fetch('/api/admin/auth/verify-mfa', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          challengeId: challengeId,
          code: loginState.mfaCode,
          method: 'totp',
        }),
      });

      if (!response.ok) {
        throw new Error('Invalid MFA code');
      }

      const data = await response.json();

      if (data.success && data.token) {
        // Store JWT token
        localStorage.setItem('admin_token', data.token);
        localStorage.setItem('admin_role', loginState.role || '');
        sessionStorage.removeItem('mfa_challenge');

        setStep('success');

        // Redirect to admin dashboard
        setTimeout(() => {
          navigate('/admin/dashboard');
        }, 1000);
      } else {
        throw new Error('Invalid MFA response');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'MFA verification failed');
    } finally {
      setLoading(false);
    }
  };

  const renderCredentialsForm = () => (
    <>
      <div className={classes.stepIndicator}>
        <LockClosed24Regular />
        <Text>Step 1 of 2: Enter your credentials</Text>
      </div>
      <form onSubmit={handleCredentialsSubmit} className={classes.form}>
        <div className={classes.inputWrapper}>
          <label className={classes.label}>Username or Email</label>
          <Input
            type="text"
            value={loginState.username}
            onChange={(e) =>
              setLoginState({ ...loginState, username: e.target.value })
            }
            placeholder="admin@example.com"
            required
            disabled={loading}
            size="large"
          />
        </div>
        <div className={classes.inputWrapper}>
          <label className={classes.label}>Password</label>
          <Input
            type="password"
            value={loginState.password}
            onChange={(e) =>
              setLoginState({ ...loginState, password: e.target.value })
            }
            placeholder="Enter your password"
            required
            disabled={loading}
            size="large"
          />
        </div>
        <Button
          type="submit"
          appearance="primary"
          size="large"
          disabled={loading || !loginState.username || !loginState.password}
          className={classes.button}
          icon={loading ? <Spinner size="tiny" /> : undefined}
        >
          {loading ? 'Verifying...' : 'Continue'}
        </Button>
      </form>
    </>
  );

  const renderMFAForm = () => (
    <>
      <div className={classes.stepIndicator}>
        <Key24Regular />
        <Text>Step 2 of 2: Multi-factor authentication</Text>
      </div>
      <MessageBar intent="info">
        <MessageBarBody>
          <MessageBarTitle>MFA Required</MessageBarTitle>
          Enter the 6-digit code from your authenticator app.
        </MessageBarBody>
      </MessageBar>
      <form onSubmit={handleMFASubmit} className={classes.form}>
        <div className={classes.inputWrapper}>
          <label className={classes.label}>Authentication Code</label>
          <Input
            type="text"
            value={loginState.mfaCode}
            onChange={(e) =>
              setLoginState({ ...loginState, mfaCode: e.target.value })
            }
            placeholder="000000"
            required
            disabled={loading}
            size="large"
            maxLength={6}
            pattern="[0-9]{6}"
          />
        </div>
        <Button
          type="submit"
          appearance="primary"
          size="large"
          disabled={loading || loginState.mfaCode.length !== 6}
          className={classes.button}
          icon={loading ? <Spinner size="tiny" /> : undefined}
        >
          {loading ? 'Verifying...' : 'Verify & Sign In'}
        </Button>
        <Button
          appearance="subtle"
          size="medium"
          onClick={() => setStep('credentials')}
          disabled={loading}
        >
          Back to credentials
        </Button>
      </form>
    </>
  );

  const renderSuccess = () => (
    <div className={classes.form}>
      <div style={{ textAlign: 'center' }}>
        <ShieldCheckmark24Regular
          style={{ fontSize: '64px', color: tokens.colorPaletteGreenForeground1 }}
        />
        <Title1>Authentication Successful</Title1>
        <Text>Redirecting to admin dashboard...</Text>
        <Spinner style={{ marginTop: '20px' }} />
      </div>
    </div>
  );

  return (
    <FluentProvider theme={webLightTheme}>
      <div className={classes.container}>
        <Card className={classes.card}>
          <div className={classes.header}>
            <Text size={600} weight="semibold">AuthAI Admin</Text>
            <Text>Sign in to continue</Text>
          </div>

          {error && (
            <MessageBar intent="error" style={{ marginBottom: '20px' }}>
              <MessageBarBody>
                <MessageBarTitle>Authentication Error</MessageBarTitle>
                {error}
              </MessageBarBody>
            </MessageBar>
          )}

          {step === 'credentials' && renderCredentialsForm()}
          {step === 'mfa' && renderMFAForm()}
          {step === 'success' && renderSuccess()}

          <div className={classes.footer}>
            <Text>
              AgentAuth Identity Platform v1.0 • Powered by RFC-0111 & RFC-0115
            </Text>
          </div>
        </Card>
      </div>
    </FluentProvider>
  );
}
