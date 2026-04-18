package admin

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/apitypes"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/httpx"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// GET ALL MCP SERVICES
func (h *Handler) handleGetAllServices(c echo.Context) error {
	xl := xlog.NewLogger("GET-SERVICES")
	xl.Infof("Get all services")
	workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)
	mcpServices := h.services.GetMcpServices(xl, workspaces.NameArg{
		Workspace: workspace,
	})
	var serviceInfos []runtime.McpServiceInfo
	for _, instance := range mcpServices {
		serviceInfos = append(serviceInfos, instance.Info())
	}
	return c.JSON(http.StatusOK, serviceInfos)
}

// DeployServer 部署单个服务
func (h *Handler) DeployServer(name string, config config.MCPServerConfig) (workspaces.AddMcpServiceResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	logger := xlog.NewLogger("DEPLOY")

	if config.Command == "" && config.URL == "" {
		return "", fmt.Errorf("服务配置必须包含 URL 或 Command")
	}

	if config.Command != "" && config.URL != "" {
		return "", fmt.Errorf("服务配置不能同时包含 URL 和 Command")
	}

	if config.Workspace == "" {
		config.Workspace = workspaces.DefaultWorkspace
	}
	return h.services.DeployServer(logger, workspaces.NameArg{
		Server:    name,
		Workspace: config.Workspace,
	}, config)
}

// handleDeploy 处理部署请求
func (h *Handler) handleDeploy(c echo.Context) error {
	xl := xlog.NewLogger("DEPLOY-REQ")
	xl.Infof("Deploy request: %v", c.Request().Body)
	var req apitypes.DeployRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	xl.Infof("Deploy request: %v", req)
	workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)

	// 初始化响应结构
	response := apitypes.DeployResponse{
		Success: true,
		Results: make(map[string]apitypes.ServiceDeployResult),
		Summary: apitypes.DeploymentSummary{
			Total: len(req.MCPServers),
		},
	}

	// 部署每个服务
	for name, config := range req.MCPServers {
		xl.Infof("Deploying %s: %v", name, config)
		if workspace != "" {
			config.Workspace = workspace
		} else if config.Workspace == "" {
			config.Workspace = workspaces.DefaultWorkspace
		}

		result, err := h.DeployServer(name, config)
		serviceResult := apitypes.ServiceDeployResult{
			Name: name,
		}

		if err != nil {
			xl.Errorf("Failed to deploy %s: %v", name, err)
			serviceResult.Status = apitypes.ServiceDeployStatusFailed
			serviceResult.Error = err.Error()
			serviceResult.Message = fmt.Sprintf("部署失败: %v", err)
			response.Summary.Failed++
			response.Success = false
		} else {
			// 根据部署结果设置状态
			switch result {
			case workspaces.AddMcpServiceResultDeployed:
				serviceResult.Status = apitypes.ServiceDeployStatusDeployed
				serviceResult.Message = "服务部署成功"
				response.Summary.Deployed++
			case workspaces.AddMcpServiceResultExisted:
				serviceResult.Status = apitypes.ServiceDeployStatusExisted
				serviceResult.Message = "服务已存在且正在运行"
				response.Summary.Existed++
			case workspaces.AddMcpServiceResultReplaced:
				serviceResult.Status = apitypes.ServiceDeployStatusReplaced
				serviceResult.Message = "服务已替换（原服务已停止或失败）"
				response.Summary.Replaced++
			}
		}

		response.Results[name] = serviceResult
	}

	// 设置整体消息
	if response.Success {
		response.Message = fmt.Sprintf("部署完成: %d个服务总计，%d个新部署，%d个已存在，%d个已替换，%d个失败",
			response.Summary.Total, response.Summary.Deployed,
			response.Summary.Existed, response.Summary.Replaced, response.Summary.Failed)
	} else {
		response.Message = fmt.Sprintf("部署完成但存在失败: %d个服务总计，%d个新部署，%d个已存在，%d个已替换，%d个失败",
			response.Summary.Total, response.Summary.Deployed,
			response.Summary.Existed, response.Summary.Replaced, response.Summary.Failed)
	}

	xl.Infof("Deployment completed: %s", response.Message)

	// 根据是否有失败来决定HTTP状态码
	statusCode := http.StatusOK
	if response.Summary.Failed > 0 {
		statusCode = http.StatusPartialContent // 206表示部分成功
	}

	return c.JSON(statusCode, response)
}

// handleDeleteMcpService 删除单个服务
func (h *Handler) handleDeleteMcpService(c echo.Context) error {
	xl := xlog.NewLogger("DELETE-SVC")
	xl.Infof("Delete request: %v", c.Request().Body)
	name := c.QueryParam("name")
	workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)
	if err := h.services.DeleteServer(xl, workspaces.NameArg{
		Server:    name,
		Workspace: workspace,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleGetServiceHealth 获取服务健康状态
func (h *Handler) handleGetServiceHealth(c echo.Context) error {
	xl := xlog.NewLogger("GET-SERVICE-HEALTH")
	serviceName := c.Param("name")
	workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)

	xl.Infof("Get service health for %s in workspace %s", serviceName, workspace)

	mcpService, err := h.services.GetMcpService(xl, workspaces.NameArg{
		Server:    serviceName,
		Workspace: workspace,
	})
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": fmt.Sprintf("Service %s not found: %v", serviceName, err)})
	}

	health := mcpService.GetHealthStatus()
	return c.JSON(http.StatusOK, health)
}
