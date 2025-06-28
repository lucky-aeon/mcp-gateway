import React, { useState } from 'react';
import {
  Box,
  Grid,
  Typography,
  Paper,
  Tabs,
  Tab,
  Chip,
  Alert,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  Api as ApiIcon,
  Code as CodeIcon,
  Description as DocsIcon,
  BugReport as DebugIcon,
} from '@mui/icons-material';
import APIExplorer from '../components/api/APIExplorer';
import APIDetails from '../components/api/APIDetails';
import APITestPanel from '../components/api/APITestPanel';
import { type APIEndpoint } from '../services/api';

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
      id={`api-tabpanel-${index}`}
      aria-labelledby={`api-tab-${index}`}
      {...other}
    >
      {value === index && <Box>{children}</Box>}
    </div>
  );
}

export default function APIDebug() {
  const [selectedAPI, setSelectedAPI] = useState<APIEndpoint | null>(null);
  const [tabValue, setTabValue] = useState(0);
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));

  const handleSelectAPI = (api: APIEndpoint) => {
    setSelectedAPI(api);
    // On mobile, switch to details/test tab when API is selected
    if (isMobile && tabValue === 0) {
      setTabValue(1);
    }
  };

  const handleTestAPI = () => {
    if (isMobile) {
      setTabValue(2); // Mobile: test tab is index 2
    } else {
      setTabValue(1); // Desktop: test tab is index 1
    }
  };

  return (
    <Box sx={{ minHeight: 'calc(100vh - 140px)', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Paper sx={{ p: 3, mb: 2, flexShrink: 0 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <DebugIcon fontSize="large" color="primary" />
            <Typography variant="h4" component="h1">
              API调试工具
            </Typography>
          </Box>
          <Chip 
            icon={<ApiIcon />}
            label="开发者工具"
            color="primary"
            variant="outlined"
          />
        </Box>
        
        <Typography variant="body1" color="textSecondary" sx={{ mb: 2 }}>
          探索、测试和调试所有可用的API端点。支持实时测试、代码生成和响应分析。
        </Typography>

        <Alert severity="info" sx={{ mb: 2 }}>
          <strong>提示:</strong> 这是一个自动生成的API文档，会实时同步最新的路由信息。
          左侧选择API端点，右侧查看详情和进行测试。
        </Alert>

        {selectedAPI && (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <Typography variant="body2" color="textSecondary">
              当前选择:
            </Typography>
            <Chip 
              label={selectedAPI.method}
              color="primary" 
              size="small"
            />
            <Typography 
              variant="body2" 
              component="code"
              sx={{ 
                fontFamily: 'monospace',
                bgcolor: 'grey.100',
                px: 1,
                py: 0.5,
                borderRadius: 1
              }}
            >
              {selectedAPI.path}
            </Typography>
          </Box>
        )}
      </Paper>

      {/* Main Content */}
      <Box sx={{ flex: 1, minHeight: 0 }}>
        {isMobile ? (
          // Mobile Layout: Use tabs
          <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
              <Tabs value={tabValue} onChange={(_, newValue) => setTabValue(newValue)} variant="fullWidth">
                <Tab icon={<ApiIcon />} label="API列表" iconPosition="start" />
                <Tab 
                  icon={<DocsIcon />} 
                  label="文档" 
                  iconPosition="start"
                  disabled={!selectedAPI}
                />
                <Tab 
                  icon={<CodeIcon />} 
                  label="测试" 
                  iconPosition="start"
                  disabled={!selectedAPI}
                />
              </Tabs>
            </Box>

            <Box sx={{ flex: 1, overflow: 'auto' }}>
              <TabPanel value={tabValue} index={0}>
                <Box sx={{ minHeight: '70vh' }}>
                  <APIExplorer 
                    onSelectAPI={handleSelectAPI}
                    selectedAPI={selectedAPI}
                  />
                </Box>
              </TabPanel>

              <TabPanel value={tabValue} index={1}>
                {selectedAPI ? (
                  <Box sx={{ minHeight: '70vh' }}>
                    <APIDetails api={selectedAPI} onTestAPI={handleTestAPI} />
                  </Box>
                ) : (
                  <Box sx={{ p: 3, textAlign: 'center' }}>
                    <Typography color="textSecondary">
                      请先选择一个API端点
                    </Typography>
                  </Box>
                )}
              </TabPanel>

              <TabPanel value={tabValue} index={2}>
                {selectedAPI ? (
                  <Box sx={{ minHeight: '70vh' }}>
                    <APITestPanel api={selectedAPI} />
                  </Box>
                ) : (
                  <Box sx={{ p: 3, textAlign: 'center' }}>
                    <Typography color="textSecondary">
                      请先选择一个API端点
                    </Typography>
                  </Box>
                )}
              </TabPanel>
            </Box>
          </Paper>
        ) : (
          // Desktop Layout: Split panels
          <Grid container spacing={2} sx={{ minHeight: '70vh' }}>
            {/* Left Panel: API Explorer */}
            <Grid item xs={12} md={4} sx={{ minHeight: '70vh' }}>
              <APIExplorer 
                onSelectAPI={handleSelectAPI}
                selectedAPI={selectedAPI}
              />
            </Grid>

            {/* Right Panel: Details and Test */}
            <Grid item xs={12} md={8} sx={{ minHeight: '70vh' }}>
              {selectedAPI ? (
                <Paper sx={{ minHeight: '70vh', display: 'flex', flexDirection: 'column' }}>
                  <Box sx={{ borderBottom: 1, borderColor: 'divider', flexShrink: 0 }}>
                    <Tabs value={tabValue} onChange={(_, newValue) => setTabValue(newValue)}>
                      <Tab 
                        icon={<DocsIcon />} 
                        label="API文档" 
                        iconPosition="start"
                      />
                      <Tab 
                        icon={<CodeIcon />} 
                        label="测试工具" 
                        iconPosition="start"
                      />
                    </Tabs>
                  </Box>

                  <Box sx={{ flex: 1, overflow: 'auto' }}>
                    <TabPanel value={tabValue} index={0}>
                      <APIDetails api={selectedAPI} onTestAPI={handleTestAPI} />
                    </TabPanel>

                    <TabPanel value={tabValue} index={1}>
                      <APITestPanel api={selectedAPI} />
                    </TabPanel>
                  </Box>
                </Paper>
              ) : (
                <Paper sx={{ height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                  <Box sx={{ textAlign: 'center', p: 4 }}>
                    <ApiIcon sx={{ fontSize: 80, color: 'grey.300', mb: 2 }} />
                    <Typography variant="h6" color="textSecondary" gutterBottom>
                      欢迎使用API调试工具
                    </Typography>
                    <Typography variant="body2" color="textSecondary" sx={{ mb: 3 }}>
                      从左侧选择一个API端点开始探索
                    </Typography>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                      <Typography variant="body2" color="textSecondary">
                        🔍 浏览所有可用的API端点
                      </Typography>
                      <Typography variant="body2" color="textSecondary">
                        📖 查看详细的API文档和示例
                      </Typography>
                      <Typography variant="body2" color="textSecondary">
                        🧪 实时测试API并查看响应
                      </Typography>
                      <Typography variant="body2" color="textSecondary">
                        💻 生成代码示例供集成使用
                      </Typography>
                    </Box>
                  </Box>
                </Paper>
              )}
            </Grid>
          </Grid>
        )}
      </Box>
    </Box>
  );
}