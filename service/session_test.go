package service

import (
	"encoding/json"
	"testing"

	"github.com/lucky-aeon/agentx/plugin-helper/xlog"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestSession(t *testing.T) {
	xl := xlog.NewLogger("test")
	session := NewSession("id")
	defer session.Close()

	mcpFileSystem := mockMcpServiceFileSystem(t)
	if mcpFileSystem == nil {
		t.Fatalf("mockMcpServiceFileSystem failed")
	}
	if err := mcpFileSystem.Start(xl); err != nil {
		t.Fatalf("mockMcpServiceFileSystem.Start failed: %v", err)
	}
	defer func() {
		err := mcpFileSystem.Stop(xl)
		if err != nil {
			t.Errorf("mockMcpServiceFileSystem.Stop failed: %v", err)
		}
	}()
	err := session.SubscribeSSE(xl, mcpFileSystem.Name, mcpFileSystem.GetSSEUrl())
	if err != nil {
		t.Fatalf("subscribeSSE failed: %v", err)
	}

	req := mcp.ListToolsRequest{
		PaginatedRequest: mcp.PaginatedRequest{
			Request: mcp.Request{
				Method: string(mcp.MethodToolsList),
			},
		},
	}
	c := session.GetEventChan()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	err = session.sendToMcp(xl, mcpFileSystem.Name, mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      mcp.NewRequestId(1),
		Request: req.Request,
	}, b)
	if err != nil {
		t.Fatalf("sendToMcp failed: %v", err)
	}
	result := <-c
	if result.Data == "" {
		t.Fatalf("result.Data is nil")
	}
	xl.Infof("Received result: %v", result)
}
