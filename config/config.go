package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Config struct {
	LogLevel            uint8         // 日志级别
	ConfigDirPath       string        // 配置文件路径
	Bind                string        // 绑定地址 // [::]:8080
	Auth                *AuthConfig   // 认证配置
	SessionGCInterval   time.Duration // Session GC间隔
	ProxySessionTimeout time.Duration // Proxy Session 超时时间
	McpServiceMgrConfig McpServiceMgrConfig
}

func InitConfig(cfgDir string) (cfg *Config, err error) {
	cfg = &Config{}
	configPath := filepath.Join(cfgDir, CONFIG_PATH)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		cfg.Default()
		cfg.ConfigDirPath = cfgDir
		return cfg, nil
	}
	file, err := os.OpenFile(configPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", configPath, err)
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(cfg)
	if err != nil {
		return nil, err
	}
	cfg.ConfigDirPath = cfgDir
	cfg.Default()
	return cfg, nil
}

func (c *Config) Default() {
	if c.Bind == "" {
		c.Bind = "[::]:8080" // 默认绑定地址
	}
	if c.Auth == nil {
		c.Auth = &AuthConfig{
			Enabled: true,
			ApiKey:  "123456", // 默认的API Key, 可在header或者query中使用
		}
	}
	if c.SessionGCInterval == 0 {
		c.SessionGCInterval = 5 * time.Minute
	}
	if c.ProxySessionTimeout == 0 {
		c.ProxySessionTimeout = 5 * c.SessionGCInterval // 默认为5倍GC间隔
	}
	if c.McpServiceMgrConfig.McpServiceRetryCount == 0 {
		c.McpServiceMgrConfig.McpServiceRetryCount = 3
	}

}

func (c *Config) GetAuthConfig() *AuthConfig {
	if c.Auth == nil {
		c.Auth = &AuthConfig{
			Enabled: true,
			ApiKey:  "123456", // 默认的API Key, 可在header或者query中使用
		}
	}
	return c.Auth
}

type AuthConfig struct {
	Enabled bool
	ApiKey  string
}

func (c *AuthConfig) IsEnabled() bool {
	return c.Enabled
}

func (c *AuthConfig) GetApiKey() string {
	return c.ApiKey
}

type McpServiceMgrConfig struct {
	McpServiceRetryCount int // 服务重试次数，服务挂掉后会重试
}

func (c *McpServiceMgrConfig) GetMcpServiceRetryCount() int {
	if c.McpServiceRetryCount == 0 {
		return 3
	}
	return c.McpServiceRetryCount
}

// MCP Config path
const MCP_CONFIG_PATH = "mcp_servers.json"

func (c *Config) GetMcpConfigPath() string {
	return filepath.Join(c.ConfigDirPath, MCP_CONFIG_PATH)
}

const CONFIG_PATH = "config.json"

// 保存这个Config信息
func (c *Config) SaveConfig() error {
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	file, err := os.OpenFile(filepath.Join(c.ConfigDirPath, CONFIG_PATH), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
