package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/types"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockServiceManager 模拟 ServiceManagerI 接口
type MockServiceManager struct {
	mock.Mock
}

func (m *MockServiceManager) DeployServer(logger xlog.Logger, name service.NameArg, config config.MCPServerConfig) (service.AddMcpServiceResult, error) {
	args := m.Called(logger, name, config)
	return args.Get(0).(service.AddMcpServiceResult), args.Error(1)
}

func (m *MockServiceManager) StopServer(logger xlog.Logger, name service.NameArg) {
	m.Called(logger, name)
}

func (m *MockServiceManager) RestartServer(logger xlog.Logger, name service.NameArg) error {
	args := m.Called(logger, name)
	return args.Error(0)
}

func (m *MockServiceManager) ListServerConfig(logger xlog.Logger, name service.NameArg) map[string]config.MCPServerConfig {
	args := m.Called(logger, name)
	return args.Get(0).(map[string]config.MCPServerConfig)
}

func (m *MockServiceManager) GetMcpService(logger xlog.Logger, name service.NameArg) (service.ExportMcpService, error) {
	args := m.Called(logger, name)
	return args.Get(0).(service.ExportMcpService), args.Error(1)
}

func (m *MockServiceManager) GetMcpServices(logger xlog.Logger, name service.NameArg) map[string]service.ExportMcpService {
	args := m.Called(logger, name)
	return args.Get(0).(map[string]service.ExportMcpService)
}

func (m *MockServiceManager) CreateProxySession(logger xlog.Logger, name service.NameArg) (*service.Session, error) {
	args := m.Called(logger, name)
	return args.Get(0).(*service.Session), args.Error(1)
}

func (m *MockServiceManager) GetProxySession(logger xlog.Logger, name service.NameArg) (*service.Session, bool) {
	args := m.Called(logger, name)
	return args.Get(0).(*service.Session), args.Bool(1)
}

func (m *MockServiceManager) GetWorkspaceSessions(logger xlog.Logger, name service.NameArg) []*service.Session {
	args := m.Called(logger, name)
	return args.Get(0).([]*service.Session)
}

func (m *MockServiceManager) CloseProxySession(logger xlog.Logger, name service.NameArg) {
	m.Called(logger, name)
}

func (m *MockServiceManager) DeleteServer(logger xlog.Logger, name service.NameArg) error {
	args := m.Called(logger, name)
	return args.Error(0)
}

func (m *MockServiceManager) Close() {
	m.Called()
}

// MockPortManager 模拟 PortManager
type MockPortManager struct {
	mock.Mock
}

func (m *MockPortManager) GetNextAvailablePort() int {
	args := m.Called()
	return args.Int(0)
}

func (m *MockPortManager) ReleasePort(port int) {
	m.Called(port)
}

func (m *MockPortManager) IsPortAvailable(port int) bool {
	args := m.Called(port)
	return args.Bool(0)
}

// 创建测试用的 ServerManager
func createTestServerManager() (*ServerManager, *MockServiceManager) {
	mockServiceMgr := &MockServiceManager{}

	serverMgr := &ServerManager{
		mcpServiceMgr: mockServiceMgr,
	}

	return serverMgr, mockServiceMgr
}

