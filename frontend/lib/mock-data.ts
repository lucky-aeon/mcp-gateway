import type {
  MCPServer,
  Workspace,
  WorkspaceDetail,
  Session,
  LogEntry,
  APIKey,
  OverviewStats,
  OpsDashboardData,
  WorkspaceMetrics,
  MCPMetrics,
  SystemMetrics,
  TimeSeriesData,
} from './types'

// Market MCPs
export const marketMCPs: MCPServer[] = [
  {
    id: 'filesystem',
    name: 'Filesystem',
    description: '提供文件系统操作能力，包括读写文件、目录管理等',
    icon: '📁',
    category: '系统',
    version: '1.2.0',
    author: 'MCP Team',
    downloads: 12500,
    rating: 4.8,
    tools: [
      {
        name: 'read_file',
        description: '读取文件内容',
        parameters: [
          { name: 'path', type: 'string', description: '文件路径', required: true },
        ],
      },
      {
        name: 'write_file',
        description: '写入文件内容',
        parameters: [
          { name: 'path', type: 'string', description: '文件路径', required: true },
          { name: 'content', type: 'string', description: '文件内容', required: true },
        ],
      },
      {
        name: 'list_directory',
        description: '列出目录内容',
        parameters: [
          { name: 'path', type: 'string', description: '目录路径', required: true },
        ],
      },
    ],
  },
  {
    id: 'database',
    name: 'Database',
    description: '数据库连接和查询工具，支持 PostgreSQL、MySQL、SQLite',
    icon: '🗄️',
    category: '数据',
    version: '2.0.1',
    author: 'MCP Team',
    downloads: 8900,
    rating: 4.6,
    tools: [
      {
        name: 'query',
        description: '执行 SQL 查询',
        parameters: [
          { name: 'sql', type: 'string', description: 'SQL 语句', required: true },
          { name: 'params', type: 'array', description: '查询参数', required: false },
        ],
      },
      {
        name: 'execute',
        description: '执行 SQL 命令',
        parameters: [
          { name: 'sql', type: 'string', description: 'SQL 语句', required: true },
        ],
      },
    ],
  },
  {
    id: 'web-search',
    name: 'Web Search',
    description: '网络搜索工具，支持多个搜索引擎',
    icon: '🔍',
    category: '网络',
    version: '1.5.0',
    author: 'Search Team',
    downloads: 15600,
    rating: 4.9,
    tools: [
      {
        name: 'search',
        description: '执行网络搜索',
        parameters: [
          { name: 'query', type: 'string', description: '搜索关键词', required: true },
          { name: 'limit', type: 'number', description: '结果数量', required: false, default: '10' },
        ],
      },
    ],
  },
  {
    id: 'github',
    name: 'GitHub',
    description: 'GitHub API 集成，管理仓库、Issues、PR 等',
    icon: '🐙',
    category: '开发',
    version: '1.8.2',
    author: 'GitHub Team',
    downloads: 22000,
    rating: 4.7,
    tools: [
      {
        name: 'list_repos',
        description: '列出用户仓库',
        parameters: [
          { name: 'username', type: 'string', description: '用户名', required: false },
        ],
      },
      {
        name: 'create_issue',
        description: '创建 Issue',
        parameters: [
          { name: 'repo', type: 'string', description: '仓库名称', required: true },
          { name: 'title', type: 'string', description: '标题', required: true },
          { name: 'body', type: 'string', description: '内容', required: false },
        ],
      },
    ],
  },
  {
    id: 'slack',
    name: 'Slack',
    description: 'Slack 消息发送和频道管理',
    icon: '💬',
    category: '通讯',
    version: '1.3.0',
    author: 'Slack Team',
    downloads: 9800,
    rating: 4.5,
    tools: [
      {
        name: 'send_message',
        description: '发送消息到频道',
        parameters: [
          { name: 'channel', type: 'string', description: '频道ID', required: true },
          { name: 'text', type: 'string', description: '消息内容', required: true },
        ],
      },
    ],
  },
  {
    id: 'calendar',
    name: 'Calendar',
    description: '日历管理工具，支持创建、查询、更新事件',
    icon: '📅',
    category: '效率',
    version: '1.1.0',
    author: 'Calendar Team',
    downloads: 6500,
    rating: 4.4,
    tools: [
      {
        name: 'list_events',
        description: '列出日历事件',
        parameters: [
          { name: 'start_date', type: 'string', description: '开始日期', required: true },
          { name: 'end_date', type: 'string', description: '结束日期', required: true },
        ],
      },
      {
        name: 'create_event',
        description: '创建日历事件',
        parameters: [
          { name: 'title', type: 'string', description: '事件标题', required: true },
          { name: 'start', type: 'string', description: '开始时间', required: true },
          { name: 'end', type: 'string', description: '结束时间', required: true },
        ],
      },
    ],
  },
]

