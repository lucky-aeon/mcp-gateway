package sessions

import (
	"os"
	"testing"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"
)

// mockMcpServiceFileSystem 返回一个可启动的 filesystem MCP，用于 sessions 集成测试。
// 之前在同一个 service 包，拆包后保留在 sessions test 内。
func mockMcpServiceFileSystem(t *testing.T) *runtime.McpService {
	t.Helper()
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	pwd += "/testdata"
	_ = os.Mkdir(pwd, 0755)
	_ = os.WriteFile(pwd+"/test.txt", []byte("Hello, World!"), 0644)
	return runtime.NewMcpService("fileSystem", config.MCPServerConfig{
		Workspace: "default",
		Command:   "npx",
		Args: []string{
			"-y",
			"@modelcontextprotocol/server-filesystem",
			pwd,
		},
	}, runtime.NewPortManager())
}
