package gateway

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"html"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/identity"
)

const protectedResourceMetadataPath = "/.well-known/oauth-protected-resource"
const authorizationServerMetadataPath = "/.well-known/oauth-authorization-server"

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

func (h *Handler) handleAuthorizationServerMetadata(c echo.Context) error {
	if !h.usesInternalAuthorizationServer() {
		return c.NoContent(http.StatusNotFound)
	}
	issuer := h.requestOrigin(c)
	return c.JSON(http.StatusOK, map[string]any{
		"issuer":                                issuer,
		"authorization_endpoint":                issuer + "/oauth/authorize",
		"token_endpoint":                        issuer + "/oauth/token",
		"registration_endpoint":                 issuer + "/oauth/register",
		"grant_types_supported":                 []string{"authorization_code", "password", "refresh_token"},
		"token_endpoint_auth_methods_supported": []string{"none"},
		"response_types_supported":              []string{"code"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
	})
}

func (h *Handler) handleOAuthRegister(c echo.Context) error {
	if !h.usesInternalAuthorizationServer() {
		return c.NoContent(http.StatusNotFound)
	}
	var req struct {
		ClientName              string   `json:"client_name"`
		RedirectURIs            []string `json:"redirect_uris"`
		GrantTypes              []string `json:"grant_types"`
		ResponseTypes           []string `json:"response_types"`
		TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
		Scope                   string   `json:"scope"`
	}
	if err := c.Bind(&req); err != nil {
		return oauthTokenError(c, http.StatusBadRequest, "invalid_client_metadata", err.Error())
	}
	clientID, err := newOAuthSecret("client_")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "server_error"})
	}
	if len(req.GrantTypes) == 0 {
		req.GrantTypes = []string{"authorization_code", "refresh_token"}
	}
	if len(req.ResponseTypes) == 0 {
		req.ResponseTypes = []string{"code"}
	}
	if req.TokenEndpointAuthMethod == "" {
		req.TokenEndpointAuthMethod = "none"
	}
	client := &oauthClient{
		ClientID:                clientID,
		ClientName:              req.ClientName,
		RedirectURIs:            append([]string(nil), req.RedirectURIs...),
		GrantTypes:              append([]string(nil), req.GrantTypes...),
		ResponseTypes:           append([]string(nil), req.ResponseTypes...),
		TokenEndpointAuthMethod: req.TokenEndpointAuthMethod,
		Scope:                   req.Scope,
		CreatedAt:               time.Now().UTC(),
	}
	h.internalOAuth().putClient(client)
	return c.JSON(http.StatusCreated, map[string]any{
		"client_id":                  client.ClientID,
		"client_name":                client.ClientName,
		"redirect_uris":              client.RedirectURIs,
		"grant_types":                client.GrantTypes,
		"response_types":             client.ResponseTypes,
		"token_endpoint_auth_method": client.TokenEndpointAuthMethod,
		"scope":                      client.Scope,
		"client_id_issued_at":        client.CreatedAt.Unix(),
	})
}

func (h *Handler) handleOAuthAuthorize(c echo.Context) error {
	if !h.usesInternalAuthorizationServer() || h.auth == nil {
		return c.NoContent(http.StatusNotFound)
	}
	if c.Request().Method == http.MethodGet {
		return h.renderOAuthLogin(c, "")
	}

	if err := c.Request().ParseForm(); err != nil {
		return oauthTokenError(c, http.StatusBadRequest, "invalid_request", "invalid form body")
	}
	if c.FormValue("response_type") != "code" {
		return c.String(http.StatusBadRequest, "unsupported response_type")
	}
	principal, err := h.authenticateOAuthLogin(c)
	if err != nil {
		return h.renderOAuthLogin(c, "Invalid account credentials or API key")
	}
	redirectURI := c.FormValue("redirect_uri")
	if strings.TrimSpace(redirectURI) == "" {
		return c.String(http.StatusBadRequest, "redirect_uri is required")
	}
	code, err := newOAuthSecret("oc_")
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to create authorization code")
	}
	h.internalOAuth().putCode(&oauthAuthorizationCode{
		Code:                code,
		AccountID:           principal.AccountID,
		RedirectURI:         redirectURI,
		CodeChallenge:       c.FormValue("code_challenge"),
		CodeChallengeMethod: c.FormValue("code_challenge_method"),
		ExpiresAt:           time.Now().UTC().Add(5 * time.Minute),
	})
	u, err := url.Parse(redirectURI)
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid redirect_uri")
	}
	q := u.Query()
	q.Set("code", code)
	if state := c.FormValue("state"); state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return c.Redirect(http.StatusFound, u.String())
}