// Installed MCPs
export const installedMCPs: MCPServer[] = [
  { ...marketMCPs[0], status: 'running', installedAt: '2024-01-15T10:30:00Z' },
  { ...marketMCPs[1], status: 'running', installedAt: '2024-01-16T14:20:00Z' },
  { ...marketMCPs[3], status: 'stopped', installedAt: '2024-01-18T09:00:00Z' },
]

// Workspaces
export const workspaces: Workspace[] = [
  {
    id: 'ws-1',
    name: '开发环境',
    description: '用于开发和测试的工作空间',
    createdAt: '2024-01-10T08:00:00Z',
    updatedAt: '2024-01-20T15:30:00Z',
    mcpCount: 3,
    sessionCount: 12,
    status: 'active',
  },
  {
    id: 'ws-2',
    name: '生产环境',
    description: '生产环境工作空间',
    createdAt: '2024-01-12T10:00:00Z',
    updatedAt: '2024-01-20T14:00:00Z',
    mcpCount: 2,
    sessionCount: 45,
    status: 'active',
  },
  {
    id: 'ws-3',
    name: '测试环境',
    description: '自动化测试专用',
    createdAt: '2024-01-15T09:00:00Z',
    updatedAt: '2024-01-19T18:00:00Z',
    mcpCount: 4,
    sessionCount: 8,
    status: 'inactive',
  },
]

// Sessions
export const sessions: Session[] = [
  {
    id: 'sess-1',
    workspaceId: 'ws-1',
    clientId: 'client-a1b2c3',
    startedAt: '2024-01-20T14:30:00Z',
    status: 'active',
    toolCalls: 23,
    lastActivity: '2024-01-20T15:25:00Z',
  },
  {
    id: 'sess-2',
    workspaceId: 'ws-1',
    clientId: 'client-d4e5f6',
    startedAt: '2024-01-20T13:00:00Z',
    endedAt: '2024-01-20T14:15:00Z',
    status: 'ended',
    toolCalls: 45,
    lastActivity: '2024-01-20T14:15:00Z',
  },
  {
    id: 'sess-3',
    workspaceId: 'ws-2',
    clientId: 'client-g7h8i9',
    startedAt: '2024-01-20T12:00:00Z',
    status: 'active',
    toolCalls: 89,
    lastActivity: '2024-01-20T15:20:00Z',
  },
]

// Logs
export const logs: LogEntry[] = [
  {
    id: 'log-1',
    timestamp: '2024-01-20T15:25:00Z',
    level: 'info',
    source: 'filesystem',
    message: 'Tool read_file executed successfully',
    metadata: { path: '/data/config.json', duration: 45 },
  },
  {
    id: 'log-2',
    timestamp: '2024-01-20T15:24:00Z',
    level: 'warn',
    source: 'database',
    message: 'Slow query detected',
    metadata: { query: 'SELECT * FROM users', duration: 2500 },
  },
  {
    id: 'log-3',
    timestamp: '2024-01-20T15:23:00Z',
    level: 'error',
    source: 'github',
    message: 'API rate limit exceeded',
    metadata: { remaining: 0, reset: '2024-01-20T16:00:00Z' },
  },
  {
    id: 'log-4',
    timestamp: '2024-01-20T15:22:00Z',
    level: 'info',
    source: 'web-search',
    message: 'Search completed',
    metadata: { query: 'MCP protocol', results: 15 },
  },
  {
    id: 'log-5',
    timestamp: '2024-01-20T15:20:00Z',
    level: 'debug',
    source: 'system',
    message: 'Session heartbeat received',
    metadata: { sessionId: 'sess-1' },
  },
]

