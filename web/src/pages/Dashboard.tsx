import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Card,
  CardContent,
  CircularProgress,
  Alert,
} from '@mui/material';
import {
  Workspaces as WorkspacesIcon,
  Apps as AppsIcon,
  Group as GroupIcon,
  TrendingUp as TrendingUpIcon,
} from '@mui/icons-material';
import { workspaceApi } from '../services/api';
import type { WorkspaceInfo } from '../services/api';

interface StatCardProps {
  title: string;
  value: number;
  icon: React.ReactNode;
  color: string;
}

function StatCard({ title, value, icon, color }: StatCardProps) {
  return (
    <Card>
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Box>
            <Typography color="textSecondary" gutterBottom variant="body2">
              {title}
            </Typography>
            <Typography variant="h4">
              {value}
            </Typography>
          </Box>
          <Box sx={{ color, fontSize: 40 }}>
            {icon}
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
}

export default function Dashboard() {
  const [workspaces, setWorkspaces] = useState<WorkspaceInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadWorkspaces();
  }, []);

  const loadWorkspaces = async () => {
    try {
      setLoading(true);
      const response = await workspaceApi.getAll();
      setWorkspaces(response.data);
    } catch (err) {
      setError('Failed to load workspaces');
      console.error('Error loading workspaces:', err);
    } finally {
      setLoading(false);
    }
  };

  const totalServices = workspaces.reduce((sum, ws) => sum + ws.service_count, 0);
  const totalSessions = workspaces.reduce((sum, ws) => sum + ws.session_count, 0);
  const runningWorkspaces = workspaces.filter(ws => ws.status === 'running').length;

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 4 }}>
        Dashboard
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          {error}
        </Alert>
      )}

      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Total Workspaces"
            value={workspaces.length}
            icon={<WorkspacesIcon />}
            color="#1976d2"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Running Workspaces"
            value={runningWorkspaces}
            icon={<TrendingUpIcon />}
            color="#2e7d32"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Total Services"
            value={totalServices}
            icon={<AppsIcon />}
            color="#ed6c02"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <StatCard
            title="Active Sessions"
            value={totalSessions}
            icon={<GroupIcon />}
            color="#9c27b0"
          />
        </Grid>
      </Grid>

      <Paper sx={{ p: 3 }}>
        <Typography variant="h6" sx={{ mb: 3 }}>
          Workspace Overview
        </Typography>
        
        {workspaces.length === 0 ? (
          <Typography color="textSecondary">
            No workspaces found. Create a workspace to get started.
          </Typography>
        ) : (
          <Grid container spacing={2}>
            {workspaces.map((workspace) => (
              <Grid item xs={12} md={6} lg={4} key={workspace.id}>
                <Card variant="outlined">
                  <CardContent>
                    <Typography variant="h6" gutterBottom>
                      {workspace.id}
                    </Typography>
                    <Typography color="textSecondary" sx={{ mb: 1 }}>
                      Status: {workspace.status}
                    </Typography>
                    <Typography variant="body2">
                      Services: {workspace.service_count} | Sessions: {workspace.session_count}
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>
        )}
      </Paper>
    </Box>
  );
}