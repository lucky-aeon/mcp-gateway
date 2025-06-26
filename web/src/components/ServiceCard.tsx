import React, { useState } from 'react';
import {
  Card,
  CardContent,
  CardActions,
  Typography,
  Button,
  Chip,
  Box,
  IconButton,
  Menu,
  MenuItem,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  DialogContentText,
} from '@mui/material';
import {
  MoreVert as MoreVertIcon,
  PlayArrow as StartIcon,
  Stop as StopIcon,
  Refresh as RestartIcon,
  Delete as DeleteIcon,
  Settings as SettingsIcon,
  BugReport as DebugIcon,
} from '@mui/icons-material';
import type { ServiceInfo } from '../services/api';

interface ServiceCardProps {
  service: ServiceInfo;
  onStart: (name: string) => void;
  onStop: (name: string) => void;
  onRestart: (name: string) => void;
  onDelete: (name: string) => void;
  onEdit: (service: ServiceInfo) => void;
  onDebug?: (service: ServiceInfo) => void;
}

export default function ServiceCard({ 
  service, 
  onStart, 
  onStop, 
  onRestart, 
  onDelete, 
  onEdit,
  onDebug
}: ServiceCardProps) {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const open = Boolean(anchorEl);

  const handleMenuClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleDelete = () => {
    onDelete(service.name);
    setDeleteDialogOpen(false);
    handleMenuClose();
  };

  const getStatusColor = (status: string) => {
    switch (status.toLowerCase()) {
      case 'running':
        return 'success';
      case 'stopped':
        return 'default';
      case 'starting':
        return 'warning';
      case 'failed':
        return 'error';
      default:
        return 'default';
    }
  };

  return (
    <>
      <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
        <CardContent sx={{ flexGrow: 1 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
            <Typography variant="h6" component="h2">
              {service.name}
            </Typography>
            <IconButton size="small" onClick={handleMenuClick}>
              <MoreVertIcon />
            </IconButton>
          </Box>
          
          <Chip
            label={service.status}
            color={getStatusColor(service.status)}
            size="small"
            sx={{ mb: 2 }}
          />

          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Port: {service.port || 'N/A'}
          </Typography>

          {service.config.command && (
            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              Command: {service.config.command}
            </Typography>
          )}

          {service.config.url && (
            <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
              URL: {service.config.url}
            </Typography>
          )}

          {service.last_error && (
            <Typography variant="body2" color="error" sx={{ mb: 1 }}>
              Error: {service.last_error}
            </Typography>
          )}
        </CardContent>

        <CardActions>
          {service.status === 'Running' ? (
            <Button size="small" startIcon={<StopIcon />} onClick={() => onStop(service.name)}>
              Stop
            </Button>
          ) : (
            <Button size="small" startIcon={<StartIcon />} onClick={() => onStart(service.name)}>
              Start
            </Button>
          )}
          <Button size="small" startIcon={<RestartIcon />} onClick={() => onRestart(service.name)}>
            Restart
          </Button>
          {onDebug && (
            <Button 
              size="small" 
              startIcon={<DebugIcon />} 
              onClick={() => onDebug(service)}
              disabled={service.status !== 'Running'}
            >
              Debug
            </Button>
          )}
        </CardActions>
      </Card>

      <Menu anchorEl={anchorEl} open={open} onClose={handleMenuClose}>
        <MenuItem onClick={() => { onEdit(service); handleMenuClose(); }}>
          <SettingsIcon sx={{ mr: 1 }} />
          Edit Config
        </MenuItem>
        <MenuItem onClick={() => { setDeleteDialogOpen(true); handleMenuClose(); }}>
          <DeleteIcon sx={{ mr: 1 }} />
          Delete
        </MenuItem>
      </Menu>

      <Dialog
        open={deleteDialogOpen}
        onClose={() => setDeleteDialogOpen(false)}
      >
        <DialogTitle>Delete Service</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Are you sure you want to delete service "{service.name}"? This action cannot be undone.
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