// API Keys
export const apiKeys: APIKey[] = [
  {
    id: 'key-1',
    name: '开发密钥',
    key: 'gw_dev_xxxxxxxxxxxxxxxxxxxxx',
    prefix: 'gw_dev_',
    createdAt: '2024-01-10T10:00:00Z',
    lastUsedAt: '2024-01-20T15:00:00Z',
    permissions: ['read', 'write', 'admin'],
    status: 'active',
  },
  {
    id: 'key-2',
    name: '生产密钥',
    key: 'gw_prod_xxxxxxxxxxxxxxxxxxxx',
    prefix: 'gw_prod_',
    createdAt: '2024-01-12T14:00:00Z',
    lastUsedAt: '2024-01-20T14:30:00Z',
    expiresAt: '2025-01-12T14:00:00Z',
    permissions: ['read', 'write'],
    status: 'active',
  },
  {
    id: 'key-3',
    name: '测试密钥',
    key: 'gw_test_xxxxxxxxxxxxxxxxxxxx',
    prefix: 'gw_test_',
    createdAt: '2024-01-05T09:00:00Z',
    permissions: ['read'],
    status: 'revoked',
  },
]

// Overview Stats
export const overviewStats: OverviewStats = {
  totalWorkspaces: 3,
  activeSessions: 2,
  totalMCPs: 3,
  totalToolCalls: 1250,
  recentActivity: [
    {
      id: 'act-1',
      type: 'tool_call',
      description: '执行了 read_file 工具',
      timestamp: '2024-01-20T15:25:00Z',
      workspaceId: 'ws-1',
      workspaceName: '开发环境',
    },
    {
      id: 'act-2',
      type: 'session_start',
      description: '新会话开始',
      timestamp: '2024-01-20T14:30:00Z',
      workspaceId: 'ws-1',
      workspaceName: '开发环境',
    },
    {
      id: 'act-3',
      type: 'mcp_added',
      description: '添加了 GitHub MCP',
      timestamp: '2024-01-20T12:00:00Z',
      workspaceId: 'ws-2',
      workspaceName: '生产环境',
    },
    {
      id: 'act-4',
      type: 'session_end',
      description: '会话结束',
      timestamp: '2024-01-20T11:30:00Z',
      workspaceId: 'ws-1',
      workspaceName: '开发环境',
    },
  ],
}

// Workspace Detail
export const getWorkspaceDetail = (id: string): WorkspaceDetail | null => {
  const workspace = workspaces.find((w) => w.id === id)
  if (!workspace) return null

  return {
    ...workspace,
    mcps: [
      {
        id: 'wm-1',
        mcpId: 'filesystem',
        name: 'Filesystem',
        icon: '📁',
        status: 'running',
        enabledTools: ['read_file', 'write_file', 'list_directory'],
        config: { env: { ROOT_PATH: '/data' } },
        addedAt: '2024-01-15T10:30:00Z',
      },
      {
        id: 'wm-2',
        mcpId: 'database',
        name: 'Database',
        icon: '🗄️',
        status: 'running',
        enabledTools: ['query', 'execute'],
        config: { env: { DB_URL: 'postgresql://localhost:5432/mydb' } },
        addedAt: '2024-01-16T14:20:00Z',
      },
    ],
    sessions: sessions.filter((s) => s.workspaceId === id),
    logs: logs,
    settings: {
      maxSessions: 100,
      sessionTimeout: 3600,
      logRetention: 7,
      allowedOrigins: ['*'],
    },
  }
}

// Ops Dashboard Data
export const systemMetrics: SystemMetrics = {
  cpuUsage: 45.2,
  memoryUsage: 62.8,
  diskUsage: 38.5,
  networkIn: 1024,
  networkOut: 512,
  uptime: 86400 * 7,
  requestRate: 125,
}

export const workspaceMetrics: WorkspaceMetrics[] = [
  {
    workspaceId: 'ws-1',
    workspaceName: '开发环境',
    totalSessions: 156,
    activeSessions: 12,
    totalToolCalls: 3240,
    avgResponseTime: 245,
    errorRate: 2.3,
    mcpCount: 3,
    lastActivity: '2024-01-20T15:25:00Z',
  },
  {
    workspaceId: 'ws-2',
    workspaceName: '生产环境',
    totalSessions: 489,
    activeSessions: 45,
    totalToolCalls: 12580,
    avgResponseTime: 189,
    errorRate: 1.1,
    mcpCount: 2,
    lastActivity: '2024-01-20T15:20:00Z',
  },
  {
    workspaceId: 'ws-3',
    workspaceName: '测试环境',
    totalSessions: 89,
    activeSessions: 8,
    totalToolCalls: 1560,
    avgResponseTime: 312,
    errorRate: 4.5,
    mcpCount: 4,
    lastActivity: '2024-01-19T18:00:00Z',
  },
]

