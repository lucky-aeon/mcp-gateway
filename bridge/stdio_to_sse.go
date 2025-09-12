package bridge

import (
	"context"
	"fmt"

	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
	client "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	server "github.com/mark3labs/mcp-go/server"
)

// StdioToSSEBridge 创建一个将 stdio MCP 服务器桥接到 SSE 的转换器
type StdioToSSEBridge struct {
	stdioClient *client.Client
	mcpServer   *server.MCPServer
	*server.SSEServer
	mcpName string
	logger  xlog.Logger
}

func NewStdioToSSEBridge(ctx context.Context, transport *transport.Stdio, mcpName string) (*StdioToSSEBridge, error) {
	// 创建带有 mcpName 的专用 logger
	logger := xlog.NewLogger("bridge").With("mcp_name", mcpName)

	stdioClient := client.NewClient(transport)

	logger.Info("Starting stdio client", "mcp_name", mcpName)
	if err := stdioClient.Start(ctx); err != nil {
		logger.Error("Failed to start stdio client", "error", err)
		return nil, fmt.Errorf("failed to start stdio client: %w", err)
	}

	// 初始化 stdio 客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "mcp-stdio-sse-bridge",
		Version: "1.0.0",
	}

	initResult, err := stdioClient.Initialize(ctx, initRequest)
	if err != nil {
		logger.Error("Failed to initialize stdio client", "error", err)
		return nil, fmt.Errorf("failed to initialize stdio client: %w", err)
	}

	logger.Info("Connected to stdio server",
		"server_name", initResult.ServerInfo.Name,
		"server_version", initResult.ServerInfo.Version,
	)

	// 2. 创建 MCP 服务器，作为桥接层
	mcpServer := server.NewMCPServer(
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	bridge := &StdioToSSEBridge{
		stdioClient: stdioClient,
		mcpServer:   mcpServer,
		mcpName:     mcpName,
		logger:      logger,
	}

	// 3. 设置工具桥接
	if err := bridge.setupToolBridge(ctx); err != nil {
		bridge.logger.Warn("Failed to setup tool bridge", "error", err)
	}

	// 4. 设置资源桥接（如果支持的话）
	if err := bridge.setupResourceBridge(ctx); err != nil {
		bridge.logger.Warnf("Resource bridging failed (server may not support resources): %v", err)
		// 不返回错误，继续启动服务器
	}

	// 5. 创建 SSE 服务器包装 MCP 服务器
	sseServer := server.NewSSEServer(
		mcpServer,
		server.WithStaticBasePath(mcpName),
		server.WithSSEEndpoint("/sse"),
		server.WithMessageEndpoint("/message"),
	)

	bridge.SSEServer = sseServer

	return bridge, nil
}

// setupToolBridge 设置工具桥接
func (b *StdioToSSEBridge) setupToolBridge(ctx context.Context) error {
	// 获取 stdio 服务器的工具列表
	toolsRequest := mcp.ListToolsRequest{}
	toolsResult, err := b.stdioClient.ListTools(ctx, toolsRequest)
	if err != nil {
		b.logger.Error("Failed to list tools from stdio server", "error", err)
		return fmt.Errorf("failed to list tools from stdio server: %w", err)
	}

	b.logger.Info("Bridging tools from stdio server", "tool_count", len(toolsResult.Tools))

	// 为每个工具创建桥接
	for _, tool := range toolsResult.Tools {
		// 复制工具定义
		bridgedTool := tool
		toolName := tool.Name

		b.logger.Debug("Bridging tool", "tool_name", toolName)

		// 创建工具处理器，将调用转发到 stdio 客户端
		b.mcpServer.AddTool(bridgedTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			b.logger.Debug("Calling tool", "tool_name", toolName)

			// 转发工具调用到 stdio 服务器
			result, err := b.stdioClient.CallTool(ctx, request)
			if err != nil {
				b.logger.Error("Tool call failed", "tool_name", toolName, "error", err)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to call tool %s: %v", toolName, err)), nil
			}

			b.logger.Debug("Tool call succeeded", "tool_name", toolName, result)
			return result, nil
		})
	}

	return nil
}

