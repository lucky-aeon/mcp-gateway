package admin

import "github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"

type marketInstallSpec struct {
	Type    string
	Command string
	Args    []string
	Env     map[string]string
}

type marketToolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type marketPackage struct {
	ID          string
	Name        string
	Version     string
	Description string
	Author      string
	Tags        []string
	Category    string
	Rating      float64
	Downloads   int
	Verified    bool
	SourceID    string
	Install     marketInstallSpec
	Tools       []marketToolSpec
	Readme      string
	Versions    []string
}

var defaultMarketSources = []map[string]interface{}{
	{
		"id":             "official",
		"name":           "MCP Official Registry",
		"url":            "https://registry.mcp.dev",
		"trusted":        true,
		"enabled":        true,
		"priority":       1,
		"scope":          "platform",
		"total_packages": 4,
		"last_synced":    "2026-04-19T00:00:00Z",
		"status":         "healthy",
	},
}

var defaultMarketPackages = []marketPackage{
	{
		ID:          "time-tools",
		Name:        "Time Tools",
		Version:     "1.0.0",
		Description: "提供时区时间查询与时间换算能力。",
		Author:      "MCP Team",
		Tags:        []string{"time", "timezone", "utility"},
		Category:    "效率",
		Rating:      4.8,
		Downloads:   12500,
		Verified:    true,
		SourceID:    "official",
		Install: marketInstallSpec{
			Type:    "uvx",
			Command: "uvx",
			Args:    []string{"mcp-server-time", "--local-timezone=Asia/Shanghai"},
			Env:     map[string]string{"TZ": "Asia/Shanghai"},
		},
		Tools: []marketToolSpec{
			{Name: "get_current_time", Description: "获取指定时区当前时间。", InputSchema: map[string]interface{}{"type": "object"}},
			{Name: "convert_time", Description: "在不同时区之间换算时间。", InputSchema: map[string]interface{}{"type": "object"}},
		},
		Readme:   "# Time Tools\n\n适合需要时区和时间换算的智能体场景。",
		Versions: []string{"1.0.0"},
	},
	{
		ID:          "filesystem-tools",
		Name:        "Filesystem Tools",
		Version:     "1.2.0",
		Description: "提供基础文件读写和目录查看能力。",
		Author:      "MCP Team",
		Tags:        []string{"filesystem", "file", "directory"},
		Category:    "系统",
		Rating:      4.7,
		Downloads:   22100,
		Verified:    true,
		SourceID:    "official",
		Install: marketInstallSpec{
			Type:    "npx",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
			Env:     map[string]string{},
		},
		Tools: []marketToolSpec{
			{Name: "read_file", Description: "读取文件内容。", InputSchema: map[string]interface{}{"type": "object"}},
			{Name: "list_directory", Description: "列出目录文件。", InputSchema: map[string]interface{}{"type": "object"}},
		},
		Readme:   "# Filesystem Tools\n\n提供常见文件系统工具。",
		Versions: []string{"1.2.0", "1.1.0"},
	},
	{
		ID:          "github-tools",
		Name:        "GitHub Tools",
		Version:     "1.1.0",
		Description: "GitHub 仓库、Issue 与 PR 相关工具。",
		Author:      "GitHub Team",
		Tags:        []string{"github", "repo", "pull-request"},
		Category:    "开发",
		Rating:      4.5,
		Downloads:   9300,
		Verified:    true,
		SourceID:    "official",
		Install: marketInstallSpec{
			Type:    "url",
			Command: "",
			Args:    nil,
			Env:     map[string]string{},
		},
		Tools: []marketToolSpec{
			{Name: "list_repos", Description: "列出仓库。", InputSchema: map[string]interface{}{"type": "object"}},
		},
		Readme:   "# GitHub Tools\n\n需要自行补充 GitHub 认证环境变量。",
		Versions: []string{"1.1.0"},
	},
	{
		ID:          "web-search-tools",
		Name:        "Web Search Tools",
		Version:     "0.9.0",
		Description: "面向搜索与网页摘要的轻量 MCP。",
		Author:      "Search Team",
		Tags:        []string{"search", "web"},
		Category:    "网络",
		Rating:      4.3,
		Downloads:   5800,
		Verified:    false,
		SourceID:    "official",
		Install: marketInstallSpec{
			Type:    "command",
			Command: "python3",
			Args:    []string{"-m", "web_search_mcp"},
			Env:     map[string]string{},
		},
		Tools: []marketToolSpec{
			{Name: "search", Description: "执行网页搜索。", InputSchema: map[string]interface{}{"type": "object"}},
		},
		Readme:   "# Web Search Tools\n\n适合检索和摘要场景。",
		Versions: []string{"0.9.0"},
	},
}

func getMarketPackage(id string) (*marketPackage, bool) {
	for _, item := range defaultMarketPackages {
		if item.ID == id {
			cp := item
			return &cp, true
		}
	}
	return nil, false
}

func packageConfigFromMarket(pkg marketPackage, workspaceID string, env map[string]string) config.MCPServerConfig {
	cfg := config.MCPServerConfig{
		Workspace: workspaceID,
		Command:   pkg.Install.Command,
		Args:      append([]string(nil), pkg.Install.Args...),
		Env:       map[string]string{},
		URL:       "",
	}
	if pkg.Install.Type == "url" {
		cfg.URL = "https://example.com/" + pkg.ID
		cfg.Command = ""
		cfg.Args = nil
	}
	for k, v := range pkg.Install.Env {
		cfg.Env[k] = v
	}
	for k, v := range env {
		cfg.Env[k] = v
	}
	return cfg
}
