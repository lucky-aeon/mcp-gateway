import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Chip,
  Tabs,
  Tab,
  IconButton,
  Tooltip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert,
} from '@mui/material';
import {
  ContentCopy as CopyIcon,
  CheckCircle as SuccessIcon,
  Error as ErrorIcon,
  AccessTime as TimeIcon,
  Code as JsonIcon,
  Description as RawIcon,
  Http as HeaderIcon,
} from '@mui/icons-material';
import { type APITestResponse } from '../../services/api';

interface ResponseViewerProps {
  response: APITestResponse;
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
      id={`response-tabpanel-${index}`}
      aria-labelledby={`response-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 2 }}>{children}</Box>}
    </div>
  );
}

export default function ResponseViewer({ response }: ResponseViewerProps) {
  const [tabValue, setTabValue] = useState(0);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatResponseTime = (nanoseconds: number): string => {
    if (nanoseconds < 1000000) {
      return `${(nanoseconds / 1000).toFixed(1)}μs`;
    } else if (nanoseconds < 1000000000) {
      return `${(nanoseconds / 1000000).toFixed(1)}ms`;
    } else {
      return `${(nanoseconds / 1000000000).toFixed(2)}s`;
    }
  };

  const getStatusColor = (statusCode: number): 'success' | 'warning' | 'error' | 'info' => {
    if (statusCode >= 200 && statusCode < 300) return 'success';
    if (statusCode >= 300 && statusCode < 400) return 'info';
    if (statusCode >= 400 && statusCode < 500) return 'warning';
    if (statusCode >= 500) return 'error';
    return 'info';
  };

  const formatJSON = (obj: any): string => {
    try {
      return JSON.stringify(obj, null, 2);
    } catch {
      return String(obj);
    }
  };

  return (
    <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
          <Typography variant="h6">响应结果</Typography>
          <Chip
            icon={response.success ? <SuccessIcon /> : <ErrorIcon />}
            label={response.success ? '成功' : '失败'}
            color={response.success ? 'success' : 'error'}
          />
        </Box>

        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
          <Chip
            label={`${response.status_code}`}
            color={getStatusColor(response.status_code)}
            size="small"
          />
          <Chip
            icon={<TimeIcon />}
            label={formatResponseTime(response.response_time)}
            size="small"
            variant="outlined"
          />
        </Box>

        {response.error && (
          <Alert severity="error" sx={{ mt: 2 }}>
            {response.error}
          </Alert>
        )}
      </Box>

      {/* Tabs */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={tabValue} onChange={(_, newValue) => setTabValue(newValue)}>
          <Tab 
            icon={<JsonIcon />} 
            label="JSON响应" 
            iconPosition="start"
            disabled={!response.response}
          />
          <Tab 
            icon={<RawIcon />} 
            label="原始响应" 
            iconPosition="start"
            disabled={!response.response_body}
          />
          <Tab 
            icon={<HeaderIcon />} 
            label="请求详情" 
            iconPosition="start"
          />
        </Tabs>
      </Box>

      {/* Tab Panels */}
      <Box sx={{ flex: 1, overflow: 'auto' }}>
        <TabPanel value={tabValue} index={0}>
          {response.response ? (
            <Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="subtitle1">格式化的JSON响应</Typography>
                <Tooltip title="复制JSON">
                  <IconButton 
                    size="small" 
                    onClick={() => copyToClipboard(formatJSON(response.response))}
                  >
                    <CopyIcon />
                  </IconButton>
                </Tooltip>
              </Box>
              <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50', overflow: 'auto' }}>
                <pre style={{ margin: 0, fontSize: '0.875rem', whiteSpace: 'pre-wrap' }}>
                  {formatJSON(response.response)}
                </pre>
              </Paper>
            </Box>
          ) : (
            <Typography color="textSecondary">无JSON响应数据</Typography>
          )}
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          {response.response_body ? (
            <Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="subtitle1">原始响应内容</Typography>
                <Tooltip title="复制原始响应">
                  <IconButton 
                    size="small" 
                    onClick={() => copyToClipboard(response.response_body || '')}
                  >
                    <CopyIcon />
                  </IconButton>
                </Tooltip>
              </Box>
              <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50', overflow: 'auto' }}>
                <pre style={{ margin: 0, fontSize: '0.875rem', whiteSpace: 'pre-wrap' }}>
                  {response.response_body}
                </pre>
              </Paper>
            </Box>
          ) : (
            <Typography color="textSecondary">无原始响应数据</Typography>
          )}
        </TabPanel>

        <TabPanel value={tabValue} index={2}>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
            {/* Request Headers */}
            {response.request_headers && Object.keys(response.request_headers).length > 0 && (
              <Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="subtitle1">请求头</Typography>
                  <Tooltip title="复制请求头">
                    <IconButton 
                      size="small" 
                      onClick={() => copyToClipboard(formatJSON(response.request_headers))}
                    >
                      <CopyIcon />
                    </IconButton>
                  </Tooltip>
                </Box>
                <TableContainer component={Paper} variant="outlined">
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>名称</TableCell>
                        <TableCell>值</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {Object.entries(response.request_headers).map(([key, value]) => (
                        <TableRow key={key}>
                          <TableCell>
                            <code style={{ fontSize: '0.875rem' }}>{key}</code>
                          </TableCell>
                          <TableCell>
                            <code style={{ fontSize: '0.875rem' }}>{value}</code>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Box>
            )}

            {/* Request Body */}
            {response.request_body && (
              <Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                  <Typography variant="subtitle1">请求体</Typography>
                  <Tooltip title="复制请求体">
                    <IconButton 
                      size="small" 
                      onClick={() => copyToClipboard(response.request_body || '')}
                    >
                      <CopyIcon />
                    </IconButton>
                  </Tooltip>
                </Box>
                <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50', overflow: 'auto' }}>
                  <pre style={{ margin: 0, fontSize: '0.875rem', whiteSpace: 'pre-wrap' }}>
                    {response.request_body}
                  </pre>
                </Paper>
              </Box>
            )}

            {/* Response Summary */}
            <Box>
              <Typography variant="subtitle1" gutterBottom>响应摘要</Typography>
              <Paper variant="outlined" sx={{ p: 2 }}>
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="body2" color="textSecondary">状态码:</Typography>
                    <Chip 
                      label={response.status_code} 
                      color={getStatusColor(response.status_code)}
                      size="small"
                    />
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="body2" color="textSecondary">响应时间:</Typography>
                    <Typography variant="body2">{formatResponseTime(response.response_time)}</Typography>
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="body2" color="textSecondary">成功状态:</Typography>
                    <Chip 
                      label={response.success ? '成功' : '失败'} 
                      color={response.success ? 'success' : 'error'}
                      size="small"
                    />
                  </Box>
                  {response.response_body && (
                    <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                      <Typography variant="body2" color="textSecondary">响应大小:</Typography>
                      <Typography variant="body2">
                        {new Blob([response.response_body]).size} bytes
                      </Typography>
                    </Box>
                  )}
                </Box>
              </Paper>
            </Box>
          </Box>
        </TabPanel>
      </Box>
    </Paper>
  );
}