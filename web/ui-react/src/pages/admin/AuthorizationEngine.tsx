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
  Switch,
  Label,
} from '@fluentui/react-components';
import {
  ShieldTask24Regular,
  DocumentAdd24Regular,
  PlayCircle24Regular,
  DatabaseSearch24Regular,
  CheckmarkCircle24Regular,
  DismissCircle24Regular,
  Eye24Regular,
  Edit24Regular,
  Delete24Regular,
} from '@fluentui/react-icons';

// Import admin API hooks
import { useAuthorizationPolicies, useAuthzMutations } from '../../hooks/useAdminApi';

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
  editorWrapper: {
    border: `1px solid ${tokens.colorNeutralStroke1}`,
    borderRadius: '4px',
    padding: '12px',
    minHeight: '300px',
  },
  codeEditor: {
    fontFamily: 'monospace',
    fontSize: '13px',
    width: '100%',
    minHeight: '280px',
    border: 'none',
    outline: 'none',
    resize: 'vertical',
    backgroundColor: tokens.colorNeutralBackground3,
    padding: '8px',
    borderRadius: '4px',
  },
  attributeList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  attributeItem: {
    padding: '12px',
    backgroundColor: tokens.colorNeutralBackground3,
    borderRadius: '4px',
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  decisionResult: {
    padding: '16px',
    borderRadius: '4px',
    marginTop: '16px',
  },
  decisionAllow: {
    backgroundColor: tokens.colorPaletteGreenBackground2,
    border: `2px solid ${tokens.colorPaletteGreenBorder2}`,
  },
  decisionDeny: {
    backgroundColor: tokens.colorPaletteRedBackground2,
    border: `2px solid ${tokens.colorPaletteRedBorder2}`,
  },
});

interface Policy {
  id: string;
  name: string;
  description: string;
  status: 'active' | 'draft' | 'disabled';
  effect: 'allow' | 'deny';
  actions: string[];
  resources: string[];
  conditions?: string;
  createdAt: string;
  updatedAt: string;
}

interface Attribute {
  id: string;
  source: string;
  key: string;
  value: string;
  type: string;
  lastUpdated: string;
}

interface Decision {
  id: string;
  timestamp: string;
  subject: string;
  action: string;
  resource: string;
  decision: 'allow' | 'deny';
  policyId: string;
  policyName: string;
  duration: number;
}

