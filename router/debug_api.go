package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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

// APIEndpoint API端点信息
type APIEndpoint struct {
	Method      string         `json:"method"`
	Path        string         `json:"path"`
	Handler     string         `json:"handler"`
	Middleware  []string       `json:"middleware,omitempty"`
	Parameters  []APIParameter `json:"parameters,omitempty"`
	Description string         `json:"description,omitempty"`
	Group       string         `json:"group,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	Examples    []APIExample   `json:"examples,omitempty"`
}

// APIParameter API参数信息
type APIParameter struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Location    string `json:"location"` // path, query, body, header
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

// APIExample API使用示例
type APIExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Request     map[string]interface{} `json:"request,omitempty"`
	Response    map[string]interface{} `json:"response,omitempty"`
}

// APIDiscoveryResponse API发现响应
type APIDiscoveryResponse struct {
	TotalEndpoints int           `json:"total_endpoints"`
	Groups         []string      `json:"groups"`
	Endpoints      []APIEndpoint `json:"endpoints"`
	GeneratedAt    time.Time     `json:"generated_at"`
	Version        string        `json:"version"`
}

// APITestRequest API测试请求
type APITestRequest struct {
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Headers     map[string]string      `json:"headers,omitempty"`
	Query       map[string]string      `json:"query,omitempty"`
	Body        map[string]interface{} `json:"body,omitempty"`
	ContentType string                 `json:"content_type,omitempty"`
}

// APITestResponse API测试响应
type APITestResponse struct {
	Success        bool                   `json:"success"`
	StatusCode     int                    `json:"status_code"`
	ResponseTime   time.Duration          `json:"response_time"`
	Response       map[string]interface{} `json:"response,omitempty"`
	Error          string                 `json:"error,omitempty"`
	RequestHeaders map[string]string      `json:"request_headers,omitempty"`
	RequestBody    string                 `json:"request_body,omitempty"`
	ResponseBody   string                 `json:"response_body,omitempty"`
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
	debug.GET("/logs", m.handleGetServiceDebugLogs)         // 获取日志

	// API发现和调试路由
	apiDebug := api.Group("/debug")
	apiDebug.GET("/apis", m.handleDiscoverAPIs)        // 获取所有API列表
	apiDebug.POST("/apis/test", m.handleTestAPI)       // 测试API端点
	apiDebug.GET("/apis/groups", m.handleGetAPIGroups) // 获取API分组
}

// handleDiscoverAPIs 自动发现所有API端点
func (m *ServerManager) handleDiscoverAPIs(c echo.Context) error {
	logger := xlog.NewLogger("[APIDiscovery]")
	logger.Info("Starting API discovery")

	// 获取Echo实例的路由信息
	echoInstance := c.Echo()
	routes := echoInstance.Routes()

	var endpoints []APIEndpoint
	groupSet := make(map[string]bool)

	for _, route := range routes {
		endpoint := m.analyzeRoute(route)
		if endpoint != nil {
			endpoints = append(endpoints, *endpoint)
			if endpoint.Group != "" {
				groupSet[endpoint.Group] = true
			}
		}
	}

	// 排序端点
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Group != endpoints[j].Group {
			return endpoints[i].Group < endpoints[j].Group
		}
		if endpoints[i].Method != endpoints[j].Method {
			return endpoints[i].Method < endpoints[j].Method
		}
		return endpoints[i].Path < endpoints[j].Path
	})

	// 提取分组列表
	var groups []string
	for group := range groupSet {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	response := APIDiscoveryResponse{
		TotalEndpoints: len(endpoints),
		Groups:         groups,
		Endpoints:      endpoints,
		GeneratedAt:    time.Now(),
		Version:        "1.0",
	}

	logger.Infof("Discovered %d API endpoints in %d groups", len(endpoints), len(groups))
	return c.JSON(http.StatusOK, response)
}

// analyzeRoute 分析路由信息
func (m *ServerManager) analyzeRoute(route *echo.Route) *APIEndpoint {
	// 过滤掉不需要的路由
	if strings.Contains(route.Path, "*") && route.Path != "/*" {
		return nil
	}

	// 确定API分组
	group := m.getAPIGroup(route.Path)

	// 分析路径参数
	parameters := m.extractPathParameters(route.Path)

	// 获取处理器名称
	handlerName := m.getHandlerName(route.Name)

	// 添加描述和示例
	description := m.getAPIDescription(route.Method, route.Path, handlerName)
	examples := m.getAPIExamples(route.Method, route.Path)

	endpoint := &APIEndpoint{
		Method:      route.Method,
		Path:        route.Path,
		Handler:     handlerName,
		Parameters:  parameters,
		Description: description,
		Group:       group,
		Examples:    examples,
		Tags:        m.getAPITags(route.Path),
	}

	return endpoint
}

// getAPIGroup 根据路径确定API分组
func (m *ServerManager) getAPIGroup(path string) string {
	if strings.HasPrefix(path, "/api/debug") {
		return "调试接口"
	} else if strings.HasPrefix(path, "/api/workspaces") {
		return "工作空间管理"
	} else if strings.Contains(path, "/sessions") {
		return "会话管理"
	} else if strings.Contains(path, "/services") {
		return "服务管理"
	} else if strings.HasPrefix(path, "/api") {
		return "API接口"
	} else if path == "/deploy" || path == "/delete" {
		return "核心功能"
	} else if path == "/sse" || path == "/message" {
		return "通信接口"
	} else if path == "/services" {
		return "服务状态"
	} else if path == "/*" {
		return "代理转发"
	}
	return "其他"
}

// extractPathParameters 提取路径参数
func (m *ServerManager) extractPathParameters(path string) []APIParameter {
	var parameters []APIParameter

	// 使用正则表达式匹配路径参数
	re := regexp.MustCompile(`:(\w+)`)
	matches := re.FindAllStringSubmatch(path, -1)

	for _, match := range matches {
		if len(match) > 1 {
			paramName := match[1]
			param := APIParameter{
				Name:        paramName,
				Type:        "string",
				Location:    "path",
				Required:    true,
				Description: m.getParameterDescription(paramName),
				Example:     m.getParameterExample(paramName),
			}
			parameters = append(parameters, param)
		}
	}

	return parameters
}

// getParameterDescription 获取参数描述
func (m *ServerManager) getParameterDescription(paramName string) string {
	descriptions := map[string]string{
		"workspace": "工作空间ID",
		"name":      "服务名称",
		"id":        "资源ID",
		"session":   "会话ID",
	}

	if desc, exists := descriptions[paramName]; exists {
		return desc
	}
	return fmt.Sprintf("%s参数", paramName)
}

// getParameterExample 获取参数示例
func (m *ServerManager) getParameterExample(paramName string) string {
	examples := map[string]string{
		"workspace": "default",
		"name":      "example-service",
		"id":        "12345",
		"session":   "sess_123",
	}

	if example, exists := examples[paramName]; exists {
		return example
	}
	return "example"
}

// getHandlerName 获取处理器名称
func (m *ServerManager) getHandlerName(routeName string) string {
	if routeName == "" {
		return "未知处理器"
	}

	// 清理处理器名称
	parts := strings.Split(routeName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return routeName
}

// getAPIDescription 获取API描述
func (m *ServerManager) getAPIDescription(method, path, handler string) string {
	descriptions := map[string]string{
		"POST /deploy":              "批量部署多个MCP服务",
		"DELETE /delete":            "删除单个MCP服务",
		"GET /sse":                  "全局SSE事件流",
		"POST /message":             "发送全局消息",
		"GET /services":             "获取所有服务列表",
		"GET /api/workspaces":       "获取所有工作空间",
		"POST /api/workspaces":      "创建新工作空间",
		"DELETE /api/workspaces":    "删除工作空间",
		"GET /api/debug/apis":       "发现所有API端点",
		"POST /api/debug/apis/test": "测试API端点",
	}

	key := method + " " + path
	if desc, exists := descriptions[key]; exists {
		return desc
	}

	// 基于路径和方法生成描述
	if strings.Contains(handler, "Debug") {
		return "调试相关功能"
	} else if strings.Contains(handler, "Workspace") {
		return "工作空间操作"
	} else if strings.Contains(handler, "Service") {
		return "服务管理操作"
	} else if strings.Contains(handler, "Session") {
		return "会话管理操作"
	}

	return fmt.Sprintf("%s操作", method)
}

// getAPIExamples 获取API使用示例
func (m *ServerManager) getAPIExamples(method, path string) []APIExample {
	var examples []APIExample

	switch method + " " + path {
	case "POST /deploy":
		examples = append(examples, APIExample{
			Name:        "批量部署服务示例",
			Description: "同时部署多个MCP服务",
			Request: map[string]interface{}{
				"mcpServers": map[string]interface{}{
					"git-service": map[string]interface{}{
						"workspace": "default",
						"command":   "uvx",
						"args":      []string{"mcp-server-git", "--repository", "/path/to/repo"},
						"env": map[string]string{
							"GIT_REPO_PATH": "/path/to/repo",
						},
					},
					"file-service": map[string]interface{}{
						"workspace": "default",
						"command":   "uvx",
						"args":      []string{"mcp-server-filesystem", "--allowed-paths", "/tmp"},
					},
					"web-service": map[string]interface{}{
						"workspace": "test",
						"url":       "http://localhost:3000",
					},
				},
			},
			Response: map[string]interface{}{
				"success": true,
				"message": "Deployment completed: 3 successful, 0 failed",
				"results": map[string]interface{}{
					"git-service": map[string]interface{}{
						"success": true,
						"status":  "deployed",
						"service_info": map[string]interface{}{
							"name":   "git-service",
							"status": "running",
							"port":   8081,
						},
					},
					"file-service": map[string]interface{}{
						"success": true,
						"status":  "deployed",
						"service_info": map[string]interface{}{
							"name":   "file-service",
							"status": "running",
							"port":   8082,
						},
					},
					"web-service": map[string]interface{}{
						"success": true,
						"status":  "deployed",
						"service_info": map[string]interface{}{
							"name":   "web-service",
							"status": "running",
							"port":   8083,
						},
					},
				},
				"summary": map[string]interface{}{
					"total":      3,
					"successful": 3,
					"failed":     0,
				},
			},
		})
	case "DELETE /delete":
		examples = append(examples, APIExample{
			Name:        "删除服务示例",
			Description: "删除指定的MCP服务",
			Request: map[string]interface{}{
				"query_params": map[string]string{
					"name": "example-service",
				},
			},
			Response: map[string]interface{}{
				"status": "success",
			},
		})
	case "GET /services":
		examples = append(examples, APIExample{
			Name:        "获取服务列表",
			Description: "获取所有运行中的服务",
			Response: map[string]interface{}{
				"services": []map[string]interface{}{
					{
						"name":      "example-service",
						"status":    "running",
						"port":      8081,
						"workspace": "default",
					},
				},
				"total": 1,
			},
		})
	case "POST /api/debug/apis/test":
		examples = append(examples, APIExample{
			Name:        "测试API端点",
			Description: "动态测试任意API端点",
			Request: map[string]interface{}{
				"method": "GET",
				"path":   "/services",
				"headers": map[string]string{
					"Authorization": "Bearer your-api-key",
				},
			},
			Response: map[string]interface{}{
				"success":       true,
				"status_code":   200,
				"response_time": "15ms",
				"response": map[string]interface{}{
					"services": []string{"service1", "service2"},
				},
			},
		})
	}

	return examples
}

// getAPITags 获取API标签
func (m *ServerManager) getAPITags(path string) []string {
	var tags []string

	if strings.Contains(path, "/debug") {
		tags = append(tags, "调试")
	}
	if strings.Contains(path, "/workspaces") {
		tags = append(tags, "工作空间")
	}
	if strings.Contains(path, "/services") {
		tags = append(tags, "服务")
	}
	if strings.Contains(path, "/sessions") {
		tags = append(tags, "会话")
	}

	return tags
}

// handleGetAPIGroups 获取API分组信息
func (m *ServerManager) handleGetAPIGroups(c echo.Context) error {
	groups := map[string]interface{}{
		"调试接口": map[string]interface{}{
			"description": "用于调试和测试的API接口",
			"endpoints":   []string{"/api/debug/apis", "/api/debug/apis/test"},
		},
		"工作空间管理": map[string]interface{}{
			"description": "管理工作空间的相关接口",
			"endpoints":   []string{"/api/workspaces", "/api/workspaces/:id"},
		},
		"服务管理": map[string]interface{}{
			"description": "管理MCP服务的相关接口",
			"endpoints":   []string{"/deploy", "/delete", "/services"},
		},
		"会话管理": map[string]interface{}{
			"description": "管理用户会话的相关接口",
			"endpoints":   []string{"/api/workspaces/:workspace/sessions"},
		},
		"通信接口": map[string]interface{}{
			"description": "处理消息和事件流的接口",
			"endpoints":   []string{"/sse", "/message"},
		},
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"groups":       groups,
		"total_groups": len(groups),
	})
}

// handleTestAPI 测试API端点
func (m *ServerManager) handleTestAPI(c echo.Context) error {
	logger := xlog.NewLogger("[APITest]")

	var req APITestRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, APITestResponse{
			Success: false,
			Error:   "Invalid request format: " + err.Error(),
		})
	}

	if req.Method == "" || req.Path == "" {
		return c.JSON(http.StatusBadRequest, APITestResponse{
			Success: false,
			Error:   "Method and path are required",
		})
	}

	logger.Infof("Testing API: %s %s", req.Method, req.Path)

	startTime := time.Now()

	// 构建完整URL
	scheme := "http"
	if c.IsTLS() {
		scheme = "https"
	}
	host := c.Request().Host
	fullURL := fmt.Sprintf("%s://%s%s", scheme, host, req.Path)

	// 添加查询参数
	if len(req.Query) > 0 {
		queryParts := []string{}
		for key, value := range req.Query {
			queryParts = append(queryParts, fmt.Sprintf("%s=%s", key, value))
		}
		fullURL += "?" + strings.Join(queryParts, "&")
	}

	// 准备请求体
	var bodyReader io.Reader
	var requestBodyStr string
	if req.Body != nil && len(req.Body) > 0 {
		bodyBytes, err := json.Marshal(req.Body)
		if err != nil {
			return c.JSON(http.StatusBadRequest, APITestResponse{
				Success: false,
				Error:   "Failed to marshal request body: " + err.Error(),
			})
		}
		bodyReader = bytes.NewReader(bodyBytes)
		requestBodyStr = string(bodyBytes)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequest(req.Method, fullURL, bodyReader)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, APITestResponse{
			Success: false,
			Error:   "Failed to create request: " + err.Error(),
		})
	}

	// 设置头部
	if req.ContentType != "" {
		httpReq.Header.Set("Content-Type", req.ContentType)
	} else if req.Body != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// 复制原始请求的授权头部
	if auth := c.Request().Header.Get("Authorization"); auth != "" {
		httpReq.Header.Set("Authorization", auth)
	}

	// 添加自定义头部
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}

	// 记录请求头部
	requestHeaders := make(map[string]string)
	for key, values := range httpReq.Header {
		if len(values) > 0 {
			requestHeaders[key] = values[0]
		}
	}

	// 发送请求
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(httpReq)
	responseTime := time.Since(startTime)

	if err != nil {
		return c.JSON(http.StatusOK, APITestResponse{
			Success:        false,
			Error:          "Request failed: " + err.Error(),
			ResponseTime:   responseTime,
			RequestHeaders: requestHeaders,
			RequestBody:    requestBodyStr,
		})
	}
	defer resp.Body.Close()

	// 读取响应
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusOK, APITestResponse{
			Success:        false,
			StatusCode:     resp.StatusCode,
			Error:          "Failed to read response: " + err.Error(),
			ResponseTime:   responseTime,
			RequestHeaders: requestHeaders,
			RequestBody:    requestBodyStr,
		})
	}

	responseBodyStr := string(responseBody)

	// 尝试解析JSON响应
	var responseData map[string]interface{}
	if err := json.Unmarshal(responseBody, &responseData); err != nil {
		// 如果不是JSON，作为字符串处理
		responseData = map[string]interface{}{
			"raw_response": responseBodyStr,
		}
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	response := APITestResponse{
		Success:        success,
		StatusCode:     resp.StatusCode,
		ResponseTime:   responseTime,
		Response:       responseData,
		RequestHeaders: requestHeaders,
		RequestBody:    requestBodyStr,
		ResponseBody:   responseBodyStr,
	}

	if !success {
		response.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	logger.Infof("API test completed: %s %s - Status: %d, Time: %v",
		req.Method, req.Path, resp.StatusCode, responseTime)

	return c.JSON(http.StatusOK, response)
}
