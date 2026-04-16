# MCP Gateway 协议配置化实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 支持通过配置文件或命令行参数选择网关对外暴露的协议（SSE 或 StreamHTTP）

**Architecture:**
- 配置层：新增 `GatewayProtocol` 字段，支持命令行参数覆盖
- 服务层：通过接口抽象桥接，支持选择 `StdioToSSEBridge` 或 `StdioToHTTPStreamBridge`
- 路由层：根据配置注册对应端点

**Tech Stack:** Go, Echo framework

---

## 文件结构

| 文件 | 职责 |
|------|------|
| `config/config.go` | 配置结构体，新增 `GatewayProtocol` 字段 |
| `main.go` | 命令行参数解析 |
| `service/service.go` | 服务层，根据配置选择桥接类型 |
| `bridge/bridge.go` | 新建，定义桥接接口抽象 |
| `router/server.go` | 路由注册，根据协议注册对应端点 |

---

## Task 1: 配置层 - 添加 GatewayProtocol 字段

**Files:**
- Modify: `config/config.go:1-126`

- [ ] **Step 1: 在 Config 结构体添加 GatewayProtocol 字段**

在 `config/config.go` 的 `Config` 结构体中添加：

```go
type Config struct {
    LogLevel            uint8
    ConfigDirPath       string
    Bind                string
    Auth                *AuthConfig
    SessionGCInterval   time.Duration
    ProxySessionTimeout time.Duration
    McpServiceMgrConfig McpServiceMgrConfig
    GatewayProtocol     string  // 新增: "sse" | "streamhttp"
}
```

- [ ] **Step 2: 在 Default() 方法中设置默认值**

在 `config/config.go` 的 `Default()` 方法中添加：

```go
if c.GatewayProtocol == "" {
    c.GatewayProtocol = "sse"  // 默认 SSE
}
```

- [ ] **Step 3: 添加验证方法**

在 `config/config.go` 添加：

```go
func (c *Config) IsStreamHTTP() bool {
    return c.GatewayProtocol == "streamhttp"
}
```

- [ ] **Step 4: 提交**

```bash
git add config/config.go
git commit -m "feat(config): add GatewayProtocol field"
```

---

## Task 2: 命令行参数解析

**Files:**
- Modify: `main.go:1-106`

- [ ] **Step 1: 添加 flag 解析变量**

在 `main.go` 添加：

```go
var protocolFlag string
flag.StringVar(&protocolFlag, "protocol", "", "Gateway protocol: sse or streamhttp")
```

- [ ] **Step 2: 在 flag 解析后覆盖配置**

在 `flag.Parse()` 后，`config.InitConfig()` 前添加：

```go
flag.Parse()
// 如果命令行指定了 protocol，覆盖配置文件
if protocolFlag != "" {
    cfg.GatewayProtocol = protocolFlag
}
```

- [ ] **Step 3: 提交**

```bash
git add main.go
git commit -m "feat(main): add --protocol command line argument"
```

---

## Task 3: 桥接接口抽象

**Files:**
- Create: `bridge/bridge.go`

- [ ] **Step 1: 创建桥接接口**

创建 `bridge/bridge.go`：

```go
package bridge

import (
    "context"
)

type Bridge interface {
    Start(addr string) error
    Close() error
    Ping(ctx context.Context) error
}

type SSEBridge interface {
    Bridge
    CompleteSseEndpoint() (string, error)
    CompleteMessageEndpoint() (string, error)
}

type HTTPStreamBridge interface {
    Bridge
    CompleteHTTPStreamEndpoint() (string, error)
}
```

- [ ] **Step 2: 让 StdioToSSEBridge 实现 SSEBridge 接口**

修改 `bridge/stdio_to_sse.go`，确保 `StdioToSSEBridge` 实现 `SSEBridge` 接口（已有方法签名匹配）。

- [ ] **Step 3: 让 StdioToHTTPStreamBridge 实现 HTTPStreamBridge 接口**

修改 `bridge/stdio_to_http_stream.go`，确保 `StdioToHTTPStreamBridge` 实现 `HTTPStreamBridge` 接口（已有方法签名匹配）。

- [ ] **Step 4: 提交**

```bash
git add bridge/bridge.go bridge/stdio_to_sse.go bridge/stdio_to_http_stream.go
git commit -m "feat(bridge): add bridge interface abstraction"
```

---

## Task 4: 服务层 - 支持协议选择

**Files:**
- Modify: `service/service.go:1-450`

- [ ] **Step 1: 添加协议相关字段到 McpService**

在 `McpService` 结构体中：

```go
type McpService struct {
    Name    string
    Config  config.MCPServerConfig
    LogFile *os.File
    logger  xlog.Logger
    Port    int

    portMgr PortManagerI

    Status CmdStatus

    RetryCount int
    RetryMax   int

    // 桥接 - 使用接口
    bridge bridge.Bridge
    isSSE  bool  // true = SSE bridge, false = HTTP Stream bridge

    // ... 其他字段不变
}
```

- [ ] **Step 2: 修改 Start() 方法支持协议选择**

在 `service/service.go:137` 的 `Start()` 方法中，修改桥接创建逻辑：

原代码（约 line 174-188）：
```go
// 使用stdio-sse桥接代替supergateway
logger.Infof("Creating stdio-sse bridge for command: %s %s", s.Config.Command, strings.Join(s.Config.Args, " "))

bridgeInstance, err := bridge.NewStdioToSSEBridge(ctx, transport.NewStdio(s.Config.Command, s.Config.GetEnvs(), s.Config.Args...), s.Name)
```