func (h *Handler) handleOAuthToken(c echo.Context) error {
	if !h.usesInternalAuthorizationServer() || h.auth == nil {
		return c.NoContent(http.StatusNotFound)
	}

	if err := c.Request().ParseForm(); err != nil {
		return oauthTokenError(c, http.StatusBadRequest, "invalid_request", "invalid form body")
	}
	grantType := c.FormValue("grant_type")
	switch grantType {
	case "authorization_code":
		code := h.internalOAuth().takeCode(c.FormValue("code"))
		if code == nil || code.ExpiresAt.Before(time.Now().UTC()) {
			return oauthTokenError(c, http.StatusUnauthorized, "invalid_grant", "invalid authorization code")
		}
		if code.RedirectURI != "" && c.FormValue("redirect_uri") != "" && code.RedirectURI != c.FormValue("redirect_uri") {
			return oauthTokenError(c, http.StatusUnauthorized, "invalid_grant", "redirect_uri mismatch")
		}
		if !validPKCEVerifier(code, c.FormValue("code_verifier")) {
			return oauthTokenError(c, http.StatusUnauthorized, "invalid_grant", "invalid code verifier")
		}
		resp, err := h.auth.IssueTokenForAccount(c.Request().Context(), code.AccountID)
		if err != nil {
			return oauthTokenError(c, http.StatusUnauthorized, "invalid_grant", "failed to issue token")
		}
		return c.JSON(http.StatusOK, oauthTokenResponse(resp))
	case "password":
		username := c.FormValue("username")
		if username == "" {
			username = c.FormValue("email")
		}
		resp, err := h.auth.AuthenticatePassword(c.Request().Context(), username, c.FormValue("password"))
		if err != nil {
			return oauthTokenError(c, http.StatusUnauthorized, "invalid_grant", "invalid username or password")
		}
		return c.JSON(http.StatusOK, oauthTokenResponse(resp))
	case "refresh_token":
		resp, err := h.auth.RefreshAccessToken(c.Request().Context(), c.FormValue("refresh_token"))
		if err != nil {
			return oauthTokenError(c, http.StatusUnauthorized, "invalid_grant", "invalid refresh token")
		}
		return c.JSON(http.StatusOK, oauthTokenResponse(resp))
	default:
		return oauthTokenError(c, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be password or refresh_token")
	}
}

func (h *Handler) authenticateOAuthLogin(c echo.Context) (*identity.Principal, error) {
	if c.FormValue("auth_method") == "api_key" {
		return h.auth.ValidateBearer(c.Request().Context(), c.FormValue("api_key"))
	}
	resp, err := h.auth.AuthenticatePassword(c.Request().Context(), c.FormValue("username"), c.FormValue("password"))
	if err != nil {
		return nil, err
	}
	token, _ := resp["access_token"].(string)
	if token == "" {
		token, _ = resp["token"].(string)
	}
	return h.auth.ValidateBearer(c.Request().Context(), token)
}

func (h *Handler) renderOAuthLogin(c echo.Context, message string) error {
	if err := c.Request().ParseForm(); err != nil {
		return c.String(http.StatusBadRequest, "invalid request")
	}
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	return c.HTML(http.StatusOK, oauthLoginHTML(c.Request().Form, message))
}

func oauthLoginHTML(form url.Values, message string) string {
	hidden := []string{"response_type", "client_id", "redirect_uri", "scope", "state", "code_challenge", "code_challenge_method"}
	var fields strings.Builder
	for _, key := range hidden {
		if value := form.Get(key); value != "" {
			fields.WriteString(`<input type="hidden" name="` + key + `" value="` + html.EscapeString(value) + `">`)
		}
	}
	errorHTML := ""
	if message != "" {
		errorHTML = `<p class="error">` + html.EscapeString(message) + `</p>`
	}
	return `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>MCP Gateway Login</title><style>body{font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#f6f7f9;margin:0;display:grid;place-items:center;min-height:100vh;color:#1f2933}.box{width:min(380px,calc(100vw - 32px));background:#fff;border:1px solid #d9dee7;border-radius:8px;padding:24px;box-shadow:0 10px 30px rgba(15,23,42,.08)}h1{font-size:20px;margin:0 0 18px}.tabs{display:grid;grid-template-columns:1fr 1fr;gap:8px;margin-bottom:14px}.tabs label{display:flex;align-items:center;justify-content:center;height:36px;margin:0;border:1px solid #c7ceda;border-radius:6px;font-size:13px;font-weight:700;cursor:pointer}.tabs input{position:absolute;opacity:0;width:1px;height:1px}.tabs label:has(input:checked){background:#1f2937;border-color:#1f2937;color:#fff}.field-label{display:block;font-size:13px;font-weight:600;margin:14px 0 6px}input[type=email],input[type=password],input[name=api_key]{box-sizing:border-box;width:100%;height:40px;border:1px solid #c7ceda;border-radius:6px;padding:0 10px;font-size:14px}button{margin-top:18px;width:100%;height:40px;border:0;border-radius:6px;background:#1f2937;color:#fff;font-weight:700}.error{margin:0 0 12px;color:#b42318;font-size:14px}.method{display:none}.method.active{display:block}</style></head><body><form class="box" method="post" action="/oauth/authorize"><h1>MCP Gateway Login</h1>` + errorHTML + fields.String() + `<div class="tabs"><label><input type="radio" name="auth_method" value="password" checked>Account</label><label><input type="radio" name="auth_method" value="api_key">API Key</label></div><div class="method active" data-method="password"><label class="field-label">Email</label><input name="username" type="email" autocomplete="username" autofocus><label class="field-label">Password</label><input name="password" type="password" autocomplete="current-password"></div><div class="method" data-method="api_key"><label class="field-label">API Key</label><input name="api_key" type="password" autocomplete="off"></div><button type="submit">Sign in</button></form><script>const form=document.querySelector("form");const radios=[...document.querySelectorAll("input[name=auth_method]")];const sync=()=>{const method=document.querySelector("input[name=auth_method]:checked").value;document.querySelectorAll(".method").forEach(el=>el.classList.toggle("active",el.dataset.method===method));form.username.required=method==="password";form.password.required=method==="password";form.api_key.required=method==="api_key";if(method==="api_key")form.api_key.focus();else form.username.focus()};radios.forEach(r=>r.addEventListener("change",sync));sync();</script></body></html>`
}

func oauthTokenResponse(resp map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{
		"token_type": "Bearer",
	}
	if token, _ := resp["access_token"].(string); token != "" {
		out["access_token"] = token
	} else if token, _ := resp["token"].(string); token != "" {
		out["access_token"] = token
	}
	if refreshToken, _ := resp["refresh_token"].(string); refreshToken != "" {
		out["refresh_token"] = refreshToken
	}
	if expiresIn, ok := resp["expires_in"]; ok {
		out["expires_in"] = expiresIn
	}
	return out
}

func oauthTokenError(c echo.Context, status int, code, description string) error {
	return c.JSON(status, map[string]string{
		"error":             code,
		"error_description": description,
	})
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
	if h.usesInternalAuthorizationServer() {
		return []string{h.requestOrigin(c)}
	}
	return nil
}

func (h *Handler) usesInternalAuthorizationServer() bool {
	cfg := h.cfg.GetAuthConfig()
	return cfg.IsEnabled() && cfg.GetMode() == "saas" && len(cfg.AuthorizationServers) == 0
}

func (h *Handler) internalOAuth() *internalOAuthServer {
	if h.oauth == nil {
		h.oauth = newInternalOAuthServer()
	}
	return h.oauth
}

type oauthAuthorizationCode struct {
	Code                string
	AccountID           string
	RedirectURI         string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
}

type internalOAuthServer struct {
	mu      sync.Mutex
	codes   map[string]*oauthAuthorizationCode
	clients map[string]*oauthClient
}

func newInternalOAuthServer() *internalOAuthServer {
	return &internalOAuthServer{
		codes:   map[string]*oauthAuthorizationCode{},
		clients: map[string]*oauthClient{},
	}
}

func (s *internalOAuthServer) putCode(code *oauthAuthorizationCode) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code.Code] = code
}

