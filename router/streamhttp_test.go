package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// 构造一条 POST /stream 请求并走 handler，返回 echo.Context 和 recorder。
func buildStreamHTTPRequest(t *testing.T, method, body string, headers map[string]string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	var bodyReader *strings.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}
	var req *http.Request
	if bodyReader != nil {
		req = httptest.NewRequest(method, "/stream", bodyReader)
	} else {
		req = httptest.NewRequest(method, "/stream", nil)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	return c, rec
}

func newTestSession(id string) *service.Session {
	return service.NewSession(id)
}

// --- Initialize: 创建 session，返回聚合 InitializeResult ---
func TestGlobalStreamHTTP_Initialize_CreatesSessionAndReturnsResult(t *testing.T) {
	srv, mockMgr := createTestServerManager()

	// mock CreateProxySession 返回新 session
	newSess := newTestSession("sess-init")
	mockMgr.On("CreateProxySession", mock.Anything, mock.MatchedBy(func(n service.NameArg) bool {
		return n.Workspace == service.DefaultWorkspace
	})).Return(newSess, nil).Once()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"inspector","version":"0.1"}}}`
	c, rec := buildStreamHTTPRequest(t, http.MethodPost, body, nil)

	err := srv.handleGlobalStreamHTTP(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "sess-init", rec.Header().Get("Mcp-Session-Id"))

	var resp map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "2.0", resp["jsonrpc"])
	assert.EqualValues(t, 1, resp["id"])

	result, ok := resp["result"].(map[string]any)
	assert.True(t, ok, "result should be object")
	assert.Equal(t, "2025-03-26", result["protocolVersion"])

	serverInfo, ok := result["serverInfo"].(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "mcp-gateway", serverInfo["name"])

	mockMgr.AssertExpectations(t)
}

// --- 非 initialize 必须带 Mcp-Session-Id ---
func TestGlobalStreamHTTP_NonInitialize_MissingSessionReturns400(t *testing.T) {
	srv, _ := createTestServerManager()

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	c, rec := buildStreamHTTPRequest(t, http.MethodPost, body, nil)

	err := srv.handleGlobalStreamHTTP(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	errObj, ok := resp["error"].(map[string]any)
	assert.True(t, ok)
	assert.EqualValues(t, -32600, errObj["code"])
}

// --- 非 initialize 带无效 session 返回 404 ---
func TestGlobalStreamHTTP_NonInitialize_InvalidSessionReturns404(t *testing.T) {
	srv, mockMgr := createTestServerManager()
	mockMgr.On("GetProxySession", mock.Anything, mock.MatchedBy(func(n service.NameArg) bool {
		return n.Session == "unknown"
	})).Return((*service.Session)(nil), false).Once()

	body := `{"jsonrpc":"2.0","id":3,"method":"tools/list"}`
	c, rec := buildStreamHTTPRequest(t, http.MethodPost, body, map[string]string{"Mcp-Session-Id": "unknown"})

	err := srv.handleGlobalStreamHTTP(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)

	mockMgr.AssertExpectations(t)
}

// --- notifications/initialized 直接 202 ---
func TestGlobalStreamHTTP_NotificationsInitialized_Returns202(t *testing.T) {
	srv, mockMgr := createTestServerManager()
	sess := newTestSession("sess-n")
	mockMgr.On("GetProxySession", mock.Anything, mock.MatchedBy(func(n service.NameArg) bool {
		return n.Session == "sess-n"
	})).Return(sess, true).Once()

	body := `{"jsonrpc":"2.0","method":"notifications/initialized"}`
	c, rec := buildStreamHTTPRequest(t, http.MethodPost, body, map[string]string{"Mcp-Session-Id": "sess-n"})

	err := srv.handleGlobalStreamHTTP(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusAccepted, rec.Code)
	assert.Empty(t, rec.Body.String())

	mockMgr.AssertExpectations(t)
}

// --- DELETE 关闭 session ---
func TestGlobalStreamHTTP_Delete_ClosesSession(t *testing.T) {
	srv, mockMgr := createTestServerManager()
	mockMgr.On("CloseProxySession", mock.Anything, mock.MatchedBy(func(n service.NameArg) bool {
		return n.Session == "sess-del"
	})).Return().Once()

	c, rec := buildStreamHTTPRequest(t, http.MethodDelete, "", map[string]string{"Mcp-Session-Id": "sess-del"})

	err := srv.handleGlobalStreamHTTP(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	mockMgr.AssertExpectations(t)
}

// --- DELETE 缺失 header 返回 400 ---
func TestGlobalStreamHTTP_Delete_MissingHeader(t *testing.T) {
	srv, _ := createTestServerManager()
	c, rec := buildStreamHTTPRequest(t, http.MethodDelete, "", nil)

	err := srv.handleGlobalStreamHTTP(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// --- 非 GET/POST/DELETE 返回 405 ---
func TestGlobalStreamHTTP_MethodNotAllowed(t *testing.T) {
	srv, _ := createTestServerManager()
	c, rec := buildStreamHTTPRequest(t, http.MethodPut, "", nil)

	err := srv.handleGlobalStreamHTTP(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
	assert.Contains(t, rec.Header().Get("Allow"), "POST")
}

// --- isJSONRPCResponse 过滤函数单元测试 ---
func TestIsJSONRPCResponse(t *testing.T) {
	cases := []struct {
		name string
		data string
		want bool
	}{
		{"result_response", `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`, true},
		{"error_response", `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"x"}}`, true},
		{"notification", `{"jsonrpc":"2.0","method":"notifications/message","params":{}}`, false},
		{"request_with_id_and_method", `{"jsonrpc":"2.0","id":1,"method":"sampling/createMessage"}`, false},
		{"invalid_json", `not-json`, false},
		{"response_with_string_id", `{"jsonrpc":"2.0","id":"abc","result":{}}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, isJSONRPCResponse(tc.data))
		})
	}
}

