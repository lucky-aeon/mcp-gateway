'use client'

import { useEffect, useMemo, useState } from 'react'
import { Check, Download, Package, Search, Star, Store } from 'lucide-react'

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
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { cn } from '@/lib/utils'
import { gatewayApi, invalidate, useGatewaySWR, type ListData, type MarketPackage, type MarketSource } from '@/lib/gateway-api'
import { runAction } from '@/lib/action-feedback'

const categories = ['全部', '系统', '数据', '网络', '开发', '通讯', '效率']

export function MarketPage() {
  const { data: sourcesData } = useGatewaySWR<ListData<MarketSource>>('/api/v1/market/sources')
  const { data: installedData } = useGatewaySWR<{ items: Array<{ package_id: string }> }>('/api/v1/installed')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('全部')
  const [selectedSource, setSelectedSource] = useState('')
  const marketPath = selectedSource ? `/api/v1/market/packages?source=${encodeURIComponent(selectedSource)}` : null
  const { data: marketData } = useGatewaySWR<ListData<MarketPackage>>(marketPath)
  const [selectedMCP, setSelectedMCP] = useState<MarketPackage | null>(null)
  const [installOptionIndex, setInstallOptionIndex] = useState(0)
  const [installingPackageId, setInstallingPackageId] = useState('')

  const installedIds = new Set((installedData?.items || []).map((m) => m.package_id))
  const sources = sourcesData?.items || []
  const selectedInstallOptions = selectedMCP?.install_options || []
  const selectedTools = selectedMCP?.tools || []

  useEffect(() => {
    if (!selectedSource && sources.length > 0) {
      setSelectedSource(sources[0].id)
    }
  }, [selectedSource, sources])

  const filteredMCPs = useMemo(() => {
    const items = marketData?.items || []
    return items.filter((mcp) => {
      const matchesSearch = `${mcp.title || mcp.name} ${mcp.description} ${mcp.canonical_name || ''}`.toLowerCase().includes(searchQuery.toLowerCase())
      const matchesCategory = selectedCategory === '全部' || mcp.category === selectedCategory
      const matchesSource = !selectedSource || (mcp.source_refs || []).some((source) => source.source_id === selectedSource) || mcp.source_id === selectedSource
      return matchesSearch && matchesCategory && matchesSource
    })
  }, [marketData?.items, searchQuery, selectedCategory, selectedSource])

  async function handleInstall(pkg: MarketPackage) {
    setInstallingPackageId(pkg.id)
    const ok = await runAction(
      async () => {
        await gatewayApi.installMarketPackage(pkg.id, {
          display_name: pkg.title || pkg.name,
          install_option_index: installOptionIndex,
        })
        await Promise.all([invalidate('/api/v1/market/packages'), invalidate('/api/v1/installed')])
      },
      { successTitle: '安装成功', successDescription: 'MCP 已保存到当前账号', errorTitle: '安装失败' }
    )
    setInstallingPackageId('')
    if (ok) setSelectedMCP(null)
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">MCP 市场</h2>
        <p className="text-muted-foreground">先选择市场源，再浏览和安装该来源的 MCP 服务</p>
      </div>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
        {sources.map((source) => (
          <button
            key={source.id}
            type="button"
            onClick={() => setSelectedSource(source.id)}
            className={cn(
              'flex min-h-28 flex-col items-start justify-between rounded-lg border bg-card p-4 text-left transition-all hover:border-primary/30 hover:bg-muted/20',
              selectedSource === source.id && 'border-primary bg-primary/5'
            )}
          >
            <div className="flex w-full items-start justify-between gap-3">
              <div className="min-w-0">
                <div className="flex items-center gap-2">
                  <Store className="h-4 w-4 text-muted-foreground" />
                  <span className="truncate font-medium">{source.name}</span>
                </div>
                <p className="mt-1 truncate text-xs text-muted-foreground">{source.url}</p>
              </div>
              <Badge variant={source.status === 'healthy' ? 'default' : 'secondary'}>{source.status}</Badge>
            </div>
            <div className="mt-3 flex flex-wrap gap-2 text-xs text-muted-foreground">
              <span>{source.kind}</span>
              <span>{source.total_items} MCP</span>
              {source.trusted && <span>trusted</span>}
            </div>
          </button>
        ))}
      </div>

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索 MCP..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full border-transparent bg-muted/50 pl-9 focus:border-border focus:bg-background sm:w-80"
          />
        </div>
        <div className="flex flex-wrap gap-2">
          {categories.map((category) => (
            <Button
              key={category}
              variant={selectedCategory === category ? 'default' : 'outline'}
              size="sm"
              className={cn(selectedCategory !== category && 'border-transparent bg-muted/50 hover:bg-muted')}
              onClick={() => setSelectedCategory(category)}
            >
              {category}
            </Button>
          ))}
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {filteredMCPs.map((mcp) => {
          const isInstalled = installedIds.has(mcp.id)
          return (
            <Card
              key={mcp.id}
              className="group cursor-pointer transition-all hover:border-primary/20 hover:shadow-md"
              onClick={() => {
                setSelectedMCP(mcp)
                const firstInstallable = (mcp.install_options || []).findIndex((option) => option.type !== 'manual' && option.type !== 'unsupported')
                setInstallOptionIndex(firstInstallable >= 0 ? firstInstallable : 0)
              }}
            >
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div>
                    <CardTitle className="text-base transition-colors group-hover:text-primary">{mcp.title || mcp.name}</CardTitle>
                    <div className="mt-1.5 flex flex-wrap gap-1.5">
                      <Badge variant="secondary" className="font-normal">{mcp.category || '未分类'}</Badge>
                      <Badge variant="outline" className="font-normal">{mcp.installability || 'manual'}</Badge>
                    </div>
                  </div>
                  {isInstalled && (
                    <Badge className="gap-1 bg-emerald-100 text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-900/30 dark:text-emerald-400">
                      <Check className="h-3 w-3" />
                      已安装
                    </Badge>
                  )}
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <CardDescription className="min-h-[2.5rem] line-clamp-2">{mcp.description}</CardDescription>
                <div className="flex items-center justify-between text-sm">
                  <div className="flex items-center gap-4 text-muted-foreground">
                    <div className="flex items-center gap-1">
                      <Star className="h-4 w-4 fill-amber-400 text-amber-400" />
                      <span className="font-medium text-foreground">{mcp.rating}</span>
                    </div>
                    <div className="flex items-center gap-1">
                      <Download className="h-4 w-4" />
                      <span>{mcp.downloads?.toLocaleString()}</span>
                    </div>
                  </div>
                  <span className="text-muted-foreground">v{mcp.version}</span>
                </div>
                <div className="flex flex-wrap gap-1.5">
                  {(mcp.source_refs || []).slice(0, 3).map((source) => (
                    <Badge key={`${mcp.id}-${source.source_id}`} variant="outline" className="font-normal">{source.source_id}</Badge>
                  ))}
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      <Dialog open={!!selectedMCP} onOpenChange={() => setSelectedMCP(null)}>
        {selectedMCP && (
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle className="text-xl">{selectedMCP.title || selectedMCP.name}</DialogTitle>
              <DialogDescription>by {selectedMCP.author} · v{selectedMCP.version}</DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">{selectedMCP.description}</p>
              <div className="flex gap-6 text-sm">
                <div><p className="text-muted-foreground">分类</p><p className="font-medium">{selectedMCP.category}</p></div>
                <div><p className="text-muted-foreground">评分</p><p className="font-medium">{selectedMCP.rating}</p></div>
                <div><p className="text-muted-foreground">下载量</p><p className="font-medium">{selectedMCP.downloads?.toLocaleString()}</p></div>
              </div>
              {selectedInstallOptions.length > 0 && (
                <div className="space-y-2">
                  <p className="text-sm font-medium">安装方式</p>
                  <Select value={String(installOptionIndex)} onValueChange={(value) => setInstallOptionIndex(Number(value))}>
                    <SelectTrigger>
                      <SelectValue placeholder="选择安装方式" />
                    </SelectTrigger>
                    <SelectContent>
                      {selectedInstallOptions.map((option, index) => (
                        <SelectItem key={`${option.source_id}-${index}`} value={String(index)}>
                          {option.type} · {option.source_id} · {option.confidence}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
              )}
              <div>
                <p className="mb-2 text-sm font-medium">工具</p>
                <div className="space-y-2">
                  {selectedTools.map((tool) => (
                    <div key={tool.name} className="rounded-xl border bg-muted/30 p-3">
                      <p className="font-mono text-sm">{tool.name}</p>
                      <p className="mt-1 text-sm text-muted-foreground">{tool.description}</p>
                    </div>
                  ))}
                </div>
              </div>
              <p className="rounded-lg border bg-muted/30 px-3 py-2 text-sm text-muted-foreground">
                安装会保存到当前账号的已安装 MCP 列表；进入工作空间后可选择添加到具体工作空间。
              </p>
            </div>
            <DialogFooter className="mt-4">
              <Button variant="outline" onClick={() => setSelectedMCP(null)}>关闭</Button>
              {installedIds.has(selectedMCP.id) ? (
                <Button disabled className="gap-2"><Check className="h-4 w-4" />已安装</Button>
              ) : (
                <Button
                  className="gap-2"
                  onClick={() => handleInstall(selectedMCP)}
                  disabled={installingPackageId === selectedMCP.id || ['manual', 'unsupported'].includes(selectedInstallOptions[installOptionIndex]?.type || selectedMCP.installability || 'manual')}
                >
                  <Package className="h-4 w-4" />
                  {installingPackageId === selectedMCP.id ? '安装中...' : '安装到账号'}
                </Button>
              )}
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  )
}
