package bridge

type Bridge interface {
	Start() error
	Stop() error
	Restart() error
	IsRunning() bool
	GetURL() string
	GetSSEURL() string
	GetMessageURL() string
}
