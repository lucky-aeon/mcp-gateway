package router

import (
	"fmt"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/types"
)

// 全局MESSAGE，这里将POST请求转发到所有MCP服务
func (m *ServerManager) handleGlobalMessage(c echo.Context) error {
	c.Logger().Infof("Global message: %v", c.Request().Body)
	sessionId := c.QueryParam("sessionId")
	if sessionId == "" {
		return c.String(http.StatusBadRequest, "missing sessionId")
	}

	// 获取session
	session, exists := m.mcpServiceMgr.GetProxySession(sessionId)
	if !exists {
		return c.String(http.StatusNotFound, "session not found")
	}

	// 读取请求体
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	// 转发消息
	c.Logger().Infof("Global message from session %s: %s", session.Id, string(body))
	mcpServices := m.mcpServiceMgr.GetMcpServices(c.Logger())
	for name, instance := range mcpServices {
		c.Logger().Infof("Forwarding message to %s", name)
		// 记录发送的消息
		session.AddMessage(name, string(body), "send")
		if err := instance.SendMessage(string(body)); err != nil {
			c.Logger().Errorf("Failed to forward message to %s: %v", name, err)
		}
	}

	return nil
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

	return m.mcpServiceMgr.DeployServer(logger, name, config)
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
	if err := m.mcpServiceMgr.DeleteServer(c.Logger(), name); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}
