package router

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/service"
	"github.com/lucky-aeon/agentx/plugin-helper/utils"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

// proxyHandler 返回代理处理函数
func (m *ServerManager) proxyHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		xl := xlog.NewLogger("PROXY")
		path := c.Request().URL.Path

		// 从路径中提取服务名和路由
		parts := strings.Split(strings.TrimPrefix(path, "/"), "/")
		if len(parts) < 2 {
			return c.String(http.StatusNotFound, "Invalid path")
		}

		serviceName := parts[0]
		lastRoute := parts[len(parts)-1] // 获取最后一个路由部分
		remainingPath := "/" + strings.Join(parts[1:], "/")

		// 获取workspace信息
		workspace := utils.GetWorkspace(c, service.DefaultWorkspace)

		// 获取服务配置
		m.RLock()
		instance, err := m.mcpServiceMgr.GetMcpService(xl, service.NameArg{
			Server:    serviceName,
			Workspace: workspace,
		})
		m.RUnlock()

		if err != nil {
			return c.String(http.StatusNotFound, "Service not found")
		}

		// 获取原始请求的查询参数
		originalQuery := c.Request().URL.RawQuery

		// 根据最后一个路由进行不同处理
		var baseURL string
		switch lastRoute {
		case "sse":
			// 对于SSE，使用完整的SSE URL
			baseURL = instance.GetSSEUrl()
		case "message":
			// 对于message，使用完整的Message URL
			baseURL = instance.GetMessageUrl()
			c.Logger().Infof("Message URL: %s", baseURL)
		default:
			// 对于其他路由，使用基础URL加上完整路径
			if url := instance.GetUrl(); url != "" {
				// 移除URL末尾的斜杠，避免双斜杠
				baseURL = strings.TrimRight(url, "/")
				if remainingPath != "/" {
					baseURL += remainingPath
				}
			} else {
				return c.String(http.StatusNotFound, "Service not available")
			}
		}

		// 构建目标URL，保留原始查询参数
		targetURL := baseURL
		if originalQuery != "" {
			// 检查baseURL是否已经包含查询参数
			if strings.Contains(baseURL, "?") {
				targetURL = baseURL + "&" + originalQuery
			} else {
				targetURL = baseURL + "?" + originalQuery
			}
		}

		c.Logger().Infof("Proxy request: %s, target URL: %s, lastRoute: %s, query: %s",
			c.Request().URL, targetURL, lastRoute, originalQuery)

		// 创建新的请求
		req, err := http.NewRequest(c.Request().Method, targetURL, c.Request().Body)
		if err != nil {
			return err
		}

		// 复制原始请求的 header
		for k, v := range c.Request().Header {
			req.Header[k] = v
		}

		// 发送请求
		client := &http.Client{
			Transport: &http.Transport{
				ForceAttemptHTTP2: false,
			},
		}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		// 复制响应 header
		for k, v := range resp.Header {
			c.Response().Header()[k] = v
		}

		// 对于 SSE 请求的特殊处理
		if utils.IsSSE(resp.Header) {
			c.Response().Header().Set("Content-Type", "text/event-stream")
			c.Response().Header().Set("Cache-Control", "no-cache")
			c.Response().Header().Set("Connection", "keep-alive")
			c.Response().WriteHeader(resp.StatusCode)
			c.Response().Flush()

			reader := bufio.NewReader(resp.Body)
			var currentEvent string

			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
					return err
				}

				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}

				// 处理事件行
				if strings.HasPrefix(line, "event: ") {
					currentEvent = strings.TrimPrefix(line, "event: ")
					fmt.Fprintf(c.Response(), "event: %s\n", currentEvent)
				} else if strings.HasPrefix(line, "data: ") {
					data := strings.TrimPrefix(line, "data: ")

					// 如果是endpoint事件，添加服务名前缀
					if currentEvent == "endpoint" && strings.HasPrefix(data, "/message") {
						data = fmt.Sprintf("/%s%s", serviceName, data)
					}

					fmt.Fprintf(c.Response(), "data: %s\n\n", data)
				} else {
					fmt.Fprintf(c.Response(), "%s\n", line)
				}
				c.Response().Flush()
			}
			return nil
		}

		// 非 SSE 请求的普通处理
		c.Response().WriteHeader(resp.StatusCode)
		_, err = io.Copy(c.Response().Writer, resp.Body)
		return err
	}
}
