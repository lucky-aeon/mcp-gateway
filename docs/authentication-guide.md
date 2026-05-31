# MCP Gateway 认证指南

## 概述

MCP Gateway 支持两种认证模式：

1. **single-key 模式**：简单的共享 API Key，适合个人使用
2. **saas 模式**：完整的用户账号系统，支持注册/登录/JWT token

## 当前配置

项目当前使用 **saas 模式**，配置如下：

```json
{
  "Auth": {
    "Enabled": true,
    "Mode": "saas",
    "AuthorizationServers": null,
    "AllowRegister": true,
    "JWTSecret": "gateway-dev-secret",
    "AccessTokenTTLMinutes": 120,
    "RefreshTokenTTLHours": 720,
    "MongoURI": "mongodb://localhost:27017",
    "MongoDatabase": "mcp_gateway"
  }
}
```

## 认证流程

### 1. 用户注册

```bash
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "your-password",
  "display_name": "Your Name"
}
```

响应：
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGci...",
    "refresh_token": "...",
    "token_type": "Bearer",
    "expires_in": 7200,
    "account": {
      "id": "...",
      "email": "user@example.com",
      "display_name": "Your Name"
    }
  }
}
```

### 2. 用户登录

```bash
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "your-password"
}
```

响应格式同注册。

### 3. 使用 Token 访问 MCP 服务

获取到 `access_token` 后，可以用它访问 MCP 协议端点：

```bash
POST /stream
Authorization: Bearer <access_token>
Content-Type: application/json

{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {},
    "clientInfo": {
      "name": "cherry-studio",
      "version": "1.0.0"
    }
  }
}
```

## Cherry Studio 配置

### 方式 1：手动配置 Token（推荐）

1. 先通过浏览器或 API 工具登录获取 token
2. 在 Cherry Studio 中添加 MCP 服务器：
   - **类型**：Streamable HTTP
   - **URL**：`http://localhost:8080/stream`
   - **Headers**：
     ```
     Authorization=Bearer <your-access-token>
     ```

### 方式 2：等待 Cherry Studio 支持登录界面

Cherry Studio 未来可能会支持直接在界面中输入用户名密码登录。

## 管理后台

访问 `http://localhost:8080` 可以打开管理后台（如果前端已部署）。

管理后台功能：
- 用户注册/登录
- 工作区管理
- MCP 服务器部署
- 会话管理
- API Key 管理

## 测试

运行测试脚本：

```bash
./test_auth.sh
```

该脚本会：
1. 注册一个测试用户
2. 登录获取 token
3. 测试管理 API
4. 测试 MCP 协议认证

## 技术细节

### 认证实现

MCP Gateway 的认证系统包含两部分：

1. **管理 API 认证**（`/api/v1/*`）
   - 使用 JWT token
   - 在 `internal/admin/v1_api.go` 中实现
   - 支持注册、登录、刷新 token

2. **MCP 协议认证**（`/stream`, `/sse`）
   - 复用管理 API 的 JWT token
   - 在 `internal/gateway/auth.go` 中实现
   - 支持标准的 `Authorization: Bearer <token>` 头

### Token 验证流程

```
1. 客户端发送请求 + Bearer token
2. Gateway 提取 token
3. 优先验证内部 JWT token（管理后台颁发的）
4. 如果配置了外部 OAuth，才尝试 OAuth 验证
5. 验证成功，返回 Principal 对象
6. 请求继续处理
```

### 与外部 OAuth 的兼容性

如果需要支持外部 OAuth 2.0 授权服务器（如 Keycloak、Auth0），可以配置：

```json
{
  "Auth": {
    "AuthorizationServers": ["https://auth.example.com"],
    "TokenIssuer": "https://auth.example.com",
    "TokenJWKSURI": "https://auth.example.com/.well-known/jwks.json",
    "TokenAudience": "http://localhost:8080/stream",
    "RequiredScopes": ["mcp:read"]
  }
}
```

此时 Gateway 会：
1. 先尝试验证内部 JWT
2. 如果失败，再验证外部 OAuth token
3. 暴露 `/.well-known/oauth-protected-resource` 元数据端点

## 常见问题

### Q: Cherry Studio 报错 "MCP auth is enabled but Auth.AuthorizationServers is empty"

**A:** 这是旧版本的错误。更新代码后，即使 `AuthorizationServers` 为空，也可以使用内部 JWT 认证。

### Q: 如何禁用认证？

**A:** 修改 `config.json`：

```json
{
  "Auth": {
    "Enabled": false
  }
}
```

### Q: 忘记密码怎么办？

**A:** 目前没有密码重置功能。可以：
1. 直接修改数据库中的密码哈希
2. 或者删除账号重新注册

### Q: Token 过期时间是多久？

**A:** 
- Access Token: 120 分钟（2小时）
- Refresh Token: 720 小时（30天）

可以在 `config.json` 中修改：
```json
{
  "Auth": {
    "AccessTokenTTLMinutes": 120,
    "RefreshTokenTTLHours": 720
  }
}
```

## 安全建议

1. **生产环境**：
   - 修改 `JWTSecret` 为强随机字符串
   - 使用 HTTPS
   - 配置防火墙限制访问

2. **密码策略**：
   - 当前最小长度：6 位
   - 建议使用强密码

3. **Token 管理**：
   - Access token 存储在内存中
   - Refresh token 可以存储在 httpOnly cookie 中
   - 定期轮换 API Key

## 开发调试

查看认证相关日志：

```bash
# 启动时设置日志级别
go run main.go --log-level debug
```

测试 token 验证：

```bash
# 获取 token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"test123456"}' \
  | jq -r '.data.access_token')

# 测试 MCP 端点
curl -X POST http://localhost:8080/stream \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
```
