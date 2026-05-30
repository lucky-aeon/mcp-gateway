package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/sessions"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestHandleV1Meta(t *testing.T) {
	e := echo.New()
	h, _ := createTestServerManager()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.handleV1Meta(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp envelope
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "single-key", data["mode"])
	assert.Equal(t, h.cfg.GatewayProtocol, data["gateway_protocol"])
}

func TestHandleV1CreateWorkspace(t *testing.T) {
	e := echo.New()
	h, mockServiceMgr := createTestServerManager()
	mockServiceMgr.On("GetMcpServices", nilLogger{}, workspaces.NameArg{Workspace: "demo-workspace"}).Return(map[string]runtime.ExportMcpService{})
	mockServiceMgr.On("GetWorkspaceSessions", nilLogger{}, workspaces.NameArg{Workspace: "demo-workspace"}).Return([]*sessions.Session{})

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", strings.NewReader(`{"name":"Demo Workspace","description":"My sandbox"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	err := h.handleV1CreateWorkspace(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp envelope
	err = json.Unmarshal(rec.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.True(t, resp.Success)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "Demo Workspace", data["name"])
	assert.Equal(t, "My sandbox", data["description"])
	assert.NotEmpty(t, data["id"])
}

func TestV1AuthMiddlewareRequiresBearer(t *testing.T) {
	e := echo.New()
	h, _ := createTestServerManager()
	h.registerV1Routes(e)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestV1MetaIsPublicRoute(t *testing.T) {
	e := echo.New()
	h, _ := createTestServerManager()
	h.registerV1Routes(e)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/meta", nil)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestV1AuthLoginAndMe(t *testing.T) {
	e := echo.New()
	h, _ := createTestServerManager()
	h.registerV1Routes(e)

	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"api_key":"123456"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()

	e.ServeHTTP(loginRec, loginReq)

	assert.Equal(t, http.StatusOK, loginRec.Code)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer 123456")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestMarketInstallCreatesAccountSnapshotThenDeploysFromInstalled(t *testing.T) {
	e := echo.New()
	h, mockServiceMgr := createTestServerManager()
	h.market = newMarketStore()

	installReq := httptest.NewRequest(http.MethodPost, "/api/v1/market/packages/time-tools/install", strings.NewReader(`{"install_option_index":0}`))
	installReq.Header.Set("Content-Type", "application/json")
	installRec := httptest.NewRecorder()
	installCtx := e.NewContext(installReq, installRec)
	installCtx.SetParamNames("id")
	installCtx.SetParamValues("time-tools")

	err := h.handleV1InstallMarketPackage(installCtx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, installRec.Code)

	var installResp envelope
	err = json.Unmarshal(installRec.Body.Bytes(), &installResp)
	assert.NoError(t, err)
	assert.True(t, installResp.Success)
	installed := installResp.Data.(map[string]interface{})
	assert.Equal(t, "time-tools", installed["package_id"])
	assert.Equal(t, "installed", installed["status"])
	installedID := installed["id"].(string)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/installed", nil)
	listRec := httptest.NewRecorder()
	listCtx := e.NewContext(listReq, listRec)
	err = h.handleV1Installed(listCtx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, listRec.Code)

	mockServiceMgr.On("DeployServer", mock.Anything, workspaces.NameArg{Server: "time-tools", Workspace: "demo"}, mock.MatchedBy(func(cfg config.MCPServerConfig) bool {
		return cfg.Workspace == "demo" && cfg.Command == "uvx" && len(cfg.Args) > 0
	})).Return(workspaces.AddMcpServiceResultDeployed, nil).Once()
	mockServiceMgr.On("GetMcpServices", nilLogger{}, workspaces.NameArg{Workspace: "demo"}).Return(map[string]runtime.ExportMcpService{}).Maybe()

	deployReq := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/demo/services:from-installed", strings.NewReader(`{"installed_id":"`+installedID+`"}`))
	deployReq.Header.Set("Content-Type", "application/json")
	deployRec := httptest.NewRecorder()
	deployCtx := e.NewContext(deployReq, deployRec)
	deployCtx.SetParamNames("ws")
	deployCtx.SetParamValues("demo")

	err = h.handleV1CreateServiceFromInstalled(deployCtx)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, deployRec.Code)
	mockServiceMgr.AssertExpectations(t)
}

func TestServiceConfigFromMapAcceptsMongoBSONTypes(t *testing.T) {
	cfg := serviceConfigFromMap(map[string]interface{}{
		"command":          "uvx",
		"args":             primitive.A{"mcp-server-time", "--local-timezone=Asia/Shanghai"},
		"env":              primitive.M{"TZ": "Asia/Shanghai"},
		"gateway_protocol": "streamhttp",
	}, "default")

	assert.Equal(t, "default", cfg.Workspace)
	assert.Equal(t, "uvx", cfg.Command)
	assert.Equal(t, []string{"mcp-server-time", "--local-timezone=Asia/Shanghai"}, cfg.Args)
	assert.Equal(t, map[string]string{"TZ": "Asia/Shanghai"}, cfg.Env)
	assert.Equal(t, "streamhttp", cfg.GatewayProtocol)
}

func TestHandleV1DeleteServiceRemovesControlPlaneRecordWhenRuntimeMissing(t *testing.T) {
	e := echo.New()
	h, mockServiceMgr := createTestServerManager()
	h.state.ensureWorkspace("default")
	h.state.upsertService("default", serviceMeta{
		Name:        "time-tools",
		WorkspaceID: "default",
		SourceType:  "installed",
		SourceRef:   "time-tools",
	})

	mockServiceMgr.On("DeleteServer", nilLogger{}, workspaces.NameArg{Workspace: "default", Server: "time-tools"}).
		Return(fmt.Errorf("MCP service time-tools not found")).
		Once()

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/default/services/time-tools", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("ws", "name")
	c.SetParamValues("default", "time-tools")

	err := h.handleV1DeleteService(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	_, ok := h.state.getService("default", "time-tools")
	assert.False(t, ok)
	mockServiceMgr.AssertExpectations(t)
}
