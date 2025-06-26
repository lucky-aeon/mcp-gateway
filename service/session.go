// Session
// 用于存储会话状态，包括接收的消息和处理结果
package service

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lucky-aeon/agentx/plugin-helper/types"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
	"github.com/tidwall/gjson"
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

	// SSE订阅 - 由主锁保护
	sseWaitGroup sync.WaitGroup
	sseConns     map[McpName]*http.Response // 存储SSE连接，用于关闭
	sseCount     atomic.Int32

	// 消息相关 - 由主锁保护
	mcpMessageUrl map[McpName]string
	messageIds    map[int64]int64

	// 工具映射 - 由主锁保护
	mcpToolsMap    map[McpName]map[McpToolName]types.McpTool
	waitToolsCount atomic.Int32

	// 避免重复返回 - 由主锁保护
	lastMsg SessionMsg
}

func NewSession(id string) *Session {
	return &Session{
		Id:              id,
		LastReceiveTime: time.Now(),
		mcpMessageUrl:   make(map[McpName]string),
		messageIds:      make(map[int64]int64),
		mcpToolsMap:     make(map[McpName]map[McpToolName]types.McpTool),
		waitToolsCount:  atomic.Int32{},
		sseConns:        make(map[McpName]*http.Response),
	}
}

func (s *Session) GetId() string {
	return s.Id
}

func (s *Session) SendMessage(xl xlog.Logger, content string) (err error) {
	// 发送消息到 MCP 服务
	var request types.McpRequest
	if err = json.Unmarshal([]byte(content), &request); err != nil {
		xl.Errorf("failed to unmarshal request: %v", err)
		return fmt.Errorf("failed to unmarshal request: %w", err)
	}
	method := request.Method
	xl = xlog.WithChildName(method, xl)

	xl.Debugf("Sending request: %v", content)

	// check is ok
	try := 3
	for !s.IsReady() {
		if try < 0 {
			return fmt.Errorf("service not ready")
		}
		time.Sleep(time.Second)
		try--
	}

	// xl.Infof("method: %s, content: %s", method, content)
	var singleMcp McpName
	if method == "tools/call" {

		params, ok := request.Params.(map[string]any)
		if !ok {
			xl.Errorf("failed to get params")
			return fmt.Errorf("failed to get params")
		}
		name, ok := params["name"].(string)
		if !ok {
			xl.Errorf("failed to get name")
			return fmt.Errorf("failed to get name")
		}
		if names := strings.Split(name, "_"); len(names) > 1 {
			singleMcp = names[0]
			params["name"] = strings.Join(names[1:], "_")
		}
		request.Params = params
	}

	// 对所有 MCP 服务器发送消息
	if singleMcp == "" {
		// xl.Infof("send to all MCP servers: %s", content)
		for s.waitToolsCount.Load() > 0 {
			time.Sleep(500 * time.Millisecond)
		}
		s.mu.Lock()
		s.mcpToolsMap = make(map[McpName]map[McpToolName]types.McpTool)
		s.mu.Unlock()
		for mcpName := range s.mcpMessageUrl {
			if method == "tools/list" {
				s.waitToolsCount.Add(1)
			}
			err = s.sendToMcp(xl, mcpName, request)
			if err != nil {
				xl.Errorf("failed to send to allmcp: %v", err)
				continue
			}
		}
	} else {
		// xl.Infof("send to single MCP server: %s, content: %s", singleMcp, content)
		err = s.sendToMcp(xl, singleMcp, request)
		if err != nil {
			xl.Errorf("failed to send to singlemcp: %v", err)
			return err
		}
	}

	return nil
}

func (s *Session) generateMessageId(realMessageId int64) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 生成唯一的消息ID
	now := int64(time.Now().UnixMilli())

	xlog.NewLogger("session-"+s.Id).Debugf("generate message id: %d, real message id: %d", now, realMessageId)
	s.messageIds[now] = realMessageId
	return now
}

func (s *Session) getRealMessageId(messageId int64) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	realMessageId, exists := s.messageIds[messageId]
	return realMessageId, exists
}

func (s *Session) removeMessageId(messageId int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.messageIds, messageId)
}

func (s *Session) sendToMcp(xl xlog.Logger, mcpName McpName, request types.McpRequest) error {
	xl = xlog.WithChildName(mcpName, xl)
	// 发送消息到 MCP 服务
	// 生成唯一的消息ID
	if request.Id != nil {
		id := s.generateMessageId(*request.Id)
		// 替换消息中的ID
		request.Id = &id
	}

	mcpMessageUrl, ok := s.mcpMessageUrl[mcpName]
	if !ok {
		err := fmt.Errorf("failed to find mcpMessageUrl for %s", mcpName)
		xl.Error(err)
		return err
	}
	// xl.Debugf("Sending message to %s: %s", mcpName, content)
	resp, err := http.Post(mcpMessageUrl, "application/json", strings.NewReader(request.ToJson()))
	if err != nil {
		xl.Errorf("failed to send message: %v", err)
		return fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		xl.Errorf("failed to send message, status code: %d", resp.StatusCode)
		return fmt.Errorf("failed to send message, status code: %d", resp.StatusCode)
	}
	return nil
}

func (s *Session) IsReady() bool {
	load := int(s.sseCount.Load())
	mcpUrls := len(s.mcpMessageUrl)
	return load == mcpUrls
}

