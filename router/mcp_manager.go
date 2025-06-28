package router

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/types"
	"github.com/lucky-aeon/agentx/plugin-helper/utils"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

// GET ALL MCP SERVICES
func (m *ServerManager) handleGetAllServices(c echo.Context) error {
	xl := xlog.NewLogger("GET-SERVICES")
	xl.Infof("Get all services")
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)
	mcpServices := m.mcpServiceMgr.GetMcpServices(xl, service.NameArg{
		Workspace: workspace,
	})
	var serviceInfos []service.McpServiceInfo
	for _, instance := range mcpServices {
		serviceInfos = append(serviceInfos, instance.Info())
	}
	return c.JSON(http.StatusOK, serviceInfos)
}

// DeployServer 部署单个服务
func (m *ServerManager) DeployServer(name string, config config.MCPServerConfig) (service.AddMcpServiceResult, error) {
	m.Lock()
	defer m.Unlock()

	logger := xlog.NewLogger("DEPLOY")

	if config.Command == "" && config.URL == "" {
		return "", fmt.Errorf("服务配置必须包含 URL 或 Command")
	}

	if config.Command != "" && config.URL != "" {
		return "", fmt.Errorf("服务配置不能同时包含 URL 和 Command")
	}

	if config.Workspace == "" {
		config.Workspace = service.DefaultWorkspace
	}
	return m.mcpServiceMgr.DeployServer(logger, service.NameArg{
		Server:    name,
		Workspace: config.Workspace,
	}, config)
}

// handleDeploy 处理部署请求
func (m *ServerManager) handleDeploy(c echo.Context) error {
	xl := xlog.NewLogger("DEPLOY-REQ")
	xl.Infof("Deploy request: %v", c.Request().Body)
	var req types.DeployRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	xl.Infof("Deploy request: %v", req)
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)

	// 初始化响应结构
	response := types.DeployResponse{
		Success: true,
		Results: make(map[string]types.ServiceDeployResult),
		Summary: types.DeploymentSummary{
			Total: len(req.MCPServers),
		},
	}

	// 部署每个服务
	for name, config := range req.MCPServers {
		xl.Infof("Deploying %s: %v", name, config)
		if workspace != "" {
			config.Workspace = workspace
		} else if config.Workspace == "" {
			config.Workspace = service.DefaultWorkspace
		}

		result, err := m.DeployServer(name, config)
		serviceResult := types.ServiceDeployResult{
			Name: name,
		}

		if err != nil {
			xl.Errorf("Failed to deploy %s: %v", name, err)
			serviceResult.Status = types.ServiceDeployStatusFailed
			serviceResult.Error = err.Error()
			serviceResult.Message = fmt.Sprintf("部署失败: %v", err)
			response.Summary.Failed++
			response.Success = false
		} else {
			// 根据部署结果设置状态
			switch result {
			case service.AddMcpServiceResultDeployed:
				serviceResult.Status = types.ServiceDeployStatusDeployed
				serviceResult.Message = "服务部署成功"
				response.Summary.Deployed++
			case service.AddMcpServiceResultExisted:
				serviceResult.Status = types.ServiceDeployStatusExisted
				serviceResult.Message = "服务已存在且正在运行"
				response.Summary.Existed++
			case service.AddMcpServiceResultReplaced:
				serviceResult.Status = types.ServiceDeployStatusReplaced
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
func (m *ServerManager) handleDeleteMcpService(c echo.Context) error {
	xl := xlog.NewLogger("DELETE-SVC")
	xl.Infof("Delete request: %v", c.Request().Body)
	name := c.QueryParam("name")
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)
	if err := m.mcpServiceMgr.DeleteServer(xl, service.NameArg{
		Server:    name,
		Workspace: workspace,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "success"})
}

// handleGetServiceHealth 获取服务健康状态
func (m *ServerManager) handleGetServiceHealth(c echo.Context) error {
	xl := xlog.NewLogger("GET-SERVICE-HEALTH")
	serviceName := c.Param("name")
	workspace := utils.GetWorkspace(c, service.DefaultWorkspace)

	xl.Infof("Get service health for %s in workspace %s", serviceName, workspace)

	mcpService, err := m.mcpServiceMgr.GetMcpService(xl, service.NameArg{
		Server:    serviceName,
		Workspace: workspace,
	})
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": fmt.Sprintf("Service %s not found: %v", serviceName, err)})
	}

	health := mcpService.GetHealthStatus()
	return c.JSON(http.StatusOK, health)
}
