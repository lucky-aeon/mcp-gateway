package service

import (
	"sync"

	"github.com/google/uuid"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

type WorkspaceManager struct {
	workspaces     map[string]*WorkSpace
	workspacesLock sync.RWMutex

	cfg         config.Config
	portManager PortManagerI
}

func NewWorkspaceManager(cfg config.Config, portManager PortManagerI) *WorkspaceManager {
	return &WorkspaceManager{workspaces: make(map[string]*WorkSpace), cfg: cfg, portManager: portManager}
}

// GetWorkspace returns a workspace by id. If the workspace does not exist, it creates a new one.
// If the workspace is created, it returns false.
func (m *WorkspaceManager) GetWorkspace(xl xlog.Logger, workId string, createIfNotExists bool) (*WorkSpace, bool) {
	m.workspacesLock.RLock()
	workspace, ok := m.workspaces[workId]
	m.workspacesLock.RUnlock()
	if !ok {
		xl.Warnf("Workspace %s not found, creating new workspace", workId)
		if createIfNotExists {
			workspace = m.createWorkspace(xl, workId)
			return workspace, true
		}
		return nil, false
	}

	return workspace, ok
}

// createWorkspace creates a new workspace and returns it.
func (m *WorkspaceManager) createWorkspace(xl xlog.Logger, workId string) *WorkSpace {
	xl.Infof("Creating new workspace, id: %s", workId)
	if workId == "" {
		workId = uuid.New().String()
	}
	workspace := NewWorkSpace(workId, config.WorkspaceConfig{
		LogConfig: config.LogConfig{
			Level: m.cfg.LogLevel,
			Path:  m.cfg.ConfigDirPath,
		},
		McpServiceMgrConfig: m.cfg.McpServiceMgrConfig,
		Servers:             make(map[string]config.MCPServerConfig),
	}, m.portManager)
	m.workspacesLock.Lock()
	m.workspaces[workspace.Id] = workspace
	m.workspacesLock.Unlock()
	return workspace
}

func (m *WorkspaceManager) GetWorkspaces() map[string]*WorkSpace {
	m.workspacesLock.RLock()
	defer m.workspacesLock.RUnlock()
	return m.workspaces
}
