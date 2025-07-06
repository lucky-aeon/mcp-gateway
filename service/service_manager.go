package service

import (
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/errs"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

type ServiceManagerI interface {
	DeployServer(logger xlog.Logger, name NameArg, config config.MCPServerConfig) (AddMcpServiceResult, error)
	StopServer(logger xlog.Logger, name NameArg)
	RestartServer(logger xlog.Logger, name NameArg) error
	ListServerConfig(logger xlog.Logger, name NameArg) map[string]config.MCPServerConfig
	GetMcpService(logger xlog.Logger, name NameArg) (ExportMcpService, error)
	GetMcpServices(logger xlog.Logger, name NameArg) map[string]ExportMcpService
	CreateProxySession(logger xlog.Logger, name NameArg) (*Session, error)
	GetProxySession(logger xlog.Logger, name NameArg) (*Session, bool)
	GetWorkspaceSessions(logger xlog.Logger, name NameArg) []*Session
	CloseProxySession(logger xlog.Logger, name NameArg)
	DeleteServer(logger xlog.Logger, name NameArg) error
	Close()
}

type NameArg struct {
	Workspace string
	Server    string
	Session   string
}

type ServiceManager struct {
	cfg          config.Config
	PortMgr      PortManagerI
	workSpaceMgr *WorkspaceManager
}

func NewServiceMgr(cfg config.Config, portMgr PortManagerI) *ServiceManager {
	return &ServiceManager{
		cfg:          cfg,
		PortMgr:      portMgr,
		workSpaceMgr: NewWorkspaceManager(cfg, portMgr),
	}
}

func (s *ServiceManager) DeployServer(logger xlog.Logger, name NameArg, config config.MCPServerConfig) (AddMcpServiceResult, error) {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.AddMcpService(logger, name.Server, config)
}

func (s *ServiceManager) StopServer(logger xlog.Logger, name NameArg) {
	workspace, ok := s.getWorkspace(logger, name.Workspace)
	if !ok {
		logger.Errorf("workspace %s not found", name.Workspace)
		return
	}
	if err := workspace.StopMcpService(logger, name.Server); err != nil {
		logger.Errorf("Failed to stop server %s: %v", name.Server, err)
	}
}

func (s *ServiceManager) RestartServer(logger xlog.Logger, name NameArg) error {
	workspace, ok := s.getWorkspace(logger, name.Workspace)
	if !ok {
		logger.Errorf("workspace %s not found", name.Workspace)
		return errs.ErrWorkspaceNotFound
	}
	if err := workspace.RestartMcpService(logger, name.Server); err != nil {
		logger.Errorf("Failed to restart server %s: %v", name.Server, err)
		return err
	}
	return nil
}

func (s *ServiceManager) ListServerConfig(logger xlog.Logger, name NameArg) map[string]config.MCPServerConfig {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.cfg.Servers
}

func (s *ServiceManager) GetMcpService(logger xlog.Logger, name NameArg) (ExportMcpService, error) {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.GetMcpService(name.Server)
}

func (s *ServiceManager) GetMcpServices(logger xlog.Logger, name NameArg) map[string]ExportMcpService {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.GetMcpServices()
}

func (s *ServiceManager) CreateProxySession(logger xlog.Logger, name NameArg) (*Session, error) {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.sessionMgr.CreateSession(logger)
}

func (s *ServiceManager) GetProxySession(logger xlog.Logger, name NameArg) (*Session, bool) {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.sessionMgr.GetSession(logger, name.Session)
}

func (s *ServiceManager) GetWorkspaceSessions(logger xlog.Logger, name NameArg) []*Session {
	workspace, ok := s.getWorkspace(logger, name.Workspace, true)
	if !ok {
		return []*Session{}
	}
	return workspace.sessionMgr.GetAllSessions(logger)
}

func (s *ServiceManager) CloseProxySession(logger xlog.Logger, name NameArg) {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	workspace.sessionMgr.CloseSession(logger, name.Session)
}

func (s *ServiceManager) DeleteServer(logger xlog.Logger, name NameArg) error {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	if err := workspace.RemoveMcpService(logger, name.Server); err != nil {
		return err
	}
	return nil
}

// Close stops all MCP services in all workspaces.
func (s *ServiceManager) Close() {
	xl := xlog.NewLogger("servicev2")
	for _, workspace := range s.workSpaceMgr.GetWorkspaces() {
		workspace.Close(xl)
	}
}

func (s *ServiceManager) getWorkspace(logger xlog.Logger, name string, noCreateIfNotExists ...bool) (*WorkSpace, bool) {
	create := true
	if len(noCreateIfNotExists) > 0 {
		create = noCreateIfNotExists[0]
	}
	workspace, ok := s.workSpaceMgr.GetWorkspace(logger, name, create)
	if !ok {
		return nil, false
	}
	return workspace, true
}

// GetWorkspaces 获取所有工作空间
func (s *ServiceManager) GetWorkspaces() map[string]*WorkSpace {
	return s.workSpaceMgr.GetWorkspaces()
}
