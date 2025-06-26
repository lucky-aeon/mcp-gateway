import { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Button,
  Grid,
  CircularProgress,
  Alert,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Fab,
} from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';
import { workspaceApi } from '../services/api';
import type { WorkspaceInfo } from '../services/api';
import WorkspaceCard from '../components/WorkspaceCard';

export default function Workspaces() {
  const [workspaces, setWorkspaces] = useState<WorkspaceInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [createDialogOpen, setCreateDialogOpen] = useState(false);
  const [newWorkspaceName, setNewWorkspaceName] = useState('');
  const [creating, setCreating] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    loadWorkspaces();
  }, []);

  const loadWorkspaces = async () => {
    try {
      setLoading(true);
      const response = await workspaceApi.getAll();
      setWorkspaces(response.data);
      setError(null);
    } catch (err) {
      setError('Failed to load workspaces');
      console.error('Error loading workspaces:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateWorkspace = async () => {
    if (!newWorkspaceName.trim()) return;

    try {
      setCreating(true);
      await workspaceApi.create(newWorkspaceName.trim());
      setCreateDialogOpen(false);
      setNewWorkspaceName('');
      await loadWorkspaces();
    } catch (err) {
      setError('Failed to create workspace');
      console.error('Error creating workspace:', err);
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteWorkspace = async (id: string) => {
    try {
      await workspaceApi.delete(id);
      await loadWorkspaces();
    } catch (err) {
      setError('Failed to delete workspace');
      console.error('Error deleting workspace:', err);
    }
  };

  const handleViewServices = (workspaceId: string) => {
    navigate(`/services?workspace=${workspaceId}`);
  };

  if (loading) {
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
          Workspaces
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setCreateDialogOpen(true)}
        >
          Create Workspace
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      {workspaces.length === 0 ? (
        <Box sx={{ textAlign: 'center', mt: 8 }}>
          <Typography variant="h6" color="textSecondary" gutterBottom>
            No workspaces found
          </Typography>
          <Typography color="textSecondary" sx={{ mb: 3 }}>
            Create your first workspace to get started
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setCreateDialogOpen(true)}
          >
            Create Workspace
          </Button>
        </Box>
      ) : (
        <Grid container spacing={3}>
          {workspaces.map((workspace) => (
            <Grid item xs={12} md={6} lg={4} key={workspace.id}>
              <WorkspaceCard
                workspace={workspace}
                onDelete={handleDeleteWorkspace}
                onViewServices={handleViewServices}
              />
            </Grid>
          ))}
        </Grid>
      )}

      <Fab
        color="primary"
        aria-label="add"
        sx={{ position: 'fixed', bottom: 16, right: 16 }}
        onClick={() => setCreateDialogOpen(true)}
      >
        <AddIcon />
      </Fab>

      <Dialog open={createDialogOpen} onClose={() => setCreateDialogOpen(false)}>
        <DialogTitle>Create New Workspace</DialogTitle>
        <DialogContent>
          <TextField
            autoFocus
            margin="dense"
            label="Workspace Name"
            type="text"
            fullWidth
            variant="outlined"
            value={newWorkspaceName}
            onChange={(e) => setNewWorkspaceName(e.target.value)}
            onKeyPress={(e) => e.key === 'Enter' && handleCreateWorkspace()}
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setCreateDialogOpen(false)}>Cancel</Button>
          <Button
            onClick={handleCreateWorkspace}
            variant="contained"
            disabled={!newWorkspaceName.trim() || creating}
          >
            {creating ? <CircularProgress size={20} /> : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}