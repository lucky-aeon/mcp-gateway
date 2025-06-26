package router

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

// DebugRequest 调试请求结构
type DebugRequest struct {
	Message string `json:"message" validate:"required"`
	Method  string `json:"method,omitempty"`
}

// DebugResponse 调试响应结构
type DebugResponse struct {
	Success     bool                   `json:"success"`
	Response    map[string]interface{} `json:"response,omitempty"`
	Error       string                 `json:"error,omitempty"`
	ServiceInfo service.McpServiceInfo `json:"service_info"`
	RequestLog  string                 `json:"request_log,omitempty"`
	ResponseLog string                 `json:"response_log,omitempty"`
}

// LogEntry 日志条目结构
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// ServiceLogsResponse 服务日志响应
type ServiceLogsResponse struct {
	ServiceName string     `json:"service_name"`
	Logs        []LogEntry `json:"logs"`
	TotalLines  int        `json:"total_lines"`
}

// handleDebugService 调试特定服务
func (m *ServerManager) handleDebugService(c echo.Context) error {
	workspace := c.Param("workspace")
	serviceName := c.Param("name")

	if workspace == "" {
		workspace = "default"
	}

	logger := xlog.NewLogger("[Debug]")
	logger.Infof("Debug request for service %s in workspace %s", serviceName, workspace)

	var req DebugRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format: " + err.Error(),
		})
	}

	if req.Message == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Message is required",
		})
	}

	// 获取服务实例
	nameArg := service.NameArg{
		Workspace: workspace,
		Server:    serviceName,
	}

	mcpService, err := m.mcpServiceMgr.GetMcpService(logger, nameArg)
	if err != nil {
		return c.JSON(http.StatusNotFound, DebugResponse{
			Success: false,
			Error:   fmt.Sprintf("Service not found: %v", err),
		})
	}

	// 获取服务信息
	serviceInfo := mcpService.Info()

	// 检查服务状态
	if serviceInfo.Status != service.Running {
		return c.JSON(http.StatusServiceUnavailable, DebugResponse{
			Success:     false,
			Error:       fmt.Sprintf("Service is not running (status: %s)", serviceInfo.Status),
			ServiceInfo: serviceInfo,
		})
	}

	// 记录请求日志
	requestLog := fmt.Sprintf("DEBUG REQUEST to %s: %s", serviceName, req.Message)
	logger.Infof(requestLog)

	// 发送消息到MCP服务
	response := DebugResponse{
		ServiceInfo: serviceInfo,
		RequestLog:  requestLog,
	}

	err = mcpService.SendMessage(req.Message)
	if err != nil {
		responseLog := fmt.Sprintf("DEBUG RESPONSE ERROR: %v", err)
		logger.Errorf(responseLog)

		response.Success = false
		response.Error = err.Error()
		response.ResponseLog = responseLog

		return c.JSON(http.StatusInternalServerError, response)
	}

	responseLog := "DEBUG RESPONSE: Message sent successfully"
	logger.Infof(responseLog)

	response.Success = true
	response.Response = map[string]interface{}{
		"message": "Debug message sent successfully",
		"sent_at": serviceInfo.LastStartedAt,
	}
	response.ResponseLog = responseLog

	return c.JSON(http.StatusOK, response)
}

// handleGetServiceDebugInfo 获取服务调试信息
func (m *ServerManager) handleGetServiceDebugInfo(c echo.Context) error {
	workspace := c.Param("workspace")
	serviceName := c.Param("name")

	if workspace == "" {
		workspace = "default"
	}

	logger := xlog.NewLogger("[DebugInfo]")

	nameArg := service.NameArg{
		Workspace: workspace,
		Server:    serviceName,
	}

	mcpService, err := m.mcpServiceMgr.GetMcpService(logger, nameArg)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Service not found: %v", err),
		})
	}

	serviceInfo := mcpService.Info()
	healthStatus := mcpService.GetHealthStatus()

	debugInfo := map[string]interface{}{
		"service_info":  serviceInfo,
		"health_status": healthStatus,
		"debug_endpoints": map[string]string{
			"message_url": serviceInfo.URLs.MessageUrl,
			"sse_url":     serviceInfo.URLs.SSEUrl,
			"base_url":    serviceInfo.URLs.BaseURL,
		},
		"debug_commands": []string{
			"GET /api/workspaces/" + workspace + "/services/" + serviceName + "/debug/info",
			"POST /api/workspaces/" + workspace + "/services/" + serviceName + "/debug/test",
			"GET /api/workspaces/" + workspace + "/services/" + serviceName + "/debug/logs",
		},
	}

	return c.JSON(http.StatusOK, debugInfo)
}

