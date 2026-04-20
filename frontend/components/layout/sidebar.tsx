'use client'

import { cn } from '@/lib/utils'
import { useAppStore } from '@/lib/store'
import {
  LayoutDashboard,
  Layers,
  Store,
  Package,
  Key,
  Play,
  Settings,
  ChevronLeft,
  ChevronRight,
  Zap,
  BarChart3,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import Link from 'next/link'
import { usePathname } from 'next/navigation'

interface NavItem {
  href: string
  label: string
  icon: React.ComponentType<{ className?: string }>
  matchPattern?: RegExp
  requireSystemAdmin?: boolean
}

const navItems: NavItem[] = [
  { href: '/dashboard', label: '概览', icon: LayoutDashboard },
  { href: '/ops-dashboard', label: '运维看板', icon: BarChart3, requireSystemAdmin: true },
  { href: '/workspaces', label: '工作空间', icon: Layers, matchPattern: /^\/workspaces/ },
  { href: '/market', label: 'MCP 市场', icon: Store },
  { href: '/installed', label: '已安装', icon: Package },
  { href: '/api-keys', label: 'API 密钥', icon: Key },
  { href: '/playground', label: 'Playground', icon: Play },
  { href: '/setup', label: '设置', icon: Settings, requireSystemAdmin: true },
]

export function Sidebar() {
  const { sidebarOpen, setSidebarOpen, currentUser } = useAppStore()
  const pathname = usePathname()

  const isActive = (item: NavItem) => {
    if (item.matchPattern) {
      return item.matchPattern.test(pathname)
    }
    return pathname === item.href
  }

  // 根据用户权限过滤菜单项
  const filteredNavItems = navItems.filter((item) => {
    if (item.requireSystemAdmin) {
      return currentUser?.builtin === true
    }
    return true
  })

  return (
    <TooltipProvider delayDuration={0}>
      <aside
        className={cn(
          'fixed left-0 top-0 z-40 flex h-full flex-col border-r border-border bg-card transition-all duration-300',
          sidebarOpen ? 'w-64' : 'w-16'
        )}
      >
        {/* Logo */}
        <div className="flex h-16 items-center border-b border-border px-3">
          <Link href="/dashboard" className="flex items-center gap-3 overflow-hidden">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-primary/80 shadow-sm">
              <Zap className="h-5 w-5 text-primary-foreground" />
            </div>
            {sidebarOpen && (
              <div className="flex flex-col">
                <span className="font-semibold text-foreground">Gateway</span>
                <span className="text-xs text-muted-foreground">Admin Console</span>
              </div>
            )}
          </Link>
        </div>

        {/* Navigation */}
        <nav className="flex-1 space-y-1 overflow-y-auto p-3">
          {filteredNavItems.map((item) => {
            const Icon = item.icon
            const active = isActive(item)

            const linkContent = (
              <Link
                key={item.href}
                href={item.href}
                className={cn(
                  'flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all',
                  active
                    ? 'bg-primary text-primary-foreground shadow-sm'
                    : 'text-muted-foreground hover:bg-accent hover:text-foreground'
                )}
              >
                <Icon className="h-5 w-5 shrink-0" />
                {sidebarOpen && <span>{item.label}</span>}
              </Link>
            )

            if (!sidebarOpen) {
              return (
                <Tooltip key={item.href}>
                  <TooltipTrigger asChild>{linkContent}</TooltipTrigger>
                  <TooltipContent side="right" sideOffset={8}>
                    <p>{item.label}</p>
                  </TooltipContent>
                </Tooltip>
              )
            }

            return <div key={item.href}>{linkContent}</div>
          })}
        </nav>

        {/* Toggle Button */}
        <div className="border-t border-border p-3">
          <Button
            variant="ghost"
            size="sm"
            className={cn(
              'w-full justify-start gap-3',
              !sidebarOpen && 'justify-center px-0'
            )}
            onClick={() => setSidebarOpen(!sidebarOpen)}
          >
            {sidebarOpen ? (
              <>
                <ChevronLeft className="h-4 w-4" />
                <span>收起侧边栏</span>
              </>
            ) : (
              <ChevronRight className="h-4 w-4" />
            )}
          </Button>
        </div>

        {/* Footer */}
        {sidebarOpen && (
          <div className="border-t border-border px-4 py-3">
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <div className="h-2 w-2 rounded-full bg-emerald-500" />
              <span>系统运行正常</span>
            </div>
          </div>
        )}
      </aside>
    </TooltipProvider>
  )
}
