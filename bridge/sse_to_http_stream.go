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

// SSEToHTTPStreamBridge 创建一个将 SSE MCP 服务器桥接到 HTTP Stream 的转换器
type SSEToHTTPStreamBridge struct {
	sseClient *client.Client
	mcpServer *server.MCPServer
	*server.StreamableHTTPServer
	mcpName string
	logger  xlog.Logger
}

func NewSSEToHTTPStreamBridge(ctx context.Context, sseBaseURL string, mcpName string, options ...transport.ClientOption) (*SSEToHTTPStreamBridge, error) {
	// 创建带有 mcpName 的专用 logger
	logger := xlog.NewLogger("bridge").With("mcp_name", mcpName)

	// 创建 SSE transport
	sseTransport, err := transport.NewSSE(sseBaseURL, options...)
	if err != nil {
		logger.Error("Failed to create SSE transport", "error", err, "base_url", sseBaseURL)
		return nil, fmt.Errorf("failed to create SSE transport: %w", err)
	}

	sseClient := client.NewClient(sseTransport)

	logger.Info("Starting SSE client", "mcp_name", mcpName, "base_url", sseBaseURL)
	if err := sseClient.Start(ctx); err != nil {
		logger.Error("Failed to start SSE client", "error", err)
		return nil, fmt.Errorf("failed to start SSE client: %w", err)
	}

	// 初始化 SSE 客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "mcp-sse-http-stream-bridge",
		Version: "1.0.0",
	}

	initResult, err := sseClient.Initialize(ctx, initRequest)
	if err != nil {
		logger.Error("Failed to initialize SSE client", "error", err)
		return nil, fmt.Errorf("failed to initialize SSE client: %w", err)
	}

	logger.Info("Connected to SSE server",
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

	bridge := &SSEToHTTPStreamBridge{
		sseClient: sseClient,
		mcpServer: mcpServer,
		mcpName:   mcpName,
		logger:    logger,
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

	// 5. 设置提示桥接（如果支持的话）
	if err := bridge.setupPromptBridge(ctx); err != nil {
		bridge.logger.Warnf("Prompt bridging failed (server may not support prompts): %v", err)
		// 不返回错误，继续启动服务器
	}

	// 6. 创建 StreamableHTTP 服务器包装 MCP 服务器
	httpStreamServer := server.NewStreamableHTTPServer(
		mcpServer,
		server.WithEndpointPath(fmt.Sprintf("/%s", mcpName)),
		server.WithStateLess(false), // 保持会话状态以支持实时通信
	)

	bridge.StreamableHTTPServer = httpStreamServer

	return bridge, nil
}

// setupToolBridge 设置工具桥接
func (b *SSEToHTTPStreamBridge) setupToolBridge(ctx context.Context) error {
	// 获取 SSE 服务器的工具列表
	toolsRequest := mcp.ListToolsRequest{}
	toolsResult, err := b.sseClient.ListTools(ctx, toolsRequest)
	if err != nil {
		b.logger.Error("Failed to list tools from SSE server", "error", err)
		return fmt.Errorf("failed to list tools from SSE server: %w", err)
	}

	b.logger.Info("Bridging tools from SSE server", "tool_count", len(toolsResult.Tools))

	// 为每个工具创建桥接
	for _, tool := range toolsResult.Tools {
		// 复制工具定义
		bridgedTool := tool
		toolName := tool.Name

		b.logger.Debug("Bridging tool", "tool_name", toolName)

		// 创建工具处理器，将调用转发到 SSE 客户端
		b.mcpServer.AddTool(bridgedTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			b.logger.Debug("Calling tool", "tool_name", toolName)

			// 转发工具调用到 SSE 服务器
			result, err := b.sseClient.CallTool(ctx, request)
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
func (b *SSEToHTTPStreamBridge) setupResourceBridge(ctx context.Context) error {
	// 获取 SSE 服务器的资源列表
	resourcesRequest := mcp.ListResourcesRequest{}
	resourcesResult, err := b.sseClient.ListResources(ctx, resourcesRequest)
	if err != nil {
		return fmt.Errorf("failed to list resources from SSE server: %w", err)
	}

	b.logger.Info("Bridging resources from SSE server", "resource_count", len(resourcesResult.Resources))

	// 为每个资源创建桥接
	for _, resource := range resourcesResult.Resources {
		// 复制资源定义
		bridgedResource := resource
		resourceURI := resource.URI

		b.logger.Debug("Bridging resource", "resource_uri", resourceURI)

		// 创建资源处理器，将请求转发到 SSE 客户端
		b.mcpServer.AddResource(bridgedResource, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			b.logger.Debug("Reading resource", "resource_uri", resourceURI)

			// 转发资源读取请求到 SSE 服务器
			result, err := b.sseClient.ReadResource(ctx, request)
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
	templatesResult, err := b.sseClient.ListResourceTemplates(ctx, templatesRequest)
	if err != nil {
		b.logger.Error("Failed to list resource templates from SSE server", "error", err)
		return fmt.Errorf("failed to list resource templates from SSE server: %w", err)
	}

	b.logger.Info("Bridging resource templates from SSE server", "template_count", len(templatesResult.ResourceTemplates))

	// 为每个资源模板创建桥接
	for _, template := range templatesResult.ResourceTemplates {
		// 复制模板定义
		bridgedTemplate := template
		templateURI := template.URITemplate

		b.logger.Debug("Bridging resource template", "template_uri", templateURI)

		// 创建模板处理器
		b.mcpServer.AddResourceTemplate(bridgedTemplate, func(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			b.logger.Debug("Reading resource template", "template_uri", templateURI)

			// 转发资源读取请求到 SSE 服务器
			result, err := b.sseClient.ReadResource(ctx, request)
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

// setupPromptBridge 设置提示桥接
func (b *SSEToHTTPStreamBridge) setupPromptBridge(ctx context.Context) error {
	// 获取 SSE 服务器的提示列表
	promptsRequest := mcp.ListPromptsRequest{}
	promptsResult, err := b.sseClient.ListPrompts(ctx, promptsRequest)
	if err != nil {
		return fmt.Errorf("failed to list prompts from SSE server: %w", err)
	}

	b.logger.Info("Bridging prompts from SSE server", "prompt_count", len(promptsResult.Prompts))

	// 为每个提示创建桥接
	for _, prompt := range promptsResult.Prompts {
		// 复制提示定义
		bridgedPrompt := prompt
		promptName := prompt.Name

		b.logger.Debug("Bridging prompt", "prompt_name", promptName)

		// 创建提示处理器，将请求转发到 SSE 客户端
		b.mcpServer.AddPrompt(bridgedPrompt, func(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
			b.logger.Debug("Getting prompt", "prompt_name", promptName)

			// 转发提示请求到 SSE 服务器
			result, err := b.sseClient.GetPrompt(ctx, request)
			if err != nil {
				b.logger.Error("Prompt get failed", "prompt_name", promptName, "error", err)
				return nil, fmt.Errorf("failed to get prompt %s: %w", promptName, err)
			}

			b.logger.Debug("Prompt get succeeded", "prompt_name", promptName, result)
			return result, nil
		})
	}

	return nil
}

// Start 启动 HTTP Stream 服务器
func (b *SSEToHTTPStreamBridge) Start(addr string) error {
	b.logger.Info("Starting HTTP Stream bridge server", "address", addr)

	if err := b.Ping(context.Background()); err != nil {
		b.logger.Error("Failed to ping SSE server", "error", err)
		return fmt.Errorf("failed to ping SSE server: %w", err)
	}

	b.logger.Info("HTTP Stream bridge server started successfully", "address", addr)
	return b.StreamableHTTPServer.Start(addr)
}

// Close 关闭桥接器
func (b *SSEToHTTPStreamBridge) Close() error {
	b.logger.Info("Closing HTTP Stream bridge")

	if b.sseClient != nil {
		if err := b.sseClient.Close(); err != nil {
			b.logger.Error("Failed to close SSE client", "error", err)
		}
		b.logger.Debug("SSE client closed")
	}

	err := b.StreamableHTTPServer.Shutdown(context.Background())
	if err != nil {
		b.logger.Error("Failed to shutdown HTTP Stream server", "error", err)
		return fmt.Errorf("failed to shutdown HTTP Stream server: %w", err)
	}

	b.logger.Info("HTTP Stream bridge closed successfully")
	return nil
}

func (b *SSEToHTTPStreamBridge) Ping(ctx context.Context) error {
	if b.sseClient == nil {
		return fmt.Errorf("SSE client is not initialized")
	}
	return b.sseClient.Ping(ctx)
}