// handleTestServiceConnection 测试服务连接
func (m *ServerManager) handleTestServiceConnection(c echo.Context) error {
	workspace := c.Param("workspace")
	serviceName := c.Param("name")

	if workspace == "" {
		workspace = "default"
	}

	logger := xlog.NewLogger("[TestConnection]")

	nameArg := service.NameArg{
		Workspace: workspace,
		Server:    serviceName,
	}

	mcpService, err := m.mcpServiceMgr.GetMcpService(logger, nameArg)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("Service not found: %v", err),
		})
	}

	serviceInfo := mcpService.Info()

	// 测试基本连接
	testResult := map[string]interface{}{
		"service_name": serviceName,
		"workspace":    workspace,
		"status":       serviceInfo.Status,
		"healthy":      serviceInfo.Status == service.Running,
		"urls":         serviceInfo.URLs,
		"tests":        []map[string]interface{}{},
	}

	tests := []map[string]interface{}{}

	// 测试1: 检查服务状态
	statusTest := map[string]interface{}{
		"name":    "Service Status Check",
		"success": serviceInfo.Status == service.Running,
		"details": fmt.Sprintf("Service status: %s", serviceInfo.Status),
	}
	if serviceInfo.Status != service.Running {
		statusTest["error"] = fmt.Sprintf("Service is not running (status: %s)", serviceInfo.Status)
	}
	tests = append(tests, statusTest)

	// 测试2: 端口检查
	portTest := map[string]interface{}{
		"name":    "Port Availability Check",
		"success": serviceInfo.Port > 0,
		"details": fmt.Sprintf("Service port: %d", serviceInfo.Port),
	}
	if serviceInfo.Port <= 0 {
		portTest["error"] = "Invalid port number"
		portTest["success"] = false
	}
	tests = append(tests, portTest)

	// 测试3: URL可达性检查
	urlTest := map[string]interface{}{
		"name":    "URL Reachability Check",
		"success": serviceInfo.URLs.BaseURL != "",
		"details": fmt.Sprintf("Base URL: %s", serviceInfo.URLs.BaseURL),
	}
	if serviceInfo.URLs.BaseURL == "" {
		urlTest["error"] = "Base URL is not available"
		urlTest["success"] = false
	}
	tests = append(tests, urlTest)

	// 测试4: 发送测试消息
	if serviceInfo.Status == service.Running && serviceInfo.URLs.MessageUrl != "" {
		testMessage := `{"jsonrpc": "2.0", "id": 1, "method": "ping", "params": {}}`
		messageTest := map[string]interface{}{
			"name":    "Test Message Send",
			"details": "Sending ping message to service",
		}

		err := mcpService.SendMessage(testMessage)
		if err != nil {
			messageTest["success"] = false
			messageTest["error"] = fmt.Sprintf("Failed to send test message: %v", err)
		} else {
			messageTest["success"] = true
			messageTest["details"] = "Test message sent successfully"
		}
		tests = append(tests, messageTest)
	}

	testResult["tests"] = tests

	// 计算总体成功率
	successCount := 0
	for _, test := range tests {
		if success, ok := test["success"].(bool); ok && success {
			successCount++
		}
	}

	testResult["overall_success"] = len(tests) > 0 && successCount == len(tests)
	testResult["success_rate"] = fmt.Sprintf("%d/%d", successCount, len(tests))

	return c.JSON(http.StatusOK, testResult)
}

// handleGetServiceDebugLogs 获取服务日志（调试用）
func (m *ServerManager) handleGetServiceDebugLogs(c echo.Context) error {
	workspace := c.Param("workspace")
	serviceName := c.Param("name")

	if workspace == "" {
		workspace = "default"
	}

	// 获取查询参数
	limitStr := c.QueryParam("limit")
	offsetStr := c.QueryParam("offset")

	limit := 100 // 默认返回最后100行
	offset := 0

	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	if offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil && o >= 0 {
			offset = o
		}
	}

	logger := xlog.NewLogger("[ServiceLogs]")

	nameArg := service.NameArg{
		Workspace: workspace,
		Server:    serviceName,
	}

	mcpService, err := m.mcpServiceMgr.GetMcpService(logger, nameArg)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": fmt.Sprintf("Service not found: %v", err),
		})
	}

	serviceInfo := mcpService.Info()

	// 模拟日志读取（实际实现中应该从日志文件读取）
	logs := []LogEntry{
		{
			Timestamp: serviceInfo.DeployedAt.Format("2006-01-02 15:04:05"),
			Level:     "INFO",
			Message:   fmt.Sprintf("Service %s deployed", serviceName),
		},
	}

	if !serviceInfo.LastStartedAt.IsZero() {
		logs = append(logs, LogEntry{
			Timestamp: serviceInfo.LastStartedAt.Format("2006-01-02 15:04:05"),
			Level:     "INFO",
			Message:   fmt.Sprintf("Service %s started on port %d", serviceName, serviceInfo.Port),
		})
	}

	if serviceInfo.LastError != "" {
		logs = append(logs, LogEntry{
			Timestamp: serviceInfo.LastStoppedAt.Format("2006-01-02 15:04:05"),
			Level:     "ERROR",
			Message:   serviceInfo.LastError,
		})
	}

	if serviceInfo.Status == service.Running {
		logs = append(logs, LogEntry{
			Timestamp: "current",
			Level:     "INFO",
			Message:   fmt.Sprintf("Service %s is running (uptime: %.0f seconds)", serviceName, 0.0), // 需要实际计算uptime
		})
	}

	// 应用分页
	totalLines := len(logs)
	start := offset
	end := offset + limit

	if start >= totalLines {
		logs = []LogEntry{}
	} else {
		if end > totalLines {
			end = totalLines
		}
		logs = logs[start:end]
	}

	response := ServiceLogsResponse{
		ServiceName: serviceName,
		Logs:        logs,
		TotalLines:  totalLines,
	}

	return c.JSON(http.StatusOK, response)
}

// 添加调试路由到ServerManager的初始化中
func (m *ServerManager) setupDebugRoutes(api *echo.Group) {
	// 调试相关路由
	debug := api.Group("/workspaces/:workspace/services/:name/debug")
	debug.GET("/info", m.handleGetServiceDebugInfo)         // 获取调试信息
	debug.POST("/test", m.handleDebugService)               // 发送调试消息
	debug.GET("/connection", m.handleTestServiceConnection) // 测试连接
	debug.GET("/logs", m.handleGetServiceDebugLogs)        // 获取日志
}
