import { useState, useEffect } from 'react';
import {
  makeStyles,
  shorthands,
  tokens,
  Button,
  Card,
  Caption1,
  Text,
  Title2,
  Title3,
  Subtitle2,
  Input,
  Label,
  Field,
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
  DialogTrigger,
  TabList,
  Tab,
  TabValue,
  Badge,
  Spinner,
} from '@fluentui/react-components';
import {
  Server24Regular,
  PlugConnected24Regular,
  Wrench24Regular,
  Add24Regular,
  Delete24Regular,
  ArrowRepeatAll24Regular,
  DocumentText24Regular,
  Play24Regular,
  Dismiss24Regular,
  Desktop24Regular,
} from '@fluentui/react-icons';
import { toast } from 'sonner';

// --- Types ---
interface MCPServer {
  id: string;
  name: string;
  description?: string;
  transport_type: string;
  command?: string;
  args?: string[];
  url?: string;
  status: 'connected' | 'disconnected';
}

interface MCPResource {
  uri: string;
  name: string;
  description?: string;
  mime_type?: string;
}

interface MCPTool {
  name: string;
  description?: string;
  input_schema?: Record<string, unknown>;
}

interface ResourceContent {
  uri: string;
  mime_type?: string;
  text?: string;
}

interface ToolResultContent {
  type: string;
  text?: string;
}

// --- Styles ---
const useStyles = makeStyles({
  container: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('24px'),
    paddingBottom: '40px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  statsGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
    ...shorthands.gap('16px'),
  },
  statCard: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.padding('20px'),
    height: '100%',
  },
  statIcon: {
    color: tokens.colorBrandForeground1,
    marginBottom: '12px',
  },
  mainContent: {
    display: 'grid',
    gridTemplateColumns: '350px 1fr',
    ...shorthands.gap('24px'),
    alignItems: 'start',
    '@media (max-width: 1000px)': {
      gridTemplateColumns: '1fr',
    },
  },
  serverList: {
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('12px'),
    maxHeight: 'calc(100vh - 250px)',
    overflowY: 'auto',
  },
  serverItem: {
    cursor: 'pointer',
    ...shorthands.padding('16px'),
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke1),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    backgroundColor: tokens.colorNeutralBackground1,
    ':hover': {
      backgroundColor: tokens.colorNeutralBackground1Hover,
      // @ts-ignore
      borderColor: tokens.colorNeutralStroke1Hover,
    },
  },
  serverItemSelected: {
    backgroundColor: tokens.colorBrandBackground2,
    // @ts-ignore
    borderColor: tokens.colorBrandStroke1,
    ':hover': {
      backgroundColor: tokens.colorBrandBackground2,
    },
  },
  serverItemHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'flex-start',
    marginBottom: '8px',
  },
  statusBadge: {
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
    fontSize: '12px',
  },
  detailsCard: {
    ...shorthands.padding('24px'),
    minHeight: '400px',
  },
  detailsHeader: {
    display: 'flex',
    flexDirection: 'column',
    marginBottom: '20px',
  },
  tabContent: {
    marginTop: '20px',
    display: 'flex',
    flexDirection: 'column',
    ...shorthands.gap('16px'),
  },
  resourceItem: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    ...shorthands.padding('12px'),
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke2),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
  },
  toolItem: {
    ...shorthands.padding('16px'),
    ...shorthands.border('1px', 'solid', tokens.colorNeutralStroke2),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    display: 'flex',
    flexDirection: 'column',
    gap: '8px',
  },
  codeBlock: {
    fontFamily: 'monospace',
    backgroundColor: tokens.colorNeutralBackground2,
    ...shorthands.padding('12px'),
    ...shorthands.borderRadius(tokens.borderRadiusMedium),
    whiteSpace: 'pre-wrap',
    fontSize: '12px',
    maxHeight: '300px',
    overflowY: 'auto',
  },
  formColumn: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  row: {
    display: 'flex',
    gap: '16px',
  },
  dialog: {
    maxWidth: '600px',
    width: '100%',
  }
});

