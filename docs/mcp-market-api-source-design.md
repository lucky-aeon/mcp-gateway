# MCP Market API 数据源接入设计

## 1. 背景

当前项目的 MCP 市场数据由后端静态列表提供：

- `internal/admin/market_catalog.go` 内置 `defaultMarketSources`
- `internal/admin/market_catalog.go` 内置 `defaultMarketPackages`
- 前端 `frontend/components/pages/market-page.tsx` 通过 `/api/v1/market/packages` 展示和安装

这套实现适合 Demo，但不适合真实市场。真实市场需要从多个 MCP 目录或注册源同步数据，统一展示，并让用户在市场列表中选择安装。

本设计只考虑通过公开 API 或标准 Registry API 接入，不做 HTML 页面爬取。

## 2. 目标

### 2.1 产品目标

- 聚合多个 MCP 市场/注册源的数据。
- 用户在一个市场页面里搜索、筛选、比较 MCP Server。
- 用户可以直接从市场列表或详情页安装到指定 Workspace。
- 市场源可配置、可启停、可手动同步、可查看同步状态。
- 对不同来源的数据做去重和合并，减少重复条目。

### 2.2 技术目标

- 所有外部市场数据先进入后端缓存/数据库，前端只调用本项目 API。
- 每个外部数据源通过独立 Adapter 接入。
- 内部使用统一 `MarketPackage` 模型。
- 只有具备结构化安装信息的数据才允许一键安装。
- 同步失败不能影响现有已缓存市场数据展示。

### 2.3 非目标

- 不做 HTML 爬取。
- 不直接从前端调用第三方市场 API。
- 不在第一阶段实现用户评分、评论、付费市场。
- 不把所有第三方字段完整暴露给前端，只保留必要展示和安装字段。

## 3. 推荐接入数据源

| 优先级 | 数据源 | 接入方式 | 用途 | 安装可信度 |
|---|---|---|---|---|
| P0 | Official MCP Registry | `https://registry.modelcontextprotocol.io/v0.1/servers` | 主索引、版本、标准 server.json | 高 |
| P0 | Smithery | `https://api.smithery.ai/servers` | verified、useCount、remote/hosted 信息 | 高 |
| P1 | Glama | `https://glama.ai/api/mcp/v1/servers` | repository、license、env schema、tools | 中 |
| P1 | PulseMCP | Public API / Registry-compatible API | 热度、分类、补充元数据 | 中 |
| P2 | Docker MCP Catalog | 机器可读 API 或 catalog endpoint | Docker 安装规格、隔离运行 | 高 |

接入原则：

- Official MCP Registry 作为主可信源。
- Smithery 和 Docker 可以作为高质量安装源。
- Glama、PulseMCP 先作为发现和补充信息源；如果返回结构化安装配置，再开放一键安装。
- 没有 API 的目录类网站不接入。

## 4. 总体架构

```text
External APIs
  ├── Official Registry
  ├── Smithery
  ├── Glama
  ├── PulseMCP
  └── Docker Catalog
        │
        ▼
Market Source Adapters
        │
        ▼
Normalizer
        │
        ▼
Dedupe + Merge
        │
        ▼
Market Store
        │
        ├── /api/v1/market/sources
        ├── /api/v1/market/packages
        ├── /api/v1/market/packages/:id
        └── install via workspace service API
```

### 4.1 核心模块

| 模块 | 职责 |
|---|---|
| Source Adapter | 调用第三方 API，处理分页、鉴权、限流、错误 |
| Normalizer | 把第三方数据转成内部统一模型 |
| Dedupe | 根据包名、仓库、包管理器标识合并重复条目 |
| Market Store | 持久化市场源、包、来源引用、同步状态 |
| Install Resolver | 从统一包模型中选择可执行安装方案 |
| Market API | 给前端提供统一查询、详情、同步、安装能力 |

## 5. 数据模型

### 5.1 MarketSource

```go
type MarketSource struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Kind        string    `json:"kind"`
    URL         string    `json:"url"`
    Trusted     bool      `json:"trusted"`
    Enabled     bool      `json:"enabled"`
    Priority    int       `json:"priority"`
    AuthType    string    `json:"auth_type,omitempty"`
    Status      string    `json:"status"`
    LastSynced  time.Time `json:"last_synced,omitempty"`
    LastError   string    `json:"last_error,omitempty"`
    TotalItems  int       `json:"total_items"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

`Kind` 可选值：

- `official_registry`
- `smithery`
- `glama`
- `pulsemcp`
- `docker_catalog`
- `custom_registry`

