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
	WorkspacePath       string        // 工作区路径：程序所有产出数据所在目录（日志、mcp_servers.json 等），默认 ./vm
	Bind                string        // 绑定地址 // [::]:8080
	Auth                *AuthConfig   // 认证配置
	SessionGCInterval   time.Duration // Session GC间隔
	ProxySessionTimeout time.Duration // Proxy Session 超时时间
	McpServiceMgrConfig McpServiceMgrConfig
	GatewayProtocol     string // 新增: "sse" | "streamhttp"

	cfgPath string `json:"-"` // 加载时使用的配置文件路径，SaveConfig 将回写到此
}

// InitConfig 从指定配置文件路径加载配置。
// 若文件不存在，则返回填充了默认值的 Config（记录路径，以便后续 SaveConfig 持久化）。
func InitConfig(cfgPath string) (cfg *Config, err error) {
	cfg = &Config{cfgPath: cfgPath}
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		cfg.Default()
		return cfg, nil
	}
	file, err := os.OpenFile(cfgPath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", cfgPath, err)
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(cfg)
	if err != nil {
		return nil, err
	}
	cfg.cfgPath = cfgPath
	cfg.Default()
	return cfg, nil
}

func (c *Config) Default() {
	if c.Bind == "" {
		c.Bind = "[::]:8080" // 默认绑定地址
	}
	if c.Auth == nil {
		c.Auth = &AuthConfig{
			Enabled:               true,
			Mode:                  "single-key",
			ApiKey:                "123456", // 默认的API Key, 可在header或者query中使用
			AllowRegister:         false,
			JWTSecret:             "gateway-dev-secret",
			AccessTokenTTLMinutes: 120,
			RefreshTokenTTLHours:  720,
			MongoURI:              "mongodb://127.0.0.1:27017",
			MongoDatabase:         "mcp_gateway",
			AdminEmail:            "admin@gateway.local",
			AdminPassword:         "admin123456",
			AdminDisplayName:      "Gateway Admin",
		}
	}
	if c.Auth.Mode == "" {
		c.Auth.Mode = "single-key"
	}
	if c.Auth.JWTSecret == "" {
		c.Auth.JWTSecret = "gateway-dev-secret"
	}
	if c.Auth.AccessTokenTTLMinutes == 0 {
		c.Auth.AccessTokenTTLMinutes = 120
	}
	if c.Auth.RefreshTokenTTLHours == 0 {
		c.Auth.RefreshTokenTTLHours = 720
	}
	if c.Auth.MongoURI == "" {
		c.Auth.MongoURI = "mongodb://127.0.0.1:27017"
	}
	if c.Auth.MongoDatabase == "" {
		c.Auth.MongoDatabase = "mcp_gateway"
	}
	if c.Auth.AdminEmail == "" {
		c.Auth.AdminEmail = "admin@gateway.local"
	}
	if c.Auth.AdminPassword == "" {
		c.Auth.AdminPassword = "admin123456"
	}
	if c.Auth.AdminDisplayName == "" {
		c.Auth.AdminDisplayName = "Gateway Admin"
	}
	if c.SessionGCInterval == 0 {
		c.SessionGCInterval = 10 * time.Second
	}
	if c.ProxySessionTimeout == 0 {
		c.ProxySessionTimeout = 1 * time.Minute
	}
	if c.McpServiceMgrConfig.McpServiceRetryCount == 0 {
		c.McpServiceMgrConfig.McpServiceRetryCount = 3
	}
	if c.GatewayProtocol == "" {
		c.GatewayProtocol = "sse" // 默认 SSE
	}
	if c.WorkspacePath == "" {
		c.WorkspacePath = "./vm" // 默认在当前运行目录下的 vm 目录
	}
}

func (c *Config) IsStreamHTTP() bool {
	return c.GatewayProtocol == "streamhttp"
}

func (c *Config) GetAuthConfig() *AuthConfig {
	if c.Auth == nil {
		c.Auth = &AuthConfig{
			Enabled:               true,
			Mode:                  "single-key",
			ApiKey:                "123456", // 默认的API Key, 可在header或者query中使用
			AllowRegister:         false,
			JWTSecret:             "gateway-dev-secret",
			AccessTokenTTLMinutes: 120,
			RefreshTokenTTLHours:  720,
			MongoURI:              "mongodb://127.0.0.1:27017",
			MongoDatabase:         "mcp_gateway",
			AdminEmail:            "admin@gateway.local",
			AdminPassword:         "admin123456",
			AdminDisplayName:      "Gateway Admin",
		}
	}
	return c.Auth
}

type AuthConfig struct {
	Enabled               bool
	Mode                  string
	ApiKey                string
	AllowRegister         bool
	JWTSecret             string
	AccessTokenTTLMinutes int
	RefreshTokenTTLHours  int
	MongoURI              string
	MongoDatabase         string
	AdminEmail            string
	AdminPassword         string
	AdminDisplayName      string
}

func (c *AuthConfig) IsEnabled() bool {
	return c.Enabled
}

func (c *AuthConfig) GetApiKey() string {
	return c.ApiKey
}

func (c *AuthConfig) GetMode() string {
	if c.Mode == "" {
		return "single-key"
	}
	return c.Mode
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
	return filepath.Join(c.WorkspacePath, MCP_CONFIG_PATH)
}

const CONFIG_PATH = "config.json"

// CfgPath 返回加载时使用的配置文件路径。
func (c *Config) CfgPath() string {
	return c.cfgPath
}

// SaveConfig 将当前配置回写到加载时使用的配置文件路径。
func (c *Config) SaveConfig() error {
	if c.cfgPath == "" {
		return fmt.Errorf("cfg path is empty, cannot save config")
	}
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	file, err := os.OpenFile(c.cfgPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer file.Close()

	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}
