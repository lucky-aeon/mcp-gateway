package service

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

type (
	CmdStatus string
)

const (
	Starting CmdStatus = "starting"
	Running  CmdStatus = "Running"
	Stopping CmdStatus = "Stopping"
	Stopped  CmdStatus = "Stopped"
)

type ExportMcpService interface {
	GetUrl() string
	GetSSEUrl() string
	GetMessageUrl() string
	GetStatus() CmdStatus
	SendMessage(message string) error
	Info() McpServiceInfo
}

// McpService 表示一个运行中的服务实例
type McpService struct {
	Name    string
	Config  config.MCPServerConfig
	Cmd     *exec.Cmd
	LogFile *os.File
	logger  xlog.Logger // 用于记录CMD输出
	Port    int         // 添加端口字段

	portMgr PortManagerI
	cfg     config.Config

	// 状态
	Status CmdStatus

	// 重试次数
	RetryCount int
	RetryMax   int
}

// NewMcpService 创建一个McpService实例
func NewMcpService(name string, config config.MCPServerConfig, portMgr PortManagerI, cfg config.Config) *McpService {
	logger := xlog.NewLogger(fmt.Sprintf("[MCP-%s]", name))
	return &McpService{
		Name:     name,
		Config:   config,
		Port:     0,
		portMgr:  portMgr,
		cfg:      cfg,
		Status:   Stopped,
		logger:   logger,
		RetryMax: cfg.McpServiceMgrConfig.GetMcpServiceRetryCount(),
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
	if s.Status != Running {
		return
	}
	logger.Infof("Killing process %s", s.Name)
	s.Status = Stopping
	defer func() {
		if s.Status == Stopping {
			s.Status = Stopped
			if s.Cmd.Process != nil {
				err2 := syscall.Kill(-s.Cmd.Process.Pid, syscall.SIGKILL) // 注意负号
				if err2 != nil {
					logger.Errorf("Failed to kill process %s: %s", s.Name, err2)
				}
				s.Cmd = nil
			}
		}
	}()
	if s.Cmd == nil {
		return
	}
	if s.LogFile != nil {
		err = s.LogFile.Close()
		if err != nil {
			logger.Errorf("Failed to close log file: %v", err)
			return
		}
	}
	if s.Cmd != nil {
		logger.Infof("Killing cmd %s", s.Cmd)
		err = s.Cmd.Process.Kill()
		if err != nil {
			logger.Errorf("kill cmd %s failed, err: %v", s.Name, err)
			return
		}
	}
	return
}

// Start 启动服务
func (s *McpService) Start(logger xlog.Logger) error {
	if s.IsSSE() {
		return fmt.Errorf("服务 %s 不是命令类型, 无需启动", s.Name)
	}
	if s.Status == Running {
		return fmt.Errorf("服务 %s 已运行", s.Name)
	}
	s.Status = Starting
	if s.Port == 0 {
		s.Port = s.portMgr.getNextAvailablePort()
	}
	logger.Infof("Assigned port: %d", s.Port)
	// 创建日志文件
	logFile, err := xlog.CreateLogFile(s.cfg.ConfigDirPath, s.Name+".log")
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}

	logger.Infof("Created log file: %s", logFile.Name())

	// 设置日志文件
	s.LogFile = logFile

	// 准备命令
	mcpRunner := fmt.Sprintf("\"%s %s\"", s.Config.Command, strings.Join(s.Config.Args, " "))
	cmd := exec.Command("/bin/sh", "-c", fmt.Sprintf("%s --stdio %s --port %d", config.COMMAND_SUPERGATEWA, mcpRunner, s.Port))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // 创建新会话，脱离终端控制
	}
	cmd.Stdout = s
	cmd.Stderr = s

	// 设置环境变量
	if len(s.Config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range s.Config.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	logger.Infof("Command environment: %v", cmd.Env)

	// 启动进程
	if err := cmd.Start(); err != nil {
		logger.Warnf("close logfile: %v", logFile.Close())
		return fmt.Errorf("failed to start command: %v", err)
	}

	s.Cmd = cmd
	s.Status = Running
	s.RetryCount = s.RetryMax
	go s.onMonitoring(logger)
	return nil
}

func (s *McpService) onMonitoring(xl xlog.Logger) {
	xl.Infof("Monitoring %s", s.Name)
	defer func() {
		if err := recover(); err != nil {
			xl.Errorf("recovered from panic: %v", err)
			if s.GetStatus() == Running {
				s.Stop(xl)
			}

		}
	}()
	err := s.Cmd.Wait()
	xl.Infof("quit Monitoring cmd %s", s.Name)
	if s.GetStatus() == Running && s.RetryCount > 0 {
		s.Restart(xl)
		return
	}
	if err != nil {
		xl.Errorf("cmd %s failed", s.Name)
	}
	s.Stop(xl)
}

// Restart 重启服务
func (s *McpService) Restart(logger xlog.Logger) {
	if s.IsSSE() {
		return
	}
	if s.RetryCount < 0 {
		logger.Warnf("no retry restart count for %s", s.Name)
		return
	} else {
		s.RetryCount--
	}
	if s.GetStatus() != Running {
		return
	}
	err := s.Stop(logger)
	if err != nil {
		logger.Errorf("stop cmd %s failed", s.Name)
	}
	err = s.Start(logger)
	if err != nil {
		logger.Errorf("Failed to restart %s: %v", s.Name, err)
		s.Restart(logger)
	}
}

// setConfig 设置配置, 下次启动时生效
func (s *McpService) setConfig(cfg config.MCPServerConfig) {
	s.Config = cfg
}

// io.Writer
func (s *McpService) Write(p []byte) (n int, err error) {
	if s.IsSSE() {
		return
	}
	lineStr := string(p)

	if s.Status == Starting && strings.HasPrefix(lineStr, "SSE endpoint:") {
		s.Status = Running
	}

	if s.LogFile != nil {
		s.LogFile.Write(p)
	}

	// find exited
	if strings.Contains(lineStr, "exited") {
		s.Restart(s.logger)
	}

	s.logger.Info(string(p))

	return len(p), nil
}

func (s *McpService) GetUrl() string {
	if s.GetStatus() != Running {
		return ""
	}
	if s.Config.URL != "" {
		return s.Config.URL
	}
	if s.Port == 0 {
		return ""
	}
	return fmt.Sprintf("http://localhost:%d", s.Port)
}

// SSE
func (s *McpService) GetSSEUrl() string {
	if s.GetStatus() != Running {
		return ""
	}
	return fmt.Sprintf("%s/sse", s.GetUrl())
}

// Message
func (s *McpService) GetMessageUrl() string {
	if s.GetStatus() != Running {
		return ""
	}
	return fmt.Sprintf("%s/message", s.GetUrl())
}

func (s *McpService) GetPort() int {
	return s.Port
}

func (s *McpService) GetStatus() CmdStatus {
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
	Name   string
	Status CmdStatus
	Config config.MCPServerConfig
}

func (s *McpService) Info() McpServiceInfo {
	return McpServiceInfo{
		Name:   s.Name,
		Status: s.Status,
		Config: s.Config,
	}
}
