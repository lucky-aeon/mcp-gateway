package gateway

import (
	"github.com/stretchr/testify/mock"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/sessions"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
)

// MockServiceManager 模拟 workspaces.ServiceManagerI，用于 gateway 包内测试。
type MockServiceManager struct {
	mock.Mock
}

func (m *MockServiceManager) DeployServer(logger xlog.Logger, name workspaces.NameArg, cfg config.MCPServerConfig) (workspaces.AddMcpServiceResult, error) {
	args := m.Called(logger, name, cfg)
	return args.Get(0).(workspaces.AddMcpServiceResult), args.Error(1)
}

func (m *MockServiceManager) StopServer(logger xlog.Logger, name workspaces.NameArg) {
	m.Called(logger, name)
}

func (m *MockServiceManager) RestartServer(logger xlog.Logger, name workspaces.NameArg) error {
	args := m.Called(logger, name)
	return args.Error(0)
}

func (m *MockServiceManager) ListServerConfig(logger xlog.Logger, name workspaces.NameArg) map[string]config.MCPServerConfig {
	args := m.Called(logger, name)
	return args.Get(0).(map[string]config.MCPServerConfig)
}

func (m *MockServiceManager) GetMcpService(logger xlog.Logger, name workspaces.NameArg) (runtime.ExportMcpService, error) {
	args := m.Called(logger, name)
	return args.Get(0).(runtime.ExportMcpService), args.Error(1)
}

func (m *MockServiceManager) GetMcpServices(logger xlog.Logger, name workspaces.NameArg) map[string]runtime.ExportMcpService {
	args := m.Called(logger, name)
	return args.Get(0).(map[string]runtime.ExportMcpService)
}

func (m *MockServiceManager) CreateProxySession(logger xlog.Logger, name workspaces.NameArg) (*sessions.Session, error) {
	args := m.Called(logger, name)
	return args.Get(0).(*sessions.Session), args.Error(1)
}

func (m *MockServiceManager) GetProxySession(logger xlog.Logger, name workspaces.NameArg) (*sessions.Session, bool) {
	args := m.Called(logger, name)
	return args.Get(0).(*sessions.Session), args.Bool(1)
}

func (m *MockServiceManager) GetWorkspaceSessions(logger xlog.Logger, name workspaces.NameArg) []*sessions.Session {
	args := m.Called(logger, name)
	return args.Get(0).([]*sessions.Session)
}

func (m *MockServiceManager) CloseProxySession(logger xlog.Logger, name workspaces.NameArg) {
	m.Called(logger, name)
}

func (m *MockServiceManager) DeleteServer(logger xlog.Logger, name workspaces.NameArg) error {
	args := m.Called(logger, name)
	return args.Error(0)
}

func (m *MockServiceManager) Close() {
	m.Called()
}

// createTestServerManager 构造一个测试用的 gateway.Handler + MockServiceManager。
// 默认启用 Streamable HTTP 协议（测试主要覆盖 /stream 行为）。
func createTestServerManager() (*Handler, *MockServiceManager) {
	mockMgr := &MockServiceManager{}
	h := &Handler{
		services: mockMgr,
		cfg: config.Config{
			GatewayProtocol: "streamhttp",
		},
	}
	return h, mockMgr
}
