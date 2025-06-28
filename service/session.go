// Session
// 用于存储会话状态，包括接收的消息和处理结果
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lucky-aeon/agentx/plugin-helper/types"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type McpName = string
type McpToolName = string

type Session struct {
	// 使用单一主锁减少死锁风险
	mu sync.RWMutex

	Id              string
	LastReceiveTime time.Time // 最后一次接收消息的时间

	// SSE事件通道 - 由主锁保护
	eventChans []chan SessionMsg
	doneChan   chan struct{}

	// 工具映射 - 由主锁保护
	mcpToolsMap    map[McpName]map[McpToolName]types.McpTool
	waitToolsCount atomic.Int32

	// 避免重复返回 - 由主锁保护
	lastMsg SessionMsg

	// V2
	mcpClients           map[McpName]client.MCPClient
	mcpinitializeResults map[McpName]*mcp.InitializeResult
}

func NewSession(id string) *Session {
	return &Session{
		Id:                   id,
		LastReceiveTime:      time.Now(),
		eventChans:           make([]chan SessionMsg, 0),
		doneChan:             make(chan struct{}),
		mcpToolsMap:          make(map[McpName]map[McpToolName]types.McpTool),
		waitToolsCount:       atomic.Int32{},
		mcpClients:           make(map[McpName]client.MCPClient),
		mcpinitializeResults: make(map[McpName]*mcp.InitializeResult),
	}
}

func (s *Session) GetId() string {
	return s.Id
}

func (s *Session) SendMessage(xl xlog.Logger, content json.RawMessage) (err error) {
	// 发送消息到 MCP 服务
	var request mcp.JSONRPCRequest
	if err = json.Unmarshal([]byte(content), &request); err != nil {
		xl.Errorf("failed to unmarshal request: %v", err)
		return fmt.Errorf("failed to unmarshal request: %w", err)
	}
	method := request.Method
	xl = xlog.WithChildName(method, xl)

	xl.Debugf("Sending request: %+v", request)

	// xl.Infof("method: %s, content: %s", method, content)
	var singleMcp McpName
	switch mcp.MCPMethod(request.Method) {
	case mcp.MethodToolsCall:
		req := mcp.CallToolRequest{}
		err := json.Unmarshal([]byte(content), &req)
		if err != nil {
			xl.Errorf("failed to unmarshal request: %v", err)
			return fmt.Errorf("failed to unmarshal request: %w", err)
		}

		// mcpName_toolName  ->  toolName
		if names := strings.Split(req.Params.Name, "_"); len(names) > 2 {
			singleMcp = names[0]
			req.Params.Name = strings.Join(names[1:], "_")
		}
	}

	// 对所有 MCP 服务器发送消息
	if singleMcp == "" {
		// xl.Infof("send to all MCP servers: %s", content)
		for s.waitToolsCount.Load() > 0 {
			time.Sleep(500 * time.Millisecond)
		}
		s.mu.Lock()
		s.mcpToolsMap = make(map[McpName]map[McpToolName]types.McpTool)
		mcpNames := make([]McpName, 0, len(s.mcpClients))
		for mcpName := range s.mcpClients {
			mcpNames = append(mcpNames, mcpName)
		}
		s.mu.Unlock()

		for _, mcpName := range mcpNames {
			if method == "tools/list" {
				s.waitToolsCount.Add(1)
			}
			err = s.sendToMcp(xl, mcpName, request, content)
			if err != nil {
				xl.Errorf("failed to send to allmcp: %v", err)
				continue
			}
		}
	} else {
		// xl.Infof("send to single MCP server: %s, content: %s", singleMcp, content)
		err = s.sendToMcp(xl, singleMcp, request, content)
		if err != nil {
			xl.Errorf("failed to send to singlemcp: %v", err)
			return err
		}
	}

	return nil
}

