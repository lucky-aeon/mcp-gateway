package gateway

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/identity"
)

var errInvalidAccessToken = errors.New("invalid access token")

func (h *Handler) validateMCPOAuthToken(c echo.Context, raw string) (*identity.Principal, error) {
	cfg := h.cfg.GetAuthConfig()

	// 优先尝试验证内部JWT token（管理后台颁发的）
	if h.auth != nil {
		if principal, err := h.auth.ValidateBearer(c.Request().Context(), raw); err == nil {
			return principal, nil
		}
	}

	// 如果内部验证失败，尝试OAuth token验证
	audience := h.tokenAudience(c)
	if strings.TrimSpace(cfg.TokenIntrospectionURL) != "" {
		return h.validateIntrospectedToken(c, raw, audience)
	}

	// 如果配置了外部OAuth，才进行JWKS验证
	if len(cfg.AuthorizationServers) > 0 {
		return h.validateJWTAccessToken(raw, audience)
	}

	return nil, errInvalidAccessToken
}

func (h *Handler) tokenAudience(c echo.Context) string {
	if aud := strings.TrimSpace(h.cfg.GetAuthConfig().TokenAudience); aud != "" {
		return aud
	}
	return h.protectedResource(c)
}

func (h *Handler) tokenIssuer() string {
	cfg := h.cfg.GetAuthConfig()
	if issuer := strings.TrimSpace(cfg.TokenIssuer); issuer != "" {
		return strings.TrimRight(issuer, "/")
	}
	for _, item := range cfg.AuthorizationServers {
		if item = strings.TrimSpace(item); item != "" {
			return strings.TrimRight(item, "/")
		}
	}
	return ""
}

func (h *Handler) validateJWTAccessToken(raw, audience string) (*identity.Principal, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(token *jwt.Token) (interface{}, error) {
		switch token.Method.(type) {
		case *jwt.SigningMethodRSA:
		default:
			return nil, fmt.Errorf("%w: unsupported signing method", errInvalidAccessToken)
		}
		kid, _ := token.Header["kid"].(string)
		return h.publicKeyForKID(kid)
	})
	if err != nil || token == nil || !token.Valid {
		return nil, errInvalidAccessToken
	}
	if err := h.validateCommonAccessTokenClaims(claims, audience); err != nil {
		return nil, err
	}
	return principalFromClaims(claims), nil
}

func (h *Handler) validateIntrospectedToken(c echo.Context, raw, audience string) (*identity.Principal, error) {
	cfg := h.cfg.GetAuthConfig()
	form := url.Values{}
	form.Set("token", raw)
	form.Set("token_type_hint", "access_token")

	req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodPost, cfg.TokenIntrospectionURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, errInvalidAccessToken
	}
	req.Header.Set(echo.HeaderContentType, "application/x-www-form-urlencoded")
	if cfg.TokenIntrospectionID != "" || cfg.TokenIntrospectionSecret != "" {
		req.SetBasicAuth(cfg.TokenIntrospectionID, cfg.TokenIntrospectionSecret)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errInvalidAccessToken
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errInvalidAccessToken
	}

	var claims jwt.MapClaims
	if err := json.NewDecoder(resp.Body).Decode(&claims); err != nil {
		return nil, errInvalidAccessToken
	}
	active, _ := claims["active"].(bool)
	if !active {
		return nil, errInvalidAccessToken
	}
	if err := h.validateCommonAccessTokenClaims(claims, audience); err != nil {
		return nil, err
	}
	return principalFromClaims(claims), nil
}

