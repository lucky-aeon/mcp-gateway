package router

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

// SessionInfo 会话信息
type SessionInfo struct {
	ID              string    `json:"id"`
	WorkspaceID     string    `json:"workspace_id"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	LastReceiveTime time.Time `json:"last_receive_time"`
	IsReady         bool      `json:"is_ready"`
}

// handleGetWorkspaceSessions 获取工作空间的会话
func (m *ServerManager) handleGetWorkspaceSessions(c echo.Context) error {
	xl := xlog.NewLogger("GET-WORKSPACE-SESSIONS")
	workspaceID := c.Param("workspace")
	xl.Infof("Get sessions for workspace: %s", workspaceID)

	// TODO: 需要扩展 ServiceManager 接口来获取会话信息
	// 目前只返回空列表，后续需要实现
	sessions := []SessionInfo{}

	return c.JSON(http.StatusOK, sessions)
}

// handleCreateSession 创建新会话
func (m *ServerManager) handleCreateSession(c echo.Context) error {
	xl := xlog.NewLogger("CREATE-SESSION")
	workspaceID := c.Param("workspace")
	xl.Infof("Create session for workspace: %s", workspaceID)

	session, err := m.mcpServiceMgr.CreateProxySession(xl, service.NameArg{
		Workspace: workspaceID,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	sessionInfo := SessionInfo{
		ID:              session.GetId(),
		WorkspaceID:     workspaceID,
		Status:          "active",
		CreatedAt:       time.Now(),
		LastReceiveTime: session.LastReceiveTime,
		IsReady:         session.IsReady(),
	}

	return c.JSON(http.StatusCreated, sessionInfo)
}

// handleDeleteSession 删除会话
func (m *ServerManager) handleDeleteSession(c echo.Context) error {
	xl := xlog.NewLogger("DELETE-SESSION")
	workspaceID := c.Param("workspace")
	sessionID := c.Param("id")
	xl.Infof("Delete session %s in workspace: %s", sessionID, workspaceID)

	m.mcpServiceMgr.CloseProxySession(xl, service.NameArg{
		Workspace: workspaceID,
		Session:   sessionID,
	})

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleGetSessionStatus 获取会话状态
func (m *ServerManager) handleGetSessionStatus(c echo.Context) error {
	xl := xlog.NewLogger("GET-SESSION-STATUS")
	sessionID := c.Param("id")
	xl.Infof("Get status for session: %s", sessionID)

	// 需要通过查询所有工作空间来找到会话
	workspaces := m.mcpServiceMgr.(*service.ServiceManager).GetWorkspaces()

	for workspaceID := range workspaces {
		session, exists := m.mcpServiceMgr.GetProxySession(xl, service.NameArg{
			Workspace: workspaceID,
			Session:   sessionID,
		})

		if exists {
			sessionInfo := SessionInfo{
				ID:              session.GetId(),
				WorkspaceID:     workspaceID,
				Status:          "active",
				CreatedAt:       time.Now(), // TODO: 需要从 session 获取实际创建时间
				LastReceiveTime: session.LastReceiveTime,
				IsReady:         session.IsReady(),
			}
			return c.JSON(http.StatusOK, sessionInfo)
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{
		"error": "Session not found",
	})
}
