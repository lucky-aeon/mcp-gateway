package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/identity"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/oplog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/sessions"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
	"github.com/mark3labs/mcp-go/mcp"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type envelope struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data"`
	Error     interface{} `json:"error"`
	Timestamp string      `json:"timestamp"`
}

type apiError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

type listData struct {
	Items    interface{} `json:"items"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

const (
	serviceDesiredStatusKey     = "desired_status"
	serviceDesiredStatusRunning = "running"
	serviceDesiredStatusStopped = "stopped"
)

func (h *Handler) registerV1Routes(e *echo.Echo) {
	publicV1 := e.Group("/api/v1")
	publicV1.GET("/meta", h.handleV1Meta)
	publicV1.POST("/auth/login", h.handleV1AuthLogin)
	publicV1.POST("/auth/register", h.handleV1AuthRegister)
	publicV1.POST("/auth/refresh", h.handleV1AuthRefresh)
	publicV1.GET("/mcp-oauth/callback", h.handleV1MCPOAuthCallback)

	v1 := e.Group("/api/v1")
	v1.Use(h.v1AuthMiddleware)

	v1.GET("/auth/me", h.handleV1AuthMe)
	v1.POST("/auth/logout", h.handleV1AuthLogout)
	v1.POST("/mcp-oauth/start", h.handleV1StartMCPOAuth)
	v1.GET("/mcp-oauth/status/:state", h.handleV1MCPOAuthStatus)
	v1.GET("/stats/overview", h.handleV1StatsOverview)

	v1.GET("/workspaces", h.handleV1ListWorkspaces)
	v1.POST("/workspaces", h.handleV1CreateWorkspace)
	v1.GET("/workspaces/:ws", h.handleV1GetWorkspace)
	v1.PATCH("/workspaces/:ws", h.handleV1PatchWorkspace)
	v1.DELETE("/workspaces/:ws", h.handleV1DeleteWorkspace)

	v1.GET("/workspaces/:ws/services", h.handleV1ListServices)
	v1.POST("/workspaces/:ws/services", h.handleV1CreateService)
	v1.POST("/workspaces/:ws/services:batch", h.handleV1BatchCreateServices)
	v1.POST("/workspaces/:ws/services:from-installed", h.handleV1CreateServiceFromInstalled)
	v1.PUT("/workspaces/:ws/services/:name", h.handleV1UpdateService)
	v1.DELETE("/workspaces/:ws/services/:name", h.handleV1DeleteService)
	v1.POST("/workspaces/:ws/services/:name/start", h.handleV1StartService)
	v1.POST("/workspaces/:ws/services/:name/stop", h.handleV1StopService)
	v1.POST("/workspaces/:ws/services/:name/restart", h.handleV1RestartService)
	v1.GET("/workspaces/:ws/services/:name/tools", h.handleV1GetServiceTools)
	v1.GET("/workspaces/:ws/services/:name/logs", h.handleV1GetServiceLogs)

	v1.GET("/workspaces/:ws/sessions", h.handleV1ListSessions)
	v1.POST("/workspaces/:ws/sessions", h.handleV1CreateSession)
	v1.DELETE("/workspaces/:ws/sessions/:id", h.handleV1DeleteSession)
	v1.GET("/sessions/:id", h.handleV1GetSession)

	v1.GET("/installed", h.handleV1Installed)
	v1.PATCH("/installed/:id", h.handleV1UpdateInstalled)
	v1.POST("/installed/:id/oauth/complete", h.handleV1CompleteInstalledOAuth)
	v1.DELETE("/installed/:id", h.handleV1DeleteInstalled)
	v1.GET("/api-keys", h.handleV1ListAPIKeys)
	v1.POST("/api-keys", h.handleV1CreateAPIKey)
	v1.POST("/api-keys/:id/revoke", h.handleV1RevokeAPIKey)
	v1.GET("/workspaces/:ws/members", h.handleV1ListWorkspaceMembers)
	v1.POST("/workspaces/:ws/members", h.handleV1AddWorkspaceMember)
	v1.GET("/audit-logs", h.handleV1ListAuditLogs)
	v1.GET("/market/sources", h.handleV1MarketSources)
	v1.POST("/market/sources/:id/sync", h.handleV1SyncMarketSource)
	v1.GET("/market/packages", h.handleV1MarketPackages)
	v1.POST("/market/packages", h.handleV1CreateMarketPackage)
	v1.GET("/market/packages/:id", h.handleV1MarketPackageDetail)
	v1.PATCH("/market/packages/:id", h.handleV1UpdateMarketPackage)
	v1.DELETE("/market/packages/:id", h.handleV1DeleteMarketPackage)
	v1.POST("/market/packages/:id/install", h.handleV1InstallMarketPackage)
	v1.GET("/system/config", h.handleV1SystemConfig)
	v1.PUT("/system/config", h.handleV1UpdateSystemConfig)
	v1.GET("/system/api-key", h.handleV1GetSystemAPIKey)
	v1.POST("/system/api-key/rotate", h.handleV1RotateSystemAPIKey)
	v1.GET("/workspaces/:ws/logs", h.handleV1WorkspaceLogs)
}

func (h *Handler) v1AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !h.cfg.GetAuthConfig().IsEnabled() {
			return next(c)
		}

		token := extractBearerToken(c.Request().Header.Get("Authorization"))
		if token == "" {
			return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "missing bearer token", nil)
		}
		if h.auth == nil {
			if token != h.cfg.GetAuthConfig().GetApiKey() {
				return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid api key", nil)
			}
			c.Set("auth.principal", &identity.Principal{
				AccountID:     "admin",
				DisplayName:   "Administrator",
				Role:          identity.RoleSystemAdmin,
				IsSystemAdmin: true,
				TokenType:     "system_api_key",
			})
			return next(c)
		}
		principal, err := h.auth.ValidateBearer(c.Request().Context(), token)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid bearer token", nil)
		}
		c.Set("auth.principal", principal)
		return next(c)
	}
}

func extractBearerToken(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	return ""
}

func (h *Handler) authMode() string {
	if h.auth != nil {
		return h.auth.Mode()
	}
	return h.cfg.GetAuthConfig().GetMode()
}

func (h *Handler) currentUser() map[string]interface{} {
	if h.auth != nil && h.auth.IsSaaS() {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"id":           "admin",
		"email":        "",
		"display_name": "Administrator",
		"role":         "owner",
		"status":       "active",
		"builtin":      true,
		"created_at":   time.Now().UTC().Format(time.RFC3339),
	}
}

func (h *Handler) currentPrincipal(c echo.Context) *identity.Principal {
	if v := c.Get("auth.principal"); v != nil {
		if principal, ok := v.(*identity.Principal); ok {
			return principal
		}
	}
	return &identity.Principal{
		AccountID:     "admin",
		DisplayName:   "Administrator",
		Role:          identity.RoleSystemAdmin,
		IsSystemAdmin: true,
		TokenType:     "system_api_key",
	}
}

func (h *Handler) workspaceRole(c echo.Context, wsID string) (string, error) {
	if h.auth == nil {
		return identity.RoleSystemAdmin, nil
	}
	return h.auth.WorkspaceRole(c.Request().Context(), wsID, h.currentPrincipal(c))
}

func (h *Handler) requireWorkspaceRole(c echo.Context, wsID, required string) error {
	role, err := h.workspaceRole(c, wsID)
	if err != nil {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "workspace access denied", nil)
	}
	if !identity.RoleAllows(role, required) {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "insufficient workspace permissions", nil)
	}
	return nil
}

func (h *Handler) visibleWorkspaceMap(c echo.Context) (map[string]bool, error) {
	if h.auth == nil {
		return nil, nil
	}
	return h.auth.VisibleWorkspaceIDs(c.Request().Context(), h.currentPrincipal(c))
}

func (h *Handler) appendAudit(c echo.Context, action, resourceType, resourceID, workspaceID string, detail map[string]interface{}) {
	principal := h.currentPrincipal(c)
	if h.auth == nil {
		h.appendOperation(c.Request().Context(), principal, oplog.LevelInfo, action, resourceType, resourceID, workspaceID, "", readableOperationMessage(action, resourceID), "", detail)
		return
	}
	h.auth.AppendAuditLog(c.Request().Context(), principal, action, resourceType, resourceID, workspaceID, detail)
	h.appendOperation(c.Request().Context(), principal, oplog.LevelInfo, action, resourceType, resourceID, workspaceID, "", readableOperationMessage(action, resourceID), "", detail)
}

func (h *Handler) appendOperation(ctx context.Context, principal *identity.Principal, level oplog.Level, action, resourceType, resourceID, workspaceID, sessionID, message, errText string, detail map[string]interface{}) {
	actorID := ""
	if principal != nil {
		actorID = principal.AccountID
	}
	fields := map[string]interface{}{
		"log_type":      "operation",
		"event_id":      uuid.NewString(),
		"action":        action,
		"workspace_id":  workspaceID,
		"session_id":    sessionID,
		"resource_type": resourceType,
		"resource_id":   resourceID,
		"actor_id":      actorID,
	}
	for k, v := range detail {
		fields[k] = v
	}
	if errText != "" {
		fields["error"] = errText
	}
	logger := xlog.NewLogger("control-plane").WithFields(fields)
	if level == oplog.LevelError {
		logger.Error(message)
		return
	}
	if level == oplog.LevelWarn {
		logger.Warn(message)
		return
	}
	if level == oplog.LevelDebug {
		logger.Debug(message)
		return
	}
	logger.Info(message)
	_ = ctx
}

func readableOperationMessage(action, resourceID string) string {
	names := map[string]string{
		"workspace.create":              "Workspace created",
		"workspace.update":              "Workspace updated",
		"workspace.delete":              "Workspace deleted",
		"service.create":                "MCP service created",
		"service.create_from_installed": "MCP service added from installed package",
		"service.update":                "MCP service updated",
		"service.delete":                "MCP service deleted",
		"service.start":                 "MCP service started",
		"service.stop":                  "MCP service stopped",
		"service.restart":               "MCP service restarted",
		"session.create":                "MCP session created",
		"session.delete":                "MCP session deleted",
		"api_key.create":                "API key created",
		"api_key.revoke":                "API key revoked",
		"workspace_member.add":          "Workspace member added",
		"market.install":                "Market package installed",
		"installed.update":              "Installed package updated",
		"installed.delete":              "Installed package deleted",
		"installed.oauth_complete":      "Installed package OAuth completed",
		"auth.login":                    "User logged in",
		"auth.register":                 "User registered",
	}
	msg, ok := names[action]
	if !ok {
		msg = action
	}
	if resourceID != "" {
		return msg + ": " + resourceID
	}
	return msg
}

func respondOK(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, envelope{
		Success:   true,
		Data:      data,
		Error:     nil,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func respondCreated(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusCreated, envelope{
		Success:   true,
		Data:      data,
		Error:     nil,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func respondError(c echo.Context, status int, code, message string, details interface{}) error {
	return c.JSON(status, envelope{
		Success: false,
		Data:    nil,
		Error: apiError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func parsePageParams(c echo.Context) (int, int) {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func paginate[T any](items []T, page, pageSize int) ([]T, int) {
	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return []T{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total
}

func (h *Handler) handleV1Meta(c echo.Context) error {
	features := map[string]bool{
		"market":    true,
		"team":      h.authMode() == "saas",
		"audit_log": h.authMode() == "saas",
	}
	return respondOK(c, map[string]interface{}{
		"mode":             h.authMode(),
		"allow_register":   h.cfg.GetAuthConfig().AllowRegister,
		"oauth_providers":  []string{},
		"gateway_protocol": h.cfg.GatewayProtocol,
		"version":          "v1",
		"login_methods": func() []string {
			if h.authMode() == "saas" {
				return []string{"password", "api_key"}
			}
			return []string{"api_key"}
		}(),
		"features": features,
	})
}

func (h *Handler) handleV1AuthLogin(c echo.Context) error {
	var req struct {
		APIKey   string `json:"api_key"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}

	switch h.authMode() {
	case "single-key":
		if strings.TrimSpace(req.APIKey) == "" {
			return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "api_key is required", nil)
		}
		if req.APIKey != h.cfg.GetAuthConfig().GetApiKey() {
			return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid api key", nil)
		}
		return respondOK(c, map[string]interface{}{
			"mode":       h.authMode(),
			"token_type": "Bearer",
			"token":      req.APIKey,
			"user":       h.currentUser(),
		})
	default:
		if strings.TrimSpace(req.APIKey) != "" {
			principal, err := h.auth.ValidateBearer(c.Request().Context(), req.APIKey)
			if err != nil {
				return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid api key", nil)
			}
			h.auth.AppendAuditLog(c.Request().Context(), principal, "auth.login", "api_key", "", principal.WorkspaceID, map[string]interface{}{"mode": "saas"})
			return respondOK(c, map[string]interface{}{
				"mode":       h.authMode(),
				"token_type": "Bearer",
				"token":      req.APIKey,
				"user": map[string]interface{}{
					"id":           principal.AccountID,
					"email":        principal.Email,
					"display_name": principal.DisplayName,
					"role":         principal.Role,
					"status":       "active",
					"builtin":      false,
				},
			})
		}
		if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
			return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "email and password are required", nil)
		}
		resp, err := h.auth.AuthenticatePassword(c.Request().Context(), req.Email, req.Password)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid email or password", nil)
		}
		h.auth.AppendAuditLog(c.Request().Context(), nil, "auth.login", "account", strings.ToLower(strings.TrimSpace(req.Email)), "", map[string]interface{}{"mode": "saas"})
		return respondOK(c, resp)
	}
}

