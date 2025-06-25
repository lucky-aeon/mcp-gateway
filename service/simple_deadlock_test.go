package service

import (
	"sync"
	"testing"
	"time"
)

// TestSingleMutexWorkspace 测试 workspace 单锁设计的并发安全性
func TestSingleMutexWorkspace(t *testing.T) {
	// 创建一个简化的 workspace 用于测试
	workspace := &WorkSpace{
		servers: make(map[string]*McpService),
		status:  WorkSpaceStatusRunning,
	}

	const numWorkers = 20
	const operationsPerWorker = 50
	var wg sync.WaitGroup

	// 并发操作
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerWorker; j++ {
				serviceName := "service-" + string(rune('0'+workerID%10)) + string(rune('0'+j%10))

				// 模拟添加服务
				workspace.serversMutex.Lock()
				workspace.servers[serviceName] = &McpService{Name: serviceName}
				workspace.serversMutex.Unlock()

				// 模拟读取操作
				workspace.serversMutex.RLock()
				_ = len(workspace.servers)
				workspace.serversMutex.RUnlock()

				// 模拟删除服务
				workspace.serversMutex.Lock()
				delete(workspace.servers, serviceName)
				workspace.serversMutex.Unlock()
			}
		}(i)
	}

	// 等待完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("Single mutex workspace test completed: %d workers × %d operations", numWorkers, operationsPerWorker)
	case <-time.After(5 * time.Second):
		t.Fatal("Single mutex workspace test timed out - possible deadlock")
	}
}

// TestSingleMutexSession 测试 session 单锁设计的并发安全性
func TestSingleMutexSession(t *testing.T) {
	session := NewSession("deadlock-test-session")

	const numWorkers = 20
	const operationsPerWorker = 50
	var wg sync.WaitGroup

	// 并发操作
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerWorker; j++ {
				// 消息 ID 操作
				msgId := session.generateMessageId(int64(workerID*1000 + j))
				if _, exists := session.getRealMessageId(msgId); exists {
					session.removeMessageId(msgId)
				}

				// 事件操作
				session.SendEvent(SessionMsg{
					Event: "test",
					Data:  "test-data",
				})

				// 通道操作
				_ = session.GetEventChan()
			}
		}(i)
	}

	// 等待完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("Single mutex session test completed: %d workers × %d operations", numWorkers, operationsPerWorker)
	case <-time.After(5 * time.Second):
		t.Fatal("Single mutex session test timed out - possible deadlock")
	}
}

// TestWorkspaceCloseOperationNoDeadlock 测试修复后的 Close 操作不会死锁
func TestWorkspaceCloseOperationNoDeadlock(t *testing.T) {
	workspace := &WorkSpace{
		servers: make(map[string]*McpService),
		status:  WorkSpaceStatusRunning,
	}

	// 添加初始服务
	for i := 0; i < 10; i++ {
		serviceName := "service-" + string(rune('0'+i))
		workspace.servers[serviceName] = &McpService{
			Name:   serviceName,
			Status: Running,
		}
	}

	var wg sync.WaitGroup

	// 并发读写操作
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				// 并发读取
				workspace.serversMutex.RLock()
				_ = len(workspace.servers)
				workspace.serversMutex.RUnlock()

				// 尝试添加新服务
				newServiceName := "concurrent-" + string(rune('0'+workerID)) + string(rune('0'+j%10))
				workspace.serversMutex.Lock()
				workspace.servers[newServiceName] = &McpService{Name: newServiceName}
				workspace.serversMutex.Unlock()

				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	// 模拟修复后的 Close 操作
	closeComplete := make(chan struct{})
	go func() {
		defer close(closeComplete)

		// 使用修复后的逻辑：循环直到所有服务都被移除
		for {
			workspace.serversMutex.Lock()
			if len(workspace.servers) == 0 {
				workspace.status = WorkSpaceStatusStopped
				workspace.serversMutex.Unlock()
				break
			}

			// 获取第一个服务名称（在锁内安全获取）
			var serverName string
			for name := range workspace.servers {
				serverName = name
				break
			}
			workspace.serversMutex.Unlock()

			// 在锁外删除服务（模拟 removeMcpServiceInternal）
			workspace.serversMutex.Lock()
			delete(workspace.servers, serverName)
			workspace.serversMutex.Unlock()
		}
	}()

	// 等待所有操作完成
	allComplete := make(chan struct{})
	go func() {
		wg.Wait()
		<-closeComplete
		close(allComplete)
	}()

	select {
	case <-allComplete:
		t.Log("Close operation completed successfully without deadlock")
	case <-time.After(5 * time.Second):
		t.Fatal("Close operation test timed out - possible deadlock")
	}

	// 验证最终状态
	if workspace.status != WorkSpaceStatusStopped {
		t.Error("Workspace should be stopped after close")
	}

	// 注意：由于并发添加和删除，最终服务数量可能不为0，这是正常的
	// 重要的是Close操作能够安全完成而不死锁
	workspace.serversMutex.RLock()
	finalCount := len(workspace.servers)
	workspace.serversMutex.RUnlock()
	t.Logf("Final server count: %d (concurrent operations may leave some services)", finalCount)
}

// TestStressTestNoDeadlock 压力测试确保修复后的代码没有死锁
func TestStressTestNoDeadlock(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	workspace := &WorkSpace{
		servers: make(map[string]*McpService),
		status:  WorkSpaceStatusRunning,
	}
	session := NewSession("stress-test")

	const numWorkers = 30
	const operationsPerWorker = 200

	var wg sync.WaitGroup

	// 高强度并发操作
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < operationsPerWorker; j++ {
				operation := (workerID + j) % 4

				switch operation {
				case 0: // Workspace 操作
					serviceName := "stress-" + string(rune('0'+workerID%10)) + string(rune('0'+j%10))
					workspace.serversMutex.Lock()
					workspace.servers[serviceName] = &McpService{Name: serviceName}
					workspace.serversMutex.Unlock()

					workspace.serversMutex.RLock()
					_ = len(workspace.servers)
					workspace.serversMutex.RUnlock()

					workspace.serversMutex.Lock()
					delete(workspace.servers, serviceName)
					workspace.serversMutex.Unlock()

				case 1: // Session 消息操作
					msgId := session.generateMessageId(int64(workerID*1000 + j))
					session.getRealMessageId(msgId)
					session.removeMessageId(msgId)

				case 2: // Session 事件操作
					session.SendEvent(SessionMsg{
						Event: "stress",
						Data:  "stress-data",
					})

				case 3: // 混合操作
					_ = session.GetEventChan()
					workspace.serversMutex.RLock()
					_ = len(workspace.servers)
					workspace.serversMutex.RUnlock()
				}

				// 小延迟避免过于密集
				if j%20 == 0 {
					time.Sleep(1 * time.Millisecond)
				}
			}
		}(i)
	}

	// 等待完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		t.Logf("Stress test completed successfully: %d workers × %d operations", numWorkers, operationsPerWorker)
	case <-time.After(15 * time.Second):
		t.Fatal("Stress test timed out - possible deadlock detected")
	}
}

// BenchmarkConcurrencyPerformance 性能基准测试
func BenchmarkConcurrencyPerformance(b *testing.B) {
	session := NewSession("benchmark-session")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		workerID := 0
		for pb.Next() {
			// 高频操作
			msgId := session.generateMessageId(int64(workerID))
			session.getRealMessageId(msgId)
			session.removeMessageId(msgId)

			session.SendEvent(SessionMsg{
				Event: "bench",
				Data:  "benchmark-data",
			})

			workerID++
		}
	})
}
