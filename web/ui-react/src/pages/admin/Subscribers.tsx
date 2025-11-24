import { useState } from 'react';
import {
  makeStyles,
  tokens,
  Card,
  Text,
  Title3,
  Button,
  Input,
  Textarea,
  Dropdown,
  Option,
  Checkbox,
  MessageBar,
  MessageBarBody,
  MessageBarTitle,
  ProgressBar,
  Field,
  Radio,
  RadioGroup,
} from '@fluentui/react-components';
import {
  PeopleTeam24Regular,
  ArrowLeft24Regular,
  ArrowRight24Regular,
  Checkmark24Regular,
  Building24Regular,
  Shield24Regular,
  Key24Regular,
  Settings24Regular,
  Globe24Regular,
  Mail24Regular,
  CheckmarkCircle24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '24px',
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    marginBottom: '8px',
  },
  card: {
    padding: '32px',
    maxWidth: '900px',
  },
  stepper: {
    display: 'flex',
    justifyContent: 'space-between',
    marginBottom: '32px',
    position: 'relative',
  },
  stepperLine: {
    position: 'absolute',
    top: '20px',
    left: '5%',
    right: '5%',
    height: '2px',
    backgroundColor: tokens.colorNeutralStroke1,
    zIndex: 0,
  },
  stepperProgress: {
    position: 'absolute',
    top: '20px',
    left: '5%',
    height: '2px',
    backgroundColor: tokens.colorBrandForeground1,
    transition: 'width 0.3s ease',
    zIndex: 1,
  },
  step: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: '8px',
    position: 'relative',
    zIndex: 2,
    flex: 1,
  },
  stepCircle: {
    width: '40px',
    height: '40px',
    borderRadius: '50%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: tokens.colorNeutralBackground1,
    border: `2px solid ${tokens.colorNeutralStroke1}`,
    fontSize: '14px',
    fontWeight: 600,
    transition: 'all 0.3s ease',
  },
  stepCircleActive: {
    backgroundColor: tokens.colorBrandBackground,
    borderTopColor: tokens.colorBrandForeground1,
    borderRightColor: tokens.colorBrandForeground1,
    borderBottomColor: tokens.colorBrandForeground1,
    borderLeftColor: tokens.colorBrandForeground1,
    color: tokens.colorNeutralForegroundOnBrand,
  },
  stepCircleCompleted: {
    backgroundColor: tokens.colorPaletteGreenBackground1,
    borderTopColor: tokens.colorPaletteGreenForeground1,
    borderRightColor: tokens.colorPaletteGreenForeground1,
    borderBottomColor: tokens.colorPaletteGreenForeground1,
    borderLeftColor: tokens.colorPaletteGreenForeground1,
    color: tokens.colorNeutralForegroundOnBrand,
  },
  stepLabel: {
    fontSize: '12px',
    textAlign: 'center',
    color: tokens.colorNeutralForeground3,
  },
  stepLabelActive: {
    color: tokens.colorNeutralForeground1,
    fontWeight: 600,
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '20px',
  },
  twoColumn: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '20px',
  },
  actions: {
    display: 'flex',
    justifyContent: 'space-between',
    marginTop: '32px',
    paddingTop: '24px',
    borderTop: `1px solid ${tokens.colorNeutralStroke2}`,
  },
  successContainer: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: '24px',
    padding: '40px',
    textAlign: 'center',
  },
  successIcon: {
    fontSize: '64px',
    color: tokens.colorPaletteGreenForeground1,
  },
});

interface SubscriberFormData {
  // Step 1: Basic Information
  tenantName: string;
  tenantId: string;
  contactEmail: string;
  contactPhone: string;
  
  // Step 2: OIDC Configuration
  oidcProvider: string;
  oidcClientId: string;
  oidcClientSecret: string;
  oidcDiscoveryUrl: string;
  
  // Step 3: Key Generation
  keyAlgorithm: string;
  keyRotationInterval: string;
  enableAutoRotation: boolean;
  
  // Step 4: Policy Templates
  policyTemplate: string;
  customPolicies: string;
  
  // Step 5: Legal Framework
  jurisdiction: string;
  complianceFrameworks: string[];
  dataRetention: string;
  
  // Step 6: Security Settings
  mfaRequired: boolean;
  tokenExpiration: string;
  sessionTimeout: string;
  ipWhitelist: string;
  
  // Step 7: Notification Preferences
  emailNotifications: boolean;
  webhookUrl: string;
  notificationEvents: string[];
  
  // Step 8: Review & Confirm
  agreedToTerms: boolean;
}

