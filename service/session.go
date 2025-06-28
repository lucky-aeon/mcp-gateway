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
	mcpToolsMap       map[McpName]map[McpToolName]mcp.Tool
	pendingToolsList  sync.WaitGroup // 等待所有MCP工具列表响应
	aggregatedTools   []mcp.Tool     // 聚合后的工具列表，工具名带MCP前缀
	toolsListComplete atomic.Bool    // 标记工具列表是否已完成聚合

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
		mcpToolsMap:          make(map[McpName]map[McpToolName]mcp.Tool),
		aggregatedTools:      make([]mcp.Tool, 0),
		toolsListComplete:    atomic.Bool{},
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
		if names := strings.Split(req.Params.Name, "_"); len(names) >= 2 {
			singleMcp = names[0]
			req.Params.Name = strings.Join(names[1:], "_")

			// 重新序列化请求以更新工具名
			updatedContent, err := json.Marshal(req)
			if err != nil {
				xl.Errorf("failed to marshal updated request: %v", err)
				return fmt.Errorf("failed to marshal updated request: %w", err)
			}
			content = updatedContent
		}
	}

	// 对所有 MCP 服务器发送消息
	if singleMcp == "" {
		// 如果是tools/list请求，需要特殊处理来聚合所有MCP的工具
		if method == "tools/list" {
			return s.handleToolsListRequest(xl, request)
		}

		// 其他请求照常处理
		s.mu.RLock()
		mcpNames := make([]McpName, 0, len(s.mcpClients))
		for mcpName := range s.mcpClients {
			mcpNames = append(mcpNames, mcpName)
		}
		s.mu.RUnlock()

		for _, mcpName := range mcpNames {
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

	result, err := s.handleMCPMethod(ctx, xl, mCli, mcpName, baseReq.Method, reqRaw)
	if err != nil {
		xl.Errorf("failed to call MCP method %s: %v", baseReq.Method, err)
		s.sendErrorResponse(baseReq.ID, err)
		return err
	}

	if result != nil {
		s.sendSuccessResponse(baseReq.ID, result)
	}

	return nil
}

// SubscribeSSE 订阅MCP服务的SSE事件
func (s *Session) SubscribeSSE(xl xlog.Logger, mcpName McpName, sseUrl string) error {
	cli, err := client.NewSSEMCPClient(sseUrl)
	if err != nil {
		return fmt.Errorf("failed to create SSE client: %w", err)
	}

	if err = cli.Start(context.TODO()); err != nil {
		return fmt.Errorf("failed to start SSE client: %w", err)
	}

	result, err := cli.Initialize(context.TODO(), mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "mcp-gateway-client",
				Version: "1.0.0",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize SSE client: %w", err)
	}

	if err = cli.Ping(context.TODO()); err != nil {
		return fmt.Errorf("failed to ping SSE client: %w", err)
	}

	xl.Info("SSE client initialized and connected successfully")

	// 优化：批量更新状态，减少锁竞争
	s.mu.Lock()
	s.mcpClients[mcpName] = cli
	s.mcpinitializeResults[mcpName] = result
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
	xl.Infof("Sending event: %s, data: %s", event.Event, event.Data)

	// 优化：一次性获取需要的数据，减少锁持有时间
	s.mu.RLock()
	isDuplicate := s.lastMsg.isDuplicate(&event)
	eventChans := make([]chan SessionMsg, len(s.eventChans))
	copy(eventChans, s.eventChans)
	s.mu.RUnlock()

	if isDuplicate {
		xl.Debugf("Event already sent: %s", event.Event)
		return
	}

	// 优化：并发发送事件到所有通道
	sentToChannels := s.broadcastEvent(eventChans, event, xl)

	// 只在成功发送后更新状态
	if sentToChannels > 0 {
		s.mu.Lock()
		s.LastReceiveTime = time.Now()
		s.lastMsg = event
		s.mu.Unlock()
		xl.Infof("Event sent to %d channels", sentToChannels)
	} else {
		xl.Warnf("Event not sent to any channels (total channels: %d)", len(eventChans))
	}
}

// broadcastEvent 并发广播事件到所有通道
func (s *Session) broadcastEvent(eventChans []chan SessionMsg, event SessionMsg, xl xlog.Logger) int {
	sentCount := int32(0)
	var wg sync.WaitGroup

	for i, eventChan := range eventChans {
		wg.Add(1)
		go func(chanIndex int, ch chan SessionMsg) {
			defer wg.Done()
			select {
			case ch <- event:
				atomic.AddInt32(&sentCount, 1)
				xl.Debugf("Sent event to channel %d", chanIndex)
			default:
				xl.Warnf("Channel %d is full, dropping event", chanIndex)
			}
		}(i, eventChan)
	}

	wg.Wait()
	return int(sentCount)
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
func (s *Session) GetMcpTools(mcpName McpName) map[McpToolName]mcp.Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if tools, ok := s.mcpToolsMap[mcpName]; ok {
		// 创建一个副本以避免外部修改
		result := make(map[McpToolName]mcp.Tool)
		for k, v := range tools {
			result[k] = v
		}
		return result
	}
	return nil
}

// GetMcpTool 获取指定 MCP 的指定工具
func (s *Session) GetMcpTool(mcpName McpName, toolName McpToolName) (mcp.Tool, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if tools, ok := s.mcpToolsMap[mcpName]; ok {
		if tool, ok := tools[toolName]; ok {
			return tool, true
		}
	}
	return mcp.Tool{}, false
}

// sendResponse 统一的响应发送方法
func (s *Session) sendResponse(requestId interface{}, result interface{}, err error) {
	var responseData []byte
	var marshalErr error

	reqId := mcp.NewRequestId(requestId)

	if err != nil {
		// 发送错误响应
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
		responseData, marshalErr = json.Marshal(response)
	} else {
		// 发送成功响应
		response := mcp.JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      reqId,
			Result:  result,
		}
		responseData, marshalErr = json.Marshal(response)
	}

	if marshalErr != nil {
		xl := xlog.NewLogger("session-" + s.Id)
		xl.Errorf("failed to marshal response: %v", marshalErr)
		return
	}

	s.SendEvent(SessionMsg{
		Event: "message",
		Data:  string(responseData),
	})
}

