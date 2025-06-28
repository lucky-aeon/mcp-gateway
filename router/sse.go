package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/utils"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

// 全局SSE，这里返回所有MCP服务的SSE事件
func (m *ServerManager) handleGlobalSSE(c echo.Context) error {
	xl := xlog.NewLogger("GLOBAL-SSE")
	xl.Infof("Global SSE request: %v", c.Request().Body)
	querySessionId, err := utils.GetSession(c)
	if err != nil {
		xl.Warnf("Get session error: %v", err)
	}
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)
	if querySessionId == "" {
		xl.Infof("No session ID provided, creating new session")
		// 没有sessionId，生成一个返回出
		// create proxy session
		session, err := m.mcpServiceMgr.CreateProxySession(xl, service.NameArg{
			Workspace: workspace,
			Session:   querySessionId,
		})
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		xl.Infof("Created new session: %s", session.Id)
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
	session, exists := m.mcpServiceMgr.GetProxySession(xl, service.NameArg{
		Workspace: workspace,
		Session:   querySessionId,
	})
	if !exists {
		return c.String(http.StatusNotFound, "session not found")
	}

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
