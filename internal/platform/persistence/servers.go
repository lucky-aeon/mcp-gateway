// Package persistence 负责把 MCP 部署状态落到磁盘、启动时回放。
package persistence

import (
	"encoding/json"
	"os"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
)

// DeployFunc 是 LoadAndDeployServers 的部署回调签名。
// 由调用方（通常是 admin.Handler）适配自己的 DeployServer 实现。
type DeployFunc func(name string, cfg config.MCPServerConfig) error

// LoadAndDeployServers 从 cfg.GetMcpConfigPath() 读取已持久化的 MCP 部署清单，
// 并异步调用 deploy 回调逐个恢复。
// 文件不存在（首次启动）返回 nil，不视作错误。
func LoadAndDeployServers(cfg config.Config, deploy DeployFunc) error {
	xl := xlog.NewLogger("[persistence]")
	data, err := os.ReadFile(cfg.GetMcpConfigPath())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var servers map[string]config.MCPServerConfig
	if err := json.Unmarshal(data, &servers); err != nil {
		return err
	}

	xl.Infof("Async loading %d servers", len(servers))
	go func() {
		for name, srv := range servers {
			xl.Infof("Loading server %s: %+v", name, srv)
			if err := deploy(name, srv); err != nil {
				xl.Errorf("Error deploying server %s: %v", name, err)
			}
		}
		xl.Infof("Loaded %d servers", len(servers))
	}()
	return nil
}