// sendSuccessResponse 发送成功响应到SSE
func (s *Session) sendSuccessResponse(requestId interface{}, result interface{}) {
	s.sendResponse(requestId, result, nil)
}

// sendErrorResponse 发送错误响应到SSE
func (s *Session) sendErrorResponse(requestId interface{}, err error) {
	s.sendResponse(requestId, nil, err)
}

// handleToolsListRequest 处理工具列表请求，等待所有MCP响应后聚合结果
func (s *Session) handleToolsListRequest(xl xlog.Logger, request mcp.JSONRPCRequest) error {
	xl.Debugf("Handling tools list request for all MCPs")

	// 重置工具列表状态
	s.mu.Lock()
	s.mcpToolsMap = make(map[McpName]map[McpToolName]mcp.Tool)
	s.aggregatedTools = make([]mcp.Tool, 0)
	s.toolsListComplete.Store(false)

	// 获取所有MCP客户端
	mcpNames := make([]McpName, 0, len(s.mcpClients))
	for mcpName := range s.mcpClients {
		mcpNames = append(mcpNames, mcpName)
	}
	s.mu.Unlock()

	if len(mcpNames) == 0 {
		xl.Warn("No MCP clients available for tools list request")
		// 发送空工具列表响应
		emptyResult := &mcp.ListToolsResult{Tools: []mcp.Tool{}}
		s.sendSuccessResponse(request.ID, emptyResult)
		return nil
	}

	// 设置等待计数
	s.pendingToolsList.Add(len(mcpNames))

	// 启动后台goroutine等待所有响应完成
	go s.waitForAllToolsResponses(xl, request.ID)

	// 向所有MCP发送工具列表请求
	for _, mcpName := range mcpNames {
		go func(name McpName) {
			defer func() {
				if r := recover(); r != nil {
					xl.Errorf("Panic in tools list request for %s: %v", name, r)
					s.pendingToolsList.Done()
				}
			}()

			err := s.sendToolsListToMcp(xl, name, request)
			if err != nil {
				xl.Errorf("Failed to send tools list request to %s: %v", name, err)
				s.pendingToolsList.Done()
			}
		}(mcpName)
	}

	return nil
}

// sendToolsListToMcp 向单个MCP发送工具列表请求
func (s *Session) sendToolsListToMcp(xl xlog.Logger, mcpName McpName, baseReq mcp.JSONRPCRequest) error {
	xl = xlog.WithChildName(mcpName, xl)

	s.mu.RLock()
	mCli, ok := s.mcpClients[mcpName]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("failed to find mcpClient for %s", mcpName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	request := mcp.ListToolsRequest{
		PaginatedRequest: mcp.PaginatedRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsList),
			},
		},
	}

	result, err := mCli.ListTools(ctx, request)
	if err != nil {
		xl.Errorf("Failed to list tools from MCP %s: %v", mcpName, err)
		return err
	}

	// 处理工具列表响应
	s.mu.Lock()
	if s.mcpToolsMap[mcpName] == nil {
		s.mcpToolsMap[mcpName] = make(map[McpToolName]mcp.Tool)
	}
	for _, tool := range result.Tools {
		s.mcpToolsMap[mcpName][tool.Name] = tool
	}
	s.mu.Unlock()

	xl.Debugf("Received %d tools from MCP %s", len(result.Tools), mcpName)

	// 标记该MCP的工具列表已完成
	s.pendingToolsList.Done()
	return nil
}

