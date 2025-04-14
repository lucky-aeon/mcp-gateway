package router

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

// 全局MESSAGE，这里将POST请求转发到所有MCP服务
func (m *ServerManager) handleGlobalMessage(c echo.Context) error {
	xl := xlog.WithEchoLogger(c.Logger())
	xl.Infof("Global message: %v", c.Request().Body)
	sessionId := c.QueryParam("sessionId")
	if sessionId == "" {
		return c.String(http.StatusBadRequest, "missing sessionId")
	}

	// 获取session
	session, exists := m.mcpServiceMgr.GetProxySession(xl, sessionId)
	if !exists {
		return c.String(http.StatusNotFound, "session not found")
	}
	// 读取请求体
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	// 记录发送的消息
	session.SendMessage(xl, string(body))

	return c.String(http.StatusOK, "Accepted")
}
