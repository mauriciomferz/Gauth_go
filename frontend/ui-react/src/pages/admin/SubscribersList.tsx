import { useState, useEffect } from 'react';
import {
  makeStyles,
  tokens,
  Card,
  Text,
  Title3,
  Button,
  Input,
  Table,
  TableBody,
  TableCell,
  TableRow,
  TableHeader,
  TableHeaderCell,
  Badge,
  Spinner,
  MessageBar,
  MessageBarBody,
  MessageBarTitle,
  Dropdown,
  Option,
  Dialog,
  DialogSurface,
  DialogTitle,
  DialogBody,
  DialogActions,
  DialogContent,
  TabList,
  Tab,
} from '@fluentui/react-components';
import {
  PeopleTeam24Regular,
  Search24Regular,
  Filter24Regular,
  ArrowSync24Regular,
  Add24Regular,
  Eye24Regular,
  Settings24Regular,
  Delete24Regular,
} from '@fluentui/react-icons';
import { useNavigate } from 'react-router-dom';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '24px',
  },
  header: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: '8px',
  },
  headerLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  headerRight: {
    display: 'flex',
    gap: '12px',
  },
  filters: {
    display: 'flex',
    gap: '12px',
    alignItems: 'center',
    padding: '16px',
    backgroundColor: tokens.colorNeutralBackground2,
    borderRadius: tokens.borderRadiusMedium,
  },
  searchInput: {
    flex: 1,
    minWidth: '300px',
  },
  tableCard: {
    padding: '0',
    overflow: 'hidden',
  },
  table: {
    width: '100%',
  },
  statusBadge: {
    textTransform: 'capitalize',
  },
  emptyState: {
    padding: '64px',
    textAlign: 'center',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    gap: '16px',
  },
  stats: {
    display: 'flex',
    gap: '12px',
    padding: '12px 16px',
    backgroundColor: tokens.colorNeutralBackground2,
    borderRadius: tokens.borderRadiusMedium,
    alignItems: 'center',
  },
});

interface Subscriber {
  id: string;
  tenantName: string;
  tenantId: string;
  contactEmail: string;
  status: string;
  createdAt: string;
  lastActivityAt: string;
  totalTokens: number;
  activeUsers: number;
  jurisdiction?: string;
  oidcProvider?: string;
}

