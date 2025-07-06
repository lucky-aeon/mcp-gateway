package router

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

// WorkspaceInfo 工作空间信息
type WorkspaceInfo struct {
	ID           string                   `json:"id"`
	Status       string                   `json:"status"`
	ServiceCount int                      `json:"service_count"`
	SessionCount int                      `json:"session_count"`
	Services     []service.McpServiceInfo `json:"services,omitempty"`
}

// handleGetAllWorkspaces 获取所有工作空间
func (m *ServerManager) handleGetAllWorkspaces(c echo.Context) error {
	xl := xlog.NewLogger("GET-WORKSPACES")
	xl.Info("Get all workspaces")

	// 通过 service manager 获取所有工作空间
	workspaces := m.mcpServiceMgr.(*service.ServiceManager).GetWorkspaces()

	var workspaceInfos []WorkspaceInfo
	for id, workspace := range workspaces {
		services := workspace.GetMcpServices()
		var serviceInfos []service.McpServiceInfo
		for _, svc := range services {
			serviceInfos = append(serviceInfos, svc.Info())
		}

		// 获取工作空间的session数量
		sessions := m.mcpServiceMgr.GetWorkspaceSessions(xl, service.NameArg{
			Workspace: id,
		})

		workspaceInfo := WorkspaceInfo{
			ID:           id,
			Status:       "running", // 简化状态，实际可以从 workspace 获取
			ServiceCount: len(services),
			SessionCount: len(sessions),
			Services:     serviceInfos,
		}
		workspaceInfos = append(workspaceInfos, workspaceInfo)
	}

	return c.JSON(http.StatusOK, workspaceInfos)
}

// handleCreateWorkspace 创建新工作空间
func (m *ServerManager) handleCreateWorkspace(c echo.Context) error {
	xl := xlog.NewLogger("CREATE-WORKSPACE")
	xl.Info("Create new workspace")

	var req struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// 创建工作空间 - 通过尝试获取一个不存在的服务来触发工作空间创建
	_, err := m.mcpServiceMgr.GetMcpService(xl, service.NameArg{
		Workspace: req.ID,
		Server:    "dummy", // 虚拟服务名，只是为了触发工作空间创建
	})

	// 预期会失败，但工作空间应该已经被创建
	if err != nil {
		xl.Infof("Workspace %s created (expected error: %v)", req.ID, err)
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"status":  "success",
		"message": "Workspace created",
		"id":      req.ID,
	})
}

// handleDeleteWorkspace 删除工作空间
func (m *ServerManager) handleDeleteWorkspace(c echo.Context) error {
	xl := xlog.NewLogger("DELETE-WORKSPACE")
	workspaceID := c.Param("id")
	xl.Infof("Delete workspace: %s", workspaceID)

	// 获取工作空间下的所有服务并删除
	services := m.mcpServiceMgr.GetMcpServices(xl, service.NameArg{
		Workspace: workspaceID,
	})

	for serviceName := range services {
		if err := m.mcpServiceMgr.DeleteServer(xl, service.NameArg{
			Workspace: workspaceID,
			Server:    serviceName,
		}); err != nil {
			xl.Errorf("Failed to delete service %s: %v", serviceName, err)
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to delete workspace services",
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleGetWorkspaceServices 获取工作空间下的服务
func (m *ServerManager) handleGetWorkspaceServices(c echo.Context) error {
	xl := xlog.NewLogger("GET-WORKSPACE-SERVICES")
	workspaceID := c.Param("id")
	xl.Infof("Get services for workspace: %s", workspaceID)

	services := m.mcpServiceMgr.GetMcpServices(xl, service.NameArg{
		Workspace: workspaceID,
	})

	serviceInfos := []service.McpServiceInfo{}
	for _, svc := range services {
		serviceInfos = append(serviceInfos, svc.Info())
	}

	return c.JSON(http.StatusOK, serviceInfos)
}
