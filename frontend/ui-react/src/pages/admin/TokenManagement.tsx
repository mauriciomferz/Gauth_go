import { useState, useEffect } from 'react';
import {
  makeStyles,
  tokens,
  Card,
  Text,
  Title3,
  Button,
  Input,
  TabList,
  Tab,
  Badge,
  Field,
  Dropdown,
  Option,
  Textarea,
  DataGrid,
  DataGridBody,
  DataGridRow,
  DataGridHeader,
  DataGridHeaderCell,
  DataGridCell,
  TableCellLayout,
  TableColumnDefinition,
  createTableColumn,
  Dialog,
  DialogTrigger,
  DialogSurface,
  DialogTitle,
  DialogBody,
  DialogActions,
  DialogContent,
  MessageBar,
  MessageBarBody,
} from '@fluentui/react-components';
import {
  Key24Regular,
  Add24Regular,
  Shield24Regular,
  Dismiss24Regular,
  ArrowSync24Regular,
  DocumentSearch24Regular,
  Copy24Regular,
} from '@fluentui/react-icons';

const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    gap: '24px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  headerLeft: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
  },
  cardsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(240px, 1fr))',
    gap: '16px',
  },
  card: {
    padding: '20px',
  },
  metricValue: {
    fontSize: '32px',
    fontWeight: 600,
    marginBottom: '8px',
  },
  metricLabel: {
    fontSize: '14px',
    color: tokens.colorNeutralForeground3,
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  twoColumn: {
    display: 'grid',
    gridTemplateColumns: '1fr 1fr',
    gap: '16px',
  },
  tokenDisplay: {
    padding: '12px',
    backgroundColor: tokens.colorNeutralBackground3,
    borderRadius: '4px',
    fontFamily: 'monospace',
    fontSize: '12px',
    wordBreak: 'break-all',
    maxHeight: '200px',
    overflowY: 'auto',
  },
});

interface Token {
  id: string;
  subscriberId: string;
  subscriberName: string;
  type: 'access' | 'refresh';
  status: 'active' | 'expired' | 'revoked';
  createdAt: string;
  expiresAt: string;
  lastUsed: string;
  usageCount: number;
}

