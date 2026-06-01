package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/identity"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/oplog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
)

func (h *Handler) appendOperation(ctx context.Context, principal *identity.Principal, level oplog.Level, action, workspaceID, sessionID, message, errText string, detail map[string]interface{}) {
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
		"resource_type": "session",
		"resource_id":   sessionID,
		"actor_id":      actorID,
	}
	for k, v := range detail {
		fields[k] = v
	}
	if errText != "" {
		fields["error"] = errText
	}
	logger := xlog.NewLogger("mcp-gateway").WithFields(fields)
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

func gatewayPrincipal(c echo.Context) *identity.Principal {
	if v := c.Get("auth.principal"); v != nil {
		if principal, ok := v.(*identity.Principal); ok {
			return principal
		}
	}
	return nil
}

type rpcLogInfo struct {
	Method    string
	RequestID string
	ToolName  string
	MCPName   string
	Action    string
	Message   string
}

func rpcLogInfoFromBody(body []byte, fallbackMethod string) rpcLogInfo {
	var raw struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Name string `json:"name"`
		} `json:"params"`
	}
	_ = json.Unmarshal(body, &raw)
	method := raw.Method
	if method == "" {
		method = fallbackMethod
	}
	info := rpcLogInfo{
		Method:    method,
		RequestID: rawIDString(raw.ID),
		Action:    "session.request",
		Message:   "MCP request: " + method,
	}
	switch method {
	case "initialize":
		info.Action = "session.initialize"
		info.Message = "MCP session initialized"
	case "tools/list":
		info.Action = "tool.list"
		info.Message = "List MCP tools"
	case "tools/call":
		info.Action = "tool.call"
		info.ToolName = raw.Params.Name
		info.MCPName, _ = splitGatewayToolName(info.ToolName)
		if info.ToolName != "" {
			info.Message = "Tool call: " + info.ToolName
		} else {
			info.Message = "Tool call"
		}
	default:
		if strings.HasPrefix(method, "notifications/") {
			info.Action = "session.notification"
			info.Message = "MCP notification: " + method
		}
	}
	return info
}

func rawIDString(raw json.RawMessage) string {
	raw = json.RawMessage(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func splitGatewayToolName(name string) (string, string) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return "", name
	}
	return parts[0], parts[1]
}

func rpcLogDetail(info rpcLogInfo, transport string) map[string]interface{} {
	detail := map[string]interface{}{
		"transport": transport,
		"method":    info.Method,
	}
	if info.RequestID != "" {
		detail["request_id"] = info.RequestID
	}
	if info.ToolName != "" {
		detail["tool_name"] = info.ToolName
	}
	if info.MCPName != "" {
		detail["mcp_name"] = info.MCPName
	}
	return detail
}

func finishRPCLogDetail(detail map[string]interface{}, started time.Time, responseBody string) (oplog.Level, string) {
	if !started.IsZero() {
		detail["duration_ms"] = time.Since(started).Milliseconds()
	}
	if errText := jsonRPCResponseError(responseBody); errText != "" {
		detail["response_status"] = "error"
		return oplog.LevelError, errText
	}
	detail["response_status"] = "ok"
	return oplog.LevelInfo, ""
}

func jsonRPCResponseError(body string) string {
	var payload struct {
		Error interface{} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil || payload.Error == nil {
		return ""
	}
	switch v := payload.Error.(type) {
	case string:
		return v
	case map[string]interface{}:
		if msg, _ := v["message"].(string); msg != "" {
			if code, ok := v["code"]; ok {
				return fmt.Sprintf("%v: %s", code, msg)
			}
			return msg
		}
	}
	raw, _ := json.Marshal(payload.Error)
	return string(raw)
}
