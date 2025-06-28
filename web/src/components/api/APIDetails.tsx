import {
  Box,
  Paper,
  Typography,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Card,
  CardContent,
  Button,
  IconButton,
  Tooltip,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  ContentCopy as CopyIcon,
  Code as CodeIcon,
  Http as HttpIcon,
} from '@mui/icons-material';
import { type APIEndpoint } from '../../services/api';

interface APIDetailsProps {
  api: APIEndpoint;
  onTestAPI?: () => void;
}

const HTTP_METHOD_COLORS: Record<string, 'success' | 'info' | 'warning' | 'error' | 'default'> = {
  GET: 'success',
  POST: 'info',
  PUT: 'warning',
  DELETE: 'error',
  PATCH: 'warning',
};

export default function APIDetails({ api, onTestAPI }: APIDetailsProps) {
  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const generateCurlCommand = () => {
    const method = api.method;
    const path = api.path;
    const headers = '-H "Authorization: Bearer your-api-key" -H "Content-Type: application/json"';
    
    let cmd = `curl -X ${method} ${headers}`;
    
    if (method === 'POST' || method === 'PUT' || method === 'PATCH') {
      cmd += ' -d \'{"key": "value"}\'';
    }
    
    cmd += ` http://localhost:8080${path}`;
    return cmd;
  };

  const generateJavaScriptCode = () => {
    const method = api.method;
    const path = api.path;
    
    return `// 使用 fetch API
const response = await fetch('${path}', {
  method: '${method}',
  headers: {
    'Authorization': 'Bearer your-api-key',
    'Content-Type': 'application/json'
  }${method !== 'GET' ? ',\n  body: JSON.stringify({ /* your data */ })' : ''}
});

const data = await response.json();
console.log(data);`;
  };

  return (
    <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ p: 3, borderBottom: 1, borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <Chip
            icon={<HttpIcon />}
            label={api.method}
            color={HTTP_METHOD_COLORS[api.method] || 'default'}
            sx={{ mr: 2 }}
          />
          <Typography 
            variant="h6" 
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
            {api.path}
          </Typography>
          <Tooltip title="复制路径">
            <IconButton onClick={() => copyToClipboard(api.path)} size="small">
              <CopyIcon />
            </IconButton>
          </Tooltip>
        </Box>

        {api.description && (
          <Typography variant="body1" color="textSecondary" sx={{ mb: 2 }}>
            {api.description}
          </Typography>
        )}

        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', mb: 2 }}>
          {api.group && (
            <Chip label={api.group} variant="outlined" size="small" />
          )}
          {api.tags?.map(tag => (
            <Chip key={tag} label={tag} variant="outlined" size="small" />
          ))}
        </Box>

        {onTestAPI && (
          <Button 
            variant="contained" 
            onClick={onTestAPI}
            startIcon={<CodeIcon />}
          >
            测试此API
          </Button>
        )}
      </Box>

      {/* Content */}
      <Box sx={{ flex: 1, overflow: 'auto', p: 3 }}>
        {/* Parameters */}
        {api.parameters && api.parameters.length > 0 && (
          <Accordion defaultExpanded>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="h6">参数</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>名称</TableCell>
                      <TableCell>类型</TableCell>
                      <TableCell>位置</TableCell>
                      <TableCell>必需</TableCell>
                      <TableCell>描述</TableCell>
                      <TableCell>示例</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {api.parameters.map((param, index) => (
                      <TableRow key={index}>
                        <TableCell>
                          <code>{param.name}</code>
                        </TableCell>
                        <TableCell>
                          <Chip label={param.type} size="small" variant="outlined" />
                        </TableCell>
                        <TableCell>
                          <Chip 
                            label={param.location} 
                            size="small"
                            color={
                              param.location === 'path' ? 'primary' :
                              param.location === 'query' ? 'secondary' :
                              param.location === 'body' ? 'info' : 'default'
                            }
                          />
                        </TableCell>
                        <TableCell>
                          <Chip 
                            label={param.required ? '是' : '否'} 
                            size="small"
                            color={param.required ? 'error' : 'default'}
                          />
                        </TableCell>
                        <TableCell>{param.description}</TableCell>
                        <TableCell>
                          {param.example && (
                            <code style={{ fontSize: '0.875rem' }}>
                              {param.example}
                            </code>
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </AccordionDetails>
          </Accordion>
        )}

        {/* Examples */}
        {api.examples && api.examples.length > 0 && (
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="h6">使用示例</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                {api.examples.map((example, index) => (
                  <Card key={index} variant="outlined">
                    <CardContent>
                      <Typography variant="subtitle1" gutterBottom>
                        {example.name}
                      </Typography>
                      {example.description && (
                        <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                          {example.description}
                        </Typography>
                      )}
                      
                      {example.request && (
                        <Box sx={{ mb: 2 }}>
                          <Typography variant="subtitle2" gutterBottom>
                            请求示例:
                          </Typography>
                          <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50' }}>
                            <pre style={{ margin: 0, fontSize: '0.875rem' }}>
                              {JSON.stringify(example.request, null, 2)}
                            </pre>
                          </Paper>
                        </Box>
                      )}
                      
                      {example.response && (
                        <Box>
                          <Typography variant="subtitle2" gutterBottom>
                            响应示例:
                          </Typography>
                          <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50' }}>
                            <pre style={{ margin: 0, fontSize: '0.875rem' }}>
                              {JSON.stringify(example.response, null, 2)}
                            </pre>
                          </Paper>
                        </Box>
                      )}
                    </CardContent>
                  </Card>
                ))}
              </Box>
            </AccordionDetails>
          </Accordion>
        )}

        {/* Code Examples */}
        <Accordion>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="h6">代码示例</Typography>
          </AccordionSummary>
          <AccordionDetails>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              {/* cURL */}
              <Card variant="outlined">
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                    <Typography variant="subtitle1">cURL</Typography>
                    <IconButton 
                      size="small" 
                      onClick={() => copyToClipboard(generateCurlCommand())}
                    >
                      <CopyIcon />
                    </IconButton>
                  </Box>
                  <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50' }}>
                    <pre style={{ margin: 0, fontSize: '0.875rem', wordWrap: 'break-word' }}>
                      {generateCurlCommand()}
                    </pre>
                  </Paper>
                </CardContent>
              </Card>

              {/* JavaScript */}
              <Card variant="outlined">
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                    <Typography variant="subtitle1">JavaScript (Fetch)</Typography>
                    <IconButton 
                      size="small" 
                      onClick={() => copyToClipboard(generateJavaScriptCode())}
                    >
                      <CopyIcon />
                    </IconButton>
                  </Box>
                  <Paper variant="outlined" sx={{ p: 2, bgcolor: 'grey.50' }}>
                    <pre style={{ margin: 0, fontSize: '0.875rem' }}>
                      {generateJavaScriptCode()}
                    </pre>
                  </Paper>
                </CardContent>
              </Card>
            </Box>
          </AccordionDetails>
        </Accordion>

        {/* Additional Info */}
        <Box sx={{ mt: 3, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
          <Typography variant="subtitle2" gutterBottom>
            API信息
          </Typography>
          <Typography variant="body2" color="textSecondary">
            处理器: {api.handler}
          </Typography>
          {api.middleware && api.middleware.length > 0 && (
            <Typography variant="body2" color="textSecondary">
              中间件: {api.middleware.join(', ')}
            </Typography>
          )}
        </Box>
      </Box>
    </Paper>
  );
}