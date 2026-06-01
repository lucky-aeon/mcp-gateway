package gateway

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/httpx"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/oplog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// 全局SSE，这里返回所有MCP服务的SSE事件
func (h *Handler) handleGlobalSSE(c echo.Context) error {
	xl := xlog.NewLogger("GLOBAL-SSE")
	xl.Infof("Global SSE request: %v", c.Request().Body)
	querySessionId, err := httpx.GetSession(c)
	if err != nil {
		xl.Warnf("Get session error: %v", err)
	}
	workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)
	if querySessionId == "" {
		xl.Infof("No session ID provided, creating new session")
		if err := h.ensureWorkspaceServicesRunning(c.Request().Context(), workspace, xl); err != nil {
			xl.Errorf("restore workspace services failed: %v", err)
			h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelError, "session.create_failed", workspace, "", "session create failed", err.Error(), nil)
			return c.String(http.StatusInternalServerError, err.Error())
		}
		// 没有sessionId，生成一个返回出
		// create proxy session
		session, err := h.services.CreateProxySession(xl, workspaces.NameArg{
			Workspace: workspace,
			Session:   querySessionId,
		})
		if err != nil {
			h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelError, "session.create_failed", workspace, "", "session create failed", err.Error(), nil)
			return c.String(http.StatusInternalServerError, err.Error())
		}
		xl.Infof("Created new session: %s", session.Id)
		h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelInfo, "session.connect", workspace, session.Id, "SSE session connected", "", map[string]interface{}{"transport": "sse", "connection": "created"})
		// 302重定向到 /sse?sessionId={session.Id}
		if workspace != "" {
			return c.Redirect(http.StatusFound, fmt.Sprintf("/sse?sessionId=%s&workspaceId=%s", session.Id, workspace))
		}
		return c.Redirect(http.StatusFound, fmt.Sprintf("/sse?sessionId=%s", session.Id))
	}
	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")

	// get session by sessionId
	session, exists := h.services.GetProxySession(xl, workspaces.NameArg{
		Workspace: workspace,
		Session:   querySessionId,
	})
	if !exists {
		h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelError, "session.stream_failed", workspace, querySessionId, "session stream failed", "session not found", map[string]interface{}{"transport": "sse"})
		return c.String(http.StatusNotFound, "session not found")
	}
	h.appendOperation(c.Request().Context(), gatewayPrincipal(c), oplog.LevelInfo, "session.connect", workspace, querySessionId, "SSE session connected", "", map[string]interface{}{"transport": "sse", "connection": "attached"})

	// 返回endpoint事件
	c.Response().WriteHeader(http.StatusOK)
	w := c.Response().Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return c.String(http.StatusInternalServerError, "flusher not supported")
	}

	if workspace != "" {
		fmt.Fprintf(w, "event: endpoint\ndata: /message?sessionId=%s&workspaceId=%s\r\n\r\n", session.Id, workspace)
	} else {
		fmt.Fprintf(w, "event: endpoint\ndata: /message?sessionId=%s\r\n\r\n", session.Id)
	}
	flusher.Flush()

	// 获取事件通道和关闭函数
	eventChan, closeChan := session.GetEventChanWithCloser()

	// 转发所有SSE事件
	for {
		select {
		case <-c.Request().Context().Done():
			// client closed connection
			xl.Infof("Client closed connection, sessionId: %s", querySessionId)
			// 关闭当前客户端的事件通道
			closeChan()
			return nil
		case event := <-eventChan:
			xl.Infof("to sse: %v", event)
			//ev := fmt.Sprintf("event: message", event.Data)
			fmt.Fprintf(w, "event: %s\n", event.Event)
			flusher.Flush()
			fmt.Fprintf(w, "data: %s\n\n", event.Data)
			flusher.Flush()
		}
	}
}
