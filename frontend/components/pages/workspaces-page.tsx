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
  DialogTrigger,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { workspaces } from '@/lib/mock-data'
import { Plus, Search, Layers, MoreHorizontal, Settings, Trash2, Copy, Package, Users, ArrowRight, Calendar } from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import Link from 'next/link'
import { cn } from '@/lib/utils'

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function WorkspacesPage() {
  const [searchQuery, setSearchQuery] = useState('')
  const [isCreateOpen, setIsCreateOpen] = useState(false)

  const filteredWorkspaces = workspaces.filter(
    (ws) =>
      ws.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      ws.description.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">工作空间</h2>
          <p className="text-muted-foreground">管理和配置您的 MCP 工作环境</p>
        </div>
        <Dialog open={isCreateOpen} onOpenChange={setIsCreateOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              创建工作空间
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>创建工作空间</DialogTitle>
              <DialogDescription>
                创建一个新的工作空间来组织和管理您的 MCP 服务
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="name">名称</Label>
                <Input id="name" placeholder="输入工作空间名称" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">描述</Label>
                <Textarea
                  id="description"
                  placeholder="输入工作空间描述（可选）"
                  rows={3}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                取消
              </Button>
              <Button onClick={() => setIsCreateOpen(false)}>创建</Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索工作空间..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-full bg-muted/50 border-transparent pl-9 focus:border-border focus:bg-background sm:max-w-sm"
        />
      </div>

      {/* Workspaces Grid */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {filteredWorkspaces.map((workspace) => (
          <Card
            key={workspace.id}
            className="group relative overflow-hidden transition-all hover:shadow-md hover:border-primary/20"
          >
            <Link href={`/workspaces/${workspace.id}`} className="absolute inset-0 z-10" />
            <CardHeader className="pb-3">
              <div className="flex items-start justify-between">
                <div className="flex items-center gap-3">
                  <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-primary/10 to-primary/5 ring-1 ring-primary/10">
                    <Layers className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <CardTitle className="text-base group-hover:text-primary transition-colors">
                      {workspace.name}
                    </CardTitle>
                    <Badge
                      variant={workspace.status === 'active' ? 'default' : 'secondary'}
                      className={cn(
                        'mt-1.5',
                        workspace.status === 'active' 
                          ? 'bg-emerald-100 text-emerald-700 hover:bg-emerald-100 dark:bg-emerald-900/30 dark:text-emerald-400' 
                          : ''
                      )}
                    >
                      {workspace.status === 'active' ? '运行中' : '已停止'}
                    </Badge>
                  </div>
                </div>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button 
                      variant="ghost" 
                      size="icon" 
                      className="relative z-20 h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
                    >
                      <MoreHorizontal className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem asChild>
                      <Link href={`/workspaces/${workspace.id}`}>
                        <Settings className="mr-2 h-4 w-4" />
                        管理
                      </Link>
                    </DropdownMenuItem>
                    <DropdownMenuItem>
                      <Copy className="mr-2 h-4 w-4" />
                      复制
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem className="text-destructive focus:text-destructive">
                      <Trash2 className="mr-2 h-4 w-4" />
                      删除
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <CardDescription className="line-clamp-2 min-h-[2.5rem]">
                {workspace.description}
              </CardDescription>
              
              <div className="flex items-center gap-4 text-sm text-muted-foreground">
                <span className="flex items-center gap-1.5">
                  <Package className="h-4 w-4" />
                  {workspace.mcpCount} MCPs
                </span>
                <span className="flex items-center gap-1.5">
                  <Users className="h-4 w-4" />
                  {workspace.sessionCount} 会话
                </span>
              </div>
              
              <div className="flex items-center justify-between border-t border-border pt-3">
                <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                  <Calendar className="h-3.5 w-3.5" />
                  更新于 {formatDate(workspace.updatedAt)}
                </p>
                <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 transition-all group-hover:opacity-100 group-hover:translate-x-1" />
              </div>
            </CardContent>
          </Card>
        ))}
      </div>

      {filteredWorkspaces.length === 0 && (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-muted">
              <Layers className="h-7 w-7 text-muted-foreground" />
            </div>
            <h3 className="mt-4 text-lg font-semibold">未找到工作空间</h3>
            <p className="mt-2 max-w-sm text-muted-foreground">
              {searchQuery
                ? '没有匹配的工作空间，请尝试其他搜索词'
                : '创建您的第一个工作空间来开始管理 MCP 服务'}
            </p>
            {!searchQuery && (
              <Button className="mt-6" onClick={() => setIsCreateOpen(true)}>
                <Plus className="mr-2 h-4 w-4" />
                创建工作空间
              </Button>
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}
