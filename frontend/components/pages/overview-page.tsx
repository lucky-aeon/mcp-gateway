'use client'

import Link from 'next/link'
import { Activity, ArrowRight, Clock, Layers, Package, TrendingUp, Users, Wrench } from 'lucide-react'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import { useGatewaySWR, type OverviewStats, type Workspace } from '@/lib/gateway-api'

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

const activityTypeLabels: Record<string, string> = {
  'session.created': '会话创建',
  'workspace.created': '工作空间创建',
  'mcp.deployed': 'MCP 部署',
}

const activityTypeStyles: Record<string, string> = {
  'session.created': 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
  'workspace.created': 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  'mcp.deployed': 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400',
}

export function OverviewPage() {
  const { data: stats } = useGatewaySWR<OverviewStats>('/api/v1/stats/overview')
  const { data: workspacesData } = useGatewaySWR<{ items: Workspace[] }>('/api/v1/workspaces')

  const workspaces = workspacesData?.items || []
  const cards = [
    {
      label: '工作空间',
      value: stats?.workspaces_count ?? 0,
      icon: Layers,
      color: 'text-blue-600 dark:text-blue-400',
      bgColor: 'bg-blue-100 dark:bg-blue-900/30',
      href: '/workspaces',
    },
    {
      label: '活跃会话',
      value: stats?.active_sessions ?? 0,
      icon: Users,
      color: 'text-emerald-600 dark:text-emerald-400',
      bgColor: 'bg-emerald-100 dark:bg-emerald-900/30',
      href: '/workspaces',
    },
    {
      label: '运行中 MCP',
      value: stats?.running_mcps ?? 0,
      icon: Package,
      color: 'text-violet-600 dark:text-violet-400',
      bgColor: 'bg-violet-100 dark:bg-violet-900/30',
      href: '/installed',
    },
    {
      label: '24h 异常 MCP',
      value: stats?.failed_mcps_24h ?? 0,
      icon: Wrench,
      color: 'text-amber-600 dark:text-amber-400',
      bgColor: 'bg-amber-100 dark:bg-amber-900/30',
      href: '/ops-dashboard',
    },
  ]

  return (
    <div className="space-y-8">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">欢迎回来</h2>
        <p className="text-muted-foreground">以下是您的 Gateway 运行概览</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {cards.map((stat) => {
          const Icon = stat.icon
          return (
            <Link key={stat.label} href={stat.href}>
              <Card className="group relative overflow-hidden transition-all hover:border-primary/20 hover:shadow-md">
                <CardContent className="p-6">
                  <div className="flex items-start justify-between">
                    <div className="space-y-2">
                      <p className="text-sm font-medium text-muted-foreground">{stat.label}</p>
                      <div className="flex items-baseline gap-2">
                        <p className="text-3xl font-bold tracking-tight">{stat.value.toLocaleString()}</p>
                        <span className="flex items-center text-xs font-medium text-emerald-600 dark:text-emerald-400">
                          <TrendingUp className="mr-0.5 h-3 w-3" />
                          实时
                        </span>
                      </div>
                    </div>
                    <div className={cn('rounded-xl p-3 transition-transform group-hover:scale-110', stat.bgColor)}>
                      <Icon className={cn('h-5 w-5', stat.color)} />
                    </div>
                  </div>
                </CardContent>
              </Card>
            </Link>
          )
        })}
      </div>

      <div className="grid gap-6 lg:grid-cols-5">
        <Card className="lg:col-span-3">
          <CardHeader className="flex flex-row items-center justify-between pb-4">
            <div>
              <CardTitle className="text-lg">工作空间</CardTitle>
              <CardDescription>最近活跃的工作空间</CardDescription>
            </div>
            <Button variant="ghost" size="sm" asChild>
              <Link href="/workspaces" className="gap-1">
                查看全部
                <ArrowRight className="h-4 w-4" />
              </Link>
            </Button>
          </CardHeader>
          <CardContent className="space-y-3">
            {workspaces.slice(0, 4).map((workspace) => (
              <Link
                key={workspace.id}
                href={`/workspaces/${workspace.id}`}
                className="group flex items-center justify-between rounded-xl border border-transparent bg-muted/50 p-4 transition-all hover:border-border hover:bg-muted"
              >
                <div className="flex items-center gap-4">
                  <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-primary/10 to-primary/5 ring-1 ring-primary/10">
                    <Layers className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <p className="font-medium transition-colors group-hover:text-primary">{workspace.name}</p>
                    <div className="flex items-center gap-3 text-sm text-muted-foreground">
                      <span>{workspace.mcp_count} MCPs</span>
                      <span>{workspace.session_count} 会话</span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Badge
                    variant="outline"
                    className={cn(
                      'border-0',
                      workspace.status === 'running'
                        ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                        : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                    )}
                  >
                    {workspace.status === 'running' ? '运行中' : '已停止'}
                  </Badge>
                  <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 transition-all group-hover:translate-x-1 group-hover:opacity-100" />
                </div>
              </Link>
            ))}
            {workspaces.length === 0 && <p className="text-sm text-muted-foreground">还没有工作空间，先创建一个开始。</p>}
          </CardContent>
        </Card>

        <Card className="lg:col-span-2">
          <CardHeader className="pb-4">
            <div className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-muted-foreground" />
              <CardTitle className="text-lg">最近活动</CardTitle>
            </div>
            <CardDescription>系统最近的操作记录</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {(stats?.recent_activity || []).slice(0, 8).map((activity) => (
                <div key={`${activity.type}-${activity.at}-${activity.message}`} className="flex items-start gap-3">
                  <div className="mt-0.5">
                    <Badge
                      variant="outline"
                      className={cn(
                        'border-0 text-xs font-medium',
                        activityTypeStyles[activity.type] || 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                      )}
                    >
                      {activityTypeLabels[activity.type] || activity.type}
                    </Badge>
                  </div>
                  <div className="min-w-0 flex-1 space-y-1">
                    <p className="text-sm leading-tight">{activity.message || '系统事件'}</p>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span>{activity.workspace_name || '系统'}</span>
                      <span>·</span>
                      <span className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {formatTime(activity.at)}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
              {!stats?.recent_activity?.length && <p className="text-sm text-muted-foreground">暂无最近活动。</p>}
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-lg">快速操作</CardTitle>
          <CardDescription>常用功能入口</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Button
              variant="outline"
              className="h-auto flex-col items-start gap-2 p-4 hover:bg-muted/50 hover:border-primary/20"
              asChild
            >
              <Link href="/workspaces">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30">
                  <Layers className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                </div>
                <div className="space-y-1 text-left">
                  <p className="font-medium">创建工作空间</p>
                  <p className="text-xs text-muted-foreground">配置新的 MCP 环境</p>
                </div>
              </Link>
            </Button>
            <Button
              variant="outline"
              className="h-auto flex-col items-start gap-2 p-4 hover:bg-muted/50 hover:border-primary/20"
              asChild
            >
              <Link href="/market">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-violet-100 dark:bg-violet-900/30">
                  <Package className="h-5 w-5 text-violet-600 dark:text-violet-400" />
                </div>
                <div className="space-y-1 text-left">
                  <p className="font-medium">浏览 MCP 市场</p>
                  <p className="text-xs text-muted-foreground">发现和安装新的 MCP</p>
                </div>
              </Link>
            </Button>
            <Button
              variant="outline"
              className="h-auto flex-col items-start gap-2 p-4 hover:bg-muted/50 hover:border-primary/20"
              asChild
            >
              <Link href="/api-keys">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-100 dark:bg-emerald-900/30">
                  <Users className="h-5 w-5 text-emerald-600 dark:text-emerald-400" />
                </div>
                <div className="space-y-1 text-left">
                  <p className="font-medium">管理 API Key</p>
                  <p className="text-xs text-muted-foreground">查看和轮换鉴权密钥</p>
                </div>
              </Link>
            </Button>
            <Button
              variant="outline"
              className="h-auto flex-col items-start gap-2 p-4 hover:bg-muted/50 hover:border-primary/20"
              asChild
            >
              <Link href="/playground">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-100 dark:bg-amber-900/30">
                  <Wrench className="h-5 w-5 text-amber-600 dark:text-amber-400" />
                </div>
                <div className="space-y-1 text-left">
                  <p className="font-medium">打开 Playground</p>
                  <p className="text-xs text-muted-foreground">直接调试 Gateway 请求</p>
                </div>
              </Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
