"use client"

import { useEffect, useRef } from "react"
import mermaid from "mermaid"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Separator } from "@/components/ui/separator"

function MermaidChart({ chart }: { chart: string }) {
  const chartRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    mermaid.initialize({
      startOnLoad: true,
      theme: "default",
      securityLevel: "loose",
    })
  }, [])

  useEffect(() => {
    if (chartRef.current) {
      const id = `mermaid-${Math.random().toString(36).substr(2, 9)}`
      mermaid.render(id, chart).then((result) => {
        if (chartRef.current) {
          chartRef.current.innerHTML = result.svg
        }
      })
    }
  }, [chart])

  return <div ref={chartRef} className="flex justify-center" />
}

export default function DocsPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-12 sm:px-6 lg:px-8">
      <div className="mb-12">
        <h1 className="text-4xl font-bold tracking-tight sm:text-5xl">MCP Gateway 用户文档</h1>
        <p className="mt-4 text-lg text-muted-foreground">
          如何在 Workspace 中部署 MCP 服务、连接网关并调用 API
        </p>
      </div>

      <div className="space-y-12">
        {/* 核心概念 */}
        <section>
          <h2 className="mb-4 text-2xl font-semibold">核心概念</h2>
          <Card>
            <CardContent className="pt-6">
              <p className="mb-6 text-muted-foreground">
                MCP Gateway 采用三层架构组织资源，理解它们的关系有助于更好地使用平台。
              </p>
              
              <div className="mb-8 grid gap-6 md:grid-cols-3">
                <div className="rounded-lg border p-4">
                  <div className="mb-2 flex items-center gap-2">
                    <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-100 text-blue-600 text-sm font-bold">WS</div>
                    <h4 className="font-semibold">Workspace</h4>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    MCP 服务的逻辑分组，用于组织和管理相关的 MCP 服务器。每个 Workspace 独立运行，互不干扰。
                  </p>
                </div>
                <div className="rounded-lg border p-4">
                  <div className="mb-2 flex items-center gap-2">
                    <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-green-100 text-green-600 text-sm font-bold">MCP</div>
                    <h4 className="font-semibold">MCP Service</h4>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    具体的 MCP 服务器实例，部署在 Workspace 中，提供工具、资源和提示等功能。
                  </p>
                </div>
                <div className="rounded-lg border p-4">
                  <div className="mb-2 flex items-center gap-2">
                    <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-purple-100 text-purple-600 text-sm font-bold">SE</div>
                    <h4 className="font-semibold">Session</h4>
                  </div>
                  <p className="text-sm text-muted-foreground">
                    客户端与网关的连接会话，可以绑定到 Workspace 中的多个 MCP 服务，实现统一的工具调用。
                  </p>
                </div>
              </div>

              <div className="rounded-lg bg-muted p-6">
                <h4 className="mb-4 font-medium">关系示意图</h4>
                <MermaidChart chart={`graph TB
    Gateway[MCP Gateway]
    
    subgraph WS_A[Workspace A]
        MCP1[MCP Service 1<br/>filesystem]
        MCP2[MCP Service 2<br/>time]
        MCP3[MCP Service 3<br/>database]
    end
    
    subgraph WS_B[Workspace B]
        MCP4[MCP Service 4<br/>weather]
    end
    
    Session1[Session 1<br/>Client Connection]
    Session2[Session 2<br/>Client Connection]
    
    Gateway --> WS_A
    Gateway --> WS_B
    
    WS_A --> MCP1
    WS_A --> MCP2
    WS_A --> MCP3
    
    WS_B --> MCP4
    
    Session1 -.-> MCP1
    Session1 -.-> MCP2
    Session1 -.-> MCP3
    
    Session2 -.-> MCP4
    
    style Gateway fill:#e0e7ff,stroke:#4f46e5,stroke-width:2px
    style WS_A fill:#dbeafe,stroke:#3b82f6,stroke-width:2px
    style WS_B fill:#dbeafe,stroke:#3b82f6,stroke-width:2px
    style MCP1 fill:#dcfce7,stroke:#22c55e,stroke-width:2px
    style MCP2 fill:#dcfce7,stroke:#22c55e,stroke-width:2px
    style MCP3 fill:#dcfce7,stroke:#22c55e,stroke-width:2px
    style MCP4 fill:#dcfce7,stroke:#22c55e,stroke-width:2px
    style Session1 fill:#f3e8ff,stroke:#a855f7,stroke-width:2px
    style Session2 fill:#f3e8ff,stroke:#a855f7,stroke-width:2px`} />
                <div className="mt-4 space-y-2 text-sm text-muted-foreground">
                  <p>• 一个 <strong>Workspace</strong> 可以包含多个 <strong>MCP Service</strong></p>
                  <p>• 一个 <strong>Session</strong> 绑定到特定的 <strong>Workspace</strong></p>
                  <p>• Session 可以调用该 Workspace 下所有 MCP Service 的工具</p>
                  <p>• 不同 Workspace 的 MCP Service 完全隔离</p>
                </div>
              </div>
            </CardContent>
          </Card>
        </section>

        <Separator />

        {/* 快速开始 */}
        <section>
          <h2 className="mb-4 text-2xl font-semibold">快速开始</h2>
          <Card>
            <CardContent className="pt-6">
              <p className="mb-4 text-muted-foreground">
                MCP Gateway 是一个 MCP 协议聚合网关，允许您在单个 Workspace 中管理多个 MCP 服务器，并通过统一的接口调用它们。
              </p>
              <div className="space-y-4">
                <div className="flex items-start gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium">1</div>
                  <div>
                    <h4 className="font-medium">创建 Workspace</h4>
                    <p className="text-sm text-muted-foreground">在控制台中创建一个新的 Workspace，用于组织您的 MCP 服务</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium">2</div>
                  <div>
                    <h4 className="font-medium">部署 MCP 服务</h4>
                    <p className="text-sm text-muted-foreground">从市场一键安装，或通过配置自定义部署 MCP 服务器</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium">3</div>
                  <div>
                    <h4 className="font-medium">连接网关</h4>
                    <p className="text-sm text-muted-foreground">使用 SSE 或 Streamable HTTP 协议连接到网关</p>
                  </div>
                </div>
                <div className="flex items-start gap-3">
                  <div className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-primary text-primary-foreground text-sm font-medium">4</div>
                  <div>
                    <h4 className="font-medium">调用 API</h4>
                    <p className="text-sm text-muted-foreground">通过网关调用 MCP 工具，网关会自动路由到对应的服务</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </section>

        <Separator />

        {/* Workspace 管理 */}
        <section>
          <h2 className="mb-4 text-2xl font-semibold">Workspace 管理</h2>
          <Card>
            <CardHeader>
              <CardTitle>创建 Workspace</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <p className="text-muted-foreground">
                Workspace 是 MCP 服务的逻辑分组，每个 Workspace 可以包含多个 MCP 服务器。
              </p>
              <div>
                <h4 className="mb-2 font-medium">通过控制台创建</h4>
                <ol className="space-y-2 text-sm text-muted-foreground list-decimal list-inside">
                  <li>登录 MCP Gateway 控制台</li>
                  <li>点击"Workspaces" → "+ New Workspace"</li>
                  <li>填写 Workspace 名称和描述（可选）</li>
                  <li>点击"Create"完成创建</li>
                </ol>
              </div>
              <div>
                <h4 className="mb-2 font-medium">通过 API 创建</h4>
                <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`POST /api/v1/workspaces HTTP/1.1
Authorization: Bearer <your-api-key>
Content-Type: application/json

{
  "id": "my-workspace",
  "name": "My Workspace",
  "description": "My MCP services"
}`}
                </pre>
              </div>
            </CardContent>
          </Card>
        </section>

        <Separator />

        {/* 部署 MCP 服务 */}
        <section>
          <h2 className="mb-4 text-2xl font-semibold">部署 MCP 服务</h2>
          
          <Tabs defaultValue="market">
            <TabsList>
              <TabsTrigger value="market">从市场安装</TabsTrigger>
              <TabsTrigger value="config">从配置部署</TabsTrigger>
              <TabsTrigger value="url">从 URL 连接</TabsTrigger>
            </TabsList>
            
            <TabsContent value="market">
              <Card>
                <CardHeader>
                  <CardTitle>从市场一键安装</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="text-muted-foreground">
                    MCP Market 提供了大量预打包的 MCP 服务器，可以一键安装到您的 Workspace。
                  </p>
                  <div>
                    <h4 className="mb-2 font-medium">通过控制台安装</h4>
                    <ol className="space-y-2 text-sm text-muted-foreground list-decimal list-inside">
                      <li>进入"Market"页面</li>
                      <li>浏览或搜索您需要的 MCP 服务</li>
                      <li>点击服务卡片进入详情页</li>
                      <li>选择目标 Workspace</li>
                      <li>配置环境变量（如需要）</li>
                      <li>点击"Install"开始安装</li>
                    </ol>
                  </div>
                  <div>
                    <h4 className="mb-2 font-medium">通过 API 安装</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`POST /api/v1/workspaces/:ws/services HTTP/1.1
Authorization: Bearer <your-api-key>
Content-Type: application/json

{
  "name": "filesystem",
  "market_package_id": "filesystem-tools",
  "version": "1.2.0",
  "env": {
    "FS_ROOT": "/tmp"
  }
}`}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
            
            <TabsContent value="config">
              <Card>
                <CardHeader>
                  <CardTitle>从配置部署</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="text-muted-foreground">
                    如果您有自定义的 MCP 服务器，可以通过命令和参数直接部署。
                  </p>
                  <div>
                    <h4 className="mb-2 font-medium">通过控制台部署</h4>
                    <ol className="space-y-2 text-sm text-muted-foreground list-decimal list-inside">
                      <li>进入目标 Workspace 的"MCPs"页面</li>
                      <li>点击"+ Deploy MCP" → "From Config"</li>
                      <li>选择部署方式：Command 或 URL</li>
                      <li>填写命令、参数和环境变量</li>
                      <li>点击"Deploy"开始部署</li>
                    </ol>
                  </div>
                  <div>
                    <h4 className="mb-2 font-medium">通过 API 部署（Command）</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`POST /api/v1/workspaces/:ws/services HTTP/1.1
Authorization: Bearer <your-api-key>
Content-Type: application/json

{
  "name": "time",
  "command": "uvx",
  "args": ["mcp-server-time", "--local-timezone=Asia/Shanghai"],
  "env": {
    "TZ": "Asia/Shanghai"
  }
}`}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
            
            <TabsContent value="url">
              <Card>
                <CardHeader>
                  <CardTitle>从 URL 连接</CardTitle>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="text-muted-foreground">
                    如果您已经有一个运行中的 MCP 服务器，可以通过 URL 将其连接到网关。
                  </p>
                  <div>
                    <h4 className="mb-2 font-medium">通过控制台连接</h4>
                    <ol className="space-y-2 text-sm text-muted-foreground list-decimal list-inside">
                      <li>进入目标 Workspace 的"MCPs"页面</li>
                      <li>点击"+ Deploy MCP" → "From Config"</li>
                      <li>选择"URL"方式</li>
                      <li>填写 MCP 服务器的 SSE 端点 URL</li>
                      <li>点击"Deploy"开始连接</li>
                    </ol>
                  </div>
                  <div>
                    <h4 className="mb-2 font-medium">通过 API 连接</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`POST /api/v1/workspaces/:ws/services HTTP/1.1
Authorization: Bearer <your-api-key>
Content-Type: application/json

{
  "name": "remote-mcp",
  "url": "http://mcp-server:8080/sse"
}`}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </section>

        <Separator />

        {/* 连接网关 */}
        <section>
          <h2 className="mb-4 text-2xl font-semibold">连接网关</h2>
          
          <Tabs defaultValue="sse">
            <TabsList>
              <TabsTrigger value="sse">SSE 协议</TabsTrigger>
              <TabsTrigger value="stream">Streamable HTTP 协议</TabsTrigger>
            </TabsList>
            
            <TabsContent value="sse">
              <Card>
                <CardHeader>
                  <CardTitle>SSE（Server-Sent Events）模式</CardTitle>
                  <Badge variant="outline" className="w-fit">传统 MCP 传输</Badge>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="text-muted-foreground">
                    SSE 是 MCP 的传统传输协议，适用于需要实时推送的场景。
                  </p>
                  
                  <div className="rounded-lg bg-muted p-6">
                    <h4 className="mb-4 font-medium">交互时序图</h4>
                    <MermaidChart chart={`sequenceDiagram
    participant Client as 客户端
    participant Gateway as MCP Gateway
    participant WS as Workspace
    participant Session as Session
    participant MCP as MCP Service
    
    Note over Client,WS: 客户端选择 Workspace
    
    Client->>Gateway: GET /sse?api_key=<key>
    activate Gateway
    Gateway->>WS: 验证 API Key
    WS-->>Gateway: Workspace 信息
    Gateway->>Session: 创建 Session
    Session-->>Gateway: Session ID
    Gateway-->>Client: SSE 连接建立
    deactivate Gateway
    
    Note over Client,Session: 建立持久 SSE 连接，Session 绑定 Workspace
    
    Client->>Gateway: POST /message (tools/list)
    activate Gateway
    Gateway->>Session: 验证 Session
    Session->>WS: 获取 Workspace 下 MCP 列表
    WS-->>Session: MCP 列表
    loop 遍历 MCP Services
        Gateway->>MCP: 转发 tools/list 请求
        activate MCP
        MCP-->>Gateway: 返回工具列表
        deactivate MCP
    end
    Gateway-->>Client: SSE event: 聚合工具列表
    deactivate Gateway
    
    Client->>Gateway: POST /message (tools/call)
    activate Gateway
    Gateway->>Session: 验证 Session
    Session->>WS: 获取目标 MCP
    WS-->>Session: MCP 信息
    Gateway->>MCP: 转发 tools/call 请求
    activate MCP
    MCP-->>Gateway: 返回调用结果
    deactivate MCP
    Gateway-->>Client: SSE event: 调用结果
    deactivate Gateway`} />
                  </div>
                  
                  <div>
                    <h4 className="mb-2 font-medium">1. 订阅 SSE 流</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`GET /sse HTTP/1.1
Host: your-gateway.com
Authorization: Bearer <your-api-key>

# 或使用查询参数
GET /sse?api_key=<your-api-key> HTTP/1.1`}
                    </pre>
                  </div>
                  
                  <div>
                    <h4 className="mb-2 font-medium">2. 发送 JSON-RPC 请求</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`POST /message HTTP/1.1
Host: your-gateway.com
Authorization: Bearer <your-api-key>
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/list"
}`}
                    </pre>
                  </div>
                  
                  <div>
                    <h4 className="mb-2 font-medium">3. 调用工具</h4>
                    <p className="mb-2 text-sm text-muted-foreground">
                      工具名称格式为 <code className="bg-muted px-1 py-0.5 rounded text-sm">&lt;服务名&gt;-&lt;工具名&gt;</code>
                    </p>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`POST /message HTTP/1.1
Host: your-gateway.com
Authorization: Bearer <your-api-key>
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "time-get_current_time",
    "arguments": {
      "timezone": "Asia/Shanghai"
    }
  }
}`}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
            
            <TabsContent value="stream">
              <Card>
                <CardHeader>
                  <CardTitle>Streamable HTTP 模式</CardTitle>
                  <Badge variant="outline" className="w-fit">MCP 2025-03-26 规范</Badge>
                </CardHeader>
                <CardContent className="space-y-4">
                  <p className="text-muted-foreground">
                    Streamable HTTP 是 MCP 的新传输协议，支持会话管理和更高效的通信。
                  </p>
                  
                  <div className="rounded-lg bg-muted p-6">
                    <h4 className="mb-4 font-medium">交互时序图</h4>
                    <MermaidChart chart={`sequenceDiagram
    participant Client as 客户端
    participant Gateway as MCP Gateway
    participant WS as Workspace
    participant Session as Session
    participant MCP as MCP Service
    
    Note over Client,WS: 客户端选择 Workspace（通过 API Key scope）
    
    Client->>Gateway: POST /stream (initialize)
    activate Gateway
    Gateway->>WS: 验证 API Key，获取 Workspace
    WS-->>Gateway: Workspace 信息
    Gateway->>Session: 创建 Session，绑定 Workspace
    Session-->>Gateway: Session ID
    Gateway-->>Client: 200 OK + Mcp-Session-Id
    deactivate Gateway
    
    Note over Client,Session: 会话已建立，Session 绑定 Workspace
    
    Client->>Gateway: POST /stream (notifications/initialized)
    activate Gateway
    Gateway->>Session: 验证 Session ID
    Session-->>Gateway: Session 有效
    Gateway-->>Client: 202 Accepted
    deactivate Gateway
    
    Client->>Gateway: POST /stream (tools/list)
    activate Gateway
    Gateway->>Session: 验证 Session ID
    Session->>WS: 获取 Workspace 下 MCP 列表
    WS-->>Session: MCP 列表
    loop 遍历 MCP Services
        Gateway->>MCP: 转发 tools/list 请求
        activate MCP
        MCP-->>Gateway: 返回工具列表
        deactivate MCP
    end
    Gateway-->>Client: 200 OK + 聚合工具列表
    deactivate Gateway
    
    Client->>Gateway: POST /stream (tools/call)
    activate Gateway
    Gateway->>Session: 验证 Session ID
    Session->>WS: 获取目标 MCP
    WS-->>Session: MCP 信息
    Gateway->>MCP: 转发 tools/call 请求
    activate MCP
    MCP-->>Gateway: 返回调用结果
    deactivate MCP
    Gateway-->>Client: 200 OK + 调用结果
    deactivate Gateway
    
    Note over Client,Session: 可选：订阅服务器事件
    
    Client->>Gateway: GET /stream (订阅事件)
    activate Gateway
    Gateway->>Session: 验证 Session ID
    Session-->>Gateway: Session 有效
    Gateway-->>Client: SSE 事件流
    deactivate Gateway
    
    Client->>Gateway: DELETE /stream (关闭会话)
    activate Gateway
    Gateway->>Session: 销毁 Session
    Session-->>Gateway: Session 已销毁
    Gateway-->>Client: 200 OK
    deactivate Gateway`} />
                  </div>
                  
                  <div>
                    <h4 className="mb-2 font-medium">1. 建立会话（initialize）</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`POST /stream HTTP/1.1
Host: your-gateway.com
Authorization: Bearer <your-api-key>
Accept: application/json, text/event-stream
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {},
    "clientInfo": {
      "name": "my-client",
      "version": "1.0.0"
    }
  }
}`}
                    </pre>
                    <p className="mt-2 text-sm text-muted-foreground">
                      响应头会包含 <code className="bg-muted px-1 py-0.5 rounded text-sm">Mcp-Session-Id</code>，请保存它用于后续请求。
                    </p>
                  </div>
                  
                  <div>
                    <h4 className="mb-2 font-medium">2. 完成握手</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`POST /stream HTTP/1.1
Host: your-gateway.com
Authorization: Bearer <your-api-key>
Content-Type: application/json
Mcp-Session-Id: <session-id>

{
  "jsonrpc": "2.0",
  "method": "notifications/initialized"
}`}
                    </pre>
                  </div>
                  
                  <div>
                    <h4 className="mb-2 font-medium">3. 调用工具</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`POST /stream HTTP/1.1
Host: your-gateway.com
Authorization: Bearer <your-api-key>
Content-Type: application/json
Mcp-Session-Id: <session-id>

{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "time_get_current_time",
    "arguments": {
      "timezone": "Asia/Shanghai"
    }
  }
}`}
                    </pre>
                    <p className="mt-2 text-sm text-muted-foreground">
                      工具名称格式为 <code className="bg-muted px-1 py-0.5 rounded text-sm">&lt;服务名&gt;_&lt;工具名&gt;</code>（使用下划线）
                    </p>
                  </div>
                  
                  <div>
                    <h4 className="mb-2 font-medium">4. 订阅服务器事件（可选）</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`GET /stream HTTP/1.1
Host: your-gateway.com
Authorization: Bearer <your-api-key>
Accept: text/event-stream
Mcp-Session-Id: <session-id>`}
                    </pre>
                  </div>
                  
                  <div>
                    <h4 className="mb-2 font-medium">5. 关闭会话</h4>
                    <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`DELETE /stream HTTP/1.1
Host: your-gateway.com
Authorization: Bearer <your-api-key>
Mcp-Session-Id: <session-id>`}
                    </pre>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </section>

        <Separator />

        {/* API 认证 */}
        <section>
          <h2 className="mb-4 text-2xl font-semibold">API 认证</h2>
          <Card>
            <CardContent className="pt-6">
              <p className="mb-4 text-muted-foreground">
                所有 API 请求都需要认证。网关支持多种认证方式：
              </p>
              <div className="space-y-4">
                <div>
                  <h4 className="mb-2 font-medium">1. Bearer Token（推荐）</h4>
                  <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`Authorization: Bearer <your-api-key>`}
                  </pre>
                </div>
                <div>
                  <h4 className="mb-2 font-medium">2. 查询参数</h4>
                  <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-sm">
                    <code>?api_key=&lt;your-api-key&gt;</code>
                  </pre>
                </div>
                <div>
                  <h4 className="mb-2 font-medium">3. 会话 ID（Streamable HTTP）</h4>
                  <p className="text-sm text-muted-foreground">
                    在建立会话后，可以使用 <code className="bg-muted px-1 py-0.5 rounded text-sm">Mcp-Session-Id</code> 头代替 API Key
                  </p>
                  <pre className="mt-2 overflow-x-auto rounded-lg bg-muted p-4 text-sm">
{`Mcp-Session-Id: <session-id>`}
                  </pre>
                </div>
              </div>
              <div className="mt-6 rounded-lg bg-muted p-4">
                <h4 className="mb-2 font-medium">获取 API Key</h4>
                <p className="text-sm text-muted-foreground">
                  在控制台的"API Keys"页面创建和管理您的 API Key。SaaS 模式下可以为不同 Workspace 创建独立的 Key。
                </p>
              </div>
            </CardContent>
          </Card>
        </section>

        <Separator />

        {/* 使用 MCP Inspector */}
        <section>
          <h2 className="mb-4 text-2xl font-semibold">使用 MCP Inspector</h2>
          <Card>
            <CardContent className="pt-6">
              <p className="mb-4 text-muted-foreground">
                MCP Inspector 是官方的 MCP 调试工具，可以方便地测试和调试 MCP 服务器。
              </p>
              
              <Tabs defaultValue="sse-inspector">
                <TabsList>
                  <TabsTrigger value="sse-inspector">SSE 模式</TabsTrigger>
                  <TabsTrigger value="stream-inspector">Streamable HTTP 模式</TabsTrigger>
                </TabsList>
                
                <TabsContent value="sse-inspector">
                  <div className="space-y-4">
                    <div>
                      <h4 className="mb-2 font-medium">配置步骤</h4>
                      <ol className="space-y-2 text-sm text-muted-foreground list-decimal list-inside">
                        <li>在 Inspector 中选择传输类型：<strong>SSE</strong></li>
                        <li>URL：<code className="bg-muted px-1 py-0.5 rounded text-sm">http://your-gateway.com/sse</code></li>
                        <li>在配置中添加自定义头：<code className="bg-muted px-1 py-0.5 rounded text-sm">Authorization: Bearer &lt;your-api-key&gt;</code></li>
                        <li>点击"Connect"连接</li>
                      </ol>
                    </div>
                  </div>
                </TabsContent>
                
                <TabsContent value="stream-inspector">
                  <div className="space-y-4">
                    <div>
                      <h4 className="mb-2 font-medium">配置步骤</h4>
                      <ol className="space-y-2 text-sm text-muted-foreground list-decimal list-inside">
                        <li>在 Inspector 中选择传输类型：<strong>Streamable HTTP</strong></li>
                        <li>URL：<code className="bg-muted px-1 py-0.5 rounded text-sm">http://your-gateway.com/stream</code></li>
                        <li>在配置 → 自定义头中添加：<code className="bg-muted px-1 py-0.5 rounded text-sm">Authorization: Bearer &lt;your-api-key&gt;</code></li>
                        <li>点击"Connect"，Inspector 会自动处理会话管理</li>
                      </ol>
                    </div>
                  </div>
                </TabsContent>
              </Tabs>
            </CardContent>
          </Card>
        </section>

        <Separator />

        {/* Playground */}
        <section>
          <h2 className="mb-4 text-2xl font-semibold">使用 Playground</h2>
          <Card>
            <CardContent className="pt-6">
              <p className="mb-4 text-muted-foreground">
                MCP Gateway 控制台内置了 Playground，可以直接在浏览器中测试 MCP 调用。
              </p>
              <div className="space-y-4">
                <div>
                  <h4 className="mb-2 font-medium">功能特性</h4>
                  <ul className="space-y-1 text-sm text-muted-foreground list-disc list-inside">
                    <li>选择 Workspace 和 MCP 服务</li>
                    <li>切换 SSE / Streamable HTTP 协议</li>
                    <li>预设请求模板（initialize、tools/list、tools/call 等）</li>
                    <li>实时查看响应和历史记录</li>
                    <li>复制为 curl / Node / Python 代码</li>
                  </ul>
                </div>
                <div>
                  <h4 className="mb-2 font-medium">访问 Playground</h4>
                  <p className="text-sm text-muted-foreground">
                    在控制台侧边栏点击"Playground"即可打开。
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>
        </section>
      </div>
    </div>
  )
}
