package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/lucky-aeon/agentx/plugin-helper/bridge"
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
	Failed   CmdStatus = "Failed"
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

	// 状态
	Status CmdStatus

	// 重试次数
	RetryCount int
	RetryMax   int

	// 进程管理
	PID           int
	ProcessGroup  int
	mutex         sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
	stdoutReader  *bufio.Reader
	stderrReader  *bufio.Reader
	lastHeartbeat time.Time

	// stdio-sse bridge
	bridge bridge.Bridge
}

// NewMcpService 创建一个McpService实例
func NewMcpService(name string, cfg config.MCPServerConfig, portMgr PortManagerI) *McpService {
	logger := xlog.NewLogger(fmt.Sprintf("[MCP-%s]", name))
	ctx, cancel := context.WithCancel(context.Background())
	return &McpService{
		Name:          name,
		Config:        cfg,
		Port:          0,
		portMgr:       portMgr,
		Status:        Stopped,
		logger:        logger,
		RetryMax:      cfg.McpServiceMgrConfig.GetMcpServiceRetryCount(),
		ctx:           ctx,
		cancel:        cancel,
		lastHeartbeat: time.Now(),
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

	// 取消context
	if s.cancel != nil {
		s.cancel()
	}

	defer func() {
		if s.Status == Stopping {
			s.Status = Stopped
		}
		s.Cmd = nil
		s.PID = 0
		s.ProcessGroup = 0
		s.bridge = nil
	}()

	// 停止stdio-sse桥接
	if s.bridge != nil {
		if err := s.bridge.Stop(); err != nil {
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

	// 如果还有遗留的进程，强制停止
	if s.Cmd != nil {
		err = s.killProcessGroup(logger)
		if err != nil {
			logger.Errorf("Failed to kill process group for %s: %v", s.Name, err)
		}
	}

	return
}

// Start 启动服务
func (s *McpService) Start(logger xlog.Logger) error {
	if s.IsSSE() {
		return fmt.Errorf("服务 %s 不是命令类型, 无需启动", s.Name)
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
	if s.Port == 0 {
		s.Port = s.portMgr.GetNextAvailablePort()
	}
	logger.Infof("Assigned port: %d", s.Port)

	// 创建日志文件
	logFile, err := xlog.CreateLogFile(s.Config.LogConfig.Path, s.Name+".log")
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}
	logger.Infof("Created log file: %s", logFile.Name())
	s.LogFile = logFile

	// 使用stdio-sse桥接代替supergateway
	logger.Infof("Creating stdio-sse bridge for command: %s %s", s.Config.Command, strings.Join(s.Config.Args, " "))

	// 创建stdio-sse桥接
	bridge := bridge.NewStdioToSSE()

	// 启动桥接
	if err := bridge.Start(); err != nil {
		logger.Warnf("close logfile: %v", logFile.Close())
		return fmt.Errorf("failed to start stdio-sse bridge: %v", err)
	}

	s.bridge = bridge
	s.Status = Running
	s.RetryCount = s.RetryMax
	s.lastHeartbeat = time.Now()

	logger.Infof("Started stdio-sse bridge for service %s on port %d", s.Name, s.Port)

	// 监控桥接状态
	go s.monitorBridge(logger)
	return nil
}

func (s *McpService) onMonitoring(xl xlog.Logger) {
	xl.Infof("Monitoring %s (PID: %d)", s.Name, s.PID)
	defer func() {
		if err := recover(); err != nil {
			xl.Errorf("recovered from panic: %v", err)
			if s.GetStatus() == Running {
				s.Stop(xl)
			}
		}
	}()

	// 启动输出监控
	go s.monitorOutput(xl)
	go s.healthCheck(xl)

	err := s.Cmd.Wait()
	xl.Infof("Process %s (PID: %d) exited", s.Name, s.PID)

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.Status == Stopping || s.Status == Stopped {
		return
	}

	if s.RetryCount > 0 {
		xl.Infof("Process %s failed, retrying (%d retries left)", s.Name, s.RetryCount)
		s.Restart(xl)
		return
	}

	// 重试次数耗尽，标记为失败
	xl.Errorf("Process %s failed permanently after %d retries", s.Name, s.RetryMax)
	s.Status = Failed

	if err != nil {
		xl.Errorf("cmd %s failed with error: %v", s.Name, err)
	}
}

// Restart 重启服务
func (s *McpService) Restart(logger xlog.Logger) {
	if s.IsSSE() {
		return
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.RetryCount <= 0 {
		logger.Warnf("No retry restart count left for %s, marking as failed", s.Name)
		s.Status = Failed
		return
	}

	s.RetryCount--
	logger.Infof("Restarting %s (attempt %d/%d)", s.Name, s.RetryMax-s.RetryCount, s.RetryMax)

	// 停止当前桥接
	if s.bridge != nil {
		if err := s.bridge.Stop(); err != nil {
			logger.Errorf("Failed to stop bridge during restart: %v", err)
		}
	}

	// 等待一段时间再重启
	time.Sleep(2 * time.Second)

	// 重新创建context
	if s.cancel != nil {
		s.cancel()
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())

	// 使用新的Start方法重启
	err := s.Start(logger)
	if err != nil {
		logger.Errorf("Failed to restart %s: %v", s.Name, err)
		if s.RetryCount > 0 {
			go func() {
				time.Sleep(5 * time.Second)
				s.Restart(logger)
			}()
		} else {
			s.Status = Failed
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

// Write io.Writer interface - 保留了兼容性，但现在主要用管道监控
func (s *McpService) Write(p []byte) (n int, err error) {
	if s.IsSSE() {
		return len(p), nil
	}

	lineStr := strings.TrimSpace(string(p))
	if lineStr == "" {
		return len(p), nil
	}

	s.processOutput(lineStr, "legacy", s.logger)
	return len(p), nil
}

func (s *McpService) GetUrl() string {
	if s.GetStatus() != Running {
		return ""
	}
	if s.Config.URL != "" {
		return s.Config.URL
	}
	if s.bridge != nil {
		return s.bridge.GetURL()
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
	if s.bridge != nil {
		return s.bridge.GetSSEURL()
	}
	return fmt.Sprintf("%s/sse", s.GetUrl())
}

// Message
func (s *McpService) GetMessageUrl() string {
	if s.GetStatus() != Running {
		return ""
	}
	if s.bridge != nil {
		return s.bridge.GetMessageURL()
	}
	return fmt.Sprintf("%s/message", s.GetUrl())
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
	Name   string
	Status CmdStatus
	Config config.MCPServerConfig
}

func (s *McpService) Info() McpServiceInfo {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return McpServiceInfo{
		Name:   s.Name,
		Status: s.Status,
		Config: s.Config,
	}
}

// killProcessGroup 杀死整个进程组，包括所有子进程
func (s *McpService) killProcessGroup(logger xlog.Logger) error {
	if s.PID == 0 {
		return nil
	}

	if s.ProcessGroup > 0 {
		// 有进程组，杀死整个组
		logger.Infof("Killing process group %d for service %s", s.ProcessGroup, s.Name)

		// 先试图SIGTERM
		err := syscall.Kill(-s.ProcessGroup, syscall.SIGTERM)
		if err != nil {
			logger.Warnf("Failed to send SIGTERM to process group %d: %v", s.ProcessGroup, err)
		} else {
			// 等待一段时间让进程正常退出
			time.Sleep(3 * time.Second)
		}

		// 强制杀死SIGKILL
		err = syscall.Kill(-s.ProcessGroup, syscall.SIGKILL)
		if err != nil {
			logger.Errorf("Failed to send SIGKILL to process group %d: %v", s.ProcessGroup, err)
			// 如果进程组杀死失败，尝试只杀主进程
			return s.killSingleProcess(logger)
		}

		logger.Infof("Successfully killed process group %d", s.ProcessGroup)
		return nil
	} else {
		// 无进程组，只杀主进程
		return s.killSingleProcess(logger)
	}
}

// killSingleProcess 杀死单个进程
func (s *McpService) killSingleProcess(logger xlog.Logger) error {
	if s.PID == 0 {
		return nil
	}

	logger.Infof("Killing single process %d for service %s", s.PID, s.Name)

	// 先试图SIGTERM
	err := syscall.Kill(s.PID, syscall.SIGTERM)
	if err != nil {
		logger.Warnf("Failed to send SIGTERM to process %d: %v", s.PID, err)
	} else {
		time.Sleep(2 * time.Second)
	}

	// 强制杀死SIGKILL
	err = syscall.Kill(s.PID, syscall.SIGKILL)
	if err != nil {
		logger.Errorf("Failed to send SIGKILL to process %d: %v", s.PID, err)
		return err
	}

	logger.Infof("Successfully killed process %d", s.PID)
	return nil
}

// stop 内部停止方法，不加锁
func (s *McpService) stop(logger xlog.Logger) error {
	if s.IsSSE() {
		return nil
	}

	if s.Status != Running && s.Status != Starting {
		return nil
	}

	logger.Infof("Stopping process %s (PID: %d)", s.Name, s.PID)
	s.Status = Stopping

	if s.cancel != nil {
		s.cancel()
	}

	defer func() {
		if s.Status == Stopping {
			s.Status = Stopped
		}
		s.Cmd = nil
		s.PID = 0
		s.ProcessGroup = 0
	}()

	if s.Cmd == nil {
		return nil
	}

	if s.LogFile != nil {
		err := s.LogFile.Close()
		if err != nil {
			logger.Errorf("Failed to close log file: %v", err)
		}
		s.LogFile = nil
	}

	return s.killProcessGroup(logger)
}

// start 内部启动方法，不加锁
func (s *McpService) start(logger xlog.Logger) error {
	if s.IsSSE() {
		return fmt.Errorf("服务 %s 不是命令类型, 无需启动", s.Name)
	}

	if s.Status == Running {
		return fmt.Errorf("服务 %s 已运行", s.Name)
	}
	if s.Status == Failed {
		return fmt.Errorf("服务 %s 已失败，无法启动", s.Name)
	}

	s.Status = Starting
	if s.Port == 0 {
		s.Port = s.portMgr.GetNextAvailablePort()
	}

	logFile, err := xlog.CreateLogFile(s.Config.LogConfig.Path, s.Name+".log")
	if err != nil {
		return fmt.Errorf("failed to create log file: %v", err)
	}
	s.LogFile = logFile

	mcpRunner := fmt.Sprintf("%s %s", s.Config.Command, strings.Join(s.Config.Args, " "))
	cmdString := fmt.Sprintf("%s --stdio \"%s\" --port %d", config.COMMAND_SUPERGATEWA, mcpRunner, s.Port)
	cmd := exec.CommandContext(s.ctx, "/bin/sh", "-c", cmdString)

	cmd.SysProcAttr = s.buildSysProcAttr(logger)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	s.stdoutReader = bufio.NewReader(stdoutPipe)
	s.stderrReader = bufio.NewReader(stderrPipe)

	if len(s.Config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range s.Config.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	if err := cmd.Start(); err != nil {
		logger.Errorf("Standard execution failed: %v", err)
		// 尝试替代执行方式
		if altErr := s.tryAlternativeExecution(logger, err); altErr != nil {
			logger.Warnf("close logfile: %v", logFile.Close())
			return fmt.Errorf("all execution methods failed - original: %v, alternative: %v", err, altErr)
		}
		// 替代方式成功，返回
		return nil
	}

	s.Cmd = cmd
	s.PID = cmd.Process.Pid
	s.ProcessGroup = cmd.Process.Pid
	s.Status = Running
	s.lastHeartbeat = time.Now()

	logger.Infof("Started process %s with PID: %d, ProcessGroup: %d", s.Name, s.PID, s.ProcessGroup)
	return nil
}

// monitorOutput 监控进程输出
func (s *McpService) monitorOutput(logger xlog.Logger) {
	defer func() {
		if err := recover(); err != nil {
			logger.Errorf("Output monitor panic for %s: %v", s.Name, err)
		}
	}()

	// 监控stdout
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				line, err := s.stdoutReader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						logger.Errorf("Error reading stdout for %s: %v", s.Name, err)
					}
					return
				}
				s.processOutput(line, "stdout", logger)
			}
		}
	}()

	// 监控stderr
	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				line, err := s.stderrReader.ReadString('\n')
				if err != nil {
					if err != io.EOF {
						logger.Errorf("Error reading stderr for %s: %v", s.Name, err)
					}
					return
				}
				s.processOutput(line, "stderr", logger)
			}
		}
	}()
}

// processOutput 处理进程输出
func (s *McpService) processOutput(line, source string, logger xlog.Logger) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// 更新心跳时间
	s.mutex.Lock()
	s.lastHeartbeat = time.Now()
	s.mutex.Unlock()

	// 检查服务状态变化
	if s.Status == Starting && strings.Contains(line, "SSE endpoint:") {
		s.mutex.Lock()
		s.Status = Running
		s.mutex.Unlock()
		logger.Infof("Service %s is now running", s.Name)
	}

	// 写入日志文件
	if s.LogFile != nil {
		fmt.Fprintf(s.LogFile, "[%s] %s\n", source, line)
	}

	// 输出到日志
	logger.Infof("[%s:%s] %s", s.Name, source, line)

	// 检查是否有错误指示符
	if strings.Contains(strings.ToLower(line), "error") ||
		strings.Contains(strings.ToLower(line), "failed") ||
		strings.Contains(strings.ToLower(line), "exited") {
		logger.Warnf("Detected potential error in %s: %s", s.Name, line)
	}
}

