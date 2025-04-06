package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
)

// GET ALL MCP SERVICES
func (m *ServerManager) handleGetAllServices(c echo.Context) error {
	c.Logger().Infof("Get all services")
	mcpServices := m.mcpServiceMgr.GetMcpServices(c.Logger())
	var serviceInfos []service.McpServiceInfo
	for _, instance := range mcpServices {
		serviceInfos = append(serviceInfos, instance.Info())
	}
	return c.JSON(http.StatusOK, serviceInfos)
}
