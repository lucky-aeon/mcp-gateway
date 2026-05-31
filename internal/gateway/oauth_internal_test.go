package gateway

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/identity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memoryIdentityStore struct {
	account      *identity.Account
	apiKey       *identity.APIKey
	refreshToken *identity.RefreshToken
}

func (s *memoryIdentityStore) Close(context.Context) error { return nil }
func (s *memoryIdentityStore) UpsertAdmin(context.Context, string, string, string) (*identity.Account, error) {
	return nil, errors.New("not implemented")
}
func (s *memoryIdentityStore) CreateAccount(context.Context, *identity.Account) error {
	return errors.New("not implemented")
}
func (s *memoryIdentityStore) FindAccountByEmail(_ context.Context, email string) (*identity.Account, error) {
	if s.account != nil && s.account.Email == strings.ToLower(strings.TrimSpace(email)) {
		return s.account, nil
	}
	return nil, errors.New("not found")
}
func (s *memoryIdentityStore) FindAccountByID(_ context.Context, id string) (*identity.Account, error) {
	if s.account != nil && s.account.ID == id {
		return s.account, nil
	}
	return nil, errors.New("not found")
}
func (s *memoryIdentityStore) UpsertWorkspaceMember(context.Context, *identity.WorkspaceMember) error {
	return errors.New("not implemented")
}
func (s *memoryIdentityStore) GetWorkspaceMember(context.Context, string, string) (*identity.WorkspaceMember, error) {
	return nil, errors.New("not found")
}
func (s *memoryIdentityStore) ListWorkspaceMembersByAccount(context.Context, string) ([]identity.WorkspaceMember, error) {
	return nil, nil
}
func (s *memoryIdentityStore) ListWorkspaceMembersByWorkspace(context.Context, string) ([]identity.WorkspaceMember, error) {
	return nil, nil
}
func (s *memoryIdentityStore) DeleteWorkspaceMembers(context.Context, string) error { return nil }
func (s *memoryIdentityStore) CreateWorkspace(context.Context, *identity.Workspace) error {
	return errors.New("not implemented")
}
func (s *memoryIdentityStore) GetWorkspace(context.Context, string) (*identity.Workspace, error) {
	return nil, errors.New("not found")
}
func (s *memoryIdentityStore) ListWorkspaces(context.Context) ([]identity.Workspace, error) {
	return nil, nil
}
func (s *memoryIdentityStore) DeleteWorkspace(context.Context, string) error { return nil }
func (s *memoryIdentityStore) CreateMCPServer(context.Context, *identity.MCPServer) error {
	return errors.New("not implemented")
}
func (s *memoryIdentityStore) GetMCPServer(context.Context, string, string) (*identity.MCPServer, error) {
	return nil, errors.New("not found")
}
func (s *memoryIdentityStore) ListMCPServers(context.Context, string) ([]identity.MCPServer, error) {
	return nil, nil
}
func (s *memoryIdentityStore) DeleteMCPServer(context.Context, string, string) error { return nil }
func (s *memoryIdentityStore) UpsertInstalledPackage(context.Context, *identity.InstalledPackage) error {
	return errors.New("not implemented")
}
func (s *memoryIdentityStore) GetInstalledPackage(context.Context, string, string) (*identity.InstalledPackage, error) {
	return nil, errors.New("not found")
}
func (s *memoryIdentityStore) ListInstalledPackages(context.Context, string) ([]identity.InstalledPackage, error) {
	return nil, nil
}
func (s *memoryIdentityStore) DeleteInstalledPackage(context.Context, string, string) error {
	return nil
}
func (s *memoryIdentityStore) CreateAPIKey(context.Context, *identity.APIKey) error {
	return errors.New("not implemented")
}
func (s *memoryIdentityStore) ListAPIKeysByAccount(context.Context, string) ([]identity.APIKey, error) {
	return nil, nil
}
func (s *memoryIdentityStore) FindAPIKeyByHash(_ context.Context, keyHash string) (*identity.APIKey, error) {
	if s.apiKey != nil && s.apiKey.KeyHash == keyHash {
		return s.apiKey, nil
	}
	return nil, errors.New("not found")
}
func (s *memoryIdentityStore) UpdateAPIKeyUsage(context.Context, string, time.Time) error { return nil }
func (s *memoryIdentityStore) RevokeAPIKey(context.Context, string, string) error         { return nil }
func (s *memoryIdentityStore) CreateRefreshToken(_ context.Context, token *identity.RefreshToken) error {
	s.refreshToken = token
	return nil
}
func (s *memoryIdentityStore) FindRefreshTokenByHash(_ context.Context, tokenHash string) (*identity.RefreshToken, error) {
	if s.refreshToken != nil && s.refreshToken.TokenHash == tokenHash {
		return s.refreshToken, nil
	}
	return nil, errors.New("not found")
}
func (s *memoryIdentityStore) DeleteRefreshToken(context.Context, string) error { return nil }
func (s *memoryIdentityStore) AppendAuditLog(context.Context, *identity.AuditLog) error {
	return nil
}
func (s *memoryIdentityStore) ListAuditLogs(context.Context, string, int) ([]identity.AuditLog, error) {
	return nil, nil
}

