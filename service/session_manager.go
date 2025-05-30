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
}

func NewSessionManager() *SessionManager {
	return &SessionManager{}
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
func (m *SessionManager) CreateSession(xl xlog.Logger) *Session {
	session := NewSession(uuid.New().String())
	m.sessionsMutex.Lock()
	m.sessions[session.Id] = session
	m.sessionsMutex.Unlock()
	return session
}

func (m *SessionManager) CloseSession(xl xlog.Logger, sessionId string) error {
	session, ok := m.GetSession(xl, sessionId)
	if !ok {
		xl.Errorf("session %s not found", sessionId)
		return fmt.Errorf("session %s not found", sessionId)
	}

	session.Close()
	m.sessionsMutex.Lock()
	delete(m.sessions, session.Id)
	m.sessionsMutex.Unlock()
	return nil
}