export default function SubscribersList() {
  const classes = useStyles();
  const navigate = useNavigate();
  const [subscribers, setSubscribers] = useState<Subscriber[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [total, setTotal] = useState(0);
  const [managementDialogOpen, setManagementDialogOpen] = useState(false);
  const [selectedSubscriber, setSelectedSubscriber] = useState<Subscriber | null>(null);
  const [managementTab, setManagementTab] = useState('config');
  const [securitySettings, setSecuritySettings] = useState<any>(null);
  const [apiKeys, setApiKeys] = useState<any[]>([]);
  const [loadingSettings, setLoadingSettings] = useState(false);

  useEffect(() => {
    fetchSubscribers();
  }, [statusFilter]);

  const fetchSubscribers = async () => {
    setLoading(true);
    setError(null);

    try {
      const params = new URLSearchParams({
        tenant_id: 'test-tenant-1', // TODO: Get from auth context
        page: '1',
        page_size: '50',
      });

      if (statusFilter !== 'all') {
        params.append('status', statusFilter);
      }

      const response = await fetch(`/api/admin/subscribers?${params.toString()}`);

      if (!response.ok) {
        if (response.status === 404) {
          throw new Error('Subscribers endpoint requires database connection');
        }
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json') {
          const errorData = await response.json();
          throw new Error(errorData.error || 'Failed to fetch subscribers');
        }
        throw new Error(`Server error: ${response.status}`);
      }

      const contentType = response.headers.get('content-type');
      if (!contentType || !contentType.includes('application/json') {
        throw new Error('Invalid response format from server');
      }

      const data = await response.json();
      setSubscribers(data.subscribers || []);
      setTotal(data.total || 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load subscribers');
      setSubscribers([]);
    } finally {
      setLoading(false);
    }
  };

  const getStatusBadge = (status: string) => {
    switch (status.toLowerCase() {
      case 'active':
        return <Badge appearance="filled" color="success">Active</Badge>;
      case 'suspended':
        return <Badge appearance="filled" color="danger">Suspended</Badge>;
      case 'pending':
        return <Badge appearance="filled" color="warning">Pending</Badge>;
      default:
        return <Badge appearance="filled">{status}</Badge>;
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  };

  const handleViewDetails = (subscriber: Subscriber) => {
    const details = `📋 Subscriber Details\n\n` +
      `🏢 Tenant Name: ${subscriber.tenantName}\n` +
      `🆔 Tenant ID: ${subscriber.tenantId}\n` +
      `📧 Contact Email: ${subscriber.contactEmail}\n` +
      `📊 Status: ${subscriber.status}\n` +
      `🎫 Total Tokens: ${subscriber.totalTokens || 0}\n` +
      `👥 Active Users: ${subscriber.activeUsers || 0}\n` +
      `📅 Created: ${formatDate(subscriber.createdAt)}\n` +
      `⏰ Last Activity: ${subscriber.lastActivityAt ? formatDate(subscriber.lastActivityAt) : 'Never'}\n` +
      `🌍 Jurisdiction: ${subscriber.jurisdiction || 'N/A'}\n` +
      `🔐 OIDC Provider: ${subscriber.oidcProvider || 'N/A'}`;
    alert(details);
    // TODO: Replace with proper detail view dialog/panel
  };

  const handleManage = async (subscriber: Subscriber) => {
    setSelectedSubscriber(subscriber);
    setManagementTab('config');
    setManagementDialogOpen(true);
    setLoadingSettings(true);

    // Load security settings
    try {
      const response = await fetch(
        `/api/admin/security-settings?tenant_id=${subscriber.tenantId}`
      );
      if (response.ok) {
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json') {
          const data = await response.json();
          setSecuritySettings(data);
        }
      }
    } catch (error) {
      console.error('Failed to load security settings:', error);
    }

    // Load API keys
    try {
      const response = await fetch(
        `/api/admin/api-keys?tenant_id=${subscriber.tenantId}`
      );
      if (response.ok) {
        const contentType = response.headers.get('content-type');
        if (contentType && contentType.includes('application/json') {
          const data = await response.json();
          setApiKeys(data.apiKeys || []);
        }
      }
    } catch (error) {
      console.error('Failed to load API keys:', error);
    }

    setLoadingSettings(false);
  };

  const handleDelete = async (subscriber: Subscriber) => {
    if (!confirm(`Are you sure you want to delete subscriber "${subscriber.tenantName}"? This action cannot be undone.`) {
      return;
    }

    try {
      const response = await fetch(`/api/admin/subscribers/${subscriber.id}?tenant_id=test-tenant-1`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        throw new Error('Failed to delete subscriber');
      }

      fetchSubscribers();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete subscriber');
    }
  };

  const filteredSubscribers = subscribers.filter((sub) => {
    if (!searchQuery) return true;
    const query = searchQuery.toLowerCase();
    return (
      sub.tenantName.toLowerCase().includes(query) ||
      sub.tenantId.toLowerCase().includes(query) ||
      sub.contactEmail.toLowerCase().includes(query)
    );
  });

  return (
    <div className={classes.container}>
      {/* Management Dialog */}
      <Dialog
        open={managementDialogOpen}
        onOpenChange={(_, data) => setManagementDialogOpen(data.open)}
      >
        <DialogSurface style={{ maxWidth: '800px', maxHeight: '90vh' }}>
          <DialogBody>
            <DialogTitle>
              Manage Subscriber: {selectedSubscriber?.tenantName}
            </DialogTitle>
            <DialogContent>
              <TabList
                selectedValue={managementTab}
                onTabSelect={(_, data) => setManagementTab(data.value as string)}
              >
                <Tab value="config">Configuration</Tab>
                <Tab value="tokens">Tokens & Permissions</Tab>
                <Tab value="audit">Audit Logs</Tab>
                <Tab value="security">Security</Tab>
                <Tab value="api">API Keys</Tab>
              </TabList>

              <div style={{ marginTop: '24px', minHeight: '300px' }}>
                {managementTab === 'config' && selectedSubscriber && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <Text weight="bold" size={400}>Subscriber Configuration</Text>
                    <div style={{ display: 'grid', gridTemplateColumns: '150px 1fr', gap: '12px' }}>
                      <Text weight="semibold">Tenant ID:</Text>
                      <Text style={{ fontFamily: 'monospace' }}>{selectedSubscriber.tenantId}</Text>

                      <Text weight="semibold">Tenant Name:</Text>
                      <Input value={selectedSubscriber.tenantName} disabled />

                      <Text weight="semibold">Contact Email:</Text>
                      <Input value={selectedSubscriber.contactEmail} disabled />

                      <Text weight="semibold">Status:</Text>
                      <div>{getStatusBadge(selectedSubscriber.status)}</div>

                      <Text weight="semibold">Jurisdiction:</Text>
                      <Text>{selectedSubscriber.jurisdiction || 'Not set'}</Text>

                      <Text weight="semibold">OIDC Provider:</Text>
                      <Text>{selectedSubscriber.oidcProvider || 'Not configured'}</Text>
                    </div>
                    <MessageBar intent="info">
                      <MessageBarBody>
                        Configuration editing will be available in the next release.
                      </MessageBarBody>
                    </MessageBar>
                  </div>
                )}

                {managementTab === 'tokens' && selectedSubscriber && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <Text weight="bold" size={400}>Tokens & Permissions</Text>
                    <div style={{ display: 'flex', gap: '16px' }}>
                      <Card style={{ flex: 1, padding: '16px' }}>
                        <Text weight="semibold">Total Tokens</Text>
                        <Text size={500}>{selectedSubscriber.totalTokens || 0}</Text>
                      </Card>
                      <Card style={{ flex: 1, padding: '16px' }}>
                        <Text weight="semibold">Active Users</Text>
                        <Text size={500}>{selectedSubscriber.activeUsers || 0}</Text>
                      </Card>
                    </div>
                    <Button
                      appearance="primary"
                      onClick={() => navigate(`/admin/tokens?tenant_id=${selectedSubscriber.tenantId}`)}
                    >
                      View All Tokens
                    </Button>
                  </div>
                )}

                {managementTab === 'audit' && selectedSubscriber && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <Text weight="bold" size={400}>Audit Logs</Text>
                    <MessageBar intent="info">
                      <MessageBarBody>
                        Audit log viewing will be available soon. Navigate to the Audit Trail page for system-wide logs.
                      </MessageBarBody>
                    </MessageBar>
                    <Button
                      appearance="primary"
                      onClick={() => {
                        setManagementDialogOpen(false);
                        navigate('/admin/audit');
                      }}
                    >
                      Go to Audit Trail
                    </Button>
                  </div>
                )}

                {managementTab === 'security' && selectedSubscriber && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <Text weight="bold" size={400}>Security Settings</Text>
                    {loadingSettings ? (
                      <Spinner label="Loading security settings..." />
                    ) : securitySettings ? (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                        {/* MFA Settings */}
                        <Card style={{ padding: '16px' }}>
                          <Text weight="semibold" size={400}>Multi-Factor Authentication</Text>
                          <div style={{ display: 'grid', gridTemplateColumns: '200px 1fr', gap: '12px', marginTop: '12px' }}>
                            <Text>MFA Enabled:</Text>
                            <Text>{securitySettings.mfaEnabled ? '✅ Yes' : '❌ No'}</Text>
                            <Text>Required for Admin:</Text>
                            <Text>{securitySettings.mfaRequiredForAdmin ? '✅ Yes' : '❌ No'}</Text>
                            <Text>Grace Period:</Text>
                            <Text>{securitySettings.mfaGracePeriodHours} hours</Text>
                            <Text>Methods:</Text>
                            <Text>{securitySettings.mfaMethods?.join(', ') || 'None configured'}</Text>
                          </div>
                        </Card>

                        {/* IP Whitelisting */}
                        <Card style={{ padding: '16px' }}>
                          <Text weight="semibold" size={400}>IP Whitelisting</Text>
                          <div style={{ display: 'grid', gridTemplateColumns: '200px 1fr', gap: '12px', marginTop: '12px' }}>
                            <Text>Enabled:</Text>
                            <Text>{securitySettings.ipWhitelistEnabled ? '✅ Yes' : '❌ No'}</Text>
                            <Text>Mode:</Text>
                            <Text>{securitySettings.ipWhitelistMode || 'allow'}</Text>
                            <Text>Whitelisted IPs:</Text>
                            <Text>{securitySettings.ipWhitelist?.join(', ') || 'None'}</Text>
                          </div>
                        </Card>

                        {/* Token Policies */}
                        <Card style={{ padding: '16px' }}>
                          <Text weight="semibold" size={400}>Token Expiration Policies</Text>
                          <div style={{ display: 'grid', gridTemplateColumns: '200px 1fr', gap: '12px', marginTop: '12px' }}>
                            <Text>Access Token TTL:</Text>
                            <Text>{securitySettings.accessTokenTtlMinutes} minutes</Text>
                            <Text>Refresh Token TTL:</Text>
                            <Text>{securitySettings.refreshTokenTtlDays} days</Text>
                            <Text>ID Token TTL:</Text>
                            <Text>{securitySettings.idTokenTtlMinutes} minutes</Text>
                            <Text>API Key Default TTL:</Text>
                            <Text>{securitySettings.apiKeyDefaultTtlDays} days</Text>
                          </div>
                        </Card>

                        {/* Session Management */}
                        <Card style={{ padding: '16px' }}>
                          <Text weight="semibold" size={400}>Session Configuration</Text>
                          <div style={{ display: 'grid', gridTemplateColumns: '200px 1fr', gap: '12px', marginTop: '12px' }}>
                            <Text>Session Timeout:</Text>
                            <Text>{securitySettings.sessionTimeoutMinutes} minutes</Text>
                            <Text>Idle Timeout:</Text>
                            <Text>{securitySettings.sessionIdleTimeoutMinutes} minutes</Text>
                            <Text>Max Concurrent:</Text>
                            <Text>{securitySettings.maxConcurrentSessions} sessions</Text>
                            <Text>Session Pinning:</Text>
                            <Text>{securitySettings.sessionPinningEnabled ? '✅ Enabled' : '❌ Disabled'}</Text>
                          </div>
                        </Card>

                        <MessageBar intent="success">
                          <MessageBarBody>
                            Security settings loaded successfully. Configuration changes will be available in the next update.
                          </MessageBarBody>
                        </MessageBar>
                      </div>
                    ) : (
                      <MessageBar intent="warning">
                        <MessageBarBody>
                          No security settings found for this subscriber.
                        </MessageBarBody>
                      </MessageBar>
                    )}
                  </div>
                )}

                {managementTab === 'api' && selectedSubscriber && (
                  <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Text weight="bold" size={400}>API Keys Management</Text>
                      <Button
                        appearance="primary"
                        onClick={async () => {
                          const keyName = prompt('Enter API key name:');
                          if (!keyName) return;

                          try {
                            const response = await fetch('/api/admin/api-keys', {
                              method: 'POST',
                              headers: { 'Content-Type': 'application/json' },
                              body: JSON.stringify({
                                tenantId: selectedSubscriber.tenantId,
                                keyName,
                                scopes: ['read', 'write'],
                                createdBy: 'admin'
                              })
                            });

                            if (response.ok) {
                              const contentType = response.headers.get('content-type');
                              if (contentType && contentType.includes('application/json') {
                                const data = await response.json();
                                alert(`API Key Created!\n\nKey: ${data.secretKey}\n\nStore this securely - it won't be shown again!`);

                                // Reload API keys
                                const listResponse = await fetch(
                                  `/api/admin/api-keys?tenant_id=${selectedSubscriber.tenantId}`
                                );
                                if (listResponse.ok) {
                                  const listContentType = listResponse.headers.get('content-type');
                                  if (listContentType && listContentType.includes('application/json') {
                                    const listData = await listResponse.json();
                                    setApiKeys(listData.apiKeys || []);
                                  }
                                }
                              } else {
                                alert('API endpoint not available in dev mode');
                              }
                            } else {
                              alert('Failed to create API key');
                            }
                          } catch (error) {
                            alert('Error creating API key: ' + error);
                          }
                        }}
                      >
                        + Create New API Key
                      </Button>
                    </div>

                    {loadingSettings ? (
                      <Spinner label="Loading API keys..." />
                    ) : apiKeys.length > 0 ? (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                        {apiKeys.map((key) => (
                          <Card key={key.id} style={{ padding: '16px' }}>
                            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'start' }}>
                              <div style={{ flex: 1 }}>
                                <Text weight="semibold" size={400}>{key.keyName}</Text>
                                <div style={{ display: 'grid', gridTemplateColumns: '120px 1fr', gap: '8px', marginTop: '8px', fontSize: '13px' }}>
                                  <Text>Key Prefix:</Text>
                                  <Text style={{ fontFamily: 'monospace' }}>{key.keyPrefix}...</Text>

                                  <Text>Status:</Text>
                                  <Badge appearance={key.status === 'active' ? 'tint' : 'filled'} color={key.status === 'active' ? 'success' : 'danger'}>
                                    {key.status}
                                  </Badge>

                                  <Text>Scopes:</Text>
                                  <Text>{key.scopes?.join(', ')}</Text>

                                  <Text>Created:</Text>
                                  <Text>{new Date(key.createdAt).toLocaleDateString()}</Text>

                                  {key.lastUsedAt && (
                                    <>
                                      <Text>Last Used:</Text>
                                      <Text>{new Date(key.lastUsedAt).toLocaleString()}</Text>
                                    </>
                                  )}

                                  {key.expiresAt && (
                                    <>
                                      <Text>Expires:</Text>
                                      <Text>{new Date(key.expiresAt).toLocaleDateString()}</Text>
                                    </>
                                  )}

                                  <Text>Total Requests:</Text>
                                  <Text>{key.totalRequests || 0}</Text>
                                </div>
                              </div>

                              <div style={{ display: 'flex', gap: '8px' }}>
                                {key.status === 'active' && (
                                  <Button
                                    appearance="secondary"
                                    size="small"
                                    onClick={async () => {
                                      const reason = prompt('Reason for revocation:');
                                      if (!confirm(`Revoke API key "${key.keyName}"?`) return;

                                      try {
                                        const response = await fetch(
                                          `/api/admin/api-keys/${key.id}/revoke?tenant_id=${selectedSubscriber.tenantId}`,
                                          {
                                            method: 'POST',
                                            headers: { 'Content-Type': 'application/json' },
                                            body: JSON.stringify({ revokedBy: 'admin', reason })
                                          }
                                        );

                                        if (response.ok) {
                                          alert('API key revoked successfully');
                                          // Reload API keys
                                          const listResponse = await fetch(
                                            `/api/admin/api-keys?tenant_id=${selectedSubscriber.tenantId}`
                                          );
                                          if (listResponse.ok) {
                                            const contentType = listResponse.headers.get('content-type');
                                            if (contentType && contentType.includes('application/json') {
                                              const listData = await listResponse.json();
                                              setApiKeys(listData.apiKeys || []);
                                            }
                                          }
                                        } else {
                                          alert('Failed to revoke API key');
                                        }
                                      } catch (error) {
                                        alert('Error revoking API key: ' + error);
                                      }
                                    }}
                                  >
                                    Revoke
                                  </Button>
                                )}
                              </div>
                            </div>
                            {key.description && (
                              <Text style={{ marginTop: '8px', fontSize: '13px', color: tokens.colorNeutralForeground3 }}>
                                {key.description}
                              </Text>
                            )}
                          </Card>
                        ))}
                      </div>
                    ) : (
                      <Card style={{ padding: '24px', textAlign: 'center' }}>
                        <Text>No API keys found for this subscriber.</Text>
                        <Text size={300} style={{ marginTop: '8px', color: tokens.colorNeutralForeground3 }}>
                          Create an API key to enable programmatic access.
                        </Text>
                      </Card>
                    )}

                    <MessageBar intent="info">
                      <MessageBarBody>
                        <MessageBarTitle>API Key Management</MessageBarTitle>
                        Create, revoke, and manage API keys for programmatic access to subscriber resources.
                      </MessageBarBody>
                    </MessageBar>
                  </div>
                )}
              </div>
            </DialogContent>
            <DialogActions>
              <Button appearance="secondary" onClick={() => setManagementDialogOpen(false)}>
                Close
              </Button>
            </DialogActions>
          </DialogBody>
        </DialogSurface>
      </Dialog>

      {/* Header */}
      <div className={classes.header}>
        <div className={classes.headerLeft}>
          <PeopleTeam24Regular style={{ fontSize: '32px', color: tokens.colorBrandForeground1 }} />
          <div>
            <Title3>Subscribers</Title3>
            <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
              Manage tenant subscriptions and access
            </Text>
          </div>
        </div>
        <div className={classes.headerRight}>
          <Button
            appearance="secondary"
            icon={<ArrowSync24Regular />}
            onClick={fetchSubscribers}
            disabled={loading}
          >
            Refresh
          </Button>
          <Button
            appearance="primary"
            icon={<Add24Regular />}
            onClick={() => navigate('/admin/subscribers')}
          >
            New Subscription
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className={classes.stats}>
        <Text weight="semibold">Total Subscribers: {total}</Text>
        <Text>•</Text>
        <Text>Active: {subscribers.filter(s => s.status === 'active').length}</Text>
        <Text>•</Text>
        <Text>Filtered: {filteredSubscribers.length}</Text>
      </div>

      {/* Filters */}
      <Card className={classes.filters}>
        <Search24Regular style={{ fontSize: '20px', color: tokens.colorNeutralForeground3 }} />
        <Input
          className={classes.searchInput}
          placeholder="Search by tenant name, ID, or email..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
        />
        <Filter24Regular style={{ fontSize: '20px', color: tokens.colorNeutralForeground3 }} />
        <Dropdown
          placeholder="Status"
          value={statusFilter}
          onOptionSelect={(_, data) => setStatusFilter(data.optionValue as string)}
          style={{ minWidth: '150px' }}
        >
          <Option value="all">All Status</Option>
          <Option value="active">Active</Option>
          <Option value="pending">Pending</Option>
          <Option value="suspended">Suspended</Option>
        </Dropdown>
      </Card>

      {/* Error Message */}
      {error && (
        <MessageBar intent="error">
          <MessageBarBody>
            <MessageBarTitle>Error</MessageBarTitle>
            {error}
          </MessageBarBody>
        </MessageBar>
      )}

      {/* Table */}
      <Card className={classes.tableCard}>
        {loading ? (
          <div className={classes.emptyState}>
            <Spinner size="large" label="Loading subscribers..." />
          </div>
        ) : filteredSubscribers.length === 0 ? (
          <div className={classes.emptyState}>
            <PeopleTeam24Regular style={{ fontSize: '64px', color: tokens.colorNeutralForeground3 }} />
            <Title3>No Subscribers Found</Title3>
            <Text>
              {searchQuery
                ? 'No subscribers match your search criteria'
                : 'Get started by creating your first subscription'}
            </Text>
            <Button
              appearance="primary"
              icon={<Add24Regular />}
              onClick={() => navigate('/admin/subscribers')}
            >
              Create New Subscription
            </Button>
          </div>
        ) : (
          <Table className={classes.table}>
            <TableHeader>
              <TableRow>
                <TableHeaderCell>Tenant Name</TableHeaderCell>
                <TableHeaderCell>Tenant ID</TableHeaderCell>
                <TableHeaderCell>Contact</TableHeaderCell>
                <TableHeaderCell>Status</TableHeaderCell>
                <TableHeaderCell>Tokens</TableHeaderCell>
                <TableHeaderCell>Users</TableHeaderCell>
                <TableHeaderCell>Created</TableHeaderCell>
                <TableHeaderCell>Last Activity</TableHeaderCell>
                <TableHeaderCell>Actions</TableHeaderCell>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredSubscribers.map((subscriber) => (
                <TableRow key={subscriber.id}>
                  <TableCell>
                    <Text weight="semibold">{subscriber.tenantName}</Text>
                  </TableCell>
                  <TableCell>
                    <Text style={{ fontFamily: 'monospace', fontSize: '12px' }}>
                      {subscriber.tenantId}
                    </Text>
                  </TableCell>
                  <TableCell>
                    <Text size={200}>{subscriber.contactEmail}</Text>
                  </TableCell>
                  <TableCell>{getStatusBadge(subscriber.status)}</TableCell>
                  <TableCell>
                    <Text>{subscriber.totalTokens?.toLocaleString() || 0}</Text>
                  </TableCell>
                  <TableCell>
                    <Text>{subscriber.activeUsers?.toLocaleString() || 0}</Text>
                  </TableCell>
                  <TableCell>
                    <Text size={200}>{formatDate(subscriber.createdAt)}</Text>
                  </TableCell>
                  <TableCell>
                    <Text size={200}>
                      {subscriber.lastActivityAt
                        ? formatDate(subscriber.lastActivityAt)
                        : 'Never'}
                    </Text>
                  </TableCell>
                  <TableCell>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      <Button
                        size="small"
                        icon={<Eye24Regular />}
                        onClick={() => handleViewDetails(subscriber)}
                        title="View Details"
                      />
                      <Button
                        size="small"
                        icon={<Settings24Regular />}
                        onClick={() => handleManage(subscriber)}
                        title="Manage"
                      />
                      <Button
                        size="small"
                        appearance="subtle"
                        icon={<Delete24Regular />}
                        onClick={() => handleDelete(subscriber)}
                        title="Delete"
                      />
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
    </div>
  );
}
