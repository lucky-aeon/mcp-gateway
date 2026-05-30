import React from 'react'
import ReactDOM from 'react-dom/client'
import { BrowserRouter, Navigate, Route, Routes, useParams } from 'react-router-dom'

import AdminLayout from '@/app/(admin)/layout'
import APIKeysRoute from '@/app/(admin)/api-keys/page'
import DashboardRoute from '@/app/(admin)/dashboard/page'
import InstalledRoute from '@/app/(admin)/installed/page'
import MarketRoute from '@/app/(admin)/market/page'
import OpsDashboardRoute from '@/app/(admin)/ops-dashboard/page'
import PlaygroundRoute from '@/app/(admin)/playground/page'
import SetupRoute from '@/app/(admin)/setup/page'
import WorkspacesRoute from '@/app/(admin)/workspaces/page'
import LoginRoute from '@/app/login/page'
import MarketingLayout from '@/app/(marketing)/layout'
import DocsRoute from '@/app/(marketing)/docs/page'
import HomeRoute from '@/app/(marketing)/page'
import PricingRoute from '@/app/(marketing)/pricing/page'
import { WorkspaceDetailPage } from '@/components/pages/workspace-detail-page'
import { ThemeProvider } from '@/components/providers/theme-provider'
import { Toaster } from '@/components/ui/toaster'

import '@/app/globals.css'

function WorkspaceDetailRoute() {
  const { id } = useParams()
  return <WorkspaceDetailPage workspaceId={id ?? ''} />
}

function App() {
  return (
    <React.StrictMode>
      <ThemeProvider defaultTheme="system" storageKey="gateway-admin-theme">
        <BrowserRouter>
          <Routes>
            <Route element={<MarketingLayout />}>
              <Route index element={<HomeRoute />} />
              <Route path="pricing" element={<PricingRoute />} />
              <Route path="docs" element={<DocsRoute />} />
            </Route>

            <Route path="login" element={<LoginRoute />} />

            <Route element={<AdminLayout />}>
              <Route path="dashboard" element={<DashboardRoute />} />
              <Route path="ops-dashboard" element={<OpsDashboardRoute />} />
              <Route path="workspaces" element={<WorkspacesRoute />} />
              <Route path="workspaces/:id" element={<WorkspaceDetailRoute />} />
              <Route path="market" element={<MarketRoute />} />
              <Route path="installed" element={<InstalledRoute />} />
              <Route path="api-keys" element={<APIKeysRoute />} />
              <Route path="playground" element={<PlaygroundRoute />} />
              <Route path="setup" element={<SetupRoute />} />
            </Route>

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
        <Toaster />
      </ThemeProvider>
    </React.StrictMode>
  )
}

ReactDOM.createRoot(document.getElementById('root')!).render(<App />)
