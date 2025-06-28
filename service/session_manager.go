package service

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

type SessionManager struct {
	// sessions
	sessions      map[string]*Session
	sessionsMutex sync.RWMutex
	curWorkspace  *WorkSpace
}

func NewSessionManager(curWorkspace *WorkSpace) *SessionManager {
	return &SessionManager{curWorkspace: curWorkspace, sessions: make(map[string]*Session)}
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

// CreateSession creates a new session.
func (m *SessionManager) CreateSession(xl xlog.Logger) (*Session, error) {
	session := NewSession(uuid.New().String())
	if m.existsSession(session.Id) {
		xl.Errorf("session %s already exists", session.Id)
		return nil, fmt.Errorf("session %s already exists", session.Id)
	}

	// 设置清理回调
	session.SetCleanupCallback(func(sessionId string) {
		xl.Infof("Auto-cleaning inactive session: %s", sessionId)
		m.CloseSession(xl, sessionId)
	})

	mcpServices := m.curWorkspace.getMcpServices()
	for _, mcpService := range mcpServices {
		if mcpService.GetStatus() != Running {
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

// GetAllSessions returns all sessions in the workspace
func (m *SessionManager) GetAllSessions(_ xlog.Logger) []*Session {
	m.sessionsMutex.RLock()
	defer m.sessionsMutex.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}
