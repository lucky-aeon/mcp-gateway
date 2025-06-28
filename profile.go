package main

import (
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
)

var startProfile bool

func init() {
	startProfile = os.Getenv("NO_Profile") == "true"
}

// StartCPUProfile 开始CPU性能分析
func StartCPUProfile(filename string) *os.File {
	if !startProfile {
		return nil
	}

	logger := xlog.NewLogger("PROFILE")

	f, err := os.Create(filename)
	if err != nil {
		logger.Errorf("Could not create CPU profile: %v", err)
		return nil
	}

	if err := pprof.StartCPUProfile(f); err != nil {
		logger.Errorf("Could not start CPU profile: %v", err)
		f.Close()
		return nil
	}

	logger.Infof("CPU profiling started, writing to %s", filename)
	return f
}

// StopCPUProfile 停止CPU性能分析
func StopCPUProfile(f *os.File) {
	if f != nil {
		pprof.StopCPUProfile()
		f.Close()
		xlog.NewLogger("PROFILE").Info("CPU profiling stopped")
	}
}

// WriteMemProfile 写入内存性能分析
func WriteMemProfile(filename string) {
	if !startProfile {
		return
	}

	logger := xlog.NewLogger("PROFILE")

	f, err := os.Create(filename)
	if err != nil {
		logger.Errorf("Could not create memory profile: %v", err)
		return
	}
	defer f.Close()

	runtime.GC() // 触发垃圾回收以获得更准确的内存使用情况
	if err := pprof.WriteHeapProfile(f); err != nil {
		logger.Errorf("Could not write memory profile: %v", err)
		return
	}

	logger.Infof("Memory profile written to %s", filename)
}

// WriteGoroutineProfile 写入协程性能分析
func WriteGoroutineProfile(filename string) {
	if !startProfile {
		return
	}

	logger := xlog.NewLogger("PROFILE")

	f, err := os.Create(filename)
	if err != nil {
		logger.Errorf("Could not create goroutine profile: %v", err)
		return
	}
	defer f.Close()

	if err := pprof.Lookup("goroutine").WriteTo(f, 0); err != nil {
		logger.Errorf("Could not write goroutine profile: %v", err)
		return
	}

	logger.Infof("Goroutine profile written to %s", filename)
}

// StartPeriodicProfiling 定期生成性能分析文件
func StartPeriodicProfiling(interval time.Duration) {
	if !startProfile {
		return
	}
	logger := xlog.NewLogger("PROFILE")
	logger.Infof("Starting periodic profiling every %v", interval)

	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				timestamp := time.Now().Format("20060102_150405")
				WriteMemProfile("mem_profile_" + timestamp + ".prof")
				WriteGoroutineProfile("goroutine_profile_" + timestamp + ".prof")
			}
		}
	}()
}
