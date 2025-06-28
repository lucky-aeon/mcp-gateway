import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Grid,
  Chip,
  IconButton,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Alert,
  CircularProgress,
  Card,
  CardContent,
  Tabs,
  Tab,
} from '@mui/material';
import {
  Send as SendIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  ExpandMore as ExpandMoreIcon,
  ContentCopy as CopyIcon,
  Clear as ClearIcon,
  Save as SaveIcon,
  History as HistoryIcon,
  Code as CodeIcon,
} from '@mui/icons-material';
import { apiDebugApi, type APIEndpoint, type APITestRequest, type APITestResponse } from '../../services/api';
import ResponseViewer from './ResponseViewer';

interface APITestPanelProps {
  api: APIEndpoint;
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
      id={`test-tabpanel-${index}`}
      aria-labelledby={`test-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 2 }}>{children}</Box>}
    </div>
  );
}

export default function APITestPanel({ api }: APITestPanelProps) {
  const [tabValue, setTabValue] = useState(0);
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<APITestResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Request parameters
  const [headers, setHeaders] = useState<Array<{ key: string; value: string }>>([
    { key: 'Authorization', value: 'Bearer 123456' },
    { key: 'Content-Type', value: 'application/json' }
  ]);
  const [queryParams, setQueryParams] = useState<Array<{ key: string; value: string }>>([]);
  const [bodyText, setBodyText] = useState('');
  const [pathParams, setPathParams] = useState<Record<string, string>>({});

  // History
  const [testHistory, setTestHistory] = useState<APITestResponse[]>([]);

  // Initialize path parameters from API definition
  useEffect(() => {
    if (api.parameters) {
      const pathParamsInit: Record<string, string> = {};
      api.parameters
        .filter(param => param.location === 'path')
        .forEach(param => {
          pathParamsInit[param.name] = param.example || '';
        });
      setPathParams(pathParamsInit);
    }
  }, [api]);

  // Auto-fill example data
  useEffect(() => {
    if (api.examples && api.examples.length > 0) {
      const firstExample = api.examples[0];
      if (firstExample.request) {
        setBodyText(JSON.stringify(firstExample.request, null, 2));
      }
    }
  }, [api]);

  const addHeader = () => {
    setHeaders([...headers, { key: '', value: '' }]);
  };

  const removeHeader = (index: number) => {
    setHeaders(headers.filter((_, i) => i !== index));
  };

  const updateHeader = (index: number, field: 'key' | 'value', value: string) => {
    const newHeaders = [...headers];
    newHeaders[index][field] = value;
    setHeaders(newHeaders);
  };

  const addQueryParam = () => {
    setQueryParams([...queryParams, { key: '', value: '' }]);
  };

  const removeQueryParam = (index: number) => {
    setQueryParams(queryParams.filter((_, i) => i !== index));
  };

  const updateQueryParam = (index: number, field: 'key' | 'value', value: string) => {
    const newParams = [...queryParams];
    newParams[index][field] = value;
    setQueryParams(newParams);
  };

  const buildTestPath = (): string => {
    let path = api.path;
    Object.entries(pathParams).forEach(([key, value]) => {
      path = path.replace(`:${key}`, value);
    });
    return path;
  };

  const handleSendRequest = async () => {
    setLoading(true);
    setError(null);
    setResponse(null);

    try {
      // Build request
      const request: APITestRequest = {
        method: api.method,
        path: buildTestPath(),
        headers: headers.reduce((acc, header) => {
          if (header.key && header.value) {
            acc[header.key] = header.value;
          }
          return acc;
        }, {} as Record<string, string>),
        query: queryParams.reduce((acc, param) => {
          if (param.key && param.value) {
            acc[param.key] = param.value;
          }
          return acc;
        }, {} as Record<string, string>),
      };

      // Add body for non-GET requests
      if (api.method !== 'GET' && bodyText.trim()) {
        try {
          request.body = JSON.parse(bodyText);
        } catch (err) {
          setError('请求体不是有效的JSON格式');
          return;
        }
      }

      const result = await apiDebugApi.testAPI(request);
      setResponse(result.data);
      setTestHistory(prev => [result.data, ...prev.slice(0, 9)]); // Keep last 10 results
      setTabValue(1); // Switch to response tab
    } catch (err: any) {
      setError(err.response?.data?.error || 'API测试失败');
    } finally {
      setLoading(false);
    }
  };

  const loadFromHistory = (historyItem: APITestResponse) => {
    if (historyItem.request_headers) {
      const headerEntries = Object.entries(historyItem.request_headers).map(([key, value]) => ({
        key,
        value
      }));
      setHeaders(headerEntries);
    }
    if (historyItem.request_body) {
      setBodyText(historyItem.request_body);
    }
  };

  const clearForm = () => {
    setHeaders([
      { key: 'Authorization', value: 'Bearer 123456' },
      { key: 'Content-Type', value: 'application/json' }
    ]);
    setQueryParams([]);
    setBodyText('');
    setResponse(null);
    setError(null);
  };

  const copyAsCode = () => {
    const curlCommand = `curl -X ${api.method} ${headers.map(h => `-H "${h.key}: ${h.value}"`).join(' ')}${bodyText ? ` -d '${bodyText}'` : ''} "http://localhost:8080${buildTestPath()}"`;
    navigator.clipboard.writeText(curlCommand);
  };

  return (
    <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={tabValue} onChange={(_, newValue) => setTabValue(newValue)}>
          <Tab icon={<SendIcon />} label="发送请求" iconPosition="start" />
          <Tab 
            icon={<CodeIcon />} 
            label="响应结果" 
            iconPosition="start"
            disabled={!response}
          />
          <Tab 
            icon={<HistoryIcon />} 
            label={`历史记录 (${testHistory.length})`} 
            iconPosition="start"
            disabled={testHistory.length === 0}
          />
        </Tabs>
      </Box>

      <Box sx={{ flex: 1, overflow: 'auto' }}>
        <TabPanel value={tabValue} index={0}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {/* Request URL */}
            <Card variant="outlined">
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  请求URL
                </Typography>
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                  <Chip label={api.method} color="primary" />
                  <Typography 
                    variant="body1" 
                    component="code"
                    sx={{ 
                      fontFamily: 'monospace',
                      bgcolor: 'grey.100',
                      px: 1,
                      py: 0.5,
                      borderRadius: 1,
                      flex: 1
                    }}
                  >
                    {buildTestPath()}
                  </Typography>
                </Box>
              </CardContent>
            </Card>

            {/* Path Parameters */}
            {api.parameters && api.parameters.some(p => p.location === 'path') && (
              <Accordion>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Typography variant="h6">路径参数</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <Grid container spacing={2}>
                    {api.parameters
                      .filter(param => param.location === 'path')
                      .map(param => (
                        <Grid item xs={12} sm={6} key={param.name}>
                          <TextField
                            fullWidth
                            label={param.name}
                            value={pathParams[param.name] || ''}
                            onChange={(e) => setPathParams({
                              ...pathParams,
                              [param.name]: e.target.value
                            })}
                            placeholder={param.example}
                            helperText={param.description}
                            required={param.required}
                          />
                        </Grid>
                      ))}
                  </Grid>
                </AccordionDetails>
              </Accordion>
            )}

            {/* Headers */}
            <Accordion defaultExpanded>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography variant="h6">请求头</Typography>
              </AccordionSummary>
              <AccordionDetails>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  {headers.map((header, index) => (
                    <Box key={index} sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                      <TextField
                        size="small"
                        label="键"
                        value={header.key}
                        onChange={(e) => updateHeader(index, 'key', e.target.value)}
                        sx={{ flex: 1 }}
                      />
                      <TextField
                        size="small"
                        label="值"
                        value={header.value}
                        onChange={(e) => updateHeader(index, 'value', e.target.value)}
                        sx={{ flex: 2 }}
                      />
                      <IconButton 
                        color="error" 
                        onClick={() => removeHeader(index)}
                        disabled={headers.length <= 1}
                      >
                        <DeleteIcon />
                      </IconButton>
                    </Box>
                  ))}
                  <Button startIcon={<AddIcon />} onClick={addHeader} variant="outlined">
                    添加请求头
                  </Button>
                </Box>
              </AccordionDetails>
            </Accordion>

            {/* Query Parameters */}
            <Accordion>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography variant="h6">查询参数</Typography>
              </AccordionSummary>
              <AccordionDetails>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  {queryParams.map((param, index) => (
                    <Box key={index} sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                      <TextField
                        size="small"
                        label="参数名"
                        value={param.key}
                        onChange={(e) => updateQueryParam(index, 'key', e.target.value)}
                        sx={{ flex: 1 }}
                      />
                      <TextField
                        size="small"
                        label="参数值"
                        value={param.value}
                        onChange={(e) => updateQueryParam(index, 'value', e.target.value)}
                        sx={{ flex: 2 }}
                      />
                      <IconButton color="error" onClick={() => removeQueryParam(index)}>
                        <DeleteIcon />
                      </IconButton>
                    </Box>
                  ))}
                  <Button startIcon={<AddIcon />} onClick={addQueryParam} variant="outlined">
                    添加查询参数
                  </Button>
                </Box>
              </AccordionDetails>
            </Accordion>

            {/* Request Body */}
            {api.method !== 'GET' && (
              <Accordion>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Typography variant="h6">请求体 (JSON)</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <TextField
                    fullWidth
                    multiline
                    rows={8}
                    value={bodyText}
                    onChange={(e) => setBodyText(e.target.value)}
                    placeholder="输入JSON格式的请求体..."
                    sx={{ fontFamily: 'monospace' }}
                  />
                </AccordionDetails>
              </Accordion>
            )}

            {/* Actions */}
            <Box sx={{ display: 'flex', gap: 2, justifyContent: 'flex-end' }}>
              <Button 
                startIcon={<ClearIcon />} 
                onClick={clearForm}
                variant="outlined"
              >
                清空
              </Button>
              <Button 
                startIcon={<CopyIcon />} 
                onClick={copyAsCode}
                variant="outlined"
              >
                复制为cURL
              </Button>
              <Button
                startIcon={loading ? <CircularProgress size={20} /> : <SendIcon />}
                onClick={handleSendRequest}
                disabled={loading}
                variant="contained"
                size="large"
              >
                {loading ? '发送中...' : '发送请求'}
              </Button>
            </Box>

            {error && (
              <Alert severity="error" sx={{ mt: 2 }}>
                {error}
              </Alert>
            )}
          </Box>
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          {response ? (
            <ResponseViewer response={response} />
          ) : (
            <Typography color="textSecondary" sx={{ textAlign: 'center', py: 4 }}>
              发送请求后查看响应结果
            </Typography>
          )}
        </TabPanel>

        <TabPanel value={tabValue} index={2}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <Typography variant="h6">测试历史</Typography>
            {testHistory.map((item, index) => (
              <Card key={index} variant="outlined">
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                    <Box sx={{ display: 'flex', gap: 1 }}>
                      <Chip 
                        label={item.success ? '成功' : '失败'} 
                        color={item.success ? 'success' : 'error'}
                        size="small"
                      />
                      <Chip label={item.status_code} size="small" />
                    </Box>
                    <Button 
                      size="small" 
                      onClick={() => loadFromHistory(item)}
                      startIcon={<SaveIcon />}
                    >
                      加载此配置
                    </Button>
                  </Box>
                  <Typography variant="body2" color="textSecondary">
                    响应时间: {(item.response_time / 1000000).toFixed(1)}ms
                  </Typography>
                  {item.error && (
                    <Typography variant="body2" color="error">
                      错误: {item.error}
                    </Typography>
                  )}
                </CardContent>
              </Card>
            ))}
          </Box>
        </TabPanel>
      </Box>
    </Paper>
  );
}