func (h *Handler) handleV1AuthRegister(c echo.Context) error {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"display_name"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}

	if h.authMode() != "saas" {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "registration is unavailable in single-key mode", nil)
	}

	if strings.TrimSpace(req.Email) == "" || strings.TrimSpace(req.Password) == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "email and password are required", nil)
	}

	resp, err := h.auth.RegisterAccount(c.Request().Context(), req.Email, req.Password, req.DisplayName)
	if err != nil {
		if err.Error() == "registration is disabled" {
			return respondError(c, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
		}
		if err.Error() == "email already exists" {
			return respondError(c, http.StatusConflict, "CONFLICT", err.Error(), nil)
		}
		return respondError(c, http.StatusBadRequest, "BAD_REQUEST", err.Error(), nil)
	}

	// 自动为用户创建默认工作区
	accountID := resp["id"].(string)
	displayName := resp["display_name"].(string)
	defaultWorkspaceID := accountID + "-default"
	defaultWorkspaceName := displayName + " 的工作空间"

	meta := h.state.upsertWorkspace(defaultWorkspaceID, defaultWorkspaceName, "默认工作空间")
	if h.auth != nil {
		principal := &identity.Principal{
			AccountID: accountID,
			Email:     req.Email,
			Role:      identity.RoleWorkspaceOwner,
		}
		_ = h.auth.AddWorkspaceOwner(c.Request().Context(), meta.ID, principal)
		_ = h.auth.CreateWorkspace(c.Request().Context(), &identity.Workspace{
			ID:          meta.ID,
			Name:        meta.Name,
			Description: meta.Description,
			CreatedAt:   meta.CreatedAt,
			UpdatedAt:   meta.LastActivityAt,
		})
	}

	h.auth.AppendAuditLog(c.Request().Context(), nil, "auth.register", "account", strings.ToLower(strings.TrimSpace(req.Email)), "", map[string]interface{}{"mode": "saas"})
	return respondOK(c, resp)
}

func (h *Handler) handleV1AuthRefresh(c echo.Context) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	if h.authMode() != "saas" {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "refresh is unavailable in single-key mode", nil)
	}
	resp, err := h.auth.RefreshAccessToken(c.Request().Context(), req.RefreshToken)
	if err != nil {
		return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid refresh token", nil)
	}
	return respondOK(c, resp)
}

func (h *Handler) handleV1AuthMe(c echo.Context) error {
	principal := h.currentPrincipal(c)
	if h.auth != nil && h.auth.IsSaaS() {
		me, err := h.auth.AccountViewFromPrincipal(c.Request().Context(), principal)
		if err != nil {
			return respondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "failed to resolve current account", nil)
		}
		return respondOK(c, me)
	}
	return respondOK(c, h.currentUser())
}

func (h *Handler) handleV1AuthLogout(c echo.Context) error {
	if h.authMode() == "single-key" {
		return respondOK(c, map[string]bool{"logged_out": true})
	}
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.Bind(&req)
	_ = h.auth.Logout(c.Request().Context(), req.RefreshToken)
	return respondOK(c, map[string]bool{"logged_out": true})
}

func (h *Handler) handleV1StatsOverview(c echo.Context) error {
	if h.authMode() == "saas" && !h.currentPrincipal(c).IsSystemAdmin {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "stats overview requires system admin", nil)
	}
	workspacesList := h.listWorkspaceViews()
	if visible, err := h.visibleWorkspaceMap(c); err == nil && visible != nil {
		filtered := make([]workspaceView, 0, len(workspacesList))
		for _, ws := range workspacesList {
			if visible[ws.ID] {
				filtered = append(filtered, ws)
			}
		}
		workspacesList = filtered
	}
	running := 0
	activeSessions := 0
	for _, ws := range workspacesList {
		running += ws.McpCount
		activeSessions += ws.SessionCount
	}

	recent := make([]map[string]interface{}, 0)
	for _, item := range h.state.listActivities(20) {
		row := map[string]interface{}{
			"at":             item.At.Format(time.RFC3339),
			"type":           item.Type,
			"workspace_id":   item.WorkspaceID,
			"workspace_name": item.WorkspaceName,
			"message":        item.Message,
		}
		if item.ServiceName != "" {
			row["service_name"] = item.ServiceName
		}
		if item.SessionID != "" {
			row["session_id"] = item.SessionID
		}
		recent = append(recent, row)
	}

	return respondOK(c, map[string]interface{}{
		"workspaces_count": len(workspacesList),
		"running_mcps":     running,
		"failed_mcps_24h":  0,
		"active_sessions":  activeSessions,
		"recent_activity":  recent,
	})
}

type workspaceView struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	OwnerID        string `json:"owner_id"`
	Status         string `json:"status"`
	McpCount       int    `json:"mcp_count"`
	SessionCount   int    `json:"session_count"`
	CreatedAt      string `json:"created_at"`
	LastActivityAt string `json:"last_activity_at"`
}

func (h *Handler) listWorkspaceViews() []workspaceView {
	h.seedStateFromRuntime()
	wsMap := h.getWorkspaceMap()
	metas := h.state.listWorkspaces()
	items := make([]workspaceView, 0, len(metas))
	for _, meta := range metas {
		serviceCount := 0
		sessionCount := 0
		status := "stopped"
		if ws, ok := wsMap[meta.ID]; ok {
			serviceCount = len(ws.GetMcpServices())
			sessionCount = len(h.services.GetWorkspaceSessions(nilLogger{}, workspaces.NameArg{Workspace: meta.ID}))
			if serviceCount > 0 {
				status = "running"
			}
		}
		items = append(items, workspaceView{
			ID:             meta.ID,
			Name:           meta.Name,
			Description:    meta.Description,
			OwnerID:        "admin",
			Status:         status,
			McpCount:       serviceCount,
			SessionCount:   sessionCount,
			CreatedAt:      meta.CreatedAt.Format(time.RFC3339),
			LastActivityAt: meta.LastActivityAt.Format(time.RFC3339),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].LastActivityAt > items[j].LastActivityAt
	})
	return items
}

func (h *Handler) handleV1ListWorkspaces(c echo.Context) error {
	page, pageSize := parsePageParams(c)
	q := strings.TrimSpace(strings.ToLower(c.QueryParam("q")))
	statusFilter := strings.TrimSpace(strings.ToLower(c.QueryParam("status")))
	visible, err := h.visibleWorkspaceMap(c)
	if err != nil {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "failed to resolve workspace visibility", nil)
	}

	// 从数据库同步工作区到内存状态
	if h.auth != nil {
		dbWorkspaces, err := h.auth.ListWorkspaces(c.Request().Context())
		if err == nil {
			for _, dbWS := range dbWorkspaces {
				if _, ok := h.state.getWorkspace(dbWS.ID); !ok {
					h.state.upsertWorkspace(dbWS.ID, dbWS.Name, dbWS.Description)
				}
			}
		}
	}

	items := h.listWorkspaceViews()
	filtered := make([]workspaceView, 0, len(items))
	for _, item := range items {
		if visible != nil && !visible[item.ID] {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(item.Name+" "+item.Description+" "+item.ID), q) {
			continue
		}
		if statusFilter != "" && strings.ToLower(item.Status) != statusFilter {
			continue
		}
		filtered = append(filtered, item)
	}
	paged, total := paginate(filtered, page, pageSize)
	return respondOK(c, listData{
		Items:    paged,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *Handler) handleV1CreateWorkspace(c echo.Context) error {
	var req struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	if req.Name == "" && req.ID == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "workspace name or id is required", nil)
	}
	if req.ID == "" {
		req.ID = slugify(req.Name)
	}
	if req.ID == "" {
		req.ID = uuid.NewString()
	}
	if _, ok := h.state.getWorkspace(req.ID); ok {
		return respondError(c, http.StatusConflict, "CONFLICT", "workspace already exists", nil)
	}
	meta := h.state.upsertWorkspace(req.ID, req.Name, req.Description)
	if h.auth != nil {
		_ = h.auth.AddWorkspaceOwner(c.Request().Context(), meta.ID, h.currentPrincipal(c))
		// 保存到数据库
		_ = h.auth.CreateWorkspace(c.Request().Context(), &identity.Workspace{
			ID:          meta.ID,
			Name:        meta.Name,
			Description: meta.Description,
			CreatedAt:   meta.CreatedAt,
			UpdatedAt:   meta.LastActivityAt,
		})
	}
	h.state.appendActivity(activityItem{
		Type:          "workspace.created",
		WorkspaceID:   meta.ID,
		WorkspaceName: meta.Name,
		Message:       "workspace created",
	})
	h.appendAudit(c, "workspace.create", "workspace", meta.ID, meta.ID, map[string]interface{}{"name": meta.Name})
	return respondCreated(c, h.workspaceViewFromMeta(meta))
}

func (h *Handler) handleV1GetWorkspace(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceViewer); err != nil {
		return err
	}
	meta, ok := h.state.getWorkspace(wsID)
	if !ok {
		h.seedStateFromRuntime()
		meta, ok = h.state.getWorkspace(wsID)
		if !ok {
			return respondError(c, http.StatusNotFound, "WORKSPACE_NOT_FOUND", "workspace not found", nil)
		}
	}
	serviceItems := h.buildServiceViews(wsID)
	sessionsItems := h.buildSessionViews(wsID)
	mini := serviceItems
	if len(mini) > 5 {
		mini = mini[:5]
	}
	return respondOK(c, map[string]interface{}{
		"id":               meta.ID,
		"name":             meta.Name,
		"description":      meta.Description,
		"owner_id":         "admin",
		"status":           h.workspaceStatus(wsID),
		"mcp_count":        len(serviceItems),
		"session_count":    len(sessionsItems),
		"created_at":       meta.CreatedAt.Format(time.RFC3339),
		"last_activity_at": meta.LastActivityAt.Format(time.RFC3339),
		"mcps":             mini,
		"sessions_active":  len(sessionsItems),
	})
}

