package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/httpx"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/sessions"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/workspaces"
	"github.com/mark3labs/mcp-go/mcp"
)

const (
	headerMcpSessionID       = "Mcp-Session-Id"
	streamHTTPWaitTimeout    = 30 * time.Second
	streamHTTPKeepAliveEvery = 30 * time.Second
	methodNotificationsInit  = "notifications/initialized"
)

// handleStreamHTTP 单服务 Streamable HTTP 反向代理。
// 仅把请求体按原样转给下游 MCP 服务暴露的 message 端点。
func (h *Handler) handleStreamHTTP(c echo.Context) error {
	xl := xlog.NewLogger("STREAMHTTP")
	serviceName := c.Param("service")
	workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)

	instance, err := h.services.GetMcpService(xl, workspaces.NameArg{
		Server:    serviceName,
		Workspace: workspace,
	})
	if err != nil {
		return c.String(http.StatusNotFound, "Service not found")
	}

	targetURL := instance.GetMessageUrl()
	if targetURL == "" {
		return c.String(http.StatusServiceUnavailable, "Service not available")
	}

	req, err := http.NewRequest(c.Request().Method, targetURL, c.Request().Body)
	if err != nil {
		return err
	}
	for k, v := range c.Request().Header {
		req.Header[k] = v
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		c.Response().Header()[k] = v
	}
	c.Response().WriteHeader(resp.StatusCode)
	_, err = io.Copy(c.Response().Writer, resp.Body)
	return err
}

// handleGlobalStreamHTTP 是全局 /stream 聚合入口，实现 MCP Streamable HTTP 协议
// (spec: 2025-03-26)。支持 POST/GET/DELETE 三种动作，会话通过
// `Mcp-Session-Id` 响应/请求头传递。
func (h *Handler) handleGlobalStreamHTTP(c echo.Context) error {
	xl := xlog.NewLogger("GLOBAL-STREAMHTTP")
	workspace := httpx.GetWorkspace(c, workspaces.DefaultWorkspace)

	switch c.Request().Method {
	case http.MethodGet:
		return h.streamHTTPEventStream(c, xl, workspace)
	case http.MethodPost:
		return h.streamHTTPHandlePost(c, xl, workspace)
	case http.MethodDelete:
		return h.streamHTTPHandleDelete(c, xl, workspace)
	default:
		c.Response().Header().Set("Allow", "GET, POST, DELETE")
		return c.NoContent(http.StatusMethodNotAllowed)
	}
}

// streamHTTPHandlePost 处理 POST /stream：
//   - 无 Mcp-Session-Id 的 `initialize` 请求：创建新 session，响应头返回 session id，
//     body 返回聚合后的 InitializeResult（不转发到下游，下游已在 CreateProxySession 内 initialize）。
//   - `notifications/initialized` 通知：直接 202，无需转发。
//   - notification（无 id）：异步转发后 202。
//   - request（有 id）：订阅 session 事件通道，转发后等待 id 匹配的响应并同步返回。
func (h *Handler) streamHTTPHandlePost(c echo.Context, xl xlog.Logger, workspace string) error {
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return writeJSONRPCError(c, http.StatusBadRequest, nil, -32700, "failed to read request body", err.Error())
	}

	peek, err := peekJSONRPC(body)
	if err != nil {
		return writeJSONRPCError(c, http.StatusBadRequest, nil, -32700, "parse error", err.Error())
	}

	sessionID := c.Request().Header.Get(headerMcpSessionID)
	// initialize 之外均要求携带 session id
	if peek.Method == string(mcp.MethodInitialize) {
		return h.streamHTTPHandleInitialize(c, xl, workspace, peek)
	}

	if sessionID == "" {
		return writeJSONRPCError(c, http.StatusBadRequest, peek.ID, -32600, fmt.Sprintf("missing %s header", headerMcpSessionID), nil)
	}

	session, ok := h.services.GetProxySession(xl, workspaces.NameArg{
		Workspace: workspace,
		Session:   sessionID,
	})
	if !ok {
		// 404 会驱动客户端重新 initialize，对齐官方 client 行为
		return c.String(http.StatusNotFound, "session not found")
	}

	// notifications/initialized 是 client → server 的通知，网关直接 ACK 即可
	if peek.Method == methodNotificationsInit {
		return c.NoContent(http.StatusAccepted)
	}

	// 通知：无 id，异步广播到下游
	if peek.IsNotification() {
		go func() {
			if err := session.SendMessage(xl, body); err != nil {
				xl.Warnf("send notification %s failed: %v", peek.Method, err)
			}
		}()
		return c.NoContent(http.StatusAccepted)
	}

	// request：订阅后转发，等待 id 匹配的响应
	return h.streamHTTPForwardAndWait(c, xl, session, body, peek)
}

