package gateway

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/labstack/echo/v4"
)

const protectedResourceMetadataPath = "/.well-known/oauth-protected-resource"

func (h *Handler) mcpAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !h.cfg.GetAuthConfig().IsEnabled() {
			return next(c)
		}

		token := extractMCPBearerToken(c.Request().Header.Get(echo.HeaderAuthorization))
		if token == "" {
			// 只有配置了OAuth服务器时才返回OAuth格式的错误
			if len(h.authorizationServers(c)) > 0 {
				return h.respondMCPUnauthorized(c, "invalid_request", "missing bearer token")
			}
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"error":             "unauthorized",
				"error_description": "missing bearer token",
			})
		}

		principal, err := h.validateMCPOAuthToken(c, token)
		if err != nil {
			// 只有配置了OAuth服务器时才返回OAuth格式的错误
			if len(h.authorizationServers(c)) > 0 {
				return h.respondMCPUnauthorized(c, "invalid_token", err.Error())
			}
			return c.JSON(http.StatusUnauthorized, map[string]any{
				"error":             "unauthorized",
				"error_description": "invalid token",
			})
		}
		c.Set("auth.principal", principal)
		return next(c)
	}
}

func extractMCPBearerToken(header string) string {
	header = strings.TrimSpace(header)
	if len(header) < len("Bearer ") || !strings.EqualFold(header[:len("Bearer ")], "Bearer ") {
		return ""
	}
	return strings.TrimSpace(header[len("Bearer "):])
}

func (h *Handler) respondMCPUnauthorized(c echo.Context, code, description string) error {
	metadataURL := h.protectedResourceMetadataURL(c)
	challenge := `Bearer resource_metadata="` + escapeAuthParam(metadataURL) + `"`
	if code != "" {
		challenge += `, error="` + escapeAuthParam(code) + `"`
	}
	if description != "" {
		challenge += `, error_description="` + escapeAuthParam(description) + `"`
	}
	if scopes := strings.Join(h.cfg.GetAuthConfig().RequiredScopes, " "); scopes != "" {
		challenge += `, scope="` + escapeAuthParam(scopes) + `"`
	}
	c.Response().Header().Set(echo.HeaderWWWAuthenticate, challenge)
	return c.JSON(http.StatusUnauthorized, map[string]any{
		"error":             code,
		"error_description": description,
	})
}

func escapeAuthParam(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	return strings.ReplaceAll(v, `"`, `\"`)
}

func (h *Handler) handleProtectedResourceMetadata(c echo.Context) error {
	if !h.cfg.GetAuthConfig().IsEnabled() {
		return c.NoContent(http.StatusNotFound)
	}
	authServers := h.authorizationServers(c)
	// 如果没有配置OAuth服务器，说明使用内部JWT认证，不暴露OAuth元数据
	if len(authServers) == 0 {
		return c.NoContent(http.StatusNotFound)
	}
	resp := map[string]any{
		"resource":              h.protectedResource(c),
		"authorization_servers": authServers,
	}
	if len(h.cfg.GetAuthConfig().ScopesSupported) > 0 {
		resp["scopes_supported"] = h.cfg.GetAuthConfig().ScopesSupported
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) authorizationServers(c echo.Context) []string {
	cfg := h.cfg.GetAuthConfig()
	servers := make([]string, 0, len(cfg.AuthorizationServers))
	for _, item := range cfg.AuthorizationServers {
		item = strings.TrimSpace(item)
		if item != "" {
			servers = append(servers, strings.TrimRight(item, "/"))
		}
	}
	if len(servers) > 0 {
		return servers
	}
	return nil
}

func (h *Handler) protectedResource(c echo.Context) string {
	path := "/"
	reqPath := c.Request().URL.Path
	if strings.HasPrefix(reqPath, protectedResourceMetadataPath) {
		path = strings.TrimPrefix(reqPath, protectedResourceMetadataPath)
		if path == "" {
			path = "/"
		}
	} else if reqPath != "" {
		path = reqPath
	}
	return h.absoluteURL(c, path)
}

func (h *Handler) protectedResourceMetadataURL(c echo.Context) string {
	path := c.Request().URL.Path
	if path == "" || path == "/" {
		path = ""
	}
	return h.absoluteURL(c, protectedResourceMetadataPath+path)
}

func (h *Handler) absoluteURL(c echo.Context, path string) string {
	prefix := firstHeaderValue(c.Request().Header.Get("X-Forwarded-Prefix"))
	if prefix != "" {
		prefix = "/" + strings.Trim(prefix, "/")
	}
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u := url.URL{
		Scheme: h.requestScheme(c),
		Host:   h.requestHost(c),
		Path:   prefix + path,
	}
	return u.String()
}

func (h *Handler) requestOrigin(c echo.Context) string {
	u := url.URL{
		Scheme: h.requestScheme(c),
		Host:   h.requestHost(c),
	}
	return u.String()
}

func (h *Handler) requestScheme(c echo.Context) string {
	if proto := firstHeaderValue(c.Request().Header.Get("X-Forwarded-Proto")); proto != "" {
		return proto
	}
	if c.Request().TLS != nil {
		return "https"
	}
	return "http"
}

func (h *Handler) requestHost(c echo.Context) string {
	if host := firstHeaderValue(c.Request().Header.Get("X-Forwarded-Host")); host != "" {
		return host
	}
	return c.Request().Host
}

func firstHeaderValue(header string) string {
	if header == "" {
		return ""
	}
	parts := strings.Split(header, ",")
	return strings.TrimSpace(parts[0])
}
