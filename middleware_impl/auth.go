package middleware_impl

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/errs"
)

type AuthMiddleware struct {
	config *config.Config
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{config: cfg}
}

func (m *AuthMiddleware) GetKeyAuthConfig() middleware.KeyAuthConfig {
	return middleware.KeyAuthConfig{
		KeyLookup: "header:Authorization:Bearer ,query:api_key", // 从Header或Query获取
		Validator: m.KeyAuthValidator,
		ErrorHandler: func(err error, c echo.Context) error {
			return c.JSON(http.StatusUnauthorized, map[string]any{"code": 401, "msg": errs.ErrAuthFailed.Error()})
		},
	}
}

func (m *AuthMiddleware) KeyAuthValidator(key string, c echo.Context) (bool, error) {
	if m.config.GetAuthConfig() == nil { // 如果没有配置，直接放行
		return false, errs.ErrAuthConfigNotFound
	}
	if key == m.config.GetAuthConfig().GetApiKey() { // 验证API Key
		return true, nil
	}

	return false, nil
}