const steps = [
  { id: 1, label: 'Basic Info', icon: <Building24Regular /> },
  { id: 2, label: 'OIDC Config', icon: <Shield24Regular /> },
  { id: 3, label: 'Key Setup', icon: <Key24Regular /> },
  { id: 4, label: 'Policies', icon: <Settings24Regular /> },
  { id: 5, label: 'Legal', icon: <Globe24Regular /> },
  { id: 6, label: 'Security', icon: <Shield24Regular /> },
  { id: 7, label: 'Notifications', icon: <Mail24Regular /> },
  { id: 8, label: 'Review', icon: <CheckmarkCircle24Regular /> },
];

export default function Subscribers() {
  const classes = useStyles();
  const [currentStep, setCurrentStep] = useState(1);
  const [formData, setFormData] = useState<SubscriberFormData>({
    tenantName: '',
    tenantId: '',
    contactEmail: '',
    contactPhone: '',
    oidcProvider: 'custom',
    oidcClientId: '',
    oidcClientSecret: '',
    oidcDiscoveryUrl: '',
    keyAlgorithm: 'RS256',
    keyRotationInterval: '90',
    enableAutoRotation: true,
    policyTemplate: 'standard',
    customPolicies: '',
    jurisdiction: 'US',
    complianceFrameworks: [],
    dataRetention: '90',
    mfaRequired: true,
    tokenExpiration: '3600',
    sessionTimeout: '7200',
    ipWhitelist: '',
    emailNotifications: true,
    webhookUrl: '',
    notificationEvents: [],
    agreedToTerms: false,
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const updateFormData = (field: keyof SubscriberFormData, value: any) => {
    setFormData({ ...formData, [field]: value });
  };

  const handleNext = () => {
    if (currentStep < 8) {
      setCurrentStep(currentStep + 1);
      setError(null);
    }
  };

  const handleBack = () => {
    if (currentStep > 1) {
      setCurrentStep(currentStep - 1);
      setError(null);
    }
  };

  const handleSubmit = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await fetch('/api/admin/subscribers', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
        body: JSON.stringify(formData),
      });

      if (!response.ok) {
        throw new Error('Failed to create subscriber');
      }

      setSuccess(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create subscriber');
    } finally {
      setLoading(false);
    }
  };

  const renderStep = () => {
    switch (currentStep) {
      case 1:
        return (
          <div className={classes.form}>
            <Title3>Basic Information</Title3>
            <Text>Provide basic tenant information to get started.</Text>
            
            <Field label="Tenant Name" required>
              <Input
                value={formData.tenantName}
                onChange={(e) => updateFormData('tenantName', e.target.value)}
                placeholder="ACME Corporation"
              />
            </Field>
            
            <Field label="Tenant ID" required>
              <Input
                value={formData.tenantId}
                onChange={(e) => updateFormData('tenantId', e.target.value)}
                placeholder="acme-corp-001"
              />
            </Field>
            
            <div className={classes.twoColumn}>
              <Field label="Contact Email" required>
                <Input
                  type="email"
                  value={formData.contactEmail}
                  onChange={(e) => updateFormData('contactEmail', e.target.value)}
                  placeholder="admin@acme.com"
                />
              </Field>
              
              <Field label="Contact Phone">
                <Input
                  type="tel"
                  value={formData.contactPhone}
                  onChange={(e) => updateFormData('contactPhone', e.target.value)}
                  placeholder="+1-555-0123"
                />
              </Field>
            </div>
          </div>
        );

      case 2:
        return (
          <div className={classes.form}>
            <Title3>OIDC Configuration</Title3>
            <Text>Configure OpenID Connect provider for authentication.</Text>
            
            <Field label="OIDC Provider">
              <Dropdown
                value={formData.oidcProvider}
                onOptionSelect={(_, data) => updateFormData('oidcProvider', data.optionValue)}
              >
                <Option value="azure">Azure AD</Option>
                <Option value="google">Google Workspace</Option>
                <Option value="okta">Okta</Option>
                <Option value="auth0">Auth0</Option>
                <Option value="custom">Custom Provider</Option>
              </Dropdown>
            </Field>
            
            <Field label="Client ID" required>
              <Input
                value={formData.oidcClientId}
                onChange={(e) => updateFormData('oidcClientId', e.target.value)}
                placeholder="client-id-from-provider"
              />
            </Field>
            
            <Field label="Client Secret" required>
              <Input
                type="password"
                value={formData.oidcClientSecret}
                onChange={(e) => updateFormData('oidcClientSecret', e.target.value)}
                placeholder="client-secret-from-provider"
              />
            </Field>
            
            <Field label="Discovery URL" required>
              <Input
                value={formData.oidcDiscoveryUrl}
                onChange={(e) => updateFormData('oidcDiscoveryUrl', e.target.value)}
                placeholder="https://provider.com/.well-known/openid-configuration"
              />
            </Field>
          </div>
        );

      case 3:
        return (
          <div className={classes.form}>
            <Title3>Key Generation Settings</Title3>
            <Text>Configure cryptographic key generation and rotation policies.</Text>
            
            <Field label="Key Algorithm">
              <RadioGroup
                value={formData.keyAlgorithm}
                onChange={(_, data) => updateFormData('keyAlgorithm', data.value)}
              >
                <Radio value="RS256" label="RS256 (RSA 2048-bit)" />
                <Radio value="RS512" label="RS512 (RSA 4096-bit)" />
                <Radio value="ES256" label="ES256 (ECDSA P-256)" />
                <Radio value="ES512" label="ES512 (ECDSA P-521)" />
              </RadioGroup>
            </Field>
            
            <Field label="Key Rotation Interval (days)">
              <Input
                type="number"
                value={formData.keyRotationInterval}
                onChange={(e) => updateFormData('keyRotationInterval', e.target.value)}
                placeholder="90"
              />
            </Field>
            
            <Checkbox
              checked={formData.enableAutoRotation}
              onChange={(_, data) => updateFormData('enableAutoRotation', data.checked)}
              label="Enable automatic key rotation"
            />
          </div>
        );

      case 4:
        return (
          <div className={classes.form}>
            <Title3>Policy Templates</Title3>
            <Text>Select authorization policy templates and customize as needed.</Text>
            
            <Field label="Policy Template">
              <RadioGroup
                value={formData.policyTemplate}
                onChange={(_, data) => updateFormData('policyTemplate', data.value)}
              >
                <Radio value="standard" label="Standard (Balanced security and usability)" />
                <Radio value="strict" label="Strict (Maximum security)" />
                <Radio value="relaxed" label="Relaxed (Development/testing)" />
                <Radio value="custom" label="Custom (Define your own)" />
              </RadioGroup>
            </Field>
            
            {formData.policyTemplate === 'custom' && (
              <Field label="Custom Policies (YAML)">
                <Textarea
                  value={formData.customPolicies}
                  onChange={(e) => updateFormData('customPolicies', e.target.value)}
                  rows={10}
                  placeholder="policies:&#10;  - name: admin&#10;    actions: [read, write, delete]"
                />
              </Field>
            )}
          </div>
        );

      case 5:
        return (
          <div className={classes.form}>
            <Title3>Legal Framework</Title3>
            <Text>Configure jurisdiction and compliance requirements.</Text>
            
            <Field label="Jurisdiction">
              <Dropdown
                value={formData.jurisdiction}
                onOptionSelect={(_, data) => updateFormData('jurisdiction', data.optionValue)}
              >
                <Option value="US">United States</Option>
                <Option value="EU">European Union</Option>
                <Option value="UK">United Kingdom</Option>
                <Option value="CA">Canada</Option>
                <Option value="AU">Australia</Option>
                <Option value="APAC">Asia-Pacific</Option>
              </Dropdown>
            </Field>
            
            <Field label="Compliance Frameworks">
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <Checkbox
                  checked={formData.complianceFrameworks.includes('GDPR')}
                  onChange={(_, data) => {
                    const frameworks = data.checked
                      ? [...formData.complianceFrameworks, 'GDPR']
                      : formData.complianceFrameworks.filter((f) => f !== 'GDPR');
                    updateFormData('complianceFrameworks', frameworks);
                  }}
                  label="GDPR (General Data Protection Regulation)"
                />
                <Checkbox
                  checked={formData.complianceFrameworks.includes('SOX')}
                  onChange={(_, data) => {
                    const frameworks = data.checked
                      ? [...formData.complianceFrameworks, 'SOX']
                      : formData.complianceFrameworks.filter((f) => f !== 'SOX');
                    updateFormData('complianceFrameworks', frameworks);
                  }}
                  label="SOX (Sarbanes-Oxley Act)"
                />
                <Checkbox
                  checked={formData.complianceFrameworks.includes('HIPAA')}
                  onChange={(_, data) => {
                    const frameworks = data.checked
                      ? [...formData.complianceFrameworks, 'HIPAA']
                      : formData.complianceFrameworks.filter((f) => f !== 'HIPAA');
                    updateFormData('complianceFrameworks', frameworks);
                  }}
                  label="HIPAA (Health Insurance Portability and Accountability Act)"
                />
              </div>
            </Field>
            
            <Field label="Data Retention Period (days)">
              <Input
                type="number"
                value={formData.dataRetention}
                onChange={(e) => updateFormData('dataRetention', e.target.value)}
                placeholder="90"
              />
            </Field>
          </div>
        );

      case 6:
        return (
          <div className={classes.form}>
            <Title3>Security Settings</Title3>
            <Text>Configure security policies and restrictions.</Text>
            
            <Checkbox
              checked={formData.mfaRequired}
              onChange={(_, data) => updateFormData('mfaRequired', data.checked)}
              label="Require Multi-Factor Authentication (MFA)"
            />
            
            <div className={classes.twoColumn}>
              <Field label="Token Expiration (seconds)">
                <Input
                  type="number"
                  value={formData.tokenExpiration}
                  onChange={(e) => updateFormData('tokenExpiration', e.target.value)}
                  placeholder="3600"
                />
              </Field>
              
              <Field label="Session Timeout (seconds)">
                <Input
                  type="number"
                  value={formData.sessionTimeout}
                  onChange={(e) => updateFormData('sessionTimeout', e.target.value)}
                  placeholder="7200"
                />
              </Field>
            </div>
            
            <Field label="IP Whitelist (comma-separated)">
              <Textarea
                value={formData.ipWhitelist}
                onChange={(e) => updateFormData('ipWhitelist', e.target.value)}
                placeholder="192.168.1.0/24, 10.0.0.0/8"
                rows={3}
              />
            </Field>
          </div>
        );

      case 7:
        return (
          <div className={classes.form}>
            <Title3>Notification Preferences</Title3>
            <Text>Configure how you receive system notifications.</Text>
            
            <Checkbox
              checked={formData.emailNotifications}
              onChange={(_, data) => updateFormData('emailNotifications', data.checked)}
              label="Enable email notifications"
            />
            
            <Field label="Webhook URL (optional)">
              <Input
                value={formData.webhookUrl}
                onChange={(e) => updateFormData('webhookUrl', e.target.value)}
                placeholder="https://your-domain.com/webhook"
              />
            </Field>
            
            <Field label="Notification Events">
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <Checkbox
                  checked={formData.notificationEvents.includes('token_created')}
                  onChange={(_, data) => {
                    const events = data.checked
                      ? [...formData.notificationEvents, 'token_created']
                      : formData.notificationEvents.filter((e) => e !== 'token_created');
                    updateFormData('notificationEvents', events);
                  }}
                  label="Token Created"
                />
                <Checkbox
                  checked={formData.notificationEvents.includes('token_revoked')}
                  onChange={(_, data) => {
                    const events = data.checked
                      ? [...formData.notificationEvents, 'token_revoked']
                      : formData.notificationEvents.filter((e) => e !== 'token_revoked');
                    updateFormData('notificationEvents', events);
                  }}
                  label="Token Revoked"
                />
                <Checkbox
                  checked={formData.notificationEvents.includes('policy_updated')}
                  onChange={(_, data) => {
                    const events = data.checked
                      ? [...formData.notificationEvents, 'policy_updated']
                      : formData.notificationEvents.filter((e) => e !== 'policy_updated');
                    updateFormData('notificationEvents', events);
                  }}
                  label="Policy Updated"
                />
                <Checkbox
                  checked={formData.notificationEvents.includes('security_alert')}
                  onChange={(_, data) => {
                    const events = data.checked
                      ? [...formData.notificationEvents, 'security_alert']
                      : formData.notificationEvents.filter((e) => e !== 'security_alert');
                    updateFormData('notificationEvents', events);
                  }}
                  label="Security Alerts"
                />
              </div>
            </Field>
          </div>
        );

      case 8:
        return (
          <div className={classes.form}>
            <Title3>Review & Confirm</Title3>
            <Text>Please review your configuration before submitting.</Text>
            
            <MessageBar intent="info">
              <MessageBarBody>
                <MessageBarTitle>Configuration Summary</MessageBarTitle>
                Review all settings carefully. You can modify these after creation in the subscriber settings page.
              </MessageBarBody>
            </MessageBar>
            
            <Card style={{ padding: '16px', backgroundColor: tokens.colorNeutralBackground3 }}>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                <div>
                  <Text weight="semibold">Tenant:</Text>
                  <Text> {formData.tenantName} ({formData.tenantId})</Text>
                </div>
                <div>
                  <Text weight="semibold">Contact:</Text>
                  <Text> {formData.contactEmail}</Text>
                </div>
                <div>
                  <Text weight="semibold">OIDC Provider:</Text>
                  <Text> {formData.oidcProvider}</Text>
                </div>
                <div>
                  <Text weight="semibold">Key Algorithm:</Text>
                  <Text> {formData.keyAlgorithm}</Text>
                </div>
                <div>
                  <Text weight="semibold">Policy Template:</Text>
                  <Text> {formData.policyTemplate}</Text>
                </div>
                <div>
                  <Text weight="semibold">Jurisdiction:</Text>
                  <Text> {formData.jurisdiction}</Text>
                </div>
                <div>
                  <Text weight="semibold">MFA Required:</Text>
                  <Text> {formData.mfaRequired ? 'Yes' : 'No'}</Text>
                </div>
              </div>
            </Card>
            
            <Checkbox
              checked={formData.agreedToTerms}
              onChange={(_, data) => updateFormData('agreedToTerms', data.checked)}
              label="I agree to the terms and conditions and confirm the accuracy of the information provided"
            />
          </div>
        );

      default:
        return null;
    }
  };

  if (success) {
    return (
      <div className={classes.container}>
        <Card className={classes.card}>
          <div className={classes.successContainer}>
            <CheckmarkCircle24Regular className={classes.successIcon} />
            <Title3>Subscriber Created Successfully!</Title3>
            <Text>
              Tenant <strong>{formData.tenantName}</strong> has been registered and configured.
            </Text>
            <Text size={300} style={{ color: tokens.colorNeutralForeground3 }}>
              An email has been sent to {formData.contactEmail} with setup instructions and credentials.
            </Text>
            <div style={{ display: 'flex', gap: '12px', marginTop: '16px' }}>
              <Button appearance="primary" onClick={() => window.location.reload()}>
                Create Another Subscriber
              </Button>
              <Button onClick={() => window.location.href = '/admin/subscribers'}>
                View All Subscribers
              </Button>
            </div>
          </div>
        </Card>
      </div>
    );
  }

  const progressWidth = ((currentStep - 1) / (steps.length - 1)) * 90;

  return (
    <div className={classes.container}>
      <div className={classes.header}>
        <PeopleTeam24Regular style={{ fontSize: '24px' }} />
        <Title3>Subscriber Onboarding</Title3>
      </div>

      <Card className={classes.card}>
        {/* Stepper */}
        <div className={classes.stepper}>
          <div className={classes.stepperLine} />
          <div className={classes.stepperProgress} style={{ width: `${progressWidth}%` }} />
          {steps.map((step) => (
            <div key={step.id} className={classes.step}>
              <div
                className={`${classes.stepCircle} ${
                  step.id === currentStep
                    ? classes.stepCircleActive
                    : step.id < currentStep
                    ? classes.stepCircleCompleted
                    : ''
                }`}
              >
                {step.id < currentStep ? <Checkmark24Regular /> : step.id}
              </div>
              <Text
                className={`${classes.stepLabel} ${
                  step.id === currentStep ? classes.stepLabelActive : ''
                }`}
              >
                {step.label}
              </Text>
            </div>
          ))}
        </div>

        {/* Progress Bar */}
        <ProgressBar value={currentStep / 8} max={1} style={{ marginBottom: '24px' }} />

        {/* Error Message */}
        {error && (
          <MessageBar intent="error" style={{ marginBottom: '20px' }}>
            <MessageBarBody>
              <MessageBarTitle>Error</MessageBarTitle>
              {error}
            </MessageBarBody>
          </MessageBar>
        )}

        {/* Step Content */}
        {renderStep()}

        {/* Actions */}
        <div className={classes.actions}>
          <Button
            appearance="secondary"
            icon={<ArrowLeft24Regular />}
            onClick={handleBack}
            disabled={currentStep === 1 || loading}
          >
            Back
          </Button>
          
          <div style={{ display: 'flex', gap: '12px' }}>
            {currentStep < 8 ? (
              <Button
                appearance="primary"
                icon={<ArrowRight24Regular />}
                iconPosition="after"
                onClick={handleNext}
                disabled={loading}
              >
                Next
              </Button>
            ) : (
              <Button
                appearance="primary"
                icon={<Checkmark24Regular />}
                onClick={handleSubmit}
                disabled={!formData.agreedToTerms || loading}
              >
                {loading ? 'Creating...' : 'Create Subscriber'}
              </Button>
            )}
          </div>
        </div>
      </Card>
    </div>
  );
}
