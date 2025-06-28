import React, { useState, useEffect, useRef } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Grid,
  Alert,
  Chip,
  List,
  ListItem,
  ListItemText,
  Divider,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Card,
  CardContent,
  CardActions,
  Tabs,
  Tab,
  IconButton,
  Tooltip,
  CircularProgress,
} from '@mui/material';
import {
  Send as SendIcon,
  Refresh as RefreshIcon,
  PlayArrow as ConnectIcon,
  Stop as DisconnectIcon,
  Clear as ClearIcon,
  Code as CodeIcon,
} from '@mui/icons-material';
import {
  SSEConnection,
  MCPMessageSender,
  debugApi,
  type ServiceInfo,
  type DebugResponse,
  type ConnectionTestResult,
  type LogEntry,
} from '../services/api';

interface MCPDebugPanelProps {
  service: ServiceInfo;
  workspaceId: string;
}

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`debug-tabpanel-${index}`}
      aria-labelledby={`debug-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

export default function MCPDebugPanel({ service, workspaceId }: MCPDebugPanelProps) {
  const [tabValue, setTabValue] = useState(0);
  
  // SSE State
  const [sseConnected, setSseConnected] = useState(false);
  const [sseMessages, setSseMessages] = useState<any[]>([]);
  const [sseConnection, setSseConnection] = useState<SSEConnection | null>(null);
  
  // Message Sending State
  const [messageSender] = useState(() => new MCPMessageSender(service.name, workspaceId));
  const [customMessage, setCustomMessage] = useState('');
  const [predefinedMethod, setPredefinedMethod] = useState('ping');
  const [messageResponse, setMessageResponse] = useState<any>(null);
  const [messageLoading, setMessageLoading] = useState(false);
  
  // Debug State
  const [debugResponse, setDebugResponse] = useState<DebugResponse | null>(null);
  const [connectionTest, setConnectionTest] = useState<ConnectionTestResult | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const predefinedMessages = {
    ping: { jsonrpc: "2.0", id: 1, method: "ping", params: {} },
    initialize: {
      jsonrpc: "2.0",
      id: 2,
      method: "initialize",
      params: {
        protocolVersion: "2024-11-05",
        capabilities: {},
        clientInfo: { name: "MCP Gateway Debug", version: "1.0.0" }
      }
    },
    'tools/list': { jsonrpc: "2.0", id: 3, method: "tools/list", params: {} },
    'resources/list': { jsonrpc: "2.0", id: 4, method: "resources/list", params: {} },
  };

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [sseMessages]);

  useEffect(() => {
    return () => {
      if (sseConnection) {
        sseConnection.disconnect();
      }
    };
  }, [sseConnection]);

  const handleConnectSSE = () => {
    const sseUrl = SSEConnection.createSSEUrl(service.name, workspaceId);

    const connection = new SSEConnection(
      sseUrl,
      (data) => {
        setSseMessages(prev => [...prev, { ...data, timestamp: new Date().toISOString() }]);
      },
      (error) => {
        console.error('SSE error:', error);
        setSseConnected(false);
        setError('SSE connection error');
      },
      () => {
        setSseConnected(true);
        setError(null);
      }
    );

    connection.connect();
    setSseConnection(connection);
  };

  const handleDisconnectSSE = () => {
    if (sseConnection) {
      sseConnection.disconnect();
      setSseConnection(null);
      setSseConnected(false);
    }
  };

  const handleSendMessage = async () => {
    if (!customMessage.trim()) return;

    setMessageLoading(true);
    setMessageResponse(null);
    setError(null);

    try {
      let messageObj;
      try {
        messageObj = JSON.parse(customMessage);
      } catch {
        setError('Invalid JSON format');
        return;
      }

      const responseData = await messageSender.sendMessage(messageObj);
      setMessageResponse(responseData);
    } catch (err: any) {
      setError(err.message || 'Failed to send message');
    } finally {
      setMessageLoading(false);
    }
  };

  const handleSendPredefined = async () => {
    setMessageLoading(true);
    setMessageResponse(null);
    setError(null);

    try {
      let result;
      switch (predefinedMethod) {
        case 'ping':
          result = await messageSender.ping();
          break;
        case 'initialize':
          result = await messageSender.initialize({
            protocolVersion: "2024-11-05",
            capabilities: {},
            clientInfo: { name: "MCP Gateway Debug", version: "1.0.0" }
          });
          break;
        case 'tools/list':
          result = await messageSender.listTools();
          break;
        case 'resources/list':
          result = await messageSender.listResources();
          break;
        default:
          throw new Error('Unknown method');
      }
      setMessageResponse(result);
    } catch (err: any) {
      setError(err.message || 'Failed to send message');
    } finally {
      setMessageLoading(false);
    }
  };

  const handleTestConnection = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await debugApi.testConnection(workspaceId, service.name);
      setConnectionTest(response.data);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to test connection');
    } finally {
      setLoading(false);
    }
  };

  const handleLoadLogs = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await debugApi.getLogs(workspaceId, service.name, 50);
      setLogs(response.data.logs);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load logs');
    } finally {
      setLoading(false);
    }
  };

  const handleDebugTest = async () => {
    setLoading(true);
    setError(null);

    try {
      const response = await debugApi.testService(workspaceId, service.name, {
        message: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "ping", params: {} })
      });
      setDebugResponse(response.data);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to run debug test');
    } finally {
      setLoading(false);
    }
  };

  const clearMessages = () => {
    setSseMessages([]);
  };

  const clearCustomMessage = () => {
    setCustomMessage('');
    setMessageResponse(null);
  };

  const loadPredefinedMessage = (method: string) => {
    const message = predefinedMessages[method as keyof typeof predefinedMessages];
    if (message) {
      setCustomMessage(JSON.stringify(message, null, 2));
    }
  };

  return (
    <Box sx={{ width: '100%' }}>
      <Paper sx={{ width: '100%' }}>
        <Tabs value={tabValue} onChange={(_, newValue) => setTabValue(newValue)}>
          <Tab label="SSE Subscription" />
          <Tab label="Send Messages" />
          <Tab label="Debug & Test" />
          <Tab label="Logs" />
        </Tabs>

        {error && (
          <Alert severity="error" sx={{ m: 2 }}>
            {error}
          </Alert>
        )}

        <TabPanel value={tabValue} index={0}>
          {/* SSE Subscription Panel */}
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
                <Chip
                  label={sseConnected ? 'Connected' : 'Disconnected'}
                  color={sseConnected ? 'success' : 'default'}
                  icon={sseConnected ? <ConnectIcon /> : <DisconnectIcon />}
                />
                <Typography variant="caption" color="textSecondary">
                  SSE URL: {SSEConnection.createSSEUrl(service.name, workspaceId)}
                </Typography>
              </Box>
            </Grid>
            
            <Grid item xs={12}>
              <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                <Button
                  variant="contained"
                  startIcon={<ConnectIcon />}
                  onClick={handleConnectSSE}
                  disabled={sseConnected}
                >
                  Connect
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<DisconnectIcon />}
                  onClick={handleDisconnectSSE}
                  disabled={!sseConnected}
                >
                  Disconnect
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<ClearIcon />}
                  onClick={clearMessages}
                >
                  Clear
                </Button>
              </Box>
            </Grid>

            <Grid item xs={12}>
              <Paper variant="outlined" sx={{ height: 400, overflow: 'auto', p: 1 }}>
                {sseMessages.length === 0 ? (
                  <Typography color="textSecondary" sx={{ textAlign: 'center', mt: 4 }}>
                    No messages received. Connect to start receiving SSE messages.
                  </Typography>
                ) : (
                  <List dense>
                    {sseMessages.map((msg, index) => (
                      <React.Fragment key={index}>
                        <ListItem>
                          <ListItemText
                            primary={
                              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                                <Typography variant="caption">
                                  {new Date(msg.timestamp).toLocaleTimeString()}
                                </Typography>
                                {msg.error && (
                                  <Chip label="Error" color="error" size="small" />
                                )}
                              </Box>
                            }
                            secondary={
                              <pre style={{ whiteSpace: 'pre-wrap', fontSize: '0.875rem' }}>
                                {typeof msg === 'string' ? msg : JSON.stringify(msg, null, 2)}
                              </pre>
                            }
                          />
                        </ListItem>
                        {index < sseMessages.length - 1 && <Divider />}
                      </React.Fragment>
                    ))}
                  </List>
                )}
                <div ref={messagesEndRef} />
              </Paper>
            </Grid>
          </Grid>
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          {/* Message Sending Panel */}
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Predefined Methods
                  </Typography>
                  <FormControl fullWidth sx={{ mb: 2 }}>
                    <InputLabel>Method</InputLabel>
                    <Select
                      value={predefinedMethod}
                      label="Method"
                      onChange={(e) => setPredefinedMethod(e.target.value)}
                    >
                      <MenuItem value="ping">Ping</MenuItem>
                      <MenuItem value="initialize">Initialize</MenuItem>
                      <MenuItem value="tools/list">List Tools</MenuItem>
                      <MenuItem value="resources/list">List Resources</MenuItem>
                    </Select>
                  </FormControl>
                </CardContent>
                <CardActions>
                  <Button
                    variant="contained"
                    startIcon={<SendIcon />}
                    onClick={handleSendPredefined}
                    disabled={messageLoading}
                    fullWidth
                  >
                    {messageLoading ? <CircularProgress size={20} /> : 'Send'}
                  </Button>
                </CardActions>
              </Card>
            </Grid>

            <Grid item xs={12} md={6}>
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    Custom Message
                  </Typography>
                  <Typography variant="caption" color="textSecondary" display="block" sx={{ mb: 1 }}>
                    Message URL: /{service.name}/message (with X-Workspace-Id: {workspaceId})
                  </Typography>
                  <TextField
                    multiline
                    rows={8}
                    fullWidth
                    value={customMessage}
                    onChange={(e) => setCustomMessage(e.target.value)}
                    placeholder="Enter JSON-RPC message..."
                    variant="outlined"
                    sx={{ mb: 2 }}
                  />
                  <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                    <Tooltip title="Load ping message">
                      <IconButton onClick={() => loadPredefinedMessage('ping')}>
                        <CodeIcon />
                      </IconButton>
                    </Tooltip>
                    <Button size="small" onClick={() => loadPredefinedMessage('ping')}>
                      Ping
                    </Button>
                    <Button size="small" onClick={() => loadPredefinedMessage('initialize')}>
                      Init
                    </Button>
                  </Box>
                </CardContent>
                <CardActions>
                  <Button
                    variant="contained"
                    startIcon={<SendIcon />}
                    onClick={handleSendMessage}
                    disabled={messageLoading || !customMessage.trim()}
                  >
                    {messageLoading ? <CircularProgress size={20} /> : 'Send'}
                  </Button>
                  <Button
                    variant="outlined"
                    startIcon={<ClearIcon />}
                    onClick={clearCustomMessage}
                  >
                    Clear
                  </Button>
                </CardActions>
              </Card>
            </Grid>

            {messageResponse && (
              <Grid item xs={12}>
                <Paper variant="outlined" sx={{ p: 2 }}>
                  <Typography variant="h6" gutterBottom>
                    Response
                  </Typography>
                  <pre style={{ whiteSpace: 'pre-wrap', overflow: 'auto' }}>
                    {JSON.stringify(messageResponse, null, 2)}
                  </pre>
                </Paper>
              </Grid>
            )}
          </Grid>
        </TabPanel>

        <TabPanel value={tabValue} index={2}>
          {/* Debug & Test Panel */}
          <Grid container spacing={3}>
            <Grid item xs={12}>
              <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
                <Button
                  variant="contained"
                  startIcon={<RefreshIcon />}
                  onClick={handleTestConnection}
                  disabled={loading}
                >
                  Test Connection
                </Button>
                <Button
                  variant="outlined"
                  startIcon={<SendIcon />}
                  onClick={handleDebugTest}
                  disabled={loading}
                >
                  Debug Test
                </Button>
              </Box>
            </Grid>

            {connectionTest && (
              <Grid item xs={12}>
                <Paper variant="outlined" sx={{ p: 2 }}>
                  <Typography variant="h6" gutterBottom>
                    Connection Test Results
                  </Typography>
                  <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                    <Chip
                      label={connectionTest.healthy ? 'Healthy' : 'Unhealthy'}
                      color={connectionTest.healthy ? 'success' : 'error'}
                    />
                    <Chip label={`Success Rate: ${connectionTest.success_rate}`} />
                  </Box>
                  <List dense>
                    {connectionTest.tests.map((test, index) => (
                      <ListItem key={index}>
                        <ListItemText
                          primary={
                            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                              <Chip
                                size="small"
                                label={test.success ? 'PASS' : 'FAIL'}
                                color={test.success ? 'success' : 'error'}
                              />
                              {test.name}
                            </Box>
                          }
                          secondary={test.error || test.details}
                        />
                      </ListItem>
                    ))}
                  </List>
                </Paper>
              </Grid>
            )}

            {debugResponse && (
              <Grid item xs={12}>
                <Paper variant="outlined" sx={{ p: 2 }}>
                  <Typography variant="h6" gutterBottom>
                    Debug Test Response
                  </Typography>
                  <Chip
                    label={debugResponse.success ? 'Success' : 'Failed'}
                    color={debugResponse.success ? 'success' : 'error'}
                    sx={{ mb: 2 }}
                  />
                  {debugResponse.error && (
                    <Alert severity="error" sx={{ mb: 2 }}>
                      {debugResponse.error}
                    </Alert>
                  )}
                  <pre style={{ whiteSpace: 'pre-wrap', overflow: 'auto' }}>
                    {JSON.stringify(debugResponse, null, 2)}
                  </pre>
                </Paper>
              </Grid>
            )}
          </Grid>
        </TabPanel>

        <TabPanel value={tabValue} index={3}>
          {/* Logs Panel */}
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <Button
                variant="contained"
                startIcon={<RefreshIcon />}
                onClick={handleLoadLogs}
                disabled={loading}
              >
                Load Logs
              </Button>
            </Grid>

            <Grid item xs={12}>
              <Paper variant="outlined" sx={{ height: 400, overflow: 'auto', p: 1 }}>
                {logs.length === 0 ? (
                  <Typography color="textSecondary" sx={{ textAlign: 'center', mt: 4 }}>
                    No logs available. Click "Load Logs" to fetch service logs.
                  </Typography>
                ) : (
                  <List dense>
                    {logs.map((log, index) => (
                      <React.Fragment key={index}>
                        <ListItem>
                          <ListItemText
                            primary={
                              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                                <Typography variant="caption">
                                  {log.timestamp}
                                </Typography>
                                <Chip
                                  size="small"
                                  label={log.level}
                                  color={
                                    log.level === 'ERROR' ? 'error' :
                                    log.level === 'WARN' ? 'warning' : 'default'
                                  }
                                />
                              </Box>
                            }
                            secondary={log.message}
                          />
                        </ListItem>
                        {index < logs.length - 1 && <Divider />}
                      </React.Fragment>
                    ))}
                  </List>
                )}
              </Paper>
            </Grid>
          </Grid>
        </TabPanel>
      </Paper>
    </Box>
  );
}