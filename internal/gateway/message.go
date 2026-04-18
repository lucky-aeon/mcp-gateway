package gateway

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/httpx"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// 全局MESSAGE，这里将POST请求转发到所有MCP服务
func (h *Handler) handleGlobalMessage(c echo.Context) error {
	xl := xlog.NewLogger("GLOBAL-MSG")
	xl.Infof("Global message: %v", c.Request().Body)
	sessionId, err := httpx.GetSession(c)
	if err != nil {
		return c.String(http.StatusBadRequest, err.Error())
	}
	workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)
	// 获取session
	session, exists := h.services.GetProxySession(xl, workspaces.NameArg{
		Workspace: workspace,
		Session:   sessionId,
	})
	if !exists {
		return c.String(http.StatusNotFound, "session not found")
	}
	// 读取请求体
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return err
	}

	// 记录发送的消息
	session.SendMessage(xl, []byte(body))

	return c.String(http.StatusOK, "Accepted")
}