### 5.2 MarketPackage

```go
type MarketPackage struct {
    ID             string                 `json:"id"`
    CanonicalName  string                 `json:"canonical_name"`
    Name           string                 `json:"name"`
    Title          string                 `json:"title"`
    Description    string                 `json:"description"`
    Author         string                 `json:"author,omitempty"`
    Version        string                 `json:"version,omitempty"`
    Tags           []string               `json:"tags"`
    Category       string                 `json:"category,omitempty"`
    Repository     string                 `json:"repository,omitempty"`
    Homepage       string                 `json:"homepage,omitempty"`
    License        string                 `json:"license,omitempty"`
    Verified       bool                   `json:"verified"`
    Rating         float64                `json:"rating,omitempty"`
    Downloads      int64                  `json:"downloads,omitempty"`
    UseCount       int64                  `json:"use_count,omitempty"`
    Installability string                 `json:"installability"`
    InstallOptions []MarketInstallOption  `json:"install_options"`
    Tools          []MarketToolSpec       `json:"tools"`
    EnvSchema      map[string]interface{} `json:"env_schema,omitempty"`
    SourceRefs     []MarketSourceRef      `json:"source_refs"`
    RawMeta         map[string]interface{} `json:"raw_meta,omitempty"`
    CreatedAt      time.Time              `json:"created_at"`
    UpdatedAt      time.Time              `json:"updated_at"`
}
```

`Installability` 可选值：

- `installable`: 可以一键安装。
- `config_required`: 需要用户补充环境变量或参数后安装。
- `manual`: 只有展示信息，不开放一键安装。
- `unsupported`: 当前 Gateway 暂不支持该安装方式。

### 5.3 MarketSourceRef

```go
type MarketSourceRef struct {
    SourceID   string                 `json:"source_id"`
    ExternalID string                 `json:"external_id"`
    URL        string                 `json:"url,omitempty"`
    Version    string                 `json:"version,omitempty"`
    UpdatedAt  time.Time              `json:"updated_at,omitempty"`
    Meta       map[string]interface{} `json:"meta,omitempty"`
}
```

### 5.4 MarketInstallOption

```go
type MarketInstallOption struct {
    Type        string                 `json:"type"`
    Command     string                 `json:"command,omitempty"`
    Args        []string               `json:"args,omitempty"`
    Env         map[string]string      `json:"env,omitempty"`
    URL         string                 `json:"url,omitempty"`
    Transport   string                 `json:"transport,omitempty"`
    Image       string                 `json:"image,omitempty"`
    PackageName string                 `json:"package_name,omitempty"`
    RequiredEnv []MarketEnvVarSpec     `json:"required_env,omitempty"`
    SourceID    string                 `json:"source_id"`
    Confidence  string                 `json:"confidence"`
    Raw         map[string]interface{} `json:"raw,omitempty"`
}
```

`Type` 可选值：

- `remote`
- `npx`
- `uvx`
- `docker`
- `command`
- `manual`

`Confidence` 可选值：

- `high`: 标准 Registry、Docker、Smithery verified/hosted 等结构化安装源。
- `medium`: 第三方 API 返回了 command/env，但未验证。
- `low`: 信息不完整，只能生成候选配置。

## 6. 数据源 Adapter

### 6.1 Adapter 接口

```go
type MarketSourceAdapter interface {
    SourceID() string
    Kind() string
    FetchPage(ctx context.Context, req MarketFetchRequest) (MarketFetchPage, error)
    FetchDetail(ctx context.Context, externalID string) (*NormalizedMarketPackage, error)
}

type MarketFetchRequest struct {
    Cursor       string
    UpdatedSince *time.Time
    Limit        int
}

type MarketFetchPage struct {
    Items      []NormalizedMarketPackage
    NextCursor string
    HasMore    bool
}
```

### 6.2 Official Registry Adapter

接口：

- `GET /v0.1/servers?limit=&cursor=&updated_since=`
- `GET /v0.1/servers/{serverName}/versions`
- `GET /v0.1/servers/{serverName}/versions/{version}`

映射：

- `server.name` -> `CanonicalName`
- `server.title` -> `Title`
- `server.description` -> `Description`
- `server.version` -> `Version`
- `server.packages` -> `InstallOptions`
- `server.remotes` -> `InstallOptions`
- `_meta.io.modelcontextprotocol.registry/official.status` -> source status

安装规则：

- `remotes[].type=streamable-http|sse` -> `Type=remote`
- `packages[].registryType=npm` -> `Type=npx`
- `packages[].registryType=pypi` -> `Type=uvx`
- 无 `packages` 且无 `remotes` -> `manual`

