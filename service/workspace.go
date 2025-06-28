package service

import (
	"fmt"
	"sync"

	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

const (
	DefaultWorkspace = "default"
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

// AddMcpServiceResult 表示添加服务的操作结果类型
type AddMcpServiceResult string

const (
	AddMcpServiceResultDeployed AddMcpServiceResult = "deployed" // 新部署成功
	AddMcpServiceResultExisted  AddMcpServiceResult = "existed"  // 已存在且运行中
	AddMcpServiceResultReplaced AddMcpServiceResult = "replaced" // 替换了停止/失败的服务
)

// AddMcpService adds a new MCP service to the workspace.
// Returns the operation result type and error if any.
func (w *WorkSpace) AddMcpService(xl xlog.Logger, serviceName string, mcpConfig config.MCPServerConfig) (AddMcpServiceResult, error) {
	xl.Infof("Adding MCP service %s", serviceName)

	// check if the service already exists
	w.serversMutex.RLock()
	existingService, serviceExists := w.servers[serviceName]
	w.serversMutex.RUnlock()

	if serviceExists {
		// service exists, check its status
		status := existingService.GetStatus()
		xl.Infof("Service %s already exists with status: %s", serviceName, status)

		switch status {
		case Running, Starting:
			// service is running or starting, skip deployment
			xl.Infof("Service %s is running/starting, skipping deployment", serviceName)
			return AddMcpServiceResultExisted, nil

		case Stopped, Failed:
			// service is stopped or failed, remove and redeploy
			xl.Infof("Service %s is stopped/failed, removing and redeploying", serviceName)
			if err := w.removeMcpServiceInternal(xl, serviceName); err != nil {
				xl.Errorf("Failed to remove existing service %s: %v", serviceName, err)
				return "", fmt.Errorf("failed to remove existing service: %v", err)
			}
			// continue to deploy new service
		}
	}

	// add to workspace config
	w.cfg.AddMcpServerCfg(serviceName, mcpConfig)

	// create service instance
	instance := NewMcpService(serviceName, mcpConfig, w.portManager)
	if err := instance.Start(xl); err != nil {
		xl.Errorf("Failed to start service %s: %v", serviceName, err)
		return "", err
	}

	// add to workspace
	w.serversMutex.Lock()
	defer w.serversMutex.Unlock()
	w.servers[serviceName] = instance

	if serviceExists {
		return AddMcpServiceResultReplaced, nil
	}
	return AddMcpServiceResultDeployed, nil
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

func (w *WorkSpace) GetStatus() WorkSpaceStatus {
	w.serversMutex.RLock()
	defer w.serversMutex.RUnlock()
	return w.status
}

func (w *WorkSpace) UpdateStatus(status WorkSpaceStatus) {
	w.serversMutex.Lock()
	w.status = status
	w.serversMutex.Unlock()
}

// getMcpService returns the MCP service with the given name. It is used internally.
func (w *WorkSpace) getMcpService(serviceName string) (*McpService, error) {

	if w.GetStatus() != WorkSpaceStatusRunning {
		if len(w.servers) == 0 {
			return nil, fmt.Errorf("workspace is not running, cannot get MCP service %s", serviceName)
		}
		w.UpdateStatus(WorkSpaceStatusRunning)
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
	return w.removeMcpServiceInternal(xl, serviceName)
}

// removeMcpServiceInternal removes the MCP service with the given name.
// This internal method handles the actual removal logic.
func (w *WorkSpace) removeMcpServiceInternal(xl xlog.Logger, serviceName string) error {
	xl.Infof("Removing MCP service %s", serviceName)

	// 先获取服务引用并停止服务
	w.serversMutex.RLock()
	server, exists := w.servers[serviceName]
	w.serversMutex.RUnlock()

	if !exists {
		return fmt.Errorf("MCP service %s not found", serviceName)
	}

	// 在锁外停止服务，避免死锁
	server.Stop(xl)

	// 最后从map中删除
	w.serversMutex.Lock()
	defer w.serversMutex.Unlock()
	delete(w.servers, serviceName)

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
	xl.Infof("Closing workspace %s", w.Id)

	// 持续循环直到所有服务都被移除，避免快照过期问题
	for {
		w.serversMutex.Lock()
		if len(w.servers) == 0 {
			w.status = WorkSpaceStatusStopped
			w.serversMutex.Unlock()
			break
		}

		// 获取第一个服务名称（在锁内安全获取）
		var serverName string
		for name := range w.servers {
			serverName = name
			break
		}
		w.serversMutex.Unlock()

		// 在锁外调用 RemoveMcpService 避免死锁
		if err := w.removeMcpServiceInternal(xl, serverName); err != nil {
			xl.Errorf("Failed to remove MCP service %s: %v", serverName, err)
			// 即使失败也要从map中删除，避免无限循环
			w.serversMutex.Lock()
			defer w.serversMutex.Unlock()
			delete(w.servers, serverName)
		}
	}
	xl.Infof("Workspace %s closed successfully", w.Id)
}
