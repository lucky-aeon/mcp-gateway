# MCP Gateway 调试功能使用指南

## 概述

MCP Gateway 现在支持完整的前端调试功能，包括：
- SSE 实时订阅
- JSON-RPC 消息发送
- 连接测试和健康检查
- 服务日志查看

## API 路由说明

### 1. SSE 订阅
- **路由**: `/{serviceName}/sse`
- **方法**: GET (EventSource)
- **参数**: 
  - Query: `workspace={workspaceId}` (可选，默认为 "default")
  - Header: `X-Workspace-Id: {workspaceId}` (备选方案)

### 2. 消息发送
- **路由**: `/{serviceName}/message`
- **方法**: POST
- **Headers**:
  - `Content-Type: application/json`
  - `Authorization: Bearer 123456`
  - `X-Workspace-Id: {workspaceId}`
- **Body**: JSON-RPC 消息

### 3. 调试 API
- **获取调试信息**: `GET /api/workspaces/{workspace}/services/{name}/debug/info`
- **发送调试消息**: `POST /api/workspaces/{workspace}/services/{name}/debug/test`
- **测试连接**: `GET /api/workspaces/{workspace}/services/{name}/debug/connection`
- **获取日志**: `GET /api/workspaces/{workspace}/services/{name}/debug/logs`

## 前端使用方式

### 1. React 组件集成

在 Services 页面中，每个运行中的服务都有一个 "Debug" 按钮：

```tsx
// 点击 Debug 按钮打开调试面板
<MCPDebugPanel service={service} workspaceId={workspaceId} />
```

调试面板包含 4 个标签页：
- **SSE Subscription**: 实时订阅服务消息
- **Send Messages**: 发送 JSON-RPC 消息
- **Debug & Test**: 连接测试和调试
- **Logs**: 查看服务日志

### 2. 直接 API 调用

#### SSE 订阅示例
```javascript
// 通过查询参数指定 workspace
const eventSource = new EventSource('/test-service/sse?workspace=default');

eventSource.onmessage = function(event) {
    console.log('Received:', event.data);
};
```

#### 消息发送示例
```javascript
async function sendPing() {
    const response = await fetch('/test-service/message', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer 123456',
            'X-Workspace-Id': 'default',
        },
        body: JSON.stringify({
            jsonrpc: "2.0",
            id: 1,
            method: "ping",
            params: {}
        }),
    });
    
    const result = await response.json();
    console.log('Response:', result);
}
```

## 测试工具

### 1. HTML 测试页面
打开 `test_frontend_debug.html` 在浏览器中进行手动测试：
- 输入服务名称和工作空间 ID
- 测试 SSE 连接
- 发送预定义或自定义消息

### 2. Shell 测试脚本
运行 `./test_debug_api.sh` 测试后端调试 API：
- 自动检查服务器状态
- 测试所有调试端点
- 显示 API 使用示例

## 常用 JSON-RPC 消息

### 1. Ping 测试
```json
{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "ping",
    "params": {}
}
```

### 2. 初始化
```json
{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "initialize",
    "params": {
        "protocolVersion": "2024-11-05",
        "capabilities": {},
        "clientInfo": {
            "name": "MCP Gateway Debug",
            "version": "1.0.0"
        }
    }
}
```

### 3. 列出工具
```json
{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/list",
    "params": {}
}
```

### 4. 列出资源
```json
{
    "jsonrpc": "2.0",
    "id": 4,
    "method": "resources/list",
    "params": {}
}
```

### 5. 调用工具
```json
{
    "jsonrpc": "2.0",
    "id": 5,
    "method": "tools/call",
    "params": {
        "name": "tool_name",
        "arguments": {
            "arg1": "value1"
        }
    }
}
```

## 故障排除

### 1. SSE 连接失败
- 检查服务是否正在运行
- 确认 workspace ID 正确
- 查看浏览器开发者工具的网络标签

### 2. 消息发送失败
- 验证 JSON 格式是否正确
- 检查 X-Workspace-Id header 是否设置
- 确认服务支持该 JSON-RPC 方法

### 3. 服务未找到
- 确认服务名称拼写正确
- 检查服务是否已部署到指定工作空间
- 使用 `/api/workspaces/{workspace}/services` 查看可用服务

## 架构说明

### 代理机制
MCP Gateway 使用代理机制处理前端请求：

1. **前端请求** → `/{serviceName}/sse` 或 `/{serviceName}/message`
2. **代理处理** → 从 header 或查询参数获取 workspace
3. **服务查找** → 在指定 workspace 中查找服务
4. **请求转发** → 转发到实际的 MCP 服务

### Workspace 支持
- 每个 MCP 服务属于特定的 workspace
- 前端通过 `X-Workspace-Id` header 或 `workspace` 查询参数指定
- 默认使用 "default" workspace

这个架构确保了前端可以安全、一致地访问 MCP 服务，而无需关心服务的实际内部地址。