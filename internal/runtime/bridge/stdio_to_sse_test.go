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

func TestSSEClient(t *testing.T) {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	pwd += "/testdata"
	os.Mkdir(pwd, 0755)
	os.WriteFile(pwd+"/test.txt", []byte("Hello, World!"), 0644)

	// 1. 创建 stdio 客户端连接到文件系统服务器
	stdioTransport := transport.NewStdio(
		"npx",
		nil, // 环境变量
		"-y",
		"@modelcontextprotocol/server-filesystem",
		pwd,
	)

	t.Log("createBridge")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	bridge, err := NewStdioToSSEBridge(ctx, stdioTransport, "filesystem")
	if err != nil {
		t.Fatalf("Failed to create bridge: %v", err)
	}

	// 启动 SSE 服务器
	go func() {
		t.Log("Starting SSE server on :8080...")
		if err := bridge.Start(":8080"); err != nil {
			t.Errorf("Failed to start SSE server: %v", err)
		}
	}()

	t.Log("wait server starting....")
	time.Sleep(1 * time.Second)
	t.Log("server started")

	// 创建 SSE 客户端连接到我们的桥接器
	sseTransport, err := transport.NewSSE("http://localhost:8080/filesystem/sse")
	if err != nil {
		t.Fatalf("Failed to create SSE transport: %v", err)
	}

	c := client.NewClient(sseTransport)

	// 启动客户端
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()
	c.Start(ctx2)

	t.Log("Connecting to SSE bridge...")

	// 初始化客户端
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-sse-client",
		Version: "1.0.0",
	}

	initResult, err := c.Initialize(ctx2, initRequest)
	if err != nil {
		t.Fatalf("Failed to initialize: %v", err)
	}

	defer func() {
		c.Close()
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

	t.Log("Test completed successfully!")
}
