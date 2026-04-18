package sessions

import (
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/runtime"
)

// ServiceLister 是 SessionManager 向外查询"当前可用 MCP 服务"的回调。
// 由调用方（通常是 workspaces.WorkSpace）提供，用于在会话建立时迭代
// 订阅每个下游 MCP 的 SSE 事件流。采用回调解耦是为了避免 sessions 包
// 反向依赖 workspaces 包（否则会形成循环依赖）。
type ServiceLister func() []*runtime.McpService

// SessionManager 管理单个工作区内的所有代理会话（proxy session）。
type SessionManager struct {
	sessions      map[string]*Session
	sessionsMutex sync.RWMutex
	listServices  ServiceLister
	sessionConfig CleanupConfig
}

// NewSessionManager 构造一个 SessionManager。
//   - listServices: 查询当前 workspace 下 MCP 服务的回调
//   - cleanupConfig: 会话闲置 TTL / 检查周期
func NewSessionManager(listServices ServiceLister, cleanupConfig CleanupConfig) *SessionManager {
	return &SessionManager{
		listServices:  listServices,
		sessions:      make(map[string]*Session),
		sessionConfig: normalizeCleanupConfig(cleanupConfig),
	}
}

// GetSession returns the session with the given id.
func (m *SessionManager) GetSession(_ xlog.Logger, sessionId string) (*Session, bool) {
	m.sessionsMutex.RLock()
	session, ok := m.sessions[sessionId]
	m.sessionsMutex.RUnlock()
	if !ok {
		return nil, false
	}
	return session, true
}

// CreateSession creates a new session and subscribes it to every running MCP service.
func (m *SessionManager) CreateSession(xl xlog.Logger) (*Session, error) {
	session := newSession(uuid.New().String(), m.sessionConfig)
	if m.existsSession(session.Id) {
		xl.Errorf("session %s already exists", session.Id)
		return nil, fmt.Errorf("session %s already exists", session.Id)
	}

	// 设置清理回调
	session.SetCleanupCallback(func(sessionId string) {
		xl.Infof("Auto-cleaning inactive session: %s", sessionId)
		m.CloseSession(xl, sessionId)
	})

	for _, mcpService := range m.listServices() {
		if mcpService.GetStatus() != runtime.Running {
			xl.Warnf("service %s is not running", mcpService.Name)
			continue
		}
		if err := session.SubscribeSSE(xl, mcpService.Name, mcpService.GetSSEUrl()); err != nil {
			xl.Errorf("failed to subscribe to SSE for service %s: %v", mcpService.Name, err)
			return nil, fmt.Errorf("failed to subscribe mcpServer[%s]", mcpService.Name)
		}
	}
	if !session.IsReady() {
		return nil, fmt.Errorf("create session %s failed", session.Id)
	}
	m.sessionsMutex.Lock()
	m.sessions[session.Id] = session
	m.sessionsMutex.Unlock()
	return session, nil
}

func (m *SessionManager) CloseSession(xl xlog.Logger, sessionId string) error {
	session, ok := m.GetSession(xl, sessionId)
	if !ok {
		xl.Errorf("session %s not found", sessionId)
		return fmt.Errorf("session %s not found", sessionId)
	}
	// 先删除session，再关闭session, 避免在关闭session时，session被其他协程访问
	m.sessionsMutex.Lock()
	delete(m.sessions, session.Id)
	m.sessionsMutex.Unlock()

	session.Close()
	return nil
}

func (m *SessionManager) existsSession(sessionId string) bool {
	m.sessionsMutex.RLock()
	defer m.sessionsMutex.RUnlock()
	_, ok := m.sessions[sessionId]
	return ok
}

// GetAllSessions returns all sessions in the workspace.
func (m *SessionManager) GetAllSessions(_ xlog.Logger) []*Session {
	m.sessionsMutex.RLock()
	defer m.sessionsMutex.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}
