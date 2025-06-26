import { useState } from 'react';
import {
  Card,
  CardContent,
  CardActions,
  Typography,
  Button,
  Chip,
  Box,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogContentText,
} from '@mui/material';
import {
  Delete as DeleteIcon,
  Apps as AppsIcon,
  Group as GroupIcon,
} from '@mui/icons-material';
import type { WorkspaceInfo } from '../services/api';

interface WorkspaceCardProps {
  workspace: WorkspaceInfo;
  onDelete: (id: string) => void;
  onViewServices: (id: string) => void;
}

export default function WorkspaceCard({ workspace, onDelete, onViewServices }: WorkspaceCardProps) {
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  const handleDelete = () => {
    onDelete(workspace.id);
    setDeleteDialogOpen(false);
  };

  return (
    <>
      <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
        <CardContent sx={{ flexGrow: 1 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
            <Typography variant="h6" component="h2">
              {workspace.id}
            </Typography>
            <IconButton
              size="small"
              color="error"
              onClick={() => setDeleteDialogOpen(true)}
            >
              <DeleteIcon />
            </IconButton>
          </Box>
          
          <Chip
            label={workspace.status}
            color={workspace.status === 'running' ? 'success' : 'default'}
            size="small"
            sx={{ mb: 2 }}
          />

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 1 }}>
            <AppsIcon fontSize="small" />
            <Typography variant="body2" color="text.secondary">
              {workspace.service_count} Services
            </Typography>
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <GroupIcon fontSize="small" />
            <Typography variant="body2" color="text.secondary">
              {workspace.session_count} Sessions
            </Typography>
          </Box>
        </CardContent>

        <CardActions>
          <Button size="small" onClick={() => onViewServices(workspace.id)}>
            View Services
          </Button>
        </CardActions>
      </Card>

      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
      >
        <DialogTitle>Delete Workspace</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete workspace "{workspace.id}"? This will also delete all services and sessions in this workspace.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleDelete} color="error" variant="contained">
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </>
  );
}