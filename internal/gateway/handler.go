package gateway

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/identity"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// Handler 处理 MCP 协议流量（SSE / Streamable HTTP / 单服务代理）。
// 它本身不持有业务状态，所有调用都委托给底层 workspaces.ServiceManagerI。
type Handler struct {
	services workspaces.ServiceManagerI
	cfg      config.Config
	auth     *identity.Service
}

// NewHandler 构造一个 gateway Handler。
func NewHandler(services workspaces.ServiceManagerI, cfg config.Config, auth *identity.Service) *Handler {
	return &Handler{services: services, cfg: cfg, auth: auth}
}

// Register 向 Echo 注册 MCP 协议入口：
//   - Streamable HTTP 模式: POST/GET/DELETE /stream, GET/POST /:service
//   - SSE 模式:           GET /sse, POST /message
//
// proxyHandler 是 wildcard 路由（/*），需要通过 RegisterProxy 单独注册在所有其它路由之后。
func (h *Handler) Register(e *echo.Echo) {
	auth := middleware.KeyAuthWithConfig(identity.NewAuthMiddleware(&h.cfg, h.auth).GetKeyAuthConfig())
	if h.cfg.IsStreamHTTP() {
		e.GET("/:service", auth(h.handleStreamHTTP))
		e.POST("/:service", auth(h.handleStreamHTTP))
		e.POST("/stream", auth(h.handleGlobalStreamHTTP))
		e.GET("/stream", auth(h.handleGlobalStreamHTTP))
		e.DELETE("/stream", auth(h.handleGlobalStreamHTTP))
	} else {
		e.GET("/sse", auth(h.handleGlobalSSE))
		e.POST("/message", auth(h.handleGlobalMessage))
	}
}

// RegisterProxy 注册通配 /* 代理路由，必须在所有其它路由之后调用。
func (h *Handler) RegisterProxy(e *echo.Echo) {
	auth := middleware.KeyAuthWithConfig(identity.NewAuthMiddleware(&h.cfg, h.auth).GetKeyAuthConfig())
	e.Any("/*", auth(h.proxyHandler()))
}
