package main

import (
	"context"
	"fmt"
	"net/http"
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

	// 创建 Echo 实例
	e := echo.New()
	e.HideBanner = true

	// 添加中间件
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.KeyAuthWithConfig(middleware_impl.NewAuthMiddleware(cfg).GetKeyAuthConfig())) // API Key 鉴权

	// 初始化服务管理器
	srvMgr := router.NewServerManager(*cfg, e)

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
	srvMgr.Close()
	if err := e.Shutdown(ctx); err != nil {
		mainLogger.Fatalf("Error during server shutdown: %v", err)
	}
	mainLogger.Info("Server shutdown completed")
}
