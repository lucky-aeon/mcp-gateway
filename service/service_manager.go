package service

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

type ServiceManagerI interface {
	DeployServer(logger xlog.Logger, name string, config config.MCPServerConfig) error
	StopServer(logger xlog.Logger, name string)
	ListServerConfig(logger xlog.Logger) map[string]config.MCPServerConfig
	GetMcpService(logger xlog.Logger, name string) (ExportMcpService, error)
	GetMcpServices(logger xlog.Logger) map[string]ExportMcpService
	CreateProxySession(logger xlog.Logger) *Session
	GetProxySession(logger xlog.Logger, id string) (*Session, bool)
	CloseProxySession(logger xlog.Logger, id string)
	DeleteServer(logger xlog.Logger, name string) error
	Close()
}

type PortManagerI interface {
	getNextAvailablePort() int
	releasePort(port int)
}

// ServiceManager 管理所有运行的服务
type ServiceManager struct {
	sync.RWMutex
	servers   map[string]*McpService
	usedPorts map[int]bool // 记录已使用的端口
	nextPort  int          // 下一个可用端口
	portMutex sync.Mutex   // 端口分配的互斥锁

	// all session-> mcp service
	sessions map[string]*McpService

	// proxy sessions
	proxySessionsMutex sync.RWMutex
	proxySessions      map[string]*Session

	cfg config.Config
}

func NewServiceManager(cfg config.Config) *ServiceManager {
	if cfg.SessionGCInterval == 0 {
		cfg.SessionGCInterval = 5 * time.Minute
	}
	mgr := &ServiceManager{
		cfg:           cfg,
		servers:       make(map[string]*McpService),
		usedPorts:     make(map[int]bool),
		nextPort:      10000,
		sessions:      make(map[string]*McpService),
		proxySessions: make(map[string]*Session),
	}
	go func() {
		mgr.loopGC()
	}()
	return mgr
}

func (m *ServiceManager) DeleteServer(logger xlog.Logger, name string) error {
	m.Lock()
	defer m.Unlock()
	if mcpService, exists := m.servers[name]; exists {
		mcpService.Stop(logger)
		delete(m.servers, name)
	} else {
		return fmt.Errorf("服务 %s 不存在", name)
	}
	m.saveConfig()
	return nil
}

func (m *ServiceManager) DeployServer(logger xlog.Logger, name string, mcpCfg config.MCPServerConfig) error {
	m.Lock()
	defer m.Unlock()

	if mcpService, exists := m.servers[name]; exists {
		logger.Infof("服务 %s 已存在, 重新配置: %v", name, mcpCfg)
		mcpService.setConfig(mcpCfg)
		// 重启服务
		mcpService.Restart(logger)
		return nil
	}

	// 创建服务实例
	instance := NewMcpService(name, mcpCfg, m)
	if err := instance.Start(logger); err != nil {
		logger.Errorf("Failed to start service %s: %v", name, err)
		return err
	}
	m.servers[name] = instance
	m.saveConfig()
	return nil
}

func (m *ServiceManager) ListServerConfig(logger xlog.Logger) map[string]config.MCPServerConfig {
	m.RLock()
	defer m.RUnlock()
	config := make(map[string]config.MCPServerConfig)
	for name, instance := range m.servers {
		config[name] = instance.Config
	}
	return config
}

func (m *ServiceManager) GetMcpService(logger xlog.Logger, name string) (ExportMcpService, error) {
	instance, err := m.getMcpService(name)
	if err != nil {
		logger.Errorf("获取服务 %s 失败: %v", name, err)
		return nil, err
	}
	return instance, nil
}

func (m *ServiceManager) getMcpService(name string) (*McpService, error) {
	m.RLock()
	defer m.RUnlock()
	if instance, exists := m.servers[name]; exists {
		return instance, nil
	}
	return nil, fmt.Errorf("服务 %s 不存在", name)
}

func (m *ServiceManager) StopServer(logger xlog.Logger, name string) {
	mcp, err := m.getMcpService(name)
	if err != nil {
		logger.Errorf("获取服务 %s 失败: %v", name, err)
		return
	}
	mcp.Stop(logger)
}

func (m *ServiceManager) saveConfig() error {
	config := make(map[string]config.MCPServerConfig)
	for name, instance := range m.servers {
		config[name] = instance.Config
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.cfg.GetMcpConfigPath(), data, 0644)
}

// getNextAvailablePort 获取下一个可用端口
func (m *ServiceManager) getNextAvailablePort() int {
	m.portMutex.Lock()
	defer m.portMutex.Unlock()
	for m.usedPorts[m.nextPort] {
		m.nextPort++
	}
	port := m.nextPort
	m.usedPorts[port] = true
	m.nextPort++
	return port
}