func TestOAuthTokenPasswordGrantUsesGatewayAccountPassword(t *testing.T) {
	hash, err := identity.HashPassword("secret123")
	require.NoError(t, err)
	store := &memoryIdentityStore{
		account: &identity.Account{
			ID:           "account-1",
			Email:        "user@example.com",
			PasswordHash: hash,
			DisplayName:  "User",
			Status:       "active",
		},
	}
	cfg := &config.Config{
		Auth: &config.AuthConfig{
			Enabled:               true,
			Mode:                  "saas",
			JWTSecret:             "test-secret",
			AccessTokenTTLMinutes: 120,
			RefreshTokenTTLHours:  720,
		},
	}
	h := &Handler{
		cfg:  cfg,
		auth: identity.NewService(cfg, store),
	}

	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("username", "user@example.com")
	form.Set("password", "secret123")
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	err = h.handleOAuthToken(echo.New().NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	assert.NoError(t, json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&resp))
	assert.NotEmpty(t, resp["access_token"])
	assert.NotEmpty(t, resp["refresh_token"])
	assert.Equal(t, "Bearer", resp["token_type"])
	assert.EqualValues(t, 7200, resp["expires_in"])

	principal, err := h.auth.ValidateBearer(context.Background(), resp["access_token"].(string))
	assert.NoError(t, err)
	assert.Equal(t, "account-1", principal.AccountID)
}

