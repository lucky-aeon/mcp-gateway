package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var baseURL = func() string {
	if v := os.Getenv("MCP_GATEWAY_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}()

func apiBase() string { return baseURL + "/api/v1" }

type AuthResponse struct {
	Success bool `json:"success"`
	Data    struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		Account      struct {
			ID          string `json:"id"`
			Email       string `json:"email"`
			DisplayName string `json:"display_name"`
		} `json:"account"`
	} `json:"data"`
	Error any `json:"error"`
}

type MCPResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  any    `json:"result"`
	Error   any    `json:"error"`
}

// TestAuthenticationFlow 测试完整的认证流程：注册 → 登录 → 管理API → MCP协议
func TestAuthenticationFlow(t *testing.T) {
	email := fmt.Sprintf("e2e_%d@example.com", time.Now().UnixNano())
	password := "test123456"
	displayName := "E2E Test User"

	var accessToken string

	t.Run("Register", func(t *testing.T) {
		resp := makeRequest(t, "POST", apiBase()+"/auth/register", map[string]string{
			"email":        email,
			"password":     password,
			"display_name": displayName,
		}, "")

		var authResp AuthResponse
		require.NoError(t, json.Unmarshal(resp, &authResp))
		require.True(t, authResp.Success, "registration failed: %s", string(resp))
		require.NotEmpty(t, authResp.Data.AccessToken)
		assert.Equal(t, email, authResp.Data.Account.Email)
	})

	t.Run("Login", func(t *testing.T) {
		resp := makeRequest(t, "POST", apiBase()+"/auth/login", map[string]string{
			"email":    email,
			"password": password,
		}, "")

		var authResp AuthResponse
		require.NoError(t, json.Unmarshal(resp, &authResp))
		require.True(t, authResp.Success, "login failed: %s", string(resp))
		require.NotEmpty(t, authResp.Data.AccessToken)

		accessToken = authResp.Data.AccessToken
	})

	t.Run("AccessAdminAPI", func(t *testing.T) {
		require.NotEmpty(t, accessToken)
		resp := makeRequest(t, "GET", apiBase()+"/auth/me", nil, accessToken)

		var result map[string]any
		require.NoError(t, json.Unmarshal(resp, &result))
		require.True(t, result["success"].(bool))

		data := result["data"].(map[string]any)
		assert.Equal(t, email, data["email"])
	})

	t.Run("AccessMCPEndpoint", func(t *testing.T) {
		require.NotEmpty(t, accessToken)
		resp := makeRequest(t, "POST", baseURL+"/stream", mcpInitializePayload(), accessToken)

		var mcpResp MCPResponse
		require.NoError(t, json.Unmarshal(resp, &mcpResp))
		assert.Equal(t, "2.0", mcpResp.JSONRPC)
		assert.Nil(t, mcpResp.Error, "MCP request returned error: %v", mcpResp.Error)
		assert.NotNil(t, mcpResp.Result)
	})
}

// TestUnauthenticated 测试未认证请求被正确拒绝
func TestUnauthenticated(t *testing.T) {
	t.Run("MCPWithoutToken", func(t *testing.T) {
		req := newRequest(t, "POST", baseURL+"/stream", mcpInitializePayload(), "")
		resp := doRequest(t, req)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("MCPWithInvalidToken", func(t *testing.T) {
		req := newRequest(t, "POST", baseURL+"/stream", mcpInitializePayload(), "invalid-token-xxx")
		resp := doRequest(t, req)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("LoginWithWrongPassword", func(t *testing.T) {
		resp := makeRequest(t, "POST", apiBase()+"/auth/login", map[string]string{
			"email":    "nonexistent@example.com",
			"password": "wrongpassword",
		}, "")

		var result map[string]any
		require.NoError(t, json.Unmarshal(resp, &result))
		assert.False(t, result["success"].(bool))
		assert.NotNil(t, result["error"])
	})
}

// mcpInitializePayload 构造 MCP initialize 请求体
func mcpInitializePayload() map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]string{
				"name":    "e2e-test-client",
				"version": "1.0.0",
			},
		},
	}
}

func newRequest(t *testing.T, method, url string, payload any, token string) *http.Request {
	t.Helper()

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		require.NoError(t, err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	require.NoError(t, err)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func doRequest(t *testing.T, req *http.Request) *http.Response {
	t.Helper()
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	return resp
}

func makeRequest(t *testing.T, method, url string, payload any, token string) []byte {
	t.Helper()
	req := newRequest(t, method, url, payload, token)
	resp := doRequest(t, req)
	defer resp.Body.Close()

	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(resp.Body)
	require.NoError(t, err)
	return buf.Bytes()
}