// --- GET /stream 过滤 JSON-RPC response，只转发 request/notification ---
func TestGlobalStreamHTTP_EventStream_DropsResponses(t *testing.T) {
	srv, mockMgr := createTestServerManager()
	sess := newTestSession("sess-stream")
	mockMgr.On("GetProxySession", mock.Anything, mock.MatchedBy(func(n service.NameArg) bool {
		return n.Session == "sess-stream"
	})).Return(sess, true).Once()

	c, rec := buildStreamHTTPRequest(t, http.MethodGet, "", map[string]string{"Mcp-Session-Id": "sess-stream"})

	// 异步注入：一条 response（应被 drop）+ 一条 notification（应被转发）
	go func() {
		time.Sleep(30 * time.Millisecond)
		sess.SendEvent(service.SessionMsg{
			Event: "message",
			Data:  `{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`,
		})
		time.Sleep(20 * time.Millisecond)
		sess.SendEvent(service.SessionMsg{
			Event: "message",
			Data:  `{"jsonrpc":"2.0","method":"notifications/message","params":{"level":"info","logger":"test","data":"hi"}}`,
		})
		time.Sleep(20 * time.Millisecond)
		// 触发 handler 退出
		c.Request().Context()
	}()

	// 手工在 80ms 后关闭 context 让 handler 退出
	ctx, cancel := context.WithCancel(c.Request().Context())
	c.SetRequest(c.Request().WithContext(ctx))
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	err := srv.handleGlobalStreamHTTP(c)
	assert.NoError(t, err)

	body := rec.Body.String()
	// response 不应出现
	assert.NotContains(t, body, `"result":{"tools":[]}`)
	// notification 应被转发
	assert.Contains(t, body, `notifications/message`)

	mockMgr.AssertExpectations(t)
}

// --- request 转发并等待响应：外部注入 SendEvent 模拟下游响应 ---
func TestGlobalStreamHTTP_Request_ForwardAndWait(t *testing.T) {
	srv, mockMgr := createTestServerManager()

	sess := newTestSession("sess-fw")
	mockMgr.On("GetProxySession", mock.Anything, mock.MatchedBy(func(n service.NameArg) bool {
		return n.Session == "sess-fw"
	})).Return(sess, true).Once()

	body := `{"jsonrpc":"2.0","id":42,"method":"tools/list"}`
	c, rec := buildStreamHTTPRequest(t, http.MethodPost, body, map[string]string{"Mcp-Session-Id": "sess-fw"})

	// handler 订阅 eventChan 后会异步等待响应：此 goroutine 延迟注入一条响应
	go func() {
		// 留出 handler 订阅与发送 SendMessage 的时间窗口
		time.Sleep(50 * time.Millisecond)
		sess.SendEvent(service.SessionMsg{
			Event: "message",
			Data:  `{"jsonrpc":"2.0","id":42,"result":{"tools":[]}}`,
		})
	}()

	err := srv.handleGlobalStreamHTTP(c)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.EqualValues(t, 42, resp["id"])
	assert.NotNil(t, resp["result"])

	mockMgr.AssertExpectations(t)
}
