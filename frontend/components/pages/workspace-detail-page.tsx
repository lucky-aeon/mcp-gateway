'use client'

import { useMemo, useState } from 'react'
import Link from '@/components/router-link'
import { AlertCircle, ArrowLeft, CheckCircle, Clock, ExternalLink, FileText, Filter, Layers, MoreHorizontal, Package, Play, Plus, RefreshCw, Settings, Square, Trash2, Users, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import { gatewayApi, invalidate, useGatewaySWR, GatewayApiError, type InstalledItem, type ListData, type LogEntry, type Service, type Session, type Workspace } from '@/lib/gateway-api'
import { useToast } from '@/hooks/use-toast'
import { runAction } from '@/lib/action-feedback'

interface WorkspaceDetailPageProps {
  workspaceId: string
}

function formatDateTime(dateString: string) {
  return new Date(dateString).toLocaleString('zh-CN', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function WorkspaceDetailPage({ workspaceId }: WorkspaceDetailPageProps) {
  const { toast } = useToast()
  const [activeTab, setActiveTab] = useState('mcps')
  const [isAddOpen, setIsAddOpen] = useState(false)
  const [isCustomOpen, setIsCustomOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [serviceActionKey, setServiceActionKey] = useState('')
  const [sessionAction, setSessionAction] = useState(false)
  const [addingInstalledId, setAddingInstalledId] = useState('')
  const [oauthAction, setOauthAction] = useState('')
  const [serviceOAuthFlows, setServiceOAuthFlows] = useState<Record<string, { state: string; status: string; error?: string }>>({})
  const [savingSettings, setSavingSettings] = useState(false)
  const [logFilter, setLogFilter] = useState('all')
  const [settingsForm, setSettingsForm] = useState<{ name?: string; description?: string }>({})
  const [customForm, setCustomForm] = useState<{ name: string; protocol: string; url?: string; command?: string; args: string; env: string; authEnabled: boolean; authState: string; authAuthorized: boolean; authStatus?: string; authLoading: boolean; authError?: string }>({
    name: '',
    protocol: 'sse',
    args: '',
    env: '',
    authEnabled: false,
    authState: '',
    authAuthorized: false,
    authLoading: false,
  })

  const { data: workspace, isLoading } = useGatewaySWR<Workspace & { mcps?: Service[] }>(`/api/v1/workspaces/${workspaceId}`)
  const { data: servicesData } = useGatewaySWR<ListData<Service>>(`/api/v1/workspaces/${workspaceId}/services`)
  const { data: sessionsData } = useGatewaySWR<ListData<Session>>(`/api/v1/workspaces/${workspaceId}/sessions`)
  const { data: logsData } = useGatewaySWR<{ logs: LogEntry[] }>(`/api/v1/workspaces/${workspaceId}/logs`)
  const { data: installedData } = useGatewaySWR<ListData<InstalledItem>>('/api/v1/installed')

  const services = servicesData?.items || []
  const sessions = sessionsData?.items || []
  const logs = logsData?.logs || []
  const installedPackages = installedData?.items || []
  const filteredLogs = logFilter === 'all' ? logs : logs.filter((log) => log.level === logFilter)
  const formValue = {
    name: settingsForm.name ?? workspace?.name ?? '',
    description: settingsForm.description ?? workspace?.description ?? '',
  }

  const stats = useMemo(
    () => ({
      running: services.filter((item) => item.status === 'running').length,
      failed: services.filter((item) => item.status === 'failed').length,
      sessions: sessions.length,
    }),
    [services, sessions.length]
  )

  async function refreshWorkspace() {
    await Promise.all([
      invalidate(`/api/v1/workspaces/${workspaceId}`),
      invalidate(`/api/v1/workspaces/${workspaceId}/services`),
      invalidate(`/api/v1/workspaces/${workspaceId}/sessions`),
      invalidate(`/api/v1/workspaces/${workspaceId}/logs`),
      invalidate('/api/v1/installed'),
      invalidate('/api/v1/stats/overview'),
    ])
  }

  async function handleServiceAction(action: 'start' | 'stop' | 'restart' | 'delete', name: string) {
    setServiceActionKey(`${action}:${name}`)
    await runAction(
      async () => {
        if (action === 'start') await gatewayApi.startService(workspaceId, name)
        if (action === 'stop') await gatewayApi.stopService(workspaceId, name)
        if (action === 'restart') await gatewayApi.restartService(workspaceId, name)
        if (action === 'delete') await gatewayApi.deleteService(workspaceId, name)
        await refreshWorkspace()
      },
      { successTitle: '操作成功', errorTitle: '操作失败' }
    )
    setServiceActionKey('')
  }

  async function handleCreateSession() {
    setSessionAction(true)
    await runAction(
      async () => {
        await gatewayApi.createSession(workspaceId)
        await refreshWorkspace()
      },
      { successTitle: '会话已创建', errorTitle: '创建会话失败' }
    )
    setSessionAction(false)
  }

  async function handleDeleteSession(id: string) {
    setSessionAction(true)
    await runAction(
      async () => {
        await gatewayApi.deleteSession(workspaceId, id)
        await refreshWorkspace()
      },
      { successTitle: '会话已终止', errorTitle: '终止会话失败' }
    )
    setSessionAction(false)
  }

  async function handleAddInstalledPackage(item: InstalledItem) {
    if (item.auth?.type === 'oauth2' && item.auth.status !== 'authorized') {
      if (item.auth.authorization_url) {
        window.open(item.auth.authorization_url, '_blank', 'noopener,noreferrer')
      }
      toast({
        title: '需要先完成 OAuth 鉴权',
        description: '新标签页完成授权后，请回到已安装 MCP 页面确认鉴权完成。',
      })
      return
    }
    setAddingInstalledId(item.id)
    const ok = await runAction(
      async () => {
        await gatewayApi.deployInstalledPackage(workspaceId, {
          installed_id: item.id,
          service_name: (item.package_name || item.package_id).toLowerCase().replace(/[^a-z0-9-]+/g, '-').replace(/^-+|-+$/g, '') || item.package_id,
        })
        await refreshWorkspace()
      },
      { successTitle: '添加成功', successDescription: 'MCP 已添加到当前工作空间', errorTitle: '添加失败' }
    )
    setAddingInstalledId('')
    if (ok) setIsAddOpen(false)
  }

  async function handleCustomDeploy() {
    setSubmitting(true)
    try {
      const body: Record<string, unknown> = {
        name: customForm.name,
        gateway_protocol: customForm.protocol,
      }
      if (customForm.url) {
        body.url = customForm.url
      }
      if (customForm.authEnabled) {
        if (!customForm.authAuthorized) {
          toast({
            variant: 'destructive',
            title: '需要先完成 OAuth 鉴权',
            description: '请先点击自动获取鉴权并完成授权，再部署到工作空间。',
          })
          return
        }
        body.auth = {
          type: 'oauth2',
          state: customForm.authState,
        }
      }
      if (customForm.command) {
        body.command = customForm.command
      }
      if (customForm.args) {
        body.args = customForm.args.split('\n').filter((arg) => arg.trim())
      }
      if (customForm.env) {
        const envMap: Record<string, string> = {}
        customForm.env.split('\n').forEach((line) => {
          const [key, ...valueParts] = line.split('=')
          if (key && valueParts.length > 0) {
            envMap[key.trim()] = valueParts.join('=').trim()
          }
        })
        body.env = envMap
      }
      await gatewayApi.createService(workspaceId, body)
      setIsCustomOpen(false)
      setCustomForm({ name: '', protocol: 'sse', args: '', env: '', authEnabled: false, authState: '', authAuthorized: false, authLoading: false })
      await refreshWorkspace()
      toast({
        title: '部署成功',
        description: 'MCP 服务已成功部署',
      })
    } catch (error) {
      if (error instanceof GatewayApiError) {
        toast({
          variant: 'destructive',
          title: '部署失败',
          description: error.message,
        })
      } else {
        toast({
          variant: 'destructive',
          title: '部署失败',
          description: '部署 MCP 服务失败，请重试',
        })
      }
    } finally {
      setSubmitting(false)
    }
  }

  async function handleStartCustomOAuth() {
    if (!customForm.url) return
    setCustomForm((v) => ({ ...v, authLoading: true, authError: undefined, authStatus: 'discovering', authAuthorized: false }))
    await runAction(
      async () => {
        const flow = await gatewayApi.startMCPOAuth({ resource_url: customForm.url || '' })
        setCustomForm((v) => ({ ...v, authState: flow.state, authStatus: flow.status, authAuthorized: flow.status === 'authorized', authLoading: false }))
        window.open(flow.authorization_url, '_blank', 'noopener,noreferrer')
      },
      { successTitle: '已打开鉴权页面', successDescription: '完成授权后回到此页面检查状态', errorTitle: 'OAuth 发现失败' }
    )
    setCustomForm((v) => v.authLoading ? { ...v, authLoading: false, authStatus: 'failed', authError: 'OAuth 发现失败' } : v)
  }

  async function handleCheckCustomOAuth() {
    if (!customForm.authState) return
    await runAction(
      async () => {
        const status = await gatewayApi.getMCPOAuthStatus(customForm.authState)
        setCustomForm((v) => ({ ...v, authStatus: status.status, authAuthorized: status.status === 'authorized', authError: status.error }))
        if (status.status !== 'authorized') {
          throw new Error(status.error || 'OAuth 尚未完成')
        }
      },
      { successTitle: 'OAuth 已完成', successDescription: '现在可以部署到工作空间', errorTitle: 'OAuth 尚未完成' }
    )
  }

  async function handleStartServiceOAuth(service: Service) {
    if (!service.url) return
    setOauthAction(`start:${service.name}`)
    await runAction(
      async () => {
        const flow = await gatewayApi.startMCPOAuth({ resource_url: service.url || '' })
        setServiceOAuthFlows((items) => ({ ...items, [service.name]: { state: flow.state, status: flow.status } }))
        window.open(flow.authorization_url, '_blank', 'noopener,noreferrer')
      },
      { successTitle: '已打开鉴权页面', successDescription: '完成授权后检查状态以更新令牌', errorTitle: 'OAuth 发现失败' }
    )
    setOauthAction('')
  }

  async function handleCheckServiceOAuth(service: Service) {
    const flow = serviceOAuthFlows[service.name]
    if (!flow?.state) return
    setOauthAction(`check:${service.name}`)
    await runAction(
      async () => {
        const status = await gatewayApi.getMCPOAuthStatus(flow.state)
        setServiceOAuthFlows((items) => ({ ...items, [service.name]: { state: flow.state, status: status.status, error: status.error } }))
        if (status.status !== 'authorized') throw new Error(status.error || 'OAuth 尚未完成')
        await gatewayApi.updateService(workspaceId, service.name, {
          name: service.name,
          url: service.url,
          gateway_protocol: service.gateway_protocol || 'streamhttp',
          auth: { type: 'oauth2', state: flow.state },
        })
        await refreshWorkspace()
      },
      { successTitle: '令牌已更新', successDescription: '该 MCP 已可继续使用', errorTitle: 'OAuth 尚未完成' }
    )
    setOauthAction('')
  }

  async function handleSaveSettings() {
    setSavingSettings(true)
    await runAction(
      async () => {
        await gatewayApi.updateWorkspace(workspaceId, formValue)
        await refreshWorkspace()
      },
      { successTitle: '保存成功', successDescription: '工作空间设置已更新', errorTitle: '保存失败' }
    )
    setSavingSettings(false)
  }

  if (isLoading) {
    return <div className="text-sm text-muted-foreground">加载中...</div>
  }

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

  return (
    <div className="space-y-6">
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
              <p className="text-muted-foreground">{workspace.description || '暂无描述'}</p>
            </div>
          </div>
        </div>
        <Badge
          className={cn(
            'mt-2 border-0',
            workspace.status === 'running'
              ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
              : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
          )}
        >
          {workspace.status === 'running' ? '运行中' : '已停止'}
        </Badge>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card><CardContent className="p-6"><p className="text-sm text-muted-foreground">运行中 MCP</p><p className="mt-2 text-3xl font-bold">{stats.running}</p></CardContent></Card>
        <Card><CardContent className="p-6"><p className="text-sm text-muted-foreground">异常 MCP</p><p className="mt-2 text-3xl font-bold">{stats.failed}</p></CardContent></Card>
        <Card><CardContent className="p-6"><p className="text-sm text-muted-foreground">活跃会话</p><p className="mt-2 text-3xl font-bold">{stats.sessions}</p></CardContent></Card>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
        <TabsList className="inline-flex h-auto gap-1 rounded-xl bg-muted p-1">
          <TabsTrigger value="mcps" className="gap-2 rounded-lg px-4 py-2"><Package className="h-4 w-4" />MCPs</TabsTrigger>
          <TabsTrigger value="sessions" className="gap-2 rounded-lg px-4 py-2"><Users className="h-4 w-4" />会话</TabsTrigger>
          <TabsTrigger value="logs" className="gap-2 rounded-lg px-4 py-2"><FileText className="h-4 w-4" />日志</TabsTrigger>
          <TabsTrigger value="settings" className="gap-2 rounded-lg px-4 py-2"><Settings className="h-4 w-4" />设置</TabsTrigger>
        </TabsList>

        <TabsContent value="mcps" className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold">已部署的 MCP</h2>
              <p className="text-sm text-muted-foreground">管理此工作空间中的 MCP 服务</p>
            </div>
            <div className="flex gap-2">
              <Button variant="outline" onClick={refreshWorkspace}>
                <RefreshCw className="mr-2 h-4 w-4" />
                刷新
              </Button>
              <Dialog open={isCustomOpen} onOpenChange={setIsCustomOpen}>
                <DialogTrigger asChild>
                  <Button variant="outline">
                    <Plus className="mr-2 h-4 w-4" />
                    自定义部署
                  </Button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-2xl">
                  <DialogHeader>
                    <DialogTitle>自定义部署 MCP</DialogTitle>
                    <DialogDescription>通过配置URL或命令部署自定义MCP服务</DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 py-4">
                    <div className="space-y-2">
                      <Label htmlFor="custom-name">服务名称 *</Label>
                      <Input
                        id="custom-name"
                        value={customForm.name}
                        onChange={(e) => setCustomForm((v) => ({ ...v, name: e.target.value }))}
                        placeholder="例如: my-mcp-server"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="custom-protocol">协议类型 *</Label>
                      <Select value={customForm.protocol} onValueChange={(value) => setCustomForm((v) => ({ ...v, protocol: value }))}>
                        <SelectTrigger id="custom-protocol">
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="sse">Remote SSE</SelectItem>
                          <SelectItem value="streamhttp">Remote Streamable HTTP</SelectItem>
                          <SelectItem value="command">Command (命令行)</SelectItem>
                        </SelectContent>
                      </Select>
                    </div>
                    {(customForm.protocol === 'sse' || customForm.protocol === 'streamhttp') && (
                      <div className="space-y-2">
                        <Label htmlFor="custom-url">服务 URL *</Label>
                        <Input
                          id="custom-url"
                          value={customForm.url || ''}
                          onChange={(e) => setCustomForm((v) => ({ ...v, url: e.target.value }))}
                          placeholder={customForm.protocol === 'streamhttp' ? '例如: https://example.com/mcp' : '例如: https://example.com/sse'}
                        />
                      </div>
                    )}
                    {(customForm.protocol === 'sse' || customForm.protocol === 'streamhttp') && (
                      <div className="space-y-3 rounded-lg border bg-muted/30 p-4">
                        <div className="flex items-center justify-between gap-3">
                          <Label htmlFor="custom-auth-enabled">OAuth 2.0 鉴权</Label>
                          <Select
                            value={customForm.authEnabled ? 'oauth2' : 'none'}
                            onValueChange={(value) => setCustomForm((v) => ({ ...v, authEnabled: value === 'oauth2', authState: '', authAuthorized: false, authStatus: undefined, authError: undefined, authLoading: false }))}
                          >
                            <SelectTrigger id="custom-auth-enabled" className="w-36">
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="none">无</SelectItem>
                              <SelectItem value="oauth2">OAuth 2.0</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        {customForm.authEnabled && (
                          <div className="flex flex-wrap items-center gap-2">
                            <Button type="button" variant="outline" size="sm" onClick={handleStartCustomOAuth} disabled={!customForm.url || customForm.authLoading}>
                              <ExternalLink className="mr-2 h-4 w-4" />
                              {customForm.authLoading ? '获取中...' : customForm.authAuthorized ? '重新鉴权' : '自动获取并鉴权'}
                            </Button>
                            <Button type="button" variant="outline" size="sm" onClick={handleCheckCustomOAuth} disabled={!customForm.authState}>
                              检查鉴权状态
                            </Button>
                            {customForm.authStatus && (
                              <Badge variant={customForm.authAuthorized ? 'default' : 'secondary'}>
                                {customForm.authAuthorized ? '已授权' : customForm.authStatus}
                              </Badge>
                            )}
                            {customForm.authError && <span className="text-sm text-destructive">{customForm.authError}</span>}
                          </div>
                        )}
                      </div>
                    )}
                    {customForm.protocol === 'command' && (
                      <>
                        <div className="space-y-2">
                          <Label htmlFor="custom-command">命令 *</Label>
                          <Input
                            id="custom-command"
                            value={customForm.command || ''}
                            onChange={(e) => setCustomForm((v) => ({ ...v, command: e.target.value }))}
                            placeholder="例如: npx"
                          />
                        </div>
                        <div className="space-y-2">
                          <Label htmlFor="custom-args">参数（每行一个，可选）</Label>
                          <Textarea
                            id="custom-args"
                            value={customForm.args}
                            onChange={(e) => setCustomForm((v) => ({ ...v, args: e.target.value }))}
                            placeholder="例如:&#10;--port=3000&#10;--host=0.0.0.0"
                            rows={3}
                          />
                        </div>
                      </>
                    )}
                    <div className="space-y-2">
                      <Label htmlFor="custom-env">环境变量（每行一个 KEY=VALUE，可选）</Label>
                      <Textarea
                        id="custom-env"
                        value={customForm.env}
                        onChange={(e) => setCustomForm((v) => ({ ...v, env: e.target.value }))}
                        placeholder="例如:&#10;API_KEY=your_key&#10;DEBUG=true"
                        rows={3}
                      />
                    </div>
                  </div>
                  <DialogFooter>
                    <Button variant="outline" onClick={() => setIsCustomOpen(false)} disabled={submitting}>取消</Button>
                    <Button onClick={handleCustomDeploy} disabled={submitting || !customForm.name || (customForm.protocol === 'command' && !customForm.command) || ((customForm.protocol === 'sse' || customForm.protocol === 'streamhttp') && !customForm.url) || (customForm.authEnabled && (!customForm.authState || !customForm.authAuthorized))}>
                      {submitting ? (
                        <>
                          <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                          部署中...
                        </>
                      ) : (
                        '部署'
                      )}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
              <Dialog open={isAddOpen} onOpenChange={setIsAddOpen}>
                <DialogTrigger asChild>
                  <Button>
                    <Plus className="mr-2 h-4 w-4" />
                    添加已安装 MCP
                  </Button>
                </DialogTrigger>
                <DialogContent className="sm:max-w-2xl">
                  <DialogHeader>
                    <DialogTitle>添加已安装 MCP</DialogTitle>
                    <DialogDescription>从当前账号已安装的 MCP 配置快照添加到此工作空间</DialogDescription>
                  </DialogHeader>
                  <div className="grid max-h-[420px] gap-3 overflow-auto py-4">
                    {installedPackages.map((pkg) => (
                      <div key={pkg.id} className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                        <div>
                          <div className="flex flex-wrap items-center gap-2">
                            <p className="font-medium">{pkg.display_name || pkg.package_name}</p>
                            {pkg.auth?.type === 'oauth2' && (
                              <Badge variant={pkg.auth.status === 'authorized' ? 'default' : 'secondary'}>
                                {pkg.auth.status === 'authorized' ? 'OAuth 已授权' : 'OAuth 待授权'}
                              </Badge>
                            )}
                          </div>
                          <p className="text-sm text-muted-foreground">
                            {pkg.package_id} · v{pkg.installed_version} · {pkg.source_id || 'local'}
                          </p>
                        </div>
                        <Button size="sm" onClick={() => handleAddInstalledPackage(pkg)} disabled={addingInstalledId === pkg.id}>
                          {pkg.auth?.type === 'oauth2' && pkg.auth.status !== 'authorized' ? (
                            <>
                              <ExternalLink className="mr-2 h-4 w-4" />
                              鉴权
                            </>
                          ) : addingInstalledId === pkg.id ? '添加中...' : '添加'}
                        </Button>
                      </div>
                    ))}
                    {installedPackages.length === 0 && (
                      <p className="text-sm text-muted-foreground">当前账号还没有已安装 MCP，请先到 MCP 市场安装。</p>
                    )}
                  </div>
                  <DialogFooter>
                    <Button variant="outline" onClick={() => setIsAddOpen(false)}>关闭</Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            {services.map((mcp) => (
              <Card key={mcp.name} className="group">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-primary/10 to-primary/5 ring-1 ring-primary/10">
                        <Package className="h-5 w-5 text-primary" />
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <CardTitle className="text-base">{mcp.name}</CardTitle>
                          <Badge
                            className={cn(
                              'text-xs',
                              mcp.status === 'running'
                                ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                                : mcp.status === 'failed'
                                  ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                                  : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                            )}
                          >
                            {mcp.status === 'running' ? '运行中' : mcp.status === 'failed' ? '失败' : '已停止'}
                          </Badge>
                          {mcp.url && mcp.auth_status === 'authorized' && <Badge variant="secondary">OAuth 已授权</Badge>}
                          {mcp.url && serviceOAuthFlows[mcp.name]?.status && (
                            <Badge variant={serviceOAuthFlows[mcp.name]?.status === 'authorized' ? 'default' : 'secondary'}>
                              {serviceOAuthFlows[mcp.name]?.status === 'authorized' ? 'OAuth 待更新' : serviceOAuthFlows[mcp.name]?.status}
                            </Badge>
                          )}
                        </div>
                        <CardDescription className="mt-1">
                          {mcp.source_type === 'market' ? `来自市场: ${mcp.source_ref}` : mcp.command || mcp.url || '自定义服务'}
                        </CardDescription>
                      </div>
                    </div>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="icon" className="h-8 w-8 opacity-0 transition-opacity group-hover:opacity-100">
                          <MoreHorizontal className="h-4 w-4" />
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end">
                        {mcp.status === 'running' ? (
                          <DropdownMenuItem onSelect={() => handleServiceAction('stop', mcp.name)} disabled={serviceActionKey === `stop:${mcp.name}`}>
                            <Square className="mr-2 h-4 w-4" />
                            {serviceActionKey === `stop:${mcp.name}` ? '停止中...' : '停止'}
                          </DropdownMenuItem>
                        ) : (
                          <DropdownMenuItem onSelect={() => handleServiceAction('start', mcp.name)} disabled={serviceActionKey === `start:${mcp.name}`}>
                            <Play className="mr-2 h-4 w-4" />
                            {serviceActionKey === `start:${mcp.name}` ? '启动中...' : '启动'}
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuItem onSelect={() => handleServiceAction('restart', mcp.name)} disabled={serviceActionKey === `restart:${mcp.name}`}>
                          <RefreshCw className="mr-2 h-4 w-4" />
                          {serviceActionKey === `restart:${mcp.name}` ? '重启中...' : '重启'}
                        </DropdownMenuItem>
                        {mcp.url && (
                          <DropdownMenuItem onSelect={() => handleStartServiceOAuth(mcp)} disabled={oauthAction === `start:${mcp.name}`}>
                            <ExternalLink className="mr-2 h-4 w-4" />
                            {oauthAction === `start:${mcp.name}` ? '获取中...' : mcp.auth_status === 'authorized' ? '重新鉴权' : '鉴权'}
                          </DropdownMenuItem>
                        )}
                        <DropdownMenuItem onSelect={() => handleServiceAction('delete', mcp.name)} disabled={serviceActionKey === `delete:${mcp.name}`} className="text-destructive">
                          <Trash2 className="mr-2 h-4 w-4" />
                          {serviceActionKey === `delete:${mcp.name}` ? '删除中...' : '删除'}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </div>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="grid gap-2 text-sm text-muted-foreground">
                    <p>工具数：{mcp.tools_count}</p>
                    <p>端口：{mcp.port || '-'}</p>
                    {mcp.url && <p>协议：{mcp.gateway_protocol === 'streamhttp' ? 'Streamable HTTP' : 'SSE'}</p>}
                    <p>部署时间：{formatDateTime(mcp.created_at)}</p>
                    {mcp.last_error && <p className="text-red-600 dark:text-red-400">错误：{mcp.last_error}</p>}
                  </div>
                  {mcp.url && serviceOAuthFlows[mcp.name]?.state && (
                    <div className="flex flex-wrap items-center gap-2">
                      <Button size="sm" variant="outline" onClick={() => handleCheckServiceOAuth(mcp)} disabled={oauthAction === `check:${mcp.name}`}>
                        {oauthAction === `check:${mcp.name}` ? '检查中...' : '检查鉴权状态并更新令牌'}
                      </Button>
                      {serviceOAuthFlows[mcp.name]?.error && <span className="text-sm text-destructive">{serviceOAuthFlows[mcp.name]?.error}</span>}
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
            {services.length === 0 && <p className="text-sm text-muted-foreground">当前工作空间还没有部署 MCP。</p>}
          </div>
        </TabsContent>

        <TabsContent value="sessions" className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold">会话管理</h2>
              <p className="text-sm text-muted-foreground">查看和管理当前连接的会话</p>
            </div>
            <Button onClick={handleCreateSession} disabled={sessionAction}>新建会话</Button>
          </div>
          <Card>
            <CardContent className="p-0">
              <Table>
                <TableHeader>
                  <TableRow className="hover:bg-transparent">
                    <TableHead>会话 ID</TableHead>
                    <TableHead>状态</TableHead>
                    <TableHead className="text-right">工具数</TableHead>
                    <TableHead>绑定 MCP</TableHead>
                    <TableHead>最后活动</TableHead>
                    <TableHead className="w-[80px]"></TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sessions.map((session) => (
                    <TableRow key={session.id}>
                      <TableCell className="font-mono text-sm">{session.id}</TableCell>
                      <TableCell>
                        <Badge
                          className={cn(
                            'border-0',
                            session.status === 'active'
                              ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                              : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                          )}
                        >
                          {session.status}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-right font-medium">{session.tools_count}</TableCell>
                      <TableCell className="text-muted-foreground">{session.bound_mcp_names.join(', ') || '无'}</TableCell>
                      <TableCell className="text-muted-foreground">{formatDateTime(session.last_receive_time)}</TableCell>
                      <TableCell>
                        <Button variant="ghost" size="sm" className="text-destructive hover:text-destructive" onClick={() => handleDeleteSession(session.id)} disabled={sessionAction}>
                          {sessionAction ? '处理中...' : '终止'}
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="logs" className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-lg font-semibold">操作日志</h2>
              <p className="text-sm text-muted-foreground">聚合当前工作空间下各服务日志</p>
            </div>
            <div className="flex items-center gap-2">
              <Filter className="h-4 w-4 text-muted-foreground" />
              <Select value={logFilter} onValueChange={setLogFilter}>
                <SelectTrigger className="w-32"><SelectValue /></SelectTrigger>
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
            <CardContent className="space-y-2 p-4">
              {filteredLogs.map((log, index) => (
                <div key={`${log.timestamp}-${index}`} className="flex items-start gap-3 rounded-xl border bg-muted/30 p-4">
                  <div className="mt-0.5">
                    {log.level === 'info' && <CheckCircle className="h-4 w-4 text-blue-500" />}
                    {log.level === 'warn' && <AlertCircle className="h-4 w-4 text-amber-500" />}
                    {log.level === 'error' && <AlertCircle className="h-4 w-4 text-red-500" />}
                    {log.level === 'debug' && <Clock className="h-4 w-4 text-slate-500" />}
                  </div>
                  <div className="min-w-0 flex-1 space-y-1">
                    <div className="flex items-center gap-2 text-xs text-muted-foreground">
                      <Badge variant="outline">{log.level.toUpperCase()}</Badge>
                      <span>{log.source || 'system'}</span>
                      <span>{formatDateTime(log.timestamp)}</span>
                    </div>
                    <p className="text-sm">{log.message}</p>
                  </div>
                </div>
              ))}
              {filteredLogs.length === 0 && <p className="text-sm text-muted-foreground">暂无日志。</p>}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="settings" className="space-y-6">
          <div>
            <h2 className="text-lg font-semibold">工作空间设置</h2>
            <p className="text-sm text-muted-foreground">更新名称和描述</p>
          </div>
          <Card>
            <CardHeader>
              <CardTitle>基本设置</CardTitle>
              <CardDescription>工作空间的基础信息</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="ws-name">名称</Label>
                <Input id="ws-name" value={formValue.name} onChange={(e) => setSettingsForm((v) => ({ ...v, name: e.target.value }))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="ws-desc">描述</Label>
                <Textarea id="ws-desc" value={formValue.description} onChange={(e) => setSettingsForm((v) => ({ ...v, description: e.target.value }))} />
              </div>
              <Button onClick={handleSaveSettings} disabled={savingSettings}>{savingSettings ? '保存中...' : '保存更改'}</Button>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
