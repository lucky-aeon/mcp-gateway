package main

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/middleware_impl"
	"github.com/lucky-aeon/agentx/plugin-helper/router"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

func main() {
	cfgDir := "./vm"
	if _, err := os.Stat(cfgDir); os.IsNotExist(err) {
		cfgDir = "."
	}
	cfg, err := config.InitConfig(cfgDir)
	if err != nil {
		panic(fmt.Errorf("failed to init config: %w", err))
	}
	defer func() {
		cfg.SaveConfig()
	}()

	// Setup logging with zap
	xlog.SetHeader(xlog.DefaultHeader)
	err = xlog.SetupFileLogging(cfg.ConfigDirPath, "plugin-proxy.log")
	if err != nil {
		panic(fmt.Errorf("failed to setup file logging: %w", err))
	}

	// Ensure log files are closed on exit
	defer xlog.CloseLogFiles()

	// Create main logger
	mainLogger := xlog.NewLogger("MAIN")
	mainLogger.Infof("Starting MCP Gateway server, log level: %d", cfg.LogLevel)

	// 启动CPU性能分析
	cpuProfile := StartCPUProfile("cpu_profile.prof")
	defer StopCPUProfile(cpuProfile)

	// 启动定期性能分析
	StartPeriodicProfiling(5 * time.Minute)

	// 创建 Echo 实例
	e := echo.New()
	e.HideBanner = true

	// 添加中间件
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.KeyAuthWithConfig(middleware_impl.NewAuthMiddleware(cfg).GetKeyAuthConfig())) // API Key 鉴权

	// 初始化服务管理器
	srvMgr := router.NewServerManager(*cfg, e)

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
	WriteMemProfile("final_mem_profile.prof")
	WriteGoroutineProfile("final_goroutine_profile.prof")

	srvMgr.Close()
	if err := e.Shutdown(ctx); err != nil {
		mainLogger.Fatalf("Error during server shutdown: %v", err)
	}
	mainLogger.Info("Server shutdown completed")
}