func TestOAuthDynamicClientRegistration(t *testing.T) {
	h := &Handler{
		cfg: &config.Config{
			Auth: &config.AuthConfig{
				Enabled: true,
				Mode:    "saas",
			},
		},
	}
	body := `{"client_name":"MCP Inspector","redirect_uris":["http://localhost:6274/oauth/callback"],"grant_types":["authorization_code","refresh_token"],"response_types":["code"],"token_endpoint_auth_method":"none"}`
	req := httptest.NewRequest(http.MethodPost, "/oauth/register", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	err := h.handleOAuthRegister(echo.New().NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	assert.NoError(t, json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&resp))
	assert.NotEmpty(t, resp["client_id"])
	assert.Equal(t, "MCP Inspector", resp["client_name"])
	assert.Equal(t, "none", resp["token_endpoint_auth_method"])
	assert.Equal(t, []any{"http://localhost:6274/oauth/callback"}, resp["redirect_uris"])
}

func TestOAuthAuthorizationCodeGrantUsesGatewayAccountPassword(t *testing.T) {
	hash, err := identity.HashPassword("secret123")
	require.NoError(t, err)
	store := &memoryIdentityStore{
		account: &identity.Account{
			ID:           "account-1",
			Email:        "user@example.com",
			PasswordHash: hash,
			DisplayName:  "User",
			Status:       "active",
		},
	}
	cfg := &config.Config{
		Auth: &config.AuthConfig{
			Enabled:               true,
			Mode:                  "saas",
			JWTSecret:             "test-secret",
			AccessTokenTTLMinutes: 120,
			RefreshTokenTTLHours:  720,
		},
	}
	h := &Handler{
		cfg:  cfg,
		auth: identity.NewService(cfg, store),
	}

	verifier := "verifier-123"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	form := url.Values{}
	form.Set("response_type", "code")
	form.Set("client_id", "test-client")
	form.Set("redirect_uri", "http://client.example/callback")
	form.Set("state", "state-1")
	form.Set("code_challenge", challenge)
	form.Set("code_challenge_method", "S256")
	form.Set("username", "user@example.com")
	form.Set("password", "secret123")
	req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	err = h.handleOAuthAuthorize(echo.New().NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get(echo.HeaderLocation)
	require.NotEmpty(t, location)
	redirected, err := url.Parse(location)
	require.NoError(t, err)
	code := redirected.Query().Get("code")
	require.NotEmpty(t, code)
	assert.Equal(t, "state-1", redirected.Query().Get("state"))

	tokenForm := url.Values{}
	tokenForm.Set("grant_type", "authorization_code")
	tokenForm.Set("code", code)
	tokenForm.Set("redirect_uri", "http://client.example/callback")
	tokenForm.Set("code_verifier", verifier)
	tokenReq := httptest.NewRequest(http.MethodPost, "/oauth/token", strings.NewReader(tokenForm.Encode()))
	tokenReq.Header.Set(echo.HeaderContentType, "application/x-www-form-urlencoded")
	tokenRec := httptest.NewRecorder()

	err = h.handleOAuthToken(echo.New().NewContext(tokenReq, tokenRec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, tokenRec.Code)

	var resp map[string]any
	assert.NoError(t, json.NewDecoder(strings.NewReader(tokenRec.Body.String())).Decode(&resp))
	assert.NotEmpty(t, resp["access_token"])
	assert.Equal(t, "Bearer", resp["token_type"])
}

func TestOAuthAuthorizationCodeGrantUsesGatewayAPIKey(t *testing.T) {
	rawAPIKey := "gk_test_oauth_login"
	hash, err := identity.HashPassword("secret123")
	require.NoError(t, err)
	store := &memoryIdentityStore{
		account: &identity.Account{
			ID:            "account-1",
			Email:         "user@example.com",
			PasswordHash:  hash,
			DisplayName:   "User",
			Status:        "active",
			IsSystemAdmin: true,
		},
		apiKey: &identity.APIKey{
			ID:        "key-1",
			AccountID: "account-1",
			KeyHash:   testHashToken(rawAPIKey),
			Status:    "active",
		},
	}
	cfg := &config.Config{
		Auth: &config.AuthConfig{
			Enabled:               true,
			Mode:                  "saas",
			JWTSecret:             "test-secret",
			AccessTokenTTLMinutes: 120,
			RefreshTokenTTLHours:  720,
		},
	}
	h := &Handler{
		cfg:  cfg,
		auth: identity.NewService(cfg, store),
	}

	verifier := "verifier-123"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])

	form := url.Values{}
	form.Set("response_type", "code")
	form.Set("client_id", "test-client")
	form.Set("redirect_uri", "http://client.example/callback")
	form.Set("state", "state-1")
	form.Set("code_challenge", challenge)
	form.Set("code_challenge_method", "S256")
	form.Set("auth_method", "api_key")
	form.Set("api_key", rawAPIKey)
	req := httptest.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	err = h.handleOAuthAuthorize(echo.New().NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, rec.Code)
	location := rec.Header().Get(echo.HeaderLocation)
	require.NotEmpty(t, location)
	redirected, err := url.Parse(location)
	require.NoError(t, err)
	assert.NotEmpty(t, redirected.Query().Get("code"))
	assert.Equal(t, "state-1", redirected.Query().Get("state"))
}

func testHashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
