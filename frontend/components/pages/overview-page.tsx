'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { overviewStats, workspaces } from '@/lib/mock-data'
import { Layers, Users, Package, Wrench, ArrowRight, Activity, TrendingUp, Clock } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'
import Link from 'next/link'

const stats = [
  {
    label: '工作空间',
    value: overviewStats.totalWorkspaces,
    change: '+2',
    changeType: 'increase' as const,
    icon: Layers,
    color: 'text-blue-600 dark:text-blue-400',
    bgColor: 'bg-blue-100 dark:bg-blue-900/30',
    href: '/workspaces',
  },
  {
    label: '活跃会话',
    value: overviewStats.activeSessions,
    change: '+12',
    changeType: 'increase' as const,
    icon: Users,
    color: 'text-emerald-600 dark:text-emerald-400',
    bgColor: 'bg-emerald-100 dark:bg-emerald-900/30',
    href: '/workspaces',
  },
  {
    label: '已安装 MCP',
    value: overviewStats.totalMCPs,
    change: '+5',
    changeType: 'increase' as const,
    icon: Package,
    color: 'text-violet-600 dark:text-violet-400',
    bgColor: 'bg-violet-100 dark:bg-violet-900/30',
    href: '/installed',
  },
  {
    label: '今日调用',
    value: overviewStats.totalToolCalls,
    change: '+18%',
    changeType: 'increase' as const,
    icon: Wrench,
    color: 'text-amber-600 dark:text-amber-400',
    bgColor: 'bg-amber-100 dark:bg-amber-900/30',
    href: '/playground',
  },
]

const activityTypeLabels: Record<string, string> = {
  session_start: '会话开始',
  session_end: '会话结束',
  tool_call: '工具调用',
  mcp_added: 'MCP 添加',
  mcp_removed: 'MCP 移除',
}

const activityTypeStyles: Record<string, string> = {
  session_start: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
  session_end: 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400',
  tool_call: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  mcp_added: 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-400',
  mcp_removed: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
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

export function OverviewPage() {
  return (
    <div className="space-y-8">
      {/* Page Header */}
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">欢迎回来</h2>
        <p className="text-muted-foreground">以下是您的 Gateway 运行概览</p>
      </div>

      {/* Stats Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {stats.map((stat) => {
          const Icon = stat.icon
          return (
            <Link key={stat.label} href={stat.href}>
              <Card className="group relative overflow-hidden transition-all hover:shadow-md hover:border-primary/20">
                <CardContent className="p-6">
                  <div className="flex items-start justify-between">
                    <div className="space-y-2">
                      <p className="text-sm font-medium text-muted-foreground">{stat.label}</p>
                      <div className="flex items-baseline gap-2">
                        <p className="text-3xl font-bold tracking-tight">{stat.value.toLocaleString()}</p>
                        <span className="flex items-center text-xs font-medium text-emerald-600 dark:text-emerald-400">
                          <TrendingUp className="mr-0.5 h-3 w-3" />
                          {stat.change}
                        </span>
                      </div>
                    </div>
                    <div className={cn('rounded-xl p-3 transition-transform group-hover:scale-110', stat.bgColor)}>
                      <Icon className={cn('h-5 w-5', stat.color)} />
                    </div>
                  </div>
                </CardContent>
                <div className="absolute inset-x-0 bottom-0 h-1 bg-gradient-to-r from-transparent via-primary/20 to-transparent opacity-0 transition-opacity group-hover:opacity-100" />
              </Card>
            </Link>
          )
        })}
      </div>

      <div className="grid gap-6 lg:grid-cols-5">
        {/* Recent Workspaces */}
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
                    <p className="font-medium group-hover:text-primary transition-colors">{workspace.name}</p>
                    <div className="flex items-center gap-3 text-sm text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <Package className="h-3.5 w-3.5" />
                        {workspace.mcpCount} MCPs
                      </span>
                      <span className="flex items-center gap-1">
                        <Users className="h-3.5 w-3.5" />
                        {workspace.sessionCount} 会话
                      </span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3">
                  <Badge 
                    variant={workspace.status === 'active' ? 'default' : 'secondary'}
                    className={cn(
                      workspace.status === 'active' 
                        ? 'bg-emerald-100 text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-900/30 dark:text-emerald-400' 
                        : ''
                    )}
                  >
                    {workspace.status === 'active' ? '运行中' : '已停止'}
                  </Badge>
                  <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 transition-all group-hover:opacity-100 group-hover:translate-x-1" />
                </div>
              </Link>
            ))}
          </CardContent>
        </Card>

        {/* Recent Activity */}
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
              {overviewStats.recentActivity.map((activity) => (
                <div
                  key={activity.id}
                  className="flex items-start gap-3"
                >
                  <div className="mt-0.5">
                    <Badge
                      variant="outline"
                      className={cn('text-xs font-medium border-0', activityTypeStyles[activity.type])}
                    >
                      {activityTypeLabels[activity.type]}
                    </Badge>
                  </div>
                  <div className="flex-1 min-w-0 space-y-1">
                    <p className="text-sm leading-tight">{activity.description}</p>
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <span>{activity.workspaceName}</span>
                      <span>·</span>
                      <span className="flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {formatTime(activity.timestamp)}
                      </span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Quick Actions */}
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
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-amber-100 dark:bg-amber-900/30">
                  <svg className="h-5 w-5 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z" />
                  </svg>
                </div>
                <div className="space-y-1 text-left">
                  <p className="font-medium">管理 API 密钥</p>
                  <p className="text-xs text-muted-foreground">创建和管理访问凭证</p>
                </div>
              </Link>
            </Button>
            <Button
              variant="outline"
              className="h-auto flex-col items-start gap-2 p-4 hover:bg-muted/50 hover:border-primary/20"
              asChild
            >
              <Link href="/playground">
                <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-emerald-100 dark:bg-emerald-900/30">
                  <svg className="h-5 w-5 text-emerald-600 dark:text-emerald-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                </div>
                <div className="space-y-1 text-left">
                  <p className="font-medium">Playground</p>
                  <p className="text-xs text-muted-foreground">在线测试工具调用</p>
                </div>
              </Link>
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
