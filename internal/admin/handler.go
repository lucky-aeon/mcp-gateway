package admin

import (
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/identity"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// Handler 承载所有管理面 API（部署/生命周期/会话/调试）。
// mu 用于在并发 DeployServer 之间串行化部署动作，
// 避免同一时间多个请求对同一个 workspace 状态机造成抖动。
type Handler struct {
	services workspaces.ServiceManagerI
	cfg      *config.Config
	auth     *identity.Service
	state    *controlPlaneState
	mu       sync.RWMutex
}

// NewHandler 构造一个 admin Handler。
func NewHandler(services workspaces.ServiceManagerI, cfg *config.Config, auth *identity.Service) *Handler {
	return &Handler{
		services: services,
		cfg:      cfg,
		auth:     auth,
		state:    newControlPlaneState(),
	}
}

// Register 挂载所有管理 API 到 Echo：
//   - /deploy, /delete, /services, /services/:name/health
//   - /api/workspaces/..., /api/workspaces/:w/services/..., /api/workspaces/:w/sessions/...
//   - /api/sessions/:id/status
//   - debug 相关路由（通过 setupDebugRoutes）
func (h *Handler) Register(e *echo.Echo) {
	protected := e.Group("")
	protected.Use(h.v1AuthMiddleware)

	protected.POST("/deploy", h.handleDeploy)
	protected.DELETE("/delete", h.handleDeleteMcpService)
	protected.GET("/services", h.handleGetAllServices)
	protected.GET("/services/:name/health", h.handleGetServiceHealth)

	api := protected.Group("/api")
	api.GET("/workspaces", h.handleGetAllWorkspaces)
	api.POST("/workspaces", h.handleCreateWorkspace)
	api.DELETE("/workspaces/:id", h.handleDeleteWorkspace)
	api.GET("/workspaces/:id/services", h.handleGetWorkspaceServices)

	api.GET("/workspaces/:workspace/sessions", h.handleGetWorkspaceSessions)
	api.POST("/workspaces/:workspace/sessions", h.handleCreateSession)
	api.DELETE("/workspaces/:workspace/sessions/:id", h.handleDeleteSession)
	api.GET("/sessions/:id/status", h.handleGetSessionStatus)

	api.POST("/workspaces/:workspace/services", h.handleDeployServiceToWorkspace)
	api.PUT("/workspaces/:workspace/services/:name", h.handleUpdateServiceConfig)
	api.POST("/workspaces/:workspace/services/:name/restart", h.handleRestartService)
	api.POST("/workspaces/:workspace/services/:name/stop", h.handleStopService)
	api.POST("/workspaces/:workspace/services/:name/start", h.handleStartService)
	api.DELETE("/workspaces/:workspace/services/:name", h.handleDeleteServiceFromWorkspace)
	api.GET("/workspaces/:workspace/services/:name/logs", h.handleGetServiceLogs)

	h.setupDebugRoutes(api)
	h.registerV1Routes(e)
}
