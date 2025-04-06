package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// 全局SSE，这里返回所有MCP服务的SSE事件
func (m *ServerManager) handleGlobalSSE(c echo.Context) error {
	c.Logger().Infof("Global SSE request: %v", c.Request().Body)
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	// create proxy session
	session := m.mcpServiceMgr.CreateProxySession()
	defer m.mcpServiceMgr.CloseProxySession(session.Id)

	// 返回endpoint事件
	c.Response().WriteHeader(http.StatusOK)
	fmt.Fprintf(c.Response(), "event: endpoint\ndata: /message?sessionId=%s\n\n", session.Id)
	c.Response().Flush()

	// 转发所有SSE事件
	for {
		select {
		case <-c.Request().Context().Done():
			return nil
		case event := <-session.GetEventChan():
			if _, err := fmt.Fprintf(c.Response(), "%s", event); err != nil {
				return err
			}
			c.Response().Flush()
		}
	}
}
