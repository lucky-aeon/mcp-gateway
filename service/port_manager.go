package service

type PortManagerI interface {
	GetNextAvailablePort() int
	ReleasePort(port int)
}

type portManager struct {
	nextPort int
}

func NewPortManager() PortManagerI {
	return &portManager{
		nextPort: 10000,
	}
}

func (pm *portManager) GetNextAvailablePort() int {
	port := pm.nextPort
	pm.nextPort++
	return port
}

func (pm *portManager) ReleasePort(port int) {
	if port < pm.nextPort {
		pm.nextPort = port
	}
}