// streamHTTPHandleInitialize 处理首次 initialize：创建 session，响应头带 session id，
// body 返回聚合 InitializeResult（下游 MCP 的 initialize 已在 session 建立时完成）。
func (h *Handler) streamHTTPHandleInitialize(c echo.Context, xl xlog.Logger, workspace string, peek jsonRPCPeek) error {
	session, err := h.services.CreateProxySession(xl, workspaces.NameArg{
		Workspace: workspace,
	})
	if err != nil {
		xl.Errorf("create session failed: %v", err)
		return writeJSONRPCError(c, http.StatusInternalServerError, peek.ID, -32000, "failed to create session", err.Error())
	}

	c.Response().Header().Set(headerMcpSessionID, session.Id)
	return writeJSONRPCResult(c, peek.ID, buildGatewayInitializeResult(session))
}

// streamHTTPForwardAndWait 在订阅 session 事件通道的前提下转发请求，并同步等待
// 匹配 id 的响应消息。
func (h *Handler) streamHTTPForwardAndWait(c echo.Context, xl xlog.Logger, session *sessions.Session, body []byte, peek jsonRPCPeek) error {
	eventChan, closer := session.GetEventChanWithCloser()
	defer closer()

	sendErrCh := make(chan error, 1)
	go func() {
		sendErrCh <- session.SendMessage(xl, body)
	}()

	ctx, cancel := context.WithTimeout(c.Request().Context(), streamHTTPWaitTimeout)
	defer cancel()

	targetID := normalizeRawID(peek.ID)
	var sendErr error
	sendDone := false
	for {
		select {
		case <-ctx.Done():
			if sendErr != nil {
				return writeJSONRPCError(c, http.StatusBadGateway, peek.ID, -32000, "forward failed", sendErr.Error())
			}
			return writeJSONRPCError(c, http.StatusGatewayTimeout, peek.ID, -32000, "timeout waiting for downstream response", nil)
		case err := <-sendErrCh:
			sendErr = err
			sendDone = true
			if err != nil {
				xl.Warnf("SendMessage returned error: %v", err)
			}
		case evt, ok := <-eventChan:
			if !ok {
				if sendDone && sendErr != nil {
					return writeJSONRPCError(c, http.StatusBadGateway, peek.ID, -32000, "forward failed", sendErr.Error())
				}
				return writeJSONRPCError(c, http.StatusInternalServerError, peek.ID, -32000, "session closed before response", nil)
			}
			if evt.Event != "" && evt.Event != "message" {
				continue
			}
			if !responseMatchesID(evt.Data, targetID) {
				continue
			}
			c.Response().Header().Set("Content-Type", "application/json")
			c.Response().WriteHeader(http.StatusOK)
			_, err := c.Response().Write([]byte(evt.Data))
			return err
		}
	}
}

// streamHTTPEventStream 处理 GET /stream：长连接 SSE 流，把 session 的事件广播
// 给当前客户端，常用于 server → client 的主动推送。
func (h *Handler) streamHTTPEventStream(c echo.Context, xl xlog.Logger, workspace string) error {
	sessionID := c.Request().Header.Get(headerMcpSessionID)
	if sessionID == "" {
		sessionID = c.QueryParam("sessionId")
	}
	if sessionID == "" {
		return c.String(http.StatusBadRequest, fmt.Sprintf("missing %s header", headerMcpSessionID))
	}

	session, ok := h.services.GetProxySession(xl, workspaces.NameArg{
		Workspace: workspace,
		Session:   sessionID,
	})
	if !ok {
		return c.String(http.StatusNotFound, "session not found")
	}

	c.Response().Header().Set("Content-Type", "text/event-stream")
	c.Response().Header().Set("Cache-Control", "no-cache")
	c.Response().Header().Set("Connection", "keep-alive")
	c.Response().WriteHeader(http.StatusOK)

	w := c.Response().Writer
	flusher, ok := w.(http.Flusher)
	if !ok {
		return c.String(http.StatusInternalServerError, "streaming not supported")
	}

	eventChan, closer := session.GetEventChanWithCloser()
	defer closer()

	ping := time.NewTicker(streamHTTPKeepAliveEvery)
	defer ping.Stop()

	for {
		select {
		case <-c.Request().Context().Done():
			xl.Infof("Client closed stream, sessionId=%s", sessionID)
			return nil
		case <-ping.C:
			if _, err := fmt.Fprintf(w, ": keepalive\n\n"); err != nil {
				return nil
			}
			flusher.Flush()
		case evt, ok := <-eventChan:
			if !ok {
				return nil
			}
			// 规范: JSON-RPC responses 必须走原始 POST 的 HTTP response，不能
			// 通过 GET SSE 流推送。GET 只承载 server→client 的 request/notification。
			if isJSONRPCResponse(evt.Data) {
				continue
			}
			eventName := evt.Event
			if eventName == "" {
				eventName = "message"
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventName, evt.Data); err != nil {
				return nil
			}
			flusher.Flush()
		}
	}
}

