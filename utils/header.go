package utils

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// IsSSE 检查是否是 SSE 请求
func IsSSE(header http.Header) bool {
	return strings.Contains(header.Get("Content-Type"), "text/event-stream")
}

// GetWorkspace 获取 workspace, 优先从 header 中获取，如果没有则从 query 中获取
func GetWorkspace(c echo.Context, defaultWorkspace ...string) string {
	workspace := c.Request().Header.Get("X-Workspace-Id")
	if workspace == "" {
		workspace = c.QueryParam("workspaceId")
	}
	// 如果 workspace 为空，则使用默认值
	if workspace == "" && len(defaultWorkspace) > 0 {
		workspace = defaultWorkspace[0]
	}
	return workspace
}

// GetSession 获取 session, 优先从 header 中获取，如果没有则从 query 中获取
func GetSession(c echo.Context) (string, error) {
	session := c.Request().Header.Get("X-Session-Id")
	if session == "" {
		session = c.QueryParam("sessionId")
	}
	if session == "" {
		return "", fmt.Errorf("missing sessionId")
	}
	return session, nil
}
