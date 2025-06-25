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
	e.POST("/deploy", m.handleDeploy)             // 部署服务
	e.DELETE("/delete", m.handleDeleteMcpService) // 删除服务
	e.GET("/sse", m.handleGlobalSSE)              // 全局SSE WIP
	e.POST("/message", m.handleGlobalMessage)     // 全局消息 WIP
	e.GET("/services", m.handleGetAllServices)    // 获取所有服务

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

	for name, srv := range config {
		xl.Infof("Loading server %s: %v", name, srv)
		if err := m.DeployServer(xl, name, srv); err != nil {
			xl.Errorf("Error deploying server %s: %v", name, err)
		}
	}
	xl.Infof("Loaded %d servers", len(config))
	return nil
}

func (m *ServerManager) Close() {
	m.mcpServiceMgr.Close()
}
