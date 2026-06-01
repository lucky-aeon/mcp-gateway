package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	internalserver "github.com/lucky-aeon/agentx/plugin-helper/internal/server"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/require"
)

const (
	e2eAPIKey         = "e2e-secret"
	e2eWorkspace      = "default"
	e2eServiceName    = "mock"
	e2eResourceURI    = "mock://resource/readme"
	e2eToolName       = "echo"
	e2eGatewayTool    = e2eServiceName + "_" + e2eToolName
	e2eExpectedText   = "mock resource from source mcp server"
	e2eGatewayAuthHdr = "Bearer " + e2eAPIKey
)

func TestGatewayStreamHTTPToMockSourceE2E(t *testing.T) {
	source := mcpserver.NewTestStreamableHTTPServer(newMockSourceMCPServer(t))
	t.Cleanup(func() {
		source.CloseClientConnections()
		source.Close()
	})

	gatewayURL := startInProcessGateway(t)
	assertGatewayAuthRequired(t, gatewayURL)
	deployMockSource(t, gatewayURL, source.URL, "streamhttp")

	cli, err := mcpclient.NewStreamableHttpClient(
		gatewayURL+"/stream",
		transport.WithHTTPHeaders(map[string]string{"Authorization": e2eGatewayAuthHdr}),
		transport.WithHTTPTimeout(10*time.Second),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	runMCPClientFlow(t, cli, "mcp-gateway")
}

func TestGatewaySSEToMockSourceE2E(t *testing.T) {
	source := mcpserver.NewTestServer(newMockSourceMCPServer(t))
	t.Cleanup(func() {
		source.CloseClientConnections()
		source.Close()
	})

	gatewayURL := startInProcessGateway(t)
	assertGatewayAuthRequired(t, gatewayURL)
	deployMockSource(t, gatewayURL, source.URL+"/sse", "sse")

	cli, err := mcpclient.NewSSEMCPClient(
		gatewayURL+"/sse",
		mcpclient.WithHeaders(map[string]string{"Authorization": e2eGatewayAuthHdr}),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.Close() })

	runMCPClientFlow(t, cli, "")
}

func startInProcessGateway(t *testing.T) string {
	t.Helper()

	cfg := testGatewayConfig(t)
	e := echo.New()
	srv := internalserver.New(cfg, e)
	ts := httptest.NewServer(e)
	t.Cleanup(func() {
		srv.Close()
		ts.Close()
	})
	return ts.URL
}

func testGatewayConfig(t *testing.T) config.Config {
	t.Helper()

	cfg := config.Config{
		WorkspacePath:       t.TempDir(),
		GatewayProtocol:     "all",
		SessionGCInterval:   time.Minute,
		ProxySessionTimeout: time.Minute,
		Auth: &config.AuthConfig{
			Enabled: true,
			Mode:    "single-key",
			ApiKey:  e2eAPIKey,
		},
	}
	cfg.Default()
	return cfg
}

func newMockSourceMCPServer(t *testing.T) *mcpserver.MCPServer {
	t.Helper()

	srv := mcpserver.NewMCPServer(
		"mock-source",
		"1.0.0",
		mcpserver.WithToolCapabilities(true),
		mcpserver.WithResourceCapabilities(true, true),
	)
	srv.AddTool(
		mcp.NewTool(
			e2eToolName,
			mcp.WithDescription("Echoes text through the mock source MCP server"),
			mcp.WithString("text", mcp.Required()),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			text, _ := args["text"].(string)
			return mcp.NewToolResultText("source echo: " + text), nil
		},
	)
	srv.AddResource(
		mcp.NewResource(
			e2eResourceURI,
			"Mock README",
			mcp.WithResourceDescription("A static mock resource exposed by the source MCP server"),
			mcp.WithMIMEType("text/plain"),
		),
		func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return []mcp.ResourceContents{
				mcp.TextResourceContents{
					URI:      req.Params.URI,
					MIMEType: "text/plain",
					Text:     e2eExpectedText,
				},
			}, nil
		},
	)
	return srv
}

func assertGatewayAuthRequired(t *testing.T, gatewayURL string) {
	t.Helper()

	req := newJSONRequest(t, http.MethodPost, gatewayURL+"/stream", mcpInitializePayload(), "")
	resp := doRequest(t, req)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	req = newJSONRequest(t, http.MethodGet, gatewayURL+"/sse", nil, "")
	resp = doRequest(t, req)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func deployMockSource(t *testing.T, gatewayURL, sourceURL, protocol string) {
	t.Helper()

	payload := map[string]any{
		"mcpServers": map[string]any{
			e2eServiceName: map[string]any{
				"url":              sourceURL,
				"gateway_protocol": protocol,
			},
		},
	}
	req := newJSONRequest(t, http.MethodPost, gatewayURL+"/deploy?workspaceId="+e2eWorkspace, payload, e2eAPIKey)
	resp := doRequest(t, req)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Success bool `json:"success"`
		Summary struct {
			Failed int `json:"failed"`
		} `json:"summary"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	require.True(t, body.Success)
	require.Zero(t, body.Summary.Failed)
}

func runMCPClientFlow(t *testing.T, cli *mcpclient.Client, expectedServerName string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	require.NoError(t, cli.Start(ctx))

	initResult, err := cli.Initialize(ctx, mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION,
			ClientInfo: mcp.Implementation{
				Name:    "gateway-e2e-client",
				Version: "1.0.0",
			},
		},
	})
	require.NoError(t, err)
	if expectedServerName != "" {
		require.Equal(t, expectedServerName, initResult.ServerInfo.Name)
	} else {
		require.NotEmpty(t, initResult.ServerInfo.Name)
	}
	require.NotNil(t, initResult.Capabilities.Tools)
	require.NotNil(t, initResult.Capabilities.Resources)

	tools, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	require.NoError(t, err)
	require.Contains(t, toolNames(tools.Tools), e2eGatewayTool)

	resources, err := cli.ListResources(ctx, mcp.ListResourcesRequest{})
	require.NoError(t, err)
	require.Contains(t, resourceURIs(resources.Resources), e2eResourceURI)

	readResult, err := cli.ReadResource(ctx, mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: e2eResourceURI},
	})
	require.NoError(t, err)
	require.Len(t, readResult.Contents, 1)
	textResource, ok := mcp.AsTextResourceContents(readResult.Contents[0])
	require.True(t, ok)
	require.Equal(t, e2eExpectedText, textResource.Text)

	toolResult, err := cli.CallTool(ctx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      e2eGatewayTool,
			Arguments: map[string]any{"text": "hello gateway"},
		},
	})
	require.NoError(t, err)
	require.False(t, toolResult.IsError)
	require.Len(t, toolResult.Content, 1)
	textContent, ok := mcp.AsTextContent(toolResult.Content[0])
	require.True(t, ok)
	require.Equal(t, "source echo: hello gateway", textContent.Text)
}

func newJSONRequest(t *testing.T, method, url string, payload any, token string) *http.Request {
	t.Helper()

	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		require.NoError(t, err)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func toolNames(tools []mcp.Tool) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

func resourceURIs(resources []mcp.Resource) []string {
	uris := make([]string, 0, len(resources))
	for _, resource := range resources {
		uris = append(uris, resource.URI)
	}
	return uris
}