func (h *Handler) handleV1PatchWorkspace(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	if _, ok := h.state.getWorkspace(wsID); !ok {
		return respondError(c, http.StatusNotFound, "WORKSPACE_NOT_FOUND", "workspace not found", nil)
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	meta := h.state.upsertWorkspace(wsID, req.Name, req.Description)
	h.appendAudit(c, "workspace.update", "workspace", wsID, wsID, map[string]interface{}{"name": req.Name})
	return respondOK(c, h.workspaceViewFromMeta(meta))
}

func (h *Handler) handleV1DeleteWorkspace(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceOwner); err != nil {
		return err
	}
	cascade := c.QueryParam("cascade") == "true"
	services := h.services.GetMcpServices(nilLogger{}, workspaces.NameArg{Workspace: wsID})
	if len(services) > 0 && !cascade {
		return respondError(c, http.StatusConflict, "CONFLICT", "workspace still has deployed services", nil)
	}
	for name := range services {
		if err := h.services.DeleteServer(nilLogger{}, workspaces.NameArg{Workspace: wsID, Server: name}); err != nil {
			return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
		h.state.deleteService(wsID, name)
	}
	for _, sess := range h.services.GetWorkspaceSessions(nilLogger{}, workspaces.NameArg{Workspace: wsID}) {
		h.services.CloseProxySession(nilLogger{}, workspaces.NameArg{Workspace: wsID, Session: sess.GetId()})
	}
	if workspaceDeleter, ok := h.services.(interface {
		DeleteWorkspace(xlog.Logger, workspaces.NameArg)
	}); ok {
		workspaceDeleter.DeleteWorkspace(nilLogger{}, workspaces.NameArg{Workspace: wsID})
	}
	h.state.deleteWorkspace(wsID)
	if h.auth != nil {
		if dbServers, err := h.auth.ListMCPServers(c.Request().Context(), wsID); err == nil {
			for _, dbServer := range dbServers {
				if err := h.auth.DeleteMCPServer(c.Request().Context(), wsID, dbServer.Name); err != nil && !isServiceNotFoundError(err) {
					return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
				}
			}
		} else {
			return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
		if err := h.auth.DeleteWorkspaceMembers(c.Request().Context(), wsID); err != nil {
			return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
		if err := h.auth.DeleteWorkspace(c.Request().Context(), wsID); err != nil && !isServiceNotFoundError(err) {
			return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
	}
	h.appendAudit(c, "workspace.delete", "workspace", wsID, wsID, nil)
	return respondOK(c, map[string]string{"id": wsID})
}

type serviceView struct {
	Name            string            `json:"name"`
	WorkspaceID     string            `json:"workspace_id"`
	SourceType      string            `json:"source_type"`
	SourceRef       string            `json:"source_ref"`
	Command         string            `json:"command,omitempty"`
	Args            []string          `json:"args,omitempty"`
	Env             map[string]string `json:"env,omitempty"`
	URL             string            `json:"url,omitempty"`
	GatewayProtocol string            `json:"gateway_protocol,omitempty"`
	AuthStatus      string            `json:"auth_status,omitempty"`
	Status          string            `json:"status"`
	Port            int               `json:"port,omitempty"`
	ToolsCount      int               `json:"tools_count"`
	LastError       string            `json:"last_error,omitempty"`
	RetryCount      int               `json:"retry_count"`
	CreatedAt       string            `json:"created_at"`
}

func (h *Handler) buildServiceViews(workspaceID string) []serviceView {
	h.seedStateFromRuntime()
	services := h.services.GetMcpServices(nilLogger{}, workspaces.NameArg{Workspace: workspaceID})
	items := make([]serviceView, 0, len(services))

	// 获取内存中所有服务元数据
	allMeta := h.state.listServices(workspaceID)
	metaMap := make(map[string]*serviceMeta)
	for _, meta := range allMeta {
		metaMap[meta.Name] = meta
	}

	// 添加运行中的服务
	for name, svc := range services {
		info := svc.Info()
		meta, _ := h.state.getService(workspaceID, name)
		sourceType := "command"
		sourceRef := ""
		createdAt := info.DeployedAt
		if meta != nil {
			if meta.SourceType != "" {
				sourceType = meta.SourceType
			}
			sourceRef = meta.SourceRef
			if !meta.CreatedAt.IsZero() {
				createdAt = meta.CreatedAt
			}
		} else if info.Config.URL != "" {
			sourceType = "url"
		}
		items = append(items, serviceView{
			Name:            name,
			WorkspaceID:     workspaceID,
			SourceType:      sourceType,
			SourceRef:       sourceRef,
			Command:         info.Config.Command,
			Args:            info.Config.Args,
			Env:             maskEnv(info.Config.Env),
			URL:             info.Config.URL,
			GatewayProtocol: info.Config.GatewayProtocol,
			AuthStatus:      serviceOAuthStatus(info.Config),
			Status:          normalizeServiceStatus(info.Status),
			Port:            info.Port,
			ToolsCount:      h.serviceToolsCount(workspaceID, name),
			LastError:       info.LastError,
			RetryCount:      info.RetryCount,
			CreatedAt:       createdAt.UTC().Format(time.RFC3339),
		})
		delete(metaMap, name)
	}

	// 添加已停止但在数据库中的服务
	for _, meta := range metaMap {
		sourceType := meta.SourceType
		if sourceType == "" {
			sourceType = "command"
		}
		items = append(items, serviceView{
			Name:        meta.Name,
			WorkspaceID: meta.WorkspaceID,
			SourceType:  sourceType,
			SourceRef:   meta.SourceRef,
			Status:      "stopped",
			ToolsCount:  0,
			CreatedAt:   meta.CreatedAt.UTC().Format(time.RFC3339),
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items
}

func (h *Handler) handleV1ListServices(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceViewer); err != nil {
		return err
	}

	// 从数据库同步 MCP 服务到内存状态（不自动重新部署）
	if h.auth != nil {
		dbServers, err := h.auth.ListMCPServers(c.Request().Context(), wsID)
		if err == nil {
			for _, dbServer := range dbServers {
				if _, ok := h.state.getService(wsID, dbServer.Name); !ok {
					// 只添加到内存状态，不重新部署
					h.state.upsertService(wsID, serviceMeta{
						Name:          dbServer.Name,
						WorkspaceID:   dbServer.WorkspaceID,
						SourceType:    dbServer.SourceType,
						SourceRef:     dbServer.SourceRef,
						Version:       dbServer.Version,
						CreatedAt:     dbServer.CreatedAt,
						InstalledFrom: "database",
					})
				}
			}
		}
	}

	return respondOK(c, listData{
		Items:    h.buildServiceViews(wsID),
		Total:    len(h.buildServiceViews(wsID)),
		Page:     1,
		PageSize: len(h.buildServiceViews(wsID)),
	})
}

func (h *Handler) handleV1CreateService(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	h.state.ensureWorkspace(wsID)
	raw := make(map[string]interface{})
	if err := c.Bind(&raw); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}

	name, cfg, meta, err := h.parseServiceRequest(c.Request().Context(), wsID, raw)
	if err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), nil)
	}
	result, err := h.DeployServer(name, cfg)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "MCP_DEPLOY_FAILED", err.Error(), nil)
	}
	meta.Name = name
	meta.WorkspaceID = wsID
	h.state.upsertService(wsID, meta)
	// 保存到数据库
	if h.auth != nil {
		_ = h.auth.CreateMCPServer(c.Request().Context(), &identity.MCPServer{
			ID:          uuid.NewString(),
			Name:        name,
			WorkspaceID: wsID,
			SourceType:  meta.SourceType,
			SourceRef:   meta.SourceRef,
			Version:     meta.Version,
			Config:      serviceConfigToMapWithDesiredStatus(cfg, serviceDesiredStatusRunning),
			CreatedAt:   meta.CreatedAt,
			UpdatedAt:   time.Now().UTC(),
		})
	}
	h.state.appendActivity(activityItem{
		Type:          "mcp.deployed",
		WorkspaceID:   wsID,
		WorkspaceName: h.workspaceName(wsID),
		ServiceName:   name,
		Message:       fmt.Sprintf("service %s %s", name, result),
	})
	h.appendAudit(c, "service.create", "service", name, wsID, map[string]interface{}{"workspace_id": wsID})
	view := h.findServiceView(wsID, name)
	return respondCreated(c, view)
}

func (h *Handler) handleV1CreateServiceFromInstalled(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	var req struct {
		InstalledID string            `json:"installed_id"`
		ServiceName string            `json:"service_name"`
		Env         map[string]string `json:"env"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	if strings.TrimSpace(req.InstalledID) == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "installed_id is required", nil)
	}
	installed, err := h.getAccountInstalledPackage(c.Request().Context(), h.currentPrincipal(c).AccountID, req.InstalledID)
	if err != nil {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "installed package not found", nil)
	}
	serviceName := strings.TrimSpace(req.ServiceName)
	if serviceName == "" {
		serviceName = slugify(valueOrDefault(installed.PackageName, installed.DisplayName))
	}
	if serviceName == "" {
		serviceName = installed.PackageID
	}
	cfg := serviceConfigFromMap(installed.ConfigSnapshot, wsID)
	for k, v := range req.Env {
		cfg.Env[k] = v
	}
	if auth := installedPackageAuthState(installed); auth != nil && auth.Status != "authorized" {
		return respondError(c, http.StatusConflict, "OAUTH_REQUIRED", "OAuth authorization is required before adding this MCP to a workspace", auth)
	}
	result, err := h.DeployServer(serviceName, cfg)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "MCP_DEPLOY_FAILED", err.Error(), nil)
	}
	meta := serviceMeta{
		Name:          serviceName,
		WorkspaceID:   wsID,
		SourceType:    "installed",
		SourceRef:     installed.ID,
		Version:       installed.Version,
		CreatedAt:     time.Now().UTC(),
		InstalledFrom: "account",
	}
	h.state.ensureWorkspace(wsID)
	h.state.upsertService(wsID, meta)
	if h.auth != nil {
		_ = h.auth.CreateMCPServer(c.Request().Context(), &identity.MCPServer{
			ID:          uuid.NewString(),
			Name:        serviceName,
			WorkspaceID: wsID,
			SourceType:  meta.SourceType,
			SourceRef:   meta.SourceRef,
			Version:     meta.Version,
			Config:      serviceConfigToMapWithDesiredStatus(cfg, serviceDesiredStatusRunning),
			CreatedAt:   meta.CreatedAt,
			UpdatedAt:   time.Now().UTC(),
		})
	}
	h.state.appendActivity(activityItem{
		Type:          "mcp.deployed",
		WorkspaceID:   wsID,
		WorkspaceName: h.workspaceName(wsID),
		ServiceName:   serviceName,
		Message:       fmt.Sprintf("added %s from installed package: %s", serviceName, result),
	})
	h.appendAudit(c, "service.create_from_installed", "service", serviceName, wsID, map[string]interface{}{
		"installed_id": installed.ID,
		"package_id":   installed.PackageID,
	})
	return respondCreated(c, h.findServiceView(wsID, serviceName))
}

func (h *Handler) handleV1BatchCreateServices(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	var req struct {
		MCPServers map[string]map[string]interface{} `json:"mcpServers"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	summary := map[string]int{
		"total":    len(req.MCPServers),
		"deployed": 0,
		"existed":  0,
		"replaced": 0,
		"failed":   0,
	}
	results := make(map[string]map[string]string, len(req.MCPServers))
	for name, body := range req.MCPServers {
		body["name"] = name
		_, cfg, meta, err := h.parseServiceRequest(c.Request().Context(), wsID, body)
		if err != nil {
			results[name] = map[string]string{"status": "failed", "message": err.Error()}
			summary["failed"]++
			continue
		}
		res, err := h.DeployServer(name, cfg)
		if err != nil {
			results[name] = map[string]string{"status": "failed", "message": err.Error()}
			summary["failed"]++
			continue
		}
		meta.Name = name
		meta.WorkspaceID = wsID
		h.state.upsertService(wsID, meta)
		if h.auth != nil {
			_ = h.auth.CreateMCPServer(c.Request().Context(), &identity.MCPServer{
				ID:          uuid.NewString(),
				Name:        name,
				WorkspaceID: wsID,
				SourceType:  meta.SourceType,
				SourceRef:   meta.SourceRef,
				Version:     meta.Version,
				Config:      serviceConfigToMapWithDesiredStatus(cfg, serviceDesiredStatusRunning),
				CreatedAt:   meta.CreatedAt,
				UpdatedAt:   time.Now().UTC(),
			})
		}
		results[name] = map[string]string{"status": string(res), "message": "ok"}
		summary[string(res)]++
	}
	return respondOK(c, map[string]interface{}{
		"summary": summary,
		"results": results,
	})
}

func (h *Handler) handleV1UpdateService(c echo.Context) error {
	wsID := c.Param("ws")
	name := c.Param("name")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	raw := make(map[string]interface{})
	if err := c.Bind(&raw); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	raw["name"] = name
	_, cfg, meta, err := h.parseServiceRequest(c.Request().Context(), wsID, raw)
	if err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), nil)
	}
	if err := h.services.DeleteServer(nilLogger{}, workspaces.NameArg{Workspace: wsID, Server: name}); err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	if _, err := h.DeployServer(name, cfg); err != nil {
		return respondError(c, http.StatusInternalServerError, "MCP_DEPLOY_FAILED", err.Error(), nil)
	}
	meta.Name = name
	meta.WorkspaceID = wsID
	h.state.upsertService(wsID, meta)
	if h.auth != nil {
		createdAt := meta.CreatedAt
		if existing, err := h.auth.GetMCPServer(c.Request().Context(), wsID, name); err == nil && existing != nil && !existing.CreatedAt.IsZero() {
			createdAt = existing.CreatedAt
		}
		_ = h.auth.CreateMCPServer(c.Request().Context(), &identity.MCPServer{
			ID:          uuid.NewString(),
			Name:        name,
			WorkspaceID: wsID,
			SourceType:  meta.SourceType,
			SourceRef:   meta.SourceRef,
			Version:     meta.Version,
			Config:      serviceConfigToMapWithDesiredStatus(cfg, serviceDesiredStatusRunning),
			CreatedAt:   createdAt,
			UpdatedAt:   time.Now().UTC(),
		})
	}
	h.appendAudit(c, "service.update", "service", name, wsID, nil)
	return respondOK(c, h.findServiceView(wsID, name))
}

