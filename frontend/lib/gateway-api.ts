'use client'

import useSWR, { mutate } from 'swr'

const API_BASE = process.env.NEXT_PUBLIC_API_BASE || 'http://127.0.0.1:8080'

type Envelope<T> = {
  success: boolean
  data: T
  error: { code: string; message: string; details?: unknown } | null
  timestamp: string
}

export type ListData<T> = {
  items: T[]
  total: number
  page: number
  page_size: number
}

export type OverviewStats = {
  workspaces_count: number
  running_mcps: number
  failed_mcps_24h: number
  active_sessions: number
  recent_activity: Array<{
    at: string
    type: string
    workspace_id?: string
    workspace_name?: string
    service_name?: string
    session_id?: string
    message?: string
  }>
}

export type Workspace = {
  id: string
  name: string
  description: string
  owner_id: string
  status: 'running' | 'stopped' | 'failed'
  mcp_count: number
  session_count: number
  created_at: string
  last_activity_at: string
}

export type Service = {
  name: string
  workspace_id: string
  source_type: 'command' | 'url' | 'market'
  source_ref: string
  command?: string
  args?: string[]
  env?: Record<string, string>
  url?: string
  status: 'starting' | 'running' | 'stopped' | 'failed'
  port?: number
  tools_count: number
  last_error?: string
  retry_count: number
  created_at: string
}

export type Session = {
  id: string
  workspace_id: string
  status: string
  is_ready: boolean
  tools_count: number
  bound_mcp_names: string[]
  created_at: string
  last_receive_time: string
}

export type LogEntry = {
  timestamp: string
  level: 'info' | 'warn' | 'error' | 'debug'
  message: string
  source?: string
  metadata?: Record<string, unknown>
}

export type InstalledItem = {
  package_id: string
  package_name: string
  installed_version: string
  latest_version: string
  workspace_id: string
  workspace_name: string
  service_name: string
  status: string
  installed_at: string
}

export type MarketPackage = {
  id: string
  name: string
  version: string
  description: string
  author: string
  tags: string[]
  rating: number
  downloads: number
  verified: boolean
  source_id: string
  category?: string
  tools: Array<{
    name: string
    description: string
    input_schema?: Record<string, unknown>
  }>
  readme?: string
  versions?: string[]
}

export type SystemConfig = {
  bind: string
  gateway_protocol: 'sse' | 'streamhttp'
  session_gc_interval_seconds: number
  proxy_session_timeout_seconds: number
  mcp_retry_count: number
  auth: {
    enabled: boolean
    mode: string
    allow_register: boolean
  }
}

export type MetaInfo = {
  mode: string
  allow_register: boolean
  oauth_providers: string[]
  gateway_protocol: 'sse' | 'streamhttp'
  version: string
  features: Record<string, boolean>
}

export type MeInfo = {
  id: string
  email: string
  display_name: string
  role: string
  status: string
  builtin: boolean
  created_at: string
}

export type LoginResponse = {
  mode: string
  token_type: string
  token: string
  refresh_token?: string
  user: MeInfo
}

export type UserAPIKey = {
  id: string
  name: string
  workspace_id?: string
  scope: string[]
  status: string
  expires_at?: string
  last_used_at?: string
  created_at: string
  key_prefix: string
  raw_key?: string
}

export class GatewayApiError extends Error {
  status: number
  code?: string
  details?: unknown

  constructor(message: string, status: number, code?: string, details?: unknown) {
    super(message)
    this.name = 'GatewayApiError'
    this.status = status
    this.code = code
    this.details = details
  }
}

function getStoredApiKey() {
  if (typeof window === 'undefined') return null
  return sessionStorage.getItem('mcp_gateway_api_key') || localStorage.getItem('mcp_gateway_api_key')
}