// setupResourceBridge 设置资源桥接
func (b *StdioToSSEBridge) setupResourceBridge(ctx context.Context) error {
	// 获取 stdio 服务器的资源列表
	resourcesRequest := mcp.ListResourcesRequest{}
	resourcesResult, err := b.stdioClient.ListResources(ctx, resourcesRequest)
	if err != nil {
		return fmt.Errorf("failed to list resources from stdio server: %w", err)
	}

	b.logger.Info("Bridging resources from stdio server", "resource_count", len(resourcesResult.Resources))

	// 为每个资源创建桥接
	for _, resource := range resourcesResult.Resources {
		// 复制资源定义
		bridgedResource := resource
		resourceURI := resource.URI

		b.logger.Debug("Bridging resource", "resource_uri", resourceURI)

		// 创建资源处理器，将请求转发到 stdio 客户端
		b.mcpServer.AddResource(bridgedResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			b.logger.Debug("Reading resource", "resource_uri", resourceURI)

			// 转发资源读取请求到 stdio 服务器
			result, err := b.stdioClient.ReadResource(ctx, request)
			if err != nil {
				b.logger.Error("Resource read failed", "resource_uri", resourceURI, "error", err)
				return nil, fmt.Errorf("failed to read resource %s: %w", resourceURI, err)
			}

			b.logger.Debug("Resource read succeeded", "resource_uri", resourceURI, result)
			return result.Contents, nil
		})
	}

	// 获取资源模板
	templatesRequest := mcp.ListResourceTemplatesRequest{}
	templatesResult, err := b.stdioClient.ListResourceTemplates(ctx, templatesRequest)
	if err != nil {
		b.logger.Error("Failed to list resource templates from stdio server", "error", err)
		return fmt.Errorf("failed to list resource templates from stdio server: %w", err)
	}

	b.logger.Info("Bridging resource templates from stdio server", "template_count", len(templatesResult.ResourceTemplates))

	// 为每个资源模板创建桥接
	for _, template := range templatesResult.ResourceTemplates {
		// 复制模板定义
		bridgedTemplate := template
		templateURI := template.URITemplate

		b.logger.Debug("Bridging resource template", "template_uri", templateURI)

		// 创建模板处理器
		b.mcpServer.AddResourceTemplate(bridgedTemplate, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			b.logger.Debug("Reading resource template", "template_uri", templateURI)

			// 转发资源读取请求到 stdio 服务器
			result, err := b.stdioClient.ReadResource(ctx, request)
			if err != nil {
				b.logger.Error("Resource template read failed", "template_uri", templateURI, "error", err)
				return nil, fmt.Errorf("failed to read resource template %+v: %w", templateURI, err)
			}

			b.logger.Debug("Resource template read succeeded", "template_uri", templateURI, result)
			return result.Contents, nil
		})
	}
	return nil
}

// StartSSEServer 启动 SSE 服务器
func (b *StdioToSSEBridge) Start(addr string) error {
	b.logger.Info("Starting SSE bridge server", "address", addr)

	if err := b.Ping(context.Background()); err != nil {
		b.logger.Error("Failed to ping stdio server", "error", err)
		return fmt.Errorf("failed to ping stdio server: %w", err)
	}

	b.logger.Info("SSE bridge server started successfully", "address", addr)
	return b.SSEServer.Start(addr)
}

// Close 关闭桥接器
func (b *StdioToSSEBridge) Close() error {
	b.logger.Info("Closing SSE bridge")

	if b.stdioClient != nil {
		b.stdioClient.Close()
		b.logger.Debug("Stdio client closed")
	}

	err := b.SSEServer.Shutdown(context.Background())
	if err != nil {
		b.logger.Error("Failed to shutdown SSE server", "error", err)
		return fmt.Errorf("failed to shutdown SSE server: %w", err)
	}

	b.logger.Info("SSE bridge closed successfully")
	return nil
}

func (b *StdioToSSEBridge) Ping(ctx context.Context) error {
	if b.stdioClient == nil {
		return fmt.Errorf("stdio client is not initialized")
	}
	return b.stdioClient.Ping(ctx)
}