func (h *Handler) handleV1DeleteService(c echo.Context) error {
	wsID := c.Param("ws")
	name := c.Param("name")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	if err := h.services.DeleteServer(nilLogger{}, workspaces.NameArg{Workspace: wsID, Server: name}); err != nil {
		if !isServiceNotFoundError(err) {
			return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
	}
	h.state.deleteService(wsID, name)
	// 从数据库删除
	if h.auth != nil {
		if err := h.auth.DeleteMCPServer(c.Request().Context(), wsID, name); err != nil && !isServiceNotFoundError(err) {
			return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
	}
	h.appendAudit(c, "service.delete", "service", name, wsID, nil)
	return respondOK(c, map[string]string{"name": name})
}

func (h *Handler) handleV1StartService(c echo.Context) error {
	wsID := c.Param("ws")
	name := c.Param("name")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceAdmin); err != nil {
		return err
	}

	var cfg config.MCPServerConfig
	var found bool

	// 先从数据库读取配置
	if h.auth != nil {
		dbServer, err := h.auth.GetMCPServer(c.Request().Context(), wsID, name)
		if err == nil && dbServer != nil {
			cfg = config.MCPServerConfig{
				Workspace: wsID,
			}
			if dbServer.Config != nil {
				cfg = serviceConfigFromMap(dbServer.Config, wsID)
			}
			found = true
		}
	}

	// 如果数据库中没有，从运行时读取
	if !found {
		cfgs := h.services.ListServerConfig(nilLogger{}, workspaces.NameArg{Workspace: wsID})
		var ok bool
		cfg, ok = cfgs[name]
		if !ok {
			h.appendOperation(c.Request().Context(), h.currentPrincipal(c), oplog.LevelError, "service.start_failed", "service", name, wsID, "", "service start failed", "service not found", nil)
			return respondError(c, http.StatusNotFound, "NOT_FOUND", "service not found", nil)
		}
	}

	if _, err := h.DeployServer(name, cfg); err != nil {
		h.appendOperation(c.Request().Context(), h.currentPrincipal(c), oplog.LevelError, "service.start_failed", "service", name, wsID, "", "service start failed", err.Error(), nil)
		return respondError(c, http.StatusInternalServerError, "MCP_DEPLOY_FAILED", err.Error(), nil)
	}
	h.markStoredServiceDesiredStatus(c.Request().Context(), wsID, name, serviceDesiredStatusRunning)

	// 等待一小段时间，确保服务启动完成
	time.Sleep(100 * time.Millisecond)

	// 从数据库读取服务元数据并更新到内存状态
	if h.auth != nil {
		dbServer, err := h.auth.GetMCPServer(c.Request().Context(), wsID, name)
		if err == nil && dbServer != nil {
			meta := serviceMeta{
				Name:          dbServer.Name,
				WorkspaceID:   dbServer.WorkspaceID,
				SourceType:    dbServer.SourceType,
				SourceRef:     dbServer.SourceRef,
				Version:       dbServer.Version,
				CreatedAt:     dbServer.CreatedAt,
				InstalledFrom: "database",
			}
			h.state.upsertService(wsID, meta)
		}
	}

	h.appendAudit(c, "service.start", "service", name, wsID, nil)
	return respondOK(c, map[string]string{"status": "running"})
}

func (h *Handler) handleV1StopService(c echo.Context) error {
	if err := h.requireWorkspaceRole(c, c.Param("ws"), identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	h.services.StopServer(nilLogger{}, workspaces.NameArg{Workspace: c.Param("ws"), Server: c.Param("name")})
	h.markStoredServiceDesiredStatus(c.Request().Context(), c.Param("ws"), c.Param("name"), serviceDesiredStatusStopped)
	h.appendAudit(c, "service.stop", "service", c.Param("name"), c.Param("ws"), nil)
	return respondOK(c, map[string]string{"status": "stopped"})
}

func (h *Handler) handleV1RestartService(c echo.Context) error {
	if err := h.requireWorkspaceRole(c, c.Param("ws"), identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	if err := h.services.RestartServer(nilLogger{}, workspaces.NameArg{Workspace: c.Param("ws"), Server: c.Param("name")}); err != nil {
		h.appendOperation(c.Request().Context(), h.currentPrincipal(c), oplog.LevelError, "service.restart_failed", "service", c.Param("name"), c.Param("ws"), "", "service restart failed", err.Error(), nil)
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.appendAudit(c, "service.restart", "service", c.Param("name"), c.Param("ws"), nil)
	return respondOK(c, map[string]string{"status": "running"})
}

func (h *Handler) handleV1GetServiceTools(c echo.Context) error {
	if err := h.requireWorkspaceRole(c, c.Param("ws"), identity.RoleWorkspaceViewer); err != nil {
		return err
	}
	tools := h.findServiceTools(c.Param("ws"), c.Param("name"))
	return respondOK(c, map[string]interface{}{"items": tools})
}

func (h *Handler) handleV1GetServiceLogs(c echo.Context) error {
	wsID := c.Param("ws")
	name := c.Param("name")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceViewer); err != nil {
		return err
	}
	tail, _ := strconv.Atoi(c.QueryParam("tail"))
	if tail <= 0 {
		tail = 200
	}
	lines, err := h.readServiceLogs(wsID, name, tail)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return respondOK(c, map[string]interface{}{
		"service_name": name,
		"total_lines":  len(lines),
		"logs":         lines,
	})
}

type sessionView struct {
	ID              string   `json:"id"`
	WorkspaceID     string   `json:"workspace_id"`
	Status          string   `json:"status"`
	IsReady         bool     `json:"is_ready"`
	ToolsCount      int      `json:"tools_count"`
	BoundMCPNames   []string `json:"bound_mcp_names"`
	CreatedAt       string   `json:"created_at"`
	LastReceiveTime string   `json:"last_receive_time"`
}

func (h *Handler) buildSessionViews(wsID string) []sessionView {
	sessionsList := h.services.GetWorkspaceSessions(nilLogger{}, workspaces.NameArg{Workspace: wsID})
	views := make([]sessionView, 0, len(sessionsList))
	serviceNames := make([]string, 0)
	for name := range h.services.GetMcpServices(nilLogger{}, workspaces.NameArg{Workspace: wsID}) {
		serviceNames = append(serviceNames, name)
	}
	sort.Strings(serviceNames)
	for _, sess := range sessionsList {
		views = append(views, sessionView{
			ID:              sess.GetId(),
			WorkspaceID:     wsID,
			Status:          "active",
			IsReady:         sess.IsToolsListReady(),
			ToolsCount:      len(sess.GetAllTools()),
			BoundMCPNames:   serviceNames,
			CreatedAt:       sess.CreatedAt.UTC().Format(time.RFC3339),
			LastReceiveTime: sess.LastReceiveTime.UTC().Format(time.RFC3339),
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].CreatedAt > views[j].CreatedAt })
	return views
}

func (h *Handler) handleV1ListSessions(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceViewer); err != nil {
		return err
	}
	page, pageSize := parsePageParams(c)
	items := h.buildSessionViews(wsID)
	paged, total := paginate(items, page, pageSize)
	return respondOK(c, listData{
		Items:    paged,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

func (h *Handler) handleV1CreateSession(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	if err := h.ensureWorkspaceServicesRunning(c.Request().Context(), wsID, nilLogger{}); err != nil {
		h.appendOperation(c.Request().Context(), h.currentPrincipal(c), oplog.LevelError, "session.create_failed", "session", "", wsID, "", "session create failed", err.Error(), nil)
		return respondError(c, http.StatusInternalServerError, "MCP_DEPLOY_FAILED", err.Error(), nil)
	}
	sess, err := h.services.CreateProxySession(nilLogger{}, workspaces.NameArg{Workspace: wsID})
	if err != nil {
		h.appendOperation(c.Request().Context(), h.currentPrincipal(c), oplog.LevelError, "session.create_failed", "session", "", wsID, "", "session create failed", err.Error(), nil)
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.state.appendActivity(activityItem{
		Type:          "session.created",
		WorkspaceID:   wsID,
		WorkspaceName: h.workspaceName(wsID),
		SessionID:     sess.GetId(),
		Message:       "session created",
	})
	h.appendAudit(c, "session.create", "session", sess.GetId(), wsID, nil)
	return respondCreated(c, sessionView{
		ID:              sess.GetId(),
		WorkspaceID:     wsID,
		Status:          "active",
		IsReady:         sess.IsToolsListReady(),
		ToolsCount:      len(sess.GetAllTools()),
		BoundMCPNames:   h.listServiceNames(wsID),
		CreatedAt:       sess.CreatedAt.UTC().Format(time.RFC3339),
		LastReceiveTime: sess.LastReceiveTime.UTC().Format(time.RFC3339),
	})
}

func (h *Handler) handleV1DeleteSession(c echo.Context) error {
	if err := h.requireWorkspaceRole(c, c.Param("ws"), identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	h.services.CloseProxySession(nilLogger{}, workspaces.NameArg{Workspace: c.Param("ws"), Session: c.Param("id")})
	h.appendAudit(c, "session.delete", "session", c.Param("id"), c.Param("ws"), nil)
	return respondOK(c, map[string]string{"id": c.Param("id")})
}

func (h *Handler) handleV1GetSession(c echo.Context) error {
	sessionID := c.Param("id")
	for wsID := range h.getWorkspaceMap() {
		sess, ok := h.services.GetProxySession(nilLogger{}, workspaces.NameArg{Workspace: wsID, Session: sessionID})
		if !ok {
			continue
		}
		if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceViewer); err != nil {
			return err
		}
		return respondOK(c, map[string]interface{}{
			"id":                sess.GetId(),
			"workspace_id":      wsID,
			"status":            "active",
			"is_ready":          sess.IsToolsListReady(),
			"tools_count":       len(sess.GetAllTools()),
			"bound_mcp_names":   h.listServiceNames(wsID),
			"created_at":        sess.CreatedAt.UTC().Format(time.RFC3339),
			"last_receive_time": sess.LastReceiveTime.UTC().Format(time.RFC3339),
			"recent_messages":   []interface{}{},
		})
	}
	return respondError(c, http.StatusNotFound, "NOT_FOUND", "session not found", nil)
}

func (h *Handler) handleV1Installed(c echo.Context) error {
	principal := h.currentPrincipal(c)
	installed, err := h.listAccountInstalledPackages(c.Request().Context(), principal.AccountID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	items := make([]map[string]interface{}, 0, len(installed))
	for _, item := range installed {
		latestVersion := item.Version
		if pkg, ok := h.market.getPackage(item.PackageID); ok && pkg.Version != "" {
			latestVersion = pkg.Version
		}
		items = append(items, installedPackageView(item, valueOrDefault(latestVersion, item.Version)))
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i]["updated_at"].(string) > items[j]["updated_at"].(string)
	})
	return respondOK(c, listData{Items: items, Total: len(items), Page: 1, PageSize: len(items)})
}

func (h *Handler) handleV1DeleteInstalled(c echo.Context) error {
	principal := h.currentPrincipal(c)
	if err := h.deleteAccountInstalledPackage(c.Request().Context(), principal.AccountID, c.Param("id")); err != nil {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "installed package not found", nil)
	}
	h.appendAudit(c, "installed.delete", "installed_package", c.Param("id"), "", nil)
	return respondOK(c, map[string]string{"id": c.Param("id")})
}

func (h *Handler) handleV1UpdateInstalled(c echo.Context) error {
	principal := h.currentPrincipal(c)
	var req struct {
		DisplayName *string           `json:"display_name"`
		Args        []string          `json:"args"`
		Env         map[string]string `json:"env"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	item, err := h.getAccountInstalledPackage(c.Request().Context(), principal.AccountID, c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "installed package not found", nil)
	}
	if req.DisplayName != nil {
		item.DisplayName = strings.TrimSpace(*req.DisplayName)
	}
	cfg := serviceConfigFromMap(item.ConfigSnapshot, "")
	if req.Args != nil {
		cfg.Args = append([]string(nil), req.Args...)
	}
	if req.Env != nil {
		cfg.Env = copyStringMap(req.Env)
	} else if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	item.ConfigSnapshot = serviceConfigToMap(cfg)
	copyInstalledAuthState(item.ConfigSnapshot, installedPackageAuthState(item))
	item.UpdatedAt = time.Now().UTC()
	updated, err := h.upsertAccountInstalledPackage(c.Request().Context(), *item)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.appendAudit(c, "installed.update", "installed_package", updated.ID, "", map[string]interface{}{
		"package_id": updated.PackageID,
	})
	return respondOK(c, installedPackageView(updated, updated.Version))
}

func (h *Handler) handleV1CompleteInstalledOAuth(c echo.Context) error {
	principal := h.currentPrincipal(c)
	item, err := h.getAccountInstalledPackage(c.Request().Context(), principal.AccountID, c.Param("id"))
	if err != nil {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "installed package not found", nil)
	}
	auth := installedPackageAuthState(item)
	if auth == nil {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "installed package does not require OAuth", nil)
	}
	auth.Status = "authorized"
	cfg := serviceConfigFromMap(item.ConfigSnapshot, "")
	item.ConfigSnapshot = serviceConfigToMap(cfg)
	copyInstalledAuthState(item.ConfigSnapshot, auth)
	item.UpdatedAt = time.Now().UTC()
	updated, err := h.upsertAccountInstalledPackage(c.Request().Context(), *item)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.appendAudit(c, "installed.oauth_complete", "installed_package", updated.ID, "", map[string]interface{}{
		"package_id": updated.PackageID,
	})
	return respondOK(c, installedPackageView(updated, updated.Version))
}

func (h *Handler) handleV1MarketSources(c echo.Context) error {
	items := h.market.listSources()
	return respondOK(c, listData{
		Items:    items,
		Total:    len(items),
		Page:     1,
		PageSize: len(items),
	})
}

func (h *Handler) handleV1SyncMarketSource(c echo.Context) error {
	sourceID := c.Param("id")
	ctx, cancel := context.WithTimeout(c.Request().Context(), 45*time.Second)
	defer cancel()
	job, err := h.market.syncSource(ctx, sourceID)
	if err != nil && job == nil {
		return respondError(c, http.StatusUnprocessableEntity, "MARKET_SOURCE_UNREACHABLE", err.Error(), nil)
	}
	if err != nil {
		return respondError(c, http.StatusBadGateway, "MARKET_SOURCE_UNREACHABLE", err.Error(), job)
	}
	return respondOK(c, job)
}

func (h *Handler) handleV1MarketPackages(c echo.Context) error {
	page, pageSize := parsePageParams(c)
	q := strings.TrimSpace(c.QueryParam("q"))
	category := strings.TrimSpace(c.QueryParam("category"))
	sourceFilter := strings.TrimSpace(c.QueryParam("source"))
	installability := strings.TrimSpace(c.QueryParam("installability"))
	verifiedOnly := c.QueryParam("verified_only") == "true"
	items := h.market.listPackages(q, sourceFilter, category, installability, verifiedOnly)
	paged, total := paginate(items, page, pageSize)
	return respondOK(c, listData{Items: paged, Total: total, Page: page, PageSize: pageSize})
}

func (h *Handler) handleV1MarketPackageDetail(c echo.Context) error {
	pkg, ok := h.market.getPackage(c.Param("id"))
	if !ok {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "package not found", nil)
	}
	return respondOK(c, marketPackageDetailResponse(*pkg))
}

func (h *Handler) handleV1CreateMarketPackage(c echo.Context) error {
	if !h.currentPrincipal(c).IsSystemAdmin {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "market management requires system admin", nil)
	}
	var req marketPackageRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	pkg, err := marketPackageFromRequest(req)
	if err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), nil)
	}
	created := h.market.createLocalPackage(pkg)
	return respondCreated(c, marketPackageDetailResponse(created))
}

func (h *Handler) handleV1UpdateMarketPackage(c echo.Context) error {
	if !h.currentPrincipal(c).IsSystemAdmin {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "market management requires system admin", nil)
	}
	var req marketPackageRequest
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	pkg, err := marketPackageFromRequest(req)
	if err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error(), nil)
	}
	updated, ok := h.market.updateLocalPackage(c.Param("id"), pkg)
	if !ok {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "local market package not found", nil)
	}
	return respondOK(c, marketPackageDetailResponse(updated))
}

func (h *Handler) handleV1DeleteMarketPackage(c echo.Context) error {
	if !h.currentPrincipal(c).IsSystemAdmin {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "market management requires system admin", nil)
	}
	if !h.market.deleteLocalPackage(c.Param("id")) {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "local market package not found", nil)
	}
	return respondOK(c, map[string]string{"id": c.Param("id")})
}

func (h *Handler) handleV1InstallMarketPackage(c echo.Context) error {
	var req struct {
		DisplayName        string            `json:"display_name"`
		InstallOptionIndex int               `json:"install_option_index"`
		Env                map[string]string `json:"env"`
		Args               []string          `json:"args"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	pkg, ok := h.market.getPackage(c.Param("id"))
	if !ok {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "package not found", nil)
	}
	cfg, err := marketPackageToServiceConfig(*pkg, req.InstallOptionIndex, "", req.Env)
	if err != nil {
		return respondError(c, http.StatusUnprocessableEntity, "PACKAGE_INSTALL_FAILED", err.Error(), nil)
	}
	if req.Args != nil {
		cfg.Args = append([]string(nil), req.Args...)
	}
	cfg.GatewayProtocol = downstreamGatewayProtocol(h.cfg.GatewayProtocol)
	configSnapshot := serviceConfigToMap(cfg)
	if auth := marketInstallOptionAuth(*pkg, req.InstallOptionIndex); auth != nil {
		copyInstalledAuthState(configSnapshot, &installedAuthState{
			Type:             auth.Type,
			AuthorizationURL: auth.AuthorizationURL,
			Instructions:     auth.Instructions,
			Status:           "pending",
		})
	}
	version := pkg.Version
	if version == "" && req.InstallOptionIndex >= 0 && req.InstallOptionIndex < len(pkg.InstallOptions) {
		version = pkg.InstallOptions[req.InstallOptionIndex].PackageName
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = valueOrDefault(pkg.Title, pkg.Name)
	}
	sourceID := sourceIDFromPackage(*pkg)
	installed := identity.InstalledPackage{
		ID:                 uuid.NewString(),
		AccountID:          h.currentPrincipal(c).AccountID,
		PackageID:          pkg.ID,
		PackageName:        valueOrDefault(pkg.Name, pkg.Title),
		DisplayName:        displayName,
		Version:            version,
		SourceID:           sourceID,
		InstallOptionIndex: req.InstallOptionIndex,
		ConfigSnapshot:     configSnapshot,
		PackageSnapshot:    marketPackageSnapshot(*pkg),
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	installed, err = h.upsertAccountInstalledPackage(c.Request().Context(), installed)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.state.appendActivity(activityItem{
		Type:    "market.installed",
		Message: fmt.Sprintf("installed %s to account", installed.PackageName),
	})
	h.appendAudit(c, "market.install", "installed_package", installed.ID, "", map[string]interface{}{
		"package_id": pkg.ID,
	})
	return respondCreated(c, installedPackageView(installed, version))
}

type marketPackageRequest struct {
	Name           string                `json:"name"`
	Title          string                `json:"title"`
	Description    string                `json:"description"`
	Author         string                `json:"author"`
	Version        string                `json:"version"`
	Tags           []string              `json:"tags"`
	Category       string                `json:"category"`
	Repository     string                `json:"repository"`
	Homepage       string                `json:"homepage"`
	License        string                `json:"license"`
	Verified       bool                  `json:"verified"`
	InstallOptions []MarketInstallOption `json:"install_options"`
	Tools          []MarketToolSpec      `json:"tools"`
}

func marketPackageFromRequest(req marketPackageRequest) (MarketPackage, error) {
	name := strings.TrimSpace(req.Name)
	title := strings.TrimSpace(req.Title)
	if name == "" && title == "" {
		return MarketPackage{}, fmt.Errorf("name or title is required")
	}
	if name == "" {
		name = slugify(title)
	}
	if title == "" {
		title = name
	}
	if strings.TrimSpace(req.Description) == "" {
		return MarketPackage{}, fmt.Errorf("description is required")
	}
	options := append([]MarketInstallOption(nil), req.InstallOptions...)
	for i := range options {
		options[i].Type = normalizeInstallType(options[i].Type)
		options[i].SourceID = localMarketSourceID
		if options[i].Confidence == "" {
			options[i].Confidence = "high"
		}
		if options[i].Env == nil {
			options[i].Env = map[string]string{}
		}
	}
	return MarketPackage{
		CanonicalName:  localMarketSourceID + "/" + name,
		Name:           name,
		Title:          title,
		Description:    strings.TrimSpace(req.Description),
		Author:         strings.TrimSpace(req.Author),
		Version:        strings.TrimSpace(req.Version),
		Tags:           req.Tags,
		Category:       strings.TrimSpace(req.Category),
		Repository:     strings.TrimSpace(req.Repository),
		Homepage:       strings.TrimSpace(req.Homepage),
		License:        strings.TrimSpace(req.License),
		Verified:       req.Verified,
		InstallOptions: options,
		Tools:          req.Tools,
	}, nil
}

func marketPackageDetailResponse(pkg MarketPackage) map[string]interface{} {
	readme, _ := pkg.RawMeta["readme"].(string)
	versions, _ := pkg.RawMeta["versions"].([]string)
	sourceID := ""
	if len(pkg.SourceRefs) > 0 {
		sourceID = pkg.SourceRefs[0].SourceID
	}
	install := map[string]interface{}{}
	if len(pkg.InstallOptions) > 0 {
		opt := pkg.InstallOptions[0]
		install = map[string]interface{}{
			"type":    opt.Type,
			"command": opt.Command,
			"args":    opt.Args,
			"env":     opt.Env,
			"url":     opt.URL,
			"auth":    opt.Auth,
		}
	}
	return map[string]interface{}{
		"id":              pkg.ID,
		"canonical_name":  pkg.CanonicalName,
		"name":            valueOrDefault(pkg.Title, pkg.Name),
		"title":           pkg.Title,
		"version":         pkg.Version,
		"description":     pkg.Description,
		"author":          pkg.Author,
		"tags":            pkg.Tags,
		"rating":          pkg.Rating,
		"downloads":       pkg.Downloads,
		"use_count":       pkg.UseCount,
		"verified":        pkg.Verified,
		"source_id":       sourceID,
		"category":        pkg.Category,
		"repository":      pkg.Repository,
		"homepage":        pkg.Homepage,
		"license":         pkg.License,
		"installability":  pkg.Installability,
		"install_options": pkg.InstallOptions,
		"install":         install,
		"tools":           pkg.Tools,
		"env_schema":      pkg.EnvSchema,
		"source_refs":     pkg.SourceRefs,
		"readme":          readme,
		"versions":        versions,
		"raw_meta":        pkg.RawMeta,
	}
}

func installedPackageView(item identity.InstalledPackage, latestVersion string) map[string]interface{} {
	auth := installedPackageAuthState(&item)
	return map[string]interface{}{
		"id":                   item.ID,
		"account_id":           item.AccountID,
		"package_id":           item.PackageID,
		"package_name":         item.PackageName,
		"display_name":         valueOrDefault(item.DisplayName, item.PackageName),
		"installed_version":    valueOrDefault(item.Version, "unknown"),
		"latest_version":       valueOrDefault(latestVersion, valueOrDefault(item.Version, "unknown")),
		"source_id":            item.SourceID,
		"install_option_index": item.InstallOptionIndex,
		"config_snapshot":      item.ConfigSnapshot,
		"package_snapshot":     item.PackageSnapshot,
		"auth":                 auth,
		"status":               "installed",
		"installed_at":         item.CreatedAt.UTC().Format(time.RFC3339),
		"updated_at":           item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

type installedAuthState struct {
	Type             string `json:"type"`
	AuthorizationURL string `json:"authorization_url,omitempty"`
	Instructions     string `json:"instructions,omitempty"`
	Status           string `json:"status"`
}

func installedPackageAuthState(item *identity.InstalledPackage) *installedAuthState {
	if item == nil || item.ConfigSnapshot == nil {
		return nil
	}
	raw, ok := item.ConfigSnapshot["auth"]
	if !ok {
		raw, ok = item.ConfigSnapshot["_auth"]
	}
	if !ok {
		return nil
	}
	authMap, ok := raw.(map[string]interface{})
	if !ok {
		if primitiveMap, ok := raw.(primitive.M); ok {
			authMap = map[string]interface{}(primitiveMap)
		} else {
			return nil
		}
	}
	authType := strings.ToLower(strings.TrimSpace(asString(authMap["type"])))
	if authType != "oauth2" {
		return nil
	}
	status := strings.ToLower(strings.TrimSpace(asString(authMap["status"])))
	if status == "" {
		status = "pending"
	}
	return &installedAuthState{
		Type:             "oauth2",
		AuthorizationURL: asString(authMap["authorization_url"]),
		Instructions:     asString(authMap["instructions"]),
		Status:           status,
	}
}

func copyInstalledAuthState(snapshot map[string]interface{}, auth *installedAuthState) {
	if snapshot == nil || auth == nil {
		return
	}
	snapshot["auth"] = map[string]interface{}{
		"type":              auth.Type,
		"authorization_url": auth.AuthorizationURL,
		"instructions":      auth.Instructions,
		"status":            auth.Status,
	}
}

func (h *Handler) upsertAccountInstalledPackage(ctx context.Context, item identity.InstalledPackage) (identity.InstalledPackage, error) {
	if item.AccountID == "" {
		item.AccountID = "admin"
	}
	if h.auth != nil && h.auth.IsSaaS() {
		if err := h.auth.UpsertInstalledPackage(ctx, &item); err != nil {
			return item, err
		}
		return item, nil
	}
	return h.state.upsertInstalledPackage(item.AccountID, item), nil
}

func (h *Handler) getAccountInstalledPackage(ctx context.Context, accountID, id string) (*identity.InstalledPackage, error) {
	if accountID == "" {
		accountID = "admin"
	}
	if h.auth != nil && h.auth.IsSaaS() {
		item, err := h.auth.GetInstalledPackage(ctx, accountID, id)
		if err != nil || item == nil {
			return nil, fmt.Errorf("installed package not found")
		}
		return item, nil
	}
	if item, ok := h.state.getInstalledPackage(accountID, id); ok {
		return item, nil
	}
	return nil, fmt.Errorf("installed package not found")
}

func (h *Handler) listAccountInstalledPackages(ctx context.Context, accountID string) ([]identity.InstalledPackage, error) {
	if accountID == "" {
		accountID = "admin"
	}
	if h.auth != nil && h.auth.IsSaaS() {
		return h.auth.ListInstalledPackages(ctx, accountID)
	}
	return h.state.listInstalledPackages(accountID), nil
}

func (h *Handler) deleteAccountInstalledPackage(ctx context.Context, accountID, id string) error {
	if accountID == "" {
		accountID = "admin"
	}
	if h.auth != nil && h.auth.IsSaaS() {
		return h.auth.DeleteInstalledPackage(ctx, accountID, id)
	}
	if h.state.deleteInstalledPackage(accountID, id) {
		return nil
	}
	return fmt.Errorf("installed package not found")
}

func serviceConfigToMap(cfg config.MCPServerConfig) map[string]interface{} {
	return map[string]interface{}{
		"url":              cfg.URL,
		"command":          cfg.Command,
		"args":             append([]string(nil), cfg.Args...),
		"env":              copyStringMap(cfg.Env),
		"gateway_protocol": cfg.GatewayProtocol,
	}
}

func serviceConfigToMapWithDesiredStatus(cfg config.MCPServerConfig, desiredStatus string) map[string]interface{} {
	out := serviceConfigToMap(cfg)
	out[serviceDesiredStatusKey] = desiredStatus
	return out
}

func serviceConfigFromMap(raw map[string]interface{}, workspaceID string) config.MCPServerConfig {
	cfg := config.MCPServerConfig{
		Workspace: workspaceID,
		Args:      []string{},
		Env:       map[string]string{},
	}
	if raw == nil {
		return cfg
	}
	cfg.URL = asString(raw["url"])
	cfg.Command = asString(raw["command"])
	cfg.Args = asStringSlice(raw["args"])
	cfg.Env = asStringMap(raw["env"])
	cfg.GatewayProtocol = asString(raw["gateway_protocol"])
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	return cfg
}

func serviceOAuthStatus(cfg config.MCPServerConfig) string {
	if cfg.URL == "" {
		return ""
	}
	if cfg.Env != nil && strings.TrimSpace(cfg.Env[remoteOAuthAccessTokenEnv]) != "" {
		return "authorized"
	}
	return ""
}

func serviceDesiredStatus(raw map[string]interface{}) string {
	if raw == nil {
		return serviceDesiredStatusRunning
	}
	status := strings.ToLower(strings.TrimSpace(asString(raw[serviceDesiredStatusKey])))
	if status == "" {
		return serviceDesiredStatusRunning
	}
	return status
}

func serviceShouldAutoStart(raw map[string]interface{}) bool {
	return serviceDesiredStatus(raw) != serviceDesiredStatusStopped
}

func (h *Handler) markStoredServiceDesiredStatus(ctx context.Context, workspaceID, name, status string) {
	if h.auth == nil || !h.auth.IsSaaS() {
		return
	}
	dbServer, err := h.auth.GetMCPServer(ctx, workspaceID, name)
	if err != nil || dbServer == nil {
		return
	}
	if dbServer.Config == nil {
		dbServer.Config = map[string]interface{}{}
	}
	dbServer.Config[serviceDesiredStatusKey] = status
	dbServer.UpdatedAt = time.Now().UTC()
	_ = h.auth.CreateMCPServer(ctx, dbServer)
}

func (h *Handler) ensureWorkspaceServicesRunning(ctx context.Context, workspaceID string, logger xlog.Logger) error {
	if h.auth == nil || !h.auth.IsSaaS() {
		return nil
	}
	dbServers, err := h.auth.ListMCPServers(ctx, workspaceID)
	if err != nil {
		return err
	}
	for _, dbServer := range dbServers {
		h.state.upsertService(workspaceID, serviceMeta{
			Name:          dbServer.Name,
			WorkspaceID:   dbServer.WorkspaceID,
			SourceType:    dbServer.SourceType,
			SourceRef:     dbServer.SourceRef,
			Version:       dbServer.Version,
			CreatedAt:     dbServer.CreatedAt,
			InstalledFrom: "database",
		})
		if !serviceShouldAutoStart(dbServer.Config) {
			continue
		}
		cfg := serviceConfigFromMap(dbServer.Config, workspaceID)
		if cfg.Workspace == "" {
			cfg.Workspace = workspaceID
		}
		if _, err := h.DeployServer(dbServer.Name, cfg); err != nil {
			return fmt.Errorf("deploy %s/%s: %w", workspaceID, dbServer.Name, err)
		}
	}
	return nil
}

func marketPackageSnapshot(pkg MarketPackage) map[string]interface{} {
	return map[string]interface{}{
		"id":             pkg.ID,
		"canonical_name": pkg.CanonicalName,
		"name":           pkg.Name,
		"title":          pkg.Title,
		"description":    pkg.Description,
		"author":         pkg.Author,
		"version":        pkg.Version,
		"source_id":      sourceIDFromPackage(pkg),
		"category":       pkg.Category,
		"repository":     pkg.Repository,
		"homepage":       pkg.Homepage,
		"license":        pkg.License,
		"verified":       pkg.Verified,
		"tags":           append([]string(nil), pkg.Tags...),
		"tools":          pkg.Tools,
	}
}

func sourceIDFromPackage(pkg MarketPackage) string {
	if len(pkg.SourceRefs) > 0 {
		return pkg.SourceRefs[0].SourceID
	}
	if pkg.hasSource(localMarketSourceID) {
		return localMarketSourceID
	}
	return ""
}

func isServiceNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "not exist")
}

func (h *Handler) handleV1WorkspaceLogs(c echo.Context) error {
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceViewer); err != nil {
		return err
	}
	tail, _ := strconv.Atoi(c.QueryParam("tail"))
	if tail <= 0 {
		tail = 200
	}
	services := h.listServiceNames(wsID)
	merged := make([]map[string]interface{}, 0)
	if h.oplog != nil {
		operationLogs, err := h.oplog.List(c.Request().Context(), oplog.Query{WorkspaceID: wsID, Limit: tail})
		if err != nil {
			return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
		}
		for _, event := range operationLogs {
			detail := event.Detail
			if detail == nil {
				detail = map[string]interface{}{}
			}
			row := map[string]interface{}{
				"timestamp":     event.Timestamp.Format(time.RFC3339Nano),
				"level":         string(event.Level),
				"message":       event.Message,
				"source":        event.Source,
				"kind":          logKind(event.Action),
				"summary":       logSummary(event, detail),
				"action":        event.Action,
				"resource_type": event.ResourceType,
				"resource_id":   event.ResourceID,
				"workspace_id":  event.WorkspaceID,
				"session_id":    event.SessionID,
				"actor_id":      event.ActorID,
				"method":        detail["method"],
				"request_id":    detail["request_id"],
				"tool_name":     detail["tool_name"],
				"mcp_name":      detail["mcp_name"],
				"transport":     detail["transport"],
				"duration_ms":   detail["duration_ms"],
				"metadata": map[string]interface{}{
					"id":            event.ID,
					"action":        event.Action,
					"resource_type": event.ResourceType,
					"resource_id":   event.ResourceID,
					"workspace_id":  event.WorkspaceID,
					"session_id":    event.SessionID,
					"actor_id":      event.ActorID,
					"detail":        detail,
				},
			}
			if event.Error != "" {
				row["metadata"].(map[string]interface{})["error"] = event.Error
			}
			merged = append(merged, row)
		}
	}
	for _, name := range services {
		lines, err := h.readServiceLogs(wsID, name, tail)
		if err != nil {
			continue
		}
		for _, line := range lines {
			line["source"] = name
			merged = append(merged, line)
		}
	}
	sort.SliceStable(merged, func(i, j int) bool {
		return logRowTime(merged[i]).After(logRowTime(merged[j]))
	})
	if len(merged) > tail {
		merged = merged[:tail]
	}
	return respondOK(c, map[string]interface{}{
		"workspace_id": wsID,
		"total_lines":  len(merged),
		"logs":         merged,
	})
}

func logKind(action string) string {
	switch {
	case strings.HasPrefix(action, "tool."):
		return "tool"
	case strings.HasPrefix(action, "session."):
		return "session"
	case strings.HasPrefix(action, "service."):
		return "mcp"
	case strings.HasPrefix(action, "workspace."):
		return "workspace"
	case strings.HasPrefix(action, "api_key."):
		return "api_key"
	case strings.HasPrefix(action, "market.") || strings.HasPrefix(action, "installed."):
		return "market"
	default:
		return "operation"
	}
}

func logSummary(event oplog.Event, detail map[string]interface{}) string {
	if tool, _ := detail["tool_name"].(string); tool != "" {
		if mcpName, _ := detail["mcp_name"].(string); mcpName != "" {
			return fmt.Sprintf("%s -> %s", tool, mcpName)
		}
		return tool
	}
	if method, _ := detail["method"].(string); method != "" {
		return method
	}
	if event.ResourceID != "" {
		return event.ResourceID
	}
	return event.Action
}

func logRowTime(row map[string]interface{}) time.Time {
	raw, _ := row["timestamp"].(string)
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}
	}
	return ts
}

func (h *Handler) handleV1SystemConfig(c echo.Context) error {
	if h.authMode() == "saas" && !h.currentPrincipal(c).IsSystemAdmin {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "system config requires system admin", nil)
	}
	return respondOK(c, h.systemConfigView())
}

func (h *Handler) handleV1UpdateSystemConfig(c echo.Context) error {
	if h.authMode() == "saas" && !h.currentPrincipal(c).IsSystemAdmin {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "system config requires system admin", nil)
	}
	var req struct {
		Bind                       *string `json:"bind"`
		GatewayProtocol            *string `json:"gateway_protocol"`
		SessionGCIntervalSeconds   *int    `json:"session_gc_interval_seconds"`
		ProxySessionTimeoutSeconds *int    `json:"proxy_session_timeout_seconds"`
		McpRetryCount              *int    `json:"mcp_retry_count"`
		Auth                       *struct {
			Enabled                  *bool    `json:"enabled"`
			Mode                     *string  `json:"mode"`
			AllowRegister            *bool    `json:"allow_register"`
			AuthorizationServers     []string `json:"authorization_servers"`
			TokenIssuer              *string  `json:"token_issuer"`
			TokenJWKSURI             *string  `json:"token_jwks_uri"`
			TokenIntrospectionURL    *string  `json:"token_introspection_url"`
			TokenIntrospectionID     *string  `json:"token_introspection_id"`
			TokenIntrospectionSecret *string  `json:"token_introspection_secret"`
			TokenAudience            *string  `json:"token_audience"`
			RequiredScopes           []string `json:"required_scopes"`
			ScopesSupported          []string `json:"scopes_supported"`
		} `json:"auth"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	if req.Bind != nil {
		h.cfg.Bind = *req.Bind
	}
	if req.GatewayProtocol != nil && isGatewayExposureProtocol(*req.GatewayProtocol) {
		h.cfg.GatewayProtocol = strings.ToLower(strings.TrimSpace(*req.GatewayProtocol))
	}
	if req.SessionGCIntervalSeconds != nil {
		h.cfg.SessionGCInterval = time.Duration(*req.SessionGCIntervalSeconds) * time.Second
	}
	if req.ProxySessionTimeoutSeconds != nil {
		h.cfg.ProxySessionTimeout = time.Duration(*req.ProxySessionTimeoutSeconds) * time.Second
	}
	if req.McpRetryCount != nil {
		h.cfg.McpServiceMgrConfig.McpServiceRetryCount = *req.McpRetryCount
	}
	if req.Auth != nil && req.Auth.Enabled != nil {
		h.cfg.Auth.Enabled = *req.Auth.Enabled
	}
	if req.Auth != nil {
		if req.Auth.Mode != nil && (*req.Auth.Mode == "single-key" || *req.Auth.Mode == "saas") {
			h.cfg.Auth.Mode = *req.Auth.Mode
		}
		if req.Auth.AllowRegister != nil {
			h.cfg.Auth.AllowRegister = *req.Auth.AllowRegister
		}
		if req.Auth.AuthorizationServers != nil {
			h.cfg.Auth.AuthorizationServers = cleanStringList(req.Auth.AuthorizationServers)
		}
		if req.Auth.TokenIssuer != nil {
			h.cfg.Auth.TokenIssuer = strings.TrimSpace(*req.Auth.TokenIssuer)
		}
		if req.Auth.TokenJWKSURI != nil {
			h.cfg.Auth.TokenJWKSURI = strings.TrimSpace(*req.Auth.TokenJWKSURI)
		}
		if req.Auth.TokenIntrospectionURL != nil {
			h.cfg.Auth.TokenIntrospectionURL = strings.TrimSpace(*req.Auth.TokenIntrospectionURL)
		}
		if req.Auth.TokenIntrospectionID != nil {
			h.cfg.Auth.TokenIntrospectionID = strings.TrimSpace(*req.Auth.TokenIntrospectionID)
		}
		if req.Auth.TokenIntrospectionSecret != nil {
			h.cfg.Auth.TokenIntrospectionSecret = *req.Auth.TokenIntrospectionSecret
		}
		if req.Auth.TokenAudience != nil {
			h.cfg.Auth.TokenAudience = strings.TrimSpace(*req.Auth.TokenAudience)
		}
		if req.Auth.RequiredScopes != nil {
			h.cfg.Auth.RequiredScopes = cleanStringList(req.Auth.RequiredScopes)
		}
		if req.Auth.ScopesSupported != nil {
			h.cfg.Auth.ScopesSupported = cleanStringList(req.Auth.ScopesSupported)
		}
	}
	if err := h.cfg.SaveConfig(); err != nil && h.cfg.CfgPath() != "" {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return respondOK(c, h.systemConfigView())
}

func (h *Handler) handleV1GetSystemAPIKey(c echo.Context) error {
	if h.authMode() != "single-key" {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "system api key is unavailable in saas mode", nil)
	}
	return respondOK(c, map[string]interface{}{
		"api_key":    h.cfg.GetAuthConfig().GetApiKey(),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) handleV1RotateSystemAPIKey(c echo.Context) error {
	if h.authMode() != "single-key" {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "system api key is unavailable in saas mode", nil)
	}
	key, err := generateAPIKey()
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.cfg.GetAuthConfig().ApiKey = key
	if err := h.cfg.SaveConfig(); err != nil && h.cfg.CfgPath() != "" {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return respondOK(c, map[string]interface{}{
		"api_key":    key,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *Handler) handleV1ListAPIKeys(c echo.Context) error {
	if h.authMode() == "single-key" {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "api keys list is unavailable in single-key mode", nil)
	}
	items, err := h.auth.ListAPIKeys(c.Request().Context(), h.currentPrincipal(c))
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	return respondOK(c, listData{Items: items, Total: len(items), Page: 1, PageSize: len(items)})
}

func (h *Handler) handleV1CreateAPIKey(c echo.Context) error {
	if h.authMode() == "single-key" {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "api keys list is unavailable in single-key mode", nil)
	}
	var req struct {
		Name        string     `json:"name"`
		WorkspaceID string     `json:"workspace_id"`
		Scope       []string   `json:"scope"`
		ExpiresAt   *time.Time `json:"expires_at"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	if strings.TrimSpace(req.Name) == "" {
		return respondError(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required", nil)
	}
	if req.WorkspaceID != "" {
		if err := h.requireWorkspaceRole(c, req.WorkspaceID, identity.RoleWorkspaceAdmin); err != nil {
			return err
		}
	}
	item, err := h.auth.CreateAPIKey(c.Request().Context(), h.currentPrincipal(c), req.Name, req.WorkspaceID, req.Scope, req.ExpiresAt)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.appendAudit(c, "api_key.create", "api_key", asString(item["id"]), req.WorkspaceID, map[string]interface{}{"name": req.Name})
	return respondCreated(c, item)
}

func (h *Handler) handleV1RevokeAPIKey(c echo.Context) error {
	if h.authMode() == "single-key" {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "api keys list is unavailable in single-key mode", nil)
	}
	if err := h.auth.RevokeAPIKey(c.Request().Context(), h.currentPrincipal(c), c.Param("id")); err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.appendAudit(c, "api_key.revoke", "api_key", c.Param("id"), "", nil)
	return respondOK(c, map[string]string{"id": c.Param("id"), "status": "revoked"})
}

func (h *Handler) handleV1ListWorkspaceMembers(c echo.Context) error {
	if h.authMode() == "single-key" {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "workspace members are unavailable in single-key mode", nil)
	}
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceViewer); err != nil {
		return err
	}
	items, err := h.auth.ListWorkspaceMembers(c.Request().Context(), wsID)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]interface{}{
			"id":           item.ID,
			"workspace_id": item.WorkspaceID,
			"account_id":   item.AccountID,
			"role":         item.Role,
			"created_at":   item.CreatedAt.Format(time.RFC3339),
		})
	}
	return respondOK(c, listData{Items: out, Total: len(out), Page: 1, PageSize: len(out)})
}

