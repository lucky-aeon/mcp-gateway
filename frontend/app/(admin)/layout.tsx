'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { Loader2 } from 'lucide-react'

import { useAppStore } from '@/lib/store'
import { Sidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'
import { cn } from '@/lib/utils'
import { clearGatewayAuth, GatewayApiError, useGatewaySWR, type MeInfo } from '@/lib/gateway-api'

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { sidebarOpen } = useAppStore()
  const router = useRouter()
  const { data: me, error, isLoading } = useGatewaySWR<MeInfo>('/api/v1/auth/me')

  useEffect(() => {
    if (error instanceof GatewayApiError && error.status === 401) {
      clearGatewayAuth()
      router.replace('/login')
    }
  }, [error, router])

  if (error && !(error instanceof GatewayApiError && error.status === 401)) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/30">
        <div className="rounded-xl border bg-card px-5 py-4 text-sm text-destructive shadow-sm">
          加载管理控制台失败：{error.message}
        </div>
      </div>
    )
  }

  if (isLoading || !me) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-muted/30">
        <div className="flex items-center gap-3 rounded-xl border bg-card px-5 py-4 text-sm text-muted-foreground shadow-sm">
          <Loader2 className="h-4 w-4 animate-spin" />
          正在验证登录状态...
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-muted/30">
      <Sidebar />
      <Header />
      <main
        className={cn(
          'min-h-[calc(100vh-4rem)] pt-16 transition-all duration-300',
          sidebarOpen ? 'pl-64' : 'pl-16'
        )}
      >
        <div className="container mx-auto max-w-7xl p-6 lg:p-8">{children}</div>
      </main>
    </div>
  )
}