func (h *Handler) validateCommonAccessTokenClaims(claims jwt.MapClaims, audience string) error {
	issuer := h.tokenIssuer()
	if issuer != "" {
		claimIssuer, _ := claims["iss"].(string)
		if strings.TrimRight(claimIssuer, "/") != issuer {
			return fmt.Errorf("%w: issuer mismatch", errInvalidAccessToken)
		}
	}
	if audience != "" && !claimContainsString(claims["aud"], audience) {
		return fmt.Errorf("%w: audience mismatch", errInvalidAccessToken)
	}
	if !claimsContainScopes(claims, h.cfg.GetAuthConfig().RequiredScopes) {
		return fmt.Errorf("%w: insufficient scope", errInvalidAccessToken)
	}
	return nil
}

func principalFromClaims(claims jwt.MapClaims) *identity.Principal {
	sub, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	name, _ := claims["name"].(string)
	if name == "" {
		name, _ = claims["preferred_username"].(string)
	}
	if name == "" {
		name = sub
	}
	return &identity.Principal{
		AccountID:   sub,
		Email:       email,
		DisplayName: name,
		Role:        identity.RoleWorkspaceViewer,
		TokenType:   "oauth_access_token",
	}
}

func claimContainsString(value any, want string) bool {
	switch v := value.(type) {
	case string:
		return v == want
	case []string:
		for _, item := range v {
			if item == want {
				return true
			}
		}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == want {
				return true
			}
		}
	}
	return false
}

func claimsContainScopes(claims jwt.MapClaims, required []string) bool {
	if len(required) == 0 {
		return true
	}
	have := map[string]bool{}
	if scope, _ := claims["scope"].(string); scope != "" {
		for _, item := range strings.Fields(scope) {
			have[item] = true
		}
	}
	for _, key := range []string{"scp", "scopes"} {
		switch values := claims[key].(type) {
		case []string:
			for _, item := range values {
				have[item] = true
			}
		case []any:
			for _, item := range values {
				if s, ok := item.(string); ok {
					have[s] = true
				}
			}
		}
	}
	for _, scope := range required {
		if !have[scope] {
			return false
		}
	}
	return true
}

func (h *Handler) publicKeyForKID(kid string) (*rsa.PublicKey, error) {
	jwksURI, err := h.jwksURI()
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(jwksURI)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errInvalidAccessToken
	}
	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&set); err != nil {
		return nil, errInvalidAccessToken
	}
	for _, key := range set.Keys {
		if key.Kty != "RSA" || (kid != "" && key.Kid != kid) {
			continue
		}
		return rsaPublicKeyFromJWK(key.N, key.E)
	}
	return nil, errInvalidAccessToken
}

func (h *Handler) jwksURI() (string, error) {
	cfg := h.cfg.GetAuthConfig()
	if uri := strings.TrimSpace(cfg.TokenJWKSURI); uri != "" {
		return uri, nil
	}
	issuer := h.tokenIssuer()
	for _, candidate := range authorizationServerMetadataCandidates(issuer) {
		uri, err := fetchJWKSURI(candidate)
		if err == nil && uri != "" {
			return uri, nil
		}
	}
	return "", errInvalidAccessToken
}

func authorizationServerMetadataCandidates(issuer string) []string {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	if issuer == "" {
		return nil
	}
	u, err := url.Parse(issuer)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil
	}
	base := u.Scheme + "://" + u.Host
	path := strings.Trim(u.EscapedPath(), "/")
	if path == "" {
		return []string{
			issuer + "/.well-known/oauth-authorization-server",
			issuer + "/.well-known/openid-configuration",
		}
	}
	return []string{
		base + "/.well-known/oauth-authorization-server/" + path,
		base + "/.well-known/openid-configuration/" + path,
		issuer + "/.well-known/openid-configuration",
	}
}

func fetchJWKSURI(metadataURL string) (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(metadataURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errInvalidAccessToken
	}
	var metadata struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return "", err
	}
	return strings.TrimSpace(metadata.JWKSURI), nil
}

func rsaPublicKeyFromJWK(nRaw, eRaw string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nRaw)
	if err != nil {
		return nil, errInvalidAccessToken
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eRaw)
	if err != nil {
		return nil, errInvalidAccessToken
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, errInvalidAccessToken
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}
