# MCP Gateway API 测试示例

## 前端 TypeScript 使用示例

### 1. 使用 MCPMessageSender 发送消息

```typescript
import { MCPMessageSender } from './services/api';

// 创建消息发送器
const sender = new MCPMessageSender('test-service', 'default');

// 发送 ping 消息
async function testPing() {
  try {
    const response = await sender.ping();
    console.log('Ping response:', response);
  } catch (error) {
    console.error('Ping failed:', error);
  }
}

// 发送自定义消息
async function testCustomMessage() {
  try {
    const response = await sender.sendMessage({
      jsonrpc: "2.0",
      id: 1,
      method: "tools/list",
      params: {}
    });
    console.log('Custom message response:', response);
  } catch (error) {
    console.error('Custom message failed:', error);
  }
}
```

### 2. 使用 SSEConnection 订阅消息

```typescript
import { SSEConnection } from './services/api';

function testSSEConnection() {
  const serviceName = 'test-service';
  const workspaceId = 'default';
  
  const sseUrl = SSEConnection.createSSEUrl(serviceName, workspaceId);
  console.log('SSE URL:', sseUrl); // /{serviceName}/sse?workspace={workspaceId}
  
  const connection = new SSEConnection(
    sseUrl,
    (data) => {
      console.log('Received message:', data);
    },
    (error) => {
      console.error('SSE error:', error);
    },
    () => {
      console.log('SSE connected');
    }
  );
  
  connection.connect();
  
  // 断开连接
  setTimeout(() => {
    connection.disconnect();
  }, 30000);
}
```

### 3. 使用调试 API

```typescript
import { debugApi } from './services/api';

async function testDebugAPI() {
  const workspaceId = 'default';
  const serviceName = 'test-service';
  
  try {
    // 获取调试信息
    const debugInfo = await debugApi.getInfo(workspaceId, serviceName);
    console.log('Debug info:', debugInfo.data);
    
    // 测试连接
    const connectionTest = await debugApi.testConnection(workspaceId, serviceName);
    console.log('Connection test:', connectionTest.data);
    
    // 发送调试消息
    const testResult = await debugApi.testService(workspaceId, serviceName, {
      message: JSON.stringify({ jsonrpc: "2.0", id: 1, method: "ping", params: {} })
    });
    console.log('Test result:', testResult.data);
    
    // 获取日志
    const logs = await debugApi.getLogs(workspaceId, serviceName, 10);
    console.log('Service logs:', logs.data);
    
  } catch (error) {
    console.error('Debug API test failed:', error);
  }
}
```

## HTTP 请求示例

### 1. SSE 订阅
```bash
# 使用 curl 测试 SSE 连接（注意：实际应用中应该使用 EventSource）
curl -N -H "Accept: text/event-stream" \
  "http://localhost:8080/test-service/sse?workspace=default"
```

### 2. 发送消息
```bash
curl -X POST "http://localhost:8080/test-service/message" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer 123456" \
  -H "X-Workspace-Id: default" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "ping",
    "params": {}
  }'
```

### 3. 调试 API 调用
```bash
# 获取调试信息
curl "http://localhost:8080/api/workspaces/default/services/test-service/debug/info" \
  -H "Authorization: Bearer 123456"

# 测试连接
curl "http://localhost:8080/api/workspaces/default/services/test-service/debug/connection" \
  -H "Authorization: Bearer 123456"

# 发送调试消息
curl -X POST "http://localhost:8080/api/workspaces/default/services/test-service/debug/test" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer 123456" \
  -d '{
    "message": "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\",\"params\":{}}"
  }'

# 获取日志
curl "http://localhost:8080/api/workspaces/default/services/test-service/debug/logs?limit=10" \
  -H "Authorization: Bearer 123456"
```

## axios 配置说明

### API 实例配置
```typescript
// 普通 API 调用（带 /api 前缀）
const api = axios.create({
  baseURL: "/api",
  headers: {
    "Content-Type": "application/json",
    Authorization: "Bearer 123456",
  },
});

// MCP 代理调用（不带 /api 前缀）
const mcpApi = axios.create({
  baseURL: "/",
  headers: {
    "Content-Type": "application/json",
    Authorization: "Bearer 123456",
  },
});
```

### 路径映射
- **普通 API**: `/api/workspaces/...` → 使用 `api` 实例
- **MCP 代理**: `/{serviceName}/message` → 使用 `mcpApi` 实例
- **SSE 代理**: `/{serviceName}/sse` → 直接使用 EventSource

## 完整的 React 组件示例

```tsx
import React, { useState } from 'react';
import { MCPMessageSender, SSEConnection } from '../services/api';

export function MCPTestComponent() {
  const [message, setMessage] = useState('');
  const [response, setResponse] = useState('');
  const [sseMessages, setSseMessages] = useState<string[]>([]);
  const [connection, setConnection] = useState<SSEConnection | null>(null);

  const sender = new MCPMessageSender('test-service', 'default');

  const handleSendMessage = async () => {
    try {
      const result = await sender.sendMessage(JSON.parse(message));
      setResponse(JSON.stringify(result, null, 2));
    } catch (error) {
      setResponse(`Error: ${error.message}`);
    }
  };

  const handleConnectSSE = () => {
    const sseUrl = SSEConnection.createSSEUrl('test-service', 'default');
    const newConnection = new SSEConnection(
      sseUrl,
      (data) => setSseMessages(prev => [...prev, JSON.stringify(data)]),
      (error) => console.error('SSE error:', error)
    );
    newConnection.connect();
    setConnection(newConnection);
  };

  const handleDisconnectSSE = () => {
    connection?.disconnect();
    setConnection(null);
  };

  return (
    <div>
      <h3>MCP Test Component</h3>
      
      <div>
        <h4>Send Message</h4>
        <textarea 
          value={message} 
          onChange={(e) => setMessage(e.target.value)}
          placeholder='{"jsonrpc": "2.0", "id": 1, "method": "ping", "params": {}}'
        />
        <button onClick={handleSendMessage}>Send</button>
        <pre>{response}</pre>
      </div>

      <div>
        <h4>SSE Connection</h4>
        <button onClick={handleConnectSSE} disabled={!!connection}>
          Connect
        </button>
        <button onClick={handleDisconnectSSE} disabled={!connection}>
          Disconnect
        </button>
        <div>
          {sseMessages.map((msg, i) => (
            <div key={i}>{msg}</div>
          ))}
        </div>
      </div>
    </div>
  );
}
```

这个架构确保了：
1. 所有 API 调用都通过 axios
2. 正确的路径映射（API vs 代理）
3. 适当的 header 配置
4. workspace 信息的正确传递