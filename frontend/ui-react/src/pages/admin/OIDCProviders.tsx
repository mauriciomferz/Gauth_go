import React, { useState, useEffect } from 'react';
import {
  makeStyles,
  shorthands,
  tokens,
  Button,
  Dialog,
  DialogTrigger,
  DialogSurface,
  DialogTitle,
  DialogBody,
  DialogActions,
  DialogContent,
  Input,
  Label,
  Field,
  Dropdown,
  Option,
  Textarea,
  Switch,
  Text,
  Title3,
  DataGrid,
  DataGridHeader,
  DataGridRow,
  DataGridHeaderCell,
  DataGridBody,
  DataGridCell,
  createTableColumn,
  TableCellLayout,
  TableColumnDefinition,
  Badge,
  Menu,
  MenuTrigger,
  MenuPopover,
  MenuList,
  MenuItem,
  MessageBar,
  MessageBarBody,
  Spinner,
} from '@fluentui/react-components';
import {
  Add24Regular,
  Delete24Regular,
  Edit24Regular,
  MoreHorizontal24Regular,
  Checkmark24Regular,
  Dismiss24Regular,
  CloudCheckmark24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('20px'),
    ...shorthands.padding('24px'),
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  grid: {
    width: '100%',
  },
  dialog: {
    maxWidth: '800px',
    width: '90vw',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('16px'),
  },
  formRow: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    ...shorthands.gap('16px'),
  },
  badge: {
    textTransform: 'capitalize',
  },
  loading: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    minHeight: '200px',
  },
  actions: {
    display: 'flex',
    ...shorthands.gap('8px'),
  },
});

interface OIDCProvider {
  id: string;
  tenantId: string;
  providerName: string;
  providerType: string;
  displayName: string;
  issuerUrl: string;
  clientId: string;
  scopes: string[];
  redirectUris: string[];
  status: 'active' | 'inactive' | 'testing' | 'error';
  isDefault: boolean;
  priority: number;
  createdAt: string;
  updatedAt: string;
}

interface OIDCProviderFormData {
  providerName: string;
  providerType: string;
  displayName: string;
  issuerUrl: string;
  clientId: string;
  clientSecret: string;
  scopes: string;
  redirectUris: string;
  postLogoutRedirectUris: string;
  azureTenantId?: string;
  azureResource?: string;
  validateIssuer: boolean;
  validateAudience: boolean;
  validateSignature: boolean;
  autoProvisionUsers: boolean;
  pkceEnabled: boolean;
  isDefault: boolean;
  priority: number;
}

const defaultFormData: OIDCProviderFormData = {
  providerName: '',
  providerType: 'azure_ad',
  displayName: '',
  issuerUrl: '',
  clientId: '',
  clientSecret: '',
  scopes: 'openid profile email',
  redirectUris: 'http://localhost:8080/auth/callback',
  postLogoutRedirectUris: '',
  validateIssuer: true,
  validateAudience: true,
  validateSignature: true,
  autoProvisionUsers: true,
  pkceEnabled: true,
  isDefault: false,
  priority: 0,
};

