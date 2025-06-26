#!/bin/bash

# 测试MCP服务调试API的脚本

BASE_URL="http://localhost:8080"
WORKSPACE="default"
SERVICE_NAME="test-service"

echo "=== MCP Gateway 调试功能测试 ==="
echo ""

# 检查服务器是否运行
echo "1. 检查服务器状态..."
if curl -s "${BASE_URL}/api/workspaces" > /dev/null; then
    echo "✓ 服务器运行正常"
else
    echo "✗ 服务器未运行，请先启动 mcp-gateway"
    exit 1
fi

echo ""

# 获取所有工作空间
echo "2. 获取工作空间列表..."
curl -s "${BASE_URL}/api/workspaces" | jq '.' 2>/dev/null || echo "获取工作空间列表"
echo ""

# 获取指定工作空间的服务
echo "3. 获取工作空间 '${WORKSPACE}' 的服务列表..."
curl -s "${BASE_URL}/api/workspaces/${WORKSPACE}/services" | jq '.' 2>/dev/null || echo "获取服务列表"
echo ""

# 检查是否有服务可以调试
SERVICES=$(curl -s "${BASE_URL}/api/workspaces/${WORKSPACE}/services" 2>/dev/null)
if [ -z "$SERVICES" ] || [ "$SERVICES" = "null" ]; then
    echo "⚠️  工作空间 '${WORKSPACE}' 中没有运行的服务"
    echo "   请先部署一个MCP服务来测试调试功能"
    echo ""
    echo "   示例部署命令:"
    echo "   curl -X POST ${BASE_URL}/deploy \\"
    echo "        -H 'Content-Type: application/json' \\"
    echo "        -d '{\"mcp_servers\":{\"test\":{\"command\":\"uvx\",\"args\":[\"mcp-server-git\"],\"workspace\":\"${WORKSPACE}\"}}}'"
    echo ""
    exit 0
fi

# 获取第一个服务名称进行测试
FIRST_SERVICE=$(echo "$SERVICES" | jq -r 'keys[0]' 2>/dev/null)
if [ "$FIRST_SERVICE" = "null" ] || [ -z "$FIRST_SERVICE" ]; then
    echo "⚠️  无法获取服务名称"
    exit 0
fi

SERVICE_NAME="$FIRST_SERVICE"
echo "4. 使用服务 '${SERVICE_NAME}' 进行调试测试..."
echo ""

# 测试调试信息接口
echo "5. 获取服务调试信息..."
curl -s "${BASE_URL}/api/workspaces/${WORKSPACE}/services/${SERVICE_NAME}/debug/info" | jq '.' 2>/dev/null || echo "调试信息接口调用"
echo ""

# 测试连接测试接口
echo "6. 测试服务连接..."
curl -s "${BASE_URL}/api/workspaces/${WORKSPACE}/services/${SERVICE_NAME}/debug/connection" | jq '.' 2>/dev/null || echo "连接测试接口调用"
echo ""

# 测试调试消息发送
echo "7. 发送调试消息..."
curl -s -X POST "${BASE_URL}/api/workspaces/${WORKSPACE}/services/${SERVICE_NAME}/debug/test" \
     -H "Content-Type: application/json" \
     -d '{"message":"{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\",\"params\":{}}"}' | jq '.' 2>/dev/null || echo "调试消息发送"
echo ""

# 测试日志获取
echo "8. 获取服务日志..."
curl -s "${BASE_URL}/api/workspaces/${WORKSPACE}/services/${SERVICE_NAME}/debug/logs?limit=5" | jq '.' 2>/dev/null || echo "日志获取接口调用"
echo ""

echo "=== 调试功能测试完成 ==="
echo ""
echo "可用的调试API接口:"
echo "  GET  ${BASE_URL}/api/workspaces/{workspace}/services/{name}/debug/info       - 获取调试信息"
echo "  POST ${BASE_URL}/api/workspaces/{workspace}/services/{name}/debug/test       - 发送调试消息"
echo "  GET  ${BASE_URL}/api/workspaces/{workspace}/services/{name}/debug/connection - 测试连接"
echo "  GET  ${BASE_URL}/api/workspaces/{workspace}/services/{name}/debug/logs       - 获取日志"
echo ""