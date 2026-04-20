'use client'

import { useEffect, useMemo, useState } from 'react'
import {
  CheckCircle,
  ChevronRight,
  Copy,
  Database,
  FileText,
  Loader2,
  Play,
  Search,
  Sparkles,
  Terminal,
  Trash2,
  XCircle,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { cn } from '@/lib/utils'
import { callGatewayMessage, gatewayApi, useGatewaySWR, type ListData, type MetaInfo, type Service, type Workspace } from '@/lib/gateway-api'

interface ToolCall {
  id: string
  title: string
  tool: string
  mcp: string
  params: Record<string, unknown>
  status: 'pending' | 'success' | 'error'
  result?: unknown
  error?: string
  duration?: number
  timestamp: string
}

const exampleCalls = [
  {
    id: 'ex-1',
    title: '获取工具列表',
    description: '列出当前会话下所有可用工具',
    request: { jsonrpc: '2.0', id: 1, method: 'tools/list' },
    icon: FileText,
  },
  {
    id: 'ex-2',
    title: '初始化会话',
    description: '发送 initialize 请求验证连接',
    request: { jsonrpc: '2.0', id: 1, method: 'initialize', params: { protocolVersion: '2025-03-26', capabilities: {}, clientInfo: { name: 'gateway-admin', version: '1.0.0' } } },
    icon: Database,
  },
  {
    id: 'ex-3',
    title: 'Ping 网关',
    description: '快速探测 session 通路是否正常',
    request: { jsonrpc: '2.0', id: 1, method: 'ping' },
    icon: Search,
  },
]

export function PlaygroundPage() {
  const { data: meta } = useGatewaySWR<MetaInfo>('/api/v1/meta')
  const { data: workspacesData } = useGatewaySWR<ListData<Workspace>>('/api/v1/workspaces')
  const [selectedWorkspace, setSelectedWorkspace] = useState('')
  const { data: servicesData } = useGatewaySWR<ListData<Service>>(selectedWorkspace ? `/api/v1/workspaces/${selectedWorkspace}/services` : null)
  const { data: sessionsData } = useGatewaySWR<ListData<{ id: string }>>(selectedWorkspace ? `/api/v1/workspaces/${selectedWorkspace}/sessions` : null)
  const [selectedSession, setSelectedSession] = useState('')
  const [requestBody, setRequestBody] = useState(JSON.stringify(exampleCalls[0].request, null, 2))
  const [responseText, setResponseText] = useState('')
  const [history, setHistory] = useState<ToolCall[]>([])
  const [isExecuting, setIsExecuting] = useState(false)
  const [activeTab, setActiveTab] = useState('config')

  useEffect(() => {
    if (!selectedWorkspace && workspacesData?.items?.[0]?.id) {
      setSelectedWorkspace(workspacesData.items[0].id)
    }
  }, [selectedWorkspace, workspacesData?.items])

  const sessions = sessionsData?.items || []
  const services = servicesData?.items || []
  const runningServices = services.filter((item) => item.status === 'running')
  const toolsSummary = useMemo(() => services.reduce((sum, item) => sum + item.tools_count, 0), [services])

  async function handleCreateSession() {
    if (!selectedWorkspace) return
    const session = await gatewayApi.createSession(selectedWorkspace)
    setSelectedSession(session.id)
  }

  async function handleExecute() {
    setIsExecuting(true)
    const callId = `call-${Date.now()}`
    let title = '自定义请求'
    try {
      const body = JSON.parse(requestBody) as Record<string, unknown>
      title = typeof body.method === 'string' ? body.method : '自定义请求'
      const newCall: ToolCall = {
        id: callId,
        title,
        tool: String(body.method || 'custom'),
        mcp: selectedWorkspace,
        params: body,
        status: 'pending',
        timestamp: new Date().toISOString(),
      }
      setHistory((prev) => [newCall, ...prev])
      setActiveTab('history')

      const started = Date.now()
      const result = await callGatewayMessage({
        sessionId: selectedSession || undefined,
        body,
        protocol: meta?.gateway_protocol || 'sse',
      })
      setResponseText(result.text)
      setHistory((prev) =>
        prev.map((item) =>
          item.id === callId
            ? {
                ...item,
                status: result.ok ? 'success' : 'error',
                result: result.text,
                error: result.ok ? undefined : `HTTP ${result.status}`,
                duration: Date.now() - started,
              }
            : item
        )
      )
    } catch (error) {
      const message = error instanceof Error ? error.message : '发送失败'
      setResponseText(message)
      setHistory((prev) =>
        prev.map((item) =>
          item.id === callId
            ? { ...item, status: 'error', error: message }
            : item
        )
      )
    } finally {
      setIsExecuting(false)
    }
  }

  function clearHistory() {
    setHistory([])
  }

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-4">
        <Card className="border-primary/20 bg-gradient-to-br from-primary/10 to-primary/5">
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">可用 MCP</p>
                <p className="text-2xl font-bold">{runningServices.length}</p>
              </div>
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <Terminal className="h-5 w-5 text-primary" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">可用工具</p>
                <p className="text-2xl font-bold">{toolsSummary}</p>
              </div>
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                <Sparkles className="h-5 w-5 text-muted-foreground" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">本次调用</p>
                <p className="text-2xl font-bold">{history.length}</p>
              </div>
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-muted">
                <Play className="h-5 w-5 text-muted-foreground" />
              </div>
            </div>
          </CardContent>
        </Card>
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">成功率</p>
                <p className="text-2xl font-bold">
                  {history.length > 0
                    ? Math.round((history.filter((h) => h.status === 'success').length / Math.max(1, history.filter((h) => h.status !== 'pending').length)) * 100) || 0
                    : '--'}
                  %
                </p>
              </div>
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-green-500/10">
                <CheckCircle className="h-5 w-5 text-green-500" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-5">
        <div className="space-y-6 lg:col-span-2">
          <Card>
            <CardHeader className="pb-4">
              <CardTitle className="text-lg">快速示例</CardTitle>
              <CardDescription>点击示例快速填充请求并测试</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {exampleCalls.map((example) => (
                <button
                  key={example.id}
                  onClick={() => {
                    setRequestBody(JSON.stringify(example.request, null, 2))
                    setActiveTab('config')
                  }}
                  className="group flex w-full items-center gap-3 rounded-lg border border-border bg-card p-3 text-left transition-all hover:border-primary/30 hover:bg-muted/50"
                >
                  <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary/10">
                    <example.icon className="h-4 w-4 text-primary" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-sm font-medium">{example.title}</p>
                    <p className="truncate text-xs text-muted-foreground">{example.description}</p>
                  </div>
                  <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground transition-colors group-hover:text-primary" />
                </button>
              ))}
            </CardContent>
          </Card>

          <Card>
            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <CardHeader className="pb-2">
                <TabsList className="grid w-full grid-cols-2">
                  <TabsTrigger value="config">配置</TabsTrigger>
                  <TabsTrigger value="history" className="relative">
                    历史
                    {history.length > 0 && (
                      <Badge variant="secondary" className="ml-1.5 h-5 px-1.5 text-xs">
                        {history.length}
                      </Badge>
                    )}
                  </TabsTrigger>
                </TabsList>
              </CardHeader>
              <CardContent>
                <TabsContent value="config" className="mt-0 space-y-4">
                  <div className="space-y-2">
                    <Label>工作空间</Label>
                    <Select value={selectedWorkspace} onValueChange={setSelectedWorkspace}>
                      <SelectTrigger><SelectValue placeholder="选择工作空间" /></SelectTrigger>
                      <SelectContent>
                        {workspacesData?.items.map((ws) => (
                          <SelectItem key={ws.id} value={ws.id}>
                            <span className="flex items-center gap-2">
                              <span className={cn('h-2 w-2 rounded-full', ws.status === 'running' ? 'bg-green-500' : 'bg-muted-foreground')} />
                              {ws.name}
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label>Session</Label>
                    <Select value={selectedSession} onValueChange={setSelectedSession}>
                      <SelectTrigger><SelectValue placeholder="选择已有会话或新建" /></SelectTrigger>
                      <SelectContent>
                        {sessions.map((session) => (
                          <SelectItem key={session.id} value={session.id}>{session.id}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    <Button variant="outline" size="sm" onClick={handleCreateSession} disabled={!selectedWorkspace}>
                      新建会话
                    </Button>
                  </div>

                  <div className="space-y-2">
                    <Label>当前工作空间服务</Label>
                    <div className="flex flex-wrap gap-2 rounded-xl border bg-muted/30 p-3">
                      {services.map((service) => (
                        <Badge key={service.name} variant="outline">{service.name} · {service.tools_count} 工具</Badge>
                      ))}
                      {!services.length && <span className="text-sm text-muted-foreground">暂无服务</span>}
                    </div>
                  </div>
                </TabsContent>

                <TabsContent value="history" className="mt-0 space-y-3">
                  <div className="flex items-center justify-end">
                    <Button variant="ghost" size="sm" onClick={clearHistory}>
                      <Trash2 className="mr-2 h-4 w-4" />
                      清空历史
                    </Button>
                  </div>
                  <ScrollArea className="h-[300px]">
                    <div className="space-y-2">
                      {history.map((item) => (
                        <div key={item.id} className="rounded-lg border bg-muted/30 p-3">
                          <div className="flex items-start justify-between gap-3">
                            <div className="min-w-0">
                              <p className="truncate text-sm font-medium">{item.title}</p>
                              <p className="truncate text-xs text-muted-foreground">{new Date(item.timestamp).toLocaleTimeString('zh-CN')}</p>
                            </div>
                            <div className="flex items-center gap-1">
                              {item.status === 'pending' && <Loader2 className="h-4 w-4 animate-spin text-amber-500" />}
                              {item.status === 'success' && <CheckCircle className="h-4 w-4 text-emerald-500" />}
                              {item.status === 'error' && <XCircle className="h-4 w-4 text-red-500" />}
                            </div>
                          </div>
                          {item.duration != null && <p className="mt-2 text-xs text-muted-foreground">耗时 {item.duration}ms</p>}
                        </div>
                      ))}
                      {!history.length && <p className="text-sm text-muted-foreground">还没有执行记录。</p>}
                    </div>
                  </ScrollArea>
                </TabsContent>
              </CardContent>
            </Tabs>
          </Card>
        </div>

        <div className="space-y-6 lg:col-span-3">
          <Card>
            <CardHeader>
              <CardTitle className="text-lg">请求编辑器</CardTitle>
              <CardDescription>直接向网关发送 JSON-RPC 请求，协议：{meta?.gateway_protocol || 'sse'}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <Textarea value={requestBody} onChange={(e) => setRequestBody(e.target.value)} rows={14} className="font-mono text-sm" />
              <div className="flex gap-2">
                <Button onClick={handleExecute} disabled={isExecuting}>
                  <Play className="mr-2 h-4 w-4" />
                  {isExecuting ? '发送中...' : '发送'}
                </Button>
                <Button variant="outline" onClick={() => navigator.clipboard.writeText(requestBody)}>
                  <Copy className="mr-2 h-4 w-4" />
                  复制请求
                </Button>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-lg">响应结果</CardTitle>
              <CardDescription>展示网关原始返回内容</CardDescription>
            </CardHeader>
            <CardContent>
              <Textarea value={responseText} readOnly rows={14} className="font-mono text-sm" />
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