func (h *Handler) handleV1AddWorkspaceMember(c echo.Context) error {
	if h.authMode() == "single-key" {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "workspace members are unavailable in single-key mode", nil)
	}
	wsID := c.Param("ws")
	if err := h.requireWorkspaceRole(c, wsID, identity.RoleWorkspaceOwner); err != nil {
		return err
	}
	var req struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	member, err := h.auth.AddWorkspaceMember(c.Request().Context(), wsID, req.Email, identity.NormalizeWorkspaceRole(req.Role))
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.appendAudit(c, "workspace_member.add", "workspace_member", member.ID, wsID, map[string]interface{}{"email": req.Email, "role": member.Role})
	return respondCreated(c, map[string]interface{}{
		"id":           member.ID,
		"workspace_id": member.WorkspaceID,
		"account_id":   member.AccountID,
		"role":         member.Role,
		"created_at":   member.CreatedAt.Format(time.RFC3339),
	})
}

func (h *Handler) handleV1ListAuditLogs(c echo.Context) error {
	if h.authMode() == "single-key" {
		return respondOK(c, listData{Items: []map[string]interface{}{}, Total: 0, Page: 1, PageSize: 0})
	}
	if !h.currentPrincipal(c).IsSystemAdmin {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "audit logs require system admin", nil)
	}
	items, err := h.auth.ListAuditLogs(c.Request().Context(), c.QueryParam("workspace_id"), 50)
	if err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		out = append(out, map[string]interface{}{
			"id":               item.ID,
			"actor_account_id": item.ActorAccountID,
			"action":           item.Action,
			"resource_type":    item.ResourceType,
			"resource_id":      item.ResourceID,
			"workspace_id":     item.WorkspaceID,
			"detail":           item.Detail,
			"created_at":       item.CreatedAt.Format(time.RFC3339),
		})
	}
	return respondOK(c, listData{Items: out, Total: len(out), Page: 1, PageSize: len(out)})
}

