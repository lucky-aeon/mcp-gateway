package router

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/utils"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

func (m *ServerManager) handleStreamHTTP(c echo.Context) error {
	xl := xlog.NewLogger("STREAMHTTP")
	serviceName := c.Param("service")
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)

	instance, err := m.mcpServiceMgr.GetMcpService(xl, service.NameArg{
		Server:    serviceName,
		Workspace: workspace,
	})
	if err != nil {
		return c.String(http.StatusNotFound, "Service not found")
	}

	targetURL := instance.GetMessageUrl()
	if targetURL == "" {
		return c.String(http.StatusServiceUnavailable, "Service not available")
	}

	req, err := http.NewRequest(c.Request().Method, targetURL, c.Request().Body)
	if err != nil {
		return err
	}
	for k, v := range c.Request().Header {
		req.Header[k] = v
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		c.Response().Header()[k] = v
	}
	c.Response().WriteHeader(resp.StatusCode)
	_, err = io.Copy(c.Response().Writer, resp.Body)
	return err
}
