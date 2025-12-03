import { useState, useEffect } from 'react';
import { Card, StatCard } from '@/components/Card';
import { Button } from '@/components/Button';
import { toast } from 'sonner';
import { 
  Server, 
  Plus, 
  Trash2, 
  RefreshCw, 
  FileText, 
  Wrench, 
  Activity,
  CheckCircle,
  XCircle,
  Loader2
} from 'lucide-react';

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

export default function MCP() {
  const [servers, setServers] = useState<MCPServer[]>([]);
  const [selectedServer, setSelectedServer] = useState<string | null>(null);
  const [resources, setResources] = useState<MCPResource[]>([]);
  const [tools, setTools] = useState<MCPTool[]>([]);
  const [loading, setLoading] = useState(false);
  const [showAddServer, setShowAddServer] = useState(false);
  const [resourceContent, setResourceContent] = useState<ResourceContent | null>(null);
  const [toolResult, setToolResult] = useState<{ content: ToolResultContent[] } | null>(null);

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

  const handleRegisterServer = async (e: React.FormEvent) => {
    e.preventDefault();
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

  const handleDisconnectServer = async (serverId: string) => {
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

  const handleCallTool = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setToolResult(null);

    try {
      const { apiClient } = await import('@/lib/api');
      if (!selectedServer) return;

      let args = {};
      try {
        args = JSON.parse(toolForm.arguments);
      } catch {
        toast.error('Invalid JSON in arguments');
        setLoading(false);
        return;
      }

      const data = await apiClient.callMCPTool(selectedServer, toolForm.name, args);
      setToolResult(data);
      toast.success('Tool executed successfully');
    } catch (error: unknown) {
      const errorMessage = error instanceof Error ? error.message : String(error);
      toast.error('Failed to call tool: ' + errorMessage);
    } finally {
      setLoading(false);
    }
  };

  const selectedServerData = servers.find(s => s.id === selectedServer);
  const connectedCount = servers.filter(s => s.status === 'connected').length;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Model Context Protocol</h1>
          <p className="text-gray-600 mt-1">Manage MCP servers and interact with AI model resources and tools</p>
        </div>
        <Button onClick={() => setShowAddServer(!showAddServer)} className="flex items-center gap-2">
          <Plus className="w-4 h-4" />
          Register Server
        </Button>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatCard
          title="Total Servers"
          value={servers.length}
          icon={<Server className="w-6 h-6" />}
          gradient="linear-gradient(to right, rgb(59, 130, 246), rgb(37, 99, 235))"
        />
        <StatCard
          title="Connected"
          value={connectedCount}
          icon={<CheckCircle className="w-6 h-6" />}
          gradient="linear-gradient(to right, rgb(34, 197, 94), rgb(22, 163, 74))"
        />
        <StatCard
          title="Available Tools"
          value={tools.length}
          icon={<Wrench className="w-6 h-6" />}
          gradient="linear-gradient(to right, rgb(168, 85, 247), rgb(147, 51, 234))"
        />
      </div>

      {/* Add Server Form */}
      {showAddServer && (
        <Card>
          <h2 className="text-xl font-semibold mb-4">Register MCP Server</h2>
          <form onSubmit={handleRegisterServer} className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Server ID *
                </label>
                <input
                  type="text"
                  required
                  value={serverForm.id}
                  onChange={(e) => setServerForm({ ...serverForm, id: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  placeholder="filesystem-server"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  Name *
                </label>
                <input
                  type="text"
                  required
                  value={serverForm.name}
                  onChange={(e) => setServerForm({ ...serverForm, name: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  placeholder="Filesystem MCP Server"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Description
              </label>
              <input
                type="text"
                value={serverForm.description}
                onChange={(e) => setServerForm({ ...serverForm, description: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md"
                placeholder="Provides file system access"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Transport Type *
              </label>
              <select
                value={serverForm.transport_type}
                onChange={(e) => setServerForm({ ...serverForm, transport_type: e.target.value })}
                className="w-full px-3 py-2 border border-gray-300 rounded-md"
              >
                <option value="stdio">Standard I/O (subprocess)</option>
                <option value="websocket">WebSocket</option>
                <option value="http-sse">HTTP Server-Sent Events</option>
              </select>
            </div>

            {serverForm.transport_type === 'stdio' ? (
              <>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Command *
                  </label>
                  <input
                    type="text"
                    required={serverForm.transport_type === 'stdio'}
                    value={serverForm.command}
                    onChange={(e) => setServerForm({ ...serverForm, command: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                    placeholder="npx -y @modelcontextprotocol/server-filesystem"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-gray-700 mb-1">
                    Arguments (comma-separated)
                  </label>
                  <input
                    type="text"
                    value={serverForm.args}
                    onChange={(e) => setServerForm({ ...serverForm, args: e.target.value })}
                    className="w-full px-3 py-2 border border-gray-300 rounded-md"
                    placeholder="/path/to/allowed/directory"
                  />
                </div>
              </>
            ) : (
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">
                  URL *
                </label>
                <input
                  type="text"
                  required={serverForm.transport_type !== 'stdio'}
                  value={serverForm.url}
                  onChange={(e) => setServerForm({ ...serverForm, url: e.target.value })}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md"
                  placeholder="ws://localhost:8000/mcp or http://localhost:8000/sse"
                />
              </div>
            )}

            <div className="flex items-center gap-4">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={serverForm.require_auth}
                  onChange={(e) => setServerForm({ ...serverForm, require_auth: e.target.checked })}
                  className="rounded"
                />
                <span className="text-sm text-gray-700">Require Authentication</span>
              </label>
            </div>

            <div className="flex gap-2">
              <Button type="submit" disabled={loading} className="flex items-center gap-2">
                {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Plus className="w-4 h-4" />}
                Register Server
              </Button>
              <Button 
                type="button" 
                variant="secondary" 
                onClick={() => setShowAddServer(false)}
              >
                Cancel
              </Button>
            </div>
          </form>
        </Card>
      )}

      {/* Server List and Details */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Server List */}
        <Card className="lg:col-span-1">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold">Servers</h2>
            <Button variant="secondary" size="sm" onClick={loadServers} disabled={loading}>
              <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            </Button>
          </div>
          {servers.length === 0 ? (
            <p className="text-gray-500 text-sm">No servers registered. Click "Register Server" to add one.</p>
          ) : (
            <div className="space-y-2">
              {servers.map((server) => (
                <div
                  key={server.id}
                  onClick={() => setSelectedServer(server.id)}
                  className={`p-3 rounded-lg border-2 cursor-pointer transition-all ${
                    selectedServer === server.id
                      ? 'border-blue-500 bg-blue-50'
                      : 'border-gray-200 hover:border-gray-300'
                  }`}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2">
                        <Server className="w-4 h-4 text-gray-500" />
                        <h3 className="font-medium text-sm">{server.name}</h3>
                      </div>
                      <p className="text-xs text-gray-500 mt-1">{server.id}</p>
                      <div className="flex items-center gap-2 mt-2">
                        {server.status === 'connected' ? (
                          <span className="flex items-center gap-1 text-xs text-green-600">
                            <Activity className="w-3 h-3" />
                            Connected
                          </span>
                        ) : (
                          <span className="flex items-center gap-1 text-xs text-gray-500">
                            <XCircle className="w-3 h-3" />
                            Disconnected
                          </span>
                        )}
                      </div>
                    </div>
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleDisconnectServer(server.id);
                      }}
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>

        {/* Server Details */}
        <div className="lg:col-span-2 space-y-6">
          {selectedServerData ? (
            <>
              <Card>
                <h2 className="text-xl font-semibold mb-4">Server Details</h2>
                <div className="space-y-3">
                  <div>
                    <span className="text-sm font-medium text-gray-700">Name:</span>
                    <span className="ml-2 text-sm text-gray-900">{selectedServerData.name}</span>
                  </div>
                  {selectedServerData.description && (
                    <div>
                      <span className="text-sm font-medium text-gray-700">Description:</span>
                      <span className="ml-2 text-sm text-gray-900">{selectedServerData.description}</span>
                    </div>
                  )}
                  <div>
                    <span className="text-sm font-medium text-gray-700">Transport:</span>
                    <span className="ml-2 text-sm text-gray-900">{selectedServerData.transport_type}</span>
                  </div>
                  {selectedServerData.command && (
                    <div>
                      <span className="text-sm font-medium text-gray-700">Command:</span>
                      <span className="ml-2 text-sm text-gray-900 font-mono">{selectedServerData.command}</span>
                    </div>
                  )}
                  <div>
                    <span className="text-sm font-medium text-gray-700">Status:</span>
                    <span className={`ml-2 text-sm font-medium ${
                      selectedServerData.status === 'connected' ? 'text-green-600' : 'text-gray-500'
                    }`}>
                      {selectedServerData.status}
                    </span>
                  </div>
                </div>
              </Card>

              {/* Resources */}
              <Card>
                <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
                  <FileText className="w-5 h-5" />
                  Resources ({resources.length})
                </h2>
                {resources.length === 0 ? (
                  <p className="text-gray-500 text-sm">No resources available</p>
                ) : (
                  <div className="space-y-2">
                    {resources.map((resource) => (
                      <div
                        key={resource.uri}
                        className="p-3 border border-gray-200 rounded-lg hover:border-blue-500 transition-colors"
                      >
                        <div className="flex items-center justify-between">
                          <div className="flex-1">
                            <h3 className="font-medium text-sm">{resource.name}</h3>
                            <p className="text-xs text-gray-500 mt-1">{resource.uri}</p>
                            {resource.description && (
                              <p className="text-xs text-gray-600 mt-1">{resource.description}</p>
                            )}
                          </div>
                          <Button
                            size="sm"
                            onClick={() => handleReadResource(resource.uri)}
                            disabled={loading}
                          >
                            Read
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
                
                {resourceContent && (
                  <div className="mt-4 p-4 bg-gray-50 rounded-lg">
                    <h3 className="font-medium text-sm mb-2">Resource Content:</h3>
                    <p className="text-xs text-gray-600 mb-2">URI: {resourceContent.uri}</p>
                    {resourceContent.mime_type && (
                      <p className="text-xs text-gray-600 mb-2">Type: {resourceContent.mime_type}</p>
                    )}
                    <pre className="text-xs bg-white p-3 rounded border border-gray-200 overflow-auto max-h-64">
                      {resourceContent.text || '(no text content)'}
                    </pre>
                  </div>
                )}
              </Card>

              {/* Tools */}
              <Card>
                <h2 className="text-xl font-semibold mb-4 flex items-center gap-2">
                  <Wrench className="w-5 h-5" />
                  Tools ({tools.length})
                </h2>
                {tools.length === 0 ? (
                  <p className="text-gray-500 text-sm">No tools available</p>
                ) : (
                  <>
                    <div className="space-y-2 mb-4">
                      {tools.map((tool) => (
                        <div
                          key={tool.name}
                          className="p-3 border border-gray-200 rounded-lg"
                        >
                          <h3 className="font-medium text-sm">{tool.name}</h3>
                          {tool.description && (
                            <p className="text-xs text-gray-600 mt-1">{tool.description}</p>
                          )}
                        </div>
                      ))}
                    </div>

                    <div className="mt-4 p-4 bg-gray-50 rounded-lg">
                      <h3 className="font-medium text-sm mb-3">Call Tool</h3>
                      <form onSubmit={handleCallTool} className="space-y-3">
                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-1">
                            Tool Name
                          </label>
                          <select
                            value={toolForm.name}
                            onChange={(e) => setToolForm({ ...toolForm, name: e.target.value })}
                            className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm"
                            required
                          >
                            <option value="">Select a tool...</option>
                            {tools.map((tool) => (
                              <option key={tool.name} value={tool.name}>
                                {tool.name}
                              </option>
                            ))}
                          </select>
                        </div>
                        <div>
                          <label className="block text-sm font-medium text-gray-700 mb-1">
                            Arguments (JSON)
                          </label>
                          <textarea
                            value={toolForm.arguments}
                            onChange={(e) => setToolForm({ ...toolForm, arguments: e.target.value })}
                            className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm font-mono"
                            rows={3}
                            placeholder='{"param": "value"}'
                          />
                        </div>
                        <Button type="submit" disabled={loading || !toolForm.name}>
                          {loading ? <Loader2 className="w-4 h-4 animate-spin mr-2" /> : null}
                          Execute Tool
                        </Button>
                      </form>

                      {toolResult && (
                        <div className="mt-4">
                          <h4 className="font-medium text-sm mb-2">Tool Result:</h4>
                          <pre className="text-xs bg-white p-3 rounded border border-gray-200 overflow-auto max-h-64">
                            {JSON.stringify(toolResult.content, null, 2)}
                          </pre>
                        </div>
                      )}
                    </div>
                  </>
                )}
              </Card>
            </>
          ) : (
            <Card>
              <p className="text-gray-500 text-center py-8">
                Select a server from the list to view details and interact with its resources and tools
              </p>
            </Card>
          )}
        </div>
      </div>
    </div>
  );
}
