# MCP Gateway

## Description

The MCP gateway is a reverse proxy server that forwards requests from clients to the MCP server or uses all MCP servers under the gateway through a unified portal.

Supports two transport protocols (switchable at startup):

- **SSE** (default, legacy MCP transport)
- **Streamable HTTP** (MCP spec `2025-03-26`)

## Features

- Deploy multiple MCP servers
- Connect to MCP server
- Use gateway to call MCP servers
- Get all MCP servers' SSE streams
- Get all MCP servers' tools
- Streamable HTTP aggregated endpoint with session management via `Mcp-Session-Id` header
- Dynamic capability aggregation (gateway only advertises capabilities that at least one downstream MCP supports)
- MCP OAuth 2.1 resource server authentication with Protected Resource Metadata discovery

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

## Configuration

The gateway reads `config.json` from the config directory (defaults to `./vm` when present, otherwise `.`). A minimal example:

```json
{
    "LogLevel": 0,
    "Bind": "[::]:8080",
    "Auth": {
        "Enabled": true,
        "AuthorizationServers": ["https://auth.example.com"],
        "TokenIssuer": "https://auth.example.com",
        "TokenJWKSURI": "https://auth.example.com/.well-known/jwks.json",
        "TokenAudience": "http://localhost:8080/stream",
        "RequiredScopes": ["mcp:read"],
        "ScopesSupported": ["mcp:read"]
    },
    "GatewayProtocol": "streamhttp",
    "McpServiceMgrConfig": {
        "McpServiceRetryCount": 3
    }
}
```

Key fields:

| Field                                   | Default       | Description                                                                        |
| --------------------------------------- | ------------- | ---------------------------------------------------------------------------------- |
| `Bind`                                  | `[::]:8080`   | Server listen address.                                                             |
| `GatewayProtocol`                       | `sse`         | Transport protocol: `sse` or `streamhttp`. Also overridable via `--protocol` flag. |
| `Auth.Enabled`                          | `true`        | Whether to enforce authentication. Set `false` for local unauthenticated MCP use.   |
| `Auth.ApiKey`                           | `123456`      | Legacy single-key management API login token; not used as MCP auth.                |
| `Auth.Mode`                             | `single-key`  | `single-key` or `saas`. SaaS mode uses MCP Gateway accounts and password login.      |
| `Auth.AuthorizationServers`             | empty         | Optional external OAuth issuer URLs. Empty in SaaS mode means the gateway advertises itself as the authorization server. |
| `Auth.TokenIssuer`                      | first authorization server | Expected `iss` claim for MCP OAuth access tokens.                                  |
| `Auth.TokenJWKSURI`                     | discovered from issuer | JWKS URL used to verify JWT access tokens.                                         |
| `Auth.TokenIntrospectionURL`            | empty         | RFC 7662 introspection endpoint for opaque access tokens. If set, introspection is used instead of JWKS. |
| `Auth.TokenAudience`                    | request resource URL | Expected `aud` claim. Configure to the MCP resource indicator used by your authorization server. |
| `Auth.RequiredScopes`                   | empty         | Scopes required for MCP requests and advertised in `WWW-Authenticate`.             |
| `Auth.ScopesSupported`                  | empty         | Scopes advertised in Protected Resource Metadata.                                  |
| `SessionGCInterval`                     | `10s`         | Interval for garbage-collecting idle proxy sessions.                               |
| `ProxySessionTimeout`                   | `1m`          | Timeout for idle proxy sessions before GC.                                         |
| `McpServiceMgrConfig.McpServiceRetryCount` | `3`        | Max retries for a failed MCP service before marking it `failed`.                   |

### Selecting the gateway protocol

Either set `GatewayProtocol` in `config.json`:

```json
{ "GatewayProtocol": "streamhttp" }
```

Or pass the CLI flag (takes precedence):

```bash
./mcp-gateway --protocol=streamhttp
```

Valid values: `sse` (default) or `streamhttp`.

## Authentication

When `Auth.Enabled` is `true`, every MCP protocol request must present a Bearer token:

```http
Authorization: Bearer <access-token>
```

The gateway no longer treats `api_key`, `sessionId`, `Mcp-Session-Id`, or `X-Session-Id` as authentication credentials. `Mcp-Session-Id` remains a transport session identifier and must be sent together with the Bearer token on authenticated Streamable HTTP requests.

For MCP OAuth discovery, the gateway exposes OAuth Protected Resource Metadata:

```http
GET /.well-known/oauth-protected-resource
GET /.well-known/oauth-protected-resource/stream
```

Unauthorized MCP requests return `401` with a `WWW-Authenticate: Bearer ... resource_metadata="..."` challenge when OAuth discovery is available. For local unauthenticated development, set `Auth.Enabled` to `false`.

In `Auth.Mode = "saas"` with no external `Auth.AuthorizationServers`, the gateway uses its own account system for MCP login. Discovery advertises the gateway origin as the authorization server and exposes:

```http
GET /.well-known/oauth-authorization-server
POST /oauth/token
POST /oauth/register
```