// healthCheck 健康检查
func (s *McpService) healthCheck(logger xlog.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.mutex.RLock()
			lastHeartbeat := s.lastHeartbeat
			status := s.Status
			s.mutex.RUnlock()

			if status != Running {
				continue
			}

			// 检查是否超时无输出
			if time.Since(lastHeartbeat) > 2*time.Minute {
				logger.Warnf("Service %s has no output for %v, checking health", s.Name, time.Since(lastHeartbeat))

				// 检查进程是否还在运行
				if !s.isProcessAlive() {
					logger.Errorf("Process %s (PID: %d) is dead, will restart", s.Name, s.PID)
					go s.Restart(logger)
					return
				}
			}
		}
	}
}

// isProcessAlive 检查进程是否还活着
func (s *McpService) isProcessAlive() bool {
	if s.PID == 0 {
		return false
	}

	// 向进程发送信号0（不会实际杀死进程，只是检查是否存在）
	err := syscall.Kill(s.PID, 0)
	return err == nil
}

// checkExecutionPermissions 检查执行权限
func (s *McpService) checkExecutionPermissions(logger xlog.Logger) error {
	// 检查 /bin/sh 是否存在且可执行
	if _, err := os.Stat("/bin/sh"); err != nil {
		logger.Warnf("/bin/sh not found, trying /bin/bash: %v", err)
		if _, err := os.Stat("/bin/bash"); err != nil {
			return fmt.Errorf("/bin/sh and /bin/bash both not found: %v", err)
		}
	}

	// 检查 supergateway 命令是否可用
	// 先检查直接路径
	if _, err := os.Stat("/usr/local/bin/" + config.COMMAND_SUPERGATEWA); err == nil {
		logger.Infof("%s found at /usr/local/bin/", config.COMMAND_SUPERGATEWA)
	} else {
		// 再检查PATH
		cmd := exec.Command("which", config.COMMAND_SUPERGATEWA)
		if err := cmd.Run(); err != nil {
			logger.Warnf("%s command not found in PATH: %v", config.COMMAND_SUPERGATEWA, err)
			// 不是致命错误，可能是绝对路径
		}
	}

	// 检查当前用户权限
	uid := os.Getuid()
	gid := os.Getgid()
	logger.Infof("Running as UID: %d, GID: %d", uid, gid)

	return nil
}