// releasePort 释放端口
func (m *ServiceManager) releasePort(port int) {
	m.portMutex.Lock()
	delete(m.usedPorts, port)
	m.portMutex.Unlock()
}

func (m *ServiceManager) GetMcpServices(logger xlog.Logger) map[string]ExportMcpService {
	m.RLock()
	defer m.RUnlock()
	exportServices := make(map[string]ExportMcpService)
	for name, instance := range m.servers {
		exportServices[name] = instance
	}
	return exportServices
}

// CreateProxySession 创建一个新的代理会话
func (m *ServiceManager) CreateProxySession(xl xlog.Logger) *Session {
	xl.Infof("Creating new proxy session")
	xl.Infof("Creating new session")
	session := NewSession(uuid.New().String())
	xl.Infof("Subscribing to all MCP services")
	// 订阅所有MCP服务的SSE事件
	m.RLock()

	for name, instance := range m.servers {
		xl.Infof("Subscribing to MCP service: %s", name)

		maxRetries := 2
		retryDelay := time.Second

		for i := 0; i <= maxRetries; i++ {
			if instance.GetStatus() == Running {
				session.SubscribeSSE(name, instance.GetSSEUrl())
				break
			}

			if i < maxRetries {
				xl.Infof("Service[%s] %s not running, retrying (%d/%d)...", instance.GetStatus(), name, i+1, maxRetries)
				time.Sleep(retryDelay)
			} else {
				xl.Warnf("Service %s still not running after %d retries, skipping", name, maxRetries)
			}
		}
	}
	m.RUnlock()

	xl.Infof("Proxy session created: %s", session.Id)
	m.proxySessionsMutex.Lock()
	defer m.proxySessionsMutex.Unlock()
	m.proxySessions[session.Id] = session
	return session
}

// CloseProxySession 关闭代理会话
func (m *ServiceManager) CloseProxySession(xl xlog.Logger, id string) {
	xl.Infof("Closing proxy session: %s", id)
	xl.Infof("Closing proxy session, has mutex: %s", id)
	if session, exists := m.proxySessions[id]; exists {
		session.Close()
		xl.Infof("Closed proxy session: %s", id)
		m.proxySessionsMutex.Lock()
		defer m.proxySessionsMutex.Unlock()
		delete(m.proxySessions, id)
	}
}

// GetProxySession 获取代理会话
func (m *ServiceManager) GetProxySession(logger xlog.Logger, id string) (*Session, bool) {
	m.proxySessionsMutex.RLock()
	defer m.proxySessionsMutex.RUnlock()

	session, exists := m.proxySessions[id]
	if !exists {
		return nil, false
	}
	return session, exists
}

// GC长时间未使用的Session
func (m *ServiceManager) loopGC() {
	tick := time.NewTicker(m.cfg.SessionGCInterval)
	defer tick.Stop()
	xl := xlog.NewLogger("[ServiceManager-GC]")

	for range tick.C {
		// GC proxy sessions
		func() {
			now := time.Now()
			xl.Infof("GC proxy sessions, last receive time: %s. timeout: %s", now, m.cfg.ProxySessionTimeout)
			for id, session := range m.proxySessions {
				if session == nil {
					m.proxySessionsMutex.Lock()
					defer m.proxySessionsMutex.Unlock()
					delete(m.proxySessions, id)
					continue
				}
				if now.Sub(session.LastReceiveTime) > m.cfg.ProxySessionTimeout {
					xl.Infof("Closing proxy session: %s, last receive time: %s. timeout: %s", id, session.LastReceiveTime, m.cfg.ProxySessionTimeout)
					session.Close()
					m.proxySessionsMutex.Lock()
					defer m.proxySessionsMutex.Unlock()
					delete(m.proxySessions, id)
					xl.Infof("Closed proxy session: %s", id)
				}
			}
		}()
	}
}

func (m *ServiceManager) Close() {
	xl := xlog.NewLogger("[ServiceManager]")
	m.RLock()
	defer m.RUnlock()
	m.proxySessionsMutex.Lock()
	defer m.proxySessionsMutex.Unlock()

	xl.Infof("Closing all proxy sessions...")
	for id, session := range m.proxySessions {
		if session != nil {
			session.Close()
		}
		delete(m.proxySessions, id)
	}

	xl.Infof("Closing all MCP services...")
	for name, instance := range m.servers {
		instance.Stop(xl)
		delete(m.servers, name)
	}

	xl.Infof("ServiceManager closed")
}
