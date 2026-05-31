package gateway

import "github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"

const remoteOAuthAccessTokenEnv = "MCP_REMOTE_AUTH_ACCESS_TOKEN"

func downstreamOAuthToken(instance runtime.ExportMcpService) string {
	info := instance.Info()
	if info.Config.Env == nil {
		return ""
	}
	return info.Config.Env[remoteOAuthAccessTokenEnv]
}
