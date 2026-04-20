package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"golang.org/x/crypto/bcrypt"
)

const (
	RoleSystemAdmin     = "system_admin"
	RoleWorkspaceOwner  = "workspace_owner"
	RoleWorkspaceAdmin  = "workspace_admin"
	RoleWorkspaceViewer = "workspace_viewer"

	apiKeyStatusActive  = "active"
	apiKeyStatusRevoked = "revoked"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type Principal struct {
	AccountID     string
	Email         string
	DisplayName   string
	Role          string
	IsSystemAdmin bool
	WorkspaceID   string
	TokenType     string
}

type Account struct {
	ID            string    `bson:"id" json:"id"`
	Email         string    `bson:"email" json:"email"`
	PasswordHash  string    `bson:"password_hash,omitempty"`
	DisplayName   string    `bson:"display_name" json:"display_name"`
	Status        string    `bson:"status" json:"status"`
	IsSystemAdmin bool      `bson:"is_system_admin" json:"is_system_admin"`
	CreatedAt     time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt     time.Time `bson:"updated_at" json:"updated_at"`
}

type WorkspaceMember struct {
	ID          string    `bson:"id" json:"id"`
	WorkspaceID string    `bson:"workspace_id" json:"workspace_id"`
	AccountID   string    `bson:"account_id" json:"account_id"`
	Role        string    `bson:"role" json:"role"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

type Workspace struct {
	ID          string    `bson:"id" json:"id"`
	Name        string    `bson:"name" json:"name"`
	Description string    `bson:"description" json:"description"`
	CreatedAt   time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at" json:"updated_at"`
}

type MCPServer struct {
	ID          string                 `bson:"id" json:"id"`
	Name        string                 `bson:"name" json:"name"`
	WorkspaceID string                 `bson:"workspace_id" json:"workspace_id"`
	SourceType  string                 `bson:"source_type" json:"source_type"`
	SourceRef   string                 `bson:"source_ref" json:"source_ref"`
	Version     string                 `bson:"version" json:"version"`
	Config      map[string]interface{} `bson:"config" json:"config"`
	CreatedAt   time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time              `bson:"updated_at" json:"updated_at"`
}

type APIKey struct {
	ID          string     `bson:"id" json:"id"`
	AccountID   string     `bson:"account_id" json:"account_id"`
	WorkspaceID string     `bson:"workspace_id,omitempty" json:"workspace_id,omitempty"`
	Name        string     `bson:"name" json:"name"`
	KeyPrefix   string     `bson:"key_prefix" json:"key_prefix"`
	KeyHash     string     `bson:"key_hash"`
	Scope       []string   `bson:"scope" json:"scope"`
	Status      string     `bson:"status" json:"status"`
	ExpiresAt   *time.Time `bson:"expires_at,omitempty" json:"expires_at,omitempty"`
	LastUsedAt  *time.Time `bson:"last_used_at,omitempty" json:"last_used_at,omitempty"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
}

type RefreshToken struct {
	ID         string    `bson:"id"`
	AccountID  string    `bson:"account_id"`
	TokenHash  string    `bson:"token_hash"`
	ExpiresAt  time.Time `bson:"expires_at"`
	CreatedAt  time.Time `bson:"created_at"`
	LastUsedAt time.Time `bson:"last_used_at"`
}

type AuditLog struct {
	ID             string                 `bson:"id" json:"id"`
	ActorAccountID string                 `bson:"actor_account_id,omitempty" json:"actor_account_id,omitempty"`
	Action         string                 `bson:"action" json:"action"`
	ResourceType   string                 `bson:"resource_type" json:"resource_type"`
	ResourceID     string                 `bson:"resource_id" json:"resource_id"`
	WorkspaceID    string                 `bson:"workspace_id,omitempty" json:"workspace_id,omitempty"`
	Detail         map[string]interface{} `bson:"detail,omitempty" json:"detail,omitempty"`
	CreatedAt      time.Time              `bson:"created_at" json:"created_at"`
}

type Store interface {
	Close(context.Context) error
	UpsertAdmin(context.Context, string, string, string) (*Account, error)
	CreateAccount(context.Context, *Account) error
	FindAccountByEmail(context.Context, string) (*Account, error)
	FindAccountByID(context.Context, string) (*Account, error)
	UpsertWorkspaceMember(context.Context, *WorkspaceMember) error
	GetWorkspaceMember(context.Context, string, string) (*WorkspaceMember, error)
	ListWorkspaceMembersByAccount(context.Context, string) ([]WorkspaceMember, error)
	ListWorkspaceMembersByWorkspace(context.Context, string) ([]WorkspaceMember, error)
	DeleteWorkspaceMembers(context.Context, string) error
	CreateWorkspace(context.Context, *Workspace) error
	GetWorkspace(context.Context, string) (*Workspace, error)
	ListWorkspaces(context.Context) ([]Workspace, error)
	DeleteWorkspace(context.Context, string) error
	CreateMCPServer(context.Context, *MCPServer) error
	GetMCPServer(context.Context, string, string) (*MCPServer, error)
	ListMCPServers(context.Context, string) ([]MCPServer, error)
	DeleteMCPServer(context.Context, string, string) error
	CreateAPIKey(context.Context, *APIKey) error
	ListAPIKeysByAccount(context.Context, string) ([]APIKey, error)
	FindAPIKeyByHash(context.Context, string) (*APIKey, error)
	UpdateAPIKeyUsage(context.Context, string, time.Time) error
	RevokeAPIKey(context.Context, string, string) error
	CreateRefreshToken(context.Context, *RefreshToken) error
	FindRefreshTokenByHash(context.Context, string) (*RefreshToken, error)
	DeleteRefreshToken(context.Context, string) error
	AppendAuditLog(context.Context, *AuditLog) error
	ListAuditLogs(context.Context, string, int) ([]AuditLog, error)
}

type Service struct {
	cfg             *config.Config
	store           Store
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

func NewService(cfg *config.Config, store Store) *Service {
	authCfg := cfg.GetAuthConfig()
	return &Service{
		cfg:             cfg,
		store:           store,
		accessTokenTTL:  time.Duration(authCfg.AccessTokenTTLMinutes) * time.Minute,
		refreshTokenTTL: time.Duration(authCfg.RefreshTokenTTLHours) * time.Hour,
	}
}

func (s *Service) Close(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	return s.store.Close(ctx)
}

func (s *Service) Mode() string {
	return s.cfg.GetAuthConfig().GetMode()
}

func (s *Service) IsSaaS() bool {
	return s.Mode() == "saas"
}

func (s *Service) Bootstrap(ctx context.Context) error {
	if !s.IsSaaS() || s.store == nil {
		return nil
	}
	_, err := s.store.UpsertAdmin(ctx, s.cfg.Auth.AdminEmail, s.cfg.Auth.AdminDisplayName, s.cfg.Auth.AdminPassword)
	return err
}

func (s *Service) RegisterAccount(ctx context.Context, email, password, displayName string) (map[string]interface{}, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, ErrForbidden
	}
	if !s.cfg.GetAuthConfig().AllowRegister {
		return nil, errors.New("registration is disabled")
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, errors.New("email is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}
	if displayName == "" {
		displayName = email
	}

	// 检查邮箱是否已存在
	if _, err := s.store.FindAccountByEmail(ctx, email); err == nil {
		return nil, errors.New("email already exists")
	}

	// 密码哈希
	passwordHash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// 创建账号
	account := &Account{
		ID:            uuid.NewString(),
		Email:         email,
		PasswordHash:  passwordHash,
		DisplayName:   displayName,
		Status:        "active",
		IsSystemAdmin: false,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := s.store.CreateAccount(ctx, account); err != nil {
		return nil, err
	}

	// 返回账号信息
	return map[string]interface{}{
		"id":           account.ID,
		"email":        account.Email,
		"display_name": account.DisplayName,
		"status":       account.Status,
		"created_at":   account.CreatedAt.UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) AuthenticatePassword(ctx context.Context, email, password string) (map[string]interface{}, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, ErrUnauthorized
	}
	account, err := s.store.FindAccountByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, ErrUnauthorized
	}
	if account.Status != "active" {
		return nil, ErrUnauthorized
	}
	if bcrypt.CompareHashAndPassword([]byte(account.PasswordHash), []byte(password)) != nil {
		return nil, ErrUnauthorized
	}
	accessToken, err := s.issueAccessToken(account)
	if err != nil {
		return nil, err
	}
	refreshRaw, refreshHash, err := newOpaqueSecret("rt_")
	if err != nil {
		return nil, err
	}
	refresh := &RefreshToken{
		ID:         uuid.NewString(),
		AccountID:  account.ID,
		TokenHash:  refreshHash,
		ExpiresAt:  time.Now().UTC().Add(s.refreshTokenTTL),
		CreatedAt:  time.Now().UTC(),
		LastUsedAt: time.Now().UTC(),
	}
	if err := s.store.CreateRefreshToken(ctx, refresh); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"mode":          "saas",
		"token_type":    "Bearer",
		"token":         accessToken,
		"refresh_token": refreshRaw,
		"user":          s.accountView(account),
	}, nil
}

func (s *Service) RefreshAccessToken(ctx context.Context, refreshToken string) (map[string]interface{}, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, ErrUnauthorized
	}
	record, err := s.store.FindRefreshTokenByHash(ctx, hashToken(refreshToken))
	if err != nil || record.ExpiresAt.Before(time.Now().UTC()) {
		return nil, ErrUnauthorized
	}
	account, err := s.store.FindAccountByID(ctx, record.AccountID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	accessToken, err := s.issueAccessToken(account)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"mode":       "saas",
		"token_type": "Bearer",
		"token":      accessToken,
		"user":       s.accountView(account),
	}, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if !s.IsSaaS() || s.store == nil || strings.TrimSpace(refreshToken) == "" {
		return nil
	}
	return s.store.DeleteRefreshToken(ctx, hashToken(refreshToken))
}

func (s *Service) ValidateBearer(ctx context.Context, token string) (*Principal, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrUnauthorized
	}

	if !s.IsSaaS() {
		if token != s.cfg.GetAuthConfig().GetApiKey() {
			return nil, ErrUnauthorized
		}
		return &Principal{
			AccountID:     "admin",
			DisplayName:   "Administrator",
			Role:          RoleSystemAdmin,
			IsSystemAdmin: true,
			TokenType:     "system_api_key",
		}, nil
	}

	if strings.Count(token, ".") == 2 {
		if principal, err := s.validateJWT(token); err == nil {
			return principal, nil
		}
	}

	apiKey, err := s.store.FindAPIKeyByHash(ctx, hashToken(token))
	if err == nil && apiKey.Status == apiKeyStatusActive {
		if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now().UTC()) {
			return nil, ErrUnauthorized
		}
		account, accErr := s.store.FindAccountByID(ctx, apiKey.AccountID)
		if accErr != nil {
			return nil, ErrUnauthorized
		}
		_ = s.store.UpdateAPIKeyUsage(ctx, apiKey.ID, time.Now().UTC())
		role := RoleWorkspaceViewer
		isSystemAdmin := account.IsSystemAdmin
		if apiKey.WorkspaceID != "" {
			if member, memberErr := s.store.GetWorkspaceMember(ctx, apiKey.WorkspaceID, account.ID); memberErr == nil {
				role = member.Role
			}
		} else if account.IsSystemAdmin {
			role = RoleSystemAdmin
		}
		return &Principal{
			AccountID:     account.ID,
			Email:         account.Email,
			DisplayName:   account.DisplayName,
			Role:          role,
			IsSystemAdmin: isSystemAdmin,
			WorkspaceID:   apiKey.WorkspaceID,
			TokenType:     "api_key",
		}, nil
	}
	return nil, ErrUnauthorized
}

func (s *Service) CreateAPIKey(ctx context.Context, principal *Principal, name, workspaceID string, scope []string, expiresAt *time.Time) (map[string]interface{}, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, ErrForbidden
	}
	raw, hash, err := newOpaqueSecret("gk_")
	if err != nil {
		return nil, err
	}
	item := &APIKey{
		ID:          uuid.NewString(),
		AccountID:   principal.AccountID,
		WorkspaceID: workspaceID,
		Name:        name,
		KeyPrefix:   keyPrefix(raw),
		KeyHash:     hash,
		Scope:       scope,
		Status:      apiKeyStatusActive,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.store.CreateAPIKey(ctx, item); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"id":           item.ID,
		"name":         item.Name,
		"workspace_id": item.WorkspaceID,
		"scope":        item.Scope,
		"status":       item.Status,
		"expires_at":   item.ExpiresAt,
		"created_at":   item.CreatedAt.Format(time.RFC3339),
		"key_prefix":   item.KeyPrefix,
		"raw_key":      raw,
	}, nil
}

func (s *Service) ListAPIKeys(ctx context.Context, principal *Principal) ([]map[string]interface{}, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, ErrForbidden
	}
	items, err := s.store.ListAPIKeysByAccount(ctx, principal.AccountID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]interface{}{
			"id":           item.ID,
			"name":         item.Name,
			"workspace_id": item.WorkspaceID,
			"scope":        item.Scope,
			"status":       item.Status,
			"expires_at":   item.ExpiresAt,
			"last_used_at": item.LastUsedAt,
			"created_at":   item.CreatedAt.Format(time.RFC3339),
			"key_prefix":   item.KeyPrefix,
		})
	}
	return out, nil
}

func (s *Service) RevokeAPIKey(ctx context.Context, principal *Principal, keyID string) error {
	if !s.IsSaaS() || s.store == nil {
		return ErrForbidden
	}
	return s.store.RevokeAPIKey(ctx, keyID, principal.AccountID)
}

func (s *Service) AccountViewFromPrincipal(ctx context.Context, principal *Principal) (map[string]interface{}, error) {
	if !s.IsSaaS() {
		return map[string]interface{}{
			"id":           "admin",
			"email":        "",
			"display_name": "Administrator",
			"role":         "owner",
			"status":       "active",
			"builtin":      true,
			"created_at":   time.Now().UTC().Format(time.RFC3339),
		}, nil
	}
	account, err := s.store.FindAccountByID(ctx, principal.AccountID)
	if err != nil {
		return nil, err
	}
	return s.accountView(account), nil
}

func (s *Service) accountView(account *Account) map[string]interface{} {
	role := RoleWorkspaceViewer
	if account.IsSystemAdmin {
		role = RoleSystemAdmin
	}
	return map[string]interface{}{
		"id":           account.ID,
		"email":        account.Email,
		"display_name": account.DisplayName,
		"role":         role,
		"status":       account.Status,
		"builtin":      account.IsSystemAdmin,
		"created_at":   account.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Service) WorkspaceRole(ctx context.Context, workspaceID string, principal *Principal) (string, error) {
	if principal == nil {
		return "", ErrUnauthorized
	}
	if !s.IsSaaS() || principal.IsSystemAdmin {
		return RoleSystemAdmin, nil
	}
	if principal.WorkspaceID != "" && principal.WorkspaceID != workspaceID {
		return "", ErrForbidden
	}
	member, err := s.store.GetWorkspaceMember(ctx, workspaceID, principal.AccountID)
	if err != nil {
		return "", ErrForbidden
	}
	return member.Role, nil
}

func (s *Service) VisibleWorkspaceIDs(ctx context.Context, principal *Principal) (map[string]bool, error) {
	if principal == nil {
		return nil, ErrUnauthorized
	}
	if !s.IsSaaS() || principal.IsSystemAdmin {
		return nil, nil
	}
	items, err := s.store.ListWorkspaceMembersByAccount(ctx, principal.AccountID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(items))
	for _, item := range items {
		out[item.WorkspaceID] = true
	}
	if principal.WorkspaceID != "" {
		out[principal.WorkspaceID] = true
	}
	return out, nil
}

func (s *Service) AddWorkspaceOwner(ctx context.Context, workspaceID string, principal *Principal) error {
	if !s.IsSaaS() || s.store == nil || principal == nil {
		return nil
	}
	return s.store.UpsertWorkspaceMember(ctx, &WorkspaceMember{
		ID:          uuid.NewString(),
		WorkspaceID: workspaceID,
		AccountID:   principal.AccountID,
		Role:        RoleWorkspaceOwner,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	})
}

func (s *Service) ListWorkspaceMembers(ctx context.Context, workspaceID string) ([]WorkspaceMember, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, nil
	}
	return s.store.ListWorkspaceMembersByWorkspace(ctx, workspaceID)
}

func (s *Service) AddWorkspaceMember(ctx context.Context, workspaceID, accountEmail, role string) (*WorkspaceMember, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, ErrForbidden
	}
	account, err := s.store.FindAccountByEmail(ctx, strings.ToLower(strings.TrimSpace(accountEmail)))
	if err != nil {
		return nil, err
	}
	member := &WorkspaceMember{
		ID:          uuid.NewString(),
		WorkspaceID: workspaceID,
		AccountID:   account.ID,
		Role:        role,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := s.store.UpsertWorkspaceMember(ctx, member); err != nil {
		return nil, err
	}
	return member, nil
}

func (s *Service) DeleteWorkspaceMembers(ctx context.Context, workspaceID string) error {
	if !s.IsSaaS() || s.store == nil {
		return nil
	}
	return s.store.DeleteWorkspaceMembers(ctx, workspaceID)
}

func (s *Service) CreateWorkspace(ctx context.Context, ws *Workspace) error {
	if !s.IsSaaS() || s.store == nil {
		return nil
	}
	return s.store.CreateWorkspace(ctx, ws)
}

func (s *Service) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, nil
	}
	return s.store.GetWorkspace(ctx, id)
}

func (s *Service) ListWorkspaces(ctx context.Context) ([]Workspace, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, nil
	}
	return s.store.ListWorkspaces(ctx)
}

func (s *Service) DeleteWorkspace(ctx context.Context, id string) error {
	if !s.IsSaaS() || s.store == nil {
		return nil
	}
	return s.store.DeleteWorkspace(ctx, id)
}

func (s *Service) CreateMCPServer(ctx context.Context, server *MCPServer) error {
	if !s.IsSaaS() || s.store == nil {
		return nil
	}
	return s.store.CreateMCPServer(ctx, server)
}

func (s *Service) GetMCPServer(ctx context.Context, workspaceID, name string) (*MCPServer, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, nil
	}
	return s.store.GetMCPServer(ctx, workspaceID, name)
}

func (s *Service) ListMCPServers(ctx context.Context, workspaceID string) ([]MCPServer, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, nil
	}
	return s.store.ListMCPServers(ctx, workspaceID)
}

func (s *Service) DeleteMCPServer(ctx context.Context, workspaceID, name string) error {
	if !s.IsSaaS() || s.store == nil {
		return nil
	}
	return s.store.DeleteMCPServer(ctx, workspaceID, name)
}

func (s *Service) AppendAuditLog(ctx context.Context, principal *Principal, action, resourceType, resourceID, workspaceID string, detail map[string]interface{}) {
	if s.store == nil || !s.IsSaaS() {
		return
	}
	actorID := ""
	if principal != nil {
		actorID = principal.AccountID
	}
	_ = s.store.AppendAuditLog(ctx, &AuditLog{
		ID:             uuid.NewString(),
		ActorAccountID: actorID,
		Action:         action,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
		WorkspaceID:    workspaceID,
		Detail:         detail,
		CreatedAt:      time.Now().UTC(),
	})
}

func (s *Service) ListAuditLogs(ctx context.Context, workspaceID string, limit int) ([]AuditLog, error) {
	if !s.IsSaaS() || s.store == nil {
		return nil, nil
	}
	return s.store.ListAuditLogs(ctx, workspaceID, limit)
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func (s *Service) issueAccessToken(account *Account) (string, error) {
	claims := jwt.MapClaims{
		"sub":             account.ID,
		"email":           account.Email,
		"display_name":    account.DisplayName,
		"is_system_admin": account.IsSystemAdmin,
		"exp":             time.Now().UTC().Add(s.accessTokenTTL).Unix(),
		"iat":             time.Now().UTC().Unix(),
		"typ":             "access",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.Auth.JWTSecret))
}

func (s *Service) validateJWT(raw string) (*Principal, error) {
	token, err := jwt.Parse(raw, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, ErrUnauthorized
		}
		return []byte(s.cfg.Auth.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrUnauthorized
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != "access" {
		return nil, ErrUnauthorized
	}
	accountID, _ := claims["sub"].(string)
	email, _ := claims["email"].(string)
	displayName, _ := claims["display_name"].(string)
	isSystemAdmin, _ := claims["is_system_admin"].(bool)
	role := RoleWorkspaceViewer
	if isSystemAdmin {
		role = RoleSystemAdmin
	}
	return &Principal{
		AccountID:     accountID,
		Email:         email,
		DisplayName:   displayName,
		Role:          role,
		IsSystemAdmin: isSystemAdmin,
		TokenType:     "access_token",
	}, nil
}

func newOpaqueSecret(prefix string) (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = prefix + base64.RawURLEncoding.EncodeToString(buf)
	hash = hashToken(raw)
	return raw, hash, nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func keyPrefix(raw string) string {
	if len(raw) <= 10 {
		return raw
	}
	return raw[:10]
}

func newID() string {
	return uuid.NewString()
}

func RoleAllows(actual, required string) bool {
	order := map[string]int{
		RoleWorkspaceViewer: 1,
		RoleWorkspaceAdmin:  2,
		RoleWorkspaceOwner:  3,
		RoleSystemAdmin:     4,
	}
	return order[actual] >= order[required]
}

func NormalizeWorkspaceRole(role string) string {
	switch strings.TrimSpace(role) {
	case RoleWorkspaceOwner, RoleWorkspaceAdmin, RoleWorkspaceViewer:
		return role
	default:
		return RoleWorkspaceViewer
	}
}

func ValidateSaaSConfig(cfg *config.Config) error {
	if cfg.GetAuthConfig().GetMode() != "saas" {
		return nil
	}
	if strings.TrimSpace(cfg.Auth.MongoURI) == "" || strings.TrimSpace(cfg.Auth.MongoDatabase) == "" {
		return fmt.Errorf("saas mode requires auth.mongo_uri and auth.mongo_database")
	}
	if strings.TrimSpace(cfg.Auth.JWTSecret) == "" {
		return fmt.Errorf("saas mode requires auth.jwt_secret")
	}
	return nil
}