`/oauth/token` accepts form-encoded `grant_type=password` with `username`/`password` and returns the same gateway JWT used by `/api/v1/auth/login`. Configure `Auth.AuthorizationServers` only when you want Keycloak, Auth0, or another external OAuth provider.

Browser-based MCP clients can also use the advertised authorization endpoint:

```http
GET /oauth/authorize
```

The built-in authorization endpoint renders a Gateway account login form and completes the OAuth authorization-code flow, including PKCE.
`/oauth/register` implements minimal dynamic client registration for clients such as MCP Inspector.

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

### Use MCP (SSE Mode)

> Available when `GatewayProtocol` is `sse` (default).

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

### Use Gateway (SSE Mode)

> Available when `GatewayProtocol` is `sse` (default).

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

### Use Gateway (Streamable HTTP Mode)

> Available when `GatewayProtocol` is `streamhttp` (set via config or `--protocol=streamhttp`).
>
> Implements the MCP Streamable HTTP transport defined in spec `2025-03-26`. The gateway exposes a single aggregated endpoint `/stream` that accepts `POST`, `GET` and `DELETE`. Session identifiers are carried in the `Mcp-Session-Id` HTTP header.

#### 1. Establish a session (initialize)

```http
POST /stream HTTP/1.1
Host: localhost:8080
Authorization: Bearer <access-token>
Accept: application/json, text/event-stream
Content-Type: application/json

{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
        "protocolVersion": "2025-03-26",
        "capabilities": {},
        "clientInfo": {"name": "my-client", "version": "1.0.0"}
    }
}
```

Response:

```http
HTTP/1.1 200 OK
Content-Type: application/json
Mcp-Session-Id: 7782f2f9-563c-4379-b961-df06e49e54c0

{
    "jsonrpc": "2.0",
    "id": 1,
    "result": {
        "protocolVersion": "2025-03-26",
        "serverInfo": {"name": "mcp-gateway", "version": "1.0.0"},
        "capabilities": { /* OR-merged from all downstream MCP servers */ },
        "instructions": "MCP Gateway aggregates multiple MCP servers. Tools are namespaced as <serverName>_<toolName>."
    }
}
```

Keep the returned `Mcp-Session-Id` and send it on every subsequent request.

#### 2. Complete the handshake (notification)

```http
POST /stream HTTP/1.1
Host: localhost:8080
Authorization: Bearer <access-token>
Content-Type: application/json
Mcp-Session-Id: 7782f2f9-563c-4379-b961-df06e49e54c0

{"jsonrpc": "2.0", "method": "notifications/initialized"}
```

Response: `202 Accepted` (empty body).

#### 3. Call tools or list resources

```http
POST /stream HTTP/1.1
Host: localhost:8080
Authorization: Bearer <access-token>
Accept: application/json, text/event-stream
Content-Type: application/json
Mcp-Session-Id: 7782f2f9-563c-4379-b961-df06e49e54c0

{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
        "name": "{mcp-server-name}_get_current_time",
        "arguments": {"timezone": "Asia/Seoul"}
    }
}
```

Aggregated tool names follow the pattern `<serverName>_<toolName>`, same rule as the SSE gateway mode.

The response arrives synchronously in the HTTP response body:

```json
{"jsonrpc": "2.0", "id": 2, "result": { /* ... */ }}
```

Notifications (JSON-RPC messages without `id`) are answered with `202 Accepted` and forwarded asynchronously.

#### 4. Subscribe to server-initiated events (optional)

```http
GET /stream HTTP/1.1
Host: localhost:8080
Authorization: Bearer <access-token>
Accept: text/event-stream
Mcp-Session-Id: 7782f2f9-563c-4379-b961-df06e49e54c0
```

The gateway keeps the connection open and emits `event: message` frames for server → client JSON-RPC **requests** and **notifications** (e.g. progress updates, log messages). JSON-RPC **responses** are never pushed here — they are returned in the HTTP response of the originating `POST /stream` request.

Lines starting with `:` are SSE keepalive comments and can be ignored.

#### 5. Close the session

```http
DELETE /stream HTTP/1.1
Host: localhost:8080
Authorization: Bearer <access-token>
Mcp-Session-Id: 7782f2f9-563c-4379-b961-df06e49e54c0
```

Response: `200 OK`.

#### Single-server passthrough

In Streamable HTTP mode you can also reach an individual MCP server directly:

```http
POST /{mcp-server-name} HTTP/1.1
GET  /{mcp-server-name} HTTP/1.1
```

The gateway forwards the request to the target MCP's `message` endpoint. Session management in this mode is the responsibility of the downstream server.

#### Connecting with MCP Inspector

1. In Inspector select **Transport Type**: `Streamable HTTP`.
2. URL: `http://localhost:8080/stream`.
3. Complete the OAuth flow in your authorization server and provide `Authorization: Bearer <access-token>`.
4. Click **Connect**. The Inspector handles the `Mcp-Session-Id` exchange automatically.