func (s *internalOAuthServer) takeCode(raw string) *oauthAuthorizationCode {
	s.mu.Lock()
	defer s.mu.Unlock()
	code := s.codes[raw]
	delete(s.codes, raw)
	return code
}

type oauthClient struct {
	ClientID                string
	ClientName              string
	RedirectURIs            []string
	GrantTypes              []string
	ResponseTypes           []string
	TokenEndpointAuthMethod string
	Scope                   string
	CreatedAt               time.Time
}

func (s *internalOAuthServer) putClient(client *oauthClient) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client.ClientID] = client
}

func validPKCEVerifier(code *oauthAuthorizationCode, verifier string) bool {
	if code.CodeChallenge == "" {
		return true
	}
	if verifier == "" {
		return false
	}
	switch strings.ToUpper(code.CodeChallengeMethod) {
	case "", "PLAIN":
		return subtle.ConstantTimeCompare([]byte(verifier), []byte(code.CodeChallenge)) == 1
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		got := base64.RawURLEncoding.EncodeToString(sum[:])
		return subtle.ConstantTimeCompare([]byte(got), []byte(code.CodeChallenge)) == 1
	default:
		return false
	}
}

func newOAuthSecret(prefix string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return prefix + base64.RawURLEncoding.EncodeToString(buf), nil
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
