import { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  IconButton,
  CircularProgress,
  Alert,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogContentText,
} from '@mui/material';
import {
  Delete as DeleteIcon,
  Add as AddIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { workspaceApi, sessionApi } from '../services/api';
import type { WorkspaceInfo, SessionInfo } from '../services/api';

export default function Sessions() {
  const [workspaces, setWorkspaces] = useState<WorkspaceInfo[]>([]);
  const [selectedWorkspace, setSelectedWorkspace] = useState<string>('');
  const [sessions, setSessions] = useState<SessionInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [sessionToDelete, setSessionToDelete] = useState<SessionInfo | null>(null);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    loadWorkspaces();
  }, []);

  useEffect(() => {
    if (selectedWorkspace) {
      loadSessions();
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

  const loadSessions = async () => {
    if (!selectedWorkspace) return;

    try {
      setLoading(true);
      const response = await sessionApi.getByWorkspace(selectedWorkspace);
      setSessions(response.data);
      setError(null);
    } catch (err) {
      setError('Failed to load sessions');
      console.error('Error loading sessions:', err);
      setSessions([]); // Set empty array on error
    } finally {
      setLoading(false);
    }
  };

  const handleCreateSession = async () => {
    if (!selectedWorkspace) return;

    try {
      setCreating(true);
      await sessionApi.create(selectedWorkspace);
      await loadSessions();
    } catch (err) {
      setError('Failed to create session');
      console.error('Error creating session:', err);
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteSession = async () => {
    if (!sessionToDelete || !selectedWorkspace) return;

    try {
      await sessionApi.delete(selectedWorkspace, sessionToDelete.id);
      await loadSessions();
      setDeleteDialogOpen(false);
      setSessionToDelete(null);
    } catch (err) {
      setError('Failed to delete session');
      console.error('Error deleting session:', err);
    }
  };

  const openDeleteDialog = (session: SessionInfo) => {
    setSessionToDelete(session);
    setDeleteDialogOpen(true);
  };

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleString();
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'active':
        return 'success';
      case 'inactive':
        return 'default';
      default:
        return 'default';
    }
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
          Sessions
        </Typography>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            variant="outlined"
            startIcon={<RefreshIcon />}
            onClick={loadSessions}
            disabled={!selectedWorkspace}
          >
            Refresh
          </Button>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreateSession}
            disabled={!selectedWorkspace || creating}
          >
            {creating ? <CircularProgress size={20} /> : 'Create Session'}
          </Button>
        </Box>
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
                {workspace.id} ({workspace.session_count} sessions)
              </MenuItem>
            ))}
          </Select>
        </FormControl>
      </Box>

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
          <CircularProgress />
        </Box>
      ) : sessions.length === 0 ? (
        <Box sx={{ textAlign: 'center', mt: 8 }}>
          <Typography variant="h6" color="textSecondary" gutterBottom>
            No sessions found
          </Typography>
          <Typography color="textSecondary" sx={{ mb: 3 }}>
            Create your first session to get started
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={handleCreateSession}
            disabled={!selectedWorkspace || creating}
          >
            {creating ? <CircularProgress size={20} /> : 'Create Session'}
          </Button>
        </Box>
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Session ID</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Ready</TableCell>
                <TableCell>Created At</TableCell>
                <TableCell>Last Activity</TableCell>
                <TableCell align="right">Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {sessions.map((session) => (
                <TableRow key={session.id}>
                  <TableCell>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                      {session.id}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={session.status}
                      color={getStatusColor(session.status)}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={session.is_ready ? 'Ready' : 'Not Ready'}
                      color={session.is_ready ? 'success' : 'default'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>{formatDate(session.created_at)}</TableCell>
                  <TableCell>{formatDate(session.last_receive_time)}</TableCell>
                  <TableCell align="right">
                    <IconButton
                      size="small"
                      color="error"
                      onClick={() => openDeleteDialog(session)}
                    >
                      <DeleteIcon />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
      >
        <DialogTitle>Delete Session</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete session "{sessionToDelete?.id}"? This action cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleDeleteSession} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}