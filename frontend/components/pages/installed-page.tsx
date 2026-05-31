'use client'

import { useMemo, useState } from 'react'
import Link from '@/components/router-link'
import {
  Calendar,
  CheckCircle,
  ExternalLink,
  MoreHorizontal,
  Package,
  RefreshCw,
  Search,
  Settings,
  Trash2,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { gatewayApi, invalidate, useGatewaySWR, type InstalledItem, type ListData } from '@/lib/gateway-api'
import { runAction } from '@/lib/action-feedback'

function parseEnv(value: string) {
  const env: Record<string, string> = {}
  value.split('\n').forEach((line) => {
    const [key, ...valueParts] = line.split('=')
    if (key.trim() && valueParts.length > 0) {
      env[key.trim()] = valueParts.join('=').trim()
    }
  })
  return env
}

function formatEnv(env?: Record<string, unknown>) {
  return Object.entries(env || {}).map(([key, value]) => `${key}=${String(value)}`).join('\n')
}

function formatDate(dateString: string) {
  return new Date(dateString).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function InstalledPage() {
  const { data } = useGatewaySWR<ListData<InstalledItem>>('/api/v1/installed')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedItem, setSelectedItem] = useState<InstalledItem | null>(null)
  const [isConfigOpen, setIsConfigOpen] = useState(false)
  const [configDisplayName, setConfigDisplayName] = useState('')
  const [configArgs, setConfigArgs] = useState('')
  const [configEnv, setConfigEnv] = useState('')
  const [savingConfig, setSavingConfig] = useState(false)
  const [authorizingId, setAuthorizingId] = useState('')
  const [deletingId, setDeletingId] = useState('')

  const items = useMemo(() => {
    const list = data?.items || []
    if (!searchQuery) return list
    const q = searchQuery.toLowerCase()
    return list.filter((item) => `${item.display_name} ${item.package_name} ${item.package_id} ${item.source_id || ''}`.toLowerCase().includes(q))
  }, [data?.items, searchQuery])

  async function handleDelete(item: InstalledItem) {
    setDeletingId(item.id)
    await runAction(
      async () => {
        await gatewayApi.deleteInstalled(item.id)
        await invalidate('/api/v1/installed')
      },
      { successTitle: '删除成功', successDescription: '已安装记录已删除', errorTitle: '删除失败' }
    )
    setDeletingId('')
  }

  function openConfig(item: InstalledItem) {
    setSelectedItem(item)
    setConfigDisplayName(item.display_name || item.package_name)
    setConfigArgs(((item.config_snapshot?.args as string[] | undefined) || []).join('\n'))
    setConfigEnv(formatEnv(item.config_snapshot?.env as Record<string, unknown> | undefined))
    setIsConfigOpen(true)
  }

  async function handleSaveConfig() {
    if (!selectedItem) return
    setSavingConfig(true)
    const ok = await runAction(
      async () => {
        await gatewayApi.updateInstalled(selectedItem.id, {
          display_name: configDisplayName,
          args: configArgs.split('\n').map((arg) => arg.trim()).filter(Boolean),
          env: parseEnv(configEnv),
        })
        await invalidate('/api/v1/installed')
      },
      { successTitle: '保存成功', successDescription: '配置快照已更新', errorTitle: '保存失败' }
    )
    setSavingConfig(false)
    if (ok) setIsConfigOpen(false)
  }

  async function handleCompleteOAuth(item: InstalledItem) {
    setAuthorizingId(item.id)
    const ok = await runAction(
      async () => {
        await gatewayApi.completeInstalledOAuth(item.id)
        await invalidate('/api/v1/installed')
      },
      { successTitle: '鉴权已确认', successDescription: '现在可以添加到工作空间', errorTitle: '确认失败' }
    )
    setAuthorizingId('')
    if (ok) setIsConfigOpen(false)
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">已安装 MCP</h2>
          <p className="text-muted-foreground">当前账号保存的 MCP 配置快照，可在工作空间中选择添加</p>
        </div>
        <Button variant="outline" onClick={() => invalidate('/api/v1/installed')}>
          <RefreshCw className="mr-2 h-4 w-4" />
          刷新
        </Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="p-6">
            <p className="text-3xl font-bold">{data?.items.length ?? 0}</p>
            <p className="text-sm text-muted-foreground">账号已安装</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-3xl font-bold">{data?.items.filter((v) => v.source_id === 'local').length ?? 0}</p>
            <p className="text-sm text-muted-foreground">自有市场</p>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <p className="text-3xl font-bold">{data?.items.filter((v) => v.installed_version !== v.latest_version).length ?? 0}</p>
            <p className="text-sm text-muted-foreground">可更新快照</p>
          </CardContent>
        </Card>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索已安装 MCP..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-full border-transparent bg-muted/50 pl-9 focus:border-border focus:bg-background sm:max-w-sm"
        />
      </div>

      <div className="space-y-3">
        {items.map((item) => (
          <Card key={item.id} className="group transition-all hover:border-primary/20 hover:shadow-sm">
            <CardContent className="p-5">
              <div className="flex items-center justify-between gap-4">
                <div className="flex min-w-0 items-center gap-4">
                  <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-muted text-lg font-semibold text-primary">
                    {(item.display_name || item.package_name).slice(0, 2).toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h3 className="font-semibold">{item.display_name || item.package_name}</h3>
                      <Badge variant="secondary">{item.source_id || 'unknown'}</Badge>
                      {item.installed_version !== item.latest_version && (
                        <Badge variant="outline">最新 v{item.latest_version}</Badge>
                      )}
                      {item.auth?.type === 'oauth2' && (
                        <Badge variant={item.auth.status === 'authorized' ? 'default' : 'secondary'}>
                          {item.auth.status === 'authorized' ? 'OAuth 已授权' : 'OAuth 待授权'}
                        </Badge>
                      )}
                    </div>
                    <div className="mt-1 text-sm text-muted-foreground">{item.package_id}</div>
                    <div className="mt-2 flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        {formatDate(item.installed_at)}
                      </span>
                      <span className="flex items-center gap-1">
                        <Package className="h-3 w-3" />
                        快照 v{item.installed_version}
                      </span>
                    </div>
                  </div>
                </div>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-9 w-9">
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={() => openConfig(item)}>
                      <Settings className="mr-2 h-4 w-4" />
                      配置快照
                    </DropdownMenuItem>
                    <DropdownMenuItem asChild>
                      <Link href="/workspaces">添加到工作空间</Link>
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={() => handleDelete(item)} disabled={deletingId === item.id} className="text-destructive">
                      <Trash2 className="mr-2 h-4 w-4" />
                      {deletingId === item.id ? '删除中...' : '删除安装记录'}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </CardContent>
          </Card>
        ))}
        {items.length === 0 && <p className="text-sm text-muted-foreground">暂无已安装 MCP。</p>}
      </div>

      <Dialog open={isConfigOpen} onOpenChange={setIsConfigOpen}>
        {selectedItem && (
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>{selectedItem.display_name || selectedItem.package_name} 配置快照</DialogTitle>
              <DialogDescription>这是安装到账号时保存的配置快照，添加到工作空间时会基于它创建服务。</DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label>包 ID</Label>
                <div className="rounded-lg border bg-muted/30 px-4 py-3 text-sm">{selectedItem.package_id}</div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="installed-display-name">显示名称</Label>
                <Input id="installed-display-name" value={configDisplayName} onChange={(e) => setConfigDisplayName(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>市场源</Label>
                <div className="rounded-lg border bg-muted/30 px-4 py-3 text-sm">{selectedItem.source_id || 'unknown'}</div>
              </div>
              <div className="space-y-2">
                <Label>安装方式</Label>
                <div className="rounded-lg border bg-muted/30 px-4 py-3 text-sm">#{selectedItem.install_option_index}</div>
              </div>
              <div className="space-y-2">
                <Label>版本</Label>
                <div className="rounded-lg border bg-muted/30 px-4 py-3 text-sm">v{selectedItem.installed_version}</div>
              </div>
            </div>
            {selectedItem.auth?.type === 'oauth2' && (
              <div className="rounded-lg border bg-muted/30 p-4 text-sm">
                <div className="flex items-center justify-between gap-3">
                  <div>
                    <p className="font-medium">OAuth 2.0 鉴权 · {selectedItem.auth.status === 'authorized' ? '已授权' : '待授权'}</p>
                    <p className="mt-1 text-muted-foreground">{selectedItem.auth.instructions || '添加到工作空间前需要完成授权。'}</p>
                  </div>
                  {selectedItem.auth.authorization_url && (
                    <Button variant="outline" size="sm" onClick={() => window.open(selectedItem.auth?.authorization_url, '_blank', 'noopener,noreferrer')}>
                      <ExternalLink className="mr-2 h-4 w-4" />
                      打开鉴权
                    </Button>
                  )}
                </div>
              </div>
            )}
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="installed-args">参数</Label>
                <Textarea id="installed-args" value={configArgs} onChange={(e) => setConfigArgs(e.target.value)} rows={5} placeholder="每行一个参数" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="installed-env">环境变量</Label>
                <Textarea id="installed-env" value={configEnv} onChange={(e) => setConfigEnv(e.target.value)} rows={5} placeholder="KEY=VALUE，每行一个" />
              </div>
            </div>
            <pre className="max-h-72 overflow-auto rounded-lg border bg-muted/30 p-4 text-xs">
              {JSON.stringify(selectedItem.config_snapshot || {}, null, 2)}
            </pre>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsConfigOpen(false)}>关闭</Button>
              {selectedItem.auth?.type === 'oauth2' && selectedItem.auth.status !== 'authorized' && (
                <Button variant="outline" onClick={() => handleCompleteOAuth(selectedItem)} disabled={authorizingId === selectedItem.id}>
                  <CheckCircle className="mr-2 h-4 w-4" />
                  {authorizingId === selectedItem.id ? '确认中...' : '已完成鉴权'}
                </Button>
              )}
              <Button variant="outline" onClick={handleSaveConfig} disabled={savingConfig}>
                {savingConfig ? '保存中...' : '保存配置'}
              </Button>
              <Button asChild>
                <Link href="/workspaces">选择工作空间添加</Link>
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  )
}
