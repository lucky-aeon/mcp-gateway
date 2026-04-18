package config

type WorkspaceConfig struct {
	Servers map[string]MCPServerConfig `json:"servers"`
	McpServiceMgrConfig
	LogConfig
	CommandBase string `json:"commandBase"`
}

type LogConfig struct {
	Level uint8  `json:"level"`
	Path  string `json:"path"`
}

func (wcfg *WorkspaceConfig) AddMcpServerCfg(name string, mcpCfg MCPServerConfig) {
	wcfg.Servers[name] = mcpCfg
}

func (wcfg *WorkspaceConfig) GetMcpServerCfg(name string) (MCPServerConfig, bool) {
	mcpCfg, ok := wcfg.Servers[name]
	if !ok {
		return MCPServerConfig{}, false
	}

	// fill Global Config, if not set
	if mcpCfg.LogConfig.Path == "" {
		mcpCfg.LogConfig.Path = wcfg.LogConfig.Path
	}
	return mcpCfg, ok
}
