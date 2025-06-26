import { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Button,
  Grid,
  CircularProgress,
  Alert,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Fab,
  Dialog,
  DialogTitle,
  DialogContent,
  IconButton,
} from '@mui/material';
import { Add as AddIcon, Close as CloseIcon } from '@mui/icons-material';
import { useSearchParams } from 'react-router-dom';
import { workspaceApi, serviceApi } from '../services/api';
import type { WorkspaceInfo, ServiceInfo, MCPServerConfig } from '../services/api';
import ServiceCard from '../components/ServiceCard';
import ServiceConfigDialog from '../components/ServiceConfigDialog';
import MCPDebugPanel from '../components/MCPDebugPanel';

export default function Services() {
  const [searchParams] = useSearchParams();
  const [workspaces, setWorkspaces] = useState<WorkspaceInfo[]>([]);
  const [selectedWorkspace, setSelectedWorkspace] = useState<string>('');
  const [services, setServices] = useState<ServiceInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [configDialogOpen, setConfigDialogOpen] = useState(false);
  const [editingService, setEditingService] = useState<ServiceInfo | null>(null);
  const [debugDialogOpen, setDebugDialogOpen] = useState(false);
  const [debuggingService, setDebuggingService] = useState<ServiceInfo | null>(null);

  useEffect(() => {
    loadWorkspaces();
  }, []);

  useEffect(() => {
    const workspaceParam = searchParams.get('workspace');
    if (workspaceParam) {
      setSelectedWorkspace(workspaceParam);
    }
  }, [searchParams]);

  useEffect(() => {
    if (selectedWorkspace) {
      loadServices();
    }
  }, [selectedWorkspace]);

  const loadWorkspaces = async () => {
    try {
      const response = await workspaceApi.getAll();
      setWorkspaces(response.data);
      if (response.data.length > 0 && !selectedWorkspace) {
        setSelectedWorkspace(response.data[0].id);
      }
    } catch (err) {
      setError('Failed to load workspaces');
      console.error('Error loading workspaces:', err);
    } finally {
      setLoading(false);
    }
  };

  const loadServices = async () => {
    if (!selectedWorkspace) return;

    try {
      setLoading(true);
      const response = await workspaceApi.getServices(selectedWorkspace);
      setServices(response.data);
      setError(null);
    } catch (err) {
      setError('Failed to load services');
      console.error('Error loading services:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateService = async (name: string, config: MCPServerConfig) => {
    if (!selectedWorkspace) return;

    try {
      await serviceApi.deploy(selectedWorkspace, { [name]: config });
      await loadServices();
    } catch (err) {
      setError('Failed to create service');
      console.error('Error creating service:', err);
    }
  };

  const handleUpdateService = async (name: string, config: MCPServerConfig) => {
    if (!selectedWorkspace) return;

    try {
      await serviceApi.update(selectedWorkspace, name, config);
      await loadServices();
    } catch (err) {
      setError('Failed to update service');
      console.error('Error updating service:', err);
    }
  };

  const handleStartService = async (name: string) => {
    if (!selectedWorkspace) return;

    try {
      await serviceApi.start(selectedWorkspace, name);
      await loadServices();
    } catch (err) {
      setError('Failed to start service');
      console.error('Error starting service:', err);
    }
  };

  const handleStopService = async (name: string) => {
    if (!selectedWorkspace) return;

    try {
      await serviceApi.stop(selectedWorkspace, name);
      await loadServices();
    } catch (err) {
      setError('Failed to stop service');
      console.error('Error stopping service:', err);
    }
  };

  const handleRestartService = async (name: string) => {
    if (!selectedWorkspace) return;

    try {
      await serviceApi.restart(selectedWorkspace, name);
      await loadServices();
    } catch (err) {
      setError('Failed to restart service');
      console.error('Error restarting service:', err);
    }
  };

  const handleDeleteService = async (name: string) => {
    if (!selectedWorkspace) return;

    try {
      await serviceApi.delete(selectedWorkspace, name);
      await loadServices();
    } catch (err) {
      setError('Failed to delete service');
      console.error('Error deleting service:', err);
    }
  };

  const handleEditService = (service: ServiceInfo) => {
    setEditingService(service);
    setConfigDialogOpen(true);
  };

  const handleConfigSave = (name: string, config: MCPServerConfig) => {
    if (editingService) {
      handleUpdateService(name, config);
    } else {
      handleCreateService(name, config);
    }
    setEditingService(null);
  };

  const handleConfigClose = () => {
    setConfigDialogOpen(false);
    setEditingService(null);
  };

  const handleDebugService = (service: ServiceInfo) => {
    setDebuggingService(service);
    setDebugDialogOpen(true);
  };

  const handleDebugClose = () => {
    setDebugDialogOpen(false);
    setDebuggingService(null);
  };

  if (loading && workspaces.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
        <Typography variant="h4">
          Services
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setConfigDialogOpen(true)}
          disabled={!selectedWorkspace}
        >
          Add Service
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      <Box sx={{ mb: 3 }}>
        <FormControl fullWidth>
          <InputLabel>Workspace</InputLabel>
          <Select
            value={selectedWorkspace}
            onChange={(e) => setSelectedWorkspace(e.target.value)}
            label="Workspace"
          >
            {workspaces.map((workspace) => (
              <MenuItem key={workspace.id} value={workspace.id}>
                {workspace.id} ({workspace.service_count} services)
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
          <CircularProgress />
        </Box>
      ) : services.length === 0 ? (
        <Box sx={{ textAlign: 'center', mt: 8 }}>
          <Typography variant="h6" color="textSecondary" gutterBottom>
            No services found
          </Typography>
          <Typography color="textSecondary" sx={{ mb: 3 }}>
            Add your first service to get started
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setConfigDialogOpen(true)}
            disabled={!selectedWorkspace}
          >
            Add Service
          </Button>
        </Box>
      ) : (
        <Grid container spacing={3}>
          {services.map((service) => (
            <Grid item xs={12} md={6} lg={4} key={service.name}>
              <ServiceCard
                service={service}
                onStart={handleStartService}
                onStop={handleStopService}
                onRestart={handleRestartService}
                onDelete={handleDeleteService}
                onEdit={handleEditService}
                onDebug={handleDebugService}
              />
            </Grid>
          ))}
        </Grid>
      )}

      <Fab
        color="primary"
        aria-label="add"
        sx={{ position: 'fixed', bottom: 16, right: 16 }}
        onClick={() => setConfigDialogOpen(true)}
        disabled={!selectedWorkspace}
      >
        <AddIcon />
      </Fab>

      <ServiceConfigDialog
        open={configDialogOpen}
        onClose={handleConfigClose}
        onSave={handleConfigSave}
        service={editingService}
        title={editingService ? 'Edit Service' : 'Add New Service'}
      />

      <Dialog
        open={debugDialogOpen}
        onClose={handleDebugClose}
        maxWidth="lg"
        fullWidth
        PaperProps={{
          sx: { height: '90vh' }
        }}
      >
        <DialogTitle>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Typography variant="h6">
              Debug Service: {debuggingService?.name}
            </Typography>
            <IconButton onClick={handleDebugClose}>
              <CloseIcon />
            </IconButton>
          </Box>
        </DialogTitle>
        <DialogContent sx={{ p: 0 }}>
          {debuggingService && selectedWorkspace && (
            <MCPDebugPanel
              service={debuggingService}
              workspaceId={selectedWorkspace}
            />
          )}
        </DialogContent>
      </Dialog>
    </Box>
  );
}