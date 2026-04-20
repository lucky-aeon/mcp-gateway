package admin

import (
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
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/sessions"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
	"github.com/mark3labs/mcp-go/mcp"
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

func (h *Handler) registerV1Routes(e *echo.Echo) {
	publicV1 := e.Group("/api/v1")
	publicV1.GET("/meta", h.handleV1Meta)
	publicV1.POST("/auth/login", h.handleV1AuthLogin)
	publicV1.POST("/auth/register", h.handleV1AuthRegister)
	publicV1.POST("/auth/refresh", h.handleV1AuthRefresh)

	v1 := e.Group("/api/v1")
	v1.Use(h.v1AuthMiddleware)

	v1.GET("/auth/me", h.handleV1AuthMe)
	v1.POST("/auth/logout", h.handleV1AuthLogout)
	v1.GET("/stats/overview", h.handleV1StatsOverview)

	v1.GET("/workspaces", h.handleV1ListWorkspaces)
	v1.POST("/workspaces", h.handleV1CreateWorkspace)
	v1.GET("/workspaces/:ws", h.handleV1GetWorkspace)
	v1.PATCH("/workspaces/:ws", h.handleV1PatchWorkspace)
	v1.DELETE("/workspaces/:ws", h.handleV1DeleteWorkspace)

	v1.GET("/workspaces/:ws/services", h.handleV1ListServices)
	v1.POST("/workspaces/:ws/services", h.handleV1CreateService)
	v1.POST("/workspaces/:ws/services:batch", h.handleV1BatchCreateServices)
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
	v1.GET("/api-keys", h.handleV1ListAPIKeys)
	v1.POST("/api-keys", h.handleV1CreateAPIKey)
	v1.POST("/api-keys/:id/revoke", h.handleV1RevokeAPIKey)
	v1.GET("/workspaces/:ws/members", h.handleV1ListWorkspaceMembers)
	v1.POST("/workspaces/:ws/members", h.handleV1AddWorkspaceMember)
	v1.GET("/audit-logs", h.handleV1ListAuditLogs)
	v1.GET("/market/sources", h.handleV1MarketSources)
	v1.GET("/market/packages", h.handleV1MarketPackages)
	v1.GET("/market/packages/:id", h.handleV1MarketPackageDetail)
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
	if h.auth == nil {
		return
	}
	h.auth.AppendAuditLog(c.Request().Context(), h.currentPrincipal(c), action, resourceType, resourceID, workspaceID, detail)
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
				return []string{"password"}
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
	h.state.deleteWorkspace(wsID)
	// 删除数据库中的工作区成员关系和工作区记录
	if h.auth != nil {
		_ = h.auth.DeleteWorkspaceMembers(c.Request().Context(), wsID)
		_ = h.auth.DeleteWorkspace(c.Request().Context(), wsID)
	}
	h.appendAudit(c, "workspace.delete", "workspace", wsID, wsID, nil)
	return respondOK(c, map[string]string{"id": wsID})
}

type serviceView struct {
	Name        string            `json:"name"`
	WorkspaceID string            `json:"workspace_id"`
	SourceType  string            `json:"source_type"`
	SourceRef   string            `json:"source_ref"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	URL         string            `json:"url,omitempty"`
	Status      string            `json:"status"`
	Port        int               `json:"port,omitempty"`
	ToolsCount  int               `json:"tools_count"`
	LastError   string            `json:"last_error,omitempty"`
	RetryCount  int               `json:"retry_count"`
	CreatedAt   string            `json:"created_at"`
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
			Name:        name,
			WorkspaceID: workspaceID,
			SourceType:  sourceType,
			SourceRef:   sourceRef,
			Command:     info.Config.Command,
			Args:        info.Config.Args,
			Env:         maskEnv(info.Config.Env),
			URL:         info.Config.URL,
			Status:      normalizeServiceStatus(info.Status),
			Port:        info.Port,
			ToolsCount:  h.serviceToolsCount(workspaceID, name),
			LastError:   info.LastError,
			RetryCount:  info.RetryCount,
			CreatedAt:   createdAt.UTC().Format(time.RFC3339),
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

	name, cfg, meta, err := parseServiceRequest(wsID, raw)
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
		configMap := map[string]interface{}{
			"url":              cfg.URL,
			"command":          cfg.Command,
			"args":             cfg.Args,
			"env":              cfg.Env,
			"gateway_protocol": cfg.GatewayProtocol,
		}
		_ = h.auth.CreateMCPServer(c.Request().Context(), &identity.MCPServer{
			ID:          uuid.NewString(),
			Name:        name,
			WorkspaceID: wsID,
			SourceType:  meta.SourceType,
			SourceRef:   meta.SourceRef,
			Version:     meta.Version,
			Config:      configMap,
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
		_, cfg, meta, err := parseServiceRequest(wsID, body)
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
	_, cfg, meta, err := parseServiceRequest(wsID, raw)
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
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
	h.state.deleteService(wsID, name)
	// 从数据库删除
	if h.auth != nil {
		_ = h.auth.DeleteMCPServer(c.Request().Context(), wsID, name)
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
				if url, ok := dbServer.Config["url"].(string); ok {
					cfg.URL = url
				}
				if cmd, ok := dbServer.Config["command"].(string); ok {
					cfg.Command = cmd
				}
				if args, ok := dbServer.Config["args"].([]interface{}); ok {
					for _, arg := range args {
						if argStr, ok := arg.(string); ok {
							cfg.Args = append(cfg.Args, argStr)
						}
					}
				}
				if env, ok := dbServer.Config["env"].(map[string]interface{}); ok {
					cfg.Env = make(map[string]string)
					for k, v := range env {
						if vStr, ok := v.(string); ok {
							cfg.Env[k] = vStr
						}
					}
				}
				if gatewayProtocol, ok := dbServer.Config["gateway_protocol"].(string); ok {
					cfg.GatewayProtocol = gatewayProtocol
				}
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
			return respondError(c, http.StatusNotFound, "NOT_FOUND", "service not found", nil)
		}
	}

	if _, err := h.DeployServer(name, cfg); err != nil {
		return respondError(c, http.StatusInternalServerError, "MCP_DEPLOY_FAILED", err.Error(), nil)
	}

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

	return respondOK(c, map[string]string{"status": "running"})
}

func (h *Handler) handleV1StopService(c echo.Context) error {
	if err := h.requireWorkspaceRole(c, c.Param("ws"), identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	h.services.StopServer(nilLogger{}, workspaces.NameArg{Workspace: c.Param("ws"), Server: c.Param("name")})
	return respondOK(c, map[string]string{"status": "stopped"})
}

func (h *Handler) handleV1RestartService(c echo.Context) error {
	if err := h.requireWorkspaceRole(c, c.Param("ws"), identity.RoleWorkspaceAdmin); err != nil {
		return err
	}
	if err := h.services.RestartServer(nilLogger{}, workspaces.NameArg{Workspace: c.Param("ws"), Server: c.Param("name")}); err != nil {
		return respondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error(), nil)
	}
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
	sess, err := h.services.CreateProxySession(nilLogger{}, workspaces.NameArg{Workspace: wsID})
	if err != nil {
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
	h.seedStateFromRuntime()
	visible, err := h.visibleWorkspaceMap(c)
	if err != nil {
		return respondError(c, http.StatusForbidden, "FORBIDDEN", "failed to resolve workspace visibility", nil)
	}
	items := make([]map[string]interface{}, 0)
	for wsID := range h.getWorkspaceMap() {
		if visible != nil && !visible[wsID] {
			continue
		}
		for _, svc := range h.buildServiceViews(wsID) {
			meta, _ := h.state.getService(wsID, svc.Name)
			if meta == nil {
				continue
			}
			items = append(items, map[string]interface{}{
				"package_id":        valueOrDefault(meta.SourceRef, svc.Name),
				"package_name":      strings.Title(strings.ReplaceAll(valueOrDefault(meta.SourceRef, svc.Name), "-", " ")),
				"installed_version": valueOrDefault(meta.Version, "unknown"),
				"latest_version":    valueOrDefault(meta.Version, "unknown"),
				"workspace_id":      wsID,
				"workspace_name":    h.workspaceName(wsID),
				"service_name":      svc.Name,
				"status":            svc.Status,
				"installed_at":      meta.CreatedAt.UTC().Format(time.RFC3339),
			})
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i]["installed_at"].(string) > items[j]["installed_at"].(string)
	})
	return respondOK(c, listData{Items: items, Total: len(items), Page: 1, PageSize: len(items)})
}

func (h *Handler) handleV1MarketSources(c echo.Context) error {
	return respondOK(c, listData{
		Items:    defaultMarketSources,
		Total:    len(defaultMarketSources),
		Page:     1,
		PageSize: len(defaultMarketSources),
	})
}

func (h *Handler) handleV1MarketPackages(c echo.Context) error {
	page, pageSize := parsePageParams(c)
	q := strings.ToLower(strings.TrimSpace(c.QueryParam("q")))
	category := strings.TrimSpace(c.QueryParam("category"))
	sourceFilter := strings.TrimSpace(c.QueryParam("source"))
	items := make([]map[string]interface{}, 0, len(defaultMarketPackages))
	for _, pkg := range defaultMarketPackages {
		if q != "" && !strings.Contains(strings.ToLower(pkg.Name+" "+pkg.Description+" "+pkg.ID), q) {
			continue
		}
		if category != "" && category != "全部" && pkg.Category != category {
			continue
		}
		if sourceFilter != "" && pkg.SourceID != sourceFilter {
			continue
		}
		items = append(items, map[string]interface{}{
			"id":          pkg.ID,
			"name":        pkg.Name,
			"version":     pkg.Version,
			"description": pkg.Description,
			"author":      pkg.Author,
			"tags":        pkg.Tags,
			"rating":      pkg.Rating,
			"downloads":   pkg.Downloads,
			"verified":    pkg.Verified,
			"source_id":   pkg.SourceID,
			"category":    pkg.Category,
			"tools":       pkg.Tools,
		})
	}
	paged, total := paginate(items, page, pageSize)
	return respondOK(c, listData{Items: paged, Total: total, Page: page, PageSize: pageSize})
}

func (h *Handler) handleV1MarketPackageDetail(c echo.Context) error {
	pkg, ok := getMarketPackage(c.Param("id"))
	if !ok {
		return respondError(c, http.StatusNotFound, "NOT_FOUND", "package not found", nil)
	}
	return respondOK(c, map[string]interface{}{
		"id":          pkg.ID,
		"name":        pkg.Name,
		"version":     pkg.Version,
		"description": pkg.Description,
		"author":      pkg.Author,
		"tags":        pkg.Tags,
		"rating":      pkg.Rating,
		"downloads":   pkg.Downloads,
		"verified":    pkg.Verified,
		"source_id":   pkg.SourceID,
		"category":    pkg.Category,
		"install": map[string]interface{}{
			"type":    pkg.Install.Type,
			"command": pkg.Install.Command,
			"args":    pkg.Install.Args,
			"env":     pkg.Install.Env,
		},
		"tools":    pkg.Tools,
		"readme":   pkg.Readme,
		"versions": pkg.Versions,
	})
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
	if len(merged) > tail {
		merged = merged[len(merged)-tail:]
	}
	return respondOK(c, map[string]interface{}{
		"workspace_id": wsID,
		"total_lines":  len(merged),
		"logs":         merged,
	})
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
			Enabled *bool `json:"enabled"`
		} `json:"auth"`
	}
	if err := c.Bind(&req); err != nil {
		return respondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	}
	if req.Bind != nil {
		h.cfg.Bind = *req.Bind
	}
	if req.GatewayProtocol != nil && (*req.GatewayProtocol == "sse" || *req.GatewayProtocol == "streamhttp") {
		h.cfg.GatewayProtocol = *req.GatewayProtocol
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

func parseServiceRequest(workspaceID string, raw map[string]interface{}) (string, config.MCPServerConfig, serviceMeta, error) {
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
		pkg, ok := getMarketPackage(pkgID)
		if !ok {
			return "", config.MCPServerConfig{}, serviceMeta{}, fmt.Errorf("market package not found")
		}
		meta.SourceType = "market"
		meta.SourceRef = pkgID
		meta.Version = asString(raw["version"])
		cfg = packageConfigFromMarket(*pkg, workspaceID, asStringMap(raw["env"]))
		if meta.Version == "" {
			meta.Version = pkg.Version
		}
		return name, cfg, meta, nil
	}

	if url := asString(raw["url"]); url != "" {
		meta.SourceType = "url"
		cfg.URL = url
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
	return name, cfg, meta, nil
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
			"enabled":        h.cfg.GetAuthConfig().Enabled,
			"mode":           h.authMode(),
			"allow_register": h.cfg.GetAuthConfig().AllowRegister,
		},
	}
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
