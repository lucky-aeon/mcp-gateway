// Package server 负责装配整个 MCP Gateway：构造 workspaces.ServiceManager、
// 分别挂载 gateway.Handler（MCP 流量）与 admin.Handler（管理 API），
// 并异步恢复持久化的 MCP 部署。
package server

import (
	"github.com/labstack/echo/v4"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/admin"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/gateway"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/persistence"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// Server 是整个网关的顶层聚合。
// 目前只持有 workspaces.ServiceManager 以便在关闭时统一终止所有 MCP 服务。
type Server struct {
	services workspaces.ServiceManagerI
}

// New 构造并返回一个 Server 实例，同时在给定的 Echo 上注册：
//   - 管理 API（admin.Handler）
//   - MCP 协议入口（gateway.Handler）
//   - /admin 静态资源（前端 dashboard）
//   - /* 单服务代理（必须最后注册）
//
// 另外异步触发一次从 mcp_servers.json 恢复 MCP 部署。
func New(cfg config.Config, e *echo.Echo) *Server {
	portMgr := runtime.NewPortManager()
	services := workspaces.NewServiceMgr(cfg, portMgr)

	adminH := admin.NewHandler(services)
	gatewayH := gateway.NewHandler(services, cfg)

	// 先注册精确匹配的路由
	adminH.Register(e)
	gatewayH.Register(e)

	// 静态托管前端 dashboard
	e.Static("/admin", "web/dist")

	// 通配路由 /* 必须最后注册（echo 的路由优先级要求）
	gatewayH.RegisterProxy(e)

	// 异步恢复 mcp_servers.json 中已部署的 MCP
	_ = persistence.LoadAndDeployServers(cfg, func(name string, mcpCfg config.MCPServerConfig) error {
		_, err := adminH.DeployServer(name, mcpCfg)
		return err
	})

	return &Server{services: services}
}

// Close 优雅关闭底层 service manager（会关闭所有 workspaces 及其 MCP 服务）。
func (s *Server) Close() {
	s.services.Close()
}
