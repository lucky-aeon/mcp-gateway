'use client'

import { useMemo, useState } from 'react'
import { Check, Copy, Eye, EyeOff, Key, Plus, RefreshCw, Shield, Clock, Trash2 } from 'lucide-react'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
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
import { gatewayApi, invalidate, saveGatewayApiKey, useGatewaySWR, type ListData, type MetaInfo, type UserAPIKey, type Workspace } from '@/lib/gateway-api'
import { useAppStore } from '@/lib/store'

function maskKey(key: string) {
  if (key.length < 12) return key
  return `${key.slice(0, 6)}••••••${key.slice(-4)}`
}

export function APIKeysPage() {
  const { data: meta } = useGatewaySWR<MetaInfo>('/api/v1/meta')
  const singleKeyMode = meta?.mode !== 'saas'

  if (singleKeyMode) {
    return <SingleKeyAPIKeysPage />
  }

  return <SaaSAPIKeysPage />
}

function SingleKeyAPIKeysPage() {
  const { data } = useGatewaySWR<{ api_key: string; updated_at: string }>('/api/v1/system/api-key')
  const [revealed, setRevealed] = useState(false)
  const [copied, setCopied] = useState(false)
  const [rotating, setRotating] = useState(false)
  const [latestKey, setLatestKey] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)

  async function copyKey(key: string) {
    await navigator.clipboard.writeText(key)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  const currentKey = latestKey || data?.api_key || ''

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">API 密钥</h2>
        <p className="text-muted-foreground">当前实例运行在 single-key 模式，所有客户端共用同一个密钥。</p>
      </div>

      <Alert>
        <Shield className="h-4 w-4" />
        <AlertTitle>Single-key 模式</AlertTitle>
        <AlertDescription>轮换后请同步更新所有接入 Gateway 的客户端配置。</AlertDescription>
      </Alert>

      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard icon={Key} tone="violet" title="总密钥数" value="1" />
        <StatCard icon={Shield} tone="emerald" title="活跃密钥" value="1" />
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-slate-100 dark:bg-slate-800">
                <Clock className="h-6 w-6 text-slate-600 dark:text-slate-400" />
              </div>
              <div>
                <p className="text-sm text-muted-foreground">最近更新</p>
                <p className="text-sm font-medium">{data?.updated_at ? new Date(data.updated_at).toLocaleString('zh-CN') : '未知'}</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <div>
            <CardTitle>当前共享密钥</CardTitle>
            <CardDescription>single-key 模式下，控制面与流量面共用同一把密钥。</CardDescription>
          </div>
          <Dialog
            open={dialogOpen}
            onOpenChange={(open) => {
              setDialogOpen(open)
              if (!open) setLatestKey(null)
            }}
          >
            <DialogTrigger asChild>
              <Button>
                <RefreshCw className="mr-2 h-4 w-4" />
                轮换密钥
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-md">
              {latestKey ? (
                <>
                  <DialogHeader>
                    <DialogTitle>密钥已轮换</DialogTitle>
                    <DialogDescription>请立即复制新密钥并同步到所有客户端。</DialogDescription>
                  </DialogHeader>
                  <div className="rounded-xl bg-muted p-4 font-mono text-sm break-all">{latestKey}</div>
                  <DialogFooter>
                    <Button variant="outline" onClick={() => copyKey(latestKey)}>
                      <Copy className="mr-2 h-4 w-4" />
                      复制
                    </Button>
                    <Button onClick={() => setDialogOpen(false)}>完成</Button>
                  </DialogFooter>
                </>
              ) : (
                <>
                  <DialogHeader>
                    <DialogTitle>轮换 API Key</DialogTitle>
                    <DialogDescription>旧密钥会立刻失效，所有接入客户端都需要改用新值。</DialogDescription>
                  </DialogHeader>
                  <DialogFooter>
                    <Button variant="outline" onClick={() => setDialogOpen(false)}>取消</Button>
                    <Button
                      disabled={rotating}
                      onClick={async () => {
                        setRotating(true)
                        try {
                          const result = await gatewayApi.rotateSystemApiKey()
                          saveGatewayApiKey(result.api_key)
                          setLatestKey(result.api_key)
                          await invalidate('/api/v1/system/api-key')
                        } finally {
                          setRotating(false)
                        }
                      }}
                    >
                      {rotating ? '轮换中...' : '确认轮换'}
                    </Button>
                  </DialogFooter>
                </>
              )}
            </DialogContent>
          </Dialog>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2 rounded-xl bg-muted p-4">
            <code className="flex-1 break-all text-sm font-mono">{revealed ? currentKey : maskKey(currentKey)}</code>
            <Button variant="ghost" size="icon" onClick={() => setRevealed((v) => !v)}>
              {revealed ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </Button>
            <Button variant="ghost" size="icon" onClick={() => copyKey(currentKey)}>
              {copied ? <Check className="h-4 w-4 text-emerald-500" /> : <Copy className="h-4 w-4" />}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function SaaSAPIKeysPage() {
  const { data: keysData } = useGatewaySWR<ListData<UserAPIKey>>('/api/v1/api-keys')
  const { data: workspacesData } = useGatewaySWR<ListData<Workspace>>('/api/v1/workspaces')
  const { currentUser } = useAppStore()
  const [dialogOpen, setDialogOpen] = useState(false)
  const [createdKey, setCreatedKey] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState({
    name: '',
    workspaceId: 'personal',
    scope: 'gateway:read,gateway:write',
    expiresAt: '',
  })

  const items = keysData?.items || []
  const activeCount = useMemo(() => items.filter((item) => item.status === 'active').length, [items])

  async function copy(text: string) {
    await navigator.clipboard.writeText(text)
  }

  async function handleCreate() {
    setCreating(true)
    try {
      const result = await gatewayApi.createAPIKey({
        name: form.name,
        workspace_id: form.workspaceId === 'personal' ? undefined : form.workspaceId,
        scope: form.scope.split(',').map((item) => item.trim()).filter(Boolean),
        expires_at: form.expiresAt ? new Date(form.expiresAt).toISOString() : undefined,
      })
      setCreatedKey(result.raw_key || null)
      await invalidate('/api/v1/api-keys')
    } finally {
      setCreating(false)
    }
  }

  async function handleRevoke(id: string) {
    await gatewayApi.revokeAPIKey(id)
    await invalidate('/api/v1/api-keys')
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">API 密钥</h2>
          {currentUser?.builtin === true && (
            <p className="text-muted-foreground">SaaS 模式下，密钥属于当前账号，并可限制到特定工作空间。</p>
          )}
        </div>
        <Dialog
          open={dialogOpen}
          onOpenChange={(open) => {
            setDialogOpen(open)
            if (!open) {
              setCreatedKey(null)
              setForm({ name: '', workspaceId: 'personal', scope: 'gateway:read,gateway:write', expiresAt: '' })
            }
          }}
        >
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              新建密钥
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-lg">
            {createdKey ? (
              <>
                <DialogHeader>
                  <DialogTitle>密钥已创建</DialogTitle>
                  <DialogDescription>明文只展示这一次，请立即复制保存。</DialogDescription>
                </DialogHeader>
                <div className="rounded-xl bg-muted p-4 font-mono text-sm break-all">{createdKey}</div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => copy(createdKey)}>
                    <Copy className="mr-2 h-4 w-4" />
                    复制
                  </Button>
                  <Button onClick={() => setDialogOpen(false)}>完成</Button>
                </DialogFooter>
              </>
            ) : (
              <>
                <DialogHeader>
                  <DialogTitle>创建 API 密钥</DialogTitle>
                  <DialogDescription>为当前账号创建个人或工作空间范围的访问凭证。</DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-2">
                  <div className="space-y-2">
                    <Label htmlFor="key-name">名称</Label>
                    <Input id="key-name" value={form.name} onChange={(e) => setForm((v) => ({ ...v, name: e.target.value }))} placeholder="例如：CI Runner" />
                  </div>
                  <div className="space-y-2">
                    <Label>绑定工作空间</Label>
                    <Select value={form.workspaceId} onValueChange={(value) => setForm((v) => ({ ...v, workspaceId: value }))}>
                      <SelectTrigger><SelectValue /></SelectTrigger>
                      <SelectContent>
                        <SelectItem value="personal">仅当前账号</SelectItem>
                        {(workspacesData?.items || []).map((workspace) => (
                          <SelectItem key={workspace.id} value={workspace.id}>{workspace.name}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="key-scope">Scope</Label>
                    <Input id="key-scope" value={form.scope} onChange={(e) => setForm((v) => ({ ...v, scope: e.target.value }))} placeholder="gateway:read,gateway:write" />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="key-expire">过期时间</Label>
                    <Input id="key-expire" type="datetime-local" value={form.expiresAt} onChange={(e) => setForm((v) => ({ ...v, expiresAt: e.target.value }))} />
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setDialogOpen(false)}>取消</Button>
                  <Button disabled={!form.name || creating} onClick={handleCreate}>
                    {creating ? '创建中...' : '创建'}
                  </Button>
                </DialogFooter>
              </>
            )}
          </DialogContent>
        </Dialog>
      </div>

      {currentUser?.builtin === true && (
        <Alert>
          <Shield className="h-4 w-4" />
          <AlertTitle>SaaS 模式</AlertTitle>
          <AlertDescription>建议为不同客户端分别创建密钥，并根据需要限制到特定工作空间。</AlertDescription>
        </Alert>
      )}

      <div className="grid gap-4 sm:grid-cols-3">
        <StatCard icon={Key} tone="violet" title="总密钥数" value={String(items.length)} />
        <StatCard icon={Shield} tone="emerald" title="活跃密钥" value={String(activeCount)} />
        <StatCard icon={Clock} tone="slate" title="最近新增" value={items[0]?.created_at ? new Date(items[0].created_at).toLocaleDateString('zh-CN') : '暂无'} compact />
      </div>

      <div className="space-y-3">
        {items.map((item) => (
          <Card key={item.id} className="transition-all hover:border-primary/20 hover:shadow-sm">
            <CardContent className="p-5">
              <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
                <div className="min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <h3 className="font-semibold">{item.name}</h3>
                    <Badge variant="outline">{item.key_prefix}...</Badge>
                    <Badge className={item.status === 'active' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' : ''}>
                      {item.status === 'active' ? '活跃' : item.status}
                    </Badge>
                  </div>
                  <div className="mt-2 flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
                    <span>范围：{item.workspace_id || '当前账号'}</span>
                    <span>Scope：{item.scope.join(', ') || '未设置'}</span>
                    <span>创建于 {new Date(item.created_at).toLocaleString('zh-CN')}</span>
                    <span>最近使用 {item.last_used_at ? new Date(item.last_used_at).toLocaleString('zh-CN') : '暂无'}</span>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" onClick={() => copy(item.key_prefix)}>
                    <Copy className="mr-2 h-4 w-4" />
                    复制前缀
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => handleRevoke(item.id)}>
                    <Trash2 className="mr-2 h-4 w-4" />
                    吊销
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
        {items.length === 0 && <p className="text-sm text-muted-foreground">还没有创建过 API 密钥。</p>}
      </div>
    </div>
  )
}

function StatCard({
  icon: Icon,
  tone,
  title,
  value,
  compact = false,
}: {
  icon: React.ComponentType<{ className?: string }>
  tone: 'violet' | 'emerald' | 'slate'
  title: string
  value: string
  compact?: boolean
}) {
  const toneMap = {
    violet: 'bg-violet-100 dark:bg-violet-900/30 text-violet-600 dark:text-violet-400',
    emerald: 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400',
    slate: 'bg-slate-100 dark:bg-slate-800 text-slate-600 dark:text-slate-400',
  }

  return (
    <Card>
      <CardContent className="p-6">
        <div className="flex items-center gap-4">
          <div className={`flex h-12 w-12 items-center justify-center rounded-xl ${toneMap[tone]}`}>
            <Icon className="h-6 w-6" />
          </div>
          <div>
            {compact ? (
              <>
                <p className="text-sm text-muted-foreground">{title}</p>
                <p className="text-sm font-medium">{value}</p>
              </>
            ) : (
              <>
                <p className="text-3xl font-bold">{value}</p>
                <p className="text-sm text-muted-foreground">{title}</p>
              </>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
