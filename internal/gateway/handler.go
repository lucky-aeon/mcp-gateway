package gateway

import (
	"github.com/labstack/echo/v4"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// Handler 处理 MCP 协议流量（SSE / Streamable HTTP / 单服务代理）。
// 它本身不持有业务状态，所有调用都委托给底层 workspaces.ServiceManagerI。
type Handler struct {
	services workspaces.ServiceManagerI
	cfg      config.Config
}

// NewHandler 构造一个 gateway Handler。
func NewHandler(services workspaces.ServiceManagerI, cfg config.Config) *Handler {
	return &Handler{services: services, cfg: cfg}
}

// Register 向 Echo 注册 MCP 协议入口：
//   - Streamable HTTP 模式: POST/GET/DELETE /stream, GET/POST /:service
//   - SSE 模式:           GET /sse, POST /message
//
// proxyHandler 是 wildcard 路由（/*），需要通过 RegisterProxy 单独注册在所有其它路由之后。
func (h *Handler) Register(e *echo.Echo) {
	if h.cfg.IsStreamHTTP() {
		e.GET("/:service", h.handleStreamHTTP)
		e.POST("/:service", h.handleStreamHTTP)
		e.POST("/stream", h.handleGlobalStreamHTTP)
		e.GET("/stream", h.handleGlobalStreamHTTP)
		e.DELETE("/stream", h.handleGlobalStreamHTTP)
	} else {
		e.GET("/sse", h.handleGlobalSSE)
		e.POST("/message", h.handleGlobalMessage)
	}
}

// RegisterProxy 注册通配 /* 代理路由，必须在所有其它路由之后调用。
func (h *Handler) RegisterProxy(e *echo.Echo) {
	e.Any("/*", h.proxyHandler())
}