func (h *Handler) parseServiceRequest(ctx context.Context, workspaceID string, raw map[string]interface{}) (string, config.MCPServerConfig, serviceMeta, error) {
	name := strings.TrimSpace(asString(raw["name"]))
	if name == "" {
		return "", config.MCPServerConfig{}, serviceMeta{}, fmt.Errorf("service name is required")
	}
	cfg := config.MCPServerConfig{
		Workspace: workspaceID,
		Args:      []string{},
		Env:       map[string]string{},
	}
	meta := serviceMeta{Name: name, WorkspaceID: workspaceID, CreatedAt: time.Now().UTC()}

	if pkgID := asString(raw["market_package_id"]); pkgID != "" {
		pkg, ok := h.market.getPackage(pkgID)
		if !ok {
			return "", config.MCPServerConfig{}, serviceMeta{}, fmt.Errorf("market package not found")
		}
		meta.SourceType = "market"
		meta.SourceRef = pkgID
		meta.Version = asString(raw["version"])
		optionIndex, _ := strconv.Atoi(asString(raw["install_option_index"]))
		cfg, err := marketPackageToServiceConfig(*pkg, optionIndex, workspaceID, asStringMap(raw["env"]))
		if err != nil {
			return "", config.MCPServerConfig{}, serviceMeta{}, err
		}
		if meta.Version == "" {
			meta.Version = pkg.Version
		}
		return name, cfg, meta, nil
	}

	if url := asString(raw["url"]); url != "" {
		meta.SourceType = "url"
		cfg.URL = url
		cfg.GatewayProtocol = asGatewayProtocol(raw["gateway_protocol"])
		if err := h.applyRequestOAuth(ctx, raw["auth"], &cfg); err != nil {
			return "", config.MCPServerConfig{}, serviceMeta{}, err
		}
		return name, cfg, meta, nil
	}

	command := asString(raw["command"])
	if command == "" {
		return "", config.MCPServerConfig{}, serviceMeta{}, fmt.Errorf("command or url is required")
	}
	meta.SourceType = "command"
	cfg.Command = command
	cfg.Args = asStringSlice(raw["args"])
	cfg.Env = asStringMap(raw["env"])
	cfg.GatewayProtocol = asGatewayProtocol(raw["gateway_protocol"])
	return name, cfg, meta, nil
}