// buildSysProcAttr 构建系统进程属性
func (s *McpService) buildSysProcAttr(logger xlog.Logger) *syscall.SysProcAttr {
	// 默认配置
	attr := &syscall.SysProcAttr{}

	// 尝试设置进程组，在某些容器环境中可能不支持
	if s.canUseProcessGroup() {
		attr.Setsid = true  // 创建新会话
		attr.Setpgid = true // 创建新进程组
		logger.Infof("Using process group for service %s", s.Name)
	} else {
		logger.Warnf("Process group not supported for service %s, using basic execution", s.Name)
	}

	return attr
}

// canUseProcessGroup 检查是否可以使用进程组
func (s *McpService) canUseProcessGroup() bool {
	// 检查环境变量，允许禁用进程组
	if os.Getenv("DISABLE_PROCESS_GROUP") == "true" {
		return false
	}

	// 检查是否在容器中运行
	if _, err := os.Stat("/.dockerenv"); err == nil {
		// 在Docker容器中，检查是否有特权
		if os.Getuid() != 0 {
			return false // 非 root 用户在容器中可能无法创建进程组
		}
	}

	return true
}

// tryAlternativeExecution 尝试替代执行方式
func (s *McpService) tryAlternativeExecution(logger xlog.Logger, originalErr error) error {
	logger.Warnf("Standard execution failed: %v, trying alternatives", originalErr)

	// 方案1：使用 /bin/bash 代替 /bin/sh
	mcpRunner := fmt.Sprintf("%s %s", s.Config.Command, strings.Join(s.Config.Args, " "))
	cmdString := fmt.Sprintf("%s --stdio \"%s\" --port %d", config.COMMAND_SUPERGATEWA, mcpRunner, s.Port)

	var cmd *exec.Cmd
	if _, err := os.Stat("/bin/bash"); err == nil {
		logger.Infof("Trying with /bin/bash: %s", cmdString)
		cmd = exec.CommandContext(s.ctx, "/bin/bash", "-c", cmdString)
	} else {
		// 方案2：直接执行命令（不通过shell）
		logger.Infof("Trying direct execution: %s", config.COMMAND_SUPERGATEWA)
		args := []string{"--stdio", mcpRunner, "--port", fmt.Sprintf("%d", s.Port)}
		cmd = exec.CommandContext(s.ctx, config.COMMAND_SUPERGATEWA, args...)
	}

	// 不设置特殊的进程属性
	cmd.SysProcAttr = &syscall.SysProcAttr{}

	// 创建管道
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe (alternative): %v", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe (alternative): %v", err)
	}

	s.stdoutReader = bufio.NewReader(stdoutPipe)
	s.stderrReader = bufio.NewReader(stderrPipe)

	// 设置环境变量
	if len(s.Config.Env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range s.Config.Env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}

	// 尝试启动
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("alternative execution also failed: %v", err)
	}

	s.Cmd = cmd
	s.PID = cmd.Process.Pid
	s.ProcessGroup = 0 // 无进程组
	s.Status = Running
	s.lastHeartbeat = time.Now()

	logger.Infof("Alternative execution successful for %s with PID: %d", s.Name, s.PID)
	return nil
}

// monitorBridge 监控stdio-sse桥接状态
func (s *McpService) monitorBridge(logger xlog.Logger) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			if s.bridge == nil {
				logger.Warnf("Bridge for service %s is nil, stopping monitor", s.Name)
				return
			}

			if !s.bridge.IsRunning() {
				logger.Errorf("Bridge for service %s is not running, marking service as failed", s.Name)
				s.mutex.Lock()
				s.Status = Failed
				s.mutex.Unlock()
				return
			}

			// 更新心跳时间
			s.mutex.Lock()
			s.lastHeartbeat = time.Now()
			s.mutex.Unlock()
		}
	}
}
