package router

import (
	"encoding/json"
	"os"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

// ServerManager 管理所有运行的服务
type ServerManager struct {
	sync.RWMutex
	mcpServiceMgr service.ServiceManagerI
	cfg           config.Config
}

// NewServerManager 初始化服务管理器
func NewServerManager(cfg config.Config, e *echo.Echo) *ServerManager {
	portMgr := service.NewPortManager()
	mcpServiceMgr := service.NewServiceMgr(cfg, portMgr)
	m := &ServerManager{
		mcpServiceMgr: mcpServiceMgr,
		cfg:           cfg,
	}

	// 注册路由
	e.POST("/deploy", m.handleDeploy)                         // 部署服务
	e.DELETE("/delete", m.handleDeleteMcpService)             // 删除服务
	e.GET("/sse", m.handleGlobalSSE)                          // 全局SSE WIP
	e.POST("/message", m.handleGlobalMessage)                 // 全局消息 WIP
	e.GET("/services", m.handleGetAllServices)                // 获取所有服务
	e.GET("/services/:name/health", m.handleGetServiceHealth) // 获取服务健康状态

	// API 路由
	api := e.Group("/api")

	// Workspace 管理
	api.GET("/workspaces", m.handleGetAllWorkspaces)
	api.POST("/workspaces", m.handleCreateWorkspace)
	api.DELETE("/workspaces/:id", m.handleDeleteWorkspace)
	api.GET("/workspaces/:id/services", m.handleGetWorkspaceServices)

	// Session 管理
	api.GET("/workspaces/:workspace/sessions", m.handleGetWorkspaceSessions)
	api.POST("/workspaces/:workspace/sessions", m.handleCreateSession)
	api.DELETE("/workspaces/:workspace/sessions/:id", m.handleDeleteSession)
	api.GET("/sessions/:id/status", m.handleGetSessionStatus)

	// 增强的服务管理
	api.POST("/workspaces/:workspace/services", m.handleDeployServiceToWorkspace)
	api.PUT("/workspaces/:workspace/services/:name", m.handleUpdateServiceConfig)
	api.POST("/workspaces/:workspace/services/:name/restart", m.handleRestartService)
	api.POST("/workspaces/:workspace/services/:name/stop", m.handleStopService)
	api.POST("/workspaces/:workspace/services/:name/start", m.handleStartService)
	api.DELETE("/workspaces/:workspace/services/:name", m.handleDeleteServiceFromWorkspace)
	api.GET("/workspaces/:workspace/services/:name/logs", m.handleGetServiceLogs)

	// 调试功能路由
	m.setupDebugRoutes(api)

	// 静态文件服务 (前端管理界面)
	e.Static("/admin", "web/dist")

	// 代理
	e.Any("/*", m.proxyHandler())
	m.loadConfig()
	return m
}
func (m *ServerManager) loadConfig() error {
	xl := xlog.NewLogger("[ServerManager]")
	data, err := os.ReadFile(m.cfg.GetMcpConfigPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var config map[string]config.MCPServerConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	xl.Infof("Async Loading %d servers", len(config))
	go func() {
		for name, srv := range config {
			xl.Infof("Loading server %s: %v", name, srv)
			if _, err := m.DeployServer(name, srv); err != nil {
				xl.Errorf("Error deploying server %s: %v", name, err)
			}
		}
		xl.Infof("Loaded %d servers", len(config))
	}()

	return nil
}

func (m *ServerManager) Close() {
	m.mcpServiceMgr.Close()
}