export default function MCP() {
  const classes = useStyles();
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [selectedServer, setSelectedServer] = useState<string | null>(null);
  const [resources, setResources] = useState<MCPResource[]>([]);
  const [tools, setTools] = useState<MCPTool[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAddServer, setShowAddServer] = useState(false);
  const [resourceContent, setResourceContent] = useState<ResourceContent | null>(null);
  const [toolResult, setToolResult] = useState<{ content: ToolResultContent[] } | null>(null);
  const [selectedTab, setSelectedTab] = useState<TabValue>('info');
  const [toolLoading, setToolLoading] = useState(false);

  // Form states
  const [serverForm, setServerForm] = useState({
    id: '',
    name: '',
    description: '',
    transport_type: 'stdio',
    command: '',
    args: '',
    url: '',
    require_auth: false,
    allowed_scopes: ''
  });

  const [toolForm, setToolForm] = useState({
    name: '',
    arguments: '{}'
  });

  useEffect(() => {
    loadServers();
  }, []);

  useEffect(() => {
    if (selectedServer) {
      loadServerDetails(selectedServer);
      setResourceContent(null);
      setToolResult(null);
      setToolForm({ name: '', arguments: '{}' });
    }
  }, [selectedServer]);

  const loadServers = async () => {
    setLoading(true);
    try {
      const { apiClient } = await import('@/lib/api');
      const data = await apiClient.listMCPServers();
      setServers(data.servers || []);
      if (data.servers?.length > 0 && !selectedServer) {
        setSelectedServer(data.servers[0].id);
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      toast.error('Failed to load MCP servers: ' + errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const loadServerDetails = async (serverId: string) => {
    try {
      const { apiClient } = await import('@/lib/api');

      // Load resources
      const resourcesData = await apiClient.listMCPResources(serverId);
      setResources(resourcesData.resources || []);

      // Load tools
      const toolsData = await apiClient.listMCPTools(serverId);
      setTools(toolsData.tools || []);
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      toast.error('Failed to load server details: ' + errorMessage);
    }
  };

  const handleRegisterServer = async () => {
    setLoading(true);
    try {
      const { apiClient } = await import('@/lib/api');

      const serverData = {
        id: serverForm.id,
        name: serverForm.name,
        description: serverForm.description,
        transport_type: serverForm.transport_type,
        command: serverForm.transport_type === 'stdio' ? serverForm.command : undefined,
        args: serverForm.transport_type === 'stdio' && serverForm.args
          ? serverForm.args.split(',').map(a => a.trim())
          : [],
        url: serverForm.transport_type !== 'stdio' ? serverForm.url : undefined,
        require_auth: serverForm.require_auth,
        allowed_scopes: serverForm.allowed_scopes
          ? serverForm.allowed_scopes.split(',').map(s => s.trim())
          : []
      };

      await apiClient.registerMCPServer(serverData);
      toast.success('MCP server registered successfully');
      setShowAddServer(false);

      // Reset form
      setServerForm({
        id: '',
        name: '',
        description: '',
        transport_type: 'stdio',
        command: '',
        args: '',
        url: '',
        require_auth: false,
        allowed_scopes: ''
      });

      loadServers();
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      toast.error('Failed to register server: ' + errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleDisconnectServer = async (serverId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (!confirm('Are you sure you want to disconnect this server?')) return;

    setLoading(true);
    try {
      const { apiClient } = await import('@/lib/api');
      await apiClient.disconnectMCPServer(serverId);
      toast.success('Server disconnected successfully');

      if (selectedServer === serverId) {
        setSelectedServer(null);
        setResources([]);
        setTools([]);
      }
      loadServers();
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      toast.error('Failed to disconnect server: ' + errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleReadResource = async (uri: string) => {
    setLoading(true);
    setResourceContent(null);

    try {
      const { apiClient } = await import('@/lib/api');
      if (!selectedServer) return;

      const data = await apiClient.readMCPResource(selectedServer, uri);
      if (data.contents && data.contents.length > 0) {
        setResourceContent(data.contents[0]);
        toast.success('Resource loaded successfully');
      }
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      toast.error('Failed to read resource: ' + errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const handleCallTool = async () => {
    if (!selectedServer || !toolForm.name) return;

    setToolLoading(true);
    setToolResult(null);

    try {
      const { apiClient } = await import('@/lib/api');

      let args = {};
      try {
        args = JSON.parse(toolForm.arguments || '{}');
      } catch {
        toast.error('Invalid JSON in arguments');
        setToolLoading(false);
        return;
      }

      const data = await apiClient.callMCPTool(selectedServer, toolForm.name, args);
      setToolResult(data);
      toast.success('Tool executed successfully');
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      toast.error('Failed to call tool: ' + errorMessage);
    } finally {
      setToolLoading(false);
    }
  };

  const selectedServerData = servers.find(s => s.id === selectedServer);
  const connectedCount = servers.filter(s => s.status === 'connected').length;

  return (
    <div className={classes.container}>
      {/* Header */}
      <div className={classes.header}>
        <div>
          <Title2>Model Context Protocol</Title2>
          <Text block style={{ color: tokens.colorNeutralForeground2, marginTop: '4px' }}>
            Manage MCP servers and interact with AI model resources and tools
          </Text>
        </div>
        <Button
          appearance="primary"
          icon={<Add24Regular />}
          onClick={() => setShowAddServer(true)}
        >
          Register Server
        </Button>
      </div>

      {/* Stats Cards */}
      <div className={classes.statsGrid}>
        <Card className={classes.statCard}>
          <Server24Regular className={classes.statIcon} style={{ fontSize: '32px' }} />
          <Text size={500} weight="semibold">{servers.length}</Text>
          <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Total Servers</Text>
        </Card>
        <Card className={classes.statCard}>
          <PlugConnected24Regular className={classes.statIcon} style={{ fontSize: '32px', color: tokens.colorPaletteGreenForeground1 }} />
          <Text size={500} weight="semibold">{connectedCount}</Text>
          <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Connected Active</Text>
        </Card>
        <Card className={classes.statCard}>
          <Wrench24Regular className={classes.statIcon} style={{ fontSize: '32px', color: tokens.colorPalettePurpleForeground2 }} />
          <Text size={500} weight="semibold">{tools.length}</Text>
          <Text size={200} style={{ color: tokens.colorNeutralForeground3 }}>Available Tools (Current)</Text>
        </Card>
      </div>

      {/* Main Content */}
      <div className={classes.mainContent}>

        {/* Left Column: Server List */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '8px' }}>
            <Subtitle2>Servers</Subtitle2>
            <Button appearance="subtle" icon={<ArrowRepeatAll24Regular />} onClick={loadServers} title="Refresh list" />
          </div>

          <div className={classes.serverList}>
            {servers.length === 0 ? (
              <Card className={classes.serverItem}>
                <Text align="center" style={{ color: tokens.colorNeutralForeground3 }}>No servers registered</Text>
              </Card>
            ) : (
              servers.map(server => (
                <div
                  key={server.id}
                  className={`${classes.serverItem} ${selectedServer === server.id ? classes.serverItemSelected : ''}`}
                  onClick={() => setSelectedServer(server.id)}
                >
                  <div className={classes.serverItemHeader}>
                    <Text weight="semibold">{server.name}</Text>
                    <Button
                      appearance="transparent"
                      icon={<Delete24Regular />}
                      size="small"
                      style={{ color: tokens.colorPaletteRedForeground1 }}
                      onClick={(e) => handleDisconnectServer(server.id, e)}
                    />
                  </div>
                  <Text size={200} block style={{ marginBottom: '8px', color: tokens.colorNeutralForeground3 }}>{server.id}</Text>

                  <div className={classes.statusBadge}>
                    {server.status === 'connected' ? (
                      <Badge color="success" shape="rounded" appearance="tint">Connected</Badge>
                    ) : (
                      <Badge color="danger" shape="rounded" appearance="tint">Disconnected</Badge>
                    )}
                    <Text size={100} style={{ color: tokens.colorNeutralForeground3 }}>
                      {server.transport_type}
                    </Text>
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Right Column: Details */}
        <Card className={classes.detailsCard}>
          {selectedServerData ? (
            <>
              <div className={classes.detailsHeader}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '8px' }}>
                  <Title3>{selectedServerData.name}</Title3>
                  {selectedServerData.status === 'connected' ? (
                    <Badge color="success" shape="rounded" appearance="filled" icon={<PlugConnected24Regular />}>Connected</Badge>
                  ) : (
                    <Badge color="danger" shape="rounded" appearance="filled" icon={<Dismiss24Regular />}>Disconnected</Badge>
                  )}
                </div>
                <Text style={{ color: tokens.colorNeutralForeground2 }}>{selectedServerData.description || 'No description provided'}</Text>
              </div>

              <TabList selectedValue={selectedTab} onTabSelect={(_, data) => setSelectedTab(data.value)}>
                <Tab value="info" icon={<Desktop24Regular />}>Info</Tab>
                <Tab value="resources" icon={<DocumentText24Regular />}>Resources ({resources.length})</Tab>
                <Tab value="tools" icon={<Wrench24Regular />}>Tools ({tools.length})</Tab>
              </TabList>

              <div className={classes.tabContent}>

                {/* INFO TAB */}
                {selectedTab === 'info' && (
                  <div className={classes.formColumn}>
                    <Field label="Server ID">
                      <Input value={selectedServerData.id} readOnly />
                    </Field>
                    <div className={classes.row}>
                      <Field label="Transport Type" style={{ flex: 1 }}>
                        <Input value={selectedServerData.transport_type} readOnly />
                      </Field>
                      <Field label="URL / Command" style={{ flex: 2 }}>
                        <Input value={selectedServerData.transport_type === 'stdio' ? selectedServerData.command : selectedServerData.url} readOnly style={{ fontFamily: "monospace" }} />
                      </Field>
                    </div>
                    {selectedServerData.args && selectedServerData.args.length > 0 && (
                      <Field label="Arguments">
                        <Input value={selectedServerData.args.join(' ')} readOnly style={{ fontFamily: "monospace" }} />
                      </Field>
                    )}
                  </div>
                )}

                {/* RESOURCES TAB */}
                {selectedTab === 'resources' && (
                  <div className={classes.formColumn}>
                    {resources.length === 0 ? (
                      <Text style={{ color: tokens.colorNeutralForeground3 }}>No resources exposed by this server.</Text>
                    ) : (
                      resources.map(res => (
                        <div key={res.uri} className={classes.resourceItem}>
                          <div style={{ overflow: 'hidden' }}>
                            <Text weight="semibold" block>{res.name}</Text>
                            <Text size={200} style={{ fontFamily: "monospace", color: tokens.colorNeutralForeground3 }}>{res.uri}</Text>
                            {res.mime_type && <Badge appearance="outline" size="small" style={{ marginLeft: '8px' }}>{res.mime_type}</Badge>}
                          </div>
                          <Button size="small" onClick={() => handleReadResource(res.uri)}>Read</Button>
                        </div>
                      ))
                    )}

                    {loading && <Spinner size="small" label="Loading resource..." />}

                    {resourceContent && (
                      <div style={{ marginTop: '16px' }}>
                        <Label weight="semibold">Content: {resourceContent.uri}</Label>
                        <div className={classes.codeBlock}>
                          {resourceContent.text || '(No text content)'}
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {/* TOOLS TAB */}
                {selectedTab === 'tools' && (
                  <div className={classes.formColumn}>
                    {/* Tool Selection */}
                    <Field label="Select Tool">
                      <Dropdown
                        placeholder="Select a tool"
                        onOptionSelect={(_, data) => setToolForm({ ...toolForm, name: data.optionValue || '' })}
                        value={toolForm.name}
                      >
                        {tools.map(t => (
                          <Option key={t.name} value={t.name} text={t.name}>
                            <div style={{ display: 'flex', flexDirection: 'column' }}>
                              <Text>{t.name}</Text>
                              <Caption1 style={{ color: tokens.colorNeutralForeground3 }}>{t.description || 'No description'}</Caption1>
                            </div>
                          </Option>
                        ))}
                      </Dropdown>
                    </Field>

                    {toolForm.name && (
                      <>
                        <Field label="Arguments (JSON)" hint="Enter arguments as a JSON object">
                          <Textarea
                            rows={5}
                            value={toolForm.arguments}
                            onChange={(e) => setToolForm({ ...toolForm, arguments: e.target.value })}
                            style={{ fontFamily: 'monospace' }}
                          />
                        </Field>
                        <Button
                          appearance="primary"
                          icon={<Play24Regular />}
                          onClick={handleCallTool}
                          disabled={toolLoading}
                        >
                          {toolLoading ? 'Executing...' : 'Execute Tool'}
                        </Button>
                      </>
                    )}

                    {toolResult && (
                      <div style={{ marginTop: '16px' }}>
                        <Label weight="semibold">Result:</Label>
                        <div className={classes.codeBlock}>
                          {JSON.stringify(toolResult.content, null, 2)}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', height: '100%', color: tokens.colorNeutralForeground3, gap: '16px' }}>
              <PlugConnected24Regular style={{ fontSize: '48px', opacity: 0.5 }} />
              <Text>Select a server from the list to view details</Text>
            </div>
          )}
        </Card>
      </div>

      {/* Register Server Dialog */}
      <Dialog open={showAddServer} onOpenChange={(_, data) => setShowAddServer(data.open)}>
        <DialogSurface className={classes.dialog}>
          <DialogBody>
            <DialogTitle>Register New MCP Server</DialogTitle>
            <DialogContent className={classes.formColumn}>
              <div className={classes.row}>
                <Field label="Server ID" required style={{ flex: 1 }}>
                  <Input
                    value={serverForm.id}
                    onChange={(e) => setServerForm({ ...serverForm, id: e.target.value })}
                    placeholder="my-server"
                  />
                </Field>
                <Field label="Display Name" required style={{ flex: 1 }}>
                  <Input
                    value={serverForm.name}
                    onChange={(e) => setServerForm({ ...serverForm, name: e.target.value })}
                    placeholder="My Server"
                  />
                </Field>
              </div>

              <Field label="Description">
                <Input
                  value={serverForm.description}
                  onChange={(e) => setServerForm({ ...serverForm, description: e.target.value })}
                />
              </Field>

              <Field label="Transport Type" required>
                <Dropdown
                  value={serverForm.transport_type === 'stdio' ? 'Standard I/O (Subprocess)' : serverForm.transport_type === 'websocket' ? 'WebSocket' : 'HTTP SSE'}
                  onOptionSelect={(_, data) => setServerForm({ ...serverForm, transport_type: data.optionValue || 'stdio' })}
                >
                  <Option value="stdio">Standard I/O (Subprocess)</Option>
                  <Option value="websocket">WebSocket</Option>
                  <Option value="http-sse">HTTP SSE</Option>
                </Dropdown>
              </Field>

              {serverForm.transport_type === 'stdio' ? (
                <>
                  <Field label="Command" required>
                    <Input
                      value={serverForm.command}
                      onChange={(e) => setServerForm({ ...serverForm, command: e.target.value })}
                      placeholder="npx -y @modelcontextprotocol/server-filesystem"
                      style={{ fontFamily: "monospace" }}
                    />
                  </Field>
                  <Field label="Arguments (comma separated)">
                    <Input
                      value={serverForm.args}
                      onChange={(e) => setServerForm({ ...serverForm, args: e.target.value })}
                      placeholder="/allowed/path"
                    />
                  </Field>
                </>
              ) : (
                <Field label="URL" required>
                  <Input
                    value={serverForm.url}
                    onChange={(e) => setServerForm({ ...serverForm, url: e.target.value })}
                    placeholder="ws://localhost:8080/mcp"
                  />
                </Field>
              )}

              <Switch
                label="Require Authentication"
                checked={serverForm.require_auth}
                onChange={(_, data) => setServerForm({ ...serverForm, require_auth: data.checked })}
              />

            </DialogContent>
            <DialogActions>
              <DialogTrigger disableButtonEnhancement>
                <Button appearance="secondary">Cancel</Button>
              </DialogTrigger>
              <Button appearance="primary" onClick={handleRegisterServer} disabled={loading}>
                {loading ? 'Registering...' : 'Register'}
              </Button>
            </DialogActions>
          </DialogBody>
        </DialogSurface>
      </Dialog>
    </div>
  );
}