// waitForAllToolsResponses 等待所有MCP的工具列表响应完成，然后聚合并发送结果
func (s *Session) waitForAllToolsResponses(xl xlog.Logger, requestId interface{}) {
	xl.Info("Waiting for all MCP tools list responses...")

	// 等待所有MCP响应完成（带超时）
	done := make(chan struct{})
	go func() {
		s.pendingToolsList.Wait()
		close(done)
	}()

	select {
	case <-done:
		xl.Info("All MCP tools list responses received")
	case <-time.After(30 * time.Second):
		xl.Warn("Timeout waiting for MCP tools list responses")
	}

	// 聚合所有工具并添加MCP名称前缀
	s.mu.Lock()
	s.aggregatedTools = make([]mcp.Tool, 0)
	for mcpName, tools := range s.mcpToolsMap {
		for _, tool := range tools {
			// 创建带前缀的工具副本
			prefixedTool := mcp.Tool{
				Name:        fmt.Sprintf("%s_%s", mcpName, tool.Name),
				Description: fmt.Sprintf("[%s] %s", mcpName, tool.Description),
				InputSchema: tool.InputSchema,
			}
			s.aggregatedTools = append(s.aggregatedTools, prefixedTool)
		}
	}
	s.mu.Unlock()

	s.toolsListComplete.Store(true)

	xl.Infof("Aggregated %d tools from %d MCPs", len(s.aggregatedTools), len(s.mcpToolsMap))

	// 聚合的工具已经是mcp.Tool格式，直接使用
	mcpTools := s.aggregatedTools

	// 发送聚合后的工具列表响应
	result := &mcp.ListToolsResult{
		Tools: mcpTools,
	}

	xl.Infof("Sending aggregated tools response with %d tools", len(mcpTools))
	s.sendSuccessResponse(requestId, result)
}

// GetAllTools 获取所有聚合后的工具列表（带MCP前缀）
func (s *Session) GetAllTools() []mcp.Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.toolsListComplete.Load() {
		return nil
	}

	// 返回副本以避免外部修改
	result := make([]mcp.Tool, len(s.aggregatedTools))
	copy(result, s.aggregatedTools)
	return result
}

// IsToolsListReady 检查工具列表是否已准备就绪
func (s *Session) IsToolsListReady() bool {
	return s.toolsListComplete.Load()
}

func (s *Session) handleMCPMethod(ctx context.Context, xl xlog.Logger, mCli client.MCPClient, mcpName McpName, method string, reqRaw json.RawMessage) (interface{}, error) {
	switch mcp.MCPMethod(method) {
	case mcp.MethodInitialize:
		return s.mcpinitializeResults[mcpName], nil

	case mcp.MethodPing:
		var request mcp.PingRequest
		if err := json.Unmarshal(reqRaw, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal ping request: %w", err)
		}
		return &mcp.EmptyResult{}, mCli.Ping(ctx)

	case mcp.MethodSetLogLevel:
		var request mcp.SetLevelRequest
		if err := json.Unmarshal(reqRaw, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal setLogLevel request: %w", err)
		}
		return &mcp.EmptyResult{}, mCli.SetLevel(ctx, request)

	case mcp.MethodResourcesList:
		var request mcp.ListResourcesRequest
		if err := json.Unmarshal(reqRaw, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal listResources request: %w", err)
		}
		return mCli.ListResources(ctx, request)

	case mcp.MethodResourcesTemplatesList:
		var request mcp.ListResourceTemplatesRequest
		if err := json.Unmarshal(reqRaw, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal listResourceTemplates request: %w", err)
		}
		return mCli.ListResourceTemplates(ctx, request)

	case mcp.MethodResourcesRead:
		var request mcp.ReadResourceRequest
		if err := json.Unmarshal(reqRaw, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal readResource request: %w", err)
		}
		return mCli.ReadResource(ctx, request)

	case mcp.MethodPromptsList:
		var request mcp.ListPromptsRequest
		if err := json.Unmarshal(reqRaw, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal listPrompts request: %w", err)
		}
		return mCli.ListPrompts(ctx, request)

	case mcp.MethodPromptsGet:
		var request mcp.GetPromptRequest
		if err := json.Unmarshal(reqRaw, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal getPrompt request: %w", err)
		}
		return mCli.GetPrompt(ctx, request)

	case mcp.MethodToolsList:
		var request mcp.ListToolsRequest
		if err := json.Unmarshal(reqRaw, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal listTools request: %w", err)
		}
		result, err := mCli.ListTools(ctx, request)
		if err == nil {
			s.updateToolsMap(mcpName, result)
		}
		return result, err

	case mcp.MethodToolsCall:
		var request mcp.CallToolRequest
		if err := json.Unmarshal(reqRaw, &request); err != nil {
			return nil, fmt.Errorf("failed to unmarshal callTool request: %w", err)
		}
		return mCli.CallTool(ctx, request)

	default:
		return nil, fmt.Errorf("unsupported method: %s", method)
	}
}

func (s *Session) updateToolsMap(mcpName McpName, result *mcp.ListToolsResult) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mcpToolsMap[mcpName] == nil {
		s.mcpToolsMap[mcpName] = make(map[McpToolName]mcp.Tool)
	}
	for _, tool := range result.Tools {
		s.mcpToolsMap[mcpName][tool.Name] = tool
	}
}

func (s *Session) ping() {

}
