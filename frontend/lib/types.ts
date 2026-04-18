// MCP Server Types
export interface MCPServer {
  id: string
  name: string
  description: string
  icon: string
  category: string
  version: string
  author: string
  downloads?: number
  rating?: number
  tools: MCPTool[]
  config?: MCPConfig
  status?: 'running' | 'stopped' | 'error'
  installedAt?: string
}

export interface MCPTool {
  name: string
  description: string
  parameters: MCPToolParameter[]
}

export interface MCPToolParameter {
  name: string
  type: string
  description: string
  required: boolean
  default?: string
}

export interface MCPConfig {
  env?: Record<string, string>
  args?: string[]
}

// Workspace Types
export interface Workspace {
  id: string
  name: string
  description: string
  createdAt: string
  updatedAt: string
  mcpCount: number
  sessionCount: number
  status: 'active' | 'inactive'
}

export interface WorkspaceDetail extends Workspace {
  mcps: WorkspaceMCP[]
  sessions: Session[]
  logs: LogEntry[]
  settings: WorkspaceSettings
}

export interface WorkspaceMCP {
  id: string
  mcpId: string
  name: string
  icon: string
  status: 'running' | 'stopped' | 'error'
  enabledTools: string[]
  config: MCPConfig
  addedAt: string
}

export interface WorkspaceSettings {
  maxSessions: number
  sessionTimeout: number
  logRetention: number
  allowedOrigins: string[]
}

// Session Types
export interface Session {
  id: string
  workspaceId: string
  clientId: string
  startedAt: string
  endedAt?: string
  status: 'active' | 'ended' | 'error'
  toolCalls: number
  lastActivity: string
}

// Log Types
export interface LogEntry {
  id: string
  timestamp: string
  level: 'info' | 'warn' | 'error' | 'debug'
  source: string
  message: string
  metadata?: Record<string, unknown>
}

// API Key Types
export interface APIKey {
  id: string
  name: string
  key: string
  prefix: string
  createdAt: string
  lastUsedAt?: string
  expiresAt?: string
  permissions: string[]
  status: 'active' | 'revoked' | 'expired'
}

// Stats Types
export interface OverviewStats {
  totalWorkspaces: number
  activeSessions: number
  totalMCPs: number
  totalToolCalls: number
  recentActivity: ActivityItem[]
}

export interface ActivityItem {
  id: string
  type: 'session_start' | 'session_end' | 'tool_call' | 'mcp_added' | 'mcp_removed'
  description: string
  timestamp: string
  workspaceId?: string
  workspaceName?: string
}

// Navigation Types
export type NavSection = 'overview' | 'workspaces' | 'market' | 'installed' | 'api-keys' | 'playground' | 'setup' | 'ops-dashboard'

// Ops Dashboard Types
export interface WorkspaceMetrics {
  workspaceId: string
  workspaceName: string
  totalSessions: number
  activeSessions: number
  totalToolCalls: number
  avgResponseTime: number
  errorRate: number
  mcpCount: number
  lastActivity: string
}

export interface MCPMetrics {
  mcpId: string
  mcpName: string
  icon: string
  totalCalls: number
  successRate: number
  avgResponseTime: number
  errorCount: number
  lastUsed: string
  status: 'running' | 'stopped' | 'error'
  workspaceUsage: WorkspaceMCPUsage[]
}

export interface WorkspaceMCPUsage {
  workspaceId: string
  workspaceName: string
  callCount: number
}

export interface SystemMetrics {
  cpuUsage: number
  memoryUsage: number
  diskUsage: number
  networkIn: number
  networkOut: number
  uptime: number
  requestRate: number
}

export interface TimeSeriesData {
  timestamp: string
  value: number
}

export interface OpsDashboardData {
  systemMetrics: SystemMetrics
  workspaceMetrics: WorkspaceMetrics[]
  mcpMetrics: MCPMetrics[]
  toolCallsHistory: TimeSeriesData[]
  errorRateHistory: TimeSeriesData[]
  responseTimeHistory: TimeSeriesData[]
  topWorkspacesByCalls: WorkspaceMetrics[]
  topMCPsByCalls: MCPMetrics[]
}
