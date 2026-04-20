'use client'

import { useMemo, useState } from 'react'
import { Check, Download, Package, Search, Star } from 'lucide-react'

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
import { gatewayApi, invalidate, useGatewaySWR, type ListData, type MarketPackage, type Workspace } from '@/lib/gateway-api'

const categories = ['全部', '系统', '数据', '网络', '开发', '通讯', '效率']

export function MarketPage() {
  const { data: marketData } = useGatewaySWR<ListData<MarketPackage>>('/api/v1/market/packages')
  const { data: installedData } = useGatewaySWR<{ items: Array<{ package_id: string }> }>('/api/v1/installed')
  const { data: workspacesData } = useGatewaySWR<ListData<Workspace>>('/api/v1/workspaces')
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('全部')
  const [selectedMCP, setSelectedMCP] = useState<MarketPackage | null>(null)
  const [installWorkspace, setInstallWorkspace] = useState('')

  const installedIds = new Set((installedData?.items || []).map((m) => m.package_id))
  const workspaces = workspacesData?.items || []

  const filteredMCPs = useMemo(() => {
    const items = marketData?.items || []
    return items.filter((mcp) => {
      const matchesSearch = `${mcp.name} ${mcp.description}`.toLowerCase().includes(searchQuery.toLowerCase())
      const matchesCategory = selectedCategory === '全部' || mcp.category === selectedCategory
      return matchesSearch && matchesCategory
    })
  }, [marketData?.items, searchQuery, selectedCategory])

  async function handleInstall(pkg: MarketPackage) {
    const targetWorkspace = installWorkspace || workspaces[0]?.id
    if (!targetWorkspace) return
    await gatewayApi.createService(targetWorkspace, {
      name: pkg.id,
      market_package_id: pkg.id,
      version: pkg.version,
    })
    await Promise.all([invalidate('/api/v1/market/packages'), invalidate('/api/v1/installed'), invalidate(`/api/v1/workspaces/${targetWorkspace}/services`)])
    setSelectedMCP(null)
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">MCP 市场</h2>
        <p className="text-muted-foreground">浏览和安装 MCP 服务扩展您的能力</p>
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
                setInstallWorkspace(workspaces[0]?.id || '')
              }}
            >
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div>
                    <CardTitle className="text-base transition-colors group-hover:text-primary">{mcp.name}</CardTitle>
                    <Badge variant="secondary" className="mt-1.5 font-normal">{mcp.category || '未分类'}</Badge>
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
              </CardContent>
            </Card>
          )
        })}
      </div>

      <Dialog open={!!selectedMCP} onOpenChange={() => setSelectedMCP(null)}>
        {selectedMCP && (
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle className="text-xl">{selectedMCP.name}</DialogTitle>
              <DialogDescription>by {selectedMCP.author} · v{selectedMCP.version}</DialogDescription>
            </DialogHeader>
            <div className="space-y-4">
              <p className="text-sm text-muted-foreground">{selectedMCP.description}</p>
              <div className="flex gap-6 text-sm">
                <div><p className="text-muted-foreground">分类</p><p className="font-medium">{selectedMCP.category}</p></div>
                <div><p className="text-muted-foreground">评分</p><p className="font-medium">{selectedMCP.rating}</p></div>
                <div><p className="text-muted-foreground">下载量</p><p className="font-medium">{selectedMCP.downloads?.toLocaleString()}</p></div>
              </div>
              <div>
                <p className="mb-2 text-sm font-medium">工具</p>
                <div className="space-y-2">
                  {selectedMCP.tools.map((tool) => (
                    <div key={tool.name} className="rounded-xl border bg-muted/30 p-3">
                      <p className="font-mono text-sm">{tool.name}</p>
                      <p className="mt-1 text-sm text-muted-foreground">{tool.description}</p>
                    </div>
                  ))}
                </div>
              </div>
              <div className="space-y-2">
                <p className="text-sm font-medium">安装到工作空间</p>
                <Select value={installWorkspace} onValueChange={setInstallWorkspace}>
                  <SelectTrigger>
                    <SelectValue placeholder="选择工作空间" />
                  </SelectTrigger>
                  <SelectContent>
                    {workspaces.map((ws) => (
                      <SelectItem key={ws.id} value={ws.id}>{ws.name}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <DialogFooter className="mt-4">
              <Button variant="outline" onClick={() => setSelectedMCP(null)}>关闭</Button>
              {installedIds.has(selectedMCP.id) ? (
                <Button disabled className="gap-2"><Check className="h-4 w-4" />已安装</Button>
              ) : (
                <Button className="gap-2" onClick={() => handleInstall(selectedMCP)} disabled={!workspaces.length}>
                  <Package className="h-4 w-4" />
                  安装
                </Button>
              )}
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  )
}
