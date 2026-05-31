package admin

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

const remoteOAuthAccessTokenEnv = "MCP_REMOTE_AUTH_ACCESS_TOKEN"

type mcpOAuthFlow struct {
	State        string
	ResourceURL  string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	CodeVerifier string
	TokenURL     string
	AuthorizeURL string
	AccessToken  string
	RefreshToken string
	Status       string
	Error        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type mcpOAuthFlowStore struct {
	mu    sync.RWMutex
	flows map[string]*mcpOAuthFlow
}

func newMCPOAuthFlowStore() *mcpOAuthFlowStore {
	return &mcpOAuthFlowStore{flows: map[string]*mcpOAuthFlow{}}
}

func (s *mcpOAuthFlowStore) put(flow *mcpOAuthFlow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if flow.CreatedAt.IsZero() {
		flow.CreatedAt = now
	}
	flow.UpdatedAt = now
	s.flows[flow.State] = flow
}

func (s *mcpOAuthFlowStore) get(state string) (*mcpOAuthFlow, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	flow, ok := s.flows[state]
	if !ok {
		return nil, false
	}
	cp := *flow
	return &cp, true
}

func (s *mcpOAuthFlowStore) update(state string, fn func(*mcpOAuthFlow)) (*mcpOAuthFlow, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	flow, ok := s.flows[state]
	if !ok {
		return nil, false
	}
	fn(flow)
	flow.UpdatedAt = time.Now().UTC()
	cp := *flow
	return &cp, true
}

func (h *Handler) handleV1StartMCPOAuth(c echo.Context) error {
	var req struct {
		ResourceURL string `json:"resource_url"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	resourceURL := strings.TrimSpace(req.ResourceURL)
	if resourceURL == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "resource_url is required", nil)
	}
	flow, err := h.startMCPOAuthFlow(c.Request().Context(), c, resourceURL)
	if err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "OAUTH_DISCOVERY_FAILED", err.Error(), nil)
	}
	h.oauth.put(flow)
	return respondOK(c, map[string]interface{}{
		"state":             flow.State,
		"authorization_url": flow.AuthorizeURL,
		"status":            flow.Status,
	})
}

func (h *Handler) handleV1MCPOAuthStatus(c echo.Context) error {
	flow, ok := h.oauth.get(c.Param("state"))
	if !ok {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "OAuth flow not found", nil)
	}
	return respondOK(c, mcpOAuthFlowView(flow))
}

func (h *Handler) handleV1MCPOAuthCallback(c echo.Context) error {
	state := strings.TrimSpace(c.QueryParam("state"))
	code := strings.TrimSpace(c.QueryParam("code"))
	if state == "" || code == "" {
		return c.HTML(http.StatusBadRequest, oauthCallbackHTML("OAuth failed", "missing state or code"))
	}
	flow, ok := h.oauth.get(state)
	if !ok {
		return c.HTML(http.StatusNotFound, oauthCallbackHTML("OAuth failed", "state not found"))
	}
	if errMsg := strings.TrimSpace(c.QueryParam("error")); errMsg != "" {
		h.oauth.update(state, func(item *mcpOAuthFlow) {
			item.Status = "failed"
			item.Error = errMsg
		})
		return c.HTML(http.StatusBadRequest, oauthCallbackHTML("OAuth failed", errMsg))
	}
	token, refresh, err := exchangeMCPOAuthCode(c.Request().Context(), flow, code)
	if err != nil {
		h.oauth.update(state, func(item *mcpOAuthFlow) {
			item.Status = "failed"
			item.Error = err.Error()
		})
		return c.HTML(http.StatusBadGateway, oauthCallbackHTML("OAuth failed", err.Error()))
	}
	h.oauth.update(state, func(item *mcpOAuthFlow) {
		item.Status = "authorized"
		item.AccessToken = token
		item.RefreshToken = refresh
		item.Error = ""
	})
	return c.HTML(http.StatusOK, oauthCallbackHTML("OAuth completed", "You can close this tab and return to MCP Gateway."))
}

func (h *Handler) startMCPOAuthFlow(ctx context.Context, c echo.Context, resourceURL string) (*mcpOAuthFlow, error) {
	authServer, err := discoverAuthorizationServer(ctx, resourceURL)
	if err != nil {
		return nil, err
	}
	meta, err := fetchAuthorizationServerMetadata(ctx, authServer)
	if err != nil {
		return nil, err
	}
	authEndpoint, _ := meta["authorization_endpoint"].(string)
	tokenEndpoint, _ := meta["token_endpoint"].(string)
	registrationEndpoint, _ := meta["registration_endpoint"].(string)
	if authEndpoint == "" || tokenEndpoint == "" {
		return nil, fmt.Errorf("authorization server metadata is missing authorization_endpoint or token_endpoint")
	}
	redirectURI := requestBaseURL(c) + "/api/v1/mcp-oauth/callback"
	clientID, clientSecret, err := registerOAuthClient(ctx, registrationEndpoint, redirectURI)
	if err != nil {
		return nil, err
	}
	state := randomURLToken(24)
	verifier := randomURLToken(48)
	challenge := codeChallenge(verifier)
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", clientID)
	params.Set("redirect_uri", redirectURI)
	params.Set("state", state)
	params.Set("code_challenge", challenge)
	params.Set("code_challenge_method", "S256")
	params.Set("resource", resourceURL)
	authorizeURL := authEndpoint
	if strings.Contains(authorizeURL, "?") {
		authorizeURL += "&" + params.Encode()
	} else {
		authorizeURL += "?" + params.Encode()
	}
	return &mcpOAuthFlow{
		State:        state,
		ResourceURL:  resourceURL,
		RedirectURI:  redirectURI,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		CodeVerifier: verifier,
		TokenURL:     tokenEndpoint,
		AuthorizeURL: authorizeURL,
		Status:       "pending",
	}, nil
}

func discoverAuthorizationServer(ctx context.Context, resourceURL string) (string, error) {
	parsed, err := url.Parse(resourceURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid resource_url")
	}
	candidates := []string{
		parsed.Scheme + "://" + parsed.Host + "/.well-known/oauth-protected-resource" + parsed.EscapedPath(),
		parsed.Scheme + "://" + parsed.Host + "/.well-known/oauth-protected-resource",
	}
	for _, candidate := range candidates {
		meta, err := getJSON(ctx, candidate)
		if err != nil {
			continue
		}
		if servers, ok := meta["authorization_servers"].([]interface{}); ok && len(servers) > 0 {
			if server, ok := servers[0].(string); ok && server != "" {
				return server, nil
			}
		}
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, resourceURL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if metadataURL := parseResourceMetadataURL(resp.Header.Get("WWW-Authenticate")); metadataURL != "" {
			if meta, metaErr := getJSON(ctx, metadataURL); metaErr == nil {
				if servers, ok := meta["authorization_servers"].([]interface{}); ok && len(servers) > 0 {
					if server, ok := servers[0].(string); ok && server != "" {
						return server, nil
					}
				}
			}
		}
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func fetchAuthorizationServerMetadata(ctx context.Context, issuer string) (map[string]interface{}, error) {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	candidates := []string{issuer + "/.well-known/oauth-authorization-server", issuer + "/.well-known/openid-configuration"}
	if strings.Contains(issuer, "/.well-known/") {
		candidates = append([]string{issuer}, candidates...)
	}
	for _, candidate := range candidates {
		meta, err := getJSON(ctx, candidate)
		if err == nil {
			return meta, nil
		}
	}
	return nil, fmt.Errorf("failed to discover authorization server metadata")
}

func registerOAuthClient(ctx context.Context, endpoint, redirectURI string) (string, string, error) {
	if strings.TrimSpace(endpoint) == "" {
		return "", "", fmt.Errorf("authorization server does not advertise dynamic client registration")
	}
	body := map[string]interface{}{
		"client_name":                "MCP Gateway",
		"redirect_uris":              []string{redirectURI},
		"grant_types":                []string{"authorization_code", "refresh_token"},
		"response_types":             []string{"code"},
		"token_endpoint_auth_method": "none",
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payload)))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("dynamic client registration failed: status %d", resp.StatusCode)
	}
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	clientID, _ := out["client_id"].(string)
	clientSecret, _ := out["client_secret"].(string)
	if clientID == "" {
		return "", "", fmt.Errorf("dynamic client registration response missing client_id")
	}
	return clientID, clientSecret, nil
}

func exchangeMCPOAuthCode(ctx context.Context, flow *mcpOAuthFlow, code string) (string, string, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", flow.RedirectURI)
	form.Set("client_id", flow.ClientID)
	form.Set("code_verifier", flow.CodeVerifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, flow.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if flow.ClientSecret != "" {
		req.SetBasicAuth(flow.ClientID, flow.ClientSecret)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("token exchange failed: status %d", resp.StatusCode)
	}
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	accessToken, _ := out["access_token"].(string)
	refreshToken, _ := out["refresh_token"].(string)
	if accessToken == "" {
		return "", "", fmt.Errorf("token response missing access_token")
	}
	return accessToken, refreshToken, nil
}

func getJSON(ctx context.Context, rawURL string) (map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s: status %d", rawURL, resp.StatusCode)
	}
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func parseResourceMetadataURL(header string) string {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		idx := strings.Index(strings.ToLower(part), "resource_metadata=")
		if idx >= 0 {
			return strings.Trim(strings.TrimSpace(part[idx+len("resource_metadata="):]), `"`)
		}
	}
	return ""
}

func requestBaseURL(c echo.Context) string {
	proto := strings.TrimSpace(c.Request().Header.Get("X-Forwarded-Proto"))
	if proto == "" {
		proto = c.Scheme()
	}
	return proto + "://" + c.Request().Host
}

func randomURLToken(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return base64.RawURLEncoding.EncodeToString(buf)
}

func codeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func mcpOAuthFlowView(flow *mcpOAuthFlow) map[string]interface{} {
	return map[string]interface{}{
		"state":             flow.State,
		"resource_url":      flow.ResourceURL,
		"authorization_url": flow.AuthorizeURL,
		"status":            flow.Status,
		"error":             flow.Error,
	}
}

func oauthCallbackHTML(title, message string) string {
	return `<!doctype html><html><head><meta charset="utf-8"><title>` + html.EscapeString(title) + `</title></head><body style="font-family:system-ui;padding:32px"><h1>` + html.EscapeString(title) + `</h1><p>` + html.EscapeString(message) + `</p><script>if(window.opener){window.opener.postMessage({type:"mcp-oauth-complete"},"*")}</script></body></html>`
}
