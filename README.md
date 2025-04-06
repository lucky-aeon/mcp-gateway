# MCP Gateway

## Description

The MCP gateway is a reverse proxy server that forwards requests from clients to the MCP server or uses all MCP servers under the gateway through a unified portal.

## Features

- Deploy multiple MCP servers
- Connect to MCP server
- Use gateway to call MCP servers, WIP
- Get all MCP servers' SSE streams, WIP
- Get all MCP servers' tools, WIP

> At present, STDIO turns to the supergateway used by SSE, and we will implement this function ourselves later
https://github.com/supercorp-ai/supergateway

## Installation

1. pull github package

```bash
docker pull ghcr.io/lucky-aeon/mcp-gateway:latest
```

2. self build docker image

```bash
docker build -t mcp-gateway .
```

## Usage

run github docker container

```bash
docker run -d --name mcp-gateway -p 8080:8080 ghcr.io/lucky-aeon/mcp-gateway
```

run self build docker container

```bash
docker run -d --name mcp-gateway -p 8080:8080 mcp-gateway
```

## API

### Deploy

support: uvx, npx. or sse url
```http
POST /deploy HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
    "mcpServers": {
        "time": {
            "url": "http://mcp-server:8080",  // url 和 command 二选一
            "command": "uvx",  // url 和 command 二选一
            "args": ["mcp-server-time", "--local-timezone=America/New_York"],  // 可选，command 的参数
            "env": {  // 可选，环境变量
                "KEY1": "VALUE1",
                "KEY2": "VALUE2"
            }
        }
    }
}
```

### Use MCP

#### GET SSE

```http
GET /{mcp-server-name}/sse HTTP/1.1
Host: localhost:8080
```

#### POST Message

```http
POST /{mcp-server-name}/message HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
    "method": "tools/call",
    "params": {
        "name": "get_current_time",
        "arguments": {
            "timezone": "Asia/Seoul"
        }
    },
    "jsonrpc": "2.0",
    "id": 2
}
```

### Use Gateway

网关和直连MCP的区别在于，只需要与网关交互，网关会自动将请求转发到对应的MCP服务器。在call 时，需要在method前面添加 `mcpServerName` 内容，标识该请求来自哪个 MCP 服务器。

#### GET SSE

```http
GET /sse HTTP/1.1
Host: localhost:8080
```

这里 sse 是整个网关下所有的 MCP 服务器的 SSE 流。

当客户端订阅 sse 时，网关会为每个 MCP 服务器创建一个 SSE 连接，并将所有 MCP 服务器的 SSE 流合并到一起。

在响应的所有tools/call 的结果中，会在method前面添加 `mcpServerName` 内容，标识该结果来自哪个 MCP 服务器。

#### POST Message

```http
POST /message HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
    "method": "tools/call",
    "params": {
        "name": "{mcp-server-name}-get_current_time",
        "arguments": {
            "timezone": "Asia/Seoul"
        }
    },
    "jsonrpc": "2.0",
    "id": 2
}
```

获取网关下所有工具

```http
POST /message HTTP/1.1
Host: localhost:8080
Content-Type: application/json

{
    "method": "tools/list",
    "jsonrpc": "2.0",
    "id": 1
}

# SSE 响应 message event

{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "tools": [
      {
        "name": "{mcpServerName}-get_current_time",
        "description": "Get current time in a specific timezones",
        "inputSchema": {
          "type": "object",
          "properties": {
            "timezone": {
              "type": "string",
              "description": "IANA timezone name (e.g., 'America/New_York', 'Europe/London'). Use 'America/New_York' as local timezone if no timezone provided by the user."
            }
          },
          "required": [
            "timezone"
          ]
        }
      },
      {
        "name": "{mcpServerName}-convert_time",
        "description": "Convert time between timezones",
        "inputSchema": {
          "type": "object",
          "properties": {
            "source_timezone": {
              "type": "string",
              "description": "Source IANA timezone name (e.g., 'America/New_York', 'Europe/London'). Use 'America/New_York' as local timezone if no source timezone provided by the user."
            },
            "time": {
              "type": "string",
              "description": "Time to convert in 24-hour format (HH:MM)"
            },
            "target_timezone": {
              "type": "string",
              "description": "Target IANA timezone name (e.g., 'Asia/Tokyo', 'America/San_Francisco'). Use 'America/New_York' as local timezone if no target timezone provided by the user."
            }
          },
          "required": [
            "source_timezone",
            "time",
            "target_timezone"
          ]
        }
      }
    ]
  }
}
```

