package types

import "github.com/lucky-aeon/agentx/plugin-helper/config"

// DeployRequest 部署请求结构
type DeployRequest struct {
	MCPServers map[string]config.MCPServerConfig `json:"mcpServers"`
}

// ServiceDeployStatus 服务部署状态
type ServiceDeployStatus string

const (
	ServiceDeployStatusExisted  ServiceDeployStatus = "existed"  // 服务已存在且正在运行
	ServiceDeployStatusDeployed ServiceDeployStatus = "deployed" // 服务新部署成功
	ServiceDeployStatusFailed   ServiceDeployStatus = "failed"   // 服务部署失败
	ServiceDeployStatusReplaced ServiceDeployStatus = "replaced" // 服务被替换（停止/失败状态的服务被重新部署）
)

// ServiceDeployResult 单个服务部署结果
type ServiceDeployResult struct {
	Name    string              `json:"name"`              // 服务名称
	Status  ServiceDeployStatus `json:"status"`            // 部署状态
	Message string              `json:"message,omitempty"` // 详细信息或错误消息
	Error   string              `json:"error,omitempty"`   // 错误详情
}

// DeployResponse 部署响应结构
type DeployResponse struct {
	Success bool                           `json:"success"` // 整体是否成功
	Message string                         `json:"message"` // 整体状态消息
	Results map[string]ServiceDeployResult `json:"results"` // 每个服务的部署结果
	Summary DeploymentSummary              `json:"summary"` // 部署汇总
}

// DeploymentSummary 部署汇总信息
type DeploymentSummary struct {
	Total    int `json:"total"`    // 总数
	Existed  int `json:"existed"`  // 已存在的服务数量
	Deployed int `json:"deployed"` // 新部署的服务数量
	Replaced int `json:"replaced"` // 被替换的服务数量
	Failed   int `json:"failed"`   // 部署失败的服务数量
}
