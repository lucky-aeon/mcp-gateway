'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'
import { opsDashboardData } from '@/lib/mock-data'
import {
  Cpu,
  HardDrive,
  MemoryStick,
  Network,
  Activity,
  Clock,
  TrendingUp,
  Zap,
  Layers,
  Package,
  AlertTriangle,
  CheckCircle,
  XCircle,
  ArrowUpRight,
  ArrowDownRight,
  Server,
  Gauge,
} from 'lucide-react'
import Link from 'next/link'
import {
  LineChart,
  Line,
  AreaChart,
  Area,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'

const COLORS = ['#6366f1', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4', '#f97316']

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`
}

function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  if (days > 0) return `${days}天 ${hours}小时`
  return `${hours}小时`
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)

  if (diffMins < 1) return '刚刚'
  if (diffMins < 60) return `${diffMins} 分钟前`
  if (diffMins < 1440) return `${Math.floor(diffMins / 60)} 小时前`
  return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })
}

function formatChartTime(timestamp: string): string {
  const date = new Date(timestamp)
  return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

function getStatusIcon(status: string) {
  switch (status) {
    case 'running':
      return <CheckCircle className="h-4 w-4 text-emerald-500" />
    case 'stopped':
      return <XCircle className="h-4 w-4 text-slate-400" />
    case 'error':
      return <AlertTriangle className="h-4 w-4 text-red-500" />
    default:
      return null
  }
}

function getStatusBadge(status: string) {
  switch (status) {
    case 'running':
      return <Badge variant="outline" className="bg-emerald-100 text-emerald-700 border-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400 dark:border-emerald-800">运行中</Badge>
    case 'stopped':
      return <Badge variant="outline" className="bg-slate-100 text-slate-700 border-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:border-slate-700">已停止</Badge>
    case 'error':
      return <Badge variant="outline" className="bg-red-100 text-red-700 border-red-200 dark:bg-red-900/30 dark:text-red-400 dark:border-red-800">错误</Badge>
    default:
      return null
  }
}

export function OpsDashboardPage() {
  const { systemMetrics, workspaceMetrics, mcpMetrics, toolCallsHistory, errorRateHistory, responseTimeHistory, topWorkspacesByCalls, topMCPsByCalls } = opsDashboardData

  // 准备图表数据
  const toolCallsChartData = toolCallsHistory.slice(-24).map(d => ({
    time: formatChartTime(d.timestamp),
    value: d.value,
  }))

  const errorRateChartData = errorRateHistory.slice(-24).map(d => ({
    time: formatChartTime(d.timestamp),
    value: Number(d.value.toFixed(2)),
  }))

  const responseTimeChartData = responseTimeHistory.slice(-24).map(d => ({
    time: formatChartTime(d.timestamp),
    value: d.value,
  }))

  const workspaceCallsData = workspaceMetrics.map(w => ({
    name: w.workspaceName,
    calls: w.totalToolCalls,
    errors: Math.round(w.totalToolCalls * (w.errorRate / 100)),
  }))

  const mcpCallsData = mcpMetrics.map(m => ({
    name: m.mcpName,
    calls: m.totalCalls,
    errors: m.errorCount,
  }))

  const mcpPieData = mcpMetrics.map(m => ({
    name: m.mcpName,
    value: m.totalCalls,
  }))

  const workspacePieData = workspaceMetrics.map(w => ({
    name: w.workspaceName,
    value: w.totalToolCalls,
  }))

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">运维看板</h2>
        <p className="text-muted-foreground">系统运行状态和性能指标监控</p>
      </div>

      {/* System Metrics */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="group hover:shadow-lg transition-all duration-300 border-border/50 bg-gradient-to-br from-card to-card/50">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">CPU 使用率</CardTitle>
            <div className={cn(
              "p-2 rounded-lg transition-all group-hover:scale-110",
              systemMetrics.cpuUsage > 80 ? "bg-red-100 dark:bg-red-900/20" :
              systemMetrics.cpuUsage > 60 ? "bg-amber-100 dark:bg-amber-900/20" :
              "bg-emerald-100 dark:bg-emerald-900/20"
            )}>
              <Cpu className={cn(
                "h-4 w-4",
                systemMetrics.cpuUsage > 80 ? "text-red-600 dark:text-red-400" :
                systemMetrics.cpuUsage > 60 ? "text-amber-600 dark:text-amber-400" :
                "text-emerald-600 dark:text-emerald-400"
              )} />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold tracking-tight">{systemMetrics.cpuUsage.toFixed(1)}%</div>
            <p className="text-xs text-muted-foreground mt-2 flex items-center gap-1">
              {systemMetrics.cpuUsage > 80 ? (
                <><ArrowUpRight className="h-3 w-3 text-red-500" />高负载警告</>
              ) : systemMetrics.cpuUsage > 60 ? (
                <><Activity className="h-3 w-3 text-amber-500" />中等负载</>
              ) : (
                <><CheckCircle className="h-3 w-3 text-emerald-500" />运行正常</>
              )}
            </p>
            <div className="mt-3 h-2 w-full bg-muted rounded-full overflow-hidden">
              <div 
                className={cn(
                  "h-full transition-all duration-500 rounded-full",
                  systemMetrics.cpuUsage > 80 ? "bg-gradient-to-r from-red-500 to-red-600" : 
                  systemMetrics.cpuUsage > 60 ? "bg-gradient-to-r from-amber-500 to-amber-600" : 
                  "bg-gradient-to-r from-emerald-500 to-emerald-600"
                )}
                style={{ width: `${systemMetrics.cpuUsage}%` }}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="group hover:shadow-lg transition-all duration-300 border-border/50 bg-gradient-to-br from-card to-card/50">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">内存使用率</CardTitle>
            <div className={cn(
              "p-2 rounded-lg transition-all group-hover:scale-110",
              systemMetrics.memoryUsage > 80 ? "bg-red-100 dark:bg-red-900/20" :
              systemMetrics.memoryUsage > 60 ? "bg-amber-100 dark:bg-amber-900/20" :
              "bg-emerald-100 dark:bg-emerald-900/20"
            )}>
              <MemoryStick className={cn(
                "h-4 w-4",
                systemMetrics.memoryUsage > 80 ? "text-red-600 dark:text-red-400" :
                systemMetrics.memoryUsage > 60 ? "text-amber-600 dark:text-amber-400" :
                "text-emerald-600 dark:text-emerald-400"
              )} />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold tracking-tight">{systemMetrics.memoryUsage.toFixed(1)}%</div>
            <p className="text-xs text-muted-foreground mt-2 flex items-center gap-1">
              {systemMetrics.memoryUsage > 80 ? (
                <><ArrowUpRight className="h-3 w-3 text-red-500" />高负载警告</>
              ) : systemMetrics.memoryUsage > 60 ? (
                <><Activity className="h-3 w-3 text-amber-500" />中等负载</>
              ) : (
                <><CheckCircle className="h-3 w-3 text-emerald-500" />运行正常</>
              )}
            </p>
            <div className="mt-3 h-2 w-full bg-muted rounded-full overflow-hidden">
              <div 
                className={cn(
                  "h-full transition-all duration-500 rounded-full",
                  systemMetrics.memoryUsage > 80 ? "bg-gradient-to-r from-red-500 to-red-600" : 
                  systemMetrics.memoryUsage > 60 ? "bg-gradient-to-r from-amber-500 to-amber-600" : 
                  "bg-gradient-to-r from-emerald-500 to-emerald-600"
                )}
                style={{ width: `${systemMetrics.memoryUsage}%` }}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="group hover:shadow-lg transition-all duration-300 border-border/50 bg-gradient-to-br from-card to-card/50">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">磁盘使用率</CardTitle>
            <div className={cn(
              "p-2 rounded-lg transition-all group-hover:scale-110",
              systemMetrics.diskUsage > 80 ? "bg-red-100 dark:bg-red-900/20" :
              systemMetrics.diskUsage > 60 ? "bg-amber-100 dark:bg-amber-900/20" :
              "bg-emerald-100 dark:bg-emerald-900/20"
            )}>
              <HardDrive className={cn(
                "h-4 w-4",
                systemMetrics.diskUsage > 80 ? "text-red-600 dark:text-red-400" :
                systemMetrics.diskUsage > 60 ? "text-amber-600 dark:text-amber-400" :
                "text-emerald-600 dark:text-emerald-400"
              )} />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold tracking-tight">{systemMetrics.diskUsage.toFixed(1)}%</div>
            <p className="text-xs text-muted-foreground mt-2 flex items-center gap-1">
              {systemMetrics.diskUsage > 80 ? (
                <><ArrowUpRight className="h-3 w-3 text-red-500" />空间不足</>
              ) : systemMetrics.diskUsage > 60 ? (
                <><Activity className="h-3 w-3 text-amber-500" />空间紧张</>
              ) : (
                <><CheckCircle className="h-3 w-3 text-emerald-500" />空间充足</>
              )}
            </p>
            <div className="mt-3 h-2 w-full bg-muted rounded-full overflow-hidden">
              <div 
                className={cn(
                  "h-full transition-all duration-500 rounded-full",
                  systemMetrics.diskUsage > 80 ? "bg-gradient-to-r from-red-500 to-red-600" : 
                  systemMetrics.diskUsage > 60 ? "bg-gradient-to-r from-amber-500 to-amber-600" : 
                  "bg-gradient-to-r from-emerald-500 to-emerald-600"
                )}
                style={{ width: `${systemMetrics.diskUsage}%` }}
              />
            </div>
          </CardContent>
        </Card>

        <Card className="group hover:shadow-lg transition-all duration-300 border-border/50 bg-gradient-to-br from-card to-card/50">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">系统运行时间</CardTitle>
            <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-900/20 transition-all group-hover:scale-110">
              <Clock className="h-4 w-4 text-blue-600 dark:text-blue-400" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold tracking-tight">{formatUptime(systemMetrics.uptime)}</div>
            <p className="text-xs text-muted-foreground mt-2 flex items-center gap-1">
              <CheckCircle className="h-3 w-3 text-emerald-500" />
              系统稳定运行
            </p>
            <div className="mt-3 flex items-center gap-2">
              <div className="h-2 flex-1 bg-muted rounded-full overflow-hidden">
                <div className="h-full bg-gradient-to-r from-blue-500 to-blue-600 rounded-full" style={{ width: '100%' }} />
              </div>
              <Badge variant="outline" className="text-xs bg-blue-100 text-blue-700 border-blue-200 dark:bg-blue-900/30 dark:text-blue-400 dark:border-blue-800">
                正常
              </Badge>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Network & Request Rate */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card className="group hover:shadow-lg transition-all duration-300 border-border/50 bg-gradient-to-br from-card to-card/50">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">网络入站</CardTitle>
            <div className="p-2 rounded-lg bg-cyan-100 dark:bg-cyan-900/20 transition-all group-hover:scale-110">
              <Network className="h-4 w-4 text-cyan-600 dark:text-cyan-400" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold tracking-tight">{formatBytes(systemMetrics.networkIn)}/s</div>
            <p className="text-xs text-muted-foreground mt-2">入站流量</p>
            <div className="mt-3 h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div className="h-full bg-gradient-to-r from-cyan-500 to-cyan-600 rounded-full animate-pulse" style={{ width: '75%' }} />
            </div>
          </CardContent>
        </Card>

        <Card className="group hover:shadow-lg transition-all duration-300 border-border/50 bg-gradient-to-br from-card to-card/50">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">网络出站</CardTitle>
            <div className="p-2 rounded-lg bg-purple-100 dark:bg-purple-900/20 transition-all group-hover:scale-110">
              <Network className="h-4 w-4 text-purple-600 dark:text-purple-400 rotate-180" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold tracking-tight">{formatBytes(systemMetrics.networkOut)}/s</div>
            <p className="text-xs text-muted-foreground mt-2">出站流量</p>
            <div className="mt-3 h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div className="h-full bg-gradient-to-r from-purple-500 to-purple-600 rounded-full animate-pulse" style={{ width: '50%' }} />
            </div>
          </CardContent>
        </Card>

        <Card className="group hover:shadow-lg transition-all duration-300 border-border/50 bg-gradient-to-br from-card to-card/50">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">请求速率</CardTitle>
            <div className="p-2 rounded-lg bg-violet-100 dark:bg-violet-900/20 transition-all group-hover:scale-110">
              <Activity className="h-4 w-4 text-violet-600 dark:text-violet-400" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold tracking-tight">{systemMetrics.requestRate}/s</div>
            <p className="text-xs text-muted-foreground mt-2">当前请求</p>
            <div className="mt-3 h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div className="h-full bg-gradient-to-r from-violet-500 to-violet-600 rounded-full animate-pulse" style={{ width: '62%' }} />
            </div>
          </CardContent>
        </Card>

        <Card className="group hover:shadow-lg transition-all duration-300 border-border/50 bg-gradient-to-br from-card to-card/50">
          <CardHeader className="flex flex-row items-center justify-between pb-3">
            <CardTitle className="text-sm font-medium">活跃会话</CardTitle>
            <div className="p-2 rounded-lg bg-indigo-100 dark:bg-indigo-900/20 transition-all group-hover:scale-110">
              <Layers className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
            </div>
          </CardHeader>
          <CardContent>
            <div className="text-3xl font-bold tracking-tight">
              {workspaceMetrics.reduce((sum, w) => sum + w.activeSessions, 0)}
            </div>
            <p className="text-xs text-muted-foreground mt-2">
              总计 {workspaceMetrics.reduce((sum, w) => sum + w.totalSessions, 0)} 会话
            </p>
            <div className="mt-3 h-1.5 w-full bg-muted rounded-full overflow-hidden">
              <div className="h-full bg-gradient-to-r from-indigo-500 to-indigo-600 rounded-full animate-pulse" style={{ width: `${(workspaceMetrics.reduce((sum, w) => sum + w.activeSessions, 0) / workspaceMetrics.reduce((sum, w) => sum + w.totalSessions, 0)) * 100}%` }} />
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Time Series Charts */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <div className="p-1.5 rounded-lg bg-indigo-100 dark:bg-indigo-900/20">
                <Activity className="h-4 w-4 text-indigo-600 dark:text-indigo-400" />
              </div>
              工具调用趋势
            </CardTitle>
            <CardDescription>过去24小时的工具调用数量变化</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={toolCallsChartData}>
                <defs>
                  <linearGradient id="colorCalls" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#6366f1" stopOpacity={0.8}/>
                    <stop offset="95%" stopColor="#6366f1" stopOpacity={0.1}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted/50" />
                <XAxis 
                  dataKey="time" 
                  className="text-xs"
                  tick={{ fontSize: 12 }}
                  stroke="hsl(var(--muted-foreground))"
                />
                <YAxis 
                  className="text-xs"
                  tick={{ fontSize: 12 }}
                  stroke="hsl(var(--muted-foreground))"
                />
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: 'hsl(var(--card))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                    boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
                  }}
                />
                <Area 
                  type="monotone" 
                  dataKey="value" 
                  stroke="#6366f1" 
                  strokeWidth={2}
                  fillOpacity={1} 
                  fill="url(#colorCalls)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <div className="p-1.5 rounded-lg bg-red-100 dark:bg-red-900/20">
                <AlertTriangle className="h-4 w-4 text-red-600 dark:text-red-400" />
              </div>
              错误率趋势
            </CardTitle>
            <CardDescription>过去24小时的错误率变化</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <LineChart data={errorRateChartData}>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted/50" />
                <XAxis 
                  dataKey="time" 
                  className="text-xs"
                  tick={{ fontSize: 12 }}
                  stroke="hsl(var(--muted-foreground))"
                />
                <YAxis 
                  className="text-xs"
                  tick={{ fontSize: 12 }}
                  domain={[0, 5]}
                  stroke="hsl(var(--muted-foreground))"
                />
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: 'hsl(var(--card))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                    boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
                  }}
                  formatter={(value) => `${value}%`}
                />
                <Line 
                  type="monotone" 
                  dataKey="value" 
                  stroke="#ef4444" 
                  strokeWidth={2}
                  dot={{ fill: '#ef4444', strokeWidth: 2, r: 4 }}
                  activeDot={{ r: 6 }}
                />
              </LineChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <div className="p-1.5 rounded-lg bg-emerald-100 dark:bg-emerald-900/20">
                <Clock className="h-4 w-4 text-emerald-600 dark:text-emerald-400" />
              </div>
              响应时间趋势
            </CardTitle>
            <CardDescription>过去24小时的平均响应时间变化</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <AreaChart data={responseTimeChartData}>
                <defs>
                  <linearGradient id="colorResponse" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#10b981" stopOpacity={0.8}/>
                    <stop offset="95%" stopColor="#10b981" stopOpacity={0.1}/>
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" className="stroke-muted/50" />
                <XAxis 
                  dataKey="time" 
                  className="text-xs"
                  tick={{ fontSize: 12 }}
                  stroke="hsl(var(--muted-foreground))"
                />
                <YAxis 
                  className="text-xs"
                  tick={{ fontSize: 12 }}
                  stroke="hsl(var(--muted-foreground))"
                />
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: 'hsl(var(--card))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                    boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
                  }}
                  formatter={(value) => `${value}ms`}
                />
                <Area 
                  type="monotone" 
                  dataKey="value" 
                  stroke="#10b981" 
                  strokeWidth={2}
                  fillOpacity={1} 
                  fill="url(#colorResponse)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <div className="p-1.5 rounded-lg bg-violet-100 dark:bg-violet-900/20">
                <PieChart className="h-4 w-4 text-violet-600 dark:text-violet-400" />
              </div>
              MCP 调用分布
            </CardTitle>
            <CardDescription>各 MCP 服务的调用占比</CardDescription>
          </CardHeader>
          <CardContent>
            <ResponsiveContainer width="100%" height={300}>
              <PieChart>
                <Pie
                  data={mcpPieData}
                  cx="50%"
                  cy="50%"
                  labelLine={false}
                  label={({ name, percent }) => `${name} ${(percent * 100).toFixed(0)}%`}
                  outerRadius={80}
                  fill="#8884d8"
                  dataKey="value"
                  strokeWidth={2}
                >
                  {mcpPieData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} stroke="hsl(var(--card))" strokeWidth={2} />
                  ))}
                </Pie>
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: 'hsl(var(--card))',
                    border: '1px solid hsl(var(--border))',
                    borderRadius: '8px',
                    boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
                  }}
                />
              </PieChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      {/* Workspace & MCP Metrics with Charts */}
      <Tabs defaultValue="workspaces" className="space-y-4">
        <TabsList className="bg-muted/50 p-1">
          <TabsTrigger value="workspaces" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">工作空间详情</TabsTrigger>
          <TabsTrigger value="mcps" className="data-[state=active]:bg-background data-[state=active]:shadow-sm">MCP 详情</TabsTrigger>
        </TabsList>

        <TabsContent value="workspaces" className="space-y-6">
          <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <div className="p-1.5 rounded-lg bg-blue-100 dark:bg-blue-900/20">
                  <Layers className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                </div>
                工作空间调用对比
              </CardTitle>
              <CardDescription>各工作空间的工具调用和错误数量对比</CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={350}>
                <BarChart data={workspaceCallsData}>
                  <CartesianGrid strokeDasharray="3 3" className="stroke-muted/50" />
                  <XAxis 
                    dataKey="name" 
                    className="text-xs"
                    tick={{ fontSize: 12 }}
                    stroke="hsl(var(--muted-foreground))"
                  />
                  <YAxis 
                    className="text-xs"
                    tick={{ fontSize: 12 }}
                    stroke="hsl(var(--muted-foreground))"
                  />
                  <Tooltip 
                    contentStyle={{ 
                      backgroundColor: 'hsl(var(--card))',
                      border: '1px solid hsl(var(--border))',
                      borderRadius: '8px',
                      boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
                    }}
                  />
                  <Legend />
                  <Bar dataKey="calls" name="调用次数" fill="#6366f1" radius={[4, 4, 0, 0]} />
                  <Bar dataKey="errors" name="错误次数" fill="#ef4444" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <div className="p-1.5 rounded-lg bg-cyan-100 dark:bg-cyan-900/20">
                  <Server className="h-4 w-4 text-cyan-600 dark:text-cyan-400" />
                </div>
                工作空间指标详情
              </CardTitle>
              <CardDescription>各工作空间的详细运行状态和性能数据</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {workspaceMetrics.map((workspace) => (
                  <Link
                    key={workspace.workspaceId}
                    href={`/workspaces/${workspace.workspaceId}`}
                    className="group block"
                  >
                    <div className="flex items-center justify-between rounded-xl border border-border/50 bg-gradient-to-r from-muted/30 to-muted/50 p-4 transition-all hover:border-primary/50 hover:shadow-md hover:bg-gradient-to-r hover:from-primary/5 hover:to-primary/10">
                      <div className="flex items-center gap-4">
                        <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-primary/20 to-primary/5 ring-1 ring-primary/20 transition-all group-hover:scale-110 group-hover:ring-primary/40">
                          <Layers className="h-6 w-6 text-primary" />
                        </div>
                        <div>
                          <p className="font-semibold group-hover:text-primary transition-colors">{workspace.workspaceName}</p>
                          <div className="flex items-center gap-3 text-sm text-muted-foreground mt-1.5">
                            <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400">
                              <Activity className="h-3 w-3" />
                              {workspace.activeSessions}/{workspace.totalSessions}
                            </span>
                            <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-violet-100 dark:bg-violet-900/20 text-violet-700 dark:text-violet-400">
                              <Package className="h-3 w-3" />
                              {workspace.mcpCount}
                            </span>
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-6">
                        <div className="text-right">
                          <p className="text-sm font-semibold">{workspace.totalToolCalls.toLocaleString()}</p>
                          <p className="text-xs text-muted-foreground">工具调用</p>
                        </div>
                        <div className="text-right">
                          <p className="text-sm font-semibold">{workspace.avgResponseTime}ms</p>
                          <p className="text-xs text-muted-foreground">平均响应</p>
                        </div>
                        <div className="text-right">
                          <p className={cn(
                            "text-sm font-semibold",
                            workspace.errorRate > 3 ? "text-red-600" : workspace.errorRate > 1 ? "text-amber-600" : "text-emerald-600"
                          )}>
                            {workspace.errorRate.toFixed(1)}%
                          </p>
                          <p className="text-xs text-muted-foreground">错误率</p>
                        </div>
                        <Badge variant="outline" className="bg-slate-100 text-slate-700 border-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:border-slate-700">
                          {formatTime(workspace.lastActivity)}
                        </Badge>
                      </div>
                    </div>
                  </Link>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="mcps" className="space-y-6">
          <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <div className="p-1.5 rounded-lg bg-amber-100 dark:bg-amber-900/20">
                  <Zap className="h-4 w-4 text-amber-600 dark:text-amber-400" />
                </div>
                MCP 调用对比
              </CardTitle>
              <CardDescription>各 MCP 服务的调用和错误数量对比</CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={350}>
                <BarChart data={mcpCallsData} layout="vertical">
                  <CartesianGrid strokeDasharray="3 3" className="stroke-muted/50" />
                  <XAxis 
                    type="number"
                    className="text-xs"
                    tick={{ fontSize: 12 }}
                    stroke="hsl(var(--muted-foreground))"
                  />
                  <YAxis 
                    dataKey="name" 
                    type="category"
                    width={80}
                    className="text-xs"
                    tick={{ fontSize: 12 }}
                    stroke="hsl(var(--muted-foreground))"
                  />
                  <Tooltip 
                    contentStyle={{ 
                      backgroundColor: 'hsl(var(--card))',
                      border: '1px solid hsl(var(--border))',
                      borderRadius: '8px',
                      boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
                    }}
                  />
                  <Legend />
                  <Bar dataKey="calls" name="调用次数" fill="#6366f1" radius={[0, 4, 4, 0]} />
                  <Bar dataKey="errors" name="错误次数" fill="#ef4444" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
            <CardHeader>
              <CardTitle className="text-lg flex items-center gap-2">
                <div className="p-1.5 rounded-lg bg-pink-100 dark:bg-pink-900/20">
                  <Gauge className="h-4 w-4 text-pink-600 dark:text-pink-400" />
                </div>
                MCP 指标详情
              </CardTitle>
              <CardDescription>各 MCP 服务的运行状态和使用情况</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {mcpMetrics.map((mcp) => (
                  <div
                    key={mcp.mcpId}
                    className="flex items-center justify-between rounded-xl border border-border/50 bg-gradient-to-r from-muted/30 to-muted/50 p-4 transition-all hover:border-primary/50 hover:shadow-md hover:bg-gradient-to-r hover:from-primary/5 hover:to-primary/10"
                  >
                    <div className="flex items-center gap-4">
                      <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-primary/20 to-primary/5 ring-1 ring-primary/20 transition-all hover:scale-110 hover:ring-primary/40 text-2xl">
                        {mcp.icon}
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <p className="font-semibold">{mcp.mcpName}</p>
                          {getStatusIcon(mcp.status)}
                        </div>
                        <div className="flex items-center gap-3 text-sm text-muted-foreground mt-1.5">
                          <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-blue-100 dark:bg-blue-900/20 text-blue-700 dark:text-blue-400">
                            <Activity className="h-3 w-3" />
                            {mcp.totalCalls.toLocaleString()}
                          </span>
                          <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-full bg-emerald-100 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-400">
                            <CheckCircle className="h-3 w-3" />
                            {mcp.successRate.toFixed(1)}%
                          </span>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-6">
                      <div className="text-right">
                        <p className="text-sm font-semibold">{mcp.avgResponseTime}ms</p>
                        <p className="text-xs text-muted-foreground">平均响应</p>
                      </div>
                      <div className="text-right">
                        <p className={cn(
                          "text-sm font-semibold",
                          mcp.errorCount > 50 ? "text-red-600" : mcp.errorCount > 20 ? "text-amber-600" : "text-emerald-600"
                        )}>
                          {mcp.errorCount}
                        </p>
                        <p className="text-xs text-muted-foreground">错误数</p>
                      </div>
                      {getStatusBadge(mcp.status)}
                      <Badge variant="outline" className="bg-slate-100 text-slate-700 border-slate-200 dark:bg-slate-800 dark:text-slate-400 dark:border-slate-700">
                        {formatTime(mcp.lastUsed)}
                      </Badge>
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Top Rankings */}
      <div className="grid gap-6 lg:grid-cols-2">
        <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <div className="p-1.5 rounded-lg bg-amber-100 dark:bg-amber-900/20">
                <TrendingUp className="h-4 w-4 text-amber-600 dark:text-amber-400" />
              </div>
              工具调用排行
            </CardTitle>
            <CardDescription>按工具调用次数排序的工作空间</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {topWorkspacesByCalls.map((workspace, index) => (
                <div
                  key={workspace.workspaceId}
                  className="flex items-center justify-between p-3 rounded-xl bg-gradient-to-r from-muted/30 to-muted/50 hover:from-amber-50/50 hover:to-amber-100/20 dark:hover:from-amber-900/10 dark:hover:to-amber-800/10 transition-all"
                >
                  <div className="flex items-center gap-3">
                    <div className={cn(
                      "flex h-9 w-9 items-center justify-center rounded-lg text-sm font-bold shadow-sm",
                      index === 0 ? "bg-gradient-to-br from-amber-400 to-amber-500 text-white" :
                      index === 1 ? "bg-gradient-to-br from-slate-300 to-slate-400 text-white" :
                      index === 2 ? "bg-gradient-to-br from-orange-400 to-orange-500 text-white" :
                      "bg-muted text-muted-foreground"
                    )}>
                      {index + 1}
                    </div>
                    <div>
                      <p className="font-semibold text-sm">{workspace.workspaceName}</p>
                      <p className="text-xs text-muted-foreground">{workspace.activeSessions} 活跃会话</p>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold text-sm">{workspace.totalToolCalls.toLocaleString()}</p>
                    <p className="text-xs text-muted-foreground">调用次数</p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        <Card className="hover:shadow-lg transition-all duration-300 border-border/50">
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <div className="p-1.5 rounded-lg bg-orange-100 dark:bg-orange-900/20">
                <Zap className="h-4 w-4 text-orange-600 dark:text-orange-400" />
              </div>
              MCP 使用排行
            </CardTitle>
            <CardDescription>按调用次数排序的 MCP 服务</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {topMCPsByCalls.map((mcp, index) => (
                <div
                  key={mcp.mcpId}
                  className="flex items-center justify-between p-3 rounded-xl bg-gradient-to-r from-muted/30 to-muted/50 hover:from-orange-50/50 hover:to-orange-100/20 dark:hover:from-orange-900/10 dark:hover:to-orange-800/10 transition-all"
                >
                  <div className="flex items-center gap-3">
                    <div className={cn(
                      "flex h-9 w-9 items-center justify-center rounded-lg text-sm font-bold shadow-sm",
                      index === 0 ? "bg-gradient-to-br from-amber-400 to-amber-500 text-white" :
                      index === 1 ? "bg-gradient-to-br from-slate-300 to-slate-400 text-white" :
                      index === 2 ? "bg-gradient-to-br from-orange-400 to-orange-500 text-white" :
                      "bg-muted text-muted-foreground"
                    )}>
                      {index + 1}
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-2xl">{mcp.icon}</span>
                      <div>
                        <p className="font-semibold text-sm">{mcp.mcpName}</p>
                        <p className="text-xs text-muted-foreground">{mcp.workspaceUsage.length} 个工作空间</p>
                      </div>
                    </div>
                  </div>
                  <div className="text-right">
                    <p className="font-semibold text-sm">{mcp.totalCalls.toLocaleString()}</p>
                    <p className="text-xs text-muted-foreground">调用次数</p>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
