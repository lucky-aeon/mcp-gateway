package service

import (
	"fmt"
	"sync"

	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

type (
	WorkSpaceStatus string
)

const (
	WorkSpaceStatusRunning WorkSpaceStatus = "running"
	WorkSpaceStatusStopped WorkSpaceStatus = "stopped"
)

type WorkSpace struct {
	Id     string
	cfg    config.WorkspaceConfig
	status WorkSpaceStatus

	// MCP
	servers      map[string]*McpService
	serversMutex sync.RWMutex

	// Other Mgr
	portManager PortManagerI
	sessionMgr  *SessionManager
}

func NewWorkSpace(workId string, cfg config.WorkspaceConfig, portManager PortManagerI) *WorkSpace {
	space := &WorkSpace{Id: workId, cfg: cfg, portManager: portManager, servers: make(map[string]*McpService)}
	// init session manager, it will be used to create session for each workspace
	space.sessionMgr = NewSessionManager(space)
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
	return w.getMcpService(serviceName)
}

// GetMcpServices returns all MCP services in the workspace.
func (w *WorkSpace) GetMcpServices() map[string]ExportMcpService {
	services := w.getMcpServices()
	exportServices := make(map[string]ExportMcpService)
	for name, service := range services {
		exportServices[name] = service
	}
	return exportServices
}

func (w *WorkSpace) getMcpServices() map[string]*McpService {
	services := make(map[string]*McpService)
	w.serversMutex.RLock()
	for name, service := range w.servers {
		services[name] = service
	}
	w.serversMutex.RUnlock()
	return services
}

// getMcpService returns the MCP service with the given name. It is used internally.
func (w *WorkSpace) getMcpService(serviceName string) (*McpService, error) {
	if w.status != WorkSpaceStatusRunning {
		return nil, fmt.Errorf("workspace is not running, cannot get MCP service %s", serviceName)
	}

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

// RemoveMcpService removes the MCP service with the given name.
func (w *WorkSpace) RemoveMcpService(xl xlog.Logger, serviceName string) error {
	xl.Infof("Removing MCP service %s", serviceName)
	if err := w.StopMcpService(xl, serviceName); err != nil {
		return err
	}

	w.serversMutex.Lock()
	delete(w.servers, serviceName)
	w.serversMutex.Unlock()
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

// Close stops all MCP services in the workspace.
func (w *WorkSpace) Close(xl xlog.Logger) {
	// 获取服务列表的快照
	w.serversMutex.RLock()
	servers := make([]*McpService, 0, len(w.servers))
	for _, server := range w.servers {
		servers = append(servers, server)
	}
	w.serversMutex.RUnlock()

	// 在释放锁后关闭服务
	for _, server := range servers {
		if err := w.RemoveMcpService(xl, server.Name); err != nil {
			xl.Errorf("Failed to remove MCP service %s: %v", server.Name, err)
		}
	}
	w.status = WorkSpaceStatusStopped
}
