'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { marketMCPs, installedMCPs } from '@/lib/mock-data'
import type { MCPServer } from '@/lib/types'
import { Search, Star, Download, Check, Package, ArrowRight } from 'lucide-react'
import { cn } from '@/lib/utils'

const categories = ['全部', '系统', '数据', '网络', '开发', '通讯', '效率']

export function MarketPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [selectedCategory, setSelectedCategory] = useState('全部')
  const [selectedMCP, setSelectedMCP] = useState<MCPServer | null>(null)

  const installedIds = installedMCPs.map((m) => m.id)

  const filteredMCPs = marketMCPs.filter((mcp) => {
    const matchesSearch =
      mcp.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      mcp.description.toLowerCase().includes(searchQuery.toLowerCase())
    const matchesCategory =
      selectedCategory === '全部' || mcp.category === selectedCategory
    return matchesSearch && matchesCategory
  })

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div>
        <h2 className="text-2xl font-semibold tracking-tight">MCP 市场</h2>
        <p className="text-muted-foreground">浏览和安装 MCP 服务扩展您的能力</p>
      </div>

      {/* Search and Filter */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索 MCP..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="w-full bg-muted/50 border-transparent pl-9 focus:border-border focus:bg-background sm:w-80"
          />
        </div>
        <div className="flex flex-wrap gap-2">
          {categories.map((category) => (
            <Button
              key={category}
              variant={selectedCategory === category ? 'default' : 'outline'}
              size="sm"
              className={cn(
                selectedCategory !== category && 'border-transparent bg-muted/50 hover:bg-muted'
              )}
              onClick={() => setSelectedCategory(category)}
            >
              {category}
            </Button>
          ))}
        </div>
      </div>

      {/* MCP Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {filteredMCPs.map((mcp) => {
          const isInstalled = installedIds.includes(mcp.id)
          return (
            <Card
              key={mcp.id}
              className="group cursor-pointer transition-all hover:shadow-md hover:border-primary/20"
              onClick={() => setSelectedMCP(mcp)}
            >
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-muted text-2xl">
                      {mcp.icon}
                    </div>
                    <div>
                      <CardTitle className="text-base group-hover:text-primary transition-colors">
                        {mcp.name}
                      </CardTitle>
                      <Badge variant="secondary" className="mt-1.5 font-normal">
                        {mcp.category}
                      </Badge>
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
                <CardDescription className="line-clamp-2 min-h-[2.5rem]">
                  {mcp.description}
                </CardDescription>
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
                <div className="flex items-center justify-end pt-2 border-t border-border">
                  <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 transition-all group-hover:opacity-100 group-hover:translate-x-1" />
                </div>
              </CardContent>
            </Card>
          )
        })}
      </div>

      {filteredMCPs.length === 0 && (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-muted">
              <Package className="h-7 w-7 text-muted-foreground" />
            </div>
            <h3 className="mt-4 text-lg font-semibold">未找到 MCP</h3>
            <p className="mt-2 max-w-sm text-muted-foreground">
              没有匹配的 MCP，请尝试其他搜索词或分类
            </p>
          </CardContent>
        </Card>
      )}

      {/* MCP Detail Dialog */}
      <Dialog open={!!selectedMCP} onOpenChange={() => setSelectedMCP(null)}>
        {selectedMCP && (
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <div className="flex items-center gap-4">
                <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-muted text-3xl">
                  {selectedMCP.icon}
                </div>
                <div>
                  <DialogTitle className="text-xl">{selectedMCP.name}</DialogTitle>
                  <DialogDescription className="mt-1">
                    by {selectedMCP.author} · v{selectedMCP.version}
                  </DialogDescription>
                </div>
              </div>
            </DialogHeader>

            <Tabs defaultValue="overview" className="mt-2">
              <TabsList className="inline-flex h-auto gap-1 rounded-lg bg-muted p-1">
                <TabsTrigger value="overview" className="rounded-md px-3 py-1.5 text-sm">概览</TabsTrigger>
                <TabsTrigger value="tools" className="rounded-md px-3 py-1.5 text-sm">工具</TabsTrigger>
              </TabsList>

              <TabsContent value="overview" className="mt-4 space-y-4">
                <div>
                  <h4 className="font-medium mb-2">描述</h4>
                  <p className="text-muted-foreground">{selectedMCP.description}</p>
                </div>
                <div className="flex gap-6">
                  <div>
                    <p className="text-sm text-muted-foreground">分类</p>
                    <Badge variant="secondary" className="mt-1.5">
                      {selectedMCP.category}
                    </Badge>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">评分</p>
                    <div className="flex items-center gap-1 mt-1.5">
                      <Star className="h-4 w-4 fill-amber-400 text-amber-400" />
                      <span className="font-medium">{selectedMCP.rating}</span>
                    </div>
                  </div>
                  <div>
                    <p className="text-sm text-muted-foreground">下载量</p>
                    <p className="font-medium mt-1.5">
                      {selectedMCP.downloads?.toLocaleString()}
                    </p>
                  </div>
                </div>
              </TabsContent>

              <TabsContent value="tools" className="mt-4">
                <div className="space-y-3">
                  {selectedMCP.tools.map((tool) => (
                    <div
                      key={tool.name}
                      className="rounded-xl border bg-muted/30 p-4"
                    >
                      <h4 className="font-mono font-medium">{tool.name}</h4>
                      <p className="text-sm text-muted-foreground mt-1">
                        {tool.description}
                      </p>
                      {tool.parameters.length > 0 && (
                        <div className="mt-3">
                          <p className="text-sm font-medium mb-2">参数</p>
                          <div className="space-y-2">
                            {tool.parameters.map((param) => (
                              <div
                                key={param.name}
                                className="flex items-start gap-2 text-sm"
                              >
                                <code className="rounded bg-muted px-1.5 py-0.5 font-mono text-xs">
                                  {param.name}
                                </code>
                                <Badge variant="outline" className="text-xs font-normal">
                                  {param.type}
                                </Badge>
                                {param.required && (
                                  <Badge variant="destructive" className="text-xs font-normal">
                                    必填
                                  </Badge>
                                )}
                                <span className="text-muted-foreground">
                                  {param.description}
                                </span>
                              </div>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </TabsContent>
            </Tabs>

            <DialogFooter className="mt-4">
              <Button variant="outline" onClick={() => setSelectedMCP(null)}>
                关闭
              </Button>
              {installedIds.includes(selectedMCP.id) ? (
                <Button disabled className="gap-2">
                  <Check className="h-4 w-4" />
                  已安装
                </Button>
              ) : (
                <Button className="gap-2">
                  <Download className="h-4 w-4" />
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