func (s *Session) sendToMcp(xl xlog.Logger, mcpName McpName, baseReq mcp.JSONRPCRequest, reqRaw json.RawMessage) error {
	xl = xlog.WithChildName(mcpName, xl)
	// 发送消息到 MCP 服务
	s.mu.RLock()
	mCli, ok := s.mcpClients[mcpName]
	s.mu.RUnlock()
	if !ok {
		err := fmt.Errorf("failed to find mcpClient for %s", mcpName)
		xl.Error(err)
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var result interface{}
	var err error

	switch mcp.MCPMethod(baseReq.Method) {
	case mcp.MethodInitialize:
		result = s.mcpinitializeResults[mcpName]

	case mcp.MethodPing:
		var request mcp.PingRequest
		err = json.Unmarshal(reqRaw, &request)
		if err != nil {
			xl.Errorf("failed to unmarshal ping request: %v", err)
			return err
		}
		err = mCli.Ping(ctx)
		result = &mcp.EmptyResult{}

	case mcp.MethodSetLogLevel:
		var request mcp.SetLevelRequest
		err = json.Unmarshal(reqRaw, &request)
		if err != nil {
			xl.Errorf("failed to unmarshal setLogLevel request: %v", err)
			return err
		}
		err = mCli.SetLevel(ctx, request)
		result = &mcp.EmptyResult{}

	case mcp.MethodResourcesList:
		var request mcp.ListResourcesRequest
		err = json.Unmarshal(reqRaw, &request)
		if err != nil {
			xl.Errorf("failed to unmarshal listResources request: %v", err)
			return err
		}
		result, err = mCli.ListResources(ctx, request)

	case mcp.MethodResourcesTemplatesList:
		var request mcp.ListResourceTemplatesRequest
		err = json.Unmarshal(reqRaw, &request)
		if err != nil {
			xl.Errorf("failed to unmarshal listResourceTemplates request: %v", err)
			return err
		}
		result, err = mCli.ListResourceTemplates(ctx, request)

	case mcp.MethodResourcesRead:
		var request mcp.ReadResourceRequest
		err = json.Unmarshal(reqRaw, &request)
		if err != nil {
			xl.Errorf("failed to unmarshal readResource request: %v", err)
			return err
		}
		result, err = mCli.ReadResource(ctx, request)

	case mcp.MethodPromptsList:
		var request mcp.ListPromptsRequest
		err = json.Unmarshal(reqRaw, &request)
		if err != nil {
			xl.Errorf("failed to unmarshal listPrompts request: %v", err)
			return err
		}
		result, err = mCli.ListPrompts(ctx, request)

	case mcp.MethodPromptsGet:
		var request mcp.GetPromptRequest
		err = json.Unmarshal(reqRaw, &request)
		if err != nil {
			xl.Errorf("failed to unmarshal getPrompt request: %v", err)
			return err
		}
		result, err = mCli.GetPrompt(ctx, request)

	case mcp.MethodToolsList:
		var request mcp.ListToolsRequest
		err = json.Unmarshal(reqRaw, &request)
		if err != nil {
			xl.Errorf("failed to unmarshal listTools request: %v", err)
			return err
		}
		result, err = mCli.ListTools(ctx, request)

		// 处理工具列表响应，更新工具映射
		if err == nil {
			if toolsResult, ok := result.(*mcp.ListToolsResult); ok {
				s.mu.Lock()
				if s.mcpToolsMap[mcpName] == nil {
					s.mcpToolsMap[mcpName] = make(map[McpToolName]types.McpTool)
				}
				for _, tool := range toolsResult.Tools {
					s.mcpToolsMap[mcpName][tool.Name] = types.McpTool{
						Name:        tool.Name,
						Description: tool.Description,
						InputSchema: tool.InputSchema,
					}
				}
				s.mu.Unlock()
				s.waitToolsCount.Add(-1)
			}
		}

	case mcp.MethodToolsCall:
		var request mcp.CallToolRequest
		err = json.Unmarshal(reqRaw, &request)
		if err != nil {
			xl.Errorf("failed to unmarshal callTool request: %v", err)
			return err
		}
		result, err = mCli.CallTool(ctx, request)

	default:
		return fmt.Errorf("unsupported method: %s", baseReq.Method)
	}

	if err != nil {
		xl.Errorf("failed to call MCP method %s: %v", baseReq.Method, err)
		// 发送错误响应
		s.sendErrorResponse(mcpName, baseReq.ID, err)
		return err
	}

	// 发送成功响应
	if result != nil {
		s.sendSuccessResponse(mcpName, baseReq.ID, result)
	}

	return nil
}

// SubscribeSSE 订阅MCP服务的SSE事件
func (s *Session) SubscribeSSE(xl xlog.Logger, mcpName McpName, sseUrl string) (err error) {
	cli, err := client.NewSSEMCPClient(sseUrl)
	if err != nil {
		xl.Errorf("failed to create SSE client: %v", err)
		return fmt.Errorf("failed to create SSE client: %v", err)
	}

	err = cli.Start(context.TODO())
	if err != nil {
		xl.Errorf("failed to start SSE client: %v", err)
		return fmt.Errorf("failed to start SSE client: %v", err)
	}

	result, err := cli.Initialize(context.TODO(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "test-sse-client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		xl.Errorf("failed to initialize SSE client: %v", err)
		return fmt.Errorf("failed to initialize SSE client: %v", err)
	}
	s.mcpinitializeResults[mcpName] = result

	xl.Info("initialized SSE client: ", result)

	err = cli.Ping(context.TODO())
	if err != nil {
		xl.Errorf("failed to ping SSE client: %v", err)
		return fmt.Errorf("failed to ping SSE client: %v", err)
	}

	s.mu.Lock()
	s.mcpClients[mcpName] = cli
	s.mu.Unlock()
	return nil
}

type SessionMsg struct {
	proxyId  int64
	clientId int64
	Event    string `json:"event"`
	Data     string `json:"data"`
}

// check lastMsg is 重复的
func (smsg *SessionMsg) isDuplicate(newMsg *SessionMsg) bool {
	if smsg.proxyId != 0 && smsg.proxyId == newMsg.proxyId {
		return true
	}
	if smsg.clientId != 0 && smsg.clientId == newMsg.clientId {
		return true
	}
	if smsg.Data == newMsg.Data {
		return true
	}
	return false
}

// Close 关闭会话
func (s *Session) Close() {
	// 先关闭所有SSE连接
	xl := xlog.NewLogger("session-" + s.Id)
	xl.Infof("Closing session: %s", s.Id)
	xl.Infof("Closing all SSE connections")

	xl.Infof("Session closed: %s", s.Id)
}

// SendEvent 发送SSE事件
func (s *Session) SendEvent(event SessionMsg) {
	xl := xlog.NewLogger("session-" + s.Id)
	xl.Debugf("Sending event: %s", event)
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.lastMsg.isDuplicate(&event) {
		xl.Debugf("Event already sent: %s", event.Event)
		return
	}

	s.LastReceiveTime = time.Now()
	for _, eventChan := range s.eventChans {
		select {
		case eventChan <- event:
			s.lastMsg = event
		default:
			// 如果通道已满，丢弃事件
		}
	}
}

// GetEventChan 获取事件通道
func (s *Session) GetEventChan() <-chan SessionMsg {
	s.mu.Lock()
	defer s.mu.Unlock()
	curChan := make(chan SessionMsg, 100)
	s.eventChans = append(s.eventChans, curChan)
	return curChan
}

// GetMcpTools 获取指定 MCP 的所有工具
func (s *Session) GetMcpTools(mcpName McpName) map[McpToolName]types.McpTool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if tools, ok := s.mcpToolsMap[mcpName]; ok {
		// 创建一个副本以避免外部修改
		result := make(map[McpToolName]types.McpTool)
		for k, v := range tools {
			result[k] = v
		}
		return result
	}
	return nil
}

// GetMcpTool 获取指定 MCP 的指定工具
func (s *Session) GetMcpTool(mcpName McpName, toolName McpToolName) (types.McpTool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if tools, ok := s.mcpToolsMap[mcpName]; ok {
		if tool, ok := tools[toolName]; ok {
			return tool, true
		}
	}
	return types.McpTool{}, false
}

// sendSuccessResponse 发送成功响应到SSE
func (s *Session) sendSuccessResponse(mcpName McpName, requestId interface{}, result interface{}) {
	// Convert interface{} to RequestId
	var reqId mcp.RequestId
	if requestId != nil {
		reqId = mcp.NewRequestId(requestId)
	}

	response := mcp.JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      reqId,
		Result:  result,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		xl := xlog.NewLogger("session-" + s.Id)
		xl.Errorf("failed to marshal response: %v", err)
		return
	}

	// 发送SSE事件
	event := SessionMsg{
		Event: "message",
		Data:  string(responseData),
	}
	s.SendEvent(event)
}

// sendErrorResponse 发送错误响应到SSE
func (s *Session) sendErrorResponse(mcpName McpName, requestId interface{}, err error) {
	// Convert interface{} to RequestId
	var reqId mcp.RequestId
	if requestId != nil {
		reqId = mcp.NewRequestId(requestId)
	}

	response := mcp.JSONRPCError{
		JSONRPC: "2.0",
		ID:      reqId,
		Error: struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Data    any    `json:"data,omitempty"`
		}{
			Code:    mcp.INTERNAL_ERROR,
			Message: err.Error(),
		},
	}

	responseData, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		xl := xlog.NewLogger("session-" + s.Id)
		xl.Errorf("failed to marshal error response: %v", jsonErr)
		return
	}

	// 发送SSE事件
	event := SessionMsg{
		Event: "message",
		Data:  string(responseData),
	}
	s.SendEvent(event)
}

func (s *Session) ping() {

}
