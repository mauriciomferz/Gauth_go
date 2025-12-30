import React, { useState, useEffect } from 'react';
import { addTenantParam } from '../../utils/tenant';
import {
  makeStyles,
  shorthands,
  tokens,
  Title3,
  Card,
  Button,
  Text,
  Badge,
  Input,
  Label,
  Textarea,
  Dropdown,
  Option,
  Switch,
  Dialog,

  DialogSurface,
  DialogTitle,
  DialogBody,
  DialogActions,
  DialogContent,
  Tab,
  TabList,
  ProgressBar,
  Spinner,
} from '@fluentui/react-components';
import {
  Add24Regular,
  Delete24Regular,
  Edit24Regular,
  ArrowSync24Regular,
  CheckmarkCircle24Regular,
  DismissCircle24Regular,
  History24Regular,

  Settings24Regular,
  Code24Regular,
  Database24Regular,
  Flag24Regular,
} from '@fluentui/react-icons';

// Import admin API hooks


const useStyles = makeStyles({
  root: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('16px'),
    ...shorthands.padding('24px'),
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  tabsContainer: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('16px'),
  },
  overviewCards: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(250px, 1fr))',
    ...shorthands.gap('16px'),
    marginBottom: '24px',
  },
  card: {
    ...shorthands.padding('16px'),
  },
  cardHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  cardTitle: {
    fontSize: tokens.fontSizeBase300,
    fontWeight: tokens.fontWeightSemibold,
  },
  cardValue: {
    fontSize: tokens.fontSizeBase600,
    fontWeight: tokens.fontWeightBold,
    color: tokens.colorBrandForeground1,
    marginTop: '8px',
  },
  variablesGrid: {
    display: 'grid',
    ...shorthands.gap('12px'),
  },
  variableCard: {
    ...shorthands.padding('12px'),
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  variableInfo: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('4px'),
    flex: 1,
  },
  variableName: {
    fontWeight: tokens.fontWeightSemibold,
    fontFamily: tokens.fontFamilyMonospace,
  },
  variableValue: {
    color: tokens.colorNeutralForeground3,
    fontFamily: tokens.fontFamilyMonospace,
    fontSize: tokens.fontSizeBase200,
  },
  variableActions: {
    display: 'flex',
    ...shorthands.gap('8px'),
  },
  editorContainer: {
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke1),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    ...shorthands.padding('16px'),
    backgroundColor: tokens.colorNeutralBackground1,
  },
  editorHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '16px',
  },
  editorContent: {
    fontFamily: tokens.fontFamilyMonospace,
    fontSize: tokens.fontSizeBase300,
    minHeight: '400px',
    ...shorthands.padding('12px'),
    backgroundColor: tokens.colorNeutralBackground2,
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
  },
  reloadSection: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('16px'),
  },
  reloadCard: {
    ...shorthands.padding('16px'),
  },
  reloadHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  reloadInfo: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('8px'),
  },
  reloadMetrics: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))',
    ...shorthands.gap('12px'),
    marginTop: '12px',
  },
  metric: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('4px'),
  },
  metricLabel: {
    fontSize: tokens.fontSizeBase200,
    color: tokens.colorNeutralForeground3,
  },
  metricValue: {
    fontSize: tokens.fontSizeBase400,
    fontWeight: tokens.fontWeightSemibold,
  },
  versionCard: {
    ...shorthands.padding('16px'),
    marginBottom: '12px',
  },
  versionHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  versionInfo: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('4px'),
  },
  versionId: {
    fontWeight: tokens.fontWeightSemibold,
    fontFamily: tokens.fontFamilyMonospace,
  },
  versionMeta: {
    display: 'flex',
    ...shorthands.gap('12px'),
    fontSize: tokens.fontSizeBase200,
    color: tokens.colorNeutralForeground3,
  },
  versionActions: {
    display: 'flex',
    ...shorthands.gap('8px'),
  },
  diffContainer: {
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke1),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    ...shorthands.padding('12px'),
    backgroundColor: tokens.colorNeutralBackground2,
    fontFamily: tokens.fontFamilyMonospace,
    fontSize: tokens.fontSizeBase200,
    maxHeight: '400px',
    overflowY: 'auto',
  },
  diffLine: {
    ...shorthands.padding('2px', '4px'),
    whiteSpace: 'pre',
  },
  diffAdded: {
    backgroundColor: '#e6ffed',
    color: '#24292e',
  },
  diffRemoved: {
    backgroundColor: '#ffeef0',
    color: '#24292e',
  },
  overrideCard: {
    ...shorthands.padding('16px'),
    marginBottom: '12px',
  },
  overrideHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  overrideInfo: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('8px'),
  },
  overrideTenant: {
    fontWeight: tokens.fontWeightSemibold,
  },
  overrideDetails: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))',
    ...shorthands.gap('12px'),
    marginTop: '8px',
  },
  flagCard: {
    ...shorthands.padding('16px'),
    marginBottom: '12px',
  },
  flagHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '12px',
  },
  flagInfo: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('4px'),
  },
  flagName: {
    fontWeight: tokens.fontWeightSemibold,
  },
  flagDescription: {
    fontSize: tokens.fontSizeBase200,
    color: tokens.colorNeutralForeground3,
  },
  flagMetrics: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(120px, 1fr))',
    ...shorthands.gap('12px'),
    marginTop: '12px',
  },
  formField: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('8px'),
    marginBottom: '16px',
  },
});

