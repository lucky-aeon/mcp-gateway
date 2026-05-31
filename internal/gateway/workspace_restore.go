package gateway

import (
	"context"
	"fmt"
	"strings"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	serviceDesiredStatusKey     = "desired_status"
	serviceDesiredStatusStopped = "stopped"
)

func (h *Handler) ensureWorkspaceServicesRunning(ctx context.Context, workspaceID string, logger xlog.Logger) error {
	if h.auth == nil || !h.auth.IsSaaS() {
		return nil
	}

	h.restoreMu.Lock()
	defer h.restoreMu.Unlock()

	dbServers, err := h.auth.ListMCPServers(ctx, workspaceID)
	if err != nil {
		return err
	}
	for _, dbServer := range dbServers {
		if !serviceShouldAutoStart(dbServer.Config) {
			continue
		}
		cfg := serviceConfigFromMap(dbServer.Config, workspaceID)
		if _, err := h.services.DeployServer(logger, workspaces.NameArg{Workspace: workspaceID, Server: dbServer.Name}, cfg); err != nil {
			return fmt.Errorf("deploy %s/%s: %w", workspaceID, dbServer.Name, err)
		}
	}
	return nil
}

func serviceShouldAutoStart(raw map[string]interface{}) bool {
	if raw == nil {
		return true
	}
	status := strings.ToLower(strings.TrimSpace(asString(raw[serviceDesiredStatusKey])))
	return status != serviceDesiredStatusStopped
}

func serviceConfigFromMap(raw map[string]interface{}, workspaceID string) config.MCPServerConfig {
	cfg := config.MCPServerConfig{
		Workspace: workspaceID,
		Args:      []string{},
		Env:       map[string]string{},
	}
	if raw == nil {
		return cfg
	}
	cfg.URL = asString(raw["url"])
	cfg.Command = asString(raw["command"])
	cfg.Args = asStringSlice(raw["args"])
	cfg.Env = asStringMap(raw["env"])
	cfg.GatewayProtocol = asString(raw["gateway_protocol"])
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	return cfg
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

func asStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch items := v.(type) {
	case []string:
		return items
	case []interface{}:
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, asString(item))
		}
		return out
	case primitive.A:
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, asString(item))
		}
		return out
	default:
		return nil
	}
}

func asStringMap(v interface{}) map[string]string {
	if v == nil {
		return map[string]string{}
	}
	switch m := v.(type) {
	case map[string]string:
		return m
	case map[string]interface{}:
		out := make(map[string]string, len(m))
		for k, val := range m {
			out[k] = asString(val)
		}
		return out
	case primitive.M:
		out := make(map[string]string, len(m))
		for k, val := range m {
			out[k] = asString(val)
		}
		return out
	default:
		return map[string]string{}
	}
}
