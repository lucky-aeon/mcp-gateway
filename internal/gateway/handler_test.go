package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/lucky-aeon/agentx/plugin-helper/internal/platform/config"
	"github.com/stretchr/testify/assert"
)

func TestRegisterProtocolRoutesRespectGatewayProtocol(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		method   string
		path     string
	}{
		{name: "sse only disables global stream", protocol: "sse", method: http.MethodPost, path: "/stream"},
		{name: "stream only disables global sse", protocol: "streamhttp", method: http.MethodGet, path: "/sse"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			h := NewHandler(&MockServiceManager{}, &config.Config{
				GatewayProtocol: tt.protocol,
				Auth:            &config.AuthConfig{Enabled: false},
			}, nil)
			h.Register(e)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			assert.Equal(t, http.StatusNotFound, rec.Code)
		})
	}
}

func TestRegisterProtocolRoutesDefaultAllowsStreamHTTP(t *testing.T) {
	e := echo.New()
	h := NewHandler(&MockServiceManager{}, &config.Config{
		GatewayProtocol: "all",
		Auth:            &config.AuthConfig{Enabled: false},
	}, nil)
	h.Register(e)

	req := httptest.NewRequest(http.MethodPost, "/stream", nil)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	assert.NotEqual(t, http.StatusNotFound, rec.Code)
}
