package admin

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/apitypes"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// handleDeployServiceToWorkspace 在指定工作空间部署服务
func (h *Handler) handleDeployServiceToWorkspace(c echo.Context) error {
	xl := xlog.NewLogger("DEPLOY-TO-WORKSPACE")
	workspaceID := c.Param("workspace")
	xl.Infof("Deploy service to workspace: %s", workspaceID)

	var req apitypes.DeployRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	for name, config := range req.MCPServers {
		config.Workspace = workspaceID
		if _, err := h.DeployServer(name, config); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleUpdateServiceConfig 更新服务配置
func (h *Handler) handleUpdateServiceConfig(c echo.Context) error {
	xl := xlog.NewLogger("UPDATE-SERVICE-CONFIG")
	workspaceID := c.Param("workspace")
	serviceName := c.Param("name")
	xl.Infof("Update service %s config in workspace: %s", serviceName, workspaceID)

	var config config.MCPServerConfig
	if err := c.Bind(&config); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	config.Workspace = workspaceID

	// 先停止服务
	h.services.StopServer(xl, workspaces.NameArg{
		Workspace: workspaceID,
		Server:    serviceName,
	})

	// 删除旧服务
	if err := h.services.DeleteServer(xl, workspaces.NameArg{
		Workspace: workspaceID,
		Server:    serviceName,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	// 重新部署服务
	if _, err := h.DeployServer(serviceName, config); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleRestartService 重启服务
func (h *Handler) handleRestartService(c echo.Context) error {
	xl := xlog.NewLogger("RESTART-SERVICE")
	workspaceID := c.Param("workspace")
	serviceName := c.Param("name")
	xl.Infof("Restart service %s in workspace: %s", serviceName, workspaceID)

	if err := h.services.RestartServer(xl, workspaces.NameArg{
		Workspace: workspaceID,
		Server:    serviceName,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleStopService 停止服务
func (h *Handler) handleStopService(c echo.Context) error {
	xl := xlog.NewLogger("STOP-SERVICE")
	workspaceID := c.Param("workspace")
	serviceName := c.Param("name")
	xl.Infof("Stop service %s in workspace: %s", serviceName, workspaceID)

	h.services.StopServer(xl, workspaces.NameArg{
		Workspace: workspaceID,
		Server:    serviceName,
	})

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleStartService 启动服务
func (h *Handler) handleStartService(c echo.Context) error {
	xl := xlog.NewLogger("START-SERVICE")
	workspaceID := c.Param("workspace")
	serviceName := c.Param("name")
	xl.Infof("Start service %s in workspace: %s", serviceName, workspaceID)

	// 获取服务配置
	configs := h.services.ListServerConfig(xl, workspaces.NameArg{
		Workspace: workspaceID,
	})

	config, ok := configs[serviceName]
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Service not found",
		})
	}

	// 部署服务
	if _, err := h.DeployServer(serviceName, config); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleDeleteServiceFromWorkspace 从工作空间删除服务
func (h *Handler) handleDeleteServiceFromWorkspace(c echo.Context) error {
	xl := xlog.NewLogger("DELETE-SERVICE-FROM-WORKSPACE")
	workspaceID := c.Param("workspace")
	serviceName := c.Param("name")
	xl.Infof("Delete service %s from workspace: %s", serviceName, workspaceID)

	if err := h.services.DeleteServer(xl, workspaces.NameArg{
		Workspace: workspaceID,
		Server:    serviceName,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleGetServiceLogs 获取服务日志 (预留接口)
func (h *Handler) handleGetServiceLogs(c echo.Context) error {
	xl := xlog.NewLogger("GET-SERVICE-LOGS")
	workspaceID := c.Param("workspace")
	serviceName := c.Param("name")
	xl.Infof("Get logs for service %s in workspace: %s", serviceName, workspaceID)

	// TODO: 实现日志获取功能
	return c.JSON(http.StatusOK, map[string]interface{}{
		"logs": []string{
			"Log functionality will be implemented in the future",
		},
	})
}
