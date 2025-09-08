package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lucky-aeon/agentx/plugin-helper/bridge"
	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
	"github.com/mark3labs/mcp-go/client/transport"
)

type (
	CmdStatus string
)

const (
	Starting CmdStatus = "starting"
	Running  CmdStatus = "Running"
	Stopping CmdStatus = "Stopping"
	Stopped  CmdStatus = "Stopped"
	Failed   CmdStatus = "Failed"
)

type ExportMcpService interface {
	GetUrl() string
	GetSSEUrl() string
	GetMessageUrl() string
	GetStatus() CmdStatus
	SendMessage(message string) error
	Info() McpServiceInfo
	GetHealthStatus() map[string]interface{}
}

// McpService 表示一个运行中的服务实例
type McpService struct {
	Name    string
	Config  config.MCPServerConfig
	LogFile *os.File
	logger  xlog.Logger // 用于记录CMD输出
	Port    int         // 添加端口字段

	portMgr PortManagerI

	// 状态
	Status CmdStatus

	// 重试次数
	RetryCount int
	RetryMax   int

	// stdio-sse bridge
	bridge *bridge.StdioToSSEBridge

	// 状态详情
	LastError      string    // 最后一次错误信息
	FailureReason  string    // 失败原因
	DeployedAt     time.Time // 部署时间
	LastStartedAt  time.Time // 最后启动时间
	LastStoppedAt  time.Time // 最后停止时间
	HealthCheckURL string    // 健康检查URL

	mutex sync.RWMutex
}

// NewMcpService 创建一个McpService实例
func NewMcpService(name string, cfg config.MCPServerConfig, portMgr PortManagerI) *McpService {
	logger := xlog.NewLogger(fmt.Sprintf("[MCP-%s]", name))
	return &McpService{
		Name:       name,
		Config:     cfg,
		Port:       0,
		portMgr:    portMgr,
		Status:     Stopped,
		logger:     logger,
		RetryMax:   cfg.McpServiceMgrConfig.GetMcpServiceRetryCount(),
		DeployedAt: time.Now(),
	}
}

// IsSSE 判断是否是SSE类型
func (s *McpService) IsSSE() bool {
	if s.Config.Command == "" && s.Config.URL != "" {
		s.Status = Running
		return true
	}
	return false
}

