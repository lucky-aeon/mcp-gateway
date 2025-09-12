# MCP Market API 详细规范

## 📋 API 概述

### Base URL
```
Production: https://your-domain.com/api/market
Development: http://localhost:8080/api/market
```

### 通用响应格式
```json
{
  "success": true,
  "data": {...},
  "error": null,
  "timestamp": "2024-01-01T12:00:00Z"
}

// 错误响应
{
  "success": false,
  "data": null,
  "error": {
    "code": "VALIDATION_ERROR", 
    "message": "Invalid parameters",
    "details": {...}
  },
  "timestamp": "2024-01-01T12:00:00Z"
}
```

### 错误码定义
```json
{
  "VALIDATION_ERROR": "参数验证失败",
  "NOT_FOUND": "资源不存在",
  "CONFLICT": "资源冲突",
  "UNAUTHORIZED": "未授权访问", 
  "INTERNAL_ERROR": "服务器内部错误",
  "MARKET_SOURCE_UNREACHABLE": "市场源无法访问",
  "PACKAGE_INSTALL_FAILED": "包安装失败",
  "WORKSPACE_NOT_FOUND": "工作区不存在"
}
```

## 🌐 市场源管理 API

### 1. 获取市场源列表
```http
GET /sources
```

**响应示例：**
```json
{
  "success": true,
  "data": {
    "sources": [
      {
        "id": "official",
        "name": "MCP Official Registry",
        "url": "https://registry.mcp.dev",
        "trusted": true,
        "enabled": true,
        "priority": 1,
        "description": "官方维护的MCP服务器注册表",
        "last_synced": "2024-01-01T12:00:00Z",
        "total_packages": 45,
        "status": "healthy"
      },
      {
        "id": "community-001",
        "name": "MCP Community Hub", 
        "url": "https://github.com/mcp-community/registry",
        "trusted": false,
        "enabled": true,
        "priority": 2,
        "description": "社区维护的MCP服务器集合",
        "last_synced": "2024-01-01T11:30:00Z",
        "total_packages": 23,
        "status": "healthy"
      }
    ],
    "total": 2
  }
}
```

### 2. 添加市场源
```http
POST /sources
Content-Type: application/json
```

**请求体：**
```json
{
  "name": "My Private Registry",
  "url": "https://private.company.com/mcp-registry", 
  "description": "公司内部MCP服务器注册表",
  "trusted": false,
  "priority": 10
}
```

