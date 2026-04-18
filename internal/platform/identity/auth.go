package identity

import (
	"net/http"

	"strings"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/errs"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
)

type AuthMiddleware struct {
	config *config.Config
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{config: cfg}
}

func (m *AuthMiddleware) GetKeyAuthConfig() middleware.KeyAuthConfig {
	return middleware.KeyAuthConfig{
		// 从 Header 或 Query 获取。新增 Mcp-Session-Id / X-Session-Id 以兼容
		// Streamable HTTP 客户端（session 走 header）。
		KeyLookup: "header:Authorization:Bearer ,query:api_key,query:sessionId,header:Mcp-Session-Id,header:X-Session-Id",
		Validator: m.KeyAuthValidator,
		ErrorHandler: func(err error, c echo.Context) error {
			return c.JSON(http.StatusUnauthorized, map[string]any{"code": 401, "msg": errs.ErrAuthFailed.Error()})
		},
	}
}

func (m *AuthMiddleware) KeyAuthValidator(key string, c echo.Context) (bool, error) {
	xl := xlog.NewLogger("AUTH")
	realPath := c.Request().URL.Path
	xl.Infof("Auth key: %s, path: %s", key, realPath)

	if m.config.GetAuthConfig() == nil { // 如果没有配置，直接放行
		xl.Infof("Auth config not found")
		return false, errs.ErrAuthConfigNotFound
	}
	xl.Infof("Auth key: %s, api key: %s", key, m.config.GetAuthConfig().GetApiKey())
	if key == m.config.GetAuthConfig().GetApiKey() { // 验证API Key
		return true, nil
	}

	checkSession := false
	switch realPath {
	case "/sse", "/message", "/stream":
		checkSession = true
	default:
		if strings.Contains(realPath, "/message") {
			checkSession = true
		}
	}

	if checkSession {
		// 检查session：query 和 header 均可放行，兼容 SSE 与 Streamable HTTP 两种协议
		if c.QueryParam("sessionId") != "" {
			return true, nil
		}
		if c.Request().Header.Get("Mcp-Session-Id") != "" {
			return true, nil
		}
		if c.Request().Header.Get("X-Session-Id") != "" {
			return true, nil
		}
		return false, nil
	}

	return false, nil
}
