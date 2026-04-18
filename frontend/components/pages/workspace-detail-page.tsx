'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { getWorkspaceDetail, marketMCPs } from '@/lib/mock-data'
import {
  ArrowLeft,
  Plus,
  Package,
  Users,
  FileText,
  Settings,
  Play,
  Square,
  Trash2,
  MoreHorizontal,
  AlertCircle,
  CheckCircle,
  Clock,
  Filter,
  Layers,
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import Link from 'next/link'

interface WorkspaceDetailPageProps {
  workspaceId: string
}

function formatDateTime(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function formatDuration(start: string, end?: string): string {
  const startDate = new Date(start)
  const endDate = end ? new Date(end) : new Date()
  const diff = endDate.getTime() - startDate.getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 60) return `${minutes}分钟`
  const hours = Math.floor(minutes / 60)
  return `${hours}小时${minutes % 60}分钟`
}

const logLevelStyles: Record<string, string> = {
  info: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  warn: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
  error: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  debug: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400',
}

const logLevelIcons: Record<string, React.ReactNode> = {
  info: <CheckCircle className="h-4 w-4 text-blue-500" />,
  warn: <AlertCircle className="h-4 w-4 text-amber-500" />,
  error: <AlertCircle className="h-4 w-4 text-red-500" />,
  debug: <Clock className="h-4 w-4 text-slate-500" />,
}

export function WorkspaceDetailPage({ workspaceId }: WorkspaceDetailPageProps) {
  const [activeTab, setActiveTab] = useState('mcps')
  const [logFilter, setLogFilter] = useState('all')
  const [isAddMCPOpen, setIsAddMCPOpen] = useState(false)

  const workspace = getWorkspaceDetail(workspaceId)

  if (!workspace) {
    return (
      <Card className="border-dashed">
        <CardContent className="flex flex-col items-center justify-center py-16 text-center">
          <div className="flex h-14 w-14 items-center justify-center rounded-full bg-muted">
            <Layers className="h-7 w-7 text-muted-foreground" />
          </div>
          <h3 className="mt-4 text-lg font-semibold">工作空间不存在</h3>
          <p className="mt-2 text-muted-foreground">该工作空间可能已被删除或 ID 无效</p>
          <Button className="mt-6" asChild>
            <Link href="/workspaces">返回列表</Link>
          </Button>
        </CardContent>
      </Card>
    )
  }

  const filteredLogs =
    logFilter === 'all'
      ? workspace.logs
      : workspace.logs.filter((log) => log.level === logFilter)

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-start gap-4">
        <Button variant="ghost" size="icon" className="mt-1" asChild>
          <Link href="/workspaces">
            <ArrowLeft className="h-5 w-5" />
          </Link>
        </Button>
        <div className="flex-1 space-y-1">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-gradient-to-br from-primary/10 to-primary/5 ring-1 ring-primary/10">
              <Layers className="h-6 w-6 text-primary" />
            </div>
            <div>
              <h1 className="text-2xl font-semibold tracking-tight">{workspace.name}</h1>
              <p className="text-muted-foreground">{workspace.description}</p>
            </div>
          </div>
        </div>
        <Badge
          className={cn(
            'mt-2',
            workspace.status === 'active' 
              ? 'bg-emerald-100 text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-900/30 dark:text-emerald-400' 
              : ''
          )}
        >
          {workspace.status === 'active' ? '运行中' : '已停止'}
        </Badge>
      </div>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
        <TabsList className="inline-flex h-auto gap-1 rounded-xl bg-muted p-1">
          <TabsTrigger 
            value="mcps" 
            className="gap-2 rounded-lg px-4 py-2 data-[state=active]:bg-background data-[state=active]:shadow-sm"
          >
            <Package className="h-4 w-4" />
            MCPs
          </TabsTrigger>
          <TabsTrigger 
            value="sessions" 
            className="gap-2 rounded-lg px-4 py-2 data-[state=active]:bg-background data-[state=active]:shadow-sm"
          >
            <Users className="h-4 w-4" />
            会话
          </TabsTrigger>
          <TabsTrigger 
            value="logs" 
            className="gap-2 rounded-lg px-4 py-2 data-[state=active]:bg-background data-[state=active]:shadow-sm"
          >
            <FileText className="h-4 w-4" />
            日志
          </TabsTrigger>
          <TabsTrigger 
            value="settings" 
            className="gap-2 rounded-lg px-4 py-2 data-[state=active]:bg-background data-[state=active]:shadow-sm"
          >
            <Settings className="h-4 w-4" />
            设置
          </TabsTrigger>
        </TabsList>

        {/* MCPs Tab */}
        <TabsContent value="mcps" className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold">已添加的 MCP</h2>
              <p className="text-sm text-muted-foreground">管理此工作空间中的 MCP 服务</p>
            </div>
            <Dialog open={isAddMCPOpen} onOpenChange={setIsAddMCPOpen}>
              <DialogTrigger asChild>
                <Button>
                  <Plus className="mr-2 h-4 w-4" />
                  添加 MCP
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-2xl">
                <DialogHeader>
                  <DialogTitle>添加 MCP</DialogTitle>
                  <DialogDescription>
                    选择要添加到此工作空间的 MCP
                  </DialogDescription>
                </DialogHeader>
                <div className="grid gap-3 py-4 max-h-[400px] overflow-auto">
                  {marketMCPs.map((mcp) => (
                    <div
                      key={mcp.id}
                      className="flex items-center justify-between rounded-xl border bg-muted/30 p-4 transition-colors hover:bg-muted/50"
                    >
                      <div className="flex items-center gap-3">
                        <span className="text-2xl">{mcp.icon}</span>
                        <div>
                          <p className="font-medium">{mcp.name}</p>
                          <p className="text-sm text-muted-foreground line-clamp-1">
                            {mcp.description}
                          </p>
                        </div>
                      </div>
                      <Button size="sm">添加</Button>
                    </div>
                  ))}
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setIsAddMCPOpen(false)}>
                    关闭
                  </Button>
                </DialogFooter>
              </DialogContent>
            </Dialog>
          </div>
          
          <div className="grid gap-4 sm:grid-cols-2">
            {workspace.mcps.map((mcp) => (
              <Card key={mcp.id} className="group">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <span className="text-2xl">{mcp.icon}</span>
                      <div>
                        <CardTitle className="text-base">{mcp.name}</CardTitle>
                        <Badge
                          variant="outline"
                          className={cn(
                            'mt-1.5 border-0',
                            mcp.status === 'running'
                              ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                              : mcp.status === 'stopped'
                              ? 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                              : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                          )}
                        >
                          {mcp.status === 'running'
                            ? '运行中'
                            : mcp.status === 'stopped'
                            ? '已停止'
                            : '错误'}
                        </Badge>
                      </div>
                    </div>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        {mcp.status === 'running' ? (
                          <DropdownMenuItem>
                            <Square className="mr-2 h-4 w-4" />
                            停止
                          </DropdownMenuItem>
                        ) : (
                          <DropdownMenuItem>
                            <Play className="mr-2 h-4 w-4" />
                            启动
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuItem>
                          <Settings className="mr-2 h-4 w-4" />
                          配置
                        </DropdownMenuItem>
                        <DropdownMenuItem className="text-destructive focus:text-destructive">
                          <Trash2 className="mr-2 h-4 w-4" />
                          移除
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-2">已启用工具</p>
                    <div className="flex flex-wrap gap-1.5">
                      {mcp.enabledTools.map((tool) => (
                        <Badge key={tool} variant="secondary" className="text-xs font-normal">
                          {tool}
                        </Badge>
                      ))}
                    </div>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    添加于 {formatDateTime(mcp.addedAt)}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </TabsContent>

        {/* Sessions Tab */}
        <TabsContent value="sessions" className="space-y-4">
          <div>
            <h2 className="text-lg font-semibold">会话管理</h2>
            <p className="text-sm text-muted-foreground">查看和管理当前连接的会话</p>
          </div>
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead>会话 ID</TableHead>
                    <TableHead>客户端</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="text-right">工具调用</TableHead>
                    <TableHead>持续时间</TableHead>
                    <TableHead>最后活动</TableHead>
                    <TableHead className="w-[80px]"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {workspace.sessions.map((session) => (
                    <TableRow key={session.id}>
                      <TableCell className="font-mono text-sm">
                        {session.id}
                      </TableCell>
                      <TableCell className="font-mono text-sm">
                        {session.clientId}
                      </TableCell>
                      <TableCell>
                        <Badge
                          className={cn(
                            'border-0',
                            session.status === 'active'
                              ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                              : session.status === 'ended'
                              ? 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                              : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                          )}
                        >
                          {session.status === 'active'
                            ? '活跃'
                            : session.status === 'ended'
                            ? '已结束'
                            : '错误'}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right font-medium">{session.toolCalls}</TableCell>
                      <TableCell className="text-muted-foreground">
                        {formatDuration(session.startedAt, session.endedAt)}
                      </TableCell>
                      <TableCell className="text-muted-foreground">{formatDateTime(session.lastActivity)}</TableCell>
                      <TableCell>
                        {session.status === 'active' && (
                          <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive">
                            终止
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        {/* Logs Tab */}
        <TabsContent value="logs" className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold">操作日志</h2>
              <p className="text-sm text-muted-foreground">查看工作空间的运行日志</p>
            </div>
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <Select value={logFilter} onValueChange={setLogFilter}>
                <SelectTrigger className="w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  <SelectItem value="info">Info</SelectItem>
                  <SelectItem value="warn">Warning</SelectItem>
                  <SelectItem value="error">Error</SelectItem>
                  <SelectItem value="debug">Debug</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          
          <Card>
            <CardContent className="p-4 space-y-2">
              {filteredLogs.map((log) => (
                <div
                  key={log.id}
                  className="flex items-start gap-3 rounded-xl border bg-muted/30 p-4"
                >
                  <div className="mt-0.5">{logLevelIcons[log.level]}</div>
                  <div className="flex-1 min-w-0 space-y-1">
                    <div className="flex items-center gap-2 flex-wrap">
                      <Badge
                        className={cn('text-xs border-0', logLevelStyles[log.level])}
                      >
                        {log.level.toUpperCase()}
                      </Badge>
                      <span className="text-sm font-medium">{log.source}</span>
                      <span className="text-xs text-muted-foreground">
                        {formatDateTime(log.timestamp)}
                      </span>
                    </div>
                    <p className="text-sm">{log.message}</p>
                    {log.metadata && (
                      <pre className="mt-2 rounded-lg bg-muted p-3 text-xs overflow-auto font-mono">
                        {JSON.stringify(log.metadata, null, 2)}
                      </pre>
                    )}
                  </div>
                </div>
              ))}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Settings Tab */}
        <TabsContent value="settings" className="space-y-6">
          <div>
            <h2 className="text-lg font-semibold">工作空间设置</h2>
            <p className="text-sm text-muted-foreground">配置工作空间的运行参数</p>
          </div>
          
          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle>基本设置</CardTitle>
                <CardDescription>工作空间的基本配置信息</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="ws-name">名称</Label>
                  <Input id="ws-name" defaultValue={workspace.name} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="ws-desc">描述</Label>
                  <Input id="ws-desc" defaultValue={workspace.description} />
                </div>
                <Button>保存更改</Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>会话设置</CardTitle>
                <CardDescription>会话相关的配置选项</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="max-sessions">最大会话数</Label>
                    <Input
                      id="max-sessions"
                      type="number"
                      defaultValue={workspace.settings.maxSessions}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="session-timeout">会话超时（秒）</Label>
                    <Input
                      id="session-timeout"
                      type="number"
                      defaultValue={workspace.settings.sessionTimeout}
                    />
                  </div>
                </div>
                <Button>保存更改</Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle>日志设置</CardTitle>
                <CardDescription>日志保留和级别设置</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="log-retention">日志保留天数</Label>
                  <Input
                    id="log-retention"
                    type="number"
                    defaultValue={workspace.settings.logRetention}
                  />
                </div>
                <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                  <div className="space-y-0.5">
                    <Label className="text-base">启用调试日志</Label>
                    <p className="text-sm text-muted-foreground">记录详细的调试信息</p>
                  </div>
                  <Switch />
                </div>
                <Button>保存更改</Button>
              </CardContent>
            </Card>

            <Card className="border-destructive/50">
              <CardHeader>
                <CardTitle className="text-destructive">危险区域</CardTitle>
                <CardDescription>不可逆的操作，请谨慎执行</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="flex items-center justify-between rounded-xl border border-destructive/20 bg-destructive/5 p-4">
                  <div className="space-y-0.5">
                    <p className="font-medium">删除工作空间</p>
                    <p className="text-sm text-muted-foreground">永久删除此工作空间及所有相关数据</p>
                  </div>
                  <Button variant="destructive" size="sm">删除</Button>
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  )
}
