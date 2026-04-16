# MCP Gateway 协议配置化设计

## 概述

MCP Gateway 目前对外只暴露 SSE 协议，计划支持通过配置选择 SSE 或 StreamHTTP 协议对外暴露。

## 需求

- 支持启动时指定网关对外暴露的协议（SSE 或 StreamHTTP）
- 支持配置文件和命令行参数两种配置方式
- 命令行参数优先级高于配置文件

## 设计

### 配置项

**配置文件 (`config.json`)** 新增字段：
```json
{
  "GatewayProtocol": "sse"  // 或 "streamhttp"
}
```

**命令行参数**：
```
--protocol=sse
--protocol=streamhttp
```

### 优先级

命令行参数 > 配置文件 > 默认值(sse)

### 涉及的端点

**SSE 模式**：
- `GET /sse` - 全局SSE事件流
- `POST /message` - 全局消息
- `/{serviceName}/sse` - 单服务SSE
- `/{serviceName}/message` - 单服务消息

**StreamHTTP 模式**：
- `POST /{serviceName}` - 单服务 StreamHTTP 调用
- `GET /{serviceName}` - 单服务 StreamHTTP 调用

### 文件修改

| 文件 | 修改内容 |
|------|----------|
| `config/config.go` | 添加 `GatewayProtocol` 字段 |
| `main.go` | 添加命令行参数解析 |
| `router/server.go` | 根据协议注册对应端点 |
| `service/service.go` | 根据协议选择 `StdioToSSEBridge` 或 `StdioToHTTPStreamBridge` |

## 实现步骤

1. **配置层**：在 `Config` 结构体添加 `GatewayProtocol` 字段，解析命令行 `--protocol` 参数
2. **服务层**：`McpService` 根据配置选择桥接类型
3. **路由层**：`ServerManager` 根据配置注册对应端点
4. **默认值**：未配置时默认使用 SSE 保持向后兼容