export default function OIDCProviders() {
  const classes = useStyles();
  const [providers, setProviders] = useState<OIDCProvider[]>([]);
  const [loading, setLoading] = useState(true);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingProvider, setEditingProvider] = useState<OIDCProvider | null>(null);
  const [formData, setFormData] = useState<OIDCProviderFormData>(defaultFormData);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);
  const [testingProvider, setTestingProvider] = useState<string | null>(null);

  const tenantId = 'test-tenant-1';

  useEffect(() => {
    loadProviders();
  }, []);

  const loadProviders = async () => {
    try {
      setLoading(true);
      const response = await fetch(`http://localhost:8080/api/admin/oidc-providers?tenant_id=${tenantId}`);
      
      if (!response.ok) {
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
          const errorData = await response.json();
          throw new Error(errorData.error || `Server error: ${response.status}`);
        } else if (response.status === 404) {
          throw new Error('OIDC Providers endpoint requires database connection. Start the backend with DB_HOST configured to access this feature.');
        } else {
          throw new Error(`Server error: ${response.status}`);
        }
      }
      
      const data = await response.json();
      setProviders(data.providers || []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load OIDC providers');
      console.error('Error loading providers:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditingProvider(null);
    setFormData(defaultFormData);
    setDialogOpen(true);
  };

  const handleEdit = (provider: OIDCProvider) => {
    setEditingProvider(provider);
    setFormData({
      providerName: provider.providerName,
      providerType: provider.providerType,
      displayName: provider.displayName,
      issuerUrl: provider.issuerUrl,
      clientId: provider.clientId,
      clientSecret: '',
      scopes: provider.scopes.join(' '),
      redirectUris: provider.redirectUris.join('\n'),
      postLogoutRedirectUris: '',
      validateIssuer: true,
      validateAudience: true,
      validateSignature: true,
      autoProvisionUsers: true,
      pkceEnabled: true,
      isDefault: provider.isDefault,
      priority: provider.priority,
    });
    setDialogOpen(true);
  };

  const handleDelete = async (providerId: string) => {
    if (!confirm('Are you sure you want to delete this OIDC provider?')) return;

    try {
      const response = await fetch(
        `http://localhost:8080/api/admin/oidc-providers/${providerId}?tenant_id=${tenantId}`,
        { method: 'DELETE' }
      );

      if (response.ok) {
        setSuccess('Provider deleted successfully');
        loadProviders();
      } else {
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
          const data = await response.json();
          setError(data.error || 'Failed to delete provider');
        } else {
          setError(`Failed to delete provider: ${response.status}`);
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete provider');
      console.error('Error deleting provider:', err);
    }
  };

  const handleTestConnectivity = async (providerId: string) => {
    try {
      setTestingProvider(providerId);
      const response = await fetch(
        `http://localhost:8080/api/admin/oidc-providers/${providerId}/test?tenant_id=${tenantId}`,
        { method: 'POST' }
      );

      if (!response.ok) {
        throw new Error(`Server error: ${response.status}`);
      }

      const contentType = response.headers.get('content-type');
      if (!contentType || !contentType.includes('application/json')) {
        throw new Error('Invalid response format');
      }

      const data = await response.json();
      if (data.success) {
        setSuccess('Provider configuration is valid');
      } else {
        setError('Provider validation failed');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to test provider connectivity');
      console.error('Error testing provider:', err);
    } finally {
      setTestingProvider(null);
    }
  };

  const handleSubmit = async () => {
    try {
      const payload = {
        providerName: formData.providerName,
        providerType: formData.providerType,
        displayName: formData.displayName,
        issuerUrl: formData.issuerUrl,
        clientId: formData.clientId,
        clientSecret: formData.clientSecret,
        scopes: formData.scopes.split(/\s+/).filter(s => s),
        redirectUris: formData.redirectUris.split('\n').filter(s => s.trim()),
        postLogoutRedirectUris: formData.postLogoutRedirectUris.split('\n').filter(s => s.trim()),
        azureTenantId: formData.azureTenantId || undefined,
        azureResource: formData.azureResource || undefined,
        validateIssuer: formData.validateIssuer,
        validateAudience: formData.validateAudience,
        validateSignature: formData.validateSignature,
        autoProvisionUsers: formData.autoProvisionUsers,
        pkceEnabled: formData.pkceEnabled,
        isDefault: formData.isDefault,
        priority: formData.priority,
      };

      const url = editingProvider
        ? `http://localhost:8080/api/admin/oidc-providers/${editingProvider.id}?tenant_id=${tenantId}`
        : `http://localhost:8080/api/admin/oidc-providers?tenant_id=${tenantId}`;

      const method = editingProvider ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (response.ok) {
        setSuccess(editingProvider ? 'Provider updated successfully' : 'Provider created successfully');
        setDialogOpen(false);
        loadProviders();
      } else {
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json')) {
          const data = await response.json();
          setError(data.error || 'Failed to save provider');
        } else {
          setError(`Failed to save provider: ${response.status}`);
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save provider');
      console.error('Error saving provider:', err);
    }
  };

  const getStatusBadge = (status: string) => {
    const colors: Record<string, 'success' | 'danger' | 'warning' | 'informative'> = {
      active: 'success',
      inactive: 'danger',
      testing: 'warning',
      error: 'danger',
    };
    return <Badge className={classes.badge} appearance="filled" color={colors[status]}>{status}</Badge>;
  };

  const columns: TableColumnDefinition<OIDCProvider>[] = [
    createTableColumn<OIDCProvider>({
      columnId: 'displayName',
      compare: (a, b) => a.displayName.localeCompare(b.displayName),
      renderHeaderCell: () => 'Provider Name',
      renderCell: (item) => (
        <TableCellLayout>
          <div>
            <div style={{ fontWeight: 600 }}>{item.displayName}</div>
            <div style={{ fontSize: '12px', color: tokens.colorNeutralForeground3 }}>
              {item.providerType.replace('_', ' ').toUpperCase()}
            </div>
          </div>
        </TableCellLayout>
      ),
    }),
    createTableColumn<OIDCProvider>({
      columnId: 'issuerUrl',
      compare: (a, b) => a.issuerUrl.localeCompare(b.issuerUrl),
      renderHeaderCell: () => 'Issuer URL',
      renderCell: (item) => (
        <TableCellLayout truncate title={item.issuerUrl}>
          {item.issuerUrl}
        </TableCellLayout>
      ),
    }),
    createTableColumn<OIDCProvider>({
      columnId: 'status',
      compare: (a, b) => a.status.localeCompare(b.status),
      renderHeaderCell: () => 'Status',
      renderCell: (item) => <TableCellLayout>{getStatusBadge(item.status)}</TableCellLayout>,
    }),
    createTableColumn<OIDCProvider>({
      columnId: 'isDefault',
      compare: (a, b) => Number(b.isDefault) - Number(a.isDefault),
      renderHeaderCell: () => 'Default',
      renderCell: (item) => (
        <TableCellLayout>
          {item.isDefault ? <Checkmark24Regular color={tokens.colorPaletteGreenForeground1} /> : '-'}
        </TableCellLayout>
      ),
    }),
    createTableColumn<OIDCProvider>({
      columnId: 'priority',
      compare: (a, b) => a.priority - b.priority,
      renderHeaderCell: () => 'Priority',
      renderCell: (item) => <TableCellLayout>{item.priority}</TableCellLayout>,
    }),
    createTableColumn<OIDCProvider>({
      columnId: 'actions',
      renderHeaderCell: () => 'Actions',
      renderCell: (item) => (
        <TableCellLayout>
          <div className={classes.actions}>
            <Button
              size="small"
              appearance="subtle"
              icon={<CloudCheckmark24Regular />}
              onClick={() => handleTestConnectivity(item.id)}
              disabled={testingProvider === item.id}
            >
              {testingProvider === item.id ? 'Testing...' : 'Test'}
            </Button>
            <Menu>
              <MenuTrigger>
                <Button size="small" appearance="subtle" icon={<MoreHorizontal24Regular />} />
              </MenuTrigger>
              <MenuPopover>
                <MenuList>
                  <MenuItem icon={<Edit24Regular />} onClick={() => handleEdit(item)}>
                    Edit
                  </MenuItem>
                  <MenuItem icon={<Delete24Regular />} onClick={() => handleDelete(item.id)}>
                    Delete
                  </MenuItem>
                </MenuList>
              </MenuPopover>
            </Menu>
          </div>
        </TableCellLayout>
      ),
    }),
  ];

  const getProviderTypeHelp = (type: string) => {
    const help: Record<string, string> = {
      azure_ad: 'For Azure Active Directory / Microsoft Entra ID authentication',
      google: 'For Google Workspace / Google Identity authentication',
      okta: 'For Okta identity provider',
      auth0: 'For Auth0 authentication platform',
      custom: 'For any custom OIDC-compliant identity provider',
    };
    return help[type] || '';
  };

  const getProviderTypeDefaults = (type: string) => {
    const defaults: Record<string, Partial<OIDCProviderFormData>> = {
      azure_ad: {
        issuerUrl: 'https://login.microsoftonline.com/common/v2.0',
        scopes: 'openid profile email',
      },
      google: {
        issuerUrl: 'https://accounts.google.com',
        scopes: 'openid profile email',
      },
      okta: {
        issuerUrl: 'https://your-domain.okta.com',
        scopes: 'openid profile email',
      },
      auth0: {
        issuerUrl: 'https://your-domain.auth0.com',
        scopes: 'openid profile email',
      },
      custom: {
        issuerUrl: '',
        scopes: 'openid profile email',
      },
    };
    return defaults[type] || {};
  };

  const handleProviderTypeChange = (type: string) => {
    const defaults = getProviderTypeDefaults(type);
    setFormData({ ...formData, providerType: type, ...defaults });
  };

  return (
    <div className={classes.container}>
      <div className={classes.header}>
        <div>
          <Title3>OIDC Providers</Title3>
          <Text>Configure OpenID Connect identity providers for authentication</Text>
        </div>
        <Button appearance="primary" icon={<Add24Regular />} onClick={handleCreate}>
          Add Provider
        </Button>
      </div>

      {error && (
        <MessageBar intent="error" onDismiss={() => setError(null)}>
          <MessageBarBody>{error}</MessageBarBody>
        </MessageBar>
      )}

      {success && (
        <MessageBar intent="success" onDismiss={() => setSuccess(null)}>
          <MessageBarBody>{success}</MessageBarBody>
        </MessageBar>
      )}

      {loading ? (
        <div className={classes.loading}>
          <Spinner label="Loading providers..." />
        </div>
      ) : providers.length === 0 ? (
        <MessageBar intent="info">
          <MessageBarBody>
            No OIDC providers configured. Click "Add Provider" to create your first provider.
          </MessageBarBody>
        </MessageBar>
      ) : (
        <DataGrid
          items={providers}
          columns={columns}
          sortable
          className={classes.grid}
          size="small"
        >
          <DataGridHeader>
            <DataGridRow>
              {({ renderHeaderCell }) => (
                <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>
              )}
            </DataGridRow>
          </DataGridHeader>
          <DataGridBody<OIDCProvider>>
            {({ item, rowId }) => (
              <DataGridRow<OIDCProvider> key={rowId}>
                {({ renderCell }) => <DataGridCell>{renderCell(item)}</DataGridCell>}
              </DataGridRow>
            )}
          </DataGridBody>
        </DataGrid>
      )}

      <Dialog open={dialogOpen} onOpenChange={(_, data) => setDialogOpen(data.open)}>
        <DialogSurface className={classes.dialog}>
          <DialogBody>
            <DialogTitle>{editingProvider ? 'Edit OIDC Provider' : 'Create OIDC Provider'}</DialogTitle>
            <DialogContent>
              <div className={classes.form}>
                <Field label="Provider Type" required>
                  <Dropdown
                    value={formData.providerType}
                    selectedOptions={[formData.providerType]}
                    onOptionSelect={(_, data) => handleProviderTypeChange(data.optionValue as string)}
                    disabled={!!editingProvider}
                  >
                    <Option value="azure_ad">Azure AD / Microsoft Entra ID</Option>
                    <Option value="google">Google Workspace</Option>
                    <Option value="okta">Okta</Option>
                    <Option value="auth0">Auth0</Option>
                    <Option value="custom">Custom OIDC Provider</Option>
                  </Dropdown>
                  <Text size={200}>{getProviderTypeHelp(formData.providerType)}</Text>
                </Field>

                <div className={classes.formRow}>
                  <Field label="Provider Name" required>
                    <Input
                      value={formData.providerName}
                      onChange={(e) => setFormData({ ...formData, providerName: e.target.value })}
                      placeholder="e.g., azure-ad-prod"
                      disabled={!!editingProvider}
                    />
                  </Field>

                  <Field label="Display Name" required>
                    <Input
                      value={formData.displayName}
                      onChange={(e) => setFormData({ ...formData, displayName: e.target.value })}
                      placeholder="e.g., Azure Active Directory"
                    />
                  </Field>
                </div>

                <Field label="Issuer URL" required>
                  <Input
                    value={formData.issuerUrl}
                    onChange={(e) => setFormData({ ...formData, issuerUrl: e.target.value })}
                    placeholder="https://login.microsoftonline.com/common/v2.0"
                  />
                </Field>

                {formData.providerType === 'azure_ad' && (
                  <Field label="Azure Tenant ID">
                    <Input
                      value={formData.azureTenantId || ''}
                      onChange={(e) => setFormData({ ...formData, azureTenantId: e.target.value })}
                      placeholder="common, organizations, consumers, or your tenant ID"
                    />
                  </Field>
                )}

                <div className={classes.formRow}>
                  <Field label="Client ID" required>
                    <Input
                      value={formData.clientId}
                      onChange={(e) => setFormData({ ...formData, clientId: e.target.value })}
                      placeholder="Your application client ID"
                    />
                  </Field>

                  <Field label="Client Secret" required={!editingProvider}>
                    <Input
                      type="password"
                      value={formData.clientSecret}
                      onChange={(e) => setFormData({ ...formData, clientSecret: e.target.value })}
                      placeholder={editingProvider ? 'Leave blank to keep existing' : 'Your client secret'}
                    />
                  </Field>
                </div>

                <Field label="Scopes">
                  <Input
                    value={formData.scopes}
                    onChange={(e) => setFormData({ ...formData, scopes: e.target.value })}
                    placeholder="openid profile email"
                  />
                </Field>

                <Field label="Redirect URIs" required>
                  <Textarea
                    value={formData.redirectUris}
                    onChange={(e) => setFormData({ ...formData, redirectUris: e.target.value })}
                    placeholder="http://localhost:8080/auth/callback&#10;https://app.example.com/auth/callback"
                    rows={3}
                  />
                  <Text size={200}>One URL per line</Text>
                </Field>

                <div className={classes.formRow}>
                  <Field label="Priority">
                    <Input
                      type="number"
                      value={String(formData.priority)}
                      onChange={(e) => setFormData({ ...formData, priority: parseInt(e.target.value) || 0 })}
                    />
                  </Field>

                  <Field label="Set as Default">
                    <Switch
                      checked={formData.isDefault}
                      onChange={(_, data) => setFormData({ ...formData, isDefault: data.checked })}
                    />
                  </Field>
                </div>

                <Field label="Security Options">
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                    <Switch
                      checked={formData.pkceEnabled}
                      onChange={(_, data) => setFormData({ ...formData, pkceEnabled: data.checked })}
                      label="Enable PKCE (Recommended)"
                    />
                    <Switch
                      checked={formData.autoProvisionUsers}
                      onChange={(_, data) => setFormData({ ...formData, autoProvisionUsers: data.checked })}
                      label="Auto-provision users"
                    />
                    <Switch
                      checked={formData.validateIssuer}
                      onChange={(_, data) => setFormData({ ...formData, validateIssuer: data.checked })}
                      label="Validate issuer"
                    />
                    <Switch
                      checked={formData.validateSignature}
                      onChange={(_, data) => setFormData({ ...formData, validateSignature: data.checked })}
                      label="Validate signature"
                    />
                  </div>
                </Field>
              </div>
            </DialogContent>
            <DialogActions>
              <Button appearance="secondary" onClick={() => setDialogOpen(false)}>
                Cancel
              </Button>
              <Button appearance="primary" onClick={handleSubmit}>
                {editingProvider ? 'Update' : 'Create'}
              </Button>
            </DialogActions>
          </DialogBody>
        </DialogSurface>
      </Dialog>
    </div>
  );
}
