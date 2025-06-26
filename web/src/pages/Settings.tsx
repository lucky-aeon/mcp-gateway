import {
  Box,
  Typography,
  Paper,
  List,
  ListItem,
  ListItemText,
  Divider,
} from '@mui/material';

export default function Settings() {
  return (
    <Box>
      <Typography variant="h4" sx={{ mb: 4 }}>
        Settings
      </Typography>

      <Paper sx={{ p: 3 }}>
        <Typography variant="h6" gutterBottom>
          System Information
        </Typography>
        
        <List>
          <ListItem>
            <ListItemText
              primary="MCP Gateway Version"
              secondary="1.0.0"
            />
          </ListItem>
          <Divider />
          
          <ListItem>
            <ListItemText
              primary="API Endpoint"
              secondary="http://localhost:8080/api"
            />
          </ListItem>
          <Divider />
          
          <ListItem>
            <ListItemText
              primary="Frontend Framework"
              secondary="React with Material-UI"
            />
          </ListItem>
          <Divider />
          
          <ListItem>
            <ListItemText
              primary="Backend Framework"
              secondary="Go with Echo"
            />
          </ListItem>
        </List>
      </Paper>

      <Paper sx={{ p: 3, mt: 3 }}>
        <Typography variant="h6" gutterBottom>
          About MCP Gateway
        </Typography>
        
        <Typography variant="body1" paragraph>
          MCP Gateway is a reverse proxy that manages multiple MCP (Model Context Protocol) servers 
          and provides unified access through a single interface.
        </Typography>
        
        <Typography variant="body1">
          This management interface allows you to create workspaces, deploy and manage MCP services, 
          and monitor active sessions.
        </Typography>
      </Paper>
    </Box>
  );
}