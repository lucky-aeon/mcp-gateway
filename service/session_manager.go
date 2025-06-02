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

	mcpServices := m.curWorkspace.getMcpServices()
	for _, mcpService := range mcpServices {
		session.SubscribeSSE(mcpService.Name, mcpService.GetSSEUrl())
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
