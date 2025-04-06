package utils

import (
	"net/http"
	"strings"
)

// IsSSE 检查是否是 SSE 请求
func IsSSE(header http.Header) bool {
	return strings.Contains(header.Get("Content-Type"), "text/event-stream")
}
