package gateway

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newOAuthAuthTestHandler(t *testing.T) (*Handler, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	const kid = "test-key"

	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA",
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(privateKey.PublicKey.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.PublicKey.E)).Bytes()),
			}},
		})
	}))
	t.Cleanup(jwks.Close)

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub":   "user-1",
		"email": "user@example.com",
		"name":  "Test User",
		"iss":   "https://auth.example.com",
		"aud":   "http://example.com/stream",
		"scope": "mcp:read mcp:write",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	token.Header["kid"] = kid
	raw, err := token.SignedString(privateKey)
	require.NoError(t, err)

	return &Handler{
		cfg: config.Config{
			Auth: &config.AuthConfig{
				Enabled:              true,
				AuthorizationServers: []string{"https://auth.example.com"},
				TokenIssuer:          "https://auth.example.com",
				TokenJWKSURI:         jwks.URL,
				TokenAudience:        "http://example.com/stream",
				RequiredScopes:       []string{"mcp:read"},
				ScopesSupported:      []string{"mcp:read", "mcp:write"},
			},
			GatewayProtocol: "streamhttp",
		},
	}, raw
}

func TestMCPAuthRequiresAuthorizationBearer(t *testing.T) {
	h, _ := newOAuthAuthTestHandler(t)
	e := echo.New()
	handler := h.mcpAuthMiddleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/stream?api_key=secret", nil)
	req.Header.Set("Mcp-Session-Id", "session-1")
	rec := httptest.NewRecorder()

	err := handler(e.NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Contains(t, rec.Header().Get(echo.HeaderWWWAuthenticate), `resource_metadata="http://example.com/.well-known/oauth-protected-resource/stream"`)
	assert.Contains(t, rec.Header().Get(echo.HeaderWWWAuthenticate), `scope="mcp:read"`)
}

func TestMCPAuthAcceptsValidOAuthJWT(t *testing.T) {
	h, token := newOAuthAuthTestHandler(t)
	e := echo.New()
	handler := h.mcpAuthMiddleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodPost, "/stream", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+token)
	rec := httptest.NewRecorder()

	err := handler(e.NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestMCPAuthAcceptsIntrospectedOpaqueToken(t *testing.T) {
	introspection := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Basic bWNwOnNlY3JldA==", r.Header.Get(echo.HeaderAuthorization))
		assert.NoError(t, r.ParseForm())
		assert.Equal(t, "opaque-token", r.Form.Get("token"))
		_ = json.NewEncoder(w).Encode(map[string]any{
			"active": true,
			"sub":    "user-2",
			"iss":    "https://auth.example.com",
			"aud":    "http://example.com/stream",
			"scope":  "mcp:read",
		})
	}))
	t.Cleanup(introspection.Close)

	h := &Handler{
		cfg: config.Config{
			Auth: &config.AuthConfig{
				Enabled:                  true,
				AuthorizationServers:     []string{"https://auth.example.com"},
				TokenIssuer:              "https://auth.example.com",
				TokenIntrospectionURL:    introspection.URL,
				TokenIntrospectionID:     "mcp",
				TokenIntrospectionSecret: "secret",
				TokenAudience:            "http://example.com/stream",
				RequiredScopes:           []string{"mcp:read"},
			},
		},
	}
	e := echo.New()
	handler := h.mcpAuthMiddleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/stream", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer opaque-token")
	rec := httptest.NewRecorder()

	err := handler(e.NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, rec.Code)
}

func TestProtectedResourceMetadata(t *testing.T) {
	h, _ := newOAuthAuthTestHandler(t)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/stream", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "gateway.example.com")
	rec := httptest.NewRecorder()

	err := h.handleProtectedResourceMetadata(e.NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	assert.NoError(t, json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&resp))
	assert.Equal(t, "https://gateway.example.com/stream", resp["resource"])
	assert.Equal(t, []any{"https://auth.example.com"}, resp["authorization_servers"])
	assert.Equal(t, []any{"mcp:read", "mcp:write"}, resp["scopes_supported"])
}

func TestProtectedResourceMetadataUsesGatewayAsAuthorizationServerInSaaSMode(t *testing.T) {
	h := &Handler{
		cfg: config.Config{
			Auth: &config.AuthConfig{
				Enabled: true,
				Mode:    "saas",
			},
		},
	}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-protected-resource/stream", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "gateway.example.com")
	rec := httptest.NewRecorder()

	err := h.handleProtectedResourceMetadata(e.NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	assert.NoError(t, json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&resp))
	assert.Equal(t, "https://gateway.example.com/stream", resp["resource"])
	assert.Equal(t, []any{"https://gateway.example.com"}, resp["authorization_servers"])
}

func TestAuthorizationServerMetadataForInternalSaaSAuth(t *testing.T) {
	h := &Handler{
		cfg: config.Config{
			Auth: &config.AuthConfig{
				Enabled: true,
				Mode:    "saas",
			},
		},
	}
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/.well-known/oauth-authorization-server", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "gateway.example.com")
	rec := httptest.NewRecorder()

	err := h.handleAuthorizationServerMetadata(e.NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	assert.NoError(t, json.NewDecoder(strings.NewReader(rec.Body.String())).Decode(&resp))
	assert.Equal(t, "https://gateway.example.com", resp["issuer"])
	assert.Equal(t, "https://gateway.example.com/oauth/authorize", resp["authorization_endpoint"])
	assert.Equal(t, "https://gateway.example.com/oauth/token", resp["token_endpoint"])
	assert.Equal(t, "https://gateway.example.com/oauth/register", resp["registration_endpoint"])
	assert.Equal(t, []any{"authorization_code", "password", "refresh_token"}, resp["grant_types_supported"])
	assert.Equal(t, []any{"code"}, resp["response_types_supported"])
}

func TestMCPAuthRejectsInvalidTokenWithoutAuthorizationServerConfig(t *testing.T) {
	h := &Handler{
		cfg: config.Config{
			Auth: &config.AuthConfig{Enabled: true},
		},
	}
	e := echo.New()
	handler := h.mcpAuthMiddleware(func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})
	req := httptest.NewRequest(http.MethodPost, "/stream", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer anything")
	rec := httptest.NewRecorder()

	err := handler(e.NewContext(req, rec))

	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, rec.Header().Get(echo.HeaderWWWAuthenticate))
}
