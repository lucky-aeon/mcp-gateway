package service

import (
	"net"
)

type PortManagerI interface {
	GetNextAvailablePort() int
	ReleasePort(port int)
}

type portManager struct{}

func NewPortManager() PortManagerI {
	return &portManager{}
}

func (pm *portManager) GetNextAvailablePort() int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		// 如果系统分配失败，回退到从 10000 开始递增
		return 10000
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()
	return port
}

func (pm *portManager) ReleasePort(port int) {
	// 系统自动分配的端口无需手动释放
}
