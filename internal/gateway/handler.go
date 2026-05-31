package gateway

import (
	"net/http"
	"sync"

	"github.com/labstack/echo/v4"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/identity"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// Handler 处理 MCP 协议流量（SSE / Streamable HTTP / 单服务代理）。
// 它本身不持有业务状态，所有调用都委托给底层 workspaces.ServiceManagerI。
type Handler struct {
	services  workspaces.ServiceManagerI
	cfg       *config.Config
	auth      *identity.Service
	oauth     *internalOAuthServer
	restoreMu sync.Mutex
}

// NewHandler 构造一个 gateway Handler。
func NewHandler(services workspaces.ServiceManagerI, cfg *config.Config, auth *identity.Service) *Handler {
	return &Handler{services: services, cfg: cfg, auth: auth, oauth: newInternalOAuthServer()}
}

// Register 向 Echo 注册 MCP 协议入口：
//   - Streamable HTTP: POST/GET/DELETE /stream, GET/POST /:service
//   - SSE:             GET /sse, POST /message
//
// proxyHandler 是 wildcard 路由（/*），需要通过 RegisterProxy 单独注册在所有其它路由之后。
func (h *Handler) Register(e *echo.Echo) {
	auth := h.mcpAuthMiddleware
	e.GET("/.well-known/oauth-protected-resource", h.handleProtectedResourceMetadata)
	e.GET("/.well-known/oauth-protected-resource/*", h.handleProtectedResourceMetadata)
	e.GET(authorizationServerMetadataPath, h.handleAuthorizationServerMetadata)
	e.GET("/oauth/authorize", h.handleOAuthAuthorize)
	e.POST("/oauth/authorize", h.handleOAuthAuthorize)
	e.POST("/oauth/register", h.handleOAuthRegister)
	e.POST("/oauth/token", h.handleOAuthToken)

	e.GET("/sse", auth(h.requireSSE(h.handleGlobalSSE)))
	e.POST("/message", auth(h.requireSSE(h.handleGlobalMessage)))
	e.POST("/stream", auth(h.requireStreamHTTP(h.handleGlobalStreamHTTP)))
	e.GET("/stream", auth(h.requireStreamHTTP(h.handleGlobalStreamHTTP)))
	e.DELETE("/stream", auth(h.requireStreamHTTP(h.handleGlobalStreamHTTP)))
	e.GET("/:service", auth(h.requireStreamHTTPOrProxy(h.handleStreamHTTP)))
	e.POST("/:service", auth(h.requireStreamHTTPOrProxy(h.handleStreamHTTP)))
}

// RegisterProxy 注册通配 /* 代理路由，必须在所有其它路由之后调用。
func (h *Handler) RegisterProxy(e *echo.Echo) {
	auth := h.mcpAuthMiddleware
	e.Any("/*", auth(h.proxyHandler()))
}

func (h *Handler) requireSSE(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if h.cfg == nil || !h.cfg.SupportsSSE() {
			return echo.NewHTTPError(http.StatusNotFound)
		}
		return next(c)
	}
}

func (h *Handler) requireStreamHTTP(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if h.cfg == nil || !h.cfg.SupportsStreamHTTP() {
			return echo.NewHTTPError(http.StatusNotFound)
		}
		return next(c)
	}
}

func (h *Handler) requireStreamHTTPOrProxy(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if h.cfg == nil || !h.cfg.SupportsStreamHTTP() {
			return h.proxyHandler()(c)
		}
		return next(c)
	}
}
