'use client'

import { useMemo, useState } from 'react'
import Link from 'next/link'
import { ArrowRight, Calendar, Copy, Layers, MoreHorizontal, Package, Plus, Search, Settings, Trash2, Users } from 'lucide-react'

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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'
import { gatewayApi, invalidate, useGatewaySWR, GatewayApiError, type ListData, type Workspace } from '@/lib/gateway-api'
import { useToast } from '@/hooks/use-toast'

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

export function WorkspacesPage() {
  const { toast } = useToast()
  const { data, isLoading } = useGatewaySWR<ListData<Workspace>>('/api/v1/workspaces')
  const [searchQuery, setSearchQuery] = useState('')
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [form, setForm] = useState({ name: '', description: '' })
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [workspaceToDelete, setWorkspaceToDelete] = useState<string | null>(null)

  const workspaces = useMemo(() => {
    const items = data?.items || []
    if (!searchQuery) return items
    const q = searchQuery.toLowerCase()
    return items.filter((ws) => `${ws.name} ${ws.description} ${ws.id}`.toLowerCase().includes(q))
  }, [data?.items, searchQuery])

  async function handleCreate() {
    setSubmitting(true)
    try {
      await gatewayApi.createWorkspace(form)
      setForm({ name: '', description: '' })
      setIsCreateOpen(false)
      await invalidate('/api/v1/workspaces')
      await invalidate('/api/v1/stats/overview')
      toast({
        title: '创建成功',
        description: '工作区已成功创建',
      })
    } catch (error) {
      if (error instanceof GatewayApiError) {
        toast({
          variant: 'destructive',
          title: '创建失败',
          description: error.message,
        })
      } else {
        toast({
          variant: 'destructive',
          title: '创建失败',
          description: '创建工作区失败，请重试',
        })
      }
    } finally {
      setSubmitting(false)
    }
  }

  function handleDeleteClick(id: string, e?: React.MouseEvent) {
    e?.stopPropagation()
    setWorkspaceToDelete(id)
    setDeleteDialogOpen(true)
  }

  async function handleDeleteConfirm() {
    if (!workspaceToDelete) return
    try {
      await gatewayApi.deleteWorkspace(workspaceToDelete, true)
      await invalidate('/api/v1/workspaces')
      await invalidate('/api/v1/stats/overview')
      toast({
        title: '删除成功',
        description: '工作区已成功删除',
      })
    } catch (error) {
      if (error instanceof GatewayApiError) {
        toast({
          variant: 'destructive',
          title: '删除失败',
          description: error.message,
        })
      } else {
        toast({
          variant: 'destructive',
          title: '删除失败',
          description: '删除工作区失败，请重试',
        })
      }
    } finally {
      setDeleteDialogOpen(false)
      setWorkspaceToDelete(null)
    }
  }

  return (
    <div className="space-y-6">
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
              <DialogDescription>创建一个新的工作空间来组织和管理您的 MCP 服务</DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="name">名称</Label>
                <Input id="name" value={form.name} onChange={(e) => setForm((v) => ({ ...v, name: e.target.value }))} placeholder="输入工作空间名称" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="description">描述</Label>
                <Textarea
                  id="description"
                  value={form.description}
                  onChange={(e) => setForm((v) => ({ ...v, description: e.target.value }))}
                  placeholder="输入工作空间描述（可选）"
                  rows={3}
                />
              </div>
            </div>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                取消
              </Button>
              <Button disabled={!form.name || submitting} onClick={handleCreate}>
                {submitting ? '创建中...' : '创建'}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-blue-100 dark:bg-blue-900/20">
                <Layers className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <p className="text-3xl font-bold">{data?.items.length ?? 0}</p>
                <p className="text-sm text-muted-foreground">工作空间</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-100 dark:bg-emerald-900/20">
                <Package className="h-6 w-6 text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <p className="text-3xl font-bold text-emerald-600 dark:text-emerald-400">
                  {data?.items.reduce((sum, ws) => sum + ws.mcp_count, 0) ?? 0}
                </p>
                <p className="text-sm text-muted-foreground">MCP 总数</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-violet-100 dark:bg-violet-900/20">
                <Users className="h-6 w-6 text-violet-600 dark:text-violet-400" />
              </div>
              <div>
                <p className="text-3xl font-bold text-violet-600 dark:text-violet-400">
                  {data?.items.reduce((sum, ws) => sum + ws.session_count, 0) ?? 0}
                </p>
                <p className="text-sm text-muted-foreground">会话总数</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="搜索工作空间..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-full border-transparent bg-muted/50 pl-9 focus:border-border focus:bg-background sm:max-w-sm"
        />
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {workspaces.map((workspace) => (
          <Card key={workspace.id} className="group relative overflow-hidden transition-all hover:border-primary/20 hover:shadow-md">
            <CardHeader className="pb-3">
              <div className="flex items-start justify-between">
                <Link href={`/workspaces/${workspace.id}`} className="flex items-center gap-3 flex-1">
                  <div className="flex h-11 w-11 items-center justify-center rounded-xl bg-gradient-to-br from-primary/10 to-primary/5 ring-1 ring-primary/10">
                    <Layers className="h-5 w-5 text-primary" />
                  </div>
                  <div>
                    <CardTitle className="text-base transition-colors group-hover:text-primary">{workspace.name}</CardTitle>
                    <Badge
                      variant="outline"
                      className={cn(
                        'mt-1.5 border-0',
                        workspace.status === 'running'
                          ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                          : 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                      )}
                    >
                      {workspace.status === 'running' ? '运行中' : '已停止'}
                    </Badge>
                  </div>
                </Link>
                <Button
                  variant="ghost"
                  size="icon"
                  onClick={(e) => handleDeleteClick(workspace.id, e)}
                  className="h-8 w-8 text-destructive hover:text-destructive hover:bg-destructive/10"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              <Link href={`/workspaces/${workspace.id}`}>
                <CardDescription className="min-h-[2.5rem] line-clamp-2">{workspace.description || '暂无描述'}</CardDescription>
                <div className="flex items-center gap-4 text-sm text-muted-foreground">
                  <span className="flex items-center gap-1.5">
                    <Package className="h-4 w-4" />
                    {workspace.mcp_count} MCPs
                  </span>
                  <span className="flex items-center gap-1.5">
                    <Users className="h-4 w-4" />
                    {workspace.session_count} 会话
                  </span>
                </div>
                <div className="flex items-center justify-between border-t border-border pt-3">
                  <p className="flex items-center gap-1.5 text-xs text-muted-foreground">
                    <Calendar className="h-3.5 w-3.5" />
                    更新于 {formatDate(workspace.last_activity_at)}
                  </p>
                  <ArrowRight className="h-4 w-4 text-muted-foreground opacity-0 transition-all group-hover:translate-x-1 group-hover:opacity-100" />
                </div>
              </Link>
            </CardContent>
          </Card>
        ))}
      </div>

      {!isLoading && workspaces.length === 0 && (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center justify-center py-16 text-center">
            <div className="flex h-14 w-14 items-center justify-center rounded-full bg-muted">
              <Layers className="h-7 w-7 text-muted-foreground" />
            </div>
            <h3 className="mt-4 text-lg font-semibold">未找到工作空间</h3>
            <p className="mt-2 max-w-sm text-muted-foreground">
              {searchQuery ? '没有匹配的工作空间，请尝试其他搜索词' : '创建您的第一个工作空间来开始管理 MCP 服务'}
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

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>确认删除工作空间</DialogTitle>
            <DialogDescription>
              此操作将永久删除该工作空间及其所有数据，删除后无法恢复。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteDialogOpen(false)}>
              取消
            </Button>
            <Button variant="destructive" onClick={handleDeleteConfirm}>
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