修改为：
```go
// 根据配置选择桥接类型
logger.Infof("Creating bridge for command: %s %s, protocol: %s", s.Config.Command, strings.Join(s.Config.Args, " "), s.Config.GatewayProtocol)

var bridgeInstance bridge.Bridge
if s.Config.GatewayProtocol == "streamhttp" {
    logger.Infof("Using HTTP Stream bridge")
    bridgeInstance, err = bridge.NewStdioToHTTPStreamBridge(ctx, transport.NewStdio(s.Config.Command, s.Config.GetEnvs(), s.Config.Args...), s.Name)
    if err == nil {
        s.isSSE = false
    }
} else {
    logger.Infof("Using SSE bridge")
    bridgeInstance, err = bridge.NewStdioToSSEBridge(ctx, transport.NewStdio(s.Config.Command, s.Config.GetEnvs(), s.Config.Args...), s.Name)
    if err == nil {
        s.isSSE = true
    }
}
```

- [ ] **Step 3: 修改 GetSSEUrl() 和 GetMessageUrl() 方法**

约 line 324-340，根据 `isSSE` 返回对应 URL：

```go
func (s *McpService) GetSSEUrl() string {
    if s.GetStatus() != Running {
        return ""
    }
    if s.isSSE {
        sseUrl, _ := s.bridge.(bridge.SSEBridge).CompleteSseEndpoint()
        return s.GetUrl() + sseUrl
    }
    return ""
}

func (s *McpService) GetMessageUrl() string {
    if s.GetStatus() != Running {
        return ""
    }
    if s.isSSE {
        mesUrl, _ := s.bridge.(bridge.SSEBridge).CompleteMessageEndpoint()
        return s.GetUrl() + mesUrl
    }
    httpUrl, _ := s.bridge.(bridge.HTTPStreamBridge).CompleteHTTPStreamEndpoint()
    return s.GetUrl() + httpUrl
}
```

- [ ] **Step 4: 提交**

```bash
git add service/service.go
git commit -m "feat(service): support protocol selection for bridge"
```

---

## Task 5: 路由层 - 根据协议注册端点

**Files:**
- Modify: `router/server.go:1-104`

- [ ] **Step 1: 添加协议判断和对应路由注册**

在 `NewServerManager` 函数中，修改路由注册逻辑：

原代码（约 line 30-34）：
```go
e.POST("/deploy", m.handleDeploy)
e.DELETE("/delete", m.handleDeleteMcpService)
e.GET("/sse", m.handleGlobalSSE)
e.POST("/message", m.handleGlobalMessage)
```

修改为：
```go
e.POST("/deploy", m.handleDeploy)
e.DELETE("/delete", m.handleDeleteMcpService)

if cfg.IsStreamHTTP() {
    // StreamHTTP 模式
    e.GET("/:service", m.handleStreamHTTP)
    e.POST("/:service", m.handleStreamHTTP)
} else {
    // SSE 模式（默认）
    e.GET("/sse", m.handleGlobalSSE)
    e.POST("/message", m.handleGlobalMessage)
}
```

- [ ] **Step 2: 实现 handleStreamHTTP 处理函数**

在 `router/server.go` 或新建 `router/streamhttp.go` 添加：

```go
func (m *ServerManager) handleStreamHTTP(c echo.Context) error {
    xl := xlog.NewLogger("STREAMHTTP")
    serviceName := c.Param("service")
    workspace := utils.GetWorkspace(c, service.DefaultWorkspace)

    instance, err := m.mcpServiceMgr.GetMcpService(xl, service.NameArg{
        Server:    serviceName,
        Workspace: workspace,
    })
    if err != nil {
        return c.String(http.StatusNotFound, "Service not found")
    }

    targetURL := instance.GetMessageUrl()
    if targetURL == "" {
        return c.String(http.StatusServiceUnavailable, "Service not available")
    }

    // 转发请求到后端
    req, err := http.NewRequest(c.Request().Method, targetURL, c.Request().Body)
    if err != nil {
        return err
    }
    for k, v := range c.Request().Header {
        req.Header[k] = v
    }

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    for k, v := range resp.Header {
        c.Response().Header()[k] = v
    }
    c.Response().WriteHeader(resp.StatusCode)
    _, err = io.Copy(c.Response().Writer, resp.Body)
    return err
}
```

- [ ] **Step 3: 提交**

```bash
git add router/server.go
git commit -m "feat(router): add StreamHTTP endpoint support"
```

---

## Task 6: 集成测试

- [ ] **Step 1: 验证 SSE 模式（默认）**

```bash
go run main.go &
sleep 2
curl -s http://localhost:8005/services | jq .
```

- [ ] **Step 2: 验证 StreamHTTP 模式**

```bash
pkill -f "mcp-gateway"
go run main.go --protocol=streamhttp &
sleep 2
curl -s http://localhost:8005/services | jq .
```

- [ ] **Step 3: 提交**

```bash
git add -A
git commit -m "test: verify protocol switching works"
```

---

## 总结

| Task | 内容 |
|------|------|
| 1 | 配置层添加 GatewayProtocol 字段 |
| 2 | 命令行参数解析 |
| 3 | 桥接接口抽象 |
| 4 | 服务层支持协议选择 |
| 5 | 路由层根据协议注册端点 |
| 6 | 集成测试 |

**Plan complete and saved to `docs/superpowers/plans/2026-04-17-gateway-protocol-design.md`.**