// SubscribeSSE 订阅MCP服务的SSE事件
func (s *Session) SubscribeSSE(mcpName McpName, sseUrl string) (err error) {
	s.sseWaitGroup.Add(1)
	s.sseCount.Add(1)
	xl := xlog.WithChildName(s.Id, xlog.NewLogger("SSE-RECEIVE-"+string(mcpName)))

	xl.Infof("Subscribing to SSE: %s", sseUrl)
	resp, err := http.Get(sseUrl)
	if err != nil {
		xl.Errorf("failed to subscribe SSE: %v", err)
		return err
	}
	// 保存连接以便后续关闭
	s.mu.Lock()
	s.sseConns[mcpName] = resp
	s.mu.Unlock()

	go func() {
		defer func() {
			s.sseWaitGroup.Done()
			s.sseCount.Add(-1)
		}()

		defer func() {
			s.mu.Lock()
			if err := resp.Body.Close(); err != nil {
				xl.Errorf("failed to close SSE: %v", err)
			}
			s.mu.Unlock()
		}()

		reader := bufio.NewReader(resp.Body)
		var currentEvent string

		for {
			select {
			case <-s.doneChan:
				if err := resp.Body.Close(); err != nil {
					xl.Errorf("failed to close SSE: %v", err)
				}
				xl.Infof("Closed SSE subscription: %s", sseUrl)
				return
			default:
				line, err := reader.ReadString('\n')
				if err != nil {
					xl.Errorf("failed to read SSE: %v", err)
					return
				}
				line = strings.TrimSpace(line)

				if line == "" {
					continue
				}

				if strings.HasPrefix(line, "event: ") {
					currentEvent = strings.TrimPrefix(line, "event: ")
				} else if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")
					// 如果是endpoint事件，保存endpoint
					if currentEvent == "endpoint" && s.mcpMessageUrl[mcpName] == "" {
						xl.Infof("Add SSE endpoint: %s", data)
						s.mcpMessageUrl[mcpName] = fmt.Sprintf("%s://%s%s", resp.Request.URL.Scheme, resp.Request.Host, data)
					}

					curMsg := SessionMsg{Event: currentEvent, Data: data}
					if gjson.Get(data, "id").Exists() {
						messageId := gjson.Get(data, "id").Int()
						// 检查是否是当前会话的消息
						realMessage, exists := s.getRealMessageId(messageId)
						curMsg.proxyId, curMsg.clientId = messageId, realMessage
						xl.Debugf("recevied: id(%d), clientId(%d)", messageId, realMessage)
						if !exists {
							continue
						}
						xl.Infof("SSE received(%s): %s", currentEvent, data)
						s.removeMessageId(messageId)
						// 将消息ID替换为当前会话ID
						data = strings.Replace(data, fmt.Sprintf(`"id":%d`, messageId), fmt.Sprintf(`"id":%d`, realMessage), 1)

						// 获取tools
						if tools := gjson.Get(data, "result.tools").Array(); len(tools) > 0 {
							isContinue := func() bool {
								s.mu.Lock()
								defer s.mu.Unlock()
								s.mcpToolsMap[mcpName] = make(map[McpToolName]types.McpTool)
								for _, toolJ := range tools {
									var tool types.McpTool
									if err := json.Unmarshal([]byte(toolJ.Raw), &tool); err != nil {
										xl.Errorf("Failed to unmarshal tool: %v", err)
										return false
									}
									tool.RealName = tool.Name
									tool.Name = fmt.Sprintf("%s_%s", mcpName, tool.Name)
									s.mcpToolsMap[mcpName][McpToolName(tool.RealName)] = tool
								}
								if s.waitToolsCount.Add(-1) > 0 {
									// 还没有准备好，继续等待
									xl.Debugf("Waiting for tools to be ready in session %s", s.Id)
									return false
								}
								xl.Debugf("Tools ready in session %s", s.Id)
								// 工具准备好，通知客户端
								allTools := make([]types.McpTool, 0, len(s.mcpToolsMap))
								for _, tools := range s.mcpToolsMap {
									for _, tool := range tools {
										allTools = append(allTools, tool)
									}
								}
								newResult := types.CreateMcpResult(gjson.Get(data, "jsonrpc").String(), int64(realMessage), map[string]any{"tools": allTools})
								data = newResult.ToJson()
								return true
							}()
							if !isContinue {
								continue
							}
						} else if get := gjson.Get(data, "result.serverInfo.name"); get.Exists() {
							// handler mcpname
							xl.Infof("replace mcpName: %s", get.String())
							data = strings.Replace(data, get.String(), "mcp-gateway", 1)
						}

						//
						// 记录接收到的消息
						// s.AddMessage(mcpName, data, "receive")

						// 如果不是endpoint事件，转发给客户端
						if currentEvent != "endpoint" {
							curMsg.Data = data
							s.SendEvent(curMsg)
						}
					}
				}
			}
		}
	}()
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
	for _, conn := range s.sseConns {
		xl.Infof("Closing SSE connection: %s", conn.Request.URL.String())
		if err := conn.Body.Close(); err != nil {
			xl.Errorf("failed to close SSE connection: %v", err)
		}
	}

	s.sseWaitGroup.Wait() // 等待所有SSE订阅goroutine结束

	xl.Infof("Session closed: %s", s.Id)
}

// SendEvent 发送SSE事件
func (s *Session) SendEvent(event SessionMsg) {
	xl := xlog.NewLogger("session-" + s.Id)
	xl.Infof("Sending event: %s", event)
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

func (s *Session) ping() {

}
