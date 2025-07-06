package service

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/lucky-aeon/agentx/plugin-helper/config"
	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

var mockPortMgr PortManagerI = NewPortManager()

func mockMcpServiceFileSystem(t *testing.T) *McpService {
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	pwd += "/testdata"
	os.Mkdir(pwd, 0755)
	os.WriteFile(pwd+"/test.txt", []byte("Hello, World!"), 0644)
	return NewMcpService("fileSystem", config.MCPServerConfig{
		Workspace: "default",
		Command:   "npx",
		Args: []string{
			"-y",
			"@modelcontextprotocol/server-filesystem",
			pwd,
		},
	}, mockPortMgr)
}

func TestMcpService_Restart_DeadlockPrevention(t *testing.T) {
	service := &McpService{
		Name:       "test-service",
		Status:     Stopped,
		RetryCount: 2,
		RetryMax:   3,
		portMgr:    mockPortMgr,
		mutex:      sync.RWMutex{},
		Config: config.MCPServerConfig{
			Command: "invalid-command", // 故意使用无效命令
			Args:    []string{"invalid-args"},
			McpServiceMgrConfig: config.McpServiceMgrConfig{
				McpServiceRetryCount: 3,
			},
		},
	}

	logger := xlog.NewLogger("test")

	// 使用channel来检测死锁
	done := make(chan bool, 1)

	go func() {
		service.Restart(logger)
		done <- true
	}()

	// 等待最多10秒，如果超时说明可能发生死锁
	select {
	case <-done:
		t.Log("Restart completed without deadlock")
	case <-time.After(10 * time.Second):
		t.Fatal("Restart method appears to be deadlocked")
	}

	// 等待一段时间让重试逻辑完成
	time.Sleep(2 * time.Second)

	// 验证服务最终状态（应该是Failed，因为命令无效）
	status := service.GetStatus()
	if status != Failed && status != Stopped {
		t.Errorf("Expected service status to be Failed or Stopped, got %s", status)
	}
}

func TestMcpService_Restart_SSEService(t *testing.T) {
	service := &McpService{
		Name:   "sse-service",
		Status: Running,
		Config: config.MCPServerConfig{
			URL: "http://example.com/sse",
		},
		portMgr: mockPortMgr,
		mutex:   sync.RWMutex{},
	}

	logger := xlog.NewLogger("test")

	service.Restart(logger)

	// SSE服务应该保持Running状态
	if service.GetStatus() != Running {
		t.Errorf("Expected SSE service status to remain Running, got %s", service.GetStatus())
	}
}

func TestMcpService_Restart_MaxRetriesReached(t *testing.T) {
	service := &McpService{
		Name:       "test-service",
		Status:     Running,
		RetryCount: 0, // 没有重试次数
		RetryMax:   3,
		portMgr:    mockPortMgr,
		mutex:      sync.RWMutex{},
	}

	logger := xlog.NewLogger("test")

	service.Restart(logger)

	// 验证服务被标记为失败
	if service.GetStatus() != Failed {
		t.Errorf("Expected service status to be Failed, got %s", service.GetStatus())
	}

	if service.FailureReason != "Max retry count reached" {
		t.Errorf("Expected failure reason to be 'Max retry count reached', got %s", service.FailureReason)
	}

	if service.LastError != "Service failed after maximum retry attempts" {
		t.Errorf("Expected last error to be 'Service failed after maximum retry attempts', got %s", service.LastError)
	}
}
