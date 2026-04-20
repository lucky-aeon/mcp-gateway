'use client'

import { Activity, AlertTriangle, ArrowUpRight, CheckCircle, Clock, Gauge, Layers, Package, Server, Users } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { cn } from '@/lib/utils'
import { useGatewaySWR, type ListData, type OverviewStats, type SystemConfig, type Workspace } from '@/lib/gateway-api'

function formatRelative(ts: string) {
  const date = new Date(ts)
  const diff = Date.now() - date.getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return '刚刚'
  if (mins < 60) return `${mins} 分钟前`
  return `${Math.floor(mins / 60)} 小时前`
}

export function OpsDashboardPage() {
  const { data: stats } = useGatewaySWR<OverviewStats>('/api/v1/stats/overview')
  const { data: config } = useGatewaySWR<SystemConfig>('/api/v1/system/config')
  const { data: workspaces } = useGatewaySWR<ListData<Workspace>>('/api/v1/workspaces')

  const systemMetrics = [
    {
      label: 'Session GC',
      value: `${config?.session_gc_interval_seconds ?? '-'}s`,
      icon: Clock,
      tone: 'bg-blue-100 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400',
    },
    {
      label: 'Session Timeout',
      value: `${config?.proxy_session_timeout_seconds ?? '-'}s`,
      icon: Gauge,
      tone: 'bg-amber-100 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400',
    },
    {
      label: 'Retry Count',
      value: `${config?.mcp_retry_count ?? '-'}`,
      icon: RefreshlessIcon,
      tone: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400',
    },
    {
      label: 'Gateway Protocol',
      value: config?.gateway_protocol || '-',
      icon: Server,
      tone: 'bg-violet-100 text-violet-600 dark:bg-violet-900/20 dark:text-violet-400',
    },
  ]

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">运维看板</h2>
        <p className="text-muted-foreground">系统运行状态和性能指标监控</p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard title="工作空间" value={stats?.workspaces_count ?? 0} icon={Layers} tone="emerald" hint="全部工作空间数" />
        <MetricCard title="运行中 MCP" value={stats?.running_mcps ?? 0} icon={Package} tone="blue" hint="当前在线服务" />
        <MetricCard title="活跃会话" value={stats?.active_sessions ?? 0} icon={Users} tone="amber" hint="活动中的会话" />
        <MetricCard title="24h 异常 MCP" value={stats?.failed_mcps_24h ?? 0} icon={AlertTriangle} tone="red" hint="最近失败记录" />
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {systemMetrics.map((item) => (
          <Card key={item.label} className="group border-border/50 bg-gradient-to-br from-card to-card/50 transition-all duration-300 hover:shadow-lg">
            <CardHeader className="flex flex-row items-center justify-between pb-3">
              <CardTitle className="text-sm font-medium">{item.label}</CardTitle>
              <div className={cn('rounded-lg p-2 transition-all group-hover:scale-110', item.tone)}>
                <item.icon className="h-4 w-4" />
              </div>
            </CardHeader>
            <CardContent>
              <div className="text-3xl font-bold tracking-tight">{item.value}</div>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Activity className="h-5 w-5 text-muted-foreground" />
              <CardTitle>最近活动</CardTitle>
            </div>
            <CardDescription>最近控制面事件</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {(stats?.recent_activity || []).slice(0, 8).map((activity) => (
              <div key={`${activity.at}-${activity.type}-${activity.message}`} className="flex items-start gap-3 rounded-xl border bg-muted/30 p-4">
                <div className="mt-0.5">
                  {activity.type === 'session.created' ? <CheckCircle className="h-4 w-4 text-emerald-500" /> : <ArrowUpRight className="h-4 w-4 text-blue-500" />}
                </div>
                <div className="min-w-0 flex-1">
                  <p className="text-sm">{activity.message || activity.type}</p>
                  <p className="mt-1 text-xs text-muted-foreground">{activity.workspace_name || '系统'} · {formatRelative(activity.at)}</p>
                </div>
              </div>
            ))}
            {!stats?.recent_activity?.length && <p className="text-sm text-muted-foreground">暂无活动记录。</p>}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Server className="h-5 w-5 text-muted-foreground" />
              <CardTitle>工作空间状态</CardTitle>
            </div>
            <CardDescription>各工作空间当前运行状态</CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {workspaces?.items.map((workspace) => (
              <div key={workspace.id} className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div>
                  <p className="font-medium">{workspace.name}</p>
                  <p className="text-sm text-muted-foreground">{workspace.mcp_count} MCPs · {workspace.session_count} Sessions</p>
                </div>
                {workspace.status === 'running' ? (
                  <Badge className="gap-1 bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400">
                    <CheckCircle className="h-3 w-3" />
                    运行中
                  </Badge>
                ) : (
                  <Badge variant="secondary">已停止</Badge>
                )}
              </div>
            ))}
            {!workspaces?.items.length && <p className="text-sm text-muted-foreground">暂无工作空间。</p>}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function MetricCard({
  title,
  value,
  icon: Icon,
  tone,
  hint,
}: {
  title: string
  value: number
  icon: React.ComponentType<{ className?: string }>
  tone: 'emerald' | 'blue' | 'amber' | 'red'
  hint: string
}) {
  const toneMap = {
    emerald: 'bg-emerald-100 text-emerald-600 dark:bg-emerald-900/20 dark:text-emerald-400',
    blue: 'bg-blue-100 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400',
    amber: 'bg-amber-100 text-amber-600 dark:bg-amber-900/20 dark:text-amber-400',
    red: 'bg-red-100 text-red-600 dark:bg-red-900/20 dark:text-red-400',
  }

  return (
    <Card className="group border-border/50 bg-gradient-to-br from-card to-card/50 transition-all duration-300 hover:shadow-lg">
      <CardHeader className="flex flex-row items-center justify-between pb-3">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <div className={cn('rounded-lg p-2 transition-all group-hover:scale-110', toneMap[tone])}>
          <Icon className="h-4 w-4" />
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-3xl font-bold tracking-tight">{value}</div>
        <p className="mt-2 text-xs text-muted-foreground">{hint}</p>
      </CardContent>
    </Card>
  )
}

function RefreshlessIcon({ className }: { className?: string }) {
  return <Gauge className={className} />
}