export default function AuthorizationEngine() {
  const classes = useStyles();
  const [selectedTab, setSelectedTab] = useState<string>('pap');
  const [policies, setPolicies] = useState<Policy[]>([]);
  const [attributes, setAttributes] = useState<Attribute[]>([]);
  const [decisions, setDecisions] = useState<Decision[]>([]);
  const [loading, setLoading] = useState(false);
  const [policyDialogOpen, setPolicyDialogOpen] = useState(false);
  const [selectedPolicy, setSelectedPolicy] = useState<Policy | null>(null);

  // PAP form state
  const [policyName, setPolicyName] = useState('');
  const [policyDescription, setPolicyDescription] = useState('');
  const [policyEffect, setPolicyEffect] = useState('allow');
  const [policyActions, setPolicyActions] = useState('');
  const [policyResources, setPolicyResources] = useState('');
  const [policyConditions, setPolicyConditions] = useState('');

  // PIP attribute form state
  const [attributeDialogOpen, setAttributeDialogOpen] = useState(false);
  const [attrSource, setAttrSource] = useState('');
  const [attrKey, setAttrKey] = useState('');
  const [attrValue, setAttrValue] = useState('');
  const [attrType, setAttrType] = useState('string');

  // PDP simulator state
  const [simSubject, setSimSubject] = useState('');
  const [simAction, setSimAction] = useState('');
  const [simResource, setSimResource] = useState('');
  const [simContext, setSimContext] = useState('');
  const [simulationResult, setSimulationResult] = useState<any>(null);

  useEffect(() => {
    fetchPolicies();
    fetchAttributes();
    fetchDecisions();
  }, []);

  const fetchPolicies = async () => {
    try {
      const response = await fetch('/api/admin/authz/policies?tenant_id=test-tenant-1', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });
      if (response.ok) {
        const data = await response.json();
        setPolicies(data.policies || []);
      }
    } catch (error) {
      console.error('Failed to fetch policies:', error);
    }
  };

  const fetchAttributes = async () => {
    try {
      const response = await fetch('/api/admin/authz/attributes?tenant_id=test-tenant-1', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });
      if (response.ok) {
        const data = await response.json();
        setAttributes(data.attributes || []);
      }
    } catch (error) {
      console.error('Failed to fetch attributes:', error);
    }
  };

  const fetchDecisions = async () => {
    try {
      const response = await fetch('/api/admin/authz/decisions?tenant_id=test-tenant-1', {
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });
      if (response.ok) {
        const data = await response.json();
        setDecisions(data.decisions || []);
      }
    } catch (error) {
      console.error('Failed to fetch decisions:', error);
    }
  };

  const handleCreatePolicy = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/admin/authz/policies?tenant_id=test-tenant-1', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
        body: JSON.stringify({
          name: policyName,
          description: policyDescription,
          effect: policyEffect,
          actions: policyActions.split(',').map(a => a.trim()),
          resources: policyResources.split(',').map(r => r.trim()),
          conditions: policyConditions,
        }),
      });

      if (response.ok) {
        fetchPolicies();
        setPolicyDialogOpen(false);
        resetPolicyForm();
      }
    } catch (error) {
      console.error('Failed to create policy:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleUpdatePolicy = async () => {
    if (!selectedPolicy) return;

    setLoading(true);
    try {
      const response = await fetch(`/api/admin/authz/policies/${selectedPolicy.id}?tenant_id=test-tenant-1`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
        body: JSON.stringify({
          name: policyName,
          description: policyDescription,
          effect: policyEffect,
          actions: policyActions.split(',').map(a => a.trim()),
          resources: policyResources.split(',').map(r => r.trim()),
          conditions: policyConditions,
        }),
      });

      if (response.ok) {
        fetchPolicies();
        setPolicyDialogOpen(false);
        resetPolicyForm();
      }
    } catch (error) {
      console.error('Failed to update policy:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDeletePolicy = async (policyId: string) => {
    try {
      await fetch(`/api/admin/authz/policies/${policyId}?tenant_id=test-tenant-1`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });
      fetchPolicies();
    } catch (error) {
      console.error('Failed to delete policy:', error);
    }
  };

  const handleSimulateDecision = async () => {
    setLoading(true);
    setSimulationResult(null);
    try {
      let contextObj = {};
      if (simContext) {
        try {
          contextObj = JSON.parse(simContext);
        } catch (e) {
          console.error('Failed to parse context JSON:', e);
          setSimulationResult({ decision: 'deny', error: 'Invalid JSON in context field' });
          setLoading(false);
          return;
        }
      }
      
      const response = await fetch('/api/admin/authz/simulate?tenant_id=test-tenant-1', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
        body: JSON.stringify({
          subject: simSubject,
          action: simAction,
          resource: simResource,
          context: contextObj,
        }),
      });

      if (response.ok) {
        const data = await response.json();
        setSimulationResult(data);
      } else {
        setSimulationResult({ decision: 'deny', error: 'Simulation request failed' });
      }
    } catch (error) {
      console.error('Failed to simulate decision:', error);
      setSimulationResult({ decision: 'deny', error: 'Simulation failed' });
    } finally {
      setLoading(false);
    }
  };

  const handleCreateAttribute = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/admin/authz/attributes?tenant_id=test-tenant-1', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
        body: JSON.stringify({
          source: attrSource,
          key: attrKey,
          value: attrValue,
          type: attrType,
        }),
      });

      if (response.ok) {
        await fetchAttributes();
        setAttributeDialogOpen(false);
        resetAttributeForm();
      }
    } catch (error) {
      console.error('Failed to create attribute:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteAttribute = async (id: string) => {
    if (!confirm('Are you sure you want to delete this attribute?')) return;
    
    try {
      const response = await fetch(`/api/admin/authz/attributes/${id}?tenant_id=test-tenant-1`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${localStorage.getItem('admin_token')}`,
        },
      });

      if (response.ok) {
        await fetchAttributes();
      }
    } catch (error) {
      console.error('Failed to delete attribute:', error);
    }
  };

  const resetPolicyForm = () => {
    setPolicyName('');
    setPolicyDescription('');
    setPolicyEffect('allow');
    setPolicyActions('');
    setPolicyResources('');
    setPolicyConditions('');
    setSelectedPolicy(null);
  };

  const resetAttributeForm = () => {
    setAttrSource('');
    setAttrKey('');
    setAttrValue('');
    setAttrType('string');
  };

  const editPolicy = (policy: Policy) => {
    setSelectedPolicy(policy);
    setPolicyName(policy.name);
    setPolicyDescription(policy.description);
    setPolicyEffect(policy.effect);
    setPolicyActions(policy.actions.join(', '));
    setPolicyResources(policy.resources.join(', '));
    setPolicyConditions(policy.conditions || '');
    setPolicyDialogOpen(true);
  };

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'active':
        return <Badge appearance="filled" color="success">Active</Badge>;
      case 'draft':
        return <Badge appearance="filled" color="warning">Draft</Badge>;
      case 'disabled':
        return <Badge appearance="filled" color="danger">Disabled</Badge>;
      default:
        return <Badge>{status}</Badge>;
    }
  };

  const policyColumns: TableColumnDefinition<Policy>[] = [
    createTableColumn<Policy>({
      columnId: 'name',
      renderHeaderCell: () => 'Policy Name',
      renderCell: (item) => (
        <TableCellLayout>
          <Text weight="semibold">{item.name}</Text>
          <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
            {item.description}
          </Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Policy>({
      columnId: 'effect',
      renderHeaderCell: () => 'Effect',
      renderCell: (item) => (
        <TableCellLayout>
          <Badge appearance="tint" color={item.effect === 'allow' ? 'success' : 'danger'}>
            {item.effect.toUpperCase()}
          </Badge>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Policy>({
      columnId: 'actions',
      renderHeaderCell: () => 'Actions',
      renderCell: (item) => (
        <TableCellLayout>
          <Text size={200}>{item.actions.join(', ')}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Policy>({
      columnId: 'status',
      renderHeaderCell: () => 'Status',
      renderCell: (item) => (
        <TableCellLayout>
          {getStatusBadge(item.status)}
        </TableCellLayout>
      ),
    }),
    createTableColumn<Policy>({
      columnId: 'updated',
      renderHeaderCell: () => 'Last Updated',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{new Date(item.updatedAt).toLocaleString()}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Policy>({
      columnId: 'operations',
      renderHeaderCell: () => 'Operations',
      renderCell: (item) => (
        <TableCellLayout>
          <div style={{ display: 'flex', gap: '8px' }}>
            <Button size="small" icon={<Eye24Regular />} />
            <Button size="small" icon={<Edit24Regular />} onClick={() => editPolicy(item)} />
            <Button size="small" icon={<Delete24Regular />} onClick={() => handleDeletePolicy(item.id)} />
          </div>
        </TableCellLayout>
      ),
    }),
  ];

  const decisionColumns: TableColumnDefinition<Decision>[] = [
    createTableColumn<Decision>({
      columnId: 'timestamp',
      renderHeaderCell: () => 'Timestamp',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{new Date(item.timestamp).toLocaleString()}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Decision>({
      columnId: 'subject',
      renderHeaderCell: () => 'Subject',
      renderCell: (item) => (
        <TableCellLayout>
          <Text weight="semibold">{item.subject}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Decision>({
      columnId: 'action',
      renderHeaderCell: () => 'Action',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{item.action}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Decision>({
      columnId: 'resource',
      renderHeaderCell: () => 'Resource',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{item.resource}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Decision>({
      columnId: 'decision',
      renderHeaderCell: () => 'Decision',
      renderCell: (item) => (
        <TableCellLayout>
          <Badge appearance="filled" color={item.decision === 'allow' ? 'success' : 'danger'}>
            {item.decision === 'allow' ? <CheckmarkCircle24Regular /> : <DismissCircle24Regular />}
            {item.decision.toUpperCase()}
          </Badge>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Decision>({
      columnId: 'policy',
      renderHeaderCell: () => 'Policy Applied',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{item.policyName}</Text>
        </TableCellLayout>
      ),
    }),
    createTableColumn<Decision>({
      columnId: 'duration',
      renderHeaderCell: () => 'Duration',
      renderCell: (item) => (
        <TableCellLayout>
          <Text>{item.duration}ms</Text>
        </TableCellLayout>
      ),
    }),
  ];

  const activePolicies = policies.filter(p => p.status === 'active').length;
  const totalDecisions = decisions.length;
  const allowedDecisions = decisions.filter(d => d.decision === 'allow').length;

  return (
    <div className={classes.container}>
      <div className={classes.header}>
        <div className={classes.headerLeft}>
          <ShieldTask24Regular style={{ fontSize: '24px' }} />
          <Title3>Authorization Engine</Title3>
        </div>
      </div>

      {/* Overview Cards */}
      <div className={classes.cardsGrid}>
        <Card className={classes.card}>
          <Text className={classes.metricValue} style={{ color: tokens.colorPaletteGreenForeground1 }}>
            {activePolicies}
          </Text>
          <Text className={classes.metricLabel}>Active Policies</Text>
        </Card>

        <Card className={classes.card}>
          <Text className={classes.metricValue}>
            {totalDecisions}
          </Text>
          <Text className={classes.metricLabel}>Total Decisions</Text>
        </Card>

        <Card className={classes.card}>
          <Text className={classes.metricValue} style={{ color: tokens.colorPaletteGreenForeground1 }}>
            {totalDecisions > 0 ? Math.round((allowedDecisions / totalDecisions) * 100) : 0}%
          </Text>
          <Text className={classes.metricLabel}>Allow Rate</Text>
        </Card>

        <Card className={classes.card}>
          <Text className={classes.metricValue}>
            {attributes.length}
          </Text>
          <Text className={classes.metricLabel}>Attributes Sources</Text>
        </Card>
      </div>

      {/* Main Tabs */}
      <Card className={classes.card}>
        <TabList selectedValue={selectedTab} onTabSelect={(_, data) => setSelectedTab(data.value as string)}>
          <Tab value="pap" icon={<DocumentAdd24Regular />}>PAP - Policy Admin</Tab>
          <Tab value="pip" icon={<DatabaseSearch24Regular />}>PIP - Attributes</Tab>
          <Tab value="pdp" icon={<PlayCircle24Regular />}>PDP - Decision Simulator</Tab>
          <Tab value="pep" icon={<ShieldTask24Regular />}>PEP - Enforcement Log</Tab>
        </TabList>

        {/* PAP - Policy Administration Point */}
        {selectedTab === 'pap' && (
            <div style={{ marginTop: '24px' }}>
              <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between' }}>
                <Text weight="semibold" size={400}>Policy Management</Text>
                <Dialog open={policyDialogOpen} onOpenChange={(_, data) => {
                  setPolicyDialogOpen(data.open);
                  if (!data.open) resetPolicyForm();
                }}>
                  <DialogTrigger>
                    <Button appearance="primary" icon={<DocumentAdd24Regular />}>
                      Create Policy
                    </Button>
                  </DialogTrigger>
                  <DialogSurface style={{ maxWidth: '800px' }}>
                    <DialogBody>
                      <DialogTitle>{selectedPolicy ? 'Edit Policy' : 'Create New Policy'}</DialogTitle>
                      <DialogContent>
                        <div className={classes.form}>
                          <Field label="Policy Name" required>
                            <Input
                              value={policyName}
                              onChange={(e) => setPolicyName(e.target.value)}
                              placeholder="read-documents-policy"
                            />
                          </Field>

                          <Field label="Description">
                            <Input
                              value={policyDescription}
                              onChange={(e) => setPolicyDescription(e.target.value)}
                              placeholder="Allow users to read documents"
                            />
                          </Field>

                          <Field label="Effect">
                            <Dropdown
                              value={policyEffect}
                              onOptionSelect={(_, data) => setPolicyEffect(data.optionValue as string)}
                            >
                              <Option value="allow">Allow</Option>
                              <Option value="deny">Deny</Option>
                            </Dropdown>
                          </Field>

                          <Field label="Actions (comma-separated)">
                            <Input
                              value={policyActions}
                              onChange={(e) => setPolicyActions(e.target.value)}
                              placeholder="read, list, download"
                            />
                          </Field>

                          <Field label="Resources (comma-separated)">
                            <Input
                              value={policyResources}
                              onChange={(e) => setPolicyResources(e.target.value)}
                              placeholder="document:*, file:/docs/*"
                            />
                          </Field>

                          <Field label="Conditions (JSON)">
                            <Textarea
                              value={policyConditions}
                              onChange={(e) => setPolicyConditions(e.target.value)}
                              placeholder='{"ip": "10.0.0.0/8", "time": "09:00-17:00"}'
                              rows={4}
                            />
                          </Field>
                        </div>
                      </DialogContent>
                      <DialogActions>
                        <Button appearance="secondary" onClick={() => {
                          setPolicyDialogOpen(false);
                          resetPolicyForm();
                        }}>
                          Cancel
                        </Button>
                        <Button
                          appearance="primary"
                          onClick={selectedPolicy ? handleUpdatePolicy : handleCreatePolicy}
                          disabled={loading || !policyName}
                        >
                          {loading ? 'Saving...' : (selectedPolicy ? 'Update Policy' : 'Create Policy')}
                        </Button>
                      </DialogActions>
                    </DialogBody>
                  </DialogSurface>
                </Dialog>
              </div>

              {policies.length === 0 ? (
                <MessageBar intent="info">
                  <MessageBarBody>
                    No policies configured. Click "Create Policy" to add your first authorization policy.
                  </MessageBarBody>
                </MessageBar>
              ) : (
                <DataGrid items={policies} columns={policyColumns} sortable resizableColumns>
                  <DataGridHeader>
                    <DataGridRow>
                      {({ renderHeaderCell }) => (
                        <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>
                      )}
                    </DataGridRow>
                  </DataGridHeader>
                  <DataGridBody<Policy>>
                    {({ item, rowId }) => (
                      <DataGridRow<Policy> key={rowId}>
                        {({ renderCell }) => (
                          <DataGridCell>{renderCell(item)}</DataGridCell>
                        )}
                      </DataGridRow>
                    )}
                  </DataGridBody>
                </DataGrid>
              )}
            </div>
          )}

          {/* PIP - Policy Information Point */}
          {selectedTab === 'pip' && (
            <div style={{ marginTop: '24px' }}>
              <div style={{ marginBottom: '16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <Text weight="semibold" size={400}>Attribute Sources</Text>
                <Dialog open={attributeDialogOpen} onOpenChange={(_, data) => {
                  setAttributeDialogOpen(data.open);
                  if (!data.open) resetAttributeForm();
                }}>
                  <DialogTrigger>
                    <Button appearance="primary" icon={<DocumentAdd24Regular />}>
                      Add Attribute
                    </Button>
                  </DialogTrigger>
                  <DialogSurface>
                    <DialogBody>
                      <DialogTitle>Add New Attribute</DialogTitle>
                      <DialogContent>
                        <div className={classes.form}>
                          <Field label="Source" required>
                            <Input
                              value={attrSource}
                              onChange={(e) => setAttrSource(e.target.value)}
                              placeholder="user_profile, network, device"
                            />
                          </Field>
                          <Field label="Key" required>
                            <Input
                              value={attrKey}
                              onChange={(e) => setAttrKey(e.target.value)}
                              placeholder="department, ip_range, device_id"
                            />
                          </Field>
                          <Field label="Value" required>
                            <Input
                              value={attrValue}
                              onChange={(e) => setAttrValue(e.target.value)}
                              placeholder="engineering, 10.0.0.0/8"
                            />
                          </Field>
                          <Field label="Type">
                            <Dropdown
                              placeholder="Select type"
                              value={attrType}
                              selectedOptions={[attrType]}
                              onOptionSelect={(_, data) => setAttrType(data.optionValue as string)}
                            >
                              <Option value="string">String</Option>
                              <Option value="number">Number</Option>
                              <Option value="boolean">Boolean</Option>
                              <Option value="json">JSON</Option>
                            </Dropdown>
                          </Field>
                        </div>
                      </DialogContent>
                      <DialogActions>
                        <Button appearance="secondary" onClick={() => setAttributeDialogOpen(false)}>
                          Cancel
                        </Button>
                        <Button
                          appearance="primary"
                          onClick={handleCreateAttribute}
                          disabled={loading || !attrSource || !attrKey || !attrValue}
                        >
                          {loading ? 'Creating...' : 'Create Attribute'}
                        </Button>
                      </DialogActions>
                    </DialogBody>
                  </DialogSurface>
                </Dialog>
              </div>
              {attributes.length === 0 ? (
                <MessageBar intent="info">
                  <MessageBarBody>
                    No attributes configured yet. Add attribute sources to enrich policy decisions.
                  </MessageBarBody>
                </MessageBar>
              ) : (
                <div className={classes.attributeList}>
                  {attributes.map((attr) => (
                    <div key={attr.id} className={classes.attributeItem}>
                      <div style={{ flex: 1 }}>
                        <Text weight="semibold">{attr.key}</Text>
                        <Text size={200} style={{ color: tokens.colorNeutralForeground3, display: 'block' }}>
                          Source: {attr.source} | Type: {attr.type} | Value: {attr.value}
                        </Text>
                        <Text size={100} style={{ color: tokens.colorNeutralForeground4, display: 'block' }}>
                          Updated: {new Date(attr.lastUpdated).toLocaleString()}
                        </Text>
                      </div>
                      <div style={{ display: 'flex', gap: '8px' }}>
                        <Button 
                          size="small" 
                          icon={<Edit24Regular />}
                          onClick={() => {
                            setAttrSource(attr.source);
                            setAttrKey(attr.key);
                            setAttrValue(attr.value);
                            setAttrType(attr.type);
                            setAttributeDialogOpen(true);
                          }}
                        >
                          Edit
                        </Button>
                        <Button 
                          size="small" 
                          icon={<Delete24Regular />}
                          onClick={() => handleDeleteAttribute(attr.id)}
                        >
                          Delete
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* PDP - Policy Decision Point */}
          {selectedTab === 'pdp' && (
            <div style={{ marginTop: '24px' }} className={classes.form}>
              <Text weight="semibold" size={400}>Decision Simulator</Text>
              <Text size={300}>Test authorization decisions with different inputs</Text>

              <Field label="Subject" required>
                <Input
                  value={simSubject}
                  onChange={(e) => setSimSubject(e.target.value)}
                  placeholder="user:john.doe@example.com"
                />
              </Field>

              <Field label="Action" required>
                <Input
                  value={simAction}
                  onChange={(e) => setSimAction(e.target.value)}
                  placeholder="read"
                />
              </Field>

              <Field label="Resource" required>
                <Input
                  value={simResource}
                  onChange={(e) => setSimResource(e.target.value)}
                  placeholder="document:12345"
                />
              </Field>

              <Field label="Context (JSON)">
                <Textarea
                  value={simContext}
                  onChange={(e) => setSimContext(e.target.value)}
                  placeholder='{"ip": "10.0.1.5", "department": "engineering"}'
                  rows={4}
                />
              </Field>

              <Button
                appearance="primary"
                icon={<PlayCircle24Regular />}
                onClick={handleSimulateDecision}
                disabled={loading || !simSubject || !simAction || !simResource}
              >
                {loading ? 'Evaluating...' : 'Simulate Decision'}
              </Button>

              {simulationResult && (
                <div className={`${classes.decisionResult} ${
                  simulationResult.decision === 'allow' ? classes.decisionAllow : classes.decisionDeny
                }`}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '12px' }}>
                    {simulationResult.decision === 'allow' ? (
                      <CheckmarkCircle24Regular style={{ fontSize: '32px', color: tokens.colorPaletteGreenForeground1 }} />
                    ) : (
                      <DismissCircle24Regular style={{ fontSize: '32px', color: tokens.colorPaletteRedForeground1 }} />
                    )}
                    <div>
                      <Text weight="bold" size={500}>
                        Decision: {simulationResult.decision.toUpperCase()}
                      </Text>
                      <Text size={300} style={{ display: 'block' }}>
                        Policy Applied: {simulationResult.policyName || 'Default Deny'}
                      </Text>
                    </div>
                  </div>
                  <Text size={200}>Evaluation Time: {simulationResult.duration || 0}ms</Text>
                  {simulationResult.reason && (
                    <Text size={200} style={{ display: 'block', marginTop: '8px' }}>
                      Reason: {simulationResult.reason}
                    </Text>
                  )}
                </div>
              )}
            </div>
          )}

          {/* PEP - Policy Enforcement Point */}
          {selectedTab === 'pep' && (
            <div style={{ marginTop: '24px' }}>
              <Text weight="semibold" size={400} style={{ marginBottom: '16px', display: 'block' }}>
                Enforcement Decision Log
              </Text>
              {decisions.length === 0 ? (
                <MessageBar intent="info">
                  <MessageBarBody>
                    No enforcement decisions logged yet. Use the PDP simulator to test authorization decisions, or decisions will appear here as they occur in the system.
                  </MessageBarBody>
                </MessageBar>
              ) : (
                <DataGrid items={decisions} columns={decisionColumns} sortable resizableColumns>
                  <DataGridHeader>
                    <DataGridRow>
                      {({ renderHeaderCell }) => (
                        <DataGridHeaderCell>{renderHeaderCell()}</DataGridHeaderCell>
                      )}
                    </DataGridRow>
                  </DataGridHeader>
                  <DataGridBody<Decision>>
                    {({ item, rowId }) => (
                      <DataGridRow<Decision> key={rowId}>
                        {({ renderCell }) => (
                          <DataGridCell>{renderCell(item)}</DataGridCell>
                        )}
                      </DataGridRow>
                    )}
                  </DataGridBody>
                </DataGrid>
              )}
            </div>
          )}
      </Card>
    </div>
  );
}
