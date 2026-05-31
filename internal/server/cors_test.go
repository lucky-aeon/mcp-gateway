package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func TestCORSExposesMcpSessionIDHeader(t *testing.T) {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(CORSConfig()))
	e.POST("/stream", func(c echo.Context) error {
		c.Response().Header().Set(HeaderMcpSessionID, "session-1")
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/stream", nil)
	req.Header.Set(echo.HeaderOrigin, "http://localhost:6274")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if got := rec.Header().Get(echo.HeaderAccessControlExposeHeaders); !strings.Contains(got, HeaderMcpSessionID) {
		t.Fatalf("expected %s to be exposed, got %q", HeaderMcpSessionID, got)
	}
}

func TestCORSAllowsMcpSessionIDHeader(t *testing.T) {
	e := echo.New()
	e.Use(middleware.CORSWithConfig(CORSConfig()))
	e.POST("/stream", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/stream", nil)
	req.Header.Set(echo.HeaderOrigin, "http://localhost:6274")
	req.Header.Set(echo.HeaderAccessControlRequestMethod, http.MethodPost)
	req.Header.Set(echo.HeaderAccessControlRequestHeaders, HeaderMcpSessionID+", Authorization")
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected preflight 204, got %d", rec.Code)
	}
	if got := rec.Header().Get(echo.HeaderAccessControlAllowHeaders); !strings.Contains(got, HeaderMcpSessionID) {
		t.Fatalf("expected %s to be allowed, got %q", HeaderMcpSessionID, got)
	}
}
