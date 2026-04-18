package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/identity"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/profiling"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/xlog"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/server"
)

func main() {
	var protocolFlag, cfgPath string
	var assumeYes bool
	flag.StringVar(&protocolFlag, "protocol", "", "Gateway protocol: sse or streamhttp")
	flag.StringVar(&cfgPath, "cfg", "./config.json", "Path to the configuration file")
	flag.BoolVar(&assumeYes, "yes", false, "Assume yes to all prompts (e.g. auto-create missing workspace directory)")
	flag.Parse()

	cfg, err := config.InitConfig(cfgPath)
	if err != nil {
		panic(fmt.Errorf("failed to init config: %w", err))
	}
	if protocolFlag != "" {
		cfg.GatewayProtocol = protocolFlag
	}

	// 确保 WorkspacePath 存在；不存在则询问用户是否创建
	if err := ensureWorkspacePath(cfg.WorkspacePath, assumeYes); err != nil {
		panic(fmt.Errorf("failed to prepare workspace path %q: %w", cfg.WorkspacePath, err))
	}

	defer func() {
		cfg.SaveConfig()
	}()

	// Setup logging with zap
	xlog.SetHeader(xlog.DefaultHeader)
	err = xlog.SetupFileLogging(cfg.WorkspacePath, "plugin-proxy.log")
	if err != nil {
		panic(fmt.Errorf("failed to setup file logging: %w", err))
	}

	// Ensure log files are closed on exit
	defer xlog.CloseLogFiles()

	// Create main logger
	mainLogger := xlog.NewLogger("MAIN")
	mainLogger.Infof("Starting MCP Gateway server, log level: %d", cfg.LogLevel)

	// 启动CPU性能分析
	cpuProfile := profiling.StartCPUProfile("cpu_profile.prof")
	defer profiling.StopCPUProfile(cpuProfile)

	// 启动定期性能分析
	profiling.StartPeriodicProfiling(5 * time.Minute)

	// 创建 Echo 实例
	e := echo.New()
	e.HideBanner = true

	// 添加中间件
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.KeyAuthWithConfig(identity.NewAuthMiddleware(cfg).GetKeyAuthConfig())) // API Key 鉴权

	// 初始化服务管理器
	srvMgr := server.New(*cfg, e)

	// 启动 pprof 调试服务器在单独端口
	go func() {
		mainLogger.Info("Starting pprof server on :6060")
		if err := http.ListenAndServe(":6060", nil); err != nil {
			mainLogger.Errorf("pprof server error: %v", err)
		}
	}()

	// 设置优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	// 启动服务器（非阻塞）
	go func() {
		mainLogger.Infof("Starting server on %s", cfg.Bind)
		if err := e.Start(cfg.Bind); err != nil && err != http.ErrServerClosed {
			mainLogger.Fatal("shutting down the server")
		}
	}()

	// 等待退出信号
	<-quit
	mainLogger.Info("Received shutdown signal, starting graceful shutdown...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 生成最终的性能分析文件
	profiling.WriteMemProfile("final_mem_profile.prof")
	profiling.WriteGoroutineProfile("final_goroutine_profile.prof")

	srvMgr.Close()
	if err := e.Shutdown(ctx); err != nil {
		mainLogger.Fatalf("Error during server shutdown: %v", err)
	}
	mainLogger.Info("Server shutdown completed")
}

// ensureWorkspacePath 确保 workspace 目录存在。
// 目录不存在时：若 assumeYes 为 true 或 stdin 为交互式终端且用户确认，则创建；否则返回错误。
func ensureWorkspacePath(path string, assumeYes bool) error {
	if path == "" {
		return fmt.Errorf("workspace path is empty")
	}
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("workspace path %q exists but is not a directory", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	create := assumeYes
	if !create {
		if !isStdinTerminal() {
			return fmt.Errorf("workspace path %q does not exist; re-run with -yes to auto-create, or create it manually", path)
		}
		fmt.Printf("Workspace directory %q does not exist. Create it? [y/N]: ", path)
		reader := bufio.NewReader(os.Stdin)
		answer, readErr := reader.ReadString('\n')
		if readErr != nil {
			return fmt.Errorf("read user input: %w", readErr)
		}
		answer = strings.TrimSpace(strings.ToLower(answer))
		create = answer == "y" || answer == "yes"
		if !create {
			return fmt.Errorf("workspace path creation declined by user")
		}
	}

	if err := os.MkdirAll(path, 0o755); err != nil {
		return fmt.Errorf("mkdir workspace path: %w", err)
	}
	fmt.Printf("Created workspace directory: %s\n", path)
	return nil
}

// isStdinTerminal 判断 stdin 是否连接到交互式终端。
func isStdinTerminal() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
