package config

// MCPServerConfig 定义单个MCP服务器的配置
type MCPServerConfig struct {
	Workspace string            `json:"workspace,omitempty"`
	URL       string            `json:"url,omitempty"`
	Command   string            `json:"command,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Env       map[string]string `json:"env,omitempty"`

	LogConfig
	McpServiceMgrConfig
}

func (c *MCPServerConfig) GetEnvs() []string {
	list := make([]string, 0, len(c.Env))
	for s := range c.Env {
		list = append(list, s)
	}
	return list
}
