package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/types"
)

// GET ALL MCP SERVICES
func (m *ServerManager) handleGetAllServices(c echo.Context) error {
	c.Logger().Infof("Get all services")
	mcpServices := m.mcpServiceMgr.GetMcpServices(c.Logger(), service.NameArg{
		Workspace: service.DefaultWorkspace,
	})
	var serviceInfos []service.McpServiceInfo
	for _, instance := range mcpServices {
		serviceInfos = append(serviceInfos, instance.Info())
	}
	return c.JSON(http.StatusOK, serviceInfos)
}

// DeployServer 部署单个服务
func (m *ServerManager) DeployServer(logger echo.Logger, name string, config config.MCPServerConfig) error {
	m.Lock()
	defer m.Unlock()

	if config.Command == "" && config.URL == "" {
		return fmt.Errorf("服务配置必须包含 URL 或 Command")
	}

	if config.Command != "" && config.URL != "" {
		return fmt.Errorf("服务配置不能同时包含 URL 和 Command")
	}

	return m.mcpServiceMgr.DeployServer(logger, service.NameArg{
		Server: name,
	}, config)
}

// handleDeploy 处理部署请求
func (m *ServerManager) handleDeploy(c echo.Context) error {
	c.Logger().Infof("Deploy request: %v", c.Request().Body)
	var req types.DeployRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	c.Logger().Infof("Deploy request: %v", req)

	for name, config := range req.MCPServers {
		c.Logger().Infof("Deploying %s: %v", name, config)
		if err := m.DeployServer(c.Logger(), name, config); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": fmt.Sprintf("Failed to deploy %s: %v", name, err),
			})
		}
	}

	c.Logger().Infof("Deployed all servers")

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleDeleteMcpService 删除单个服务
func (m *ServerManager) handleDeleteMcpService(c echo.Context) error {
	c.Logger().Infof("Delete request: %v", c.Request().Body)
	name := c.QueryParam("name")
	if err := m.mcpServiceMgr.DeleteServer(c.Logger(), service.NameArg{
		Server: name,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}
