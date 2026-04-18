'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
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
import { installedMCPs } from '@/lib/mock-data'
import type { MCPServer } from '@/lib/types'
import {
  Search,
  Play,
  Square,
  Settings,
  Trash2,
  MoreHorizontal,
  Package,
  RefreshCw,
  Wrench,
  Calendar,
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'
import Link from 'next/link'

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function InstalledPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedMCP, setSelectedMCP] = useState<MCPServer | null>(null)
  const [isConfigOpen, setIsConfigOpen] = useState(false)

  const filteredMCPs = installedMCPs.filter(
    (mcp) =>
      mcp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      mcp.description.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const handleConfigure = (mcp: MCPServer) => {
    setSelectedMCP(mcp)
    setIsConfigOpen(true)
  }

  const runningCount = installedMCPs.filter((m) => m.status === 'running').length
  const stoppedCount = installedMCPs.filter((m) => m.status === 'stopped').length

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">已安装 MCP</h2>
          <p className="text-muted-foreground">管理和配置已安装的 MCP 服务</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" asChild>
            <Link href="/market">
              <Package className="mr-2 h-4 w-4" />
              浏览市场
            </Link>
          </Button>
          <Button variant="outline">
            <RefreshCw className="mr-2 h-4 w-4" />
            检查更新
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-violet-100 dark:bg-violet-900/30">
                <Package className="h-6 w-6 text-violet-600 dark:text-violet-400" />
              </div>
              <div>
                <p className="text-3xl font-bold">{installedMCPs.length}</p>
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
                <p className="text-3xl font-bold text-emerald-600 dark:text-emerald-400">{runningCount}</p>
                <p className="text-sm text-muted-foreground">运行中</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-slate-100 dark:bg-slate-800">
                <Square className="h-6 w-6 text-slate-600 dark:text-slate-400" />
              </div>
              <div>
                <p className="text-3xl font-bold text-slate-600 dark:text-slate-400">{stoppedCount}</p>
                <p className="text-sm text-muted-foreground">已停止</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索已安装 MCP..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-full bg-muted/50 border-transparent pl-9 focus:border-border focus:bg-background sm:max-w-sm"
        />
      </div>

      {/* MCP List */}
      <div className="space-y-3">
        {filteredMCPs.map((mcp) => (
          <Card key={mcp.id} className="group transition-all hover:shadow-sm hover:border-primary/20">
            <CardContent className="p-5">
              <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-4 min-w-0">
                  <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-muted text-2xl">
                    {mcp.icon}
                  </div>
                  <div className="min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      <h3 className="font-semibold">{mcp.name}</h3>
                      <Badge
                        className={cn(
                          'border-0',
                          mcp.status === 'running'
                            ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                            : mcp.status === 'stopped'
                            ? 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                            : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                        )}
                      >
                        {mcp.status === 'running'
                          ? '运行中'
                          : mcp.status === 'stopped'
                          ? '已停止'
                          : '错误'}
                      </Badge>
                    </div>
                    <p className="text-sm text-muted-foreground mt-1 line-clamp-1">
                      {mcp.description}
                    </p>
                    <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                      <span>v{mcp.version}</span>
                      <span className="flex items-center gap-1">
                        <Calendar className="h-3 w-3" />
                        {formatDate(mcp.installedAt!)}
                      </span>
                      <span className="flex items-center gap-1">
                        <Wrench className="h-3 w-3" />
                        {mcp.tools.length} 个工具
                      </span>
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  {mcp.status === 'running' ? (
                    <Button variant="outline" size="sm" className="hidden sm:flex">
                      <Square className="mr-2 h-4 w-4" />
                      停止
                    </Button>
                  ) : (
                    <Button variant="outline" size="sm" className="hidden sm:flex">
                      <Play className="mr-2 h-4 w-4" />
                      启动
                    </Button>
                  )}
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button variant="ghost" size="icon">
                        <MoreHorizontal className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      {mcp.status === 'running' ? (
                        <DropdownMenuItem className="sm:hidden">
                          <Square className="mr-2 h-4 w-4" />
                          停止
                        </DropdownMenuItem>
                      ) : (
                        <DropdownMenuItem className="sm:hidden">
                          <Play className="mr-2 h-4 w-4" />
                          启动
                        </DropdownMenuItem>
                      )}
                      <DropdownMenuItem onClick={() => handleConfigure(mcp)}>
                        <Settings className="mr-2 h-4 w-4" />
                        配置
                      </DropdownMenuItem>
                      <DropdownMenuItem>
                        <RefreshCw className="mr-2 h-4 w-4" />
                        更新
                      </DropdownMenuItem>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem className="text-destructive focus:text-destructive">
                        <Trash2 className="mr-2 h-4 w-4" />
                        卸载
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {filteredMCPs.length === 0 && (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-muted">
              <Package className="h-7 w-7 text-muted-foreground" />
            </div>
            <h3 className="mt-4 text-lg font-semibold">未安装 MCP</h3>
            <p className="mt-2 max-w-sm text-muted-foreground">
              {searchQuery
                ? '没有匹配的 MCP，请尝试其他搜索词'
                : '前往市场安装您的第一个 MCP'}
            </p>
            {!searchQuery && (
              <Button className="mt-6" asChild>
                <Link href="/market">
                  <Package className="mr-2 h-4 w-4" />
                  浏览市场
                </Link>
              </Button>
            )}
          </CardContent>
        </Card>
      )}

      {/* Configure Dialog */}
      <Dialog open={isConfigOpen} onOpenChange={setIsConfigOpen}>
        {selectedMCP && (
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <div className="flex items-center gap-3">
                <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-muted text-2xl">
                  {selectedMCP.icon}
                </div>
                <div>
                  <DialogTitle>{selectedMCP.name} 配置</DialogTitle>
                  <DialogDescription>
                    配置 MCP 的环境变量和工具权限
                  </DialogDescription>
                </div>
              </div>
            </DialogHeader>

            <Tabs defaultValue="env" className="mt-2">
              <TabsList className="inline-flex h-auto gap-1 rounded-lg bg-muted p-1">
                <TabsTrigger value="env" className="rounded-md px-3 py-1.5 text-sm">环境变量</TabsTrigger>
                <TabsTrigger value="tools" className="rounded-md px-3 py-1.5 text-sm">工具权限</TabsTrigger>
              </TabsList>

              <TabsContent value="env" className="mt-4 space-y-4">
                <div className="space-y-4">
                  <div className="space-y-2">
                    <Label>API_KEY</Label>
                    <Input type="password" placeholder="输入 API 密钥" />
                  </div>
                  <div className="space-y-2">
                    <Label>BASE_URL</Label>
                    <Input placeholder="https://api.example.com" />
                  </div>
                </div>
              </TabsContent>

              <TabsContent value="tools" className="mt-4">
                <div className="space-y-3">
                  {selectedMCP.tools.map((tool) => (
                    <div
                      key={tool.name}
                      className="flex items-center justify-between rounded-xl border bg-muted/30 p-4"
                    >
                      <div>
                        <p className="font-mono font-medium">{tool.name}</p>
                        <p className="text-sm text-muted-foreground">
                          {tool.description}
                        </p>
                      </div>
                      <Switch defaultChecked />
                    </div>
                  ))}
                </div>
              </TabsContent>
            </Tabs>

            <DialogFooter className="mt-4">
              <Button variant="outline" onClick={() => setIsConfigOpen(false)}>
                取消
              </Button>
              <Button onClick={() => setIsConfigOpen(false)}>保存配置</Button>
            </DialogFooter>
          </DialogContent>
        )}
      </Dialog>
    </div>
  )
}
