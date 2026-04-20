'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import {
  Calendar,
  MoreHorizontal,
  Package,
  Play,
  RefreshCw,
  Search,
  Settings,
  Square,
  Wrench,
} from 'lucide-react'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Switch } from '@/components/ui/switch'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'
import { gatewayApi, invalidate, useGatewaySWR, type InstalledItem, type ListData } from '@/lib/gateway-api'

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

  const items = useMemo(() => {
    const list = data?.items || []
    if (!searchQuery) return list
    const q = searchQuery.toLowerCase()
    return list.filter((item) => `${item.package_name} ${item.workspace_name} ${item.service_name}`.toLowerCase().includes(q))
  }, [data?.items, searchQuery])

  async function handleStatusAction(item: InstalledItem) {
    if (item.status === 'running') {
      await gatewayApi.stopService(item.workspace_id, item.service_name)
    } else {
      await gatewayApi.startService(item.workspace_id, item.service_name)
    }
    await Promise.all([invalidate('/api/v1/installed'), invalidate(`/api/v1/workspaces/${item.workspace_id}/services`)])
  }

  function openConfig(item: InstalledItem) {
    setSelectedItem(item)
    setIsConfigOpen(true)
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">已安装 MCP</h2>
          <p className="text-muted-foreground">查看不同工作空间中已安装的 MCP 服务</p>
        </div>
        <Button variant="outline" onClick={() => invalidate('/api/v1/installed')}>
          <RefreshCw className="mr-2 h-4 w-4" />
          刷新
        </Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-violet-100 dark:bg-violet-900/30">
                <Package className="h-6 w-6 text-violet-600 dark:text-violet-400" />
              </div>
              <div>
                <p className="text-3xl font-bold">{data?.items.length ?? 0}</p>
                <p className="text-sm text-muted-foreground">已安装</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-100 dark:bg-emerald-900/30">
                <Play className="h-6 w-6 text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <p className="text-3xl font-bold text-emerald-600 dark:text-emerald-400">
                  {data?.items.filter((v) => v.status === 'running').length ?? 0}
                </p>
                <p className="text-sm text-muted-foreground">运行中</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-amber-100 dark:bg-amber-900/30">
                <RefreshCw className="h-6 w-6 text-amber-600 dark:text-amber-400" />
              </div>
              <div>
                <p className="text-3xl font-bold text-amber-600 dark:text-amber-400">
                  {data?.items.filter((v) => v.installed_version !== v.latest_version).length ?? 0}
                </p>
                <p className="text-sm text-muted-foreground">可升级</p>
              </div>
            </div>
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
          <Card key={`${item.workspace_id}-${item.service_name}`} className="group transition-all hover:border-primary/20 hover:shadow-sm">
            <CardContent className="p-5">
              <div className="flex items-center justify-between gap-4">
                <div className="flex min-w-0 items-center gap-4">
                  <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-muted text-lg font-semibold text-primary">
                    {item.package_name.slice(0, 2).toUpperCase()}
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-semibold">{item.package_name}</h3>
                      <Badge
                        className={cn(
                          'border-0',
                          item.status === 'running'
                            ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                            : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                        )}
                      >
                        {item.status === 'running' ? '运行中' : item.status === 'failed' ? '错误' : '已停止'}
                      </Badge>
                      {item.installed_version !== item.latest_version && (
                        <Badge variant="outline" className="border-amber-200 text-amber-700 dark:border-amber-800 dark:text-amber-400">
                          可升级至 v{item.latest_version}
                        </Badge>
                      )}
                    </div>
                    <div className="mt-1 text-sm text-muted-foreground">
                      服务名 {item.service_name}，部署在 {item.workspace_name}
                    </div>
                    <div className="mt-2 flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
                      <span className="flex items-center gap-1">
                        <Wrench className="h-3 w-3" />
                        工作空间 {item.workspace_name}
                      </span>
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        {formatDate(item.installed_at)}
                      </span>
                      <span className="flex items-center gap-1">
                        <Package className="h-3 w-3" />
                        当前 v{item.installed_version}
                      </span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <Button variant="outline" size="sm" onClick={() => handleStatusAction(item)}>
                    {item.status === 'running' ? <Square className="mr-2 h-4 w-4" /> : <Play className="mr-2 h-4 w-4" />}
                    {item.status === 'running' ? '停止' : '启动'}
                  </Button>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon" className="h-9 w-9">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={() => openConfig(item)}>
                        <Settings className="mr-2 h-4 w-4" />
                        配置预览
                      </DropdownMenuItem>
                      <DropdownMenuItem asChild>
                        <Link href={`/workspaces/${item.workspace_id}`}>前往工作空间</Link>
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem onClick={() => invalidate('/api/v1/installed')}>
                        <RefreshCw className="mr-2 h-4 w-4" />
                        刷新状态
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
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
              <DialogTitle>{selectedItem.package_name} 配置预览</DialogTitle>
              <DialogDescription>保留原型阶段的配置体验，但当前仍以真实运行数据为准。</DialogDescription>
            </DialogHeader>

            <Tabs defaultValue="meta" className="mt-2">
              <TabsList className="inline-flex h-auto gap-1 rounded-lg bg-muted p-1">
                <TabsTrigger value="meta" className="rounded-md px-3 py-1.5 text-sm">基本信息</TabsTrigger>
                <TabsTrigger value="tools" className="rounded-md px-3 py-1.5 text-sm">工具权限</TabsTrigger>
              </TabsList>

              <TabsContent value="meta" className="mt-4 space-y-4">
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label>包名</Label>
                    <div className="rounded-xl border bg-muted/30 px-4 py-3 text-sm">{selectedItem.package_name}</div>
                  </div>
                  <div className="space-y-2">
                    <Label>服务名</Label>
                    <div className="rounded-xl border bg-muted/30 px-4 py-3 text-sm">{selectedItem.service_name}</div>
                  </div>
                  <div className="space-y-2">
                    <Label>工作空间</Label>
                    <div className="rounded-xl border bg-muted/30 px-4 py-3 text-sm">{selectedItem.workspace_name}</div>
                  </div>
                  <div className="space-y-2">
                    <Label>版本</Label>
                    <div className="rounded-xl border bg-muted/30 px-4 py-3 text-sm">
                      当前 v{selectedItem.installed_version} / 最新 v{selectedItem.latest_version}
                    </div>
                  </div>
                </div>
              </TabsContent>

              <TabsContent value="tools" className="mt-4">
                <div className="space-y-3">
                  {['工具发现', '执行权限', '日志采集'].map((tool) => (
                    <div key={tool} className="flex items-center justify-between rounded-xl border bg-muted/30 p-4">
                      <div>
                        <p className="font-medium">{tool}</p>
                        <p className="text-sm text-muted-foreground">保留原型的细节配置表现，后续可继续接真实权限模型。</p>
                      </div>
                      <Switch defaultChecked />
                    </div>
                  ))}
                </div>
              </TabsContent>
            </Tabs>

            <DialogFooter>
              <Button variant="outline" onClick={() => setIsConfigOpen(false)}>关闭</Button>
              <Button asChild>
                <Link href={`/workspaces/${selectedItem.workspace_id}`}>前往工作空间继续配置</Link>
              </Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  )
}
