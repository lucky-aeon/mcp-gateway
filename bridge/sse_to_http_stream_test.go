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

func TestSSEToHTTPStreamBridge(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	pwd += "/testdata"
	_ = os.Mkdir(pwd, 0755)
	_ = os.WriteFile(pwd+"/test_ssetostream.txt", []byte("Hello, World from SSE Bridge!"), 0644)

	// 1. 首先启动一个 SSE 服务器作为上游服务器
	// 创建 stdio 客户端连接到文件系统服务器
	stdioTransport := transport.NewStdio(
		"npx",
		nil, // 环境变量
		"-y",
		"@modelcontextprotocol/server-filesystem",
		pwd,
	)

	t.Log("Creating upstream SSE bridge")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	upstreamBridge, err := NewStdioToSSEBridge(ctx, stdioTransport, "filesystem")
	if err != nil {
		t.Fatalf("Failed to create upstream SSE bridge: %v", err)
	}

	// 启动上游 SSE 服务器
	upstreamStarted := make(chan error, 1)
	go func() {
		t.Log("Starting upstream SSE server on :8082...")
		if err := upstreamBridge.Start(":8082"); err != nil && err.Error() != "http: Server closed" {
			upstreamStarted <- err
		}
		close(upstreamStarted)
	}()

	t.Log("Waiting for upstream SSE server to start...")
	time.Sleep(2 * time.Second)

	// 2. 现在创建 SSE to HTTP Stream 桥接器
	t.Log("Creating SSE to HTTP Stream bridge")
	ctx2, cancel2 := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel2()

	bridge, err := NewSSEToHTTPStreamBridge(ctx2, "http://localhost:8082/filesystem/sse", "filesystem-bridge")
	if err != nil {
		t.Fatalf("Failed to create SSE to HTTP Stream bridge: %v", err)
	}

	// 启动 HTTP Stream 服务器
	bridgeStarted := make(chan error, 1)
	go func() {
		t.Log("Starting HTTP Stream bridge server on :8083...")
		if err := bridge.Start(":8083"); err != nil && err.Error() != "http: Server closed" {
			bridgeStarted <- err
		}
		close(bridgeStarted)
	}()

	t.Log("Waiting for HTTP Stream bridge server to start...")
	time.Sleep(2 * time.Second)

	// 3. 创建 HTTP Stream 客户端连接到我们的桥接器
	httpStreamTransport, err := transport.NewStreamableHTTP("http://localhost:8083/filesystem-bridge")
	if err != nil {
		t.Fatalf("Failed to create HTTP Stream transport: %v", err)
	}

	c := client.NewClient(httpStreamTransport)

	// 启动客户端
	ctx3, cancel3 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel3()
	_ = c.Start(ctx3)

	t.Log("Connecting to SSE-to-HTTP Stream bridge...")

	// 初始化客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-sse-to-http-stream-client",
		Version: "1.0.0",
	}

	initResult, err := c.Initialize(ctx3, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	defer func() {
		_ = c.Close()
		t.Log("Closing bridges...")
		_ = bridge.Close()
		_ = upstreamBridge.Close()

		// 等待服务器正常关闭
		select {
		case err := <-bridgeStarted:
			if err != nil {
				t.Logf("Bridge server error: %v", err)
			}
		case <-time.After(1 * time.Second):
		}

		select {
		case err := <-upstreamStarted:
			if err != nil {
				t.Logf("Upstream server error: %v", err)
			}
		case <-time.After(1 * time.Second):
		}
	}()

	t.Logf("Connected to bridge server: %s %s",
		initResult.ServerInfo.Name,
		initResult.ServerInfo.Version,
	)

	// 4. 测试工具桥接
	t.Log("Listing available tools...")
	toolsRequest := mcp.ListToolsRequest{}
	toolsResult, err := c.ListTools(ctx3, toolsRequest)
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

		result, err := c.CallTool(ctx3, callRequest)
		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		} else {
			t.Logf("Tool result: %+v", result.Content)
		}

		t.Log("Testing read_file tool...")
		callRequest = mcp.CallToolRequest{}
		callRequest.Params.Name = "read_file"
		callRequest.Params.Arguments = map[string]any{
			"path": pwd + "/test_ssetostream.txt",
		}

		result, err = c.CallTool(ctx3, callRequest)
		if err != nil {
			t.Fatalf("Failed to call tool: %v", err)
		} else {
			t.Logf("Tool result: %+v", result.Content)
		}
	}

	// 5. 测试资源桥接（如果支持的话）
	t.Log("Listing available resources...")
	resourcesRequest := mcp.ListResourcesRequest{}
	resourcesResult, err := c.ListResources(ctx3, resourcesRequest)
	if err != nil {
		t.Logf("Resource listing not supported or failed (this is okay): %v", err)
	} else {
		t.Logf("Found %d resources:", len(resourcesResult.Resources))
		for _, resource := range resourcesResult.Resources {
			t.Logf("- %s: %s", resource.URI, resource.Name)
		}
	}

	// 6. 测试提示桥接（如果支持的话）
	t.Log("Listing available prompts...")
	promptsRequest := mcp.ListPromptsRequest{}
	promptsResult, err := c.ListPrompts(ctx3, promptsRequest)
	if err != nil {
		t.Logf("Prompt listing not supported or failed (this is okay): %v", err)
	} else {
		t.Logf("Found %d prompts:", len(promptsResult.Prompts))
		for _, prompt := range promptsResult.Prompts {
			t.Logf("- %s: %s", prompt.Name, prompt.Description)
		}
	}

	t.Log("SSE to HTTP Stream bridge test completed successfully!")
}
