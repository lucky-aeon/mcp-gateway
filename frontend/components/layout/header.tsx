'use client'

import { useAppStore } from '@/lib/store'
import { Bell, Search, User, ChevronRight, LogOut, Settings, UserCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { cn } from '@/lib/utils'
import { usePathname, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'

const routeTitles: Record<string, { title: string; description?: string }> = {
  '/': { title: '概览', description: '系统运行状态和快速操作' },
  '/workspaces': { title: '工作空间', description: '管理您的工作空间' },
  '/market': { title: 'MCP 市场', description: '浏览和安装 MCP 服务' },
  '/installed': { title: '已安装', description: '管理已安装的 MCP 服务' },
  '/api-keys': { title: 'API 密钥', description: '管理 API 访问凭证' },
  '/playground': { title: 'Playground', description: '测试 MCP 工具调用' },
  '/setup': { title: '设置', description: '系统配置与偏好设置' },
}

const notifications = [
  {
    id: '1',
    title: 'MCP 更新可用',
    description: 'Filesystem MCP 有新版本 v1.3.0 可用',
    time: '5 分钟前',
    read: false,
    type: 'update',
  },
  {
    id: '2',
    title: '会话异常',
    description: '检测到来自异常 IP 的会话请求',
    time: '15 分钟前',
    read: false,
    type: 'warning',
  },
  {
    id: '3',
    title: 'API 调用达到阈值',
    description: '今日 API 调用量已达到 80%',
    time: '1 小时前',
    read: true,
    type: 'info',
  },
  {
    id: '4',
    title: '系统备份完成',
    description: '定时备份任务已成功完成',
    time: '2 小时前',
    read: true,
    type: 'success',
  },
]

function getBreadcrumbs(pathname: string) {
  const segments = pathname.split('/').filter(Boolean)
  const breadcrumbs: { label: string; href: string }[] = []

  if (pathname === '/') {
    return [{ label: '概览', href: '/' }]
  }

  let currentPath = ''
  for (const segment of segments) {
    currentPath += `/${segment}`
    const routeInfo = routeTitles[currentPath]
    if (routeInfo) {
      breadcrumbs.push({ label: routeInfo.title, href: currentPath })
    } else if (segment && !segment.startsWith('[')) {
      if (segments[0] === 'workspaces' && segments.length > 1) {
        breadcrumbs.push({ label: '工作空间详情', href: currentPath })
      }
    }
  }

  return breadcrumbs.length > 0 ? breadcrumbs : [{ label: '概览', href: '/' }]
}

export function Header() {
  const { sidebarOpen } = useAppStore()
  const pathname = usePathname()
  const router = useRouter()

  const breadcrumbs = getBreadcrumbs(pathname)
  const unreadCount = notifications.filter((n) => !n.read).length

  const handleLogout = () => {
    localStorage.removeItem('isLoggedIn')
    router.push('/login')
  }

  return (
    <header
      className={cn(
        'fixed right-0 top-0 z-30 flex h-16 items-center justify-between border-b border-border bg-card/80 backdrop-blur-sm px-6 transition-all duration-300',
        sidebarOpen ? 'left-64' : 'left-16'
      )}
    >
      <div className="flex items-center gap-2">
        <nav className="flex items-center gap-1 text-sm">
          {breadcrumbs.map((crumb, index) => (
            <div key={crumb.href} className="flex items-center gap-1">
              {index > 0 && <ChevronRight className="h-4 w-4 text-muted-foreground" />}
              {index === breadcrumbs.length - 1 ? (
                <span className="font-medium text-foreground">{crumb.label}</span>
              ) : (
                <Link 
                  href={crumb.href} 
                  className="text-muted-foreground hover:text-foreground transition-colors"
                >
                  {crumb.label}
                </Link>
              )}
            </div>
          ))}
        </nav>
      </div>

      <div className="flex items-center gap-3">
        {/* Search */}
        <div className="relative hidden md:block">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="搜索 MCP、工作空间..."
            className="w-72 bg-muted/50 border-transparent pl-9 focus:border-border focus:bg-background"
          />
        </div>

        {/* Notifications */}
        <Popover>
          <PopoverTrigger asChild>
            <Button variant="ghost" size="icon" className="relative">
              <Bell className="h-5 w-5" />
              {unreadCount > 0 && (
                <span className="absolute right-1.5 top-1.5 flex h-2 w-2">
                  <span className="absolute inline-flex h-full w-full animate-ping rounded-full bg-destructive opacity-75" />
                  <span className="relative inline-flex h-2 w-2 rounded-full bg-destructive" />
                </span>
              )}
            </Button>
          </PopoverTrigger>
          <PopoverContent align="end" className="w-96 p-0">
            <div className="flex items-center justify-between border-b border-border px-4 py-3">
              <div className="flex items-center gap-2">
                <h4 className="font-semibold">通知</h4>
                {unreadCount > 0 && (
                  <Badge variant="secondary" className="h-5 px-1.5 text-xs">
                    {unreadCount} 条未读
                  </Badge>
                )}
              </div>
              <Button variant="ghost" size="sm" className="h-8 text-xs">
                全部已读
              </Button>
            </div>
            <ScrollArea className="h-[360px]">
              <div className="divide-y divide-border">
                {notifications.map((notification) => (
                  <div
                    key={notification.id}
                    className={cn(
                      'flex gap-3 px-4 py-3 transition-colors hover:bg-muted/50 cursor-pointer',
                      !notification.read && 'bg-primary/5'
                    )}
                  >
                    <div
                      className={cn(
                        'mt-0.5 h-2 w-2 shrink-0 rounded-full',
                        notification.type === 'warning' && 'bg-amber-500',
                        notification.type === 'update' && 'bg-blue-500',
                        notification.type === 'success' && 'bg-green-500',
                        notification.type === 'info' && 'bg-muted-foreground'
                      )}
                    />
                    <div className="flex-1 space-y-1">
                      <p className={cn('text-sm', !notification.read && 'font-medium')}>
                        {notification.title}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {notification.description}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {notification.time}
                      </p>
                    </div>
                  </div>
                ))}
              </div>
            </ScrollArea>
            <div className="border-t border-border p-2">
              <Button variant="ghost" className="w-full text-sm" asChild>
                <Link href="/setup">查看全部通知</Link>
              </Button>
            </div>
          </PopoverContent>
        </Popover>

        {/* User Menu */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="icon" className="rounded-full">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-primary to-primary/80">
                <User className="h-4 w-4 text-primary-foreground" />
              </div>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-56">
            <DropdownMenuLabel>
              <div className="flex flex-col">
                <span>管理员</span>
                <span className="text-xs font-normal text-muted-foreground">admin@gateway.local</span>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem asChild>
              <Link href="/setup" className="flex items-center cursor-pointer">
                <UserCircle className="mr-2 h-4 w-4" />
                个人资料
              </Link>
            </DropdownMenuItem>
            <DropdownMenuItem asChild>
              <Link href="/setup" className="flex items-center cursor-pointer">
                <Settings className="mr-2 h-4 w-4" />
                账户设置
              </Link>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem 
              className="text-destructive focus:text-destructive cursor-pointer"
              onClick={handleLogout}
            >
              <LogOut className="mr-2 h-4 w-4" />
              退出登录
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </header>
  )
}
