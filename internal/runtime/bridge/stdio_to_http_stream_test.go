package bridge

import (
	"context"
	"os"
	"testing"
	"time"

	client "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestHTTPStreamClient(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	pwd += "/testdata"
	_ = os.Mkdir(pwd, 0755)
	_ = os.WriteFile(pwd+"/test.txt", []byte("Hello, World!"), 0644)

	// 1. 创建 stdio 客户端连接到文件系统服务器
	stdioTransport := transport.NewStdio(
		"npx",
		nil, // 环境变量
		"-y",
		"@modelcontextprotocol/server-filesystem",
		pwd,
	)

	t.Log("createHTTPStreamBridge")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bridge, err := NewStdioToHTTPStreamBridge(ctx, stdioTransport, "filesystem")
	if err != nil {
		t.Fatalf("Failed to create bridge: %v", err)
	}

	// 启动 HTTP Stream 服务器
	go func() {
		t.Log("Starting HTTP Stream server on :8081...")
		if err := bridge.Start(":8081"); err != nil {
			t.Errorf("Failed to start HTTP Stream server: %v", err)
		}
	}()

	t.Log("wait server starting....")
	time.Sleep(1 * time.Second)
	t.Log("server started")

	// 创建 HTTP Stream 客户端连接到我们的桥接器
	httpStreamTransport, err := transport.NewStreamableHTTP("http://localhost:8081/filesystem")
	if err != nil {
		t.Fatalf("Failed to create HTTP Stream transport: %v", err)
	}

	c := client.NewClient(httpStreamTransport)

	// 启动客户端
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()
	_ = c.Start(ctx2)

	t.Log("Connecting to HTTP Stream bridge...")

	// 初始化客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-http-stream-client",
		Version: "1.0.0",
	}

	initResult, err := c.Initialize(ctx2, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	defer func() {
		_ = c.Close()
	}()

	t.Logf("Connected to bridge server: %s %s",
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
	)

	// 列出可用工具
	t.Log("Listing available tools...")
	toolsRequest := mcp.ListToolsRequest{}
	toolsResult, err := c.ListTools(ctx2, toolsRequest)
	if err != nil {
		t.Fatalf("Failed to list tools: %v", err)
	}

	t.Logf("Found %d tools:", len(toolsResult.Tools))
	for _, tool := range toolsResult.Tools {
		t.Logf("- %s: %s", tool.Name, tool.Description)
	}

	// 测试调用一个工具
	if len(toolsResult.Tools) > 0 {
		// 测试 list_directory 工具
		t.Log("Testing list_directory tool...")
		callRequest := mcp.CallToolRequest{}
		callRequest.Params.Name = "list_directory"
		callRequest.Params.Arguments = map[string]any{
			"path": pwd,
		}

		result, err := c.CallTool(ctx2, callRequest)
		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		} else {
			t.Logf("Tool result: %+v", result.Content)
		}

		t.Log("Testing look test.txt file content...")
		callRequest = mcp.CallToolRequest{}
		callRequest.Params.Name = "read_file"
		callRequest.Params.Arguments = map[string]any{
			"path": pwd + "/test.txt",
		}

		result, err = c.CallTool(ctx2, callRequest)
		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		} else {
			t.Logf("Tool result: %+v", result.Content)
		}
	}

	// 测试资源桥接（如果支持的话）
	t.Log("Listing available resources...")
	resourcesRequest := mcp.ListResourcesRequest{}
	resourcesResult, err := c.ListResources(ctx2, resourcesRequest)
	if err != nil {
		t.Logf("Resource listing not supported or failed (this is okay): %v", err)
	} else {
		t.Logf("Found %d resources:", len(resourcesResult.Resources))
		for _, resource := range resourcesResult.Resources {
			t.Logf("- %s: %s", resource.URI, resource.Name)
		}
	}

	t.Log("HTTP Stream test completed successfully!")
}