// streamHTTPHandleDelete 关闭 session。
func (h *Handler) streamHTTPHandleDelete(c echo.Context, xl xlog.Logger, workspace string) error {
	sessionID := c.Request().Header.Get(headerMcpSessionID)
	if sessionID == "" {
		return c.String(http.StatusBadRequest, fmt.Sprintf("missing %s header", headerMcpSessionID))
	}
	h.services.CloseProxySession(xl, workspaces.NameArg{
		Workspace: workspace,
		Session:   sessionID,
	})
	return c.NoContent(http.StatusOK)
}

// --------- helpers ---------

// jsonRPCPeek 用来从 request body 里快速抽取 JSON-RPC 关键字段，避免反序列化成
// 完整结构体时受 params 中动态字段影响。
type jsonRPCPeek struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
}

// IsNotification 返回是否是 JSON-RPC notification（没有 id 字段或 id 为 null）。
func (p jsonRPCPeek) IsNotification() bool {
	if len(p.ID) == 0 {
		return true
	}
	trimmed := bytes.TrimSpace(p.ID)
	return len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null"))
}

func peekJSONRPC(body []byte) (jsonRPCPeek, error) {
	var p jsonRPCPeek
	if err := json.Unmarshal(body, &p); err != nil {
		return p, err
	}
	return p, nil
}

// normalizeRawID 去除两侧空白后返回 id 的规范化字节切片。
func normalizeRawID(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return nil
	}
	return bytes.TrimSpace(raw)
}

// responseMatchesID 解析响应 body，判断其 id 是否与目标一致。
func responseMatchesID(data string, target []byte) bool {
	if len(target) == 0 {
		return false
	}
	var peek struct {
		ID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal([]byte(data), &peek); err != nil {
		return false
	}
	return bytes.Equal(bytes.TrimSpace(peek.ID), target)
}

// isJSONRPCResponse 判断一条消息是否为 JSON-RPC response（含 result 或 error、
// 有 id 且不含 method）。GET /stream 的 SSE 流只承载 server→client 的 request
// 或 notification，不应该广播 response。
func isJSONRPCResponse(data string) bool {
	var peek struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Result json.RawMessage `json:"result"`
		Error  json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal([]byte(data), &peek); err != nil {
		return false
	}
	if peek.Method != "" {
		return false
	}
	if len(bytes.TrimSpace(peek.ID)) == 0 {
		return false
	}
	return len(peek.Result) > 0 || len(peek.Error) > 0
}

// writeJSONRPCResult 写入 2xx JSON-RPC response。
func writeJSONRPCResult(c echo.Context, id json.RawMessage, result any) error {
	payload := map[string]any{
		"jsonrpc": mcp.JSONRPC_VERSION,
		"result":  result,
	}
	if len(id) > 0 {
		payload["id"] = json.RawMessage(id)
	} else {
		payload["id"] = nil
	}
	c.Response().Header().Set("Content-Type", "application/json")
	return c.JSON(http.StatusOK, payload)
}

// writeJSONRPCError 写入 JSON-RPC 错误响应。statusCode 用于 HTTP 层，
// JSON body 内是符合 JSON-RPC 规范的错误结构。
func writeJSONRPCError(c echo.Context, statusCode int, id json.RawMessage, code int, message string, data any) error {
	errObj := map[string]any{
		"code":    code,
		"message": message,
	}
	if data != nil {
		errObj["data"] = data
	}
	payload := map[string]any{
		"jsonrpc": mcp.JSONRPC_VERSION,
		"error":   errObj,
	}
	if len(id) > 0 {
		payload["id"] = json.RawMessage(id)
	} else {
		payload["id"] = nil
	}
	c.Response().Header().Set("Content-Type", "application/json")
	return c.JSON(statusCode, payload)
}

// buildGatewayInitializeResult 构造网关层聚合的 InitializeResult。
// capabilities 由 session 从已在线下游 MCP 的 initialize 结果 OR 合并而来，
// 确保网关只向 client 声明至少有一个下游真的支持的能力。
func buildGatewayInitializeResult(session *sessions.Session) *mcp.InitializeResult {
	return &mcp.InitializeResult{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
		ServerInfo: mcp.Implementation{
			Name:    "mcp-gateway",
			Version: "1.0.0",
		},
		Capabilities: session.AggregateCapabilities(),
		Instructions: "MCP Gateway aggregates multiple MCP servers. Tools are namespaced as <serverName>_<toolName>.",
	}
}