export default function TokenManagement() {
  const classes = useStyles();
  const [selectedTab, setSelectedTab] = useState<string>('overview');
  const [tokens, setTokens] = useState<Token[]>([]);
  const [loading, setLoading] = useState(false);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [validateDialogOpen, setValidateDialogOpen] = useState(false);
  const [createdToken, setCreatedToken] = useState<string>('');
  
  // Form states
  const [subscriberId, setSubscriberId] = useState('');
  const [tokenType, setTokenType] = useState('access');
  const [ttl, setTtl] = useState('3600');
  const [scopes, setScopes] = useState('');
  const [validateTokenInput, setValidateTokenInput] = useState('');
  const [validationResult, setValidationResult] = useState<any>(null);

  useEffect(() => {
    fetchTokens();
  }, []);

  const fetchTokens = async () => {
    try {
      const response = await fetch('/api/admin/tokens', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });
      if (response.ok) {
        const data = await response.json();
        setTokens(data.tokens || []);
      }
    } catch (error) {
      console.error('Failed to fetch tokens:', error);
    }
  };

  const handleCreateToken = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/admin/tokens/create', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
        body: JSON.stringify({
          subscriberId,
          tokenType,
          ttl: parseInt(ttl),
          scopes: scopes.split(',').map(s => s.trim()),
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setCreatedToken(data.token);
        fetchTokens();
      }
    } catch (error) {
      console.error('Failed to create token:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleValidateToken = async () => {
    setLoading(true);
    setValidationResult(null);
    try {
      const response = await fetch('/api/admin/tokens/validate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
        body: JSON.stringify({ token: validateTokenInput }),
      });

      if (response.ok) {
        const data = await response.json();
        setValidationResult(data);
      }
    } catch (error) {
      console.error('Failed to validate token:', error);
      setValidationResult({ valid: false, error: 'Validation failed' });
    } finally {
      setLoading(false);
    }
  };

  const handleRevokeToken = async (tokenId: string) => {
    try {
      await fetch(`/api/admin/tokens/${tokenId}/revoke`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });
      fetchTokens();
    } catch (error) {
      console.error('Failed to revoke token:', error);
    }
  };

  const handleRefreshToken = async (tokenId: string) => {
    try {
      const response = await fetch(`/api/admin/tokens/${tokenId}/refresh`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });
      if (response.ok) {
        fetchTokens();
      }
    } catch (error) {
      console.error('Failed to refresh token:', error);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return <Badge appearance="filled" color="success">Active</Badge>;
      case 'expired':
        return <Badge appearance="filled" color="warning">Expired</Badge>;
      case 'revoked':
        return <Badge appearance="filled" color="danger">Revoked</Badge>;
      default:
        return <Badge>{status}</Badge>;
    }
  };

  const columns: TableColumnDefinition<Token>[] = [
    createTableColumn<Token>({
      columnId: 'subscriber',
      renderHeaderCell: () => 'Subscriber',
      renderCell: (item) => (
        <TableCellLayout>
          <Text weight="semibold">{item.subscriberName}</Text>
          <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
            {item.subscriberId}
          </Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Token>({
      columnId: 'type',
      renderHeaderCell: () => 'Type',
      renderCell: (item) => (
        <TableCellLayout>
          <Badge appearance="tint">{item.type}</Badge>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Token>({
      columnId: 'status',
      renderHeaderCell: () => 'Status',
      renderCell: (item) => (
        <TableCellLayout>
          {getStatusBadge(item.status)}
        </TableCellLayout>
      ),
    }),
    createTableColumn<Token>({
      columnId: 'created',
      renderHeaderCell: () => 'Created',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{new Date(item.createdAt).toLocaleString()}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Token>({
      columnId: 'expires',
      renderHeaderCell: () => 'Expires',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{new Date(item.expiresAt).toLocaleString()}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Token>({
      columnId: 'usage',
      renderHeaderCell: () => 'Usage',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{item.usageCount} times</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Token>({
      columnId: 'actions',
      renderHeaderCell: () => 'Actions',
      renderCell: (item) => (
        <TableCellLayout>
          <div style={{ display: 'flex', gap: '8px' }}>
            {item.status === 'active' && item.type === 'refresh' && (
              <Button
                size="small"
                icon={<ArrowSync24Regular />}
                onClick={() => handleRefreshToken(item.id)}
              />
            )}
            {item.status === 'active' && (
              <Button
                size="small"
                icon={<Dismiss24Regular />}
                onClick={() => handleRevokeToken(item.id)}
              />
            )}
          </div>
        </TableCellLayout>
      ),
    }),
  ];

  const activeTokens = tokens.filter(t => t.status === 'active').length;
  const expiredTokens = tokens.filter(t => t.status === 'expired').length;
  const revokedTokens = tokens.filter(t => t.status === 'revoked').length;

  return (
    <div className={classes.container}>
      <div className={classes.header}>
        <div className={classes.headerLeft}>
          <Key24Regular style={{ fontSize: '24px' }} />
          <Title3>Token Management</Title3>
        </div>
        <Dialog open={createDialogOpen} onOpenChange={(_, data) => setCreateDialogOpen(data.open)}>
          <DialogTrigger>
            <Button appearance="primary" icon={<Add24Regular />}>
              Create Token
            </Button>
          </DialogTrigger>
          <DialogSurface>
            <DialogBody>
              <DialogTitle>Create New Token</DialogTitle>
              <DialogContent>
                <div className={classes.form}>
                  <Field label="Subscriber ID" required>
                    <Input
                      value={subscriberId}
                      onChange={(e) => setSubscriberId(e.target.value)}
                      placeholder="sub-acme-001"
                    />
                  </Field>

                  <Field label="Token Type">
                    <Dropdown
                      value={tokenType}
                      onOptionSelect={(_, data) => setTokenType(data.optionValue as string)}
                    >
                      <Option value="access">Access Token</Option>
                      <Option value="refresh">Refresh Token</Option>
                    </Dropdown>
                  </Field>

                  <Field label="TTL (seconds)">
                    <Input
                      type="number"
                      value={ttl}
                      onChange={(e) => setTtl(e.target.value)}
                      placeholder="3600"
                    />
                  </Field>

                  <Field label="Scopes (comma-separated)">
                    <Textarea
                      value={scopes}
                      onChange={(e) => setScopes(e.target.value)}
                      placeholder="read, write, admin"
                      rows={3}
                    />
                  </Field>

                  {createdToken && (
                    <div>
                      <Text weight="semibold" style={{ marginBottom: '8px', display: 'block' }}>
                        Generated Token:
                      </Text>
                      <div className={classes.tokenDisplay}>
                        {createdToken}
                      </div>
                      <Button
                        size="small"
                        icon={<Copy24Regular />}
                        onClick={() => copyToClipboard(createdToken)}
                        style={{ marginTop: '8px' }}
                      >
                        Copy to Clipboard
                      </Button>
                    </div>
                  )}
                </div>
              </DialogContent>
              <DialogActions>
                <Button appearance="secondary" onClick={() => setCreateDialogOpen(false)}>
                  Close
                </Button>
                <Button
                  appearance="primary"
                  onClick={handleCreateToken}
                  disabled={loading || !subscriberId}
                >
                  {loading ? 'Creating...' : 'Create Token'}
                </Button>
              </DialogActions>
            </DialogBody>
          </DialogSurface>
        </Dialog>
      </div>

      {/* Overview Cards */}
      <div className={classes.cardsGrid}>
        <Card className={classes.card}>
          <Text className={classes.metricValue} style={{ color: tokens.colorPaletteGreenForeground1 }}>
            {activeTokens}
          </Text>
          <Text className={classes.metricLabel}>Active Tokens</Text>
        </Card>

        <Card className={classes.card}>
          <Text className={classes.metricValue} style={{ color: tokens.colorPaletteYellowForeground1 }}>
            {expiredTokens}
          </Text>
          <Text className={classes.metricLabel}>Expired Tokens</Text>
        </Card>

        <Card className={classes.card}>
          <Text className={classes.metricValue} style={{ color: tokens.colorPaletteRedForeground1 }}>
            {revokedTokens}
          </Text>
          <Text className={classes.metricLabel}>Revoked Tokens</Text>
        </Card>

        <Card className={classes.card}>
          <Text className={classes.metricValue}>
            {tokens.length}
          </Text>
          <Text className={classes.metricLabel}>Total Tokens</Text>
        </Card>
      </div>

      {/* Tabs */}
      <Card className={classes.card}>
        <TabList selectedValue={selectedTab} onTabSelect={(_, data) => setSelectedTab(data.value as string)}>
          <Tab value="overview" icon={<Key24Regular />}>Overview</Tab>
          <Tab value="validate" icon={<Shield24Regular />}>Validate</Tab>
          <Tab value="search" icon={<DocumentSearch24Regular />}>Search</Tab>
        </TabList>

        {selectedTab === 'overview' && (
          <div style={{ marginTop: '24px' }}>
            <DataGrid items={tokens} columns={columns} sortable resizableColumns>
                <DataGridHeader>
                  <DataGridRow>
                    {({ renderHeaderCell }) => (
                      <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>
                    )}
                  </DataGridRow>
                </DataGridHeader>
                <DataGridBody<Token>>
                  {({ item, rowId }) => (
                    <DataGridRow<Token> key={rowId}>
                      {({ renderCell }) => (
                        <DataGridCell>{renderCell(item)}</DataGridCell>
                      )}
                    </DataGridRow>
                  )}
                </DataGridBody>
              </DataGrid>
            </div>
          )}

        {selectedTab === 'validate' && (
          <div style={{ marginTop: '24px' }} className={classes.form}>
              <Field label="Token to Validate">
                <Textarea
                  value={validateTokenInput}
                  onChange={(e) => setValidateTokenInput(e.target.value)}
                  placeholder="Paste JWT token here..."
                  rows={6}
                />
              </Field>

              <Button
                appearance="primary"
                icon={<Shield24Regular />}
                onClick={handleValidateToken}
                disabled={loading || !validateTokenInput}
              >
                {loading ? 'Validating...' : 'Validate Token'}
              </Button>

              {validationResult && (
                <MessageBar intent={validationResult.valid ? 'success' : 'error'}>
                  <MessageBarBody>
                    {validationResult.valid ? (
                      <div>
                        <Text weight="semibold">Token is valid</Text>
                        <div style={{ marginTop: '12px' }}>
                          <Text size={200}>Subscriber: {validationResult.subscriberId}</Text><br />
                          <Text size={200}>Type: {validationResult.type}</Text><br />
                          <Text size={200}>Expires: {new Date(validationResult.expiresAt).toLocaleString()}</Text>
                        </div>
                      </div>
                    ) : (
                      <Text weight="semibold">Token is invalid: {validationResult.error}</Text>
                    )}
                  </MessageBarBody>
                </MessageBar>
              )}
            </div>
          )}

          {selectedTab === 'search' && (
            <div style={{ marginTop: '24px' }} className={classes.form}>
              <div className={classes.twoColumn}>
                <Field label="Subscriber ID">
                  <Input placeholder="Search by subscriber..." />
                </Field>
                <Field label="Status">
                  <Dropdown placeholder="All statuses">
                    <Option value="active">Active</Option>
                    <Option value="expired">Expired</Option>
                    <Option value="revoked">Revoked</Option>
                  </Dropdown>
                </Field>
              </div>
              <Button appearance="primary">Search Tokens</Button>
            </div>
          )}
      </Card>
    </div>
  );
}