interface ConfigVariable {
  key: string;
  value: string;
  type: 'string' | 'number' | 'boolean' | 'json';
  sensitive: boolean;
  description: string;
  lastModified: string;
  modifiedBy: string;
}

interface ConfigVersion {
  id: string;
  timestamp: string;
  author: string;
  message: string;
  changeCount: number;
  type: 'manual' | 'auto' | 'rollback';
}

interface TenantOverride {
  id: string;
  tenantId: string;
  tenantName: string;
  overrides: Record<string, string>;
  createdAt: string;
  updatedAt: string;
  active: boolean;
}

interface FeatureFlag {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  type: 'boolean' | 'percentage' | 'targeting';
  percentage?: number;
  targetTenants?: string[];
  createdAt: string;
  updatedAt: string;
}

interface ServiceStatus {
  name: string;
  status: 'running' | 'stopped' | 'error';
  lastReload: string;
  configVersion: string;
  uptime: string;
}

const ConfigurationManager: React.FC = () => {
  const styles = useStyles();
  const [selectedTab, setSelectedTab] = useState<string>('variables');
  const [loading, setLoading] = useState(false);

  // Variables tab state
  const [variables, setVariables] = useState<ConfigVariable[]>([]);
  const [variableDialogOpen, setVariableDialogOpen] = useState(false);
  const [editingVariable, setEditingVariable] = useState<ConfigVariable | null>(null);
  const [variableForm, setVariableForm] = useState({
    key: '',
    value: '',
    type: 'string' as 'string' | 'number' | 'boolean' | 'json',
    sensitive: false,
    description: '',
  });

  // Editor tab state
  const [configFormat, setConfigFormat] = useState<'yaml' | 'json'>('yaml');
  const [configContent, setConfigContent] = useState('');
  const [configModified, setConfigModified] = useState(false);

  // Reload tab state
  const [services, setServices] = useState<ServiceStatus[]>([]);
  const [reloading, setReloading] = useState(false);

  // Version history tab state
  const [versions, setVersions] = useState<ConfigVersion[]>([]);
  const [selectedVersion, setSelectedVersion] = useState<ConfigVersion | null>(null);
  const [diffContent, setDiffContent] = useState<string>('');

  // Tenant overrides tab state
  const [overrides, setOverrides] = useState<TenantOverride[]>([]);
  const [overrideDialogOpen, setOverrideDialogOpen] = useState(false);
  const [overrideForm, setOverrideForm] = useState({
    tenantId: '',
    tenantName: '',
    overrides: {} as Record<string, string>,
  });

  // Feature flags tab state
  const [flags, setFlags] = useState<FeatureFlag[]>([]);
  const [flagDialogOpen, setFlagDialogOpen] = useState(false);
  const [flagForm, setFlagForm] = useState({
    name: '',
    description: '',
    enabled: false,
    type: 'boolean' as 'boolean' | 'percentage' | 'targeting',
    percentage: 0,
    targetTenants: [] as string[],
  });

  useEffect(() => {
    fetchVariables();
    fetchConfigContent();
    fetchServices();
    fetchVersions();
    fetchOverrides();
    fetchFlags();
  }, []);

  const fetchVariables = async () => {
    try {
      const response = await fetch(addTenantParam('/api/admin/config/variables'));
      if (!response.ok || !response.headers.get('content-type')?.includes('application/json') {
        setVariables([]);
        return;
      }
      const data = await response.json();
      setVariables(data.variables || []);
    } catch (error) {
      if (!(error instanceof SyntaxError) {
        console.error('Failed to fetch variables:', error);
      }
    }
  };

  const fetchConfigContent = async () => {
    try {
      const response = await fetch(addTenantParam(`/api/admin/config/${configFormat}`));
      if (!response.ok || !response.headers.get('content-type')?.includes('application/json') {
        setConfigContent('');
        return;
      }
      const data = await response.json();
      setConfigContent(data.content || '');
    } catch (error) {
      if (!(error instanceof SyntaxError) {
        console.error('Failed to fetch config content:', error);
      }
    }
  };

  const fetchServices = async () => {
    try {
      const response = await fetch(addTenantParam('/api/admin/config/services'));
      if (!response.ok || !response.headers.get('content-type')?.includes('application/json') {
        setServices([]);
        return;
      }
      const data = await response.json();
      setServices(data.services || []);
    } catch (error) {
      if (!(error instanceof SyntaxError) {
        console.error('Failed to fetch services:', error);
      }
    }
  };

  const fetchVersions = async () => {
    try {
      const response = await fetch(addTenantParam('/api/admin/config/versions'));
      if (!response.ok || !response.headers.get('content-type')?.includes('application/json') {
        setVersions([]);
        return;
      }
      const data = await response.json();
      setVersions(data.versions || []);
    } catch (error) {
      if (!(error instanceof SyntaxError) {
        console.error('Failed to fetch versions:', error);
      }
    }
  };

  const fetchOverrides = async () => {
    try {
      const response = await fetch(addTenantParam('/api/admin/config/tenant-overrides'));
      if (!response.ok || !response.headers.get('content-type')?.includes('application/json') {
        setOverrides([]);
        return;
      }
      const data = await response.json();
      setOverrides(data.overrides || []);
    } catch (error) {
      if (!(error instanceof SyntaxError) {
        console.error('Failed to fetch overrides:', error);
      }
    }
  };

  const fetchFlags = async () => {
    try {
      const response = await fetch(addTenantParam('/api/admin/config/feature-flags'));
      if (!response.ok || !response.headers.get('content-type')?.includes('application/json') {
        setFlags([]);
        return;
      }
      const data = await response.json();
      setFlags(data.flags || []);
    } catch (error) {
      if (!(error instanceof SyntaxError) {
        console.error('Failed to fetch flags:', error);
      }
    }
  };

  const handleCreateVariable = async () => {
    try {
      setLoading(true);
      const response = await fetch(addTenantParam('/api/admin/config/variables'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(variableForm),
      });
      if (response.ok) {
        setVariableDialogOpen(false);
        setVariableForm({
          key: '',
          value: '',
          type: 'string',
          sensitive: false,
          description: '',
        });
        fetchVariables();
      }
    } catch (error) {
      console.error('Failed to create variable:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleEditVariable = (variable: ConfigVariable) => {
    setEditingVariable(variable);
    setVariableForm({
      key: variable.key,
      value: variable.value,
      type: variable.type,
      sensitive: variable.sensitive,
      description: variable.description,
    });
    setVariableDialogOpen(true);
  };

  const handleUpdateVariable = async () => {
    try {
      setLoading(true);
      const response = await fetch(addTenantParam(`/api/admin/config/variables/${editingVariable?.key}`), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(variableForm),
      });
      if (response.ok) {
        setVariableDialogOpen(false);
        setEditingVariable(null);
        setVariableForm({
          key: '',
          value: '',
          type: 'string',
          sensitive: false,
          description: '',
        });
        fetchVariables();
      }
    } catch (error) {
      console.error('Failed to update variable:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteVariable = async (key: string) => {
    if (!confirm(`Are you sure you want to delete variable "${key}"?`) return;
    try {
      const response = await fetch(addTenantParam(`/api/admin/config/variables/${key}`), {
        method: 'DELETE',
      });
      if (response.ok) {
        fetchVariables();
      }
    } catch (error) {
      console.error('Failed to delete variable:', error);
    }
  };

  const handleSaveConfig = async () => {
    try {
      setLoading(true);
      const response = await fetch(addTenantParam(`/api/admin/config/${configFormat}`), {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: configContent }),
      });
      if (response.ok) {
        setConfigModified(false);
        fetchVersions();
      }
    } catch (error) {
      console.error('Failed to save config:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleReloadService = async (serviceName: string) => {
    try {
      setReloading(true);
      const response = await fetch(addTenantParam('/api/admin/config/reload'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ service: serviceName }),
      });
      if (response.ok) {
        fetchServices();
      }
    } catch (error) {
      console.error('Failed to reload service:', error);
    } finally {
      setReloading(false);
    }
  };

  const handleViewDiff = async (versionId: string) => {
    try {
      const response = await fetch(addTenantParam(`/api/admin/config/versions/${versionId}/diff`));
      const data = await response.json();
      setDiffContent(data.diff || '');
      setSelectedVersion(versions.find(v => v.id === versionId) || null);
    } catch (error) {
      console.error('Failed to fetch diff:', error);
    }
  };

  const handleRollback = async (versionId: string) => {
    if (!confirm('Are you sure you want to rollback to this version?') return;
    try {
      setLoading(true);
      const response = await fetch(addTenantParam(`/api/admin/config/versions/${versionId}/rollback`), {
        method: 'POST',
      });
      if (response.ok) {
        fetchVersions();
        fetchConfigContent();
        fetchVariables();
      }
    } catch (error) {
      console.error('Failed to rollback:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateOverride = async () => {
    try {
      setLoading(true);
      const response = await fetch(addTenantParam('/api/admin/config/tenant-overrides'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(overrideForm),
      });
      if (response.ok) {
        setOverrideDialogOpen(false);
        setOverrideForm({
          tenantId: '',
          tenantName: '',
          overrides: {},
        });
        fetchOverrides();
      }
    } catch (error) {
      console.error('Failed to create override:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleOverride = async (id: string, active: boolean) => {
    try {
      const response = await fetch(addTenantParam(`/api/admin/config/tenant-overrides/${id}/toggle`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ active }),
      });
      if (response.ok) {
        fetchOverrides();
      }
    } catch (error) {
      console.error('Failed to toggle override:', error);
    }
  };

  const handleDeleteOverride = async (id: string) => {
    if (!confirm('Are you sure you want to delete this tenant override?') return;
    try {
      const response = await fetch(addTenantParam(`/api/admin/config/tenant-overrides/${id}`), {
        method: 'DELETE',
      });
      if (response.ok) {
        fetchOverrides();
      }
    } catch (error) {
      console.error('Failed to delete override:', error);
    }
  };

  const handleCreateFlag = async () => {
    try {
      setLoading(true);
      const response = await fetch(addTenantParam('/api/admin/config/feature-flags'), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(flagForm),
      });
      if (response.ok) {
        setFlagDialogOpen(false);
        setFlagForm({
          name: '',
          description: '',
          enabled: false,
          type: 'boolean',
          percentage: 0,
          targetTenants: [],
        });
        fetchFlags();
      }
    } catch (error) {
      console.error('Failed to create flag:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleFlag = async (id: string, enabled: boolean) => {
    try {
      const response = await fetch(addTenantParam(`/api/admin/config/feature-flags/${id}/toggle`), {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ enabled }),
      });
      if (response.ok) {
        fetchFlags();
      }
    } catch (error) {
      console.error('Failed to toggle flag:', error);
    }
  };

  const handleDeleteFlag = async (id: string) => {
    if (!confirm('Are you sure you want to delete this feature flag?') return;
    try {
      const response = await fetch(addTenantParam(`/api/admin/config/feature-flags/${id}`), {
        method: 'DELETE',
      });
      if (response.ok) {
        fetchFlags();
      }
    } catch (error) {
      console.error('Failed to delete flag:', error);
    }
  };

  const getStatusBadge = (status: string) => {
    const colors: Record<string, 'success' | 'danger' | 'warning'> = {
      running: 'success',
      stopped: 'danger',
      error: 'danger',
    };
    return <Badge color={colors[status] || 'warning'}>{status.toUpperCase()}</Badge>;
  };

  const getTypeBadge = (type: string) => {
    const colors: Record<string, 'success' | 'danger' | 'warning' | 'informative'> = {
      manual: 'informative',
      auto: 'success',
      rollback: 'warning',
    };
    return <Badge color={colors[type] || 'informative'}>{type}</Badge>;
  };

  const renderDiffLine = (line: string, index: number) => {
    if (line.startsWith('+') {
      return <div key={index} className={`${styles.diffLine} ${styles.diffAdded}`}>{line}</div>;
    } else if (line.startsWith('-') {
      return <div key={index} className={`${styles.diffLine} ${styles.diffRemoved}`}>{line}</div>;
    } else {
      return <div key={index} className={styles.diffLine}>{line}</div>;
    }
  };

  // Calculate overview metrics
  const totalVariables = variables.length;
  const sensitiveVariables = variables.filter(v => v.sensitive).length;
  const activeOverrides = overrides.filter(o => o.active).length;
  const enabledFlags = flags.filter(f => f.enabled).length;

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <Title3>Configuration Management</Title3>
      </div>

      {/* Overview Cards */}
      <div className={styles.overviewCards}>
        <Card className={styles.card}>
          <div className={styles.cardHeader}>
            <Settings24Regular />
            <Text className={styles.cardTitle}>Total Variables</Text>
          </div>
          <Text className={styles.cardValue}>{totalVariables}</Text>
          <Text size={200}>{sensitiveVariables} sensitive</Text>
        </Card>
        <Card className={styles.card}>
          <div className={styles.cardHeader}>
            <Code24Regular />
            <Text className={styles.cardTitle}>Config Files</Text>
          </div>
          <Text className={styles.cardValue}>{versions.length}</Text>
          <Text size={200}>Versions tracked</Text>
        </Card>
        <Card className={styles.card}>
          <div className={styles.cardHeader}>
            <Database24Regular />
            <Text className={styles.cardTitle}>Tenant Overrides</Text>
          </div>
          <Text className={styles.cardValue}>{activeOverrides}</Text>
          <Text size={200}>{overrides.length} total</Text>
        </Card>
        <Card className={styles.card}>
          <div className={styles.cardHeader}>
            <Flag24Regular />
            <Text className={styles.cardTitle}>Feature Flags</Text>
          </div>
          <Text className={styles.cardValue}>{enabledFlags}</Text>
          <Text size={200}>{flags.length} total</Text>
        </Card>
      </div>

      {/* Tabs */}
      <div className={styles.tabsContainer}>
        <TabList selectedValue={selectedTab} onTabSelect={(_, data) => setSelectedTab(data.value as string)}>
          <Tab value="variables">Environment Variables</Tab>
          <Tab value="editor">YAML/JSON Editor</Tab>
          <Tab value="reload">Hot Reload</Tab>
          <Tab value="versions">Version History</Tab>
          <Tab value="overrides">Tenant Overrides</Tab>
          <Tab value="flags">Feature Flags</Tab>
        </TabList>

        {/* Environment Variables Tab */}
        {selectedTab === 'variables' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '16px' }}>
              <Text weight="semibold">Environment Variables</Text>
              <Button
                appearance="primary"
                icon={<Add24Regular />}
                onClick={() => {
                  setEditingVariable(null);
                  setVariableForm({
                    key: '',
                    value: '',
                    type: 'string',
                    sensitive: false,
                    description: '',
                  });
                  setVariableDialogOpen(true);
                }}
              >
                Add Variable
              </Button>
            </div>

            <div className={styles.variablesGrid}>
              {variables.map((variable) => (
                <Card key={variable.key} className={styles.variableCard}>
                  <div className={styles.variableInfo}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <Text className={styles.variableName}>{variable.key}</Text>
                      <Badge color={variable.sensitive ? 'danger' : 'informative'}>{variable.type}</Badge>
                      {variable.sensitive && <Badge color="warning">SENSITIVE</Badge>}
                    </div>
                    <Text className={styles.variableValue}>
                      {variable.sensitive ? '••••••••' : variable.value}
                    </Text>
                    <Text size={200}>{variable.description}</Text>
                    <Text size={100} style={{ color: tokens.colorNeutralForeground4 }}>
                      Modified by {variable.modifiedBy} on {variable.lastModified}
                    </Text>
                  </div>
                  <div className={styles.variableActions}>
                    <Button
                      icon={<Edit24Regular />}
                      size="small"
                      onClick={() => handleEditVariable(variable)}
                    />
                    <Button
                      icon={<Delete24Regular />}
                      size="small"
                      onClick={() => handleDeleteVariable(variable.key)}
                    />
                  </div>
                </Card>
              ))}
            </div>

            <Dialog open={variableDialogOpen} onOpenChange={(_, data) => setVariableDialogOpen(data.open)}>
              <DialogSurface>
                <DialogBody>
                  <DialogTitle>{editingVariable ? 'Edit Variable' : 'Add Variable'}</DialogTitle>
                  <DialogContent>
                    <div className={styles.formField}>
                      <Label required>Key</Label>
                      <Input
                        value={variableForm.key}
                        onChange={(e) => setVariableForm({ ...variableForm, key: e.target.value })}
                        disabled={editingVariable !== null}
                      />
                    </div>
                    <div className={styles.formField}>
                      <Label required>Value</Label>
                      <Input
                        value={variableForm.value}
                        onChange={(e) => setVariableForm({ ...variableForm, value: e.target.value })}
                        type={variableForm.sensitive ? 'password' : 'text'}
                      />
                    </div>
                    <div className={styles.formField}>
                      <Label>Type</Label>
                      <Dropdown
                        value={variableForm.type}
                        selectedOptions={[variableForm.type]}
                        onOptionSelect={(_, data) =>
                          setVariableForm({ ...variableForm, type: data.optionValue as any })
                        }
                      >
                        <Option value="string">String</Option>
                        <Option value="number">Number</Option>
                        <Option value="boolean">Boolean</Option>
                        <Option value="json">JSON</Option>
                      </Dropdown>
                    </div>
                    <div className={styles.formField}>
                      <Label>Description</Label>
                      <Textarea
                        value={variableForm.description}
                        onChange={(e) => setVariableForm({ ...variableForm, description: e.target.value })}
                      />
                    </div>
                    <div className={styles.formField}>
                      <Switch
                        checked={variableForm.sensitive}
                        onChange={(e) => setVariableForm({ ...variableForm, sensitive: e.currentTarget.checked })}
                        label="Sensitive (encrypted storage)"
                      />
                    </div>
                  </DialogContent>
                  <DialogActions>
                    <Button appearance="secondary" onClick={() => setVariableDialogOpen(false)}>
                      Cancel
                    </Button>
                    <Button
                      appearance="primary"
                      onClick={editingVariable ? handleUpdateVariable : handleCreateVariable}
                      disabled={loading || !variableForm.key || !variableForm.value}
                    >
                      {loading ? <Spinner size="tiny" /> : editingVariable ? 'Update' : 'Create'}
                    </Button>
                  </DialogActions>
                </DialogBody>
              </DialogSurface>
            </Dialog>
          </div>
        )}

        {/* YAML/JSON Editor Tab */}
        {selectedTab === 'editor' && (
          <div className={styles.editorContainer}>
            <div className={styles.editorHeader}>
              <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                <Text weight="semibold">Configuration Editor</Text>
                <Dropdown
                  value={configFormat}
                  selectedOptions={[configFormat]}
                  onOptionSelect={(_, data) => {
                    setConfigFormat(data.optionValue as 'yaml' | 'json');
                    fetchConfigContent();
                  }}
                >
                  <Option value="yaml">YAML</Option>
                  <Option value="json">JSON</Option>
                </Dropdown>
                {configModified && <Badge color="warning">Modified</Badge>}
              </div>
              <div style={{ display: 'flex', gap: '8px' }}>
                <Button
                  appearance="secondary"
                  onClick={() => {
                    fetchConfigContent();
                    setConfigModified(false);
                  }}
                >
                  Discard Changes
                </Button>
                <Button
                  appearance="primary"
                  onClick={handleSaveConfig}
                  disabled={!configModified || loading}
                >
                  {loading ? <Spinner size="tiny" /> : 'Save Configuration'}
                </Button>
              </div>
            </div>
            <Textarea
              className={styles.editorContent}
              value={configContent}
              onChange={(e) => {
                setConfigContent(e.target.value);
                setConfigModified(true);
              }}
              resize="vertical"
            />
          </div>
        )}

        {/* Hot Reload Tab */}
        {selectedTab === 'reload' && (
          <div className={styles.reloadSection}>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '16px' }}>
              <Text weight="semibold">Service Status & Hot Reload</Text>
              <Button
                appearance="primary"
                icon={<ArrowSync24Regular />}
                onClick={() => handleReloadService('all')}
                disabled={reloading}
              >
                {reloading ? <Spinner size="tiny" /> : 'Reload All Services'}
              </Button>
            </div>

            {services.map((service) => (
              <Card key={service.name} className={styles.reloadCard}>
                <div className={styles.reloadHeader}>
                  <div className={styles.reloadInfo}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <Text weight="semibold">{service.name}</Text>
                      {getStatusBadge(service.status)}
                    </div>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                      Config Version: {service.configVersion}
                    </Text>
                  </div>
                  <Button
                    icon={<ArrowSync24Regular />}
                    onClick={() => handleReloadService(service.name)}
                    disabled={reloading || service.status === 'stopped'}
                  >
                    Reload
                  </Button>
                </div>
                <div className={styles.reloadMetrics}>
                  <div className={styles.metric}>
                    <Text className={styles.metricLabel}>Last Reload</Text>
                    <Text className={styles.metricValue}>{service.lastReload}</Text>
                  </div>
                  <div className={styles.metric}>
                    <Text className={styles.metricLabel}>Uptime</Text>
                    <Text className={styles.metricValue}>{service.uptime}</Text>
                  </div>
                  <div className={styles.metric}>
                    <Text className={styles.metricLabel}>Status</Text>
                    <Text className={styles.metricValue}>
                      {service.status === 'running' ? 'Healthy' : 'Unhealthy'}
                    </Text>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}

        {/* Version History Tab */}
        {selectedTab === 'versions' && (
          <div>
            <Text weight="semibold" style={{ marginBottom: '16px', display: 'block' }}>
              Configuration Version History
            </Text>

            {versions.map((version) => (
              <Card key={version.id} className={styles.versionCard}>
                <div className={styles.versionHeader}>
                  <div className={styles.versionInfo}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <Text className={styles.versionId}>{version.id}</Text>
                      {getTypeBadge(version.type)}
                    </div>
                    <div className={styles.versionMeta}>
                      <Text>{version.author}</Text>
                      <Text>•</Text>
                      <Text>{version.timestamp}</Text>
                      <Text>•</Text>
                      <Text>{version.changeCount} changes</Text>
                    </div>
                    <Text size={200}>{version.message}</Text>
                  </div>
                  <div className={styles.versionActions}>
                    <Button
                      icon={<History24Regular />}
                      size="small"
                      onClick={() => handleViewDiff(version.id)}
                    >
                      View Diff
                    </Button>
                    <Button
                      icon={<ArrowSync24Regular />}
                      size="small"
                      onClick={() => handleRollback(version.id)}
                      disabled={loading}
                    >
                      Rollback
                    </Button>
                  </div>
                </div>

                {selectedVersion?.id === version.id && diffContent && (
                  <div style={{ marginTop: '16px' }}>
                    <Text weight="semibold" size={200} style={{ marginBottom: '8px', display: 'block' }}>
                      Diff Preview
                    </Text>
                    <div className={styles.diffContainer}>
                      {diffContent.split('\n').map((line, index) => renderDiffLine(line, index))}
                    </div>
                  </div>
                )}
              </Card>
            ))}
          </div>
        )}

        {/* Tenant Overrides Tab */}
        {selectedTab === 'overrides' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '16px' }}>
              <Text weight="semibold">Tenant-Specific Configuration Overrides</Text>
              <Button
                appearance="primary"
                icon={<Add24Regular />}
                onClick={() => setOverrideDialogOpen(true)}
              >
                Add Override
              </Button>
            </div>

            {overrides.map((override) => (
              <Card key={override.id} className={styles.overrideCard}>
                <div className={styles.overrideHeader}>
                  <div className={styles.overrideInfo}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <Text className={styles.overrideTenant}>{override.tenantName}</Text>
                      {override.active ? (
                        <Badge color="success" icon={<CheckmarkCircle24Regular />}>ACTIVE</Badge>
                      ) : (
                        <Badge color="danger" icon={<DismissCircle24Regular />}>INACTIVE</Badge>
                      )}
                    </div>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                      Tenant ID: {override.tenantId}
                    </Text>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                      Created: {override.createdAt} • Updated: {override.updatedAt}
                    </Text>
                  </div>
                  <div style={{ display: 'flex', gap: '8px' }}>
                    <Switch
                      checked={override.active}
                      onChange={(e) => handleToggleOverride(override.id, e.currentTarget.checked)}
                    />
                    <Button
                      icon={<Delete24Regular />}
                      size="small"
                      onClick={() => handleDeleteOverride(override.id)}
                    />
                  </div>
                </div>
                <div className={styles.overrideDetails}>
                  {Object.entries(override.overrides).map(([key, value]) => (
                    <div key={key} className={styles.metric}>
                      <Text className={styles.metricLabel}>{key}</Text>
                      <Text className={styles.metricValue} style={{ fontFamily: tokens.fontFamilyMonospace }}>
                        {value}
                      </Text>
                    </div>
                  ))}
                </div>
              </Card>
            ))}

            <Dialog open={overrideDialogOpen} onOpenChange={(_, data) => setOverrideDialogOpen(data.open)}>
              <DialogSurface>
                <DialogBody>
                  <DialogTitle>Add Tenant Override</DialogTitle>
                  <DialogContent>
                    <div className={styles.formField}>
                      <Label required>Tenant ID</Label>
                      <Input
                        value={overrideForm.tenantId}
                        onChange={(e) => setOverrideForm({ ...overrideForm, tenantId: e.target.value })}
                      />
                    </div>
                    <div className={styles.formField}>
                      <Label required>Tenant Name</Label>
                      <Input
                        value={overrideForm.tenantName}
                        onChange={(e) => setOverrideForm({ ...overrideForm, tenantName: e.target.value })}
                      />
                    </div>
                    <div className={styles.formField}>
                      <Label>Override Configuration (JSON)</Label>
                      <Textarea
                        placeholder='{"key1": "value1", "key2": "value2"}'
                        onChange={(e) => {
                          try {
                            const parsed = JSON.parse(e.target.value);
                            setOverrideForm({ ...overrideForm, overrides: parsed });
                          } catch (_err) {
                            // Invalid JSON, ignore
                          }
                        }}
                      />
                    </div>
                  </DialogContent>
                  <DialogActions>
                    <Button appearance="secondary" onClick={() => setOverrideDialogOpen(false)}>
                      Cancel
                    </Button>
                    <Button
                      appearance="primary"
                      onClick={handleCreateOverride}
                      disabled={loading || !overrideForm.tenantId || !overrideForm.tenantName}
                    >
                      {loading ? <Spinner size="tiny" /> : 'Create'}
                    </Button>
                  </DialogActions>
                </DialogBody>
              </DialogSurface>
            </Dialog>
          </div>
        )}

        {/* Feature Flags Tab */}
        {selectedTab === 'flags' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: '16px' }}>
              <Text weight="semibold">Feature Flags</Text>
              <Button
                appearance="primary"
                icon={<Add24Regular />}
                onClick={() => setFlagDialogOpen(true)}
              >
                Add Flag
              </Button>
            </div>

            {flags.map((flag) => (
              <Card key={flag.id} className={styles.flagCard}>
                <div className={styles.flagHeader}>
                  <div className={styles.flagInfo}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                      <Text className={styles.flagName}>{flag.name}</Text>
                      <Badge color={flag.type === 'boolean' ? 'informative' : flag.type === 'percentage' ? 'warning' : 'success'}>
                        {flag.type}
                      </Badge>
                      {flag.enabled ? (
                        <Badge color="success" icon={<CheckmarkCircle24Regular />}>ENABLED</Badge>
                      ) : (
                        <Badge color="danger" icon={<DismissCircle24Regular />}>DISABLED</Badge>
                      )}
                    </div>
                    <Text className={styles.flagDescription}>{flag.description}</Text>
                    <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>
                      Created: {flag.createdAt} • Updated: {flag.updatedAt}
                    </Text>
                  </div>
                  <div style={{ display: 'flex', gap: '8px' }}>
                    <Switch
                      checked={flag.enabled}
                      onChange={(e) => handleToggleFlag(flag.id, e.currentTarget.checked)}
                    />
                    <Button
                      icon={<Delete24Regular />}
                      size="small"
                      onClick={() => handleDeleteFlag(flag.id)}
                    />
                  </div>
                </div>
                <div className={styles.flagMetrics}>
                  {flag.type === 'percentage' && (
                    <div className={styles.metric}>
                      <Text className={styles.metricLabel}>Rollout</Text>
                      <ProgressBar value={(flag.percentage || 0) / 100} max={1} />
                      <Text className={styles.metricValue}>{flag.percentage || 0}%</Text>
                    </div>
                  )}
                  {flag.type === 'targeting' && flag.targetTenants && (
                    <div className={styles.metric}>
                      <Text className={styles.metricLabel}>Target Tenants</Text>
                      <Text className={styles.metricValue}>{flag.targetTenants.length}</Text>
                    </div>
                  )}
                </div>
              </Card>
            ))}

            <Dialog open={flagDialogOpen} onOpenChange={(_, data) => setFlagDialogOpen(data.open)}>
              <DialogSurface>
                <DialogBody>
                  <DialogTitle>Add Feature Flag</DialogTitle>
                  <DialogContent>
                    <div className={styles.formField}>
                      <Label required>Name</Label>
                      <Input
                        value={flagForm.name}
                        onChange={(e) => setFlagForm({ ...flagForm, name: e.target.value })}
                        placeholder="enable-new-feature"
                      />
                    </div>
                    <div className={styles.formField}>
                      <Label>Description</Label>
                      <Textarea
                        value={flagForm.description}
                        onChange={(e) => setFlagForm({ ...flagForm, description: e.target.value })}
                      />
                    </div>
                    <div className={styles.formField}>
                      <Label>Type</Label>
                      <Dropdown
                        value={flagForm.type}
                        selectedOptions={[flagForm.type]}
                        onOptionSelect={(_, data) =>
                          setFlagForm({ ...flagForm, type: data.optionValue as any })
                        }
                      >
                        <Option value="boolean">Boolean (On/Off)</Option>
                        <Option value="percentage">Percentage Rollout</Option>
                        <Option value="targeting">Tenant Targeting</Option>
                      </Dropdown>
                    </div>
                    {flagForm.type === 'percentage' && (
                      <div className={styles.formField}>
                        <Label>Rollout Percentage</Label>
                        <Input
                          type="number"
                          value={flagForm.percentage.toString()}
                          onChange={(e) => setFlagForm({ ...flagForm, percentage: parseInt(e.target.value) || 0 })}
                          min="0"
                          max="100"
                        />
                      </div>
                    )}
                    {flagForm.type === 'targeting' && (
                      <div className={styles.formField}>
                        <Label>Target Tenant IDs (comma-separated)</Label>
                        <Input
                          placeholder="tenant-1,tenant-2,tenant-3"
                          onChange={(e) =>
                            setFlagForm({ ...flagForm, targetTenants: e.target.value.split(',').map(t => t.trim() })
                          }
                        />
                      </div>
                    )}
                    <div className={styles.formField}>
                      <Switch
                        checked={flagForm.enabled}
                        onChange={(e) => setFlagForm({ ...flagForm, enabled: e.currentTarget.checked })}
                        label="Enable immediately"
                      />
                    </div>
                  </DialogContent>
                  <DialogActions>
                    <Button appearance="secondary" onClick={() => setFlagDialogOpen(false)}>
                      Cancel
                    </Button>
                    <Button
                      appearance="primary"
                      onClick={handleCreateFlag}
                      disabled={loading || !flagForm.name}
                    >
                      {loading ? <Spinner size="tiny" /> : 'Create'}
                    </Button>
                  </DialogActions>
                </DialogBody>
              </DialogSurface>
            </Dialog>
          </div>
        )}
      </div>
    </div>
  );
};

export default ConfigurationManager;