func asGatewayProtocol(v interface{}) string {
	protocol := strings.ToLower(strings.TrimSpace(asString(v)))
	if isServiceGatewayProtocol(protocol) {
		return protocol
	}
	return ""
}

func isGatewayExposureProtocol(protocol string) bool {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "all", "sse", "streamhttp":
		return true
	default:
		return false
	}
}

func isServiceGatewayProtocol(protocol string) bool {
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "sse", "streamhttp":
		return true
	default:
		return false
	}
}

func downstreamGatewayProtocol(protocol string) string {
	if strings.ToLower(strings.TrimSpace(protocol)) == "streamhttp" {
		return "streamhttp"
	}
	return "sse"
}

func (h *Handler) applyRequestOAuth(ctx context.Context, v interface{}, cfg *config.MCPServerConfig) error {
	if v == nil {
		return nil
	}
	authMap, ok := v.(map[string]interface{})
	if !ok {
		if primitiveMap, ok := v.(primitive.M); ok {
			authMap = map[string]interface{}(primitiveMap)
		} else {
			return nil
		}
	}
	authType := strings.ToLower(strings.TrimSpace(asString(authMap["type"])))
	if authType == "" {
		return nil
	}
	if authType != "oauth2" {
		return fmt.Errorf("unsupported auth type %q", authType)
	}
	state := strings.TrimSpace(asString(authMap["state"]))
	if state != "" {
		flow, ok := h.oauth.get(state)
		if !ok || flow.Status != "authorized" || flow.AccessToken == "" {
			return fmt.Errorf("OAuth authorization is required before deploying this MCP")
		}
		if cfg.Env == nil {
			cfg.Env = map[string]string{}
		}
		cfg.Env[remoteOAuthAccessTokenEnv] = flow.AccessToken
		return nil
	}
	if strings.ToLower(strings.TrimSpace(asString(authMap["status"]))) != "authorized" {
		return fmt.Errorf("OAuth authorization is required before deploying this MCP")
	}
	_ = ctx
	return nil
}

