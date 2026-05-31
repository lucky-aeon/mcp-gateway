'use client'

import { useEffect, useState } from 'react'
import { Check, Database, Pencil, Plus, RefreshCw, Save, Server, Shield, Bell, Monitor, Moon, Settings, Store, Sun, Trash2, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { gatewayApi, invalidate, useGatewaySWR, type GatewayExposureProtocol, type ListData, type MarketPackage, type MarketPackageInput, type MarketSource, type SystemConfig } from '@/lib/gateway-api'
import { useTheme } from '@/components/providers/theme-provider'
import { cn } from '@/lib/utils'
import { runAction } from '@/lib/action-feedback'

export function SetupPage() {
  const { data } = useGatewaySWR<SystemConfig>('/api/v1/system/config')
  const { data: sourcesData } = useGatewaySWR<ListData<MarketSource>>('/api/v1/market/sources')
  const { data: localMarketData } = useGatewaySWR<ListData<MarketPackage>>('/api/v1/market/packages?source=local')
  const [saved, setSaved] = useState(false)
  const [form, setForm] = useState<SystemConfig | null>(null)
  const [syncingSource, setSyncingSource] = useState('')
  const [savingConfig, setSavingConfig] = useState(false)
  const [savingMarketPackage, setSavingMarketPackage] = useState(false)
  const [deletingPackageId, setDeletingPackageId] = useState('')
  const [marketSheetOpen, setMarketSheetOpen] = useState(false)
  const [editingPackageId, setEditingPackageId] = useState('')
  const [marketForm, setMarketForm] = useState({
    name: '',
    title: '',
    description: '',
    version: '1.0.0',
    category: '开发',
    tags: '',
    command: '',
    args: '',
    url: '',
    installType: 'command',
  })
  const { theme, setTheme, compactMode, setCompactMode } = useTheme()
  const marketSources = sourcesData?.items || []
  const localPackages = localMarketData?.items || []

  useEffect(() => {
    if (data) setForm(data)
  }, [data])

  async function handleSave() {
    if (!form) return
    setSavingConfig(true)
    const ok = await runAction(
      async () => {
        await gatewayApi.updateSystemConfig(form)
      },
      { successTitle: '保存成功', successDescription: '系统配置已更新', errorTitle: '保存失败' }
    )
    setSavingConfig(false)
    if (ok) {
      setSaved(true)
      setTimeout(() => setSaved(false), 2000)
    }
  }

  const authProvider = (form?.auth.authorization_servers?.length || 0) > 0 ? 'external' : 'internal'
  const listValue = (items?: string[]) => (items || []).join('\n')
  const setAuthField = <K extends keyof SystemConfig['auth']>(key: K, value: SystemConfig['auth'][K]) => {
    setForm((v) => v ? { ...v, auth: { ...v.auth, [key]: value } } : v)
  }
  const splitList = (value: string) => value.split(/[\n,]/).map((item) => item.trim()).filter(Boolean)

  async function handleSyncSource(sourceId: string) {
    setSyncingSource(sourceId)
    await runAction(
      async () => {
      await gatewayApi.syncMarketSource(sourceId)
      await Promise.all([invalidate('/api/v1/market/sources'), invalidate('/api/v1/market/packages')])
      },
      { successTitle: '同步完成', successDescription: '市场源数据已更新', errorTitle: '同步失败' }
    )
    setSyncingSource('')
  }

  function resetMarketForm() {
    setEditingPackageId('')
    setMarketForm({
      name: '',
      title: '',
      description: '',
      version: '1.0.0',
      category: '开发',
      tags: '',
      command: '',
      args: '',
      url: '',
      installType: 'command',
    })
  }

  function openCreateMarketPackage() {
    resetMarketForm()
    setMarketSheetOpen(true)
  }

  function editMarketPackage(pkg: MarketPackage) {
    const option = pkg.install_options?.[0]
    setEditingPackageId(pkg.id)
    setMarketForm({
      name: pkg.name || '',
      title: pkg.title || pkg.name || '',
      description: pkg.description || '',
      version: pkg.version || '1.0.0',
      category: pkg.category || '开发',
      tags: (pkg.tags || []).join(', '),
      command: option?.command || '',
      args: (option?.args || []).join(' '),
      url: option?.url || '',
      installType: option?.type || 'command',
    })
    setMarketSheetOpen(true)
  }

  function localMarketPayload(): MarketPackageInput {
    const args = marketForm.args.split(/\s+/).map((item) => item.trim()).filter(Boolean)
    const installType = marketForm.installType
    return {
      name: marketForm.name.trim(),
      title: marketForm.title.trim() || marketForm.name.trim(),
      description: marketForm.description.trim(),
      version: marketForm.version.trim(),
      category: marketForm.category.trim(),
      tags: marketForm.tags.split(',').map((item) => item.trim()).filter(Boolean),
      verified: true,
      install_options: [{
        type: installType,
        command: installType === 'remote' ? undefined : marketForm.command.trim(),
        args: installType === 'remote' ? undefined : args,
        url: installType === 'remote' ? marketForm.url.trim() : undefined,
        env: {},
        source_id: 'local',
        confidence: 'high',
      }],
      tools: [],
    }
  }

  async function saveLocalMarketPackage() {
    const payload = localMarketPayload()
    setSavingMarketPackage(true)
    const ok = await runAction(
      async () => {
        if (editingPackageId) {
          await gatewayApi.updateMarketPackage(editingPackageId, payload)
        } else {
          await gatewayApi.createMarketPackage(payload)
        }
        await Promise.all([invalidate('/api/v1/market/packages'), invalidate('/api/v1/market/packages?source=local'), invalidate('/api/v1/market/sources')])
      },
      {
        successTitle: editingPackageId ? '保存成功' : '添加成功',
        successDescription: '自有市场 MCP 已更新',
        errorTitle: editingPackageId ? '保存失败' : '添加失败',
      }
    )
    setSavingMarketPackage(false)
    if (ok) {
      resetMarketForm()
      setMarketSheetOpen(false)
    }
  }

  async function deleteLocalMarketPackage(id: string) {
    setDeletingPackageId(id)
    const ok = await runAction(
      async () => {
        await gatewayApi.deleteMarketPackage(id)
        await Promise.all([invalidate('/api/v1/market/packages'), invalidate('/api/v1/market/packages?source=local'), invalidate('/api/v1/market/sources')])
      },
      { successTitle: '删除成功', successDescription: '自有市场 MCP 已删除', errorTitle: '删除失败' }
    )
    setDeletingPackageId('')
    if (ok && editingPackageId === id) resetMarketForm()
  }

  if (!form) {
    return <div className="text-sm text-muted-foreground">加载配置中...</div>
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">设置</h2>
        <p className="text-muted-foreground">管理 Gateway 实例的核心系统配置</p>
      </div>

      <Tabs defaultValue="general" className="space-y-6">
        <TabsList className="inline-flex h-auto gap-1 rounded-xl bg-muted p-1">
          <TabsTrigger value="general" className="gap-2 rounded-lg px-4 py-2"><Settings className="h-4 w-4" />常规</TabsTrigger>
          <TabsTrigger value="server" className="gap-2 rounded-lg px-4 py-2"><Server className="h-4 w-4" />服务器</TabsTrigger>
          <TabsTrigger value="security" className="gap-2 rounded-lg px-4 py-2"><Shield className="h-4 w-4" />安全</TabsTrigger>
          <TabsTrigger value="market" className="gap-2 rounded-lg px-4 py-2"><Store className="h-4 w-4" />市场源</TabsTrigger>
          <TabsTrigger value="notifications" className="gap-2 rounded-lg px-4 py-2"><Bell className="h-4 w-4" />通知</TabsTrigger>
          <TabsTrigger value="advanced" className="gap-2 rounded-lg px-4 py-2"><Database className="h-4 w-4" />高级</TabsTrigger>
        </TabsList>

        <TabsContent value="general" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>基本信息</CardTitle>
              <CardDescription>Gateway 实例的基本配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="grid gap-6 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="instance-name">实例名称</Label>
                  <Input id="instance-name" defaultValue="My Gateway" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="timezone">时区</Label>
                  <Select defaultValue="Asia/Shanghai">
                    <SelectTrigger id="timezone"><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Asia/Shanghai">Asia/Shanghai (UTC+8)</SelectItem>
                      <SelectItem value="America/New_York">America/New_York (UTC-5)</SelectItem>
                      <SelectItem value="Europe/London">Europe/London (UTC+0)</SelectItem>
                      <SelectItem value="Asia/Tokyo">Asia/Tokyo (UTC+9)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="instance-desc">描述</Label>
                <Textarea id="instance-desc" defaultValue="MCP Gateway 管理控制台" rows={3} />
              </div>
              <div className="space-y-3">
                <Label>主题模式</Label>
                <div className="grid grid-cols-3 gap-3">
                  {[
                    { id: 'light', label: '浅色', icon: Sun },
                    { id: 'dark', label: '深色', icon: Moon },
                    { id: 'system', label: '跟随系统', icon: Monitor },
                  ].map((item) => (
                    <button
                      key={item.id}
                      onClick={() => setTheme(item.id as 'light' | 'dark' | 'system')}
                      className={cn(
                        'flex flex-col items-center gap-2 rounded-xl border-2 p-4 transition-all hover:bg-muted/50',
                        theme === item.id ? 'border-primary bg-primary/5' : 'border-transparent bg-muted/30'
                      )}
                    >
                      <div className={cn('flex h-10 w-10 items-center justify-center rounded-lg', theme === item.id ? 'bg-primary text-primary-foreground' : 'bg-muted')}>
                        <item.icon className="h-5 w-5" />
                      </div>
                      <span className="text-sm font-medium">{item.label}</span>
                    </button>
                  ))}
                </div>
              </div>
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">紧凑模式</Label>
                  <p className="text-sm text-muted-foreground">减少界面间距，显示更多内容</p>
                </div>
                <Switch checked={compactMode} onCheckedChange={setCompactMode} />
              </div>
            </CardContent>
          </Card>

        </TabsContent>

        <TabsContent value="server" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>服务器配置</CardTitle>
              <CardDescription>Gateway 服务器的运行配置</CardDescription>
            </CardHeader>
            <CardContent className="grid gap-6 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="bind">绑定地址</Label>
                <Input id="bind" value={form.bind} onChange={(e) => setForm((v) => v ? { ...v, bind: e.target.value } : v)} />
              </div>
              <div className="space-y-2">
                <Label>网关协议</Label>
                <Select value={form.gateway_protocol} onValueChange={(value: GatewayExposureProtocol) => setForm((v) => v ? { ...v, gateway_protocol: value } : v)}>
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">SSE + Streamable HTTP</SelectItem>
                    <SelectItem value="sse">SSE</SelectItem>
                    <SelectItem value="streamhttp">Streamable HTTP</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="gc">Session GC 间隔（秒）</Label>
                <Input id="gc" type="number" value={form.session_gc_interval_seconds} onChange={(e) => setForm((v) => v ? { ...v, session_gc_interval_seconds: Number(e.target.value) } : v)} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="timeout">Session 超时（秒）</Label>
                <Input id="timeout" type="number" value={form.proxy_session_timeout_seconds} onChange={(e) => setForm((v) => v ? { ...v, proxy_session_timeout_seconds: Number(e.target.value) } : v)} />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>认证设置</CardTitle>
              <CardDescription>API 认证和授权配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              <div className="flex items-center justify-between rounded-lg border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">启用认证</Label>
                  <p className="text-sm text-muted-foreground">要求管理 API 和 MCP 请求携带有效 Bearer token</p>
                </div>
                <Switch checked={form.auth.enabled} onCheckedChange={(checked) => setAuthField('enabled', checked)} />
              </div>

              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label>账号模式</Label>
                  <Select value={form.auth.mode} onValueChange={(value) => setAuthField('mode', value)}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="saas">SaaS 账号体系</SelectItem>
                      <SelectItem value="single-key">Single Key</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="flex items-center justify-between rounded-lg border bg-muted/30 p-4">
                  <div className="space-y-0.5">
                    <Label>允许注册</Label>
                    <p className="text-sm text-muted-foreground">开放账号注册入口</p>
                  </div>
                  <Switch checked={form.auth.allow_register} disabled={form.auth.mode !== 'saas'} onCheckedChange={(checked) => setAuthField('allow_register', checked)} />
                </div>
              </div>

              <div className="space-y-2">
                <Label>MCP OAuth 登录方式</Label>
                <Select
                  value={authProvider}
                  onValueChange={(value) => {
                    if (value === 'internal') {
                      setForm((v) => v ? {
                        ...v,
                        auth: {
                          ...v.auth,
                          mode: 'saas',
                          authorization_servers: [],
                          token_issuer: '',
                          token_jwks_uri: '',
                          token_introspection_url: '',
                          token_introspection_id: '',
                          token_audience: '',
                          required_scopes: [],
                          scopes_supported: [],
                        },
                      } : v)
                    } else {
                      setAuthField('authorization_servers', form.auth.authorization_servers?.length ? form.auth.authorization_servers : [''])
                    }
                  }}
                >
                  <SelectTrigger><SelectValue /></SelectTrigger>
                  <SelectContent>
                    <SelectItem value="internal">内置 Gateway 账号密码登录</SelectItem>
                    <SelectItem value="external">外部 OAuth 服务</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {authProvider === 'internal' ? (
                <div className="rounded-lg border bg-muted/30 p-4 text-sm text-muted-foreground">
                  当前会通过 MCP Gateway 自身账号体系登录，客户端发现到的授权服务为当前 Gateway 地址，并使用 <span className="font-mono">/oauth/authorize</span> 与 <span className="font-mono">/oauth/token</span>。
                </div>
              ) : (
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2 sm:col-span-2">
                    <Label htmlFor="authorization-servers">授权服务器 Issuer</Label>
                    <Textarea
                      id="authorization-servers"
                      value={listValue(form.auth.authorization_servers)}
                      onChange={(e) => setAuthField('authorization_servers', splitList(e.target.value))}
                      rows={2}
                      placeholder="https://auth.example.com"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="token-issuer">Token Issuer</Label>
                    <Input id="token-issuer" value={form.auth.token_issuer || ''} onChange={(e) => setAuthField('token_issuer', e.target.value)} placeholder="https://auth.example.com" />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="jwks-uri">JWKS URI</Label>
                    <Input id="jwks-uri" value={form.auth.token_jwks_uri || ''} onChange={(e) => setAuthField('token_jwks_uri', e.target.value)} placeholder="https://auth.example.com/.well-known/jwks.json" />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="introspection-url">Introspection URL</Label>
                    <Input id="introspection-url" value={form.auth.token_introspection_url || ''} onChange={(e) => setAuthField('token_introspection_url', e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="introspection-id">Introspection Client ID</Label>
                    <Input id="introspection-id" value={form.auth.token_introspection_id || ''} onChange={(e) => setAuthField('token_introspection_id', e.target.value)} />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="token-audience">Token Audience</Label>
                    <Input id="token-audience" value={form.auth.token_audience || ''} onChange={(e) => setAuthField('token_audience', e.target.value)} placeholder="http://localhost:8080/stream" />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="required-scopes">Required Scopes</Label>
                    <Input id="required-scopes" value={(form.auth.required_scopes || []).join(' ')} onChange={(e) => setAuthField('required_scopes', e.target.value.split(/\s+/).filter(Boolean))} placeholder="mcp:read" />
                  </div>
                  <div className="space-y-2 sm:col-span-2">
                    <Label htmlFor="supported-scopes">Scopes Supported</Label>
                    <Input id="supported-scopes" value={(form.auth.scopes_supported || []).join(' ')} onChange={(e) => setAuthField('scopes_supported', e.target.value.split(/\s+/).filter(Boolean))} placeholder="mcp:read mcp:write" />
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="market" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>MCP 市场源</CardTitle>
              <CardDescription>管理系统可用的 MCP 市场 API 数据源</CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {marketSources.length === 0 ? (
                <div className="rounded-xl border bg-muted/30 p-4 text-sm text-muted-foreground">暂无市场源</div>
              ) : (
                marketSources.map((source) => (
                  <div key={source.id} className="flex flex-col gap-4 rounded-xl border bg-muted/20 p-4 sm:flex-row sm:items-center sm:justify-between">
                    <div className="min-w-0 space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <p className="font-medium">{source.name}</p>
                        <Badge variant={source.status === 'healthy' ? 'default' : 'secondary'}>{source.status}</Badge>
                        {source.trusted && <Badge variant="outline">trusted</Badge>}
                        {!source.enabled && <Badge variant="secondary">disabled</Badge>}
                      </div>
                      <p className="break-all text-sm text-muted-foreground">{source.url}</p>
                      <div className="flex flex-wrap gap-x-4 gap-y-1 text-xs text-muted-foreground">
                        <span>类型：{source.kind}</span>
                        <span>包数量：{source.total_items}</span>
                        <span>优先级：{source.priority}</span>
                        <span>最后同步：{source.last_synced ? new Date(source.last_synced).toLocaleString() : '未同步'}</span>
                      </div>
                      {source.last_error && <p className="text-xs text-destructive">{source.last_error}</p>}
                    </div>
                    <Button
                      variant="outline"
                      className="gap-2"
                      onClick={() => handleSyncSource(source.id)}
                      disabled={source.kind === 'local_market' || !source.enabled || !!syncingSource}
                    >
                      <RefreshCw className={cn('h-4 w-4', syncingSource === source.id && 'animate-spin')} />
                      同步
                    </Button>
                  </div>
                ))
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <CardTitle>自有市场 MCP</CardTitle>
                <CardDescription>维护 Gateway Local Market 中可供用户安装的 MCP</CardDescription>
              </div>
              <Button className="gap-2" onClick={openCreateMarketPackage}>
                <Plus className="h-4 w-4" />
                添加 MCP
              </Button>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {localPackages.length === 0 ? (
                  <div className="rounded-xl border bg-muted/30 p-4 text-sm text-muted-foreground">自有市场暂无 MCP</div>
                ) : (
                  localPackages.map((pkg) => (
                    <div key={pkg.id} className="flex flex-col gap-3 rounded-xl border bg-muted/20 p-4 sm:flex-row sm:items-center sm:justify-between">
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="font-medium">{pkg.title || pkg.name}</p>
                          <Badge variant="outline">{pkg.installability || 'manual'}</Badge>
                          <Badge variant="secondary">v{pkg.version || 'unknown'}</Badge>
                        </div>
                        <p className="mt-1 line-clamp-2 text-sm text-muted-foreground">{pkg.description}</p>
                      </div>
                      <div className="flex gap-2">
                        <Button variant="outline" size="sm" className="gap-2" onClick={() => editMarketPackage(pkg)}>
                          <Pencil className="h-4 w-4" />
                          编辑
                        </Button>
                        <Button variant="outline" size="sm" className="gap-2 text-destructive" onClick={() => deleteLocalMarketPackage(pkg.id)} disabled={deletingPackageId === pkg.id}>
                          <Trash2 className="h-4 w-4" />
                          {deletingPackageId === pkg.id ? '删除中...' : '删除'}
                        </Button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="notifications" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>通知偏好</CardTitle>
              <CardDescription>为后续通知能力预留的页面配置</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">服务异常提醒</Label>
                  <p className="text-sm text-muted-foreground">MCP 启动失败或异常停止时提醒</p>
                </div>
                <Switch defaultChecked />
              </div>
              <div className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">会话阈值提醒</Label>
                  <p className="text-sm text-muted-foreground">会话数过高时提醒</p>
                </div>
                <Switch />
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="advanced" className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>高级参数</CardTitle>
              <CardDescription>更细粒度的运行时参数</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="retry">MCP 重试次数</Label>
                <Input id="retry" type="number" value={form.mcp_retry_count} onChange={(e) => setForm((v) => v ? { ...v, mcp_retry_count: Number(e.target.value) } : v)} />
              </div>
              <div className="rounded-xl border bg-muted/30 p-4 text-sm text-muted-foreground">
                当前模式：{form.auth.mode}，注册能力：{form.auth.allow_register ? '开启' : '关闭'}
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <Sheet open={marketSheetOpen} onOpenChange={setMarketSheetOpen}>
        <SheetContent side="right" className="w-full overflow-y-auto sm:max-w-xl">
          <SheetHeader>
            <SheetTitle>{editingPackageId ? '编辑 MCP' : '添加 MCP'}</SheetTitle>
            <SheetDescription>配置自有市场中的 MCP 元数据和安装方式</SheetDescription>
          </SheetHeader>

          <div className="space-y-5 px-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="market-name">标识</Label>
                <Input id="market-name" value={marketForm.name} onChange={(e) => setMarketForm((v) => ({ ...v, name: e.target.value }))} placeholder="my-mcp-server" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="market-title">名称</Label>
                <Input id="market-title" value={marketForm.title} onChange={(e) => setMarketForm((v) => ({ ...v, title: e.target.value }))} placeholder="My MCP Server" />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="market-description">描述</Label>
              <Textarea id="market-description" value={marketForm.description} onChange={(e) => setMarketForm((v) => ({ ...v, description: e.target.value }))} rows={4} />
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="market-version">版本</Label>
                <Input id="market-version" value={marketForm.version} onChange={(e) => setMarketForm((v) => ({ ...v, version: e.target.value }))} />
              </div>
              <div className="space-y-2">
                <Label htmlFor="market-category">分类</Label>
                <Input id="market-category" value={marketForm.category} onChange={(e) => setMarketForm((v) => ({ ...v, category: e.target.value }))} />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="market-tags">标签</Label>
              <Input id="market-tags" value={marketForm.tags} onChange={(e) => setMarketForm((v) => ({ ...v, tags: e.target.value }))} placeholder="github, repo, automation" />
            </div>

            <div className="space-y-2">
              <Label>安装方式</Label>
              <Select value={marketForm.installType} onValueChange={(value) => setMarketForm((v) => ({ ...v, installType: value }))}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="command">Command</SelectItem>
                  <SelectItem value="npx">npx</SelectItem>
                  <SelectItem value="uvx">uvx</SelectItem>
                  <SelectItem value="remote">Remote URL</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {marketForm.installType === 'remote' ? (
              <div className="space-y-2">
                <Label htmlFor="market-url">URL</Label>
                <Input id="market-url" value={marketForm.url} onChange={(e) => setMarketForm((v) => ({ ...v, url: e.target.value }))} placeholder="https://example.com/mcp" />
              </div>
            ) : (
              <div className="grid gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="market-command">命令</Label>
                  <Input id="market-command" value={marketForm.command} onChange={(e) => setMarketForm((v) => ({ ...v, command: e.target.value }))} placeholder="npx / uvx / python3" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="market-args">参数</Label>
                  <Input id="market-args" value={marketForm.args} onChange={(e) => setMarketForm((v) => ({ ...v, args: e.target.value }))} placeholder="-y @scope/mcp-server" />
                </div>
              </div>
            )}
          </div>

          <SheetFooter className="border-t">
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:justify-end">
              <Button variant="outline" className="gap-2" onClick={() => setMarketSheetOpen(false)}>
                <X className="h-4 w-4" />
                取消
              </Button>
              <Button className="gap-2" onClick={saveLocalMarketPackage} disabled={savingMarketPackage || !marketForm.description.trim() || (!marketForm.name.trim() && !marketForm.title.trim())}>
                {editingPackageId ? <Save className="h-4 w-4" /> : <Plus className="h-4 w-4" />}
                {savingMarketPackage ? '提交中...' : editingPackageId ? '保存 MCP' : '添加 MCP'}
              </Button>
            </div>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <Button onClick={handleSave} disabled={savingConfig}>
        {saved ? <Check className="mr-2 h-4 w-4" /> : <Save className="mr-2 h-4 w-4" />}
        {savingConfig ? '保存中...' : saved ? '已保存' : '保存配置'}
      </Button>
    </div>
  )
}