### 6.3 Smithery Adapter

接口：

- `GET https://api.smithery.ai/servers`
- 详情接口按 Smithery OpenAPI 后续接入。

映射：

- `qualifiedName` -> `CanonicalName`
- `displayName` -> `Title`
- `description` -> `Description`
- `verified` -> `Verified`
- `useCount` -> `UseCount`
- `remote/isDeployed/bySmithery` -> install/source metadata

安装规则：

- `remote=true` 且有 MCP endpoint -> `Type=remote`
- hosted/deployed 且有结构化连接信息 -> `Type=remote`
- 仅有详情页或 homepage -> `manual`

### 6.4 Glama Adapter

接口：

- `GET https://glama.ai/api/mcp/v1/servers?first=&after=`

映射：

- `name` -> `Name`
- `namespace/name` -> `CanonicalName`
- `description` -> `Description`
- `repository.url` -> `Repository`
- `spdxLicense.name` -> `License`
- `environmentVariablesJsonSchema` -> `EnvSchema`
- `tools` -> `Tools`

安装规则：

- 有 remote connector URL -> `Type=remote`
- 有 install command/config -> `Type=command|npx|uvx`
- 只有 repository/env schema -> `config_required` 或 `manual`

### 6.5 PulseMCP Adapter

接口：

- 采用 PulseMCP 公开 API。
- 如果其实现 Registry-compatible API，优先按 Official Registry Adapter 解析。

映射：

- 标准 Registry 字段直接复用。
- 额外热度、分类、安全评分进入 `RawMeta` 或 source-specific metadata。

安装规则：

- 标准 `packages/remotes` 可一键安装。
- 仅补充 metadata 时不影响现有安装选项。

### 6.6 Docker Catalog Adapter

接口：

- 仅接入 Docker 提供的机器可读 catalog/API。
- 如果没有稳定 API，先不纳入第一阶段。

映射：

- image/name -> `InstallOptions.Type=docker`
- verified/publisher -> `Verified`
- description/categories -> 展示字段

安装规则：

- Docker 镜像完整且支持当前 Gateway runtime -> `installable`
- 需要 secret/env -> `config_required`

## 7. 去重与合并

### 7.1 去重键

按以下优先级生成 `canonical_key`：

1. Official Registry `server.name`
2. repository URL 规范化后 hash
3. package registry + package identifier
4. source kind + external ID

Repository URL 需要规范化：

- 去掉 `.git`
- GitHub URL 统一成 `https://github.com/{owner}/{repo}`
- owner/repo 全部转小写
- 去掉 query/hash

### 7.2 合并策略

| 字段 | 合并规则 |
|---|---|
| name/title | 优先 Official Registry，其次 verified source，其次 priority 高的 source |
| description | 选择最长且非空；Official Registry 可覆盖 |
| version | 每个 source 保留，主版本选择 Official latest |
| repository | 选择出现次数最多或 trusted source 提供的 |
| tags/category | 合并去重 |
| verified | 任一 trusted source verified 即 true |
| downloads/useCount/rating | 按 source 分别保留，展示时可选择最高可信 source |
| install_options | 全部保留，按 confidence 排序 |
| tools/env_schema | 合并去重，冲突时保留 source metadata |

### 7.3 安装选项排序

排序规则：

1. `docker` + verified
2. `remote` + trusted source
3. Official Registry `packages`
4. Smithery hosted/remote
5. Glama/PulseMCP structured command
6. manual

## 8. 同步流程

### 8.1 定时同步

```text
Scheduler
  └── enabled sources
        └── adapter.FetchPage
              └── normalize
                    └── upsert source package refs
                          └── dedupe merge
                                └── update sync checkpoint
```

建议：

- 默认每 1 小时同步一次。
- 支持手动同步单个 source。
- 每个 source 独立 checkpoint。
- 同步失败只更新 source status，不删除已有缓存。

### 8.2 增量同步

优先使用：

- cursor pagination
- `updated_since`
- ETag / Last-Modified

如果数据源不支持增量：

- 分页全量拉取。
- 按 `external_id + updated_at/version` 判断是否变化。
- 控制同步频率和超时时间。

### 8.3 同步状态

```go
type MarketSyncJob struct {
    ID           string
    SourceID     string
    Status       string
    Cursor       string
    StartedAt    time.Time
    FinishedAt   *time.Time
    ItemsFetched int
    ItemsUpserted int
    ErrorMessage string
}
```

状态：

