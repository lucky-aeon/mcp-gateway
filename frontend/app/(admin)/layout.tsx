'use client'

import { useAppStore } from '@/lib/store'
import { Sidebar } from '@/components/layout/sidebar'
import { Header } from '@/components/layout/header'
import { cn } from '@/lib/utils'

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { sidebarOpen } = useAppStore()

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
