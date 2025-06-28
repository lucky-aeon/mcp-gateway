import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  List,
  ListItem,
  ListItemButton,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  InputAdornment,
  Badge,
  IconButton,
  Tooltip,
  Skeleton,
  Alert,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  Search as SearchIcon,
  Refresh as RefreshIcon,
  Http as HttpIcon,
  GetApp as GetIcon,
  PostAdd as PostIcon,
  Edit as PutIcon,
  Delete as DeleteIcon,
} from '@mui/icons-material';
import { apiDebugApi, type APIEndpoint, type APIDiscoveryResponse } from '../../services/api';

interface APIExplorerProps {
  onSelectAPI: (api: APIEndpoint) => void;
  selectedAPI?: APIEndpoint | null;
}

const HTTP_METHOD_COLORS: Record<string, 'success' | 'info' | 'warning' | 'error' | 'default'> = {
  GET: 'success',
  POST: 'info',
  PUT: 'warning',
  DELETE: 'error',
  PATCH: 'warning',
};

const HTTP_METHOD_ICONS: Record<string, React.ReactElement> = {
  GET: <GetIcon />,
  POST: <PostIcon />,
  PUT: <PutIcon />,
  DELETE: <DeleteIcon />,
};

export default function APIExplorer({ onSelectAPI, selectedAPI }: APIExplorerProps) {
  const [searchTerm, setSearchTerm] = useState('');
  const [expandedGroups, setExpandedGroups] = useState<string[]>([]);
  const [apiData, setApiData] = useState<APIDiscoveryResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // 加载API数据
  const loadAPIData = async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await apiDebugApi.discoverAPIs();
      setApiData(response.data);
      // 默认展开所有分组
      setExpandedGroups(response.data.groups);
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to load API data');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadAPIData();
  }, []);

  // 按分组整理API
  const groupedAPIs = React.useMemo(() => {
    if (!apiData) return {};
    
    const grouped: Record<string, APIEndpoint[]> = {};
    
    apiData.endpoints.forEach(api => {
      const group = api.group || '其他';
      if (!grouped[group]) {
        grouped[group] = [];
      }
      grouped[group].push(api);
    });
    
    return grouped;
  }, [apiData]);

  // 过滤API
  const filteredGroupedAPIs = React.useMemo(() => {
    if (!searchTerm) return groupedAPIs;
    
    const filtered: Record<string, APIEndpoint[]> = {};
    
    Object.entries(groupedAPIs).forEach(([group, apis]) => {
      const filteredApis = apis.filter(api => 
        api.path.toLowerCase().includes(searchTerm.toLowerCase()) ||
        api.method.toLowerCase().includes(searchTerm.toLowerCase()) ||
        (api.description && api.description.toLowerCase().includes(searchTerm.toLowerCase())) ||
        (api.tags && api.tags.some(tag => tag.toLowerCase().includes(searchTerm.toLowerCase())))
      );
      
      if (filteredApis.length > 0) {
        filtered[group] = filteredApis;
      }
    });
    
    return filtered;
  }, [groupedAPIs, searchTerm]);

  const handleGroupToggle = (group: string) => {
    setExpandedGroups(prev => 
      prev.includes(group) 
        ? prev.filter(g => g !== group)
        : [...prev, group]
    );
  };


  if (loading) {
    return (
      <Paper sx={{ p: 2, height: '100%' }}>
        <Box sx={{ mb: 2 }}>
          <Skeleton variant="text" width="60%" height={32} />
          <Skeleton variant="rectangular" width="100%" height={40} sx={{ mt: 1 }} />
        </Box>
        {[1, 2, 3].map(i => (
          <Box key={i} sx={{ mb: 2 }}>
            <Skeleton variant="text" width="40%" height={24} />
            <Skeleton variant="rectangular" width="100%" height={60} />
          </Box>
        ))}
      </Paper>
    );
  }

  if (error) {
    return (
      <Paper sx={{ p: 2, height: '100%' }}>
        <Alert 
          severity="error" 
          action={
            <IconButton color="inherit" size="small" onClick={loadAPIData}>
              <RefreshIcon />
            </IconButton>
          }
        >
          {error}
        </Alert>
      </Paper>
    );
  }

  return (
    <Paper sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
          <Typography variant="h6">
            API Explorer
            {apiData && (
              <Chip 
                label={`${apiData.total_endpoints} APIs`} 
                size="small" 
                sx={{ ml: 1 }} 
              />
            )}
          </Typography>
          <Tooltip title="刷新API列表">
            <IconButton onClick={loadAPIData} size="small">
              <RefreshIcon />
            </IconButton>
          </Tooltip>
        </Box>
        
        <TextField
          fullWidth
          size="small"
          placeholder="搜索API (路径、方法、描述、标签)..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Box>

      {/* API List */}
      <Box sx={{ flex: 1, overflow: 'auto' }}>
        {Object.keys(filteredGroupedAPIs).length === 0 ? (
          <Box sx={{ p: 3, textAlign: 'center' }}>
            <Typography color="textSecondary">
              {searchTerm ? '没有找到匹配的API' : '没有可用的API'}
            </Typography>
          </Box>
        ) : (
          Object.entries(filteredGroupedAPIs).map(([group, apis]) => (
            <Accordion 
              key={group}
              expanded={expandedGroups.includes(group)}
              onChange={() => handleGroupToggle(group)}
              disableGutters
            >
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography variant="subtitle1" sx={{ fontWeight: 'medium' }}>
                  {group}
                  <Badge badgeContent={apis.length} color="primary" sx={{ ml: 1 }}>
                    <Box component="span" />
                  </Badge>
                </Typography>
              </AccordionSummary>
              <AccordionDetails sx={{ p: 0 }}>
                <List dense>
                  {apis.map((api, index) => (
                    <ListItem key={`${api.method}-${api.path}-${index}`} disablePadding>
                      <ListItemButton
                        selected={selectedAPI?.path === api.path && selectedAPI?.method === api.method}
                        onClick={() => onSelectAPI(api)}
                        sx={{ 
                          flexDirection: 'column', 
                          alignItems: 'flex-start',
                          '&.Mui-selected': {
                            backgroundColor: 'action.selected',
                          }
                        }}
                      >
                        <Box sx={{ display: 'flex', alignItems: 'center', width: '100%', mb: 0.5 }}>
                          <Chip
                            icon={HTTP_METHOD_ICONS[api.method] || <HttpIcon />}
                            label={api.method}
                            color={HTTP_METHOD_COLORS[api.method] || 'default'}
                            size="small"
                            sx={{ minWidth: 70, mr: 1 }}
                          />
                          <Typography 
                            variant="body2" 
                            sx={{ 
                              fontFamily: 'monospace',
                              flex: 1,
                              overflow: 'hidden',
                              textOverflow: 'ellipsis'
                            }}
                          >
                            {api.path}
                          </Typography>
                        </Box>
                        
                        {api.description && (
                          <Typography 
                            variant="caption" 
                            color="textSecondary"
                            sx={{ ml: 9, mb: 0.5 }}
                          >
                            {api.description}
                          </Typography>
                        )}
                        
                        {api.tags && api.tags.length > 0 && (
                          <Box sx={{ display: 'flex', gap: 0.5, ml: 9 }}>
                            {api.tags.map(tag => (
                              <Chip 
                                key={tag}
                                label={tag} 
                                size="small" 
                                variant="outlined"
                                sx={{ height: 20, fontSize: '0.6rem' }}
                              />
                            ))}
                          </Box>
                        )}
                      </ListItemButton>
                    </ListItem>
                  ))}
                </List>
              </AccordionDetails>
            </Accordion>
          ))
        )}
      </Box>

      {/* Footer */}
      {apiData && (
        <Box sx={{ p: 1, borderTop: 1, borderColor: 'divider', bgcolor: 'grey.50' }}>
          <Typography variant="caption" color="textSecondary">
            更新时间: {new Date(apiData.generated_at).toLocaleString()}
          </Typography>
        </Box>
      )}
    </Paper>
  );
}