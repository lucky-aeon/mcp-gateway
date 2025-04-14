package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"

	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/middleware_impl"
	"github.com/lucky-aeon/agentx/plugin-helper/router"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

func main() {
	cfgDir := "/etc/proxy"
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

	xlog.SetHeader(xlog.DefaultHeader)
	log.Infof("log level: %d", cfg.LogLevel)
	log.SetLevel(log.Lvl(cfg.LogLevel))

	// 创建proxy log
	proxyLogFile, err := xlog.CreateLogFile(cfg.ConfigDirPath, "plugin-proxy.log")
	if err != nil {
		panic(fmt.Errorf("failed to create proxy log file: %w", err))
	}

	// 创建 Echo 实例
	e := echo.New()
	e.Logger.SetOutput(io.MultiWriter(proxyLogFile, os.Stdout))

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
		if err := e.Start(cfg.Bind); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	// 等待退出信号
	<-quit

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srvMgr.Close()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}