func asString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	default:
		return fmt.Sprintf("%v", v)
	}
}

func asStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch items := v.(type) {
	case []string:
		return items
	case []interface{}:
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, asString(item))
		}
		return out
	case primitive.A:
		out := make([]string, 0, len(items))
		for _, item := range items {
			out = append(out, asString(item))
		}
		return out
	default:
		return nil
	}
}

func asStringMap(v interface{}) map[string]string {
	if v == nil {
		return map[string]string{}
	}
	switch m := v.(type) {
	case map[string]string:
		return m
	case map[string]interface{}:
		out := make(map[string]string, len(m))
		for k, val := range m {
			out[k] = asString(val)
		}
		return out
	case primitive.M:
		out := make(map[string]string, len(m))
		for k, val := range m {
			out[k] = asString(val)
		}
		return out
	default:
		return map[string]string{}
	}
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.ReplaceAll(s, " ", "-")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}

func maskEnv(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		key := strings.ToLower(k)
		if strings.Contains(key, "key") || strings.Contains(key, "token") || strings.Contains(key, "secret") || strings.Contains(key, "password") {
			out[k] = "******"
			continue
		}
		out[k] = v
	}
	return out
}

func normalizeServiceStatus(status runtime.CmdStatus) string {
	switch strings.ToLower(string(status)) {
	case "running":
		return "running"
	case "starting":
		return "starting"
	case "stopped":
		return "stopped"
	case "failed":
		return "failed"
	default:
		return strings.ToLower(string(status))
	}
}

func (h *Handler) findServiceView(wsID, name string) serviceView {
	for _, item := range h.buildServiceViews(wsID) {
		if item.Name == name {
			return item
		}
	}
	return serviceView{}
}

func (h *Handler) workspaceViewFromMeta(meta *workspaceMeta) workspaceView {
	if meta == nil {
		return workspaceView{}
	}
	countServices := len(h.services.GetMcpServices(nilLogger{}, workspaces.NameArg{Workspace: meta.ID}))
	countSessions := len(h.services.GetWorkspaceSessions(nilLogger{}, workspaces.NameArg{Workspace: meta.ID}))
	return workspaceView{
		ID:             meta.ID,
		Name:           meta.Name,
		Description:    meta.Description,
		OwnerID:        "admin",
		Status:         h.workspaceStatus(meta.ID),
		McpCount:       countServices,
		SessionCount:   countSessions,
		CreatedAt:      meta.CreatedAt.UTC().Format(time.RFC3339),
		LastActivityAt: meta.LastActivityAt.UTC().Format(time.RFC3339),
	}
}

func (h *Handler) workspaceStatus(wsID string) string {
	if len(h.services.GetMcpServices(nilLogger{}, workspaces.NameArg{Workspace: wsID})) > 0 {
		return "running"
	}
	return "stopped"
}

func (h *Handler) workspaceName(wsID string) string {
	if meta, ok := h.state.getWorkspace(wsID); ok && meta.Name != "" {
		return meta.Name
	}
	return wsID
}

func (h *Handler) getWorkspaceMap() map[string]*workspaces.WorkSpace {
	if sm, ok := h.services.(*workspaces.ServiceManager); ok {
		return sm.GetWorkspaces()
	}
	return map[string]*workspaces.WorkSpace{}
}

func (h *Handler) seedStateFromRuntime() {
	for wsID, ws := range h.getWorkspaceMap() {
		meta := h.state.ensureWorkspace(wsID)
		if meta.Name == "" {
			meta.Name = wsID
		}
		for name, svc := range ws.GetMcpServices() {
			info := svc.Info()
			sourceType := "command"
			sourceRef := ""
			if info.Config.URL != "" {
				sourceType = "url"
			}
			h.state.upsertService(wsID, serviceMeta{
				Name:        name,
				WorkspaceID: wsID,
				SourceType:  sourceType,
				SourceRef:   sourceRef,
				CreatedAt:   info.DeployedAt.UTC(),
			})
		}
	}
}

func (h *Handler) serviceToolsCount(wsID, name string) int {
	return len(h.findServiceTools(wsID, name))
}

func (h *Handler) findServiceTools(wsID, name string) []map[string]interface{} {
	sessionsList := h.services.GetWorkspaceSessions(nilLogger{}, workspaces.NameArg{Workspace: wsID})
	items := make([]map[string]interface{}, 0)
	for _, sess := range sessionsList {
		for toolName, tool := range sess.GetMcpTools(name) {
			items = append(items, map[string]interface{}{
				"name":         toolName,
				"description":  tool.Description,
				"input_schema": tool.InputSchema,
			})
		}
		if len(items) > 0 {
			break
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return asString(items[i]["name"]) < asString(items[j]["name"])
	})
	return items
}

func (h *Handler) readServiceLogs(wsID, name string, tail int) ([]map[string]interface{}, error) {
	path := filepath.Join(h.cfg.WorkspacePath, fmt.Sprintf("%s.%s.log", wsID, name))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) > tail {
		lines = lines[len(lines)-tail:]
	}
	out := make([]map[string]interface{}, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		level := "info"
		switch {
		case strings.Contains(strings.ToLower(line), "error"):
			level = "error"
		case strings.Contains(strings.ToLower(line), "warn"):
			level = "warn"
		case strings.Contains(strings.ToLower(line), "debug"):
			level = "debug"
		}
		out = append(out, map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"level":     level,
			"message":   line,
		})
	}
	return out, nil
}

func (h *Handler) listServiceNames(wsID string) []string {
	names := make([]string, 0)
	for name := range h.services.GetMcpServices(nilLogger{}, workspaces.NameArg{Workspace: wsID}) {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (h *Handler) systemConfigView() map[string]interface{} {
	return map[string]interface{}{
		"bind":                          h.cfg.Bind,
		"gateway_protocol":              h.cfg.GatewayProtocol,
		"session_gc_interval_seconds":   int(h.cfg.SessionGCInterval.Seconds()),
		"proxy_session_timeout_seconds": int(h.cfg.ProxySessionTimeout.Seconds()),
		"mcp_retry_count":               h.cfg.McpServiceMgrConfig.GetMcpServiceRetryCount(),
		"auth": map[string]interface{}{
			"enabled":                        h.cfg.GetAuthConfig().Enabled,
			"mode":                           h.authMode(),
			"allow_register":                 h.cfg.GetAuthConfig().AllowRegister,
			"authorization_servers":          h.cfg.GetAuthConfig().AuthorizationServers,
			"token_issuer":                   h.cfg.GetAuthConfig().TokenIssuer,
			"token_jwks_uri":                 h.cfg.GetAuthConfig().TokenJWKSURI,
			"token_introspection_url":        h.cfg.GetAuthConfig().TokenIntrospectionURL,
			"token_introspection_id":         h.cfg.GetAuthConfig().TokenIntrospectionID,
			"token_introspection_secret_set": h.cfg.GetAuthConfig().TokenIntrospectionSecret != "",
			"token_audience":                 h.cfg.GetAuthConfig().TokenAudience,
			"required_scopes":                h.cfg.GetAuthConfig().RequiredScopes,
			"scopes_supported":               h.cfg.GetAuthConfig().ScopesSupported,
		},
	}
}

func cleanStringList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func generateAPIKey() (string, error) {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "mkg_" + hex.EncodeToString(buf), nil
}

func valueOrDefault(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

type nilLogger struct{}

func (nilLogger) Debug(args ...interface{})                 {}
func (nilLogger) Debugf(format string, args ...interface{}) {}
func (nilLogger) Info(args ...interface{})                  {}
func (nilLogger) Infof(format string, args ...interface{})  {}
func (nilLogger) Warn(args ...interface{})                  {}
func (nilLogger) Warnf(format string, args ...interface{})  {}
func (nilLogger) Error(args ...interface{})                 {}
func (nilLogger) Errorf(format string, args ...interface{}) {}
func (nilLogger) Fatal(args ...interface{})                 {}
func (nilLogger) Fatalf(format string, args ...interface{}) {}
func (nilLogger) With(key string, value interface{}) xlog.Logger {
	return nilLogger{}
}
func (nilLogger) WithFields(fields map[string]interface{}) xlog.Logger {
	return nilLogger{}
}
func (nilLogger) Name() string { return "nil" }

var _ xlog.Logger = nilLogger{}

var _ = json.RawMessage{}
var _ = mcp.Tool{}
var _ = sessions.Session{}
