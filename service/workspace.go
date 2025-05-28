package service

import (
	"fmt"
	"sync"

	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

type WorkSpace struct {
	Id  string
	cfg config.WorkspaceConfig

	// MCP
	servers      map[string]*McpService
	serversMutex sync.RWMutex

	// Other Mgr
	portManager PortManagerI
}

func NewWorkSpace(workId string, cfg config.WorkspaceConfig, portManager PortManagerI) *WorkSpace {
	space := &WorkSpace{Id: workId, cfg: cfg, portManager: portManager}

	return space
}

// AddMcpService adds a new MCP service to the workspace.
func (w *WorkSpace) AddMcpService(xl xlog.Logger, serviceName string, mcpConfig config.MCPServerConfig) error {
	xl.Infof("Adding MCP service %s", serviceName)

	// check if the service already exists in the workspace config
	if _, ok := w.cfg.GetMcpServerCfg(serviceName); ok {
		xl.Warnf("MCP service %s already exists in workspace config, skipping", serviceName)
		return fmt.Errorf("MCP service %s already exists in workspace config, skipping", serviceName)
	}
	w.cfg.AddMcpServerCfg(serviceName, mcpConfig)

	// create service instance
	instance := NewMcpService(serviceName, mcpConfig, w.portManager)
	if err := instance.Start(xl); err != nil {
		xl.Errorf("Failed to start service %s: %v", serviceName, err)
		return err
	}

	// add to workspace
	w.serversMutex.Lock()
	w.servers[serviceName] = instance
	w.serversMutex.Unlock()
	return nil
}

// GetMcpService returns the MCP service with the given name.
func (w *WorkSpace) GetMcpService(serviceName string) (ExportMcpService, error) {
	w.serversMutex.RLock()
	server, ok := w.servers[serviceName]
	w.serversMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("MCP service %s not found", serviceName)
	}
	return server, nil
}

// getMcpService returns the MCP service with the given name. It is used internally.
func (w *WorkSpace) getMcpService(serviceName string) (*McpService, error) {
	w.serversMutex.RLock()
	server, ok := w.servers[serviceName]
	w.serversMutex.RUnlock()
	if !ok {
		return nil, fmt.Errorf("MCP service %s not found", serviceName)
	}
	return server, nil
}

// RestartMcpService restarts the MCP service with the given name.
func (w *WorkSpace) RestartMcpService(xl xlog.Logger, serviceName string) error {
	xl.Infof("Restarting MCP service %s", serviceName)

	server, err := w.getMcpService(serviceName)
	if err != nil {
		return err
	}
	server.Restart(xl)
	return nil
}

// StopMcpService stops the MCP service with the given name.
func (w *WorkSpace) StopMcpService(xl xlog.Logger, serviceName string) error {
	xl.Infof("Stopping MCP service %s", serviceName)

	server, err := w.getMcpService(serviceName)
	if err != nil {
		return err
	}
	server.Stop(xl)
	return nil
}

// SetMcpServiceConfig sets the MCP service config.
func (w *WorkSpace) SetMcpServiceConfig(xl xlog.Logger, serviceName string, mcpConfig config.MCPServerConfig) error {
	xl.Infof("Setting MCP service %s config", serviceName)
	server, err := w.getMcpService(serviceName)
	if err != nil {
		return err
	}
	return server.setConfig(mcpConfig)
}
