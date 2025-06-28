package service

import (
	"os"
	"testing"

	"github.com/lucky-aeon/agentx/plugin-helper/config"
)

var mockPortMgr PortManagerI = NewPortManager()

func mockMcpServiceFileSystem(t *testing.T) *McpService {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	pwd += "/testdata"
	os.Mkdir(pwd, 0755)
	os.WriteFile(pwd+"/test.txt", []byte("Hello, World!"), 0644)
	return NewMcpService("fileSystem", config.MCPServerConfig{
		Workspace: "default",
		Command:   "npx",
		Args: []string{
			"-y",
			"@modelcontextprotocol/server-filesystem",
			pwd,
		},
	}, mockPortMgr)
}
