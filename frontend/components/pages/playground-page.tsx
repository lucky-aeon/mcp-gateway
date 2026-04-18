'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { workspaces, installedMCPs } from '@/lib/mock-data'
import { 
  Play, 
  Terminal, 
  Clock, 
  CheckCircle, 
  XCircle, 
  Loader2, 
  Sparkles,
  FileText,
  Database,
  Search,
  Copy,
  ChevronRight,
  Trash2
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { ScrollArea } from '@/components/ui/scroll-area'

interface ToolCall {
  id: string
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
    title: '读取配置文件',
    description: '使用 Filesystem MCP 读取 JSON 配置文件',
    mcp: 'filesystem',
    tool: 'read_file',
    params: { path: '/data/config.json' },
    icon: FileText,
  },
  {
    id: 'ex-2',
    title: '查询用户数据',
    description: '使用 Database MCP 执行 SQL 查询',
    mcp: 'database',
    tool: 'query',
    params: { sql: 'SELECT * FROM users LIMIT 10' },
    icon: Database,
  },
  {
    id: 'ex-3',
    title: '列出目录',
    description: '列出指定目录下的所有文件',
    mcp: 'filesystem',
    tool: 'list_directory',
    params: { path: '/data' },
    icon: Search,
  },
]

export function PlaygroundPage() {
  const [selectedWorkspace, setSelectedWorkspace] = useState(workspaces[0]?.id || '')
  const [selectedMCP, setSelectedMCP] = useState('')
  const [selectedTool, setSelectedTool] = useState('')
  const [params, setParams] = useState<Record<string, string>>({})
  const [isExecuting, setIsExecuting] = useState(false)
  const [history, setHistory] = useState<ToolCall[]>([])
  const [activeTab, setActiveTab] = useState('config')

  const runningMCPs = installedMCPs.filter((m) => m.status === 'running')
  const selectedMCPData = installedMCPs.find((m) => m.id === selectedMCP)
  const selectedToolData = selectedMCPData?.tools.find((t) => t.name === selectedTool)

  const handleParamChange = (name: string, value: string) => {
    setParams((prev) => ({ ...prev, [name]: value }))
  }

  const handleExecute = async () => {
    if (!selectedTool || !selectedMCPData) return

    setIsExecuting(true)
    const callId = `call-${Date.now()}`
    const newCall: ToolCall = {
      id: callId,
      tool: selectedTool,
      mcp: selectedMCPData.name,
      params: { ...params },
      status: 'pending',
      timestamp: new Date().toISOString(),
    }
    setHistory((prev) => [newCall, ...prev])
    setActiveTab('history')

    await new Promise((resolve) => setTimeout(resolve, 800 + Math.random() * 800))

    const success = Math.random() > 0.15
    const mockResults: Record<string, unknown> = {
      read_file: {
        content: '{\n  "name": "gateway",\n  "version": "1.0.0",\n  "enabled": true\n}',
        size: 256,
        modified: '2024-01-20T10:30:00Z',
      },
      list_directory: {
        files: [
          { name: 'config.json', type: 'file', size: 256 },
          { name: 'data.csv', type: 'file', size: 1024 },
          { name: 'logs', type: 'directory' },
        ],
        count: 3,
      },
      query: {
        rows: [
          { id: 1, name: '张三', email: 'zhangsan@example.com' },
          { id: 2, name: '李四', email: 'lisi@example.com' },
        ],
        rowCount: 2,
        executionTime: '45ms',
      },
    }

    setHistory((prev) =>
      prev.map((call) =>
        call.id === callId
          ? {
              ...call,
              status: success ? 'success' : 'error',
              result: success ? mockResults[selectedTool] || { success: true } : undefined,
              error: success ? undefined : '执行失败：连接超时或参数无效',
              duration: Math.floor(300 + Math.random() * 800),
            }
          : call
      )
    )
    setIsExecuting(false)
  }

  const handleExampleClick = (example: typeof exampleCalls[0]) => {
    setSelectedMCP(example.mcp)
    setSelectedTool(example.tool)
    setParams(example.params as Record<string, string>)
    setActiveTab('config')
  }

  const clearHistory = () => {
    setHistory([])
  }

  const copyResult = (result: unknown) => {
    navigator.clipboard.writeText(JSON.stringify(result, null, 2))
  }

  return (
    <div className="space-y-6">
      {/* Header Stats */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card className="bg-gradient-to-br from-primary/10 to-primary/5 border-primary/20">
          <CardContent className="pt-6">
            <div className="flex items-center justify-between">
              <div>
                <p className="text-sm text-muted-foreground">可用 MCP</p>
                <p className="text-2xl font-bold">{runningMCPs.length}</p>
              </div>
              <div className="h-10 w-10 rounded-lg bg-primary/10 flex items-center justify-center">
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
                <p className="text-2xl font-bold">
                  {runningMCPs.reduce((sum, m) => sum + m.tools.length, 0)}
                </p>
              </div>
              <div className="h-10 w-10 rounded-lg bg-muted flex items-center justify-center">
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
              <div className="h-10 w-10 rounded-lg bg-muted flex items-center justify-center">
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
                    ? Math.round(
                        (history.filter((h) => h.status === 'success').length / 
                         history.filter((h) => h.status !== 'pending').length) * 100
                      ) || 0
                    : '--'}%
                </p>
              </div>
              <div className="h-10 w-10 rounded-lg bg-green-500/10 flex items-center justify-center">
                <CheckCircle className="h-5 w-5 text-green-500" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-6 lg:grid-cols-5">
        {/* Left Panel - Configuration */}
        <div className="lg:col-span-2 space-y-6">
          <Card>
            <CardHeader className="pb-4">
              <CardTitle className="text-lg">快速示例</CardTitle>
              <CardDescription>点击示例快速填充参数并测试</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {exampleCalls.map((example) => (
                <button
                  key={example.id}
                  onClick={() => handleExampleClick(example)}
                  className="w-full flex items-center gap-3 p-3 rounded-lg border border-border bg-card hover:bg-muted/50 hover:border-primary/30 transition-all text-left group"
                >
                  <div className="h-9 w-9 rounded-lg bg-primary/10 flex items-center justify-center shrink-0">
                    <example.icon className="h-4 w-4 text-primary" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="font-medium text-sm truncate">{example.title}</p>
                    <p className="text-xs text-muted-foreground truncate">{example.description}</p>
                  </div>
                  <ChevronRight className="h-4 w-4 text-muted-foreground group-hover:text-primary transition-colors shrink-0" />
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
                      <SelectTrigger>
                        <SelectValue placeholder="选择工作空间" />
                      </SelectTrigger>
                      <SelectContent>
                        {workspaces.map((ws) => (
                          <SelectItem key={ws.id} value={ws.id}>
                            <span className="flex items-center gap-2">
                              <span className={cn(
                                "h-2 w-2 rounded-full",
                                ws.status === 'active' ? 'bg-green-500' : 'bg-muted-foreground'
                              )} />
                              {ws.name}
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label>MCP 服务</Label>
                    <Select 
                      value={selectedMCP} 
                      onValueChange={(v) => {
                        setSelectedMCP(v)
                        setSelectedTool('')
                        setParams({})
                      }}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder="选择 MCP" />
                      </SelectTrigger>
                      <SelectContent>
                        {runningMCPs.map((mcp) => (
                          <SelectItem key={mcp.id} value={mcp.id}>
                            <span className="flex items-center gap-2">
                              <span>{mcp.icon}</span>
                              <span>{mcp.name}</span>
                              <Badge variant="outline" className="text-xs">
                                {mcp.tools.length} 工具
                              </Badge>
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label>工具</Label>
                    <Select 
                      value={selectedTool} 
                      onValueChange={(v) => {
                        setSelectedTool(v)
                        setParams({})
                      }}
                      disabled={!selectedMCP}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder={selectedMCP ? "选择工具" : "请先选择 MCP"} />
                      </SelectTrigger>
                      <SelectContent>
                        {selectedMCPData?.tools.map((tool) => (
                          <SelectItem key={tool.name} value={tool.name}>
                            <span className="flex flex-col items-start">
                              <span className="font-mono text-sm">{tool.name}</span>
                            </span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {selectedToolData && (
                      <p className="text-xs text-muted-foreground">
                        {selectedToolData.description}
                      </p>
                    )}
                  </div>

                  {selectedToolData && (
                    <div className="space-y-3 pt-2 border-t border-border">
                      <Label className="text-muted-foreground">参数</Label>
                      {selectedToolData.parameters.map((param) => (
                        <div key={param.name} className="space-y-1.5">
                          <div className="flex items-center gap-2">
                            <Label htmlFor={param.name} className="text-sm font-mono">
                              {param.name}
                            </Label>
                            <Badge variant="outline" className="text-xs font-normal">
                              {param.type}
                            </Badge>
                            {param.required && (
                              <span className="text-xs text-destructive">*</span>
                            )}
                          </div>
                          <Input
                            id={param.name}
                            placeholder={param.description}
                            value={params[param.name] || ''}
                            onChange={(e) => handleParamChange(param.name, e.target.value)}
                            className="font-mono text-sm"
                          />
                        </div>
                      ))}
                    </div>
                  )}

                  <Button
                    className="w-full"
                    onClick={handleExecute}
                    disabled={isExecuting || !selectedTool}
                    size="lg"
                  >
                    {isExecuting ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        执行中...
                      </>
                    ) : (
                      <>
                        <Play className="mr-2 h-4 w-4" />
                        执行工具
                      </>
                    )}
                  </Button>
                </TabsContent>

                <TabsContent value="history" className="mt-0">
                  {history.length === 0 ? (
                    <div className="flex flex-col items-center justify-center py-8 text-center">
                      <Terminal className="h-10 w-10 text-muted-foreground/50" />
                      <p className="mt-3 text-sm text-muted-foreground">
                        暂无执行历史
                      </p>
                      <p className="text-xs text-muted-foreground">
                        执行工具后，历史记录将显示在这里
                      </p>
                    </div>
                  ) : (
                    <div className="space-y-2">
                      <div className="flex justify-end">
                        <Button variant="ghost" size="sm" onClick={clearHistory}>
                          <Trash2 className="mr-1.5 h-3.5 w-3.5" />
                          清除
                        </Button>
                      </div>
                      <ScrollArea className="h-[300px]">
                        <div className="space-y-2">
                          {history.map((call) => (
                            <div
                              key={call.id}
                              className={cn(
                                'rounded-lg border p-3 cursor-pointer transition-colors hover:bg-muted/50',
                                call.status === 'success' && 'border-green-500/30',
                                call.status === 'error' && 'border-red-500/30',
                                call.status === 'pending' && 'border-border'
                              )}
                            >
                              <div className="flex items-center justify-between">
                                <div className="flex items-center gap-2">
                                  {call.status === 'pending' && (
                                    <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />
                                  )}
                                  {call.status === 'success' && (
                                    <CheckCircle className="h-3.5 w-3.5 text-green-500" />
                                  )}
                                  {call.status === 'error' && (
                                    <XCircle className="h-3.5 w-3.5 text-red-500" />
                                  )}
                                  <code className="text-sm font-medium">{call.tool}</code>
                                </div>
                                {call.duration && (
                                  <span className="text-xs text-muted-foreground">
                                    {call.duration}ms
                                  </span>
                                )}
                              </div>
                              <p className="text-xs text-muted-foreground mt-1">
                                {call.mcp}
                              </p>
                            </div>
                          ))}
                        </div>
                      </ScrollArea>
                    </div>
                  )}
                </TabsContent>
              </CardContent>
            </Tabs>
          </Card>
        </div>

        {/* Right Panel - Result */}
        <div className="lg:col-span-3">
          <Card className="h-full min-h-[600px]">
            <CardHeader className="pb-3 border-b border-border">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Terminal className="h-5 w-5 text-primary" />
                  <CardTitle className="text-lg">执行结果</CardTitle>
                </div>
                {history.length > 0 && history[0].result && (
                  <Button 
                    variant="outline" 
                    size="sm"
                    onClick={() => copyResult(history[0].result)}
                  >
                    <Copy className="mr-1.5 h-3.5 w-3.5" />
                    复制
                  </Button>
                )}
              </div>
            </CardHeader>
            <CardContent className="p-0">
              {history.length === 0 || !history[0] ? (
                <div className="flex flex-col items-center justify-center h-[500px] text-center px-4">
                  <div className="h-16 w-16 rounded-2xl bg-muted flex items-center justify-center mb-4">
                    <Terminal className="h-8 w-8 text-muted-foreground" />
                  </div>
                  <h3 className="font-semibold text-lg mb-2">准备就绪</h3>
                  <p className="text-muted-foreground max-w-sm text-sm">
                    选择一个 MCP 和工具，配置参数后点击执行，结果将实时显示在这里
                  </p>
                  <div className="mt-6 flex items-center gap-2 text-xs text-muted-foreground">
                    <Sparkles className="h-4 w-4" />
                    <span>试试左边的快速示例开始探索</span>
                  </div>
                </div>
              ) : (
                <div className="p-4 space-y-4">
                  {/* Latest Call Info */}
                  <div className="flex items-center justify-between p-3 rounded-lg bg-muted/50">
                    <div className="flex items-center gap-3">
                      {history[0].status === 'pending' && (
                        <Loader2 className="h-5 w-5 animate-spin text-primary" />
                      )}
                      {history[0].status === 'success' && (
                        <CheckCircle className="h-5 w-5 text-green-500" />
                      )}
                      {history[0].status === 'error' && (
                        <XCircle className="h-5 w-5 text-red-500" />
                      )}
                      <div>
                        <div className="flex items-center gap-2">
                          <code className="font-semibold">{history[0].tool}</code>
                          <Badge variant="outline">{history[0].mcp}</Badge>
                        </div>
                        <p className="text-xs text-muted-foreground">
                          {new Date(history[0].timestamp).toLocaleString('zh-CN')}
                        </p>
                      </div>
                    </div>
                    {history[0].duration && (
                      <div className="flex items-center gap-1.5 text-sm text-muted-foreground">
                        <Clock className="h-4 w-4" />
                        <span>{history[0].duration}ms</span>
                      </div>
                    )}
                  </div>

                  {/* Request */}
                  <div>
                    <p className="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">
                      请求参数
                    </p>
                    <pre className="rounded-lg bg-muted p-4 text-sm overflow-auto font-mono">
                      {JSON.stringify(history[0].params, null, 2)}
                    </pre>
                  </div>

                  {/* Response */}
                  {history[0].status !== 'pending' && (
                    <div>
                      <p className="text-xs font-medium text-muted-foreground mb-2 uppercase tracking-wider">
                        {history[0].status === 'success' ? '响应结果' : '错误信息'}
                      </p>
                      {history[0].result ? (
                        <pre className="rounded-lg bg-green-500/5 border border-green-500/20 p-4 text-sm overflow-auto font-mono text-green-700 dark:text-green-400">
                          {JSON.stringify(history[0].result, null, 2)}
                        </pre>
                      ) : (
                        <pre className="rounded-lg bg-red-500/5 border border-red-500/20 p-4 text-sm overflow-auto font-mono text-red-600 dark:text-red-400">
                          {history[0].error}
                        </pre>
                      )}
                    </div>
                  )}

                  {history[0].status === 'pending' && (
                    <div className="flex items-center justify-center py-8">
                      <div className="flex items-center gap-3 text-muted-foreground">
                        <Loader2 className="h-5 w-5 animate-spin" />
                        <span>正在执行...</span>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
