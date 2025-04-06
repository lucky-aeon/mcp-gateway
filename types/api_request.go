package types

import "github.com/lucky-aeon/agentx/plugin-helper/config"

// DeployRequest 部署请求结构
type DeployRequest struct {
	MCPServers map[string]config.MCPServerConfig `json:"mcpServers"`
}
