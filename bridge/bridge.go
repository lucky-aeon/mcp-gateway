package bridge

import (
	"context"
)

type Bridge interface {
	Start(addr string) error
	Close() error
	Ping(ctx context.Context) error
}

type SSEBridge interface {
	Bridge
	CompleteSseEndpoint() (string, error)
	CompleteMessageEndpoint() (string, error)
}

type HTTPStreamBridge interface {
	Bridge
	CompleteHTTPStreamEndpoint() (string, error)
}