// Stop 停止服务
func (s *McpService) Stop(logger xlog.Logger) (err error) {
	if s.IsSSE() {
		return
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Status != Running && s.Status != Starting {
		return
	}

	logger.Infof("Stopping service %s", s.Name)
	s.Status = Stopping
	s.LastStoppedAt = time.Now()
	defer func() {
		if s.Status == Stopping {
			s.Status = Stopped
		}
		s.bridge = nil
	}()

	// 停止stdio-sse桥接
	if s.bridge != nil {
		if err := s.bridge.Close(); err != nil {
			logger.Errorf("Failed to stop stdio-sse bridge: %v", err)
		}
	}

	// 关闭日志文件
	if s.LogFile != nil {
		err = s.LogFile.Close()
		if err != nil {
			logger.Errorf("Failed to close log file: %v", err)
		}
		s.LogFile = nil
	}
	return
}

// Start 启动服务
func (s *McpService) Start(logger xlog.Logger) error {
	if s.IsSSE() {
		logger.Infof("服务 %s 是 SSE 类型，无需启动进程", s.Name)
		return nil
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Status == Running {
		return fmt.Errorf("服务 %s 已运行", s.Name)
	}
	if s.Status == Failed {
		return fmt.Errorf("服务 %s 已失败，无法启动", s.Name)
	}

	s.Status = Starting
	s.LastStartedAt = time.Now()
	s.LastError = ""
	s.FailureReason = ""

	if s.Port == 0 {
		s.Port = s.portMgr.GetNextAvailablePort()
	}
	logger.Infof("Assigned port: %d", s.Port)

	// 创建日志文件
	logFile, err := xlog.CreateLogFile(s.Config.LogConfig.Path, s.Name+".log")
	if err != nil {
		s.LastError = fmt.Sprintf("failed to create log file: %v", err)
		s.FailureReason = "Log file creation failed"
		s.Status = Failed
		return fmt.Errorf("failed to create log file: %v", err)
	}
	logger.Infof("Created log file: %s", logFile.Name())
	s.LogFile = logFile

	// 使用stdio-sse桥接代替supergateway
	logger.Infof("Creating stdio-sse bridge for command: %s %s", s.Config.Command, strings.Join(s.Config.Args, " "))

	// 创建stdio-sse桥接
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	bridgeInstance, err := bridge.NewStdioToSSEBridge(ctx, transport.NewStdio(s.Config.Command, s.Config.GetEnvs(), s.Config.Args...), s.Name)
	if err != nil {
		logger.Warnf("close logfile: %v", logFile.Close())
		s.LastError = fmt.Sprintf("failed to create stdio-sse bridge: %v", err)
		s.FailureReason = "Bridge creation failed"
		s.Status = Failed
		return fmt.Errorf("failed to create stdio-sse bridge: %w", err)
	}

	s.bridge = bridgeInstance

	// 使用通道来同步服务器启动状态
	startupChan := make(chan error, 1)

	// 在goroutine中启动bridge服务器（会阻塞运行）
	go func() {
		defer close(startupChan)
		logger.Infof("Starting bridge server on port %d", s.Port)

		// 启动服务器，这里会阻塞
		if err := bridgeInstance.Start(fmt.Sprintf("0.0.0.0:%d", s.Port)); err != nil {
			logger.Errorf("Bridge server failed: %v", err)
			startupChan <- err
			return
		}
	}()

	// 等待服务器启动结果，最多等待3秒
	startupTimeout := time.NewTimer(3 * time.Second)
	defer startupTimeout.Stop()

	select {
	case err := <-startupChan:
		if err != nil {
			logger.Warnf("close logfile: %v", logFile.Close())
			s.LastError = err.Error()
			s.FailureReason = "Bridge server startup failed"
			s.Status = Failed
			return fmt.Errorf("bridge server startup failed: %w", err)
		}
		// 这里不应该到达，因为Start()成功时会一直阻塞
		break
	case <-startupTimeout.C:
		// 超时意味着服务器可能正在正常运行（因为Start()会阻塞）
		// 我们可以通过健康检查来验证
		logger.Infof("Bridge server startup timeout - checking if server is running")

		// 简单检查：尝试ping bridge
		if err := bridgeInstance.Ping(context.Background()); err != nil {
			logger.Warnf("close logfile: %v", logFile.Close())
			s.LastError = fmt.Sprintf("Bridge health check failed: %v", err)
			s.FailureReason = "Bridge server not responding"
			s.Status = Failed
			return fmt.Errorf("bridge server not responding: %w", err)
		}

		logger.Infof("Bridge server is running and responding to ping")
		break
	}

	s.Status = Running
	s.RetryCount = s.RetryMax
	s.HealthCheckURL = fmt.Sprintf("http://0.0.0.0:%d/health", s.Port)

	logger.Infof("Started stdio-sse bridge for service %s on port %d", s.Name, s.Port)

	// 监控桥接状态
	return nil
}

// Restart 重启服务
func (s *McpService) Restart(logger xlog.Logger) {
	if s.IsSSE() {
		logger.Infof("服务 %s 是 SSE 类型，无需重启进程", s.Name)
		return
	}

	// 检查重试次数，避免在锁内调用自身
	s.mutex.Lock()
	if s.RetryCount <= 0 {
		logger.Warnf("No retry restart count left for %s, marking as failed", s.Name)
		s.Status = Failed
		s.FailureReason = "Max retry count reached"
		s.LastError = "Service failed after maximum retry attempts"
		s.mutex.Unlock()
		return
	}

	s.RetryCount--
	currentAttempt := s.RetryMax - s.RetryCount
	retryCount := s.RetryCount
	logger.Infof("Restarting %s (attempt %d/%d)", s.Name, currentAttempt, s.RetryMax)
	s.mutex.Unlock()

	if err := s.Stop(logger); err != nil {
		logger.Errorf("Failed to stop service %s during restart: %v", s.Name, err)
	}

	// 在锁外调用Start
	err := s.Start(logger)
	if err != nil {
		logger.Errorf("Failed to restart %s: %v", s.Name, err)

		s.mutex.Lock()
		s.LastError = fmt.Sprintf("Failed to restart: %v", err)
		if retryCount > 0 {
			s.FailureReason = fmt.Sprintf("Restart attempt %d/%d failed", currentAttempt, s.RetryMax)
			s.mutex.Unlock()
			// 在锁外延时重启，避免死锁
			time.AfterFunc(5*time.Second, func() {
				s.Restart(logger)
			})
		} else {
			s.Status = Failed
			s.FailureReason = "All restart attempts failed"
			s.mutex.Unlock()
		}
	}
}

// setConfig 设置配置, 下次启动时生效
func (s *McpService) setConfig(cfg config.MCPServerConfig) error {
	if s.Status != Stopped {
		return fmt.Errorf("service %s is running, cannot set config", s.Name)
	}
	s.Config = cfg
	return nil
}

func (s *McpService) GetUrl() string {
	if s.GetStatus() != Running {
		return ""
	}
	if s.Config.URL != "" {
		return s.Config.URL
	}
	if s.bridge != nil {
		return "http://127.0.0.1:" + strconv.Itoa(s.Port)
	}

	return ""
}

// SSE
func (s *McpService) GetSSEUrl() string {
	if s.GetStatus() != Running {
		return ""
	}
	sseUrl, _ := s.bridge.CompleteSseEndpoint()
	return s.GetUrl() + sseUrl
}

// Message
func (s *McpService) GetMessageUrl() string {
	if s.GetStatus() != Running {
		return ""
	}
	mesUrl, _ := s.bridge.CompleteMessageEndpoint()
	return s.GetUrl() + mesUrl
}

func (s *McpService) GetPort() int {
	return s.Port
}

func (s *McpService) GetStatus() CmdStatus {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.Status
}

func (s *McpService) SendMessage(message string) error {
	// 发送消息到 MCP 服务
	resp, err := http.Post(s.GetMessageUrl(), "application/json", strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message, status code: %d", resp.StatusCode)
	}

	return nil
}

type McpServiceInfo struct {
	Name          string                 `json:"name"`
	Status        CmdStatus              `json:"status"`
	Config        config.MCPServerConfig `json:"config"`
	Port          int                    `json:"port"`
	LastError     string                 `json:"last_error,omitempty"`
	FailureReason string                 `json:"failure_reason,omitempty"`
	DeployedAt    time.Time              `json:"deployed_at"`
	LastStartedAt time.Time              `json:"last_started_at,omitempty"`
	LastStoppedAt time.Time              `json:"last_stopped_at,omitempty"`
	RetryCount    int                    `json:"retry_count"`
	RetryMax      int                    `json:"retry_max"`
	URLs          ServiceURLs            `json:"urls"`
}

type ServiceURLs struct {
	BaseURL    string `json:"base_url,omitempty"`
	SSEUrl     string `json:"sse_url,omitempty"`
	MessageUrl string `json:"message_url,omitempty"`
}

func (s *McpService) Info() McpServiceInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return McpServiceInfo{
		Name:          s.Name,
		Status:        s.Status,
		Config:        s.Config,
		Port:          s.Port,
		LastError:     s.LastError,
		FailureReason: s.FailureReason,
		DeployedAt:    s.DeployedAt,
		LastStartedAt: s.LastStartedAt,
		LastStoppedAt: s.LastStoppedAt,
		RetryCount:    s.RetryCount,
		RetryMax:      s.RetryMax,
		URLs: ServiceURLs{
			BaseURL:    s.GetUrl(),
			SSEUrl:     s.GetSSEUrl(),
			MessageUrl: s.GetMessageUrl(),
		},
	}
}

// GetHealthStatus returns detailed health information for the service
func (s *McpService) GetHealthStatus() map[string]interface{} {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	health := map[string]interface{}{
		"name":            s.Name,
		"status":          s.Status,
		"healthy":         s.Status == Running,
		"port":            s.Port,
		"deployed_at":     s.DeployedAt,
		"last_started_at": s.LastStartedAt,
		"last_stopped_at": s.LastStoppedAt,
		"retry_count":     s.RetryCount,
		"retry_max":       s.RetryMax,
	}

	if s.LastError != "" {
		health["last_error"] = s.LastError
	}

	if s.FailureReason != "" {
		health["failure_reason"] = s.FailureReason
	}

	if s.HealthCheckURL != "" {
		health["health_check_url"] = s.HealthCheckURL
	}

	// Calculate uptime if service is running
	if s.Status == Running && !s.LastStartedAt.IsZero() {
		health["uptime_seconds"] = time.Since(s.LastStartedAt).Seconds()
	}

	return health
}
