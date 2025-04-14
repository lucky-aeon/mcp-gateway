package types

import "encoding/json"

// Property 定义输入参数的属性
type McpToolProperty struct {
	Type        string `json:"type"`        // 参数类型
	Description string `json:"description"` // 参数描述
}

// InputSchema 定义工具的输入参数结构
type McpToolInputSchema struct {
	Type       string                     `json:"type"`       // 通常是 "object"
	Properties map[string]McpToolProperty `json:"properties"` // 参数属性定义
	Required   []string                   `json:"required"`   // 必需的参数列表
}

// McpTool 定义工具的结构
type McpTool struct {
	RealName    string `json:"-"`           // 工具UUID
	Name        string `json:"name"`        // 工具名称
	Description string `json:"description"` // 工具描述
	InputSchema any    `json:"inputSchema"` // 输入参数结构
}

type McpResultBase struct {
	Jsonrpc string `json:"jsonrpc"`
	Id      *int64 `json:"id,omitempty"`
}

type McpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type McpResult struct {
	McpResultBase
	Result map[string]any `json:"result"`
}

func (r *McpResult) ToJson() string {
	json, _ := json.Marshal(r)
	return string(json)
}

func CreateMcpResult(jsonrpc string, id int64, result map[string]any) *McpResult {
	return &McpResult{
		McpResultBase: McpResultBase{
			Jsonrpc: jsonrpc,
			Id:      &id,
		},
		Result: result,
	}
}

type McpRequest struct {
	McpResultBase
	Method string `json:"method,omitempty"`
	Params any    `json:"params,omitempty"`
}

func (r *McpRequest) ToJson() string {
	json, _ := json.Marshal(r)
	return string(json)
}
