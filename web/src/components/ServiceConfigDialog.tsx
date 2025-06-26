import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Box,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Typography,
  Chip,
  IconButton,
} from '@mui/material';
import { Add as AddIcon, Delete as DeleteIcon } from '@mui/icons-material';
import type { MCPServerConfig, ServiceInfo } from '../services/api';

interface ServiceConfigDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (name: string, config: MCPServerConfig) => void;
  service?: ServiceInfo | null;
  title: string;
}

export default function ServiceConfigDialog({
  open,
  onClose,
  onSave,
  service,
  title,
}: ServiceConfigDialogProps) {
  const [serviceName, setServiceName] = useState('');
  const [config, setConfig] = useState<MCPServerConfig>({
    command: '',
    url: '',
    args: [],
    env: {},
  });
  const [configType, setConfigType] = useState<'command' | 'url'>('command');
  const [newEnvKey, setNewEnvKey] = useState('');
  const [newEnvValue, setNewEnvValue] = useState('');
  const [newArg, setNewArg] = useState('');

  useEffect(() => {
    if (service) {
      setServiceName(service.name);
      setConfig({
        command: service.config.command || '',
        url: service.config.url || '',
        args: [],
        env: {},
      });
      setConfigType(service.config.command ? 'command' : 'url');
    } else {
      setServiceName('');
      setConfig({
        command: '',
        url: '',
        args: [],
        env: {},
      });
      setConfigType('command');
    }
  }, [service, open]);

  const handleSave = () => {
    const finalConfig: MCPServerConfig = {
      ...config,
      [configType === 'command' ? 'url' : 'command']: undefined,
    };
    onSave(serviceName, finalConfig);
    onClose();
  };

  const addEnvVar = () => {
    if (newEnvKey && newEnvValue) {
      setConfig({
        ...config,
        env: { ...config.env, [newEnvKey]: newEnvValue },
      });
      setNewEnvKey('');
      setNewEnvValue('');
    }
  };

  const removeEnvVar = (key: string) => {
    const newEnv = { ...config.env };
    delete newEnv[key];
    setConfig({ ...config, env: newEnv });
  };

  const addArg = () => {
    if (newArg) {
      setConfig({
        ...config,
        args: [...(config.args || []), newArg],
      });
      setNewArg('');
    }
  };

  const removeArg = (index: number) => {
    const newArgs = [...(config.args || [])];
    newArgs.splice(index, 1);
    setConfig({ ...config, args: newArgs });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{title}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          <TextField
            label="Service Name"
            value={serviceName}
            onChange={(e) => setServiceName(e.target.value)}
            disabled={!!service}
            fullWidth
          />

          <FormControl fullWidth>
            <InputLabel>Configuration Type</InputLabel>
            <Select
              value={configType}
              onChange={(e) => setConfigType(e.target.value as 'command' | 'url')}
            >
              <MenuItem value="command">Command</MenuItem>
              <MenuItem value="url">URL</MenuItem>
            </Select>
          </FormControl>

          {configType === 'command' ? (
            <TextField
              label="Command"
              value={config.command}
              onChange={(e) => setConfig({ ...config, command: e.target.value })}
              fullWidth
              placeholder="e.g., uvx mcp-server-filesystem"
            />
          ) : (
            <TextField
              label="URL"
              value={config.url}
              onChange={(e) => setConfig({ ...config, url: e.target.value })}
              fullWidth
              placeholder="e.g., http://localhost:3000"
            />
          )}

          {configType === 'command' && (
            <>
              <Typography variant="subtitle2">Arguments</Typography>
              <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                <TextField
                  label="New Argument"
                  value={newArg}
                  onChange={(e) => setNewArg(e.target.value)}
                  size="small"
                  onKeyPress={(e) => e.key === 'Enter' && addArg()}
                />
                <IconButton onClick={addArg}>
                  <AddIcon />
                </IconButton>
              </Box>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {config.args?.map((arg, index) => (
                  <Chip
                    key={index}
                    label={arg}
                    onDelete={() => removeArg(index)}
                    deleteIcon={<DeleteIcon />}
                  />
                ))}
              </Box>
            </>
          )}

          <Typography variant="subtitle2">Environment Variables</Typography>
          <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
            <TextField
              label="Key"
              value={newEnvKey}
              onChange={(e) => setNewEnvKey(e.target.value)}
              size="small"
            />
            <TextField
              label="Value"
              value={newEnvValue}
              onChange={(e) => setNewEnvValue(e.target.value)}
              size="small"
              onKeyPress={(e) => e.key === 'Enter' && addEnvVar()}
            />
            <IconButton onClick={addEnvVar}>
              <AddIcon />
            </IconButton>
          </Box>
          <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
            {Object.entries(config.env || {}).map(([key, value]) => (
              <Chip
                key={key}
                label={`${key}=${value}`}
                onDelete={() => removeEnvVar(key)}
                deleteIcon={<DeleteIcon />}
              />
            ))}
          </Box>
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={!serviceName}>
          Save
        </Button>
      </DialogActions>
    </Dialog>
  );
}