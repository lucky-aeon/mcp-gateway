'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { apiKeys } from '@/lib/mock-data'
import type { APIKey } from '@/lib/types'
import {
  Plus,
  Key,
  Copy,
  Trash2,
  MoreHorizontal,
  Eye,
  EyeOff,
  Shield,
  Clock,
  Check,
} from 'lucide-react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { cn } from '@/lib/utils'

function formatDate(dateString: string): string {
  const date = new Date(dateString)
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

function maskKey(key: string): string {
  if (key.length <= 12) return key
  return key.substring(0, 12) + '••••••••••••'
}

export function APIKeysPage() {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [newKeyResult, setNewKeyResult] = useState<string | null>(null)
  const [visibleKeys, setVisibleKeys] = useState<Set<string>>(new Set())
  const [keyToRevoke, setKeyToRevoke] = useState<APIKey | null>(null)
  const [copiedKey, setCopiedKey] = useState<string | null>(null)

  const toggleKeyVisibility = (keyId: string) => {
    const newVisible = new Set(visibleKeys)
    if (newVisible.has(keyId)) {
      newVisible.delete(keyId)
    } else {
      newVisible.add(keyId)
    }
    setVisibleKeys(newVisible)
  }

  const handleCreateKey = () => {
    const newKey = `gw_${Math.random().toString(36).substring(2, 15)}_${Math.random().toString(36).substring(2, 15)}`
    setNewKeyResult(newKey)
  }

  const copyToClipboard = (text: string, keyId?: string) => {
    navigator.clipboard.writeText(text)
    if (keyId) {
      setCopiedKey(keyId)
      setTimeout(() => setCopiedKey(null), 2000)
    }
  }

  const activeCount = apiKeys.filter((k) => k.status === 'active').length
  const revokedCount = apiKeys.filter((k) => k.status === 'revoked').length

  return (
    <div className="space-y-6">
      {/* Page Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">API 密钥</h2>
          <p className="text-muted-foreground">管理用于访问 Gateway API 的密钥</p>
        </div>
        <Dialog
          open={isCreateOpen}
          onOpenChange={(open) => {
            setIsCreateOpen(open)
            if (!open) setNewKeyResult(null)
          }}
        >
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              创建密钥
            </Button>
          </DialogTrigger>
          <DialogContent className="sm:max-w-md">
            {newKeyResult ? (
              <>
                <DialogHeader>
                  <DialogTitle>密钥已创建</DialogTitle>
                  <DialogDescription>
                    请立即复制此密钥，关闭后将无法再次查看完整密钥
                  </DialogDescription>
                </DialogHeader>
                <div className="py-4">
                  <div className="flex items-center gap-2 rounded-xl bg-muted p-4">
                    <code className="flex-1 break-all text-sm font-mono">{newKeyResult}</code>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => copyToClipboard(newKeyResult)}
                    >
                      <Copy className="h-4 w-4" />
                    </Button>
                  </div>
                  <p className="mt-3 text-sm text-muted-foreground">
                    请将此密钥保存到安全的位置，它只会显示一次。
                  </p>
                </div>
                <DialogFooter>
                  <Button onClick={() => setIsCreateOpen(false)}>完成</Button>
                </DialogFooter>
              </>
            ) : (
              <>
                <DialogHeader>
                  <DialogTitle>创建 API 密钥</DialogTitle>
                  <DialogDescription>
                    创建一个新的 API 密钥用于访问 Gateway
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-4">
                  <div className="space-y-2">
                    <Label htmlFor="key-name">密钥名称</Label>
                    <Input id="key-name" placeholder="例如：生产环境密钥" />
                  </div>
                  <div className="space-y-3">
                    <Label>权限</Label>
                    <div className="space-y-3">
                      <div className="flex items-start space-x-3">
                        <Checkbox id="perm-read" defaultChecked className="mt-0.5" />
                        <div>
                          <Label htmlFor="perm-read" className="font-medium">读取</Label>
                          <p className="text-xs text-muted-foreground">查看工作空间和会话信息</p>
                        </div>
                      </div>
                      <div className="flex items-start space-x-3">
                        <Checkbox id="perm-write" defaultChecked className="mt-0.5" />
                        <div>
                          <Label htmlFor="perm-write" className="font-medium">写入</Label>
                          <p className="text-xs text-muted-foreground">创建和修改工作空间</p>
                        </div>
                      </div>
                      <div className="flex items-start space-x-3">
                        <Checkbox id="perm-admin" className="mt-0.5" />
                        <div>
                          <Label htmlFor="perm-admin" className="font-medium">管理</Label>
                          <p className="text-xs text-muted-foreground">管理 MCP 和系统设置</p>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="key-expiry">过期时间（可选）</Label>
                    <Input id="key-expiry" type="date" />
                  </div>
                </div>
                <DialogFooter>
                  <Button variant="outline" onClick={() => setIsCreateOpen(false)}>
                    取消
                  </Button>
                  <Button onClick={handleCreateKey}>创建密钥</Button>
                </DialogFooter>
              </>
            )}
          </DialogContent>
        </Dialog>
      </div>

      {/* Stats */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-violet-100 dark:bg-violet-900/30">
                <Key className="h-6 w-6 text-violet-600 dark:text-violet-400" />
              </div>
              <div>
                <p className="text-3xl font-bold">{apiKeys.length}</p>
                <p className="text-sm text-muted-foreground">总密钥数</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-emerald-100 dark:bg-emerald-900/30">
                <Shield className="h-6 w-6 text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <p className="text-3xl font-bold text-emerald-600 dark:text-emerald-400">{activeCount}</p>
                <p className="text-sm text-muted-foreground">活跃密钥</p>
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="p-6">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-slate-100 dark:bg-slate-800">
                <Clock className="h-6 w-6 text-slate-600 dark:text-slate-400" />
              </div>
              <div>
                <p className="text-3xl font-bold text-slate-600 dark:text-slate-400">{revokedCount}</p>
                <p className="text-sm text-muted-foreground">已撤销</p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Keys Table */}
      <Card>
        <CardHeader>
          <CardTitle>API 密钥列表</CardTitle>
          <CardDescription>管理您的所有 API 密钥</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>名称</TableHead>
                <TableHead>密钥</TableHead>
                <TableHead>权限</TableHead>
                <TableHead>状态</TableHead>
                <TableHead>创建时间</TableHead>
                <TableHead>最后使用</TableHead>
                <TableHead className="w-[60px]"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {apiKeys.map((apiKey) => (
                <TableRow key={apiKey.id}>
                  <TableCell className="font-medium">{apiKey.name}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <code className="text-sm font-mono text-muted-foreground">
                        {visibleKeys.has(apiKey.id)
                          ? apiKey.key
                          : maskKey(apiKey.key)}
                      </code>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7"
                        onClick={() => toggleKeyVisibility(apiKey.id)}
                      >
                        {visibleKeys.has(apiKey.id) ? (
                          <EyeOff className="h-3.5 w-3.5" />
                        ) : (
                          <Eye className="h-3.5 w-3.5" />
                        )}
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7"
                        onClick={() => copyToClipboard(apiKey.key, apiKey.id)}
                      >
                        {copiedKey === apiKey.id ? (
                          <Check className="h-3.5 w-3.5 text-emerald-500" />
                        ) : (
                          <Copy className="h-3.5 w-3.5" />
                        )}
                      </Button>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {apiKey.permissions.map((perm) => (
                        <Badge key={perm} variant="secondary" className="text-xs font-normal">
                          {perm}
                        </Badge>
                      ))}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge
                      className={cn(
                        'border-0',
                        apiKey.status === 'active'
                          ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                          : apiKey.status === 'revoked'
                          ? 'bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-400'
                          : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                      )}
                    >
                      {apiKey.status === 'active'
                        ? '活跃'
                        : apiKey.status === 'revoked'
                        ? '已撤销'
                        : '已过期'}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">{formatDate(apiKey.createdAt)}</TableCell>
                  <TableCell className="text-muted-foreground">
                    {apiKey.lastUsedAt ? formatDate(apiKey.lastUsedAt) : '-'}
                  </TableCell>
                  <TableCell>
                    {apiKey.status === 'active' && (
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="h-8 w-8">
                            <MoreHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          <DropdownMenuItem
                            onClick={() => copyToClipboard(apiKey.key, apiKey.id)}
                          >
                            <Copy className="mr-2 h-4 w-4" />
                            复制密钥
                          </DropdownMenuItem>
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            className="text-destructive focus:text-destructive"
                            onClick={() => setKeyToRevoke(apiKey)}
                          >
                            <Trash2 className="mr-2 h-4 w-4" />
                            撤销密钥
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </CardContent>
      </Card>

      {/* Revoke Confirmation */}
      <AlertDialog open={!!keyToRevoke} onOpenChange={() => setKeyToRevoke(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确定要撤销此密钥吗？</AlertDialogTitle>
            <AlertDialogDescription>
              撤销后，使用此密钥的所有应用程序将无法访问 Gateway API。此操作不可撤销。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              onClick={() => setKeyToRevoke(null)}
            >
              撤销密钥
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