func TestHandleDeploy_Success_NewDeployment(t *testing.T) {
	// 设置 Echo
	e := echo.New()

	// 创建测试用的 ServerManager
	serverMgr, mockServiceMgr := createTestServerManager()

	// 模拟请求体
	deployReq := types.DeployRequest{
		MCPServers: map[string]config.MCPServerConfig{
			"test-service-1": {
				Command:   "uvx",
				Args:      []string{"mcp-server-git"},
				Workspace: "default",
			},
			"test-service-2": {
				URL:       "http://localhost:3000",
				Workspace: "default",
			},
		},
	}

	// 设置 mock 期望
	mockServiceMgr.On("DeployServer", mock.AnythingOfType("*xlog.zapLogger"), mock.MatchedBy(func(nameArg service.NameArg) bool {
		return nameArg.Server == "test-service-1" && nameArg.Workspace == "default"
	}), mock.MatchedBy(func(cfg config.MCPServerConfig) bool {
		return cfg.Command == "uvx" && len(cfg.Args) == 1 && cfg.Args[0] == "mcp-server-git"
	})).Return(service.AddMcpServiceResultDeployed, nil)

	mockServiceMgr.On("DeployServer", mock.AnythingOfType("*xlog.zapLogger"), mock.MatchedBy(func(nameArg service.NameArg) bool {
		return nameArg.Server == "test-service-2" && nameArg.Workspace == "default"
	}), mock.MatchedBy(func(cfg config.MCPServerConfig) bool {
		return cfg.URL == "http://localhost:3000"
	})).Return(service.AddMcpServiceResultDeployed, nil)

	// 创建请求
	reqBody, _ := json.Marshal(deployReq)
	req := httptest.NewRequest(http.MethodPost, "/deploy", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 执行处理函数
	err := serverMgr.handleDeploy(c)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// 解析响应
	var response types.DeployResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证响应内容
	assert.True(t, response.Success)
	assert.Equal(t, 2, response.Summary.Total)
	assert.Equal(t, 2, response.Summary.Deployed)
	assert.Equal(t, 0, response.Summary.Existed)
	assert.Equal(t, 0, response.Summary.Replaced)
	assert.Equal(t, 0, response.Summary.Failed)

	// 验证每个服务的结果
	assert.Equal(t, types.ServiceDeployStatusDeployed, response.Results["test-service-1"].Status)
	assert.Equal(t, types.ServiceDeployStatusDeployed, response.Results["test-service-2"].Status)
	assert.Contains(t, response.Results["test-service-1"].Message, "服务部署成功")
	assert.Contains(t, response.Results["test-service-2"].Message, "服务部署成功")

	// 验证 mock 调用
	mockServiceMgr.AssertExpectations(t)
}

func TestHandleDeploy_MixedResults(t *testing.T) {
	// 设置 Echo
	e := echo.New()

	// 创建测试用的 ServerManager
	serverMgr, mockServiceMgr := createTestServerManager()

	// 模拟请求体
	deployReq := types.DeployRequest{
		MCPServers: map[string]config.MCPServerConfig{
			"existing-service": {
				Command:   "uvx",
				Args:      []string{"mcp-server-git"},
				Workspace: "default",
			},
			"new-service": {
				Command:   "uvx",
				Args:      []string{"mcp-server-filesystem"},
				Workspace: "default",
			},
			"replaced-service": {
				URL:       "http://localhost:3001",
				Workspace: "default",
			},
			"failed-service": {
				Command:   "invalid-command",
				Workspace: "default",
			},
		},
	}

	// 设置 mock 期望
	mockServiceMgr.On("DeployServer", mock.AnythingOfType("*xlog.zapLogger"), mock.MatchedBy(func(nameArg service.NameArg) bool {
		return nameArg.Server == "existing-service"
	}), mock.AnythingOfType("config.MCPServerConfig")).
		Return(service.AddMcpServiceResultExisted, nil)

	mockServiceMgr.On("DeployServer", mock.AnythingOfType("*xlog.zapLogger"), mock.MatchedBy(func(nameArg service.NameArg) bool {
		return nameArg.Server == "new-service"
	}), mock.AnythingOfType("config.MCPServerConfig")).
		Return(service.AddMcpServiceResultDeployed, nil)

	mockServiceMgr.On("DeployServer", mock.AnythingOfType("*xlog.zapLogger"), mock.MatchedBy(func(nameArg service.NameArg) bool {
		return nameArg.Server == "replaced-service"
	}), mock.AnythingOfType("config.MCPServerConfig")).
		Return(service.AddMcpServiceResultReplaced, nil)

	mockServiceMgr.On("DeployServer", mock.AnythingOfType("*xlog.zapLogger"), mock.MatchedBy(func(nameArg service.NameArg) bool {
		return nameArg.Server == "failed-service"
	}), mock.AnythingOfType("config.MCPServerConfig")).
		Return(service.AddMcpServiceResult(""), fmt.Errorf("invalid command"))

	// 创建请求
	reqBody, _ := json.Marshal(deployReq)
	req := httptest.NewRequest(http.MethodPost, "/deploy", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 执行处理函数
	err := serverMgr.handleDeploy(c)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, http.StatusPartialContent, rec.Code) // 有失败的情况应该返回 206

	// 解析响应
	var response types.DeployResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证响应内容
	assert.False(t, response.Success) // 有失败所以整体不成功
	assert.Equal(t, 4, response.Summary.Total)
	assert.Equal(t, 1, response.Summary.Deployed)
	assert.Equal(t, 1, response.Summary.Existed)
	assert.Equal(t, 1, response.Summary.Replaced)
	assert.Equal(t, 1, response.Summary.Failed)

	// 验证每个服务的结果
	assert.Equal(t, types.ServiceDeployStatusExisted, response.Results["existing-service"].Status)
	assert.Equal(t, types.ServiceDeployStatusDeployed, response.Results["new-service"].Status)
	assert.Equal(t, types.ServiceDeployStatusReplaced, response.Results["replaced-service"].Status)
	assert.Equal(t, types.ServiceDeployStatusFailed, response.Results["failed-service"].Status)

	// 验证错误信息
	assert.Contains(t, response.Results["failed-service"].Error, "invalid command")
	assert.Contains(t, response.Results["failed-service"].Message, "部署失败")

	// 验证 mock 调用
	mockServiceMgr.AssertExpectations(t)

	// 打印响应 JSON 以供参考
	fmt.Printf("Mixed Results Response JSON:\n%s\n", rec.Body.String())
}

func TestHandleDeploy_AllFailed(t *testing.T) {
	// 设置 Echo
	e := echo.New()

	// 创建测试用的 ServerManager
	serverMgr, mockServiceMgr := createTestServerManager()

	// 模拟请求体
	deployReq := types.DeployRequest{
		MCPServers: map[string]config.MCPServerConfig{
			"failed-service-1": {
				Command:   "invalid-command-1",
				Workspace: "default",
			},
			"failed-service-2": {
				Command:   "invalid-command-2",
				Workspace: "default",
			},
		},
	}

	// 设置 mock 期望
	mockServiceMgr.On("DeployServer", mock.AnythingOfType("*xlog.zapLogger"), mock.MatchedBy(func(nameArg service.NameArg) bool {
		return nameArg.Server == "failed-service-1"
	}), mock.AnythingOfType("config.MCPServerConfig")).
		Return(service.AddMcpServiceResult(""), fmt.Errorf("command not found"))

	mockServiceMgr.On("DeployServer", mock.AnythingOfType("*xlog.zapLogger"), mock.MatchedBy(func(nameArg service.NameArg) bool {
		return nameArg.Server == "failed-service-2"
	}), mock.AnythingOfType("config.MCPServerConfig")).
		Return(service.AddMcpServiceResult(""), fmt.Errorf("permission denied"))

	// 创建请求
	reqBody, _ := json.Marshal(deployReq)
	req := httptest.NewRequest(http.MethodPost, "/deploy", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 执行处理函数
	err := serverMgr.handleDeploy(c)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, http.StatusPartialContent, rec.Code)

	// 解析响应
	var response types.DeployResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证响应内容
	assert.False(t, response.Success)
	assert.Equal(t, 2, response.Summary.Total)
	assert.Equal(t, 0, response.Summary.Deployed)
	assert.Equal(t, 0, response.Summary.Existed)
	assert.Equal(t, 0, response.Summary.Replaced)
	assert.Equal(t, 2, response.Summary.Failed)

	// 验证错误信息
	assert.Contains(t, response.Results["failed-service-1"].Error, "command not found")
	assert.Contains(t, response.Results["failed-service-2"].Error, "permission denied")

	// 验证 mock 调用
	mockServiceMgr.AssertExpectations(t)
}

func TestHandleDeploy_InvalidJSON(t *testing.T) {
	// 设置 Echo
	e := echo.New()

	// 创建测试用的 ServerManager
	serverMgr, _ := createTestServerManager()

	// 创建无效的 JSON 请求
	req := httptest.NewRequest(http.MethodPost, "/deploy", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 执行处理函数
	err := serverMgr.handleDeploy(c)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestHandleDeploy_EmptyRequest(t *testing.T) {
	// 设置 Echo
	e := echo.New()

	// 创建测试用的 ServerManager
	serverMgr, _ := createTestServerManager()

	// 模拟空的请求体
	deployReq := types.DeployRequest{
		MCPServers: map[string]config.MCPServerConfig{},
	}

	// 创建请求
	reqBody, _ := json.Marshal(deployReq)
	req := httptest.NewRequest(http.MethodPost, "/deploy", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// 执行处理函数
	err := serverMgr.handleDeploy(c)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// 解析响应
	var response types.DeployResponse
	err = json.Unmarshal(rec.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证响应内容
	assert.True(t, response.Success)
	assert.Equal(t, 0, response.Summary.Total)
	assert.Equal(t, 0, len(response.Results))
}
