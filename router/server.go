package router

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
)

// ServerManager 管理所有运行的服务
type ServerManager struct {
	sync.RWMutex
	mcpServiceMgr service.ServiceManagerI
	cfg           config.Config
}

var manager *ServerManager

// NewServerManager 初始化服务管理器
func NewServerManager(cfg config.Config, e *echo.Echo) *ServerManager {
	m := &ServerManager{
		mcpServiceMgr: service.NewServiceManager(cfg),
		cfg:           cfg,
	}

	// 注册路由
	e.POST("/deploy", manager.handleDeploy)             // 部署服务
	e.DELETE("/delete", manager.handleDeleteMcpService) // 删除服务
	e.GET("/sse", manager.handleGlobalSSE)              // 全局SSE WIP
	e.POST("/message", manager.handleGlobalMessage)     // 全局消息 WIP
	e.GET("/services", manager.handleGetAllServices)    // 获取所有服务

	// 代理
	e.Any("/*", proxyHandler())
	m.loadConfig()
	return m
}
func (m *ServerManager) loadConfig() error {
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
		fmt.Printf("Loading server %s: %v\n", name, srv)
		if err := m.DeployServer(echo.New().Logger, name, srv); err != nil {
			fmt.Printf("Error deploying server %s: %v\n", name, err)
		}
	}
	fmt.Printf("Loaded %d servers\n", len(config))
	return nil
}