async function request<T>(path: string, init?: RequestInit, options?: { auth?: boolean }): Promise<T> {
  const headers = new Headers(init?.headers)
  if (!headers.has('Content-Type') && init?.body != null) {
    headers.set('Content-Type', 'application/json')
  }
  if (options?.auth !== false) {
    const key = getStoredApiKey()
    if (key) {
      headers.set('Authorization', `Bearer ${key}`)
    }
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...init,
    headers,
  })

  const payload = (await response.json()) as Envelope<T>
  if (!response.ok || !payload.success) {
    throw new GatewayApiError(
      payload.error?.message || 'Request failed',
      response.status,
      payload.error?.code,
      payload.error?.details
    )
  }
  return payload.data
}

const fetcher = <T,>(path: string) => request<T>(path)

export function useGatewaySWR<T>(path: string | null) {
  return useSWR<T>(path, fetcher)
}

export function invalidate(path: string) {
  return mutate(path, undefined, { revalidate: true })
}

export const gatewayApi = {
  getMeta: () => request<MetaInfo>('/api/v1/meta', undefined, { auth: false }),
  login: (body: { api_key?: string; email?: string; password?: string }) =>
    request<LoginResponse>('/api/v1/auth/login', { method: 'POST', body: JSON.stringify(body) }, { auth: false }),
  register: (body: { email: string; password: string; display_name?: string }) =>
    request<{ id: string; email: string; display_name: string; status: string; created_at: string }>('/api/v1/auth/register', { method: 'POST', body: JSON.stringify(body) }, { auth: false }),
  getMe: () => request<MeInfo>('/api/v1/auth/me'),
  getStats: () => request<OverviewStats>('/api/v1/stats/overview'),
  listWorkspaces: () => request<ListData<Workspace>>('/api/v1/workspaces'),
  createWorkspace: (body: { id?: string; name: string; description?: string }) =>
    request<Workspace>('/api/v1/workspaces', { method: 'POST', body: JSON.stringify(body) }),
  getWorkspace: (workspaceId: string) => request<Workspace & { mcps?: Service[]; sessions_active?: number }>(`/api/v1/workspaces/${workspaceId}`),
  updateWorkspace: (workspaceId: string, body: { name?: string; description?: string }) =>
    request<Workspace>(`/api/v1/workspaces/${workspaceId}`, { method: 'PATCH', body: JSON.stringify(body) }),
  deleteWorkspace: (workspaceId: string, cascade = true) =>
    request<{ id: string }>(`/api/v1/workspaces/${workspaceId}?cascade=${cascade}`, { method: 'DELETE' }),
  listServices: (workspaceId: string) => request<ListData<Service>>(`/api/v1/workspaces/${workspaceId}/services`),
  createService: (workspaceId: string, body: Record<string, unknown>) =>
    request<Service>(`/api/v1/workspaces/${workspaceId}/services`, { method: 'POST', body: JSON.stringify(body) }),
  deleteService: (workspaceId: string, name: string) =>
    request<{ name: string }>(`/api/v1/workspaces/${workspaceId}/services/${name}`, { method: 'DELETE' }),
  restartService: (workspaceId: string, name: string) =>
    request<{ status: string }>(`/api/v1/workspaces/${workspaceId}/services/${name}/restart`, { method: 'POST' }),
  stopService: (workspaceId: string, name: string) =>
    request<{ status: string }>(`/api/v1/workspaces/${workspaceId}/services/${name}/stop`, { method: 'POST' }),
  startService: (workspaceId: string, name: string) =>
    request<{ status: string }>(`/api/v1/workspaces/${workspaceId}/services/${name}/start`, { method: 'POST' }),
  listServiceTools: (workspaceId: string, name: string) =>
    request<{ items: Array<{ name: string; description: string; input_schema?: Record<string, unknown> }> }>(
      `/api/v1/workspaces/${workspaceId}/services/${name}/tools`
    ),
  listWorkspaceLogs: (workspaceId: string) =>
    request<{ workspace_id: string; total_lines: number; logs: LogEntry[] }>(`/api/v1/workspaces/${workspaceId}/logs`),
  listSessions: (workspaceId: string) => request<ListData<Session>>(`/api/v1/workspaces/${workspaceId}/sessions`),
  createSession: (workspaceId: string) =>
    request<Session>(`/api/v1/workspaces/${workspaceId}/sessions`, { method: 'POST', body: '{}' }),
  deleteSession: (workspaceId: string, sessionId: string) =>
    request<{ id: string }>(`/api/v1/workspaces/${workspaceId}/sessions/${sessionId}`, { method: 'DELETE' }),
  getSession: (sessionId: string) => request<Session & { recent_messages: unknown[] }>(`/api/v1/sessions/${sessionId}`),
  listInstalled: () => request<ListData<InstalledItem>>('/api/v1/installed'),
  listAPIKeys: () => request<ListData<UserAPIKey>>('/api/v1/api-keys'),
  createAPIKey: (body: { name: string; workspace_id?: string; scope: string[]; expires_at?: string }) =>
    request<UserAPIKey>('/api/v1/api-keys', { method: 'POST', body: JSON.stringify(body) }),
  revokeAPIKey: (id: string) => request<{ id: string; status: string }>(`/api/v1/api-keys/${id}/revoke`, { method: 'POST' }),
  listMarketSources: () => request<ListData<Record<string, unknown>>>('/api/v1/market/sources'),
  listMarketPackages: () => request<ListData<MarketPackage>>('/api/v1/market/packages'),
  getMarketPackage: (id: string) => request<MarketPackage>(`/api/v1/market/packages/${id}`),
  getSystemConfig: () => request<SystemConfig>('/api/v1/system/config'),
  updateSystemConfig: (body: Partial<SystemConfig>) =>
    request<SystemConfig>('/api/v1/system/config', { method: 'PUT', body: JSON.stringify(body) }),
  getSystemApiKey: () => request<{ api_key: string; updated_at: string }>('/api/v1/system/api-key'),
  rotateSystemApiKey: () => request<{ api_key: string; updated_at: string }>('/api/v1/system/api-key/rotate', { method: 'POST' }),
}

