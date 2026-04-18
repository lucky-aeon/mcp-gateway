package xlog

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	logFiles   = make(map[string]*os.File)
	filesMutex sync.RWMutex
)

func CreateLogDir(baseDir string) error {
	if err := os.MkdirAll(filepath.Join(baseDir, "logs"), 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	return nil
}

func CreateLogFile(baseDir, fileName string) (*os.File, error) {
	// fail-fast：避免 baseDir 为空时 filepath.Join 退化成 "./logs/..."，导致日志文件跑到 CWD。
	// 调用方（例如 runtime.McpService.Start）必须保证已从 workspace 继承了正确的日志根目录。
	if baseDir == "" {
		return nil, fmt.Errorf("xlog.CreateLogFile: empty baseDir for file %q", fileName)
	}
	err := CreateLogDir(baseDir)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(filepath.Join(baseDir, "logs", fileName), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Store file reference for potential cleanup
	filesMutex.Lock()
	logFiles[fileName] = file
	filesMutex.Unlock()

	return file, nil
}

// CloseLogFiles closes all opened log files
func CloseLogFiles() {
	filesMutex.Lock()
	defer filesMutex.Unlock()

	for name, file := range logFiles {
		if err := file.Close(); err != nil {
			fmt.Printf("Error closing log file %s: %v\n", name, err)
		}
	}
	clear(logFiles)

	// Sync the global logger
	Sync()
}
