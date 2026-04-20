package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/sessions"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
	"github.com/stretchr/testify/assert"
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