export function saveGatewayApiKey(key: string, remember = true) {
  if (typeof window !== 'undefined') {
    if (remember) {
      localStorage.setItem('mcp_gateway_api_key', key)
      sessionStorage.removeItem('mcp_gateway_api_key')
      return
    }
    sessionStorage.setItem('mcp_gateway_api_key', key)
    localStorage.removeItem('mcp_gateway_api_key')
  }
}

export function clearGatewayAuth() {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('mcp_gateway_api_key')
    sessionStorage.removeItem('mcp_gateway_api_key')
    localStorage.removeItem('mcp_gateway_refresh_token')
    sessionStorage.removeItem('mcp_gateway_refresh_token')
  }
}

export function hasGatewayAuth() {
  return !!getStoredApiKey()
}

export function saveGatewayRefreshToken(token: string, remember = true) {
  if (typeof window === 'undefined') return
  if (remember) {
    localStorage.setItem('mcp_gateway_refresh_token', token)
    sessionStorage.removeItem('mcp_gateway_refresh_token')
    return
  }
  sessionStorage.setItem('mcp_gateway_refresh_token', token)
  localStorage.removeItem('mcp_gateway_refresh_token')
}

export async function callGatewayMessage({
  sessionId,
  body,
  protocol = 'sse',
}: {
  sessionId?: string
  body: Record<string, unknown>
  protocol?: 'sse' | 'streamhttp'
}) {
  const headers = new Headers()
  headers.set('Content-Type', 'application/json')
  const apiKey = getStoredApiKey()
  if (apiKey) {
    headers.set('Authorization', `Bearer ${apiKey}`)
  }

  const url = new URL(`${API_BASE}${protocol === 'streamhttp' ? '/stream' : '/message'}`)
  if (protocol === 'streamhttp') {
    if (sessionId) headers.set('Mcp-Session-Id', sessionId)
  } else if (sessionId) {
    url.searchParams.set('sessionId', sessionId)
  }

  const response = await fetch(url.toString(), {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  })

  const text = await response.text()
  return {
    ok: response.ok,
    status: response.status,
    text,
  }
}