export const mcpMetrics: MCPMetrics[] = [
  {
    mcpId: 'filesystem',
    mcpName: 'Filesystem',
    icon: '📁',
    totalCalls: 5420,
    successRate: 98.5,
    avgResponseTime: 156,
    errorCount: 81,
    lastUsed: '2024-01-20T15:25:00Z',
    status: 'running',
    workspaceUsage: [
      { workspaceId: 'ws-1', workspaceName: '开发环境', callCount: 2340 },
      { workspaceId: 'ws-2', workspaceName: '生产环境', callCount: 2890 },
      { workspaceId: 'ws-3', workspaceName: '测试环境', callCount: 190 },
    ],
  },
  {
    mcpId: 'database',
    mcpName: 'Database',
    icon: '🗄️',
    totalCalls: 3890,
    successRate: 96.8,
    avgResponseTime: 234,
    errorCount: 124,
    lastUsed: '2024-01-20T15:24:00Z',
    status: 'running',
    workspaceUsage: [
      { workspaceId: 'ws-1', workspaceName: '开发环境', callCount: 890 },
      { workspaceId: 'ws-2', workspaceName: '生产环境', callCount: 2980 },
      { workspaceId: 'ws-3', workspaceName: '测试环境', callCount: 20 },
    ],
  },
  {
    mcpId: 'web-search',
    mcpName: 'Web Search',
    icon: '🔍',
    totalCalls: 2150,
    successRate: 99.2,
    avgResponseTime: 412,
    errorCount: 17,
    lastUsed: '2024-01-20T15:22:00Z',
    status: 'running',
    workspaceUsage: [
      { workspaceId: 'ws-1', workspaceName: '开发环境', callCount: 1200 },
      { workspaceId: 'ws-2', workspaceName: '生产环境', callCount: 850 },
      { workspaceId: 'ws-3', workspaceName: '测试环境', callCount: 100 },
    ],
  },
  {
    mcpId: 'github',
    mcpName: 'GitHub',
    icon: '🐙',
    totalCalls: 1890,
    successRate: 94.5,
    avgResponseTime: 289,
    errorCount: 104,
    lastUsed: '2024-01-20T15:20:00Z',
    status: 'stopped',
    workspaceUsage: [
      { workspaceId: 'ws-1', workspaceName: '开发环境', callCount: 650 },
      { workspaceId: 'ws-2', workspaceName: '生产环境', callCount: 1240 },
    ],
  },
  {
    mcpId: 'slack',
    mcpName: 'Slack',
    icon: '💬',
    totalCalls: 890,
    successRate: 97.8,
    avgResponseTime: 178,
    errorCount: 20,
    lastUsed: '2024-01-20T15:18:00Z',
    status: 'running',
    workspaceUsage: [
      { workspaceId: 'ws-1', workspaceName: '开发环境', callCount: 340 },
      { workspaceId: 'ws-2', workspaceName: '生产环境', callCount: 550 },
    ],
  },
]

const generateTimeSeriesData = (hours: number, baseValue: number, variance: number): TimeSeriesData[] => {
  const data: TimeSeriesData[] = []
  const now = new Date()
  for (let i = hours; i >= 0; i--) {
    const timestamp = new Date(now.getTime() - i * 3600000)
    const value = baseValue + (Math.random() - 0.5) * variance * 2
    data.push({
      timestamp: timestamp.toISOString(),
      value: Math.max(0, Math.round(value)),
    })
  }
  return data
}

export const toolCallsHistory: TimeSeriesData[] = generateTimeSeriesData(24, 150, 50)
export const errorRateHistory: TimeSeriesData[] = generateTimeSeriesData(24, 2.5, 1.5)
export const responseTimeHistory: TimeSeriesData[] = generateTimeSeriesData(24, 220, 80)

export const opsDashboardData: OpsDashboardData = {
  systemMetrics,
  workspaceMetrics,
  mcpMetrics,
  toolCallsHistory,
  errorRateHistory,
  responseTimeHistory,
  topWorkspacesByCalls: workspaceMetrics.sort((a, b) => b.totalToolCalls - a.totalToolCalls),
  topMCPsByCalls: mcpMetrics.sort((a, b) => b.totalCalls - a.totalCalls),
}
