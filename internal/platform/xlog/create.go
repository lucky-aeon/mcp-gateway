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
