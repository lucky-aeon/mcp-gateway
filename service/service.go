package service

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

type ExportMcpService interface {
	GetUrl() string
	GetSSEUrl() string
	GetMessageUrl() string
	GetStatus() string
	SendMessage(message string) error
	Info() McpServiceInfo
}

// McpService 表示一个运行中的服务实例
type McpService struct {
	Name       string
	Config     config.MCPServerConfig
	Cmd        *exec.Cmd
	LogFile    *os.File
	logger     xlog.Logger // 用于记录CMD输出
	StopSignal chan struct{}
	Port       int // 添加端口字段

	portMgr PortManagerI
	cfg     config.Config

	// 状态
	Status string

	// 重试次数
	RetryCount int
}

// NewMcpService 创建一个McpService实例
func NewMcpService(name string, config config.MCPServerConfig, portMgr PortManagerI, cfg config.Config) *McpService {
	logger := xlog.NewLogger(fmt.Sprintf("[Service-%s]", name))
	return &McpService{
		Name:       name,
		Config:     config,
		StopSignal: nil,
		Port:       0,
		portMgr:    portMgr,
		cfg:        cfg,
		Status:     "stopped",
		logger:     logger,
		RetryCount: cfg.McpServiceMgrConfig.GetMcpServiceRetryCount(),
	}
}

// IsSSE 判断是否是SSE类型
func (s *McpService) IsSSE() bool {
	if s.Config.Command == "" && s.Config.URL != "" {
		s.Status = "running"
		return true
	}
	return false
}

// Stop 停止服务
func (s *McpService) Stop(logger xlog.Logger) {
	if s.IsSSE() {
		return
	}
	if s.Status != "running" {
		return
	}
	logger.Infof("Killing process %s", s.Name)
	if s.Cmd == nil {
		return
	}
	if s.StopSignal != nil {
		close(s.StopSignal)
		s.StopSignal = nil
	}
	if s.LogFile != nil {
		s.LogFile.Close()
	}
	if s.Cmd != nil {
		s.Cmd.Process.Kill()
		s.Cmd = nil
	}

	if s.Port != 0 {
		s.portMgr.releasePort(s.Port)
		s.Port = 0
	}
	s.Status = "stopped"
}

// Start 启动服务
func (s *McpService) Start(logger xlog.Logger) error {
	if s.IsSSE() {
		return fmt.Errorf("服务 %s 不是命令类型, 无需启动", s.Name)
	}
	if s.Status == "running" {
		return fmt.Errorf("服务 %s 已运行", s.Name)
	}
	s.Status = "starting"
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
		logFile.Close()
		return fmt.Errorf("failed to start command: %v", err)
	}

	s.Cmd = cmd
	s.StopSignal = make(chan struct{})

	// 启动监控
	go func() {
		s.monitorProcess()
	}()

	s.Status = "running"
	return nil
}

// Restart 重启服务
func (s *McpService) Restart(logger xlog.Logger) {
	if s.IsSSE() {
		return
	}
	s.Stop(logger)
	s.Start(logger)
}

// setConfig 设置配置, 下次启动时生效
func (s *McpService) setConfig(cfg config.MCPServerConfig) {
	s.Config = cfg
}

// monitorProcess 监控进程
func (s *McpService) monitorProcess() {
	if s.IsSSE() {
		return
	}
	s.logger.Infof("Monitoring process %s", s.Name)
	for {
		select {
		case <-s.StopSignal:
			s.logger.Infof("Process %s stopped", s.Name)
			return
		default:
			if err := s.Cmd.Wait(); err != nil {
				s.logger.Infof("Process %s exited with error: %v, restarting...", s.Name, err)
				if s.RetryCount > s.cfg.McpServiceMgrConfig.GetMcpServiceRetryCount() {
					s.logger.Infof("Process %s exited with error: %v, retry count exceeded, giving up", s.Name, err)
					s.Stop(s.logger)
					return
				}
				s.RetryCount++
				s.Stop(s.logger)
				s.Start(s.logger)
			}
		}
	}

}

// io.Writer
func (s *McpService) Write(p []byte) (n int, err error) {
	if s.IsSSE() {
		return
	}
	if s.LogFile != nil {
		s.LogFile.Write(p)
	}

	// find exited
	if strings.Contains(string(p), "exited") {
		s.Stop(s.logger)
	}

	s.logger.Info(string(p))

	return len(p), nil
}

func (s *McpService) GetUrl() string {
	if s.GetStatus() != "running" {
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
	if s.GetStatus() != "running" {
		return ""
	}
	return fmt.Sprintf("%s/sse", s.GetUrl())
}

// Message
func (s *McpService) GetMessageUrl() string {
	if s.GetStatus() != "running" {
		return ""
	}
	return fmt.Sprintf("%s/message", s.GetUrl())
}

func (s *McpService) GetPort() int {
	return s.Port
}

func (s *McpService) GetStatus() string {
	return s.Status
}

func (s *McpService) SendMessage(message string) error {
	// 发送消息到 MCP 服务
	resp, err := http.Post(s.GetMessageUrl(), "application/json", strings.NewReader(message))
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message, status code: %d", resp.StatusCode)
	}

	return nil
}

type McpServiceInfo struct {
	Name   string
	Status string
	Config config.MCPServerConfig
}

func (s *McpService) Info() McpServiceInfo {
	return McpServiceInfo{
		Name:   s.Name,
		Status: s.Status,
		Config: s.Config,
	}
}
