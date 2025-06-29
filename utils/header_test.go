package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestIsSSE(t *testing.T) {
	tests := []struct {
		name     string
		header   http.Header
		expected bool
	}{
		{
			name:     "SSE request",
			header:   http.Header{"Content-Type": []string{"text/event-stream"}},
			expected: true,
		},
		{
			name:     "Non-SSE request",
			header:   http.Header{"Content-Type": []string{"application/json"}},
			expected: false,
		},
		{
			name:     "Empty header",
			header:   http.Header{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsSSE(tt.header))
		})
	}
}

func TestGetWorkspace(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name           string
		headerValue    string
		queryValue     string
		defaultValue   []string
		expectedResult string
	}{
		{
			name:           "Header has workspace",
			headerValue:    "workspace-123",
			queryValue:     "",
			defaultValue:   nil,
			expectedResult: "workspace-123",
		},
		{
			name:           "Query has workspace",
			headerValue:    "",
			queryValue:     "workspace-456",
			defaultValue:   nil,
			expectedResult: "workspace-456",
		},
		{
			name:           "Default workspace",
			headerValue:    "",
			queryValue:     "",
			defaultValue:   []string{"default-workspace"},
			expectedResult: "default-workspace",
		},
		{
			name:           "No workspace",
			headerValue:    "",
			queryValue:     "",
			defaultValue:   nil,
			expectedResult: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Workspace-Id", tt.headerValue)
			}
			if tt.queryValue != "" {
				q := req.URL.Query()
				q.Add("workspaceId", tt.queryValue)
				req.URL.RawQuery = q.Encode()
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			result := GetWorkspace(c, tt.defaultValue...)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestGetSession(t *testing.T) {
	e := echo.New()

	tests := []struct {
		name           string
		headerValue    string
		queryValue     string
		expectedResult string
		expectError    bool
	}{
		{
			name:           "Header has session",
			headerValue:    "session-123",
			queryValue:     "",
			expectedResult: "session-123",
			expectError:    false,
		},
		{
			name:           "Query has session",
			headerValue:    "",
			queryValue:     "session-456",
			expectedResult: "session-456",
			expectError:    false,
		},
		{
			name:           "No session",
			headerValue:    "",
			queryValue:     "",
			expectedResult: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.headerValue != "" {
				req.Header.Set("X-Session-Id", tt.headerValue)
			}
			if tt.queryValue != "" {
				q := req.URL.Query()
				q.Add("sessionId", tt.queryValue)
				req.URL.RawQuery = q.Encode()
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			result, err := GetSession(c)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}
