package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/types"
	"github.com/lucky-aeon/agentx/plugin-helper/utils"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

// GET ALL MCP SERVICES
func (m *ServerManager) handleGetAllServices(c echo.Context) error {
	xl := xlog.NewLogger("GET-SERVICES")
	xl.Infof("Get all services")
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)
	mcpServices := m.mcpServiceMgr.GetMcpServices(xl, service.NameArg{
		Workspace: workspace,
	})
	var serviceInfos []service.McpServiceInfo
	for _, instance := range mcpServices {
		serviceInfos = append(serviceInfos, instance.Info())
	}
	return c.JSON(http.StatusOK, serviceInfos)
}

// DeployServer 部署单个服务
func (m *ServerManager) DeployServer(name string, config config.MCPServerConfig) error {
	m.Lock()
	defer m.Unlock()

	logger := xlog.NewLogger("DEPLOY")

	if config.Command == "" && config.URL == "" {
		return fmt.Errorf("服务配置必须包含 URL 或 Command")
	}

	if config.Command != "" && config.URL != "" {
		return fmt.Errorf("服务配置不能同时包含 URL 和 Command")
	}

	if config.Workspace == "" {
		config.Workspace = service.DefaultWorkspace
	}
	return m.mcpServiceMgr.DeployServer(logger, service.NameArg{
		Server:    name,
		Workspace: config.Workspace,
	}, config)
}

// handleDeploy 处理部署请求
func (m *ServerManager) handleDeploy(c echo.Context) error {
	xl := xlog.NewLogger("DEPLOY-REQ")
	xl.Infof("Deploy request: %v", c.Request().Body)
	var req types.DeployRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	xl.Infof("Deploy request: %v", req)
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)
	for name, config := range req.MCPServers {
		xl.Infof("Deploying %s: %v", name, config)
		if workspace != "" {
			config.Workspace = workspace
		} else if config.Workspace == "" {
			config.Workspace = service.DefaultWorkspace
		}
		if err := m.DeployServer(name, config); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to deploy %s: %v", name, err),
			})
		}
	}

	xl.Infof("Deployed all servers")

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleDeleteMcpService 删除单个服务
func (m *ServerManager) handleDeleteMcpService(c echo.Context) error {
	xl := xlog.NewLogger("DELETE-SVC")
	xl.Infof("Delete request: %v", c.Request().Body)
	name := c.QueryParam("name")
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)
	if err := m.mcpServiceMgr.DeleteServer(xl, service.NameArg{
		Server:    name,
		Workspace: workspace,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleGetServiceHealth 获取服务健康状态
func (m *ServerManager) handleGetServiceHealth(c echo.Context) error {
	xl := xlog.NewLogger("GET-SERVICE-HEALTH")
	serviceName := c.Param("name")
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)

	xl.Infof("Get service health for %s in workspace %s", serviceName, workspace)

	mcpService, err := m.mcpServiceMgr.GetMcpService(xl, service.NameArg{
		Server:    serviceName,
		Workspace: workspace,
	})
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": fmt.Sprintf("Service %s not found: %v", serviceName, err)})
	}

	health := mcpService.GetHealthStatus()
	return c.JSON(http.StatusOK, health)
}