- `pending`
- `running`
- `success`
- `failed`
- `partial`

## 9. API 设计

### 9.1 市场源列表

```http
GET /api/v1/market/sources
```

返回：

```json
{
  "items": [
    {
      "id": "official",
      "name": "Official MCP Registry",
      "kind": "official_registry",
      "url": "https://registry.modelcontextprotocol.io",
      "trusted": true,
      "enabled": true,
      "priority": 1,
      "status": "healthy",
      "last_synced": "2026-05-30T08:00:00Z",
      "total_items": 1200
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

### 9.2 添加市场源

```http
POST /api/v1/market/sources
```

```json
{
  "id": "company-registry",
  "name": "Company MCP Registry",
  "kind": "custom_registry",
  "url": "https://mcp.company.com",
  "trusted": true,
  "enabled": true,
  "priority": 10
}
```

第一阶段可以只允许系统内置源，后续再开放 UI 添加。

### 9.3 手动同步

```http
POST /api/v1/market/sources/:id/sync
```

返回：

```json
{
  "sync_id": "sync_123",
  "source_id": "official",
  "status": "running",
  "started_at": "2026-05-30T08:00:00Z"
}
```

### 9.4 查询市场包

```http
GET /api/v1/market/packages?q=&source=&category=&installability=&verified_only=&page=&page_size=&sort=
```

`sort` 可选：

- `relevance`
- `updated`
- `popularity`
- `verified`
- `source_priority`

返回：

```json
{
  "items": [
    {
      "id": "pkg_abc",
      "canonical_name": "io.modelcontextprotocol/filesystem",
      "name": "filesystem",
      "title": "Filesystem",
      "description": "File system operations for MCP",
      "version": "1.2.0",
      "verified": true,
      "category": "系统",
      "tags": ["filesystem", "files"],
      "repository": "https://github.com/modelcontextprotocol/servers",
      "installability": "config_required",
      "install_options": [
        {
          "type": "npx",
          "command": "npx",
          "args": ["-y", "@modelcontextprotocol/server-filesystem"],
          "source_id": "official",
          "confidence": "high"
        }
      ],
      "source_refs": [
        {
          "source_id": "official",
          "external_id": "io.modelcontextprotocol/filesystem"
        },
        {
          "source_id": "smithery",
          "external_id": "modelcontextprotocol/filesystem"
        }
      ]
    }
  ],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

### 9.5 市场包详情

```http
GET /api/v1/market/packages/:id
```

详情比列表多返回：

- `tools`
- `env_schema`
- `readme`
- `source_refs`
- `raw_meta` 的安全子集
- 完整 `install_options`

### 9.6 安装市场包

建议新增独立安装接口，而不是让前端直接拼 `createService`：

```http
POST /api/v1/market/packages/:id/install
```

请求：

```json
{
  "workspace_id": "default",
  "service_name": "filesystem",
  "install_option_index": 0,
  "env": {
    "FILESYSTEM_ROOT": "/workspace"
  }
}
```

后端职责：

- 校验 Workspace 权限。
- 校验 install option 是否可安装。
- 合并默认 env 和用户 env。
- 生成 `config.MCPServerConfig`。
- 调用现有 `DeployServer` 或 workspace service 创建流程。
- 记录安装来源、版本、source ref。

## 10. 前端调整

### 10.1 Market 页面

新增能力：

- source 筛选。
- installability 筛选。
- verified only 开关。
- 安装方式 badge：`Remote`、`Docker`、`npx`、`uvx`、`Manual`。
- source badges：Official、Smithery、Glama、PulseMCP。
- 同步状态提示。

卡片展示字段：

- 标题、描述、分类、标签。
- verified 标识。
- 来源数量。
- 安装状态。
- 安装方式。
- popularity 指标。

### 10.2 包详情弹窗

新增：

- 来源列表。
- 安装方式选择。
- 必填环境变量表单。
- 工具列表。
- repository/homepage 链接。
- 安装不可用原因。

### 10.3 市场源管理

第一阶段可以只读展示：

- source name
- status
- last synced
- total packages
- sync button

第二阶段再支持新增/编辑 custom registry。

## 11. 安全设计

### 11.1 外部 API 调用

- 设置超时：默认 10s。
- 限制响应大小。
- 只允许 HTTPS，内网私有源可通过显式配置放开。
- 禁止跟随到内网地址，防止 SSRF。
- API token 只存在后端配置，不返回前端。

### 11.2 安装安全

- 未验证源默认不自动安装，除非用户显式确认。
- `manual` 和 `unsupported` 不允许一键安装。
- command/args 不允许 shell 拼接执行，只保存结构化数组。
- env secret 不在响应里返回明文。
- Docker 镜像优先使用固定 tag，避免隐式 latest。

### 11.3 数据可信度

市场展示需要区分：

- source trusted
- package verified
- install option confidence
- remote server vs local command

这些状态不能混成一个“已验证”标识。

## 12. 存储设计

第一阶段可使用项目现有持久化方式；如果没有数据库表迁移体系，可以先用 JSON 文件缓存，后续迁到数据库。

推荐表：

- `market_sources`
- `market_packages`
- `market_package_sources`
- `market_sync_jobs`

### 12.1 market_sources

| 字段 | 类型 |
|---|---|
| id | string |
| name | string |
| kind | string |
| url | string |
| trusted | bool |
| enabled | bool |
| priority | int |
| status | string |
| last_synced_at | timestamp |
| last_error | text |
| sync_cursor | string |
| created_at | timestamp |
| updated_at | timestamp |

### 12.2 market_packages

| 字段 | 类型 |
|---|---|
| id | string |
| canonical_key | string |
| canonical_name | string |
| name | string |
| title | string |
| description | text |
| version | string |
| category | string |
| tags_json | json |
| repository | string |
| homepage | string |
| license | string |
| verified | bool |
| installability | string |
| install_options_json | json |
| tools_json | json |
| env_schema_json | json |
| raw_meta_json | json |
| created_at | timestamp |
| updated_at | timestamp |

### 12.3 market_package_sources

| 字段 | 类型 |
|---|---|
| id | string |
| package_id | string |
| source_id | string |
| external_id | string |
| external_url | string |
| version | string |
| source_updated_at | timestamp |
| raw_json | json |
| created_at | timestamp |
| updated_at | timestamp |

## 13. 分阶段实施

### Phase 1: API 多源基础

- 抽象 `MarketSourceAdapter`。
- 接入 Official MCP Registry。
- 接入 Smithery list API。
- 新增市场数据缓存/store。
- `/api/v1/market/packages` 改为读取 store。
- 保留静态列表作为 fallback。

### Phase 2: 安装链路

- 实现 install resolver。
- 新增 `/api/v1/market/packages/:id/install`。
- 前端支持安装方式选择和 env 表单。
- 安装记录保留 source ref 和 version。

### Phase 3: 更多 API 源

- 接入 Glama。
- 接入 PulseMCP。
- 如果 Docker Catalog 有稳定 API，接入 Docker。
- 完善去重、合并、排序。

### Phase 4: 市场源管理

- 市场源启停。
- 手动同步。
- 同步历史。
- 自定义 Registry API 源。

## 14. 与现有代码的关系

### 后端

当前：

- `/api/v1/market/sources` 返回 `defaultMarketSources`
- `/api/v1/market/packages` 返回 `defaultMarketPackages`
- `packageConfigFromMarket` 只支持当前静态安装配置

调整：

- 保留现有 API path，不破坏前端调用。
- `defaultMarketPackages` 作为 fallback seed。
- 新增 market store 和 sync service。
- `handleV1MarketPackages` 从 store 查询。
- `handleV1MarketPackageDetail` 从 store 查询详情。
- `handleV1CreateService` 中的 `market_package_id` 逻辑逐步迁移到 install API。

### 前端

当前：

- Market 页面本地搜索和分类筛选。
- 点击卡片弹窗，直接调用 `gatewayApi.createService`。

调整：

- 查询参数下推后端。
- 增加 source/installability/verified 筛选。
- 安装调用新 install API。
- 没有可安装选项时展示“查看配置”或“暂不支持安装”。

## 15. 开放问题

- 项目最终使用 JSON 文件、SQLite、MongoDB 还是现有 persistence 层存储市场缓存？
- Docker MCP Catalog 是否存在稳定公开 API，需要确认后决定是否进入 Phase 1。
- Smithery 详情和安装 endpoint 是否需要 API key？如果需要，是否允许用户配置 source credential？
- 多租户模式下市场源是全局配置还是 workspace/account 级配置？
- 安装远程 MCP 时，Gateway 当前 runtime 对 streamable-http 和 SSE 的支持范围需要再核对。

## 16. 推荐首版范围

首版建议只做：

- Official MCP Registry
- Smithery
- 后端缓存
- 基础去重
- 市场列表 source filter
- 详情页 install options
- 可安装项一键安装
- 不可安装项只展示详情

这样能尽快把静态市场替换成真实 API 数据，同时避免爬虫、复杂评分和私有源管理带来的不确定性。