**响应：**
```json
{
  "success": true,
  "data": {
    "id": "private-001",
    "name": "My Private Registry",
    "url": "https://private.company.com/mcp-registry",
    "trusted": false,
    "enabled": true,
    "priority": 10,
    "status": "pending",
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

### 3. 更新市场源
```http
PUT /sources/{source_id}
```

**请求体：**
```json
{
  "name": "Updated Registry Name",
  "enabled": false,
  "priority": 5
}
```

### 4. 删除市场源
```http
DELETE /sources/{source_id}
```

### 5. 同步市场源
```http
POST /sources/{source_id}/sync
```

**响应：**
```json
{
  "success": true,
  "data": {
    "sync_id": "sync-001",
    "status": "in_progress",
    "started_at": "2024-01-01T12:00:00Z"
  }
}
```

## 📦 包搜索和信息 API

### 1. 搜索MCP包
```http
GET /search?q={query}&source={source_id}&category={category}&limit={limit}&offset={offset}&sort={sort}
```

**查询参数：**
- `q`: 搜索关键词（可选）
- `source`: 指定市场源ID（可选，不指定则搜索所有源）
- `category`: 分类筛选（可选）
- `verified_only`: 只显示已验证包（可选，true/false）
- `limit`: 每页数量（默认20，最大100）
- `offset`: 偏移量（默认0）
- `sort`: 排序方式（relevance, downloads, updated）

**响应示例：**
```json
{
  "success": true,
  "data": {
    "packages": [
      {
        "id": "filesystem-tools",
        "name": "filesystem-tools",
        "version": "1.2.0",
        "description": "File system operations for MCP",
        "author": "example-dev", 
        "source_id": "official",
        "source_name": "MCP Official Registry",
        "tags": ["filesystem", "files", "productivity"],
        "category": "filesystem",
        "downloads": 2100,
        "license": "MIT",
        "verified": true,
        "updated_at": "2024-01-01T10:00:00Z",
        "summary": "提供文件读写、目录操作等基础文件系统功能"
      }
    ],
    "total": 45,
    "has_more": true,
    "search_time_ms": 23
  }
}
```

### 2. 获取包详细信息
```http
GET /packages/{package_id}?source={source_id}
```

**响应示例：**
```json
{
  "success": true,
  "data": {
    "id": "filesystem-tools",
    "name": "filesystem-tools", 
    "version": "1.2.0",
    "description": "File system operations for MCP",
    "author": "example-dev",
    "source_id": "official",
    "source_name": "MCP Official Registry",
    "repository": "https://github.com/user/mcp-filesystem",
    "license": "MIT",
    "tags": ["filesystem", "files", "productivity"],
    "category": "filesystem",
    "downloads": 2100,
    "verified": true,
    "readme": "# MCP Filesystem Tools\n\n提供文件系统操作功能...",
    "changelog": "## v1.2.0\n- 新增目录监听功能\n- 修复权限问题",
    "install_spec": {
      "type": "uvx",
      "command": "uvx mcp-filesystem", 
      "args": ["--port", "{port}"],
      "env_vars": {
        "FILESYSTEM_ROOT": {
          "description": "文件系统根目录",
          "required": false,
          "default": "/workspace"
        }
      },
      "requirements": {
        "python": ">=3.8",
        "system": "linux,darwin,windows"
      }
    },
    "capabilities": {
      "tools": [
        {
          "name": "read_file",
          "description": "读取文件内容"
        },
        {
          "name": "write_file", 
          "description": "写入文件内容"
        },
        {
          "name": "list_directory",
          "description": "列出目录内容"
        }
      ],
      "resources": [
        {
          "name": "file_contents",
          "description": "文件内容资源"
        }
      ]
    },
    "versions": ["1.2.0", "1.1.0", "1.0.0"],
    "created_at": "2023-06-01T12:00:00Z",
    "updated_at": "2024-01-01T10:00:00Z"
  }
}
```

### 3. 获取分类列表
```http
GET /categories
```

**响应：**
```json
{
  "success": true,
  "data": {
    "categories": [
      {
        "id": "filesystem",
        "name": "文件系统",
        "description": "文件操作、目录管理等功能",
        "icon": "folder",
        "package_count": 15
      },
      {
        "id": "network", 
        "name": "网络工具",
        "description": "HTTP请求、API调用等网络功能",
        "icon": "cloud",
        "package_count": 8
      },
      {
        "id": "ai",
        "name": "AI集成",
        "description": "与AI模型和服务的集成",
        "icon": "smart_toy", 
        "package_count": 31
      }
    ]
  }
}
```


## 🚀 安装管理 API

### 1. 安装包到工作区
```http
POST /install
Content-Type: application/json
```

**请求体：**
```json
{
  "package_id": "filesystem-tools",
  "source_id": "official",
  "workspace": "default",
  "config": {
    "env_vars": {
      "FILESYSTEM_ROOT": "/custom/path"
    }
  }
}
```

**响应：**
```json
{
  "success": true,
  "data": {
    "install_id": "install-001",
    "status": "in_progress",
    "package_id": "filesystem-tools",
    "workspace": "default",
    "started_at": "2024-01-01T12:00:00Z",
    "progress": {
      "current_step": "downloading",
      "total_steps": 4,
      "percentage": 25
    }
  }
}
```

### 2. 获取安装状态
```http
GET /install/{install_id}/status
```

**响应：**
```json
{
  "success": true,
  "data": {
    "install_id": "install-001",
    "status": "completed",
    "package_id": "filesystem-tools",
    "workspace": "default",
    "started_at": "2024-01-01T12:00:00Z",
    "completed_at": "2024-01-01T12:02:00Z",
    "service_info": {
      "name": "filesystem-tools",
      "port": 10001,
      "status": "running",
      "health_url": "http://localhost:10001/health"
    },
    "logs": [
      {
        "timestamp": "2024-01-01T12:00:30Z",
        "level": "info",
        "message": "开始下载包..."
      },
      {
        "timestamp": "2024-01-01T12:01:45Z", 
        "level": "info",
        "message": "安装完成，服务已启动"
      }
    ]
  }
}
```

### 3. 获取已安装包列表
```http
GET /installed?workspace={workspace}&status={status}
```

**响应：**
```json
{
  "success": true,
  "data": {
    "packages": [
      {
        "package_id": "filesystem-tools",
        "name": "filesystem-tools",
        "workspace": "default", 
        "source_id": "official",
        "installed_at": "2024-01-01T12:00:00Z",
        "status": "running",
        "service_info": {
          "port": 10001,
          "uptime_seconds": 86400,
          "health_status": "healthy"
        }
      }
    ],
    "total": 5
  }
}
```

### 4. 卸载包
```http
DELETE /installed/{package_id}?workspace={workspace}
```

**响应：**
```json
{
  "success": true,
  "data": {
    "uninstall_id": "uninstall-001",
    "status": "in_progress",
    "package_id": "filesystem-tools",
    "workspace": "default"
  }
}
```


## 📊 统计和监控 API

### 1. 获取市场统计信息
```http
GET /stats
```

**响应：**
```json
{
  "success": true,
  "data": {
    "total_packages": 68,
    "total_downloads": 15420,
    "active_sources": 3,
    "top_packages": [
      {
        "package_id": "filesystem-tools",
        "downloads": 2100
      }
    ],
    "categories_stats": [
      {
        "category": "ai",
        "package_count": 31,
        "total_downloads": 8500
      }
    ]
  }
}
```

## 🔧 系统管理 API

### 1. 健康检查
```http
GET /health
```

**响应：**
```json
{
  "success": true,
  "data": {
    "status": "healthy",
    "version": "1.0.0",
    "sources_status": [
      {
        "source_id": "official", 
        "status": "healthy",
        "response_time_ms": 120
      }
    ],
    "database_status": "healthy",
    "cache_status": "healthy"
  }
}
```


## 📋 数据模型

### MarketSource (市场源)
```typescript
interface MarketSource {
  id: string;
  name: string;
  url: string;
  trusted: boolean;
  enabled: boolean;
  priority: number;
  description?: string;
  last_synced?: string;
  total_packages?: number;
  status: 'healthy' | 'error' | 'pending';
  created_at: string;
  updated_at: string;
}
```

### MCPPackage (MCP包)
```typescript
interface MCPPackage {
  id: string;
  name: string;
  version: string;
  description: string;
  author: string;
  source_id: string;
  source_name: string;
  repository?: string;
  license: string;
  tags: string[];
  category: string;
  downloads: number;
  verified: boolean;
  readme: string;
  changelog?: string;
  install_spec: InstallSpec;
  capabilities: {
    tools: Array<{name: string; description: string;}>;
    resources: Array<{name: string; description: string;}>;
  };
  created_at: string;
  updated_at: string;
}
```

### InstallSpec (安装规范)
```typescript
interface InstallSpec {
  type: 'uvx' | 'npm' | 'docker' | 'url';
  command: string;
  args: string[];
  env_vars: Record<string, {
    description: string;
    required: boolean;
    default?: string;
  }>;
  requirements: {
    python?: string;
    node?: string;
    system: string;
  };
}
```

### InstalledPackage (已安装包)
```typescript
interface InstalledPackage {
  package_id: string;
  name: string;
  workspace: string;
  source_id: string;
  installed_at: string;
  status: 'running' | 'stopped' | 'failed';
  service_info: {
    port: number;
    uptime_seconds: number;
    health_status: 'healthy' | 'unhealthy';
  };
}
```