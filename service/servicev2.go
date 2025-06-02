package service

import (
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

type NameArg struct {
	Workspace string
	Server    string
	Session   string
}

type ServiceV2 struct {
	cfg          config.Config
	PortMgr      PortManagerI
	workSpaceMgr *WorkspaceManager
}

func NewServiceV2(cfg config.Config, portMgr PortManagerI) *ServiceV2 {
	return &ServiceV2{
		cfg:          cfg,
		PortMgr:      portMgr,
		workSpaceMgr: NewWorkspaceManager(cfg, portMgr),
	}
}

func (s *ServiceV2) DeployServer(logger xlog.Logger, name NameArg, config config.MCPServerConfig) error {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.AddMcpService(logger, name.Server, config)
}

func (s *ServiceV2) StopServer(logger xlog.Logger, name NameArg) {
	workspace, ok := s.getWorkspace(logger, name.Workspace)
	if !ok {
		logger.Errorf("workspace %s not found", name.Workspace)
		return
	}
	if err := workspace.StopMcpService(logger, name.Server); err != nil {
		logger.Errorf("Failed to stop server %s: %v", name.Server, err)
	}
}

func (s *ServiceV2) ListServerConfig(logger xlog.Logger, name NameArg) map[string]config.MCPServerConfig {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.cfg.Servers
}

func (s *ServiceV2) GetMcpService(logger xlog.Logger, name NameArg) (ExportMcpService, error) {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.GetMcpService(name.Server)
}

func (s *ServiceV2) GetMcpServices(logger xlog.Logger, name NameArg) map[string]ExportMcpService {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.GetMcpServices()
}

func (s *ServiceV2) CreateProxySession(logger xlog.Logger, name NameArg) (*Session, error) {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.sessionMgr.CreateSession(logger)
}

func (s *ServiceV2) GetProxySession(logger xlog.Logger, name NameArg) (*Session, bool) {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	return workspace.sessionMgr.GetSession(logger, name.Session)
}

func (s *ServiceV2) CloseProxySession(logger xlog.Logger, name NameArg) {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	workspace.sessionMgr.CloseSession(logger, name.Session)
}

func (s *ServiceV2) DeleteServer(logger xlog.Logger, name NameArg) error {
	workspace, _ := s.getWorkspace(logger, name.Workspace)
	if err := workspace.RemoveMcpService(logger, name.Server); err != nil {
		return err
	}
	return nil
}

// Close stops all MCP services in all workspaces.
func (s *ServiceV2) Close() {
	xl := xlog.NewLogger("servicev2")
	for _, workspace := range s.workSpaceMgr.GetWorkspaces() {
		workspace.Close(xl)
	}
}

func (s *ServiceV2) getWorkspace(logger xlog.Logger, name string, noCreateIfNotExists ...bool) (*WorkSpace, bool) {
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
