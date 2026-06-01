package gateway

import (
	"io"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/httpx"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/oplog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// 全局MESSAGE，这里将POST请求转发到所有MCP服务
func (h *Handler) handleGlobalMessage(c echo.Context) error {
	xl := xlog.NewLogger("GLOBAL-MSG")
	xl.Infof("Global message: %v", c.Request().Body)
	sessionId, err := httpx.GetSession(c)
	if err != nil {
		workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)
		h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelError, "session.message_failed", workspace, "", "session message failed", err.Error(), nil)
		return c.String(http.StatusBadRequest, err.Error())
	}
	workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)
	// 获取session
	session, exists := h.services.GetProxySession(xl, workspaces.NameArg{
		Workspace: workspace,
		Session:   sessionId,
	})
	if !exists {
		h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelError, "session.message_failed", workspace, sessionId, "session message failed", "session not found", nil)
		return c.String(http.StatusNotFound, "session not found")
	}
	// 读取请求体
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelError, "session.message_failed", workspace, sessionId, "session message failed", err.Error(), nil)
		return err
	}
	info := rpcLogInfoFromBody(body, "")
	detail := rpcLogDetail(info, "sse-message")

	// 记录发送的消息
	if err := session.SendMessage(xl, []byte(body)); err != nil {
		h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelError, info.Action+"_failed", workspace, sessionId, info.Message+" failed", err.Error(), detail)
		return c.String(http.StatusBadGateway, err.Error())
	}
	h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelInfo, info.Action, workspace, sessionId, info.Message, "", detail)

	return c.String(http.StatusOK, "Accepted